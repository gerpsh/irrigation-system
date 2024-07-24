package main

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

// test whether types are set up correctly
func TestParsing(t *testing.T) {
	h, err := os.ReadFile("./fixtures/history.json")
	if err != nil {
		t.Errorf("could not read history file: %v", err)
	}
	f, err := os.ReadFile("./fixtures/forecast.json")
	if err != nil {
		t.Errorf("could not read forecast file: %v", err)
	}

	var history WeatherForecastResponse
	err = json.Unmarshal(h, &history)
	if err != nil {
		t.Errorf("could unmarshal history file: %v", err)
	}
	var forecast WeatherForecastResponse
	err = json.Unmarshal(f, &forecast)
	if err != nil {
		t.Errorf("could unmarshal forecast file: %v", err)
	}
}

func TestParseWeatherTimeline(t *testing.T) {
	h, err := os.ReadFile("./fixtures/history.json")
	if err != nil {
		t.Errorf("could not read history file: %v", err)
	}
	f, err := os.ReadFile("./fixtures/forecast.json")
	if err != nil {
		t.Errorf("could not read forecast file: %v", err)
	}

	var history WeatherForecastResponse
	err = json.Unmarshal(h, &history)
	if err != nil {
		t.Errorf("could unmarshal history file: %v", err)
	}
	var forecast WeatherForecastResponse
	err = json.Unmarshal(f, &forecast)
	if err != nil {
		t.Errorf("could unmarshal forecast file: %v", err)
	}

	timeline := append(history.Forecast.Days[0].Hours, forecast.Forecast.Days[0].Hours...)

	now := time.Date(2024, 5, 31, 3, 0, 0, 0, time.Local)
	c := &Config{
		RainLookahead: 6,
		RainLookback:  6,
		RainThreshold: 7,
		HotThreshold:  80,
	}
	weather := ParseWeatherTimeline(c, now, timeline)
	if weather.PastPrecip != 15.0 {
		t.Errorf("past precipitation value incorrect, expected 15.0, got %v", weather.PastPrecip)
	}
	if weather.FuturePrecip != 6.0 {
		t.Errorf("future precipation value incorrect, expected 5.0, got %v", weather.FuturePrecip)
	}
}

func TestShouldWater(t *testing.T) {
	c := &Config{
		RainThreshold: 12.0,
		HotThreshold:  80,
	}

	weather := &WeatherData{
		Current: &CurrentWeather{
			Temp:     81,
			Humidity: 50,
			Condition: &WeatherCondition{
				Text: "Sunny/Clear",
				Code: 1000,
			},
		},
		PastPrecip:   5,
		FuturePrecip: 5,
	}

	if ShouldWaterPrimary(c, weather) != true {
		t.Error("ShouldWaterPrimary with weather returned false, expected true")
	}

	if ShouldWaterPrimary(c, nil) != true {
		t.Error("ShouldWaterPrimary without weather returned false, expected true")
	}

	if ShouldWaterSecondary(c, weather) != true {
		t.Error("ShouldWaterSecondary with weather returned false, expected true")
	}

	if ShouldWaterSecondary(c, nil) != false {
		t.Error("ShouldWaterSecondary without weather returned true, expected false")
	}

	weather = &WeatherData{
		Current: &CurrentWeather{
			Temp:     75,
			Humidity: 50,
			Condition: &WeatherCondition{
				Text: "Sunny/Clear",
				Code: 1000,
			},
		},
		PastPrecip:   20,
		FuturePrecip: 20,
	}

	if ShouldWaterPrimary(c, weather) != false {
		t.Error("ShouldWaterPrimary with weather return true, expected false")
	}

	if ShouldWaterSecondary(c, weather) != false {
		t.Error("ShouldWaterSecondary with weather return false, expected true")
	}
}

func TestGetWeatherForecast(t *testing.T) {
	config, err := ReadConfig("/home/shaefferg/code/go/src/github.com/gerpsh/irrigation-system/config.json")
	if err != nil {
		t.Errorf("could not read config: %v", err)
	}

	_, err = GetWeatherForecast(config)
	if err != nil {
		t.Errorf("could get weather forecast: %v", err)
	}

}

func TestGetWeatherHistory(t *testing.T) {
	config, err := ReadConfig("/home/shaefferg/code/go/src/github.com/gerpsh/irrigation-system/config.json")
	if err != nil {
		t.Errorf("could not read config: %v", err)
	}

	_, err = GetWeatherHistory(config, time.Now())
	if err != nil {
		t.Errorf("could get weather forecast: %v", err)
	}
}

func TestGetWeatherTimeline(t *testing.T) {
	config, err := ReadConfig("/home/shaefferg/code/go/src/github.com/gerpsh/irrigation-system/config.json")
	if err != nil {
		t.Errorf("could not read config: %v", err)
	}

	_, err = GetWeatherTimeline(config)
	if err != nil {
		t.Errorf("could not get weather timeline: %v", err)
	}
}
