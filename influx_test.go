package influx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	cfg := Config{
		Endpoint:  "http://127.0.0.1:8086",
		Precision: "incorrect",
	}
	assert.Panics(t, func() {
		_, e := NewWriter(cfg)
		assert.NoError(t, e)
	})

	cfg.Precision = "ms"
	assert.Panics(t, func() {
		_, e := NewWriter(cfg)
		assert.NoError(t, e)
	})

	cfg.BatchInterval = "1s"
	_, e := NewWriter(cfg)
	assert.NoError(t, e)
}
