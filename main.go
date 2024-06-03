package main

import (
	"fmt"
	"log"
	"time"
)

func main() {
	config, err := ReadConfig("/home/shaefferg/code/go/src/github.com/gerpsh/irrigation-system/config.json")
	if err != nil {
		log.Fatalf("could not read config: %v", err)
	}

	log.Println("running...")
	for {
		now := time.Now()
		totalWateringTime := 0
		for _, tp := range config.WaterTimepoints["primary"] {
			// if it's currently a primary watering time
			if now.Hour() == tp["hour"] && now.Minute() == tp["minute"] {
				if config.UseWeather {
					// get relevant weather data if we're using weather
					weather, err := GetWeatherTimeline(config)
					if err != nil {
						_ = LogError(config, fmt.Errorf("could not create weather timeline: %v", err))
						log.Fatalf("could not create weather timeline: %v", err)
					}
					// if we should water based on the weather, do it
					if ShouldWaterPrimary(config, weather) {
						for _, v := range config.Valves {
							err = v.LongWater(config)
							if err != nil {
								_ = LogError(config, fmt.Errorf("could not water on valve %v: %v", v.ID, err))
								log.Fatalf("could not water on valve %v: %v", v.ID, err)
							}
							totalWateringTime += v.LongWaterTime
						}
						err = LogEvent(config, weather.Current, "long")
						if err != nil {
							log.Printf("could not log watering event: %v\n", err)
						}
					}
					// if we're not using the weather, just water if it's a primary timepoint
				} else {
					for _, v := range config.Valves {
						err = v.LongWater(config)
						if err != nil {
							_ = LogError(config, fmt.Errorf("could not water on valve %v: %v", v.ID, err))
							log.Fatalf("could not water on valve %v: %v", v.ID, err)
						}
						totalWateringTime += v.LongWaterTime
					}
					err = LogEvent(config, nil, "long")
					if err != nil {
						log.Printf("could not log watering event: %v\n", err)
					}
				}
			}
		}
		for _, tp := range config.WaterTimepoints["secondary"] {
			// if it's a secondary watering timepoint, check the weather and water if necessary
			// otherwise, don't water
			if now.Hour() == tp["hour"] && now.Minute() == tp["minute"] {
				if config.UseWeather {
					weather, err := GetWeatherTimeline(config)
					if err != nil {
						_ = LogError(config, fmt.Errorf("could not create weather timeline: %v", err))
						log.Fatalf("could not create weather timeline: %v", err)
					}
					if ShouldWaterSecondary(config, weather) {
						for _, v := range config.Valves {
							err = v.ShortWater(config)
							if err != nil {
								_ = LogError(config, fmt.Errorf("could not water on valve %v: %v", v.ID, err))
								log.Fatalf("could not water on valve %v: %v", v.ID, err)
							}
							totalWateringTime += v.ShortWaterTime
						}
						err = LogEvent(config, weather.Current, "short")
						if err != nil {
							log.Printf("could not log watering event: %v\n", err)
						}
					}
				}
			}
		}
		// if we're still in the hour and minute of a scheduled water, wait a bit longer for the minute to change
		// so that we don't water twice in the same minute
		if totalWateringTime < 60 {
			time.Sleep(time.Second * time.Duration(60))
			// otherwise, continue to frequently check if it's time to water
		} else {
			time.Sleep(time.Second * time.Duration(3))
		}
	}
}
