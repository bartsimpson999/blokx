package main

import (
	"time"

	"github.com/opentradingnetworkfoundation/otn-go/coinmarketcap"
	"go.uber.org/zap"
)

type CoinmarketcapConfig struct {
	Asset string `json:"asset"`
	ID    string `json:"id"`
}

type SinusoidConfig struct {
	Asset  string  `json:"asset"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Period int64   `json:"period"`
}

type AssetFeedConfig struct {
	Coinmarketcap []CoinmarketcapConfig `json:"coinmarketcap"`
	Sinusoid      []SinusoidConfig      `json:"sinusoid"`
	Publishers    []string              `json:"publishers"`
	Bulksize      int                   `json:"bulksize"`
	Interval      int64                 `json:"interval"`
	Filter        *PriceFilterConfig    `json:"filter"`
}

type PricePublisher interface {
	PublishPrice(assetID, publisher string, price float64) error
}

type AssetFeed struct {
	cmc        *coinmarketcap.ProClient
	log        *zap.SugaredLogger
	cfg        *AssetFeedConfig
	cmcSymbols []string
	filterBtc  *PriceFilter
	filterUsd  *PriceFilter
	ticker     *time.Ticker
	publisher  PricePublisher
}

const (
	symbolBTC = "BTC"
	symbolUSD = "USD"
	symbolOTN = "OTN"
	idBTC     = "1"
	idOTN     = "2069"
)

func makeCmcSymbols(cfg *AssetFeedConfig) []string {
	if len(cfg.Coinmarketcap) == 0 {
		return []string{}
	}

	cmcSymbols := make([]string, 0, len(cfg.Coinmarketcap))

	for _, item := range cfg.Coinmarketcap {
		if item.Asset != symbolUSD {
			cmcSymbols = append(cmcSymbols, item.Asset)
		}
	}

	return append(cmcSymbols, symbolOTN)
}

func NewAssetFeed(cmc *coinmarketcap.ProClient, logger *zap.SugaredLogger, cfg *AssetFeedConfig, publisher PricePublisher) *AssetFeed {
	feed := &AssetFeed{
		cmc:        cmc,
		cmcSymbols: makeCmcSymbols(cfg),
		log:        logger,
		cfg:        cfg,
		publisher:  publisher,
	}

	if cfg.Filter != nil {
		feed.filterBtc = NewPriceFilter(cfg.Filter)
		feed.filterUsd = NewPriceFilter(cfg.Filter)
	}

	return feed
}

func (c *AssetFeed) Start() {
	c.ticker = time.NewTicker(time.Duration(c.cfg.Interval) * time.Second)
	go c.worker(c.ticker)
}

func (c *AssetFeed) Stop() {
	if c.ticker != nil {
		c.ticker.Stop()
		c.ticker = nil
	}
}

func (c *AssetFeed) worker(ticker *time.Ticker) {
	for t := range ticker.C {
		c.reportSinus(t)
		c.reportCoinmarketcap(t)
	}
}

func (p *AssetFeed) reportSinus(now time.Time) {
	for _, params := range p.cfg.Sinusoid {
		price := GetSinusoidPrice(&params, now)
		for _, publisher := range p.cfg.Publishers {
			p.publisher.PublishPrice(params.Asset, publisher, price)
		}
	}
}

func (c *AssetFeed) reportCoinmarketcap(now time.Time) {
	if len(c.cmcSymbols) == 0 {
		return
	}

	bulk, err := c.cmc.Quotes(
		coinmarketcap.QuotesSymbols(c.cmcSymbols),
		coinmarketcap.QuotesConvert([]string{symbolBTC, symbolUSD}),
	)

	if err != nil {
		c.log.Errorf("Failed to get OTN info: %v", err)
		return
	}

	coreUsdPrice := bulk[symbolOTN].Quotes[symbolUSD].Price
	coreBtcPrice := bulk[symbolOTN].Quotes[symbolBTC].Price

	// check price filter if configured
	if c.filterBtc != nil {
		c.filterBtc.Add(coreBtcPrice, now)
		avg := c.filterBtc.Avg()
		c.log.Infof("OTN/BTC: spot = %f, avg = %f, diff = %f ", coreBtcPrice, avg, coreBtcPrice-avg)
		coreBtcPrice = avg
	}

	if c.filterUsd != nil {
		c.filterUsd.Add(coreUsdPrice, now)
		coreUsdPrice = c.filterUsd.Avg()
	}

	btcUsdPrice := bulk[symbolBTC].Quotes[symbolUSD].Price

	c.log.Infof("BTC: %f USD, %f OTN", btcUsdPrice, 1/coreBtcPrice)
	c.log.Infof("OTN: %f USD (%f via BTC), %f BTC",
		coreUsdPrice, coreBtcPrice*btcUsdPrice, coreBtcPrice)

	for _, asset := range c.cfg.Coinmarketcap {
		var price float64
		if asset.Asset == "USD" {
			price = 1 / coreUsdPrice
		} else {
			info, ok := bulk[asset.Asset]
			if ok {
				price = info.Quotes[symbolBTC].Price / coreBtcPrice
			}
		}

		if price != 0 {
			for _, publisher := range c.cfg.Publishers {
				c.publisher.PublishPrice(asset.Asset, publisher, price)
			}
		}
	}
}
