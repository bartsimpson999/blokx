package main

import (
	"fmt"
	"sync"

	"github.com/juju/errors"
	"github.com/opentradingnetworkfoundation/otn-go/coinmarketcap"
	"github.com/opentradingnetworkfoundation/otn-go/api"
	"github.com/opentradingnetworkfoundation/otn-go/objects"
	"github.com/opentradingnetworkfoundation/otn-go/wallet"
	"go.uber.org/zap"
)

type PriceReporter struct {
	rpc        api.BitsharesAPI
	cmc        *coinmarketcap.ProClient
	wallet     wallet.Wallet
	assetCache *api.AssetCache
	cfg        []AssetFeedConfig
	log        *zap.SugaredLogger

	// mutable state
	coreAsset *objects.Asset
	mutex     sync.Mutex
	feeds     []*AssetFeed
	connected bool
}

func (p *PriceReporter) Start() {
	p.log.Info("Starting reporting")

	// Load core asset
	if p.coreAsset == nil {
		coreAsset := p.assetCache.GetByID(objects.NewGrapheneID("1.3.0"))
		if coreAsset == nil {
			p.log.Error("Cannot get core asset")
			return
		}

		p.coreAsset = coreAsset
	}

	// Feeds are started on first connect to node
	if p.feeds == nil {
		feeds := make([]*AssetFeed, len(p.cfg))
		for i := range p.cfg {
			feeds[i] = NewAssetFeed(p.cmc, p.log, &p.cfg[i], p)
			feeds[i].Start()
		}

		p.feeds = feeds
	}

	p.connected = true
}

func (p *PriceReporter) Stop() {
	p.connected = false
}

func (p *PriceReporter) PublishPrice(assetID string, publisher string, price float64) (err error) {
	{
		// check if connected
		p.mutex.Lock()
		defer p.mutex.Unlock()
		if p.connected == false {
			return nil
		}
	}

	p.log.Infof("[%s] Publish price %f from %s", assetID, price, publisher)

	rate := 1.0 / price
	publisherID := objects.NewGrapheneID(objects.ObjectID(publisher))
	asset := p.assetCache.GetAsset(assetID)

	if asset == nil {
		return fmt.Errorf("Failed to get asset '%s'", assetID)
	}

	op := objects.NewAssetPublishFeedOperation()

	op.Publisher = *publisherID
	op.AssetID = asset.ID
	op.Feed.SettlementPrice.Set(asset, p.coreAsset, rate)
	op.Feed.CoreExchangeRate.Set(asset, p.coreAsset, rate*1.05)

	_, err = p.rpc.SignAndBroadcast(p.wallet.GetKeys(), &p.coreAsset.ID, op)

	if err != nil {
		err = errors.Annotate(err, "Failed to broadcast transaction")
		p.log.Errorf("%v", err.Error())
		return err
	}

	return nil
}

func NewPriceReporter(rpc api.BitsharesAPI, wallet wallet.Wallet, cfg *PriceReporterConfig) (*PriceReporter, error) {
	l, err := cfg.Logger.Build()
	if err != nil {
		return nil, err
	}
	cmc := coinmarketcap.NewProClient(
		&coinmarketcap.Options{URL: cfg.CoinmarketcapProxy})

	pr := &PriceReporter{
		rpc:        rpc,
		log:        l.Sugar(),
		cmc:        cmc,
		wallet:     wallet,
		cfg:        cfg.AssetFeeds,
		assetCache: api.NewAssetCache(rpc),
	}

	rpc.RegisterCallback(pr.loginEventsHandler)
	return pr, nil
}

func (p *PriceReporter) loginEventsHandler(event api.BitsharesAPIEvent) {
	p.log.Infof("Got login API event %v", event)
	p.mutex.Lock()
	defer p.mutex.Unlock()

	switch event {
	case api.BitsharesAPIEventLogin:
		p.Start()
	case api.BitsharesAPIEventLogout:
		p.Stop()
	}
}
