package cmc

import (
	"fmt"
	"math"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/opentradingnetworkfoundation/otn-go/coinmarketcap"
	"github.com/opentradingnetworkfoundation/market-maker/mm"
	"github.com/opentradingnetworkfoundation/otn-go/objects"
)

const (
	DefaultInterval = time.Minute
)

type priceProviderFactory struct {
	cfg      *Config
	client   *coinmarketcap.Client
	interval time.Duration
	log      *zap.SugaredLogger

	providers   map[string]bool
	priceCache  map[string]*coinmarketcap.Ticker
	symbolMap   map[string]*coinmarketcap.Listing
	lastUpdated time.Time
	mutex       sync.Mutex
}

// PriceProviderFactory interface
func (f *priceProviderFactory) GetProvider(market *mm.Market) (mm.PriceProvider, error) {
	if _, ok := f.symbolMap[market.Base.Symbol]; !ok {
		return nil, fmt.Errorf("Unknown asset '%s'", market.Base.Symbol)
	}
	if _, ok := f.symbolMap[market.Quote.Symbol]; !ok {
		return nil, fmt.Errorf("Unknown asset '%s'", market.Quote.Symbol)
	}
	f.mutex.Lock()
	f.lastUpdated = time.Time{}
	f.providers[market.Base.Symbol] = true
	f.providers[market.Quote.Symbol] = true
	f.mutex.Unlock()
	return &priceProvider{market: market, factory: f}, nil
}

func (f *priceProviderFactory) getTicker(symbol string) *coinmarketcap.Ticker {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if f.lastUpdated.Add(f.interval).Before(time.Now()) {
		f.update()
		f.lastUpdated = time.Now()
	}

	return f.priceCache[symbol]
}

func mapTickersBySymbol(tickers map[string]*coinmarketcap.Ticker) map[string]*coinmarketcap.Ticker {
	result := make(map[string]*coinmarketcap.Ticker)
	for _, t := range tickers {
		result[t.Symbol] = t
	}
	return result
}

func (f *priceProviderFactory) update() {
	bulk, err := f.client.Tickers(&coinmarketcap.TickersOptions{
		Limit:   f.cfg.BulkSize,
		Convert: "BTC",
	})

	if err != nil {
		f.log.Errorw("Failed to get tickers", "error", err)
		return
	}

	tickers := mapTickersBySymbol(bulk)

	for sym := range f.providers {
		t, ok := tickers[sym]
		if !ok {
			f.log.Infow("Fetching separate info", "symbol", sym)
			t, err = f.client.Ticker(&coinmarketcap.TickerOptions{
				ID:      f.symbolMap[sym].ID,
				Convert: "BTC",
			})
			if err != nil {
				f.log.Errorw("Failed to get ticker", "symbol", sym, "error", err)
			}
		}

		// place result into cache
		if t != nil {
			f.priceCache[sym] = t
		}
	}
}

type priceProvider struct {
	market  *mm.Market
	factory *priceProviderFactory
}

func (p *priceProvider) GetPrice() (price objects.Price) {
	baseTicker := p.factory.getTicker(p.market.Base.Symbol)
	quoteTicker := p.factory.getTicker(p.market.Quote.Symbol)

	if baseTicker == nil || quoteTicker == nil {
		return
	}

	baseBtc := baseTicker.Quotes["BTC"].Price
	quoteBtc := quoteTicker.Quotes["BTC"].Price

	baseAmount := math.Pow10(p.market.Base.Precision)
	quoteAmount := baseAmount * baseBtc / quoteBtc

	return objects.Price{
		Base: objects.AssetAmount{
			Asset:  p.market.Base.ID,
			Amount: objects.Int64(baseAmount),
		},
		Quote: objects.AssetAmount{
			Asset:  p.market.Quote.ID,
			Amount: objects.Int64(quoteAmount),
		},
	}
}

func NewFactory(cfg *Config, log *zap.SugaredLogger) (prov mm.PriceProviderFactory, err error) {
	client := coinmarketcap.NewClient(&coinmarketcap.Options{
		URL: cfg.URL,
	})

	interval := DefaultInterval
	if cfg.Interval != "" {
		interval, err = time.ParseDuration(cfg.Interval)
		if err != nil {
			return nil, err
		}
	}

	lst, err := client.Listings()
	if err != nil {
		return
	}

	prov = &priceProviderFactory{
		cfg:        cfg,
		client:     client,
		log:        log,
		interval:   interval,
		providers:  make(map[string]bool),
		priceCache: make(map[string]*coinmarketcap.Ticker),
		symbolMap:  coinmarketcap.MapListingsBySymbol(lst),
	}

	return
}
