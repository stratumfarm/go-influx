package influx

import "time"

func mustParseDuration(s string) time.Duration {
	value, err := time.ParseDuration(s)
	if err != nil {
		panic("[NEVER] Can't parse duration `" + s + "`: " + err.Error())
	}
	return value
}
