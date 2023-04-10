package mm

import (
	"github.com/opentradingnetworkfoundation/otn-go/objects"
	"go.uber.org/zap"
)

type OrderBook struct {
	Sell objects.LimitOrders
	Buy  objects.LimitOrders
	log  *zap.SugaredLogger
}

func NewOrderBook(orders objects.LimitOrders, market *Market, logger *zap.SugaredLogger) OrderBook {
	return OrderBook{
		Sell: FilterByAsset(orders, market.Base.ID),
		Buy:  FilterByAsset(orders, market.Quote.ID),
		log:  logger,
	}
}

func (o *OrderBook) Orders() objects.LimitOrders {
	return append(o.Sell, o.Buy...)
}

func (o *OrderBook) BuyAmount() uint64 {
	return OrdersAmount(o.Buy)
}

func (o *OrderBook) SellAmount() uint64 {
	return OrdersAmount(o.Sell)
}

func (o *OrderBook) Log(market *Market) {
	o.log.Info("SELL:")
	o.logOrders(o.Sell, market, false)
	o.log.Info("BUY:")
	o.logOrders(o.Buy, market, true)
}

func FilterByAsset(orders objects.LimitOrders, asset objects.GrapheneID) objects.LimitOrders {
	var result objects.LimitOrders
	for _, order := range orders {
		if order.SellPrice.Base.Asset == asset {
			result = append(result, order)
		}
	}
	return result
}

func FilterBySeller(orders objects.LimitOrders, seller objects.GrapheneID) objects.LimitOrders {
	var result objects.LimitOrders
	for _, order := range orders {
		if order.Seller == seller {
			result = append(result, order)
		}
	}
	return result
}

func OrdersAmount(orders objects.LimitOrders) uint64 {
	var amount uint64
	for _, order := range orders {
		amount += uint64(order.ForSale)
	}
	return amount
}

func (this *OrderBook) logOrders(orders objects.LimitOrders, market *Market, inverse bool) {
	for _, o := range orders {
		price := market.GetRate(o.SellPrice).Value()
		if inverse {
			price = 1 / price
		}
		this.log.Infof("%.6f: %d (fee: %d)", price, o.ForSale, o.DeferredFee)
	}
}
