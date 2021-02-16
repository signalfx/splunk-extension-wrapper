package util

import (
	"testing"
	"time"
)

var fakeNow = time.Now()

type fakeClock struct{}

func (fc fakeClock) now() time.Time {
	return fakeNow
}

func TestTicker(t *testing.T) {
	fakeNow = time.Now()

	ticker := newTicker(15*time.Second, fakeClock{})

	if !ticker.Tick() {
		t.Error("it was expected to tick")
	}

	fakeNow = fakeNow.Add(time.Minute)

	if !ticker.Tick() {
		t.Errorf("it was expected to tick at least once")
	}

	if ticker.Tick() {
		t.Errorf("it shouldn't cumulate ticks")
	}

	fakeNow = fakeNow.Add(time.Minute)

	if !ticker.Tick() {
		t.Errorf("it was expected to tick once again")
	}
}
