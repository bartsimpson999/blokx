package cmc

import (
	"testing"

	"go.uber.org/zap"

	"github.com/opentradingnetworkfoundation/market-maker/mm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/opentradingnetworkfoundation/otn-go/objects"
)

var (
	// IDs
	idOTN = *objects.NewGrapheneID("1.3.0")
	idBTC = *objects.NewGrapheneID("1.3.1")
	idETH = *objects.NewGrapheneID("1.3.2")

	// Assets
	assetOTN = objects.Asset{ID: idOTN, Symbol: "OTN", Precision: 8}
	assetBTC = objects.Asset{ID: idBTC, Symbol: "BTC", Precision: 8}
	assetETH = objects.Asset{ID: idETH, Symbol: "ETH", Precision: 8}

	// Markets
	marketOTNBTC = mm.Market{Base: assetOTN, Quote: assetBTC}
	marketETHBTC = mm.Market{Base: assetETH, Quote: assetBTC}
)

func TestProvider(t *testing.T) {
	cfg := &Config{BulkSize: 10}
	f, err := NewFactory(cfg, zap.NewNop().Sugar())
	require.NoError(t, err)

	p1, err := f.GetProvider(&marketOTNBTC)
	require.NoError(t, err)

	otnBtcPrice := p1.GetPrice()
	assert.True(t, otnBtcPrice.Valid())
	assert.Equal(t, otnBtcPrice.Base.Asset, idOTN)
	assert.Equal(t, otnBtcPrice.Quote.Asset, idBTC)
}
