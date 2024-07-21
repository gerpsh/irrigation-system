package main

import (
	"fmt"
	"log"
	"time"
)

/*
Watering happens in two different scenarios:
1. At a primary timepoint, where there will ALWAYS be watering,
unless there will a certain amount of rain within a certain timeframe surrounding the timepoint,
as defined in the config

2. At a secondary timepoint, where there will NEVER be watering,
unless the weather is especially hot and dry, as defined in the config.

The rules for gauging rainfall are:
If there has been <config.PastRainThreshold> mm of rainfall in the past <config.RainLookback> hours,
or it is forecasted to rain <config.FutureRainThreshold> mm in the next <config.RainLookahead> hours,
then do not water.  Otherwise, proceed as usual.

The rules for deciding to water at a secondary timepoint are:
If it is currently <config.HotThreshold> degrees F or higher,
and below <config.DryThreshold>,
water during the secondary timepoint

Without using weather data, the system essentially runs on a timer,
with watering occuring at every primary timepoint, and none of the secondary timepoints

The main routine is an infinite loop, where on each loop the time is checked.
If we're currently in a timepoint as defined in the config,
we'll check the weather if applicable and decide whether or not to water
*/

func main() {
	config, err := ReadConfig("/home/shaefferg/code/go/src/github.com/gerpsh/irrigation-system/config.json")
	if err != nil {
		log.Fatalf("could not read config: %v", err)
	}
	err = config.CheckConfig()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("running...")
	for {
		waterTime := 0
		// create nil pointer to weather data, fill it later if we're using weather
		var weather *WeatherData
		now := time.Now()
		for _, v := range config.Valves {
			if is, tp := v.IsWaterTimepoint(config, now); is {
				_ = config.OnlineCheck()
				weather, err = GetWeatherTimeline(config)
				if err != nil {
					logerr := LogError(config, fmt.Errorf("could not create weather timeline: %v", err))
					if logerr != nil {
						log.Printf("could not log error: %v\n", err)
					}
					log.Fatalf("could not create weather timeline: %v", err)
				}
				if ShouldWater(config, weather, tp) {
					err = v.Water(config, tp.Duration)
					if err != nil {
						logerr := LogError(config, fmt.Errorf("could not water on valve %v (%v): %v", v.ID, v.Name, err))
						if logerr != nil {
							log.Printf("could not log error: %v\n", err)
						}
						log.Fatalf("could not create weather timeline: %v", err)
					}
					waterTime += tp.Duration
					err = v.LogEvent(config, weather.Current, fmt.Sprintf("%v", tp.Duration))
					if err != nil {
						logerr := LogError(config, err)
						if logerr != nil {
							log.Printf("could not log error: %v", logerr)
						}
					}
				}
			}
		}
		// if we're still meeting timepoint criteria after watering,
		// wait until the minute has passed so we don't do multiple waters
		// on a single valve
		if waterTime > 0 && waterTime < 60 {
			time.Sleep(time.Duration(60-waterTime) * time.Second)
		}
		time.Sleep(time.Second * time.Duration(1))
	}
}
