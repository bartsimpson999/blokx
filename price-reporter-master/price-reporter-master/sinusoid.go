package main

import (
	"math"
	"time"
)

func GetSinusoidPrice(cfg *SinusoidConfig, at time.Time) float64 {
	pos := at.Unix() % cfg.Period
	radian := float64(pos) / float64(cfg.Period) * math.Pi * 2
	amplitude := (cfg.Max - cfg.Min) / 2
	return math.Sin(radian)*amplitude + cfg.Min + amplitude
}
