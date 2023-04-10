package main

import (
	"math"
	"time"
)

type PriceFilterConfig struct {
	Period    int     `json:"period"`
	Threshold float64 `json:"threshold"`
}

// PriceFilter implements moving average over specified period of time,
// filtering out values that are off from average more than (avg*threshold)
type PriceFilter struct {
	period    time.Duration
	threshold float64
	sum       float64
	values    []pricePoint
}

type pricePoint struct {
	Price     float64
	Timestamp time.Time
}

func NewPriceFilter(cfg *PriceFilterConfig) *PriceFilter {
	return &PriceFilter{
		period:    time.Duration(cfg.Period) * time.Second,
		threshold: cfg.Threshold,
	}
}

func (p *PriceFilter) Add(price float64, timestamp time.Time) bool {
	// Zero price is invalid
	if price == 0 {
		return false
	}

	cutoff := timestamp.Add(-p.period)

	for len(p.values) > 0 {
		if p.values[0].Timestamp.Before(cutoff) {
			p.sum -= p.values[0].Price
			p.values = p.values[1:]
		} else {
			break
		}
	}

	shouldReport := true
	avg := p.Avg()
	if !math.IsNaN(avg) {
		if price > avg*(1+p.threshold) || price < avg*(1-p.threshold) {
			shouldReport = false
		}
	}

	p.values = append(p.values, pricePoint{price, timestamp})
	p.sum += price

	return shouldReport
}

func (p *PriceFilter) Avg() float64 {
	if len(p.values) == 0 {
		return math.NaN()
	}

	return p.sum / float64(len(p.values))
}
