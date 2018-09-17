package influx

import "time"

// NewRuntimeCollector starts goroutine which write to influx writer every duration
func NewRuntimeCollector(influx *Writer, d time.Duration) {
	r := NewRegistry()
	RegisterRuntimeMemStats(r)
	go func() {
		for range time.Tick(d) {
			CaptureRuntimeMemStatsOnce(r)
			influx.Write(newRuntimeMetric(r))
		}
	}()
}
