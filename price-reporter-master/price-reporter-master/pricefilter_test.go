package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPriceFilterAllWithin(t *testing.T) {
	f := NewPriceFilter(&PriceFilterConfig{Period: 10, Threshold: 0.2})
	now := time.Now()

	assert.True(t, f.Add(1, now))
	assert.True(t, f.Add(1, now))
	assert.True(t, f.Add(1, now))

	assert.Equal(t, float64(1.0), f.Avg())

	assert.True(t, f.Add(1.2, now))

	assert.Equal(t, float64(1.05), f.Avg())

	// 1.05*1.2 = 1.26
	assert.False(t, f.Add(1.27, now))

	// Avg = (1*3 + 1.2 + 1.27) / 5 = 1.094
	assert.Equal(t, float64(1.094), f.Avg())
}

func TestPriceFilterMovingWindow(t *testing.T) {
	f := NewPriceFilter(&PriceFilterConfig{Period: 10, Threshold: 0.5})
	now := time.Now()

	assert.True(t, f.Add(1, now))
	assert.True(t, f.Add(1, now))

	// Prev avg is out of the time frame
	assert.True(t, f.Add(2, now.Add(11*time.Second)))
	assert.Equal(t, 2.0, f.Avg())

	assert.True(t, f.Add(3, now.Add(12*time.Second)))
	assert.Equal(t, 2.5, f.Avg())

	assert.False(t, f.Add(3.76, now.Add(13*time.Second)))
}
