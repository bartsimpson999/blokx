package mm

import (
	"math/big"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestReserveFee(t *testing.T) {
	balance := big.NewFloat(280)

	assert.Equal(t, big.NewFloat(279).String(),
		reserveFee(balance, decimal.RequireFromString("1"), 0).String())

	assert.Equal(t, big.NewFloat(270).String(),
		reserveFee(balance, decimal.RequireFromString("10"), 0).String())

	assert.Equal(t, big.NewFloat(180).String(),
		reserveFee(balance, decimal.RequireFromString("10"), 1).String())

	assert.Equal(t, big.NewFloat(279).String(),
		reserveFee(balance, decimal.RequireFromString("0.1"), 1).String())

	assert.Equal(t, big.NewFloat(279).String(),
		reserveFee(balance, decimal.RequireFromString("0.1"), 1).String())

	assert.Equal(t, big.NewFloat(278).String(),
		reserveFee(balance, decimal.RequireFromString("0.02"), 2).String())

	// Fee is > than balance
	assert.Equal(t, big.NewFloat(0).String(),
		reserveFee(balance, decimal.RequireFromString("0.02"), 8).String())

	assert.Equal(t, big.NewFloat(0).String(),
		reserveFee(balance, decimal.RequireFromString("280.01"), 0).String())

	// Fee equals balance
	assert.Equal(t, big.NewFloat(0).String(),
		reserveFee(balance, decimal.RequireFromString("280"), 0).String())
}
