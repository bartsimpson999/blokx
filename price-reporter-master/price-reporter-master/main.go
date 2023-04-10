package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/opentradingnetworkfoundation/otn-go/api"
	"github.com/opentradingnetworkfoundation/otn-go/wallet"
)

func main() {
	configPath := flag.String("cfg", "price-reporter.json", "Configuration file path")
	flag.Parse()

	cfg := &PriceReporterConfig{}
	log.Println("Loading configuration from", *configPath)
	err := LoadConfig(*configPath, cfg)
	if err != nil {
		log.Fatalln("Failed to load configuration:", err)
	}

	rpcConn := api.NewConnection(cfg.TrustedNode)
	rpc := api.New(rpcConn)
	wallet := wallet.NewWallet()
	if err := wallet.AddPrivateKeys(cfg.Keys); err != nil {
		log.Fatal("Failed to import private keys: ", err)
	}
	pr, err := NewPriceReporter(rpc, wallet, cfg)
	if err != nil {
		log.Fatal("Unable to create price reporter: ", err)
	}
	if err := rpcConn.Connect(); err != nil {
		log.Fatalf("Unable to open connection to API node %s: %v", cfg.TrustedNode, err)
	}

	// wait for signal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-signalChan:
	}

	pr.Stop()
}
