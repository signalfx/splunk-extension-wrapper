package util

import "time"

type clock interface {
	now() time.Time
}

type systemClock struct{}

func (sc systemClock) now() time.Time {
	return time.Now()
}
