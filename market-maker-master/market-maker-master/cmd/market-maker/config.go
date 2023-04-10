package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/hashicorp/consul/api"
	"github.com/juju/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/opentradingnetworkfoundation/market-maker/mm"
	"github.com/opentradingnetworkfoundation/market-maker/mm/cmc"
	"github.com/opentradingnetworkfoundation/otn-go/consul"
	"github.com/opentradingnetworkfoundation/otn-go/secrets"
)

type PriceProviderConfig struct {
	CMC *cmc.Config
}

type MarketMakerConfig struct {
	NodeAddr      string                 `json:"node_addr"`
	Account       string                 `json:"account"`
	InstanceLock  string                 `json:"instance_lock"`
	FeeReserve    decimal.Decimal        `json:"fee_reserve"`
	Markets       []mm.MarketConfig      `json:"markets"`
	PriceProvider PriceProviderConfig    `json:"price_provider"`
	Keys          []string               `json:"keys"`
	Secrets       *secrets.StorageConfig `json:"secrets"`
	Logger        zap.Config             `json:"logger"`
}

func postProcessConfig(cfg *MarketMakerConfig) error {
	cfg.NodeAddr = os.ExpandEnv(cfg.NodeAddr)

	if cfg.Secrets != nil {
		stg, err := secrets.NewSecretStorage(cfg.Secrets)
		if err != nil {
			return errors.Annotate(err, "create secret storage")
		}

		keys, err := stg.ReadStringArray("account")
		if err != nil {
			return errors.Annotate(err, "get account keys")
		}

		cfg.Keys = append(cfg.Keys, keys...)
	}

	return nil
}

const consulPrefix = "consul://"

type ConfigLoader struct {
	sources  []string
	onChange func()

	keyVersions map[string]uint64
	client      *api.Client
	keyWatch    *consul.KeyWatch
	mutex       sync.Mutex
}

func NewConfigLoader(sources []string) (*ConfigLoader, error) {
	client, err := consul.NewClient()
	if err != nil {
		return nil, err
	}
	return &ConfigLoader{
		sources:     sources,
		client:      client,
		keyVersions: make(map[string]uint64),
		keyWatch:    consul.NewKeyWatch(client),
	}, nil
}

func (c *ConfigLoader) Load(cfg interface{}) (err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, src := range c.sources {
		var data []byte
		if strings.HasPrefix(src, consulPrefix) {
			key := strings.TrimPrefix(src, consulPrefix)
			kv, _, err := c.client.KV().Get(key, nil)
			if err != nil {
				return errors.Annotatef(err, "Failed to read consul key %s", key)
			}
			if kv == nil {
				return fmt.Errorf("consul key %s does not exist", key)
			}
			data = kv.Value
			c.keyVersions[key] = kv.ModifyIndex
		} else {
			data, err = ioutil.ReadFile(src)
			if err != nil {
				return err
			}
		}

		if err := json.Unmarshal(data, cfg); err != nil {
			return errors.Annotatef(err, "Failed to parse configuration from %s", src)
		}
	}

	return nil
}

func (c *ConfigLoader) Watch(onChange func()) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.onChange = onChange
	for key := range c.keyVersions {
		c.keyWatch.AddHandler(key, c.onKeyChange)
	}
}

func (c *ConfigLoader) onKeyChange(kv *consul.KVPair) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	currentVersion := c.keyVersions[kv.Key]
	if kv.ModifyIndex > currentVersion {
		c.keyVersions[kv.Key] = kv.ModifyIndex
		c.onChange()
	}
}

func (c *ConfigLoader) Shutdown() {
	c.keyWatch.Shutdown()
}
