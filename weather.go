package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type WeatherCondition struct {
	Text string
	Code int
}

// current condition in weather api response
type CurrentWeather struct {
	Temp      float64           `json:"temp_f"`
	IsDay     int               `json:"is_day"`
	Humidity  int               `json:"humidity"`
	Precip    float64           `json:"precip_in"`
	Condition *WeatherCondition `json:"condition"`
}

type WeatherTime struct {
	time.Time
}

// special parse for weather api times
func (wt *WeatherTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	if s == "null" {
		wt.Time = time.Time{}
	}
	t, err := time.ParseInLocation("2006-01-02 15:04", s, time.Local)
	if err != nil {
		return err
	}
	wt.Time = t
	return nil
}

// individual weather hour in weather api response
type WeatherHour struct {
	Time     WeatherTime `json:"time"`
	PrecipMM float64     `json:"precip_mm"`
}

type Date struct {
	time.Time
}

// special parse for weather api response dates
func (d *Date) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	if s == "null" {
		d.Time = time.Time{}
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return err
	}
	d.Time = t
	return nil
}

// individual day within forecast response
type ForecastDay struct {
	Date  Date           `json:"date"`
	Hours []*WeatherHour `json:"hour"`
}

// aggregate of forecast days
type ForecastSection struct {
	Days []*ForecastDay `json:"forecastDay"`
}

type WeatherForecastResponse struct {
	CurrentWeather *CurrentWeather  `json:"current"`
	Forecast       *ForecastSection `json:"forecast"`
}

// abstracted weather data, derived from forecast, history responses
type WeatherData struct {
	Current      *CurrentWeather
	PastPrecip   float64 // number of mm in lookback period
	FuturePrecip float64 // number of mm in lookahead period
}

// Parse hourly data from weather api responses to determine past and projected precipitation
func ParseWeatherTimeline(c *Config, now time.Time, tps []*WeatherHour) *WeatherData {
	pastSum := 0.000
	futureSum := 0.000

	for _, tp := range tps {
		if tp.Time.After(now.Add(time.Duration(-c.RainLookback-1)*time.Hour)) && tp.Time.Before(now) {
			pastSum += tp.PrecipMM
		}

		if tp.Time.Before(now.Add(time.Duration(c.RainLookahead)*time.Hour)) && tp.Time.After(now) {
			futureSum += tp.PrecipMM
		}
	}

	return &WeatherData{
		PastPrecip:   pastSum,
		FuturePrecip: futureSum,
	}
}

// Fetch and parse today's and tomorrow's weather forecast from weather api
func GetWeatherForecast(c *Config) (*WeatherForecastResponse, error) {
	// forecast gets todays weather and tomorrow's
	resp, err := http.Get(c.WeatherForecastUrl)
	if err != nil {
		return nil, fmt.Errorf("could not fetch weather forecast: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read forecast response body: %v", err)
	}
	var weather WeatherForecastResponse
	err = json.Unmarshal(body, &weather)
	if err != nil {
		return nil, fmt.Errorf("could not parse forecast response: %v", err)
	}
	return &weather, nil
}

// Fetch and parse weather history request from weather API
func GetWeatherHistory(c *Config, now time.Time) (*WeatherForecastResponse, error) {
	yesterdayStr := now.Add(time.Hour * time.Duration(-24)).Format("2006-01-02")
	formattedUrl := strings.ReplaceAll(c.WeatherHistoryUrl, "{}", yesterdayStr)
	resp, err := http.Get(formattedUrl)
	if err != nil {
		return nil, fmt.Errorf("could not fetch weather history: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read weather history response body: %v", err)
	}

	// history responses are identical to forecast responses, except for no current weather
	var history WeatherForecastResponse
	err = json.Unmarshal(body, &history)
	if err != nil {
		return nil, fmt.Errorf("could not parse history response: %v", err)
	}
	return &history, nil
}

// get amount of precipitation for lookback + lookahead interval, along with current weather
func GetWeatherTimeline(c *Config) (*WeatherData, error) {
	now := time.Now()

	weather, err := GetWeatherForecast(c)
	if err != nil {
		return nil, fmt.Errorf("could not get weather forecast: %v", err)
	}

	fc := weather.CurrentWeather

	// grab the hour by hour weather details
	timepoints := make([]*WeatherHour, 0)
	for _, d := range weather.Forecast.Days {
		for _, h := range d.Hours {
			timepoints = append(timepoints, h)
		}
	}

	// Decide whether we need weather history, i.e. whether the lookback takes us into yesterday
	if now.Add(time.Duration(-c.RainLookback)*time.Hour).Day() != now.Day() {
		history, err := GetWeatherHistory(c, now)
		if err != nil {
			return nil, fmt.Errorf("could not get weather history: %v", err)
		}

		historyTps := make([]*WeatherHour, 0)
		for _, h := range history.Forecast.Days[0].Hours {
			historyTps = append(historyTps, h)
		}

		timepoints = append(historyTps, timepoints...)
	}

	data := ParseWeatherTimeline(c, now, timepoints)
	data.Current = fc

	return data, nil
}

func (cw *CurrentWeather) IsHot(c *Config) bool {
	return cw.Temp > c.HotThreshold
}

// unused, TODO: determine how to set humidity threshold in conjunction with forecast
func (cw *CurrentWeather) IsDry(c *Config) bool {
	return cw.Humidity < c.DryThreshold
}

func codeInList(code string, list []string) bool {
	for _, c := range list {
		if code == c {
			return true
		}
	}
	return false
}

// unused, TODO: determine whether this is even worth using
func (cw *CurrentWeather) IsSunny(c *Config) bool {
	return codeInList(fmt.Sprintf("%v", cw.Condition.Code), c.SunnyWeatherCodes)
}

// Determine whether to water during a primary timepoint based on weather history/forecast
func ShouldWaterPrimary(c *Config, data *WeatherData) bool {
	// return false if it's been/will be rainy
	if data != nil {
		if data.PastPrecip >= c.PastRainThreshold || data.FuturePrecip >= c.FutureRainThreshold {
			return false
		}
	}
	return true
}

// Determine whether to water during a primary timepoint based on weather history/forecast
// and current conditions
func ShouldWaterSecondary(c *Config, data *WeatherData) bool {
	// return false if it's been/will be rainy
	if data != nil {
		if data.PastPrecip >= c.PastRainThreshold || data.FuturePrecip >= c.FutureRainThreshold {
			return false
		}
		if data.Current.IsHot(c) {
			return true
		}
	}
	return false
}
