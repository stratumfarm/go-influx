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
