package mm

import (
	"fmt"

	"github.com/opentradingnetworkfoundation/otn-go/objects"
)

type Market struct {
	Base  objects.Asset
	Quote objects.Asset
}

func (m *Market) DisplayName() string {
	return fmt.Sprintf("%s/%s", m.Base.Symbol, m.Quote.Symbol)
}

func (m *Market) NewPrice(baseAmount, quoteAmount uint64) objects.Price {
	return objects.Price{
		Base: objects.AssetAmount{
			Asset:  m.Base.ID,
			Amount: objects.Int64(baseAmount),
		},
		Quote: objects.AssetAmount{
			Asset:  m.Base.ID,
			Amount: objects.Int64(quoteAmount),
		},
	}
}

func (m *Market) GetRate(price objects.Price) objects.Rate {
	if price.Valid() {
		if price.Base.Asset == m.Base.ID && price.Quote.Asset == m.Quote.ID {
			return price.Rate(m.Base.Precision, m.Quote.Precision)
		}

		if price.Base.Asset == m.Quote.ID && price.Quote.Asset == m.Base.ID {
			return price.Rate(m.Quote.Precision, m.Base.Precision)
		}
	}

	return objects.Rate(0)
}
