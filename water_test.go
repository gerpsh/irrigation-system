package main

import (
	"testing"
	"time"
)

func TestIsWaterTimepoint(t *testing.T) {
	config := &Config{
		WaterTimepoints: map[string][]*WaterTimepoint{
			"primary": []*WaterTimepoint{
				&WaterTimepoint{Hour: 9, Minute: 1, Days: []int{0, 1, 2, 3, 4, 5, 6}},
			},
		},
	}

	tp := time.Date(2024, 6, 17, 9, 1, 0, 0, time.Local)

	is := IsPrimaryWaterTimepoint(config, tp)
	if !is {
		t.Errorf("expected time point to be true, returned false")
	}
}
