package main

import (
	"fmt"
	"slices"
	"time"

	"github.com/stianeikeland/go-rpio/v4"
)

type WaterTimepoint struct {
	Days     []int  `json:"days"`
	Hour     int    `json:"hour"`
	Minute   int    `json:"minute"`
	Type     string `json:"type"`
	Duration int    `json:"duration"`
}

type Valve struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Pin        int               `json:"pin"`
	Timepoints []*WaterTimepoint `json:"timepoints"`
}

// activate valve-connected gpio pin, keep output on for specified time
func (v *Valve) Water(c *Config, duration int) error {
	err := rpio.Open()
	if err != nil {
		return fmt.Errorf("could not open gpio memory range: %v", err)
	}

	pin := rpio.Pin(v.Pin)
	pin.Output()
	pin.High()

	time.Sleep(time.Second * time.Duration(duration))
	pin.Low()

	err = rpio.Close()
	if err != nil {
		return fmt.Errorf("could not close gpio: %v", err)
	}

	return nil
}

// check if the time is within a primary watering timepoint
func (v *Valve) IsWaterTimepoint(c *Config, t time.Time) (bool, string, int) {
	for _, tp := range v.Timepoints {
		if (tp.Hour == t.Local().Hour()) &&
			(tp.Minute == t.Local().Minute()) &&
			slices.Contains(tp.Days, int(t.Local().Weekday())) {
			return true, tp.Type, tp.Duration
		}
		fmt.Println()
	}
	return false, "", 0
}
