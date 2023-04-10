package main

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/juju/errors"
	"github.com/opentradingnetworkfoundation/otn-go/secrets"
	"go.uber.org/zap"
)

type PriceReporterConfig struct {
	CoinmarketcapProxy string                 `json:"coinmarketcap_proxy"`
	Logger             zap.Config             `json:"logger"`
	AssetFeeds         []AssetFeedConfig      `json:"feeds"`
	Keys               []string               `json:"keys"`
	TrustedNode        string                 `json:"trusted_node"`
	Publishers         []string               `json:"publishers"`
	Secrets            *secrets.StorageConfig `json:"secrets"`
}

func LoadConfig(filename string, cfg *PriceReporterConfig) error {
	// Read the config file.
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return errors.Annotate(err, "Error reading file")
	}
	cfg.Logger = zap.NewProductionConfig()
	// Decode json.
	err = json.Unmarshal([]byte(data), cfg)
	if err != nil {
		return errors.Annotate(err, "Failed to parse json")
	}

	// empty publishers list means use default global publishers list
	for i := range cfg.AssetFeeds {
		if len(cfg.AssetFeeds[i].Publishers) == 0 {
			cfg.AssetFeeds[i].Publishers = cfg.Publishers
		}
	}

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

	cfg.TrustedNode = os.ExpandEnv(cfg.TrustedNode)
	return nil
}
