package main

import (
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/stianeikeland/go-rpio/v4"
)

type Valve struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Pin            int    `json:"pin"`
	ShortWaterTime int    `json:"short_water_time"`
	LongWaterTime  int    `json:"long_water_time"`
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
func IsPrimaryWaterTimepoint(c *Config, t time.Time) bool {
	for _, tp := range c.WaterTimepoints["primary"] {

		if (tp.Hour == t.Local().Hour()) &&
			(tp.Minute == t.Local().Minute()) &&
			slices.Contains(tp.Days, int(t.Local().Weekday())) {
			return true
		}
		fmt.Println()
	}
	return false
}

// check if the time is within a primary watering timepoint
func IsSecondaryWaterTimepoint(c *Config, t time.Time) bool {
	for _, tp := range c.WaterTimepoints["secondary"] {
		if (tp.Hour == t.Local().Hour()) &&
			(tp.Minute == t.Local().Minute()) &&
			slices.Contains(tp.Days, int(t.Local().Weekday())) {
			return true
		}
	}
	return false
}

// water for specified amount of time at secondary timepoint
func (v *Valve) ShortWater(c *Config) error {
	err := v.Water(c, v.ShortWaterTime)
	if err != nil {
		return err
	}
	return nil
}

// water for specified amount of time at primary timepoint
func (v *Valve) LongWater(c *Config) error {
	err := v.Water(c, v.LongWaterTime)
	if err != nil {
		return err
	}
	return nil
}

// Water all valves for 'long' amount of time
func LongWaterAll(c *Config, cw *CurrentWeather) error {
	for _, v := range c.Valves {
		err := v.LongWater(c)
		if err != nil {
			logerr := LogError(c, err)
			if logerr != nil {
				log.Printf("could not log error: %v", logerr)
			}
			return fmt.Errorf("could not water on valve %v: %v", v.ID, err)
		}
		err = v.LogEvent(c, cw, "long")
		if err != nil {
			logerr := LogError(c, err)
			if logerr != nil {
				log.Printf("could not log error: %v", logerr)
			}
			return fmt.Errorf("could not log event: %v", err)
		}
		le := &LogEntry{
			Type:      "event",
			Timestamp: time.Now(),
			Message:   FormatEventMessage(cw, "long", v.ID, v.Name),
		}
		err = PushNotif(c, le)
		if err != nil {
			_ = LogError(c, err)
			return fmt.Errorf("could not send push notif: %v", err)
		}
	}
	time.Sleep(time.Minute * time.Duration(1))
	return nil
}

// Water all valves for 'short' amount of time
func ShortWaterAll(c *Config, cw *CurrentWeather) error {
	for _, v := range c.Valves {
		err := v.ShortWater(c)
		if err != nil {
			logerr := LogError(c, err)
			if logerr != nil {
				log.Printf("could not log error: %v", logerr)
			}
			return fmt.Errorf("could not short water on valve %v: %v", v.ID, err)
		}
		err = v.LogEvent(c, cw, "short")
		if err != nil {
			logerr := LogError(c, err)
			if logerr != nil {
				log.Printf("could not log error: %v", logerr)
			}
			return fmt.Errorf("could not log event: %v", err)
		}
		le := &LogEntry{
			Type:      "event",
			Timestamp: time.Now(),
			Message:   FormatEventMessage(cw, "short", v.ID, v.Name),
		}
		err = PushNotif(c, le)
		if err != nil {
			logerr := LogError(c, err)
			if logerr != nil {
				log.Printf("could not log error: %v", logerr)
			}
			return fmt.Errorf("could not send push notif: %v", err)
		}
	}

	return nil
}
