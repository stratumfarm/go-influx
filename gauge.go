package influx

import "sync/atomic"

type standardGauge struct {
	value int64
}

// Gauge hold an int64 value that can be set arbitrarily.
type Gauge interface {
	Update(int64)
	Value() int64
}

// Update updates the gauge's value.
func (g *standardGauge) Update(v int64) {
	atomic.StoreInt64(&g.value, v)
}

// Value returns the gauge's current value.
func (g *standardGauge) Value() int64 {
	return atomic.LoadInt64(&g.value)
}

// NewGauge constructs a new StandardGauge.
func NewGauge() Gauge {
	return &standardGauge{0}
}
