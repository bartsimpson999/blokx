package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/opentradingnetworkfoundation/market-maker/mm/blockchain"
	"github.com/opentradingnetworkfoundation/market-maker/mm/cmc"

	"github.com/opentradingnetworkfoundation/market-maker/mm"
	"github.com/opentradingnetworkfoundation/otn-go/api"
	"github.com/opentradingnetworkfoundation/otn-go/otn-microservice"
	"github.com/opentradingnetworkfoundation/otn-go/wallet"
	"go.uber.org/zap"
)

type App struct {
	marketMakers []*mm.MarketMaker

	cfg          *MarketMakerConfig
	log          *zap.SugaredLogger
	api          api.BitsharesAPI
	balanceMutex sync.Mutex
	signalled    bool
}

func NewApp(cfg *MarketMakerConfig) (*App, error) {
	if len(cfg.Markets) == 0 {
		return nil, fmt.Errorf("No markets configured")
	}

	lg, err := cfg.Logger.Build()
	if err != nil {
		return nil, err
	}
	zap.RedirectStdLog(lg)
	app := &App{
		cfg: cfg,
		log: lg.Sugar(),
	}

	return app, nil
}

func (a *App) Start(rpc api.BitsharesAPI) {
	wallet := wallet.NewWallet()
	if err := wallet.AddPrivateKeys(a.cfg.Keys); err != nil {
		a.log.Fatal("Failed to import keys")
	}

	marketMakers := make([]*mm.MarketMaker, len(a.cfg.Markets))
	var provFactory mm.PriceProviderFactory

	if a.cfg.PriceProvider.CMC != nil {
		f, err := cmc.NewFactory(a.cfg.PriceProvider.CMC, a.log)
		if err != nil {
			a.log.Fatal("Failed to create CMC provider: %s", err)
		}
		provFactory = f
	} else {
		provFactory = blockchain.NewFactory(rpc)
	}

	for i, marketCfg := range a.cfg.Markets {
		marketMakerConfig := &mm.Config{
			Market:         marketCfg,
			UpdateInterval: time.Second * 3,
			Account:        a.cfg.Account,
			FeeReserve:     a.cfg.FeeReserve,
		}
		marketMakers[i] = mm.NewMarketMaker(
			marketMakerConfig, rpc, wallet, provFactory, a.log, &a.balanceMutex)
	}

	a.marketMakers = marketMakers
	a.log.Info("Start markets")

	startedCount := 0
	for _, market := range a.marketMakers {
		if err := market.Start(); err != nil {
			a.log.Errorf("Failed to start market %s: %s", market.Market().DisplayName(), err)
		} else {
			startedCount++
		}
	}
}

func (a *App) Stop() {
	a.log.Info("Stop markets")
	for _, market := range a.marketMakers {
		market.Stop()
	}
}

func (a *App) SignalHandler(s os.Signal) {
	log.Printf("Got %s signal...", s.String())
	a.signalled = true
}

var (
	configPath string
)

func main() {
	flag.StringVar(&configPath, "cfg", "otn-market-maker.json", "Configuration file path")
	flag.Parse()

	log.Println("Loading configuration from", configPath)

	cfgSources := strings.Split(configPath, ",")
	cfgLoader, err := NewConfigLoader(cfgSources)
	if err != nil {
		log.Fatal("Failed to create configuration loader: ", err)
	}
	defer cfgLoader.Shutdown()

	cfg := &MarketMakerConfig{}
	cfg.Logger = zap.NewProductionConfig()

	if err := cfgLoader.Load(cfg); err != nil {
		log.Fatal("Failed to load configuration: ", err)
	}
	postProcessConfig(cfg)

	app, err := NewApp(cfg)
	if err != nil {
		log.Fatal(err)
	}

	sc := &otn.StarterConfig{
		InstanceLock: app.cfg.InstanceLock,
		TrustedNode:  app.cfg.NodeAddr,
	}
	starter := otn.NewStarter(app, sc)
	doneChan := make(chan struct{})

	cfgLoader.Watch(func() {
		// stop service if we got new config
		log.Printf("Config changed, stopping service")
		doneChan <- struct{}{}
	})

	starter.Run(doneChan)
}
