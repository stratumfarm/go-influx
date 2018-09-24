package influx

import (
	"time"
)

//Metric interface
type Metric interface {
	Measurement() string
	Tags() map[string]string
	Values() map[string]interface{}
	Time() time.Time
}

// SimpleMetric is a simple struct that doesn't produce any calculatons just return it as-is
type SimpleMetric struct {
	Name       string
	TagsMap    map[string]string
	ValuesMap  map[string]interface{}
	CreateTime time.Time
}

// Measurement returns Measurement name
func (r SimpleMetric) Measurement() string {
	return r.Name
}

// Tags returns  prepared tags
func (r SimpleMetric) Tags() map[string]string {
	return r.TagsMap
}

// Values returns prepared values
func (r SimpleMetric) Values() map[string]interface{} {
	return r.ValuesMap
}

//Time return time for the value
func (r SimpleMetric) Time() time.Time {
	return r.CreateTime
}
