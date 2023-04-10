package mm

import (
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/juju/errors"
	"github.com/shopspring/decimal"

	"github.com/opentradingnetworkfoundation/otn-go/api"
	"github.com/opentradingnetworkfoundation/otn-go/objects"
	"github.com/opentradingnetworkfoundation/otn-go/wallet"
	"go.uber.org/zap"
)

type Config struct {
	Market         MarketConfig
	UpdateInterval time.Duration
	Account        string
	FeeReserve     decimal.Decimal
}

const (
	orderBookDepth       = 50
	orderAmountThreshold = 10 // do not create orders with less than this amount
)

type MarketMaker struct {
	rpc           api.BitsharesAPI
	log           *zap.SugaredLogger
	assetCache    *api.AssetCache
	cfg           *Config
	balanceMutex  *sync.Mutex
	feeAsset      objects.GrapheneID
	wallet        wallet.Wallet
	factory       PriceProviderFactory
	market        Market
	ticker        *time.Ticker
	account       *objects.Account
	baseBalance   objects.AssetAmount
	quoteBalance  objects.AssetAmount
	priceProvider PriceProvider
	orderDuration time.Duration

	// Mutable
	lastPrice        float64
	lastMarketUpdate time.Time
}

func (m *MarketMaker) Market() *Market {
	return &m.market
}

func (m *MarketMaker) worker() error {
	// TODO: subscribe on market change (m.onMarketChange)
	for t := range m.ticker.C {
		m.makeMarket(t)
	}

	return nil
}

func (m *MarketMaker) CancelOrders() error {
	orderBook, err := m.loadOrderBook()
	if err != nil {
		return errors.Annotate(err, "loadOrderBook")
	}

	orderBook.Log(&m.market)

	cancelOps := m.createCancelOrders(orderBook)
	if len(cancelOps) > 0 {
		_, err = api.SignAndBroadcast(m.rpc, m.wallet.GetKeys(), m.feeAsset, cancelOps...)
		if err != nil {
			return errors.Annotate(err, "SignAndBroadcast")
		}
	}

	return nil
}

func (m *MarketMaker) makeMarket(t time.Time) {
	price := m.priceProvider.GetPrice()
	rate := m.market.GetRate(price).Value()

	// if failed to get price, remove all active orders
	if rate == 0 {
		m.log.Error("Failed to get price")
		m.CancelOrders()
		return
	}

	m.log.Infof("Price: %f, inverse: %f", rate, 1/rate)
	change := math.Abs(m.lastPrice-rate) / rate

	// if price change is less than threshold and orders are not expired, skip update
	// we use m.orderDuration/2 to give us some time to update market before orders will expire
	if change < m.cfg.Market.Threshold && m.lastMarketUpdate.Add(m.orderDuration/2).After(time.Now()) {
		m.log.Debug("Price change is within threshold, skipping update")
		return
	}

	orderBook, err := m.loadOrderBook()
	if err != nil {
		m.log.Errorf("Failed to load order book: %v", err)
		return
	}

	m.balanceMutex.Lock()
	defer m.balanceMutex.Unlock()

	if err := m.updateBalances(); err != nil {
		m.log.Errorf("Failed to update balances: %v", err)
		// continue anyway
	}

	cancelOps := m.createCancelOrders(orderBook)

	createOps, err := m.createOrders(price, orderBook)
	if err != nil {
		m.log.Errorf("Failed to update orders: %v", err)
	}

	ops := append(cancelOps, createOps...)

	if len(ops) > 0 {
		_, err = api.SignAndBroadcast(m.rpc, m.wallet.GetKeys(), m.feeAsset, ops...)

		if err != nil {
			m.log.Errorf("Failed to update market: %v", err)
		}
	}

	m.lastPrice = rate
	m.lastMarketUpdate = t
}

func (m *MarketMaker) dumpOrders(orders objects.LimitOrders) {
	m.log.Infof("Total orders: %d", len(orders))
	for _, o := range orders {
		m.log.Infof("%#v\n", o)
	}
}

func (m *MarketMaker) loadOrderBook() (OrderBook, error) {
	dbAPI, err := m.rpc.DatabaseAPI()
	if err != nil {
		return OrderBook{}, err
	}

	orders, err := dbAPI.GetLimitOrders(m.market.Base.ID, m.market.Quote.ID, orderBookDepth)
	if err != nil {
		return OrderBook{}, err
	}

	return NewOrderBook(FilterBySeller(orders, m.account.ID), &m.market, m.log), nil
}

func (m *MarketMaker) createCancelOrders(orderBook OrderBook) []objects.Operation {
	var ops []objects.Operation
	for _, o := range FilterBySeller(orderBook.Orders(), m.account.ID) {
		ops = append(ops, objects.NewLimitOrderCancelOperation(o.ID, o.Seller))
	}

	return ops
}

func reserveFee(balance *big.Float, feeAmount decimal.Decimal, precision int) *big.Float {
	fee := new(big.Float).SetInt64(feeAmount.Shift(int32(precision)).IntPart())
	result := new(big.Float)
	if balance.Cmp(fee) > 0 {
		result.Sub(balance, fee)
	}

	return result
}

func (m *MarketMaker) createOrders(price objects.Price, orderBook OrderBook) ([]objects.Operation, error) {
	rate := new(big.Float).Quo(
		new(big.Float).SetUint64(uint64(price.Base.Amount)),
		new(big.Float).SetUint64(uint64(price.Quote.Amount)))

	orderCount := new(big.Float).SetInt64(int64(m.cfg.Market.OrderCount))

	baseAvailable := new(big.Float).SetUint64(uint64(m.baseBalance.Amount) + orderBook.SellAmount())
	quoteAvailable := new(big.Float).SetUint64(uint64(m.quoteBalance.Amount) + orderBook.BuyAmount())

	if !m.cfg.FeeReserve.IsZero() {
		if m.market.Base.ID == m.feeAsset {
			baseAvailable = reserveFee(baseAvailable, m.cfg.FeeReserve, m.market.Base.Precision)
		}
		if m.market.Quote.ID == m.feeAsset {
			quoteAvailable = reserveFee(quoteAvailable, m.cfg.FeeReserve, m.market.Quote.Precision)
		}
	}

	quoteAvailable.Mul(quoteAvailable, rate)
	baseLimit := new(big.Float).SetUint64(uint64(m.market.Base.CreateAmount(m.cfg.Market.Amount).Amount))
	quoteLimit := baseLimit

	if baseLimit.Cmp(baseAvailable) > 0 {
		baseLimit = baseAvailable
	}

	if quoteLimit.Cmp(quoteAvailable) > 0 {
		quoteLimit = quoteAvailable
	}

	spread := m.cfg.Market.Spread
	expiration := objects.NewTime(time.Now().Add(m.orderDuration))

	sellOrderVolume := new(big.Float).Quo(baseLimit, orderCount)
	buyOrderVolume := new(big.Float).Quo(quoteLimit, orderCount)

	m.log.Infof("Order volume: sell=%s buy=%s",
		sellOrderVolume.String(), buyOrderVolume.String())

	totalSell := uint64(0)
	totalBuy := uint64(0)

	var ops []objects.Operation

	for i := 0; i < m.cfg.Market.OrderCount; i++ {
		spreadValue := big.NewFloat(1.0 + spread/2)

		{
			sellAmount, _ := sellOrderVolume.Uint64()
			recvAmountFloat := new(big.Float).Quo(sellOrderVolume, rate)
			recvAmountFloat.Mul(recvAmountFloat, spreadValue)
			recvAmount, _ := recvAmountFloat.Uint64()

			if sellAmount > orderAmountThreshold && recvAmount > orderAmountThreshold {
				sellOrder := &objects.LimitOrderCreateOperation{
					Seller: m.account.ID,
					AmountToSell: objects.AssetAmount{
						Asset:  m.market.Base.ID,
						Amount: objects.Int64(sellAmount)},
					MinToReceive: objects.AssetAmount{
						Asset:  m.market.Quote.ID,
						Amount: objects.Int64(recvAmount)},
					FillOrKill: false,
					Expiration: expiration,
					Extensions: objects.Extensions{},
				}

				m.log.Debugf("Sell order: sell=%d recv=%d", sellAmount, recvAmount)

				totalBuy += sellAmount
				ops = append(ops, sellOrder)
			}
		}

		{
			sellAmountFloat := new(big.Float).Quo(buyOrderVolume, rate)
			sellAmountFloat.Quo(sellAmountFloat, spreadValue)
			sellAmount, _ := sellAmountFloat.Uint64()
			recvAmount, _ := buyOrderVolume.Uint64()

			if sellAmount > orderAmountThreshold && recvAmount > orderAmountThreshold {
				buyOrder := &objects.LimitOrderCreateOperation{
					Seller: m.account.ID,
					AmountToSell: objects.AssetAmount{
						Asset:  m.market.Quote.ID,
						Amount: objects.Int64(sellAmount)},
					MinToReceive: objects.AssetAmount{
						Asset:  m.market.Base.ID,
						Amount: objects.Int64(recvAmount)},
					FillOrKill: false,
					Expiration: expiration,
					Extensions: objects.Extensions{},
				}

				m.log.Debugf("Buy order: sell=%d recv=%d", sellAmount, recvAmount)

				totalSell += recvAmount
				ops = append(ops, buyOrder)
			}
		}

		spread = spread + m.cfg.Market.SpreadStep
	}

	return ops, nil
}

func (m *MarketMaker) onMarketChange(msg interface{}) error {
	m.log.Infof("onMarketChange: %#v\n", msg)
	return nil
}

func (m *MarketMaker) loadObjects() error {
	dbAPI, err := m.rpc.DatabaseAPI()
	if err != nil {
		return errors.Annotate(err, "Failed to get dbAPI")
	}
	acc, err := dbAPI.GetAccountByName(m.cfg.Account)
	if err != nil {
		return errors.Annotate(err, "Failed to get account")
	}

	m.account = acc

	base := m.assetCache.GetBySymbol(m.cfg.Market.Base)
	if base == nil {
		return errors.NotFoundf("Asset %s", m.cfg.Market.Base)
	}

	quote := m.assetCache.GetBySymbol(m.cfg.Market.Quote)
	if quote == nil {
		return errors.NotFoundf("Asset %s", m.cfg.Market.Quote)
	}

	m.market.Base = *base
	m.market.Quote = *quote

	return nil
}

func (m *MarketMaker) updateBalances() error {
	dbAPI, err := m.rpc.DatabaseAPI()
	if err != nil {
		return err
	}

	balances, err := dbAPI.GetAccountBalances(m.account.ID, m.market.Base.ID, m.market.Quote.ID)
	if err != nil {
		return err
	}

	if balances[0].Asset == m.market.Base.ID {
		m.baseBalance = balances[0]
		m.quoteBalance = balances[1]
	} else {
		m.baseBalance = balances[1]
		m.quoteBalance = balances[0]
	}

	baseBalance := m.market.Base.GetRate(m.baseBalance)
	quoteBalance := m.market.Quote.GetRate(m.quoteBalance)

	m.log.Infof("Balance base=%f quote=%f", baseBalance, quoteBalance)

	return nil
}

func (m *MarketMaker) Start() error {
	m.ticker = time.NewTicker(m.cfg.UpdateInterval)

	if err := m.loadObjects(); err != nil {
		return err
	}

	if err := m.updateBalances(); err != nil {
		return err
	}

	pp, err := m.factory.GetProvider(&m.market)
	if err != nil {
		return err
	}
	m.priceProvider = pp

	go m.worker()
	return nil
}

func (m *MarketMaker) Stop() {
	if m.ticker != nil {
		m.ticker.Stop()
	}
}

func NewMarketMaker(
	cfg *Config,
	rpc api.BitsharesAPI,
	wallet wallet.Wallet,
	factory PriceProviderFactory,
	logger *zap.SugaredLogger,
	balanceMutex *sync.Mutex,
) *MarketMaker {
	dbAPI, err := rpc.DatabaseAPI()
	if err != nil {
		logger.Fatalf("Unable to get database API")
	}

	return &MarketMaker{
		rpc:           rpc,
		log:           logger.With("base", cfg.Market.Base, "quote", cfg.Market.Quote),
		assetCache:    api.NewAssetCache(dbAPI),
		cfg:           cfg,
		balanceMutex:  balanceMutex,
		wallet:        wallet,
		factory:       factory,
		feeAsset:      *objects.NewGrapheneID("1.3.0"),
		orderDuration: time.Duration(cfg.Market.Expiration) * time.Second,
	}
}
