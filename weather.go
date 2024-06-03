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

func (wt *WeatherTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	if s == "null" {
		wt.Time = time.Time{}
	}
	t, err := time.Parse("2006-01-02 15:04", s)
	if err != nil {
		return err
	}
	wt.Time = t
	return nil
}

type WeatherHour struct {
	Time     WeatherTime `json:"time"`
	PrecipMM float64     `json:"precip_mm"`
}

type Date struct {
	time.Time
}

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

type ForecastDay struct {
	Date  Date           `json:"date"`
	Hours []*WeatherHour `json:"hour"`
}

type ForecastSection struct {
	Days []*ForecastDay `json:"forecastDay"`
}

type WeatherForecastResponse struct {
	CurrentWeather *CurrentWeather  `json:"current"`
	Forecast       *ForecastSection `json:"forecast"`
}

func (cw *CurrentWeather) IsHot(c *Config) bool {
	return cw.Temp > c.HotThreshold
}

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

func (cw *CurrentWeather) IsSunny(c *Config) bool {
	return codeInList(fmt.Sprintf("%v", cw.Condition.Code), c.SunnyWeatherCodes)
}

// func (cw *CurrentWeather) IsCloudy(c *Config) bool {
// 	return codeInList(fmt.Sprintf("%v", cw.Condition.Code), c.CloudyWeatherCodes)
// }

// func (cw *CurrentWeather) IsSortaRaining(c *Config) bool {
// 	return codeInList(fmt.Sprintf("%v", cw.Condition.Code), c.SortaRainyWeatherCodes)
// }

// func (cw *CurrentWeather) IsDefRaining(c *Config) bool {
// 	return codeInList(fmt.Sprintf("%v", cw.Condition.Code), c.DefRainyWeatherCodes)
// }

type WeatherData struct {
	Current      *CurrentWeather
	PastPrecip   float64
	FuturePrecip float64
}

func ParseWeatherTimeline(c *Config, now time.Time, tps []*WeatherHour) *WeatherData {
	pastSum := 0.000
	futureSum := 0.000

	for _, tp := range tps {
		if tp.Time.After(now.Add(time.Duration(-c.RainLookback)*time.Hour)) && tp.Time.Before(now) {
			pastSum = pastSum + tp.PrecipMM
		}

		if tp.Time.Before(now.Add(time.Duration(c.RainLookahead)*time.Hour)) && tp.Time.After(now) {
			futureSum = futureSum + tp.PrecipMM
		}
	}

	return &WeatherData{
		PastPrecip:   pastSum,
		FuturePrecip: futureSum,
	}
}

// get amount of water for lookback + lookahead interval, along with current weather
func GetWeatherTimeline(c *Config) (*WeatherData, error) {
	now := time.Now()

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

	// grab the hour by hour weather details
	timepoints := make([]*WeatherHour, 0)
	for _, d := range weather.Forecast.Days {
		for _, h := range d.Hours {
			timepoints = append(timepoints, h)
		}
	}

	// Decide whether we need weather history, i.e. whether the lookback takes us into yesterday
	if now.Add(time.Duration(-c.RainLookback)*time.Hour).Day() != now.Day() {
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

		historyTps := make([]*WeatherHour, 0)
		for _, h := range history.Forecast.Days[0].Hours {
			historyTps = append(historyTps, h)
		}

		timepoints = append(historyTps, timepoints...)
	}

	data := ParseWeatherTimeline(c, now, timepoints)
	data.Current = weather.CurrentWeather

	return data, nil

}

func ShouldWaterPrimary(c *Config, data *WeatherData) bool {
	// return false if it's been/will be rainy
	if data.PastPrecip >= c.PastRainThreshold || data.FuturePrecip >= c.FutureRainThreshold {
		return false
	}
	return true
}

func ShouldWaterSecondary(c *Config, data *WeatherData) bool {
	// return false if it's been/will be rainy
	if data.PastPrecip >= c.PastRainThreshold || data.FuturePrecip >= c.FutureRainThreshold {
		return false
	}
	if data.Current.IsSunny(c) && data.Current.IsHot(c) && data.Current.IsDry(c) {
		return true
	}
	return false
}
