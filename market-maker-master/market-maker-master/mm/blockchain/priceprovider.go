package blockchain

import (
	"math/big"

	"github.com/opentradingnetworkfoundation/market-maker/mm"
	"github.com/opentradingnetworkfoundation/otn-go/api"
	"github.com/opentradingnetworkfoundation/otn-go/objects"
)

var coreAsset = *objects.NewGrapheneID("1.3.0")

func corePrice() objects.Price {
	return objects.Price{
		Base:  objects.AssetAmount{Asset: coreAsset, Amount: 1},
		Quote: objects.AssetAmount{Asset: coreAsset, Amount: 1},
	}
}

type assetProviderFactory struct {
	rpc api.BitsharesAPI
}

func NewFactory(rpc api.BitsharesAPI) mm.PriceProviderFactory {
	return &assetProviderFactory{rpc}
}

func (f *assetProviderFactory) GetProvider(market *mm.Market) (mm.PriceProvider, error) {
	dbAPI, err := f.rpc.DatabaseAPI()
	if err != nil {
		return &assetPriceProvider{}, err
	}
	return &assetPriceProvider{
		rpc:    dbAPI,
		market: market,
	}, nil
}

type assetPriceProvider struct {
	rpc    api.DatabaseAPI
	market *mm.Market
}

func inversePrice(p objects.Price) objects.Price {
	return objects.Price{Base: p.Quote, Quote: p.Base}
}

func (p *assetPriceProvider) getAssetPrice(asset *objects.Asset) objects.Price {
	var price objects.Price
	if asset.BitassetDataID.Valid() {
		data, err := p.rpc.GetObjects(asset.BitassetDataID)
		if err != nil {
			return objects.Price{}
		}
		feed := data[0].(objects.BitAssetData)
		price = feed.CurrentFeed.SettlementPrice
	} else {
		price = corePrice()
	}

	if price.Quote.Asset != coreAsset {
		return inversePrice(price)
	}

	return price
}

func priceAsFloat(price objects.Price) *big.Float {
	return new(big.Float).Quo(
		new(big.Float).SetInt64(int64(price.Quote.Amount)),
		new(big.Float).SetInt64(int64(price.Base.Amount)))
}

func (p *assetPriceProvider) GetPrice() objects.Price {
	basePrice := p.getAssetPrice(&p.market.Base)
	quotePrice := p.getAssetPrice(&p.market.Quote)

	if !basePrice.Valid() || !quotePrice.Valid() {
		return objects.Price{}
	}

	return objects.Price{
		Base: objects.AssetAmount{
			Asset:  p.market.Base.ID,
			Amount: basePrice.Base.Amount * quotePrice.Quote.Amount},
		Quote: objects.AssetAmount{
			Asset:  p.market.Quote.ID,
			Amount: basePrice.Quote.Amount * quotePrice.Base.Amount},
	}
}
