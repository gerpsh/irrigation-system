package main

import (
	"fmt"
	"time"

	"github.com/stianeikeland/go-rpio/v4"
)

type Valve struct {
	ID             string `json:"id"`
	Pin            int    `json:"pin"`
	ShortWaterTime int    `json:"short_water_time"`
	LongWaterTime  int    `json:"long_water_time"`
	State          bool
}

// activate specified gpio pin, keep output on for specified time
func (v *Valve) Water(c *Config, long bool) error {
	err := rpio.Open()
	if err != nil {
		return fmt.Errorf("could not open gpio memory range: %v", err)
	}

	pin := rpio.Pin(v.Pin)
	pin.Output()
	pin.High()
	if long {
		time.Sleep(time.Second * time.Duration(v.LongWaterTime))
	} else {
		time.Sleep(time.Second * time.Duration(v.ShortWaterTime))
	}
	pin.Low()
	err = rpio.Close()
	if err != nil {
		return fmt.Errorf("could not close gpio: %v", err)
	}
	return nil
}

// check if the time is within a primary watering timepoint
func IsPrimaryWaterTimepoint(c *Config, t time.Time) bool {
	for _, tp := range c.WaterTimepoints["primary"] {
		if (tp["hour"] == t.Local().Hour()) && (tp["minute"] == t.Local().Minute()) {
			return true
		}
	}
	return false
}

// check if the time is within a primary watering timepoint
func IsSecondaryWaterTimepoint(c *Config, t time.Time) bool {
	for _, tp := range c.WaterTimepoints["secondary"] {
		if (tp["hour"] == t.Local().Hour()) && (tp["minute"] == t.Local().Minute()) {
			return true
		}
	}
	return false
}

func (v *Valve) ShortWater(c *Config) error {
	err := v.Water(c, false)
	if err != nil {
		return err
	}
	return nil
}

func (v *Valve) LongWater(c *Config) error {
	err := v.Water(c, true)
	if err != nil {
		return err
	}
	return nil
}
