package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetSinusoidPrice(t *testing.T) {
	cfg := &SinusoidConfig{
		Asset:  "1.3.1",
		Min:    1,
		Max:    2,
		Period: 60,
	}

	median := cfg.Min + (cfg.Max-cfg.Min)/2
	at := time.Unix(cfg.Period, 0)
	assert.Equal(t, median, GetSinusoidPrice(cfg, at))
	assert.Equal(t, cfg.Max, GetSinusoidPrice(cfg, at.Add(15*time.Second)))
	assert.Equal(t, cfg.Min, GetSinusoidPrice(cfg, at.Add(45*time.Second)))
	assert.Equal(t, median, GetSinusoidPrice(cfg, at.Add(60*time.Second)))
}
