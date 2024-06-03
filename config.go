package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

type Config struct {
	UseDBLog               bool   `json:"use_db_log"`     // true if using db log, false if using log file
	EventLogFile           string `json:"event_log_file"` // file path for event log file if using log file
	ErrorLogFile           string `json:"error_log_file"` // like above but for errors
	LogDBURI               string `json:"log_db_uri"`     // database connection string if using db log
	LogDB                  *sql.DB
	ErrorTable             string                      `json:"error_table"`
	EventTable             string                      `json:"event_table"`
	UsePushover            bool                        `json:"use_pushover"`
	PushoverUserKeys       []string                    `json:"pushover_user_keys"`
	PushoverAppToken       string                      `json:"pushover_app_token"`
	Valves                 []*Valve                    `json:"valves"`               // see water.go
	WaterTimepoints        map[string][]map[string]int `json:"water_timepoints"`     // default timepoints
	UseWeather             bool                        `json:"weather"`              // whether or not to check weather when deciding to water
	ApiKey                 string                      `json:"api_key"`              // weatherapi.com api key
	Location               string                      `json:"location"`             // use a zip code in the USA
	WeatherForecastUrl     string                      `json:"weather_forecast_url"` // url for weather forecast with formatting characters
	WeatherHistoryUrl      string                      `json:"weather_history_url"`  // likewise but for history, with extra placeholder for history date
	RainLookback           int                         `json:"rain_lookback"`        // how many hours to look back to measure rainfall
	PastRainThreshold      float64                     `json:"past_rain_threshold"`  // threshold of rain in mm to decide whether to water
	RainLookahead          int                         `json:"rain_lookahead"`       // hours to look ahead to measure rainfail
	FutureRainThreshold    float64                     `json:"future_rain_theshold"` // threshold of rain in mm to decide whether to water
	WeatherCodes           map[string]string           `json:"weather_codes"`        // weather codes with corresponding text description. [0] refers to day description, [1] refers to night
	SunnyWeatherCodes      []string                    `json:"sunny_weather_code"`
	HotThreshold           float64                     `json:"hot_threshold"` // temp in F that is considered hot, used to determine whether to do a secondary water
	DryThreshold           int                         `json:"dry_threshold"` // humidity pct that is considered dry, used to determine whether to do a secondary water
	CloudyWeatherCodes     []string                    `json:"cloudy_weather_codes"`
	SortaRainyWeatherCodes []string                    `json:"sorta_rainy_weather_codes"`
	DefRainyWeatherCodes   []string                    `json:"def_rainy_weather_codes"`
}

func ReadConfig(path string) (*Config, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read config file contents: %v", err)
	}

	var c *Config
	err = json.Unmarshal(f, &c)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal json to config struct: %v", err)
	}

	c.WeatherForecastUrl = fmt.Sprintf(c.WeatherForecastUrl, c.ApiKey, c.Location)
	c.WeatherHistoryUrl = fmt.Sprintf(c.WeatherHistoryUrl, c.ApiKey, c.Location)

	if c.UseDBLog {
		db, err := sql.Open("postgres", c.LogDBURI)
		if err != nil {
			return nil, fmt.Errorf("could not connect to log db: %v", err)
		}
		c.LogDB = db
	}

	return c, nil
}
