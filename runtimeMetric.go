package influx

import "time"

type runtimeMetric struct {
	tags   map[string]string
	values map[string]interface{}
	time   time.Time
}

func (r runtimeMetric) Measurement() string {
	return "gopprof"
}

func (r runtimeMetric) Tags() map[string]string {
	return r.tags
}

func (r runtimeMetric) Values() map[string]interface{} {
	return r.values
}

func (r runtimeMetric) Time() time.Time {
	return r.time
}
func newRuntimeMetric(r Registry) Metric {
	tm := time.Now().UTC()
	values := make(map[string]interface{})
	r.Each(func(name string, value interface{}) {
		switch metric := value.(type) {
		case Gauge:
			values[name] = metric.Value()
		case Counter:
			values[name] = metric.Count()
		}
	})
	return runtimeMetric{
		time:   tm,
		values: values,
		tags:   nil,
	}
}
