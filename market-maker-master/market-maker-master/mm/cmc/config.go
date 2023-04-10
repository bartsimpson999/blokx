package cmc

type Config struct {
	// Coinmarketcap API URL
	URL string
	// Number of tickers to get in one request
	BulkSize int
	// Interval for polling
	Interval string
}
