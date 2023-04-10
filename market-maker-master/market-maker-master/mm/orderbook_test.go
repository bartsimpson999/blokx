package mm_test

import (
	"log"
	"testing"

	"github.com/opentradingnetworkfoundation/market-maker/mm"
	"github.com/opentradingnetworkfoundation/otn-go/objects"
)

func TestOrderBook(t *testing.T) {

	seller := *objects.NewGrapheneID("1.2.17")
	otnAsset := *objects.NewGrapheneID("1.3.0")
	btcAsset := *objects.NewGrapheneID("1.3.2")

	orderBook := mm.OrderBook{
		Sell: objects.LimitOrders{
			objects.LimitOrder{
				ID:          *objects.NewGrapheneID("1.7.110"),
				Seller:      seller,
				ForSale:     0xe8d4a51000,
				DeferredFee: 0x77359400,
				SellPrice: objects.Price{
					Base:  objects.AssetAmount{Asset: otnAsset, Amount: 0xe8d4a51000},
					Quote: objects.AssetAmount{Asset: btcAsset, Amount: 0x11c2276d},
				}},
			objects.LimitOrder{
				ID:          *objects.NewGrapheneID("1.7.112"),
				Seller:      seller,
				ForSale:     0xe8d4a51000,
				DeferredFee: 0x77359400,
				SellPrice: objects.Price{
					Base:  objects.AssetAmount{Asset: otnAsset, Amount: 0xe8d4a51000},
					Quote: objects.AssetAmount{Asset: btcAsset, Amount: 0x11ee81b9}}},
			objects.LimitOrder{
				ID:          *objects.NewGrapheneID("1.7.114"),
				Seller:      seller,
				ForSale:     0xe8d4a51000,
				DeferredFee: 0x77359400,
				SellPrice: objects.Price{
					Base:  objects.AssetAmount{Asset: otnAsset, Amount: 0xe8d4a51000},
					Quote: objects.AssetAmount{Asset: btcAsset, Amount: 0x121adc05}}}},

		Buy: objects.LimitOrders{
			objects.LimitOrder{
				ID:          *objects.NewGrapheneID("1.7.111"),
				Seller:      seller,
				ForSale:     0x10e71847,
				DeferredFee: 0x77359400,
				SellPrice: objects.Price{
					Base:  objects.AssetAmount{Asset: btcAsset, Amount: 0x10e71847},
					Quote: objects.AssetAmount{Asset: otnAsset, Amount: 0xe8d4a51000}}},
			objects.LimitOrder{
				ID:          *objects.NewGrapheneID("1.7.113"),
				Seller:      seller,
				ForSale:     0x10bd4983,
				DeferredFee: 0x77359400,
				SellPrice: objects.Price{
					Base:  objects.AssetAmount{Asset: btcAsset, Amount: 0x10bd4983},
					Quote: objects.AssetAmount{Asset: otnAsset, Amount: 0xe8d4a51000}}},
			objects.LimitOrder{
				ID:          *objects.NewGrapheneID("1.7.115"),
				Seller:      seller,
				ForSale:     0x10944796,
				DeferredFee: 0x77359400,
				SellPrice: objects.Price{
					Base:  objects.AssetAmount{Asset: btcAsset, Amount: 0x10944796},
					Quote: objects.AssetAmount{Asset: otnAsset, Amount: 0xe8d4a51000}}}}}

	log.Println(orderBook.SellAmount())
	log.Println(orderBook.BuyAmount())
}
