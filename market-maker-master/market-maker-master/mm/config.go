package mm

type MarketConfig struct {
	Base       string  `json:"base"`
	Quote      string  `json:"quote"`
	Spread     float64 `json:"spread"`
	Threshold  float64 `json:"threshold"`
	Expiration int     `json:"expiration"`
	Amount     float64 `json:"amount"`
	OrderCount int     `json:"orders"`
	SpreadStep float64 `json:"spread_step"`
}
