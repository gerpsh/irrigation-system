package main

import (
	"fmt"
	"slices"
	"time"

	"github.com/stianeikeland/go-rpio/v4"
)

type WaterTimepoint struct {
	Days     []int  `json:"days"`     // days of week (0-6) to water
	Hour     int    `json:"hour"`     // hour of water timepoint (0-23)
	Minute   int    `json:"minute"`   // minute of water timpoint (0-59)
	Type     string `json:"type"`     // type of water, primary or secondary
	Duration int    `json:"duration"` // amount of time to water in seconds
}

type Valve struct {
	ID         string            `json:"id"`         // id of valve, arbitrary, used for logging
	Name       string            `json:"name"`       // string name of valve, arbitrary, used for logging
	Pin        int               `json:"pin"`        // gpio pin # that controls the valve, pinctrl convention
	Timepoints []*WaterTimepoint `json:"timepoints"` // list of timepoints that describes the water schedule for the valve
}

// activate valve-connected gpio pin, keep output on for specified duration
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
func (v *Valve) IsWaterTimepoint(c *Config, t time.Time) (bool, *WaterTimepoint) {
	for _, tp := range v.Timepoints {
		if (tp.Hour == t.Local().Hour()) &&
			(tp.Minute == t.Local().Minute()) &&
			slices.Contains(tp.Days, int(t.Local().Weekday())) {
			return true, tp
		}
	}
	return false, nil
}
