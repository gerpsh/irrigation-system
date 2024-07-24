package main

import (
	"testing"
	"time"
)

func TestPushover(t *testing.T) {
	config, err := ReadConfig("/home/shaefferg/code/go/src/github.com/gerpsh/irrigation-system/config.json")
	if err != nil {
		t.Errorf("could not read config: %v", err)
	}
	if config.UsePushover {
		le := &LogEntry{
			Type:      "event",
			Timestamp: time.Now(),
			Message:   "Test",
		}

		err = PushNotif(config, le)
		if err != nil {
			t.Errorf("could not send push notif: %v", err)
		}
	}
}

func TestLogDB(t *testing.T) {
	config, err := ReadConfig("/home/shaefferg/code/go/src/github.com/gerpsh/irrigation-system/config.json")
	if err != nil {
		t.Errorf("could not read config: %v", err)
	}
	cw := &CurrentWeather{
		Temp:     0.0,
		IsDay:    0,
		Humidity: 0.0,
		Precip:   0.0,
		Condition: &WeatherCondition{
			Text: "Test",
			Code: 0,
		},
	}

	wd := &WeatherData{
		Current:      cw,
		PastPrecip:   0.0,
		FuturePrecip: 0.0,
	}

	if config.UseDBLog {
		err = config.Valves[0].LogEvent(config, wd, "test", false)
		if err != nil {
			t.Errorf("could not log to db: %v", err)
		}
	}
}
