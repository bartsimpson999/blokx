package mm

import (
	"github.com/opentradingnetworkfoundation/otn-go/objects"
)

// PriceProvider reports price for a given asset
type PriceProvider interface {
	GetPrice() objects.Price
}

// PriceProviderFactory creates PriceProvider for the given market
type PriceProviderFactory interface {
	GetProvider(market *Market) (PriceProvider, error)
}
