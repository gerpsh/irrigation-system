package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// Entry that specifies the days of the week [1...7], hour, and minute of the day
// that watering should occur
type WaterTimepoint struct {
	Days   []int `json:"days"`
	Hour   int   `json:"hour"`
	Minute int   `json:"minute"`
}

// edit config in file named "config.json", must have this name
type Config struct {
	UseDBLog               bool   `json:"use_db_log"`     // true if using db log, false if using log file
	EventLogFile           string `json:"event_log_file"` // file path for event log file if using log file
	ErrorLogFile           string `json:"error_log_file"` // like above but for errors
	LogDBURI               string `json:"log_db_uri"`     // database connection string if using db log
	LogDB                  *sql.DB
	ErrorTable             string                       `json:"error_table"`
	EventTable             string                       `json:"event_table"`
	UsePushover            bool                         `json:"use_pushover"`
	PushoverUserKeys       []string                     `json:"pushover_user_keys"`
	PushoverAppToken       string                       `json:"pushover_app_token"`
	Valves                 []*Valve                     `json:"valves"`                // see water.go
	WaterTimepoints        map[string][]*WaterTimepoint `json:"water_timepoints"`      // default timepoints
	UseWeather             bool                         `json:"use_weather"`           // whether or not to check weather when deciding to water
	WeatherApiKey          string                       `json:"weather_api_key"`       // weatherapi.com api key
	Location               string                       `json:"location"`              // use a zip code in the USA
	WeatherForecastUrl     string                       `json:"weather_forecast_url"`  // url for weather forecast with formatting characters
	WeatherHistoryUrl      string                       `json:"weather_history_url"`   // likewise but for history, with extra placeholder for history date
	RainLookback           int                          `json:"rain_lookback"`         // how many hours to look back to measure rainfall
	PastRainThreshold      float64                      `json:"past_rain_threshold"`   // threshold of rain in mm to decide whether to water
	RainLookahead          int                          `json:"rain_lookahead"`        // hours to look ahead to measure rainfail
	FutureRainThreshold    float64                      `json:"future_rain_threshold"` // threshold of rain in mm to decide whether to water
	WeatherCodes           map[string]string            `json:"weather_codes"`         // weather codes with corresponding text description. [0] refers to day description, [1] refers to night
	SunnyWeatherCodes      []string                     `json:"sunny_weather_code"`
	HotThreshold           float64                      `json:"hot_threshold"` // temp in F that is considered hot, used to determine whether to do a secondary water
	DryThreshold           int                          `json:"dry_threshold"` // humidity pct that is considered dry, used to determine whether to do a secondary water
	CloudyWeatherCodes     []string                     `json:"cloudy_weather_codes"`
	SortaRainyWeatherCodes []string                     `json:"sorta_rainy_weather_codes"`
	DefRainyWeatherCodes   []string                     `json:"def_rainy_weather_codes"`
	CheckOnlineUrl         string                       `json:"check_online_url"` // url to use to check if device is internet connected
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

	c.WeatherForecastUrl = fmt.Sprintf(c.WeatherForecastUrl, c.WeatherApiKey, c.Location)
	c.WeatherHistoryUrl = fmt.Sprintf(c.WeatherHistoryUrl, c.WeatherApiKey, c.Location)

	if c.UseDBLog {
		db, err := sql.Open("postgres", c.LogDBURI)
		if err != nil {
			return nil, fmt.Errorf("could not connect to log db: %v", err)
		}
		c.LogDB = db
	}

	return c, nil
}

// Check config to make sure that configuration dependencies are met
// E.g. if UseWeather is set to true but WeatherAPIKey is blank, return an error
func (c *Config) CheckConfig() error {
	if c.EventLogFile == "" {
		return fmt.Errorf("event log file required, please provide a full path in event_log_file config field")
	}
	if c.ErrorLogFile == "" {
		return fmt.Errorf("error log file required, please provide a full path in error_log_file config field")
	}
	if c.UseDBLog && (c.LogDBURI == "" || c.ErrorTable == "" || c.EventTable == "") {
		return fmt.Errorf(`please provide full config details for log db (log_db_uri, event_table, error_table) to use log db, \
					otherwise set use_log_db to false`)
	}

	return nil
}

// check if we're connected to the internet
func (c *Config) OnlineCheck() error {
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	_, err := client.Get(c.CheckOnlineUrl)
	if err != nil {
		c.UseDBLog = false
		c.UsePushover = false
		c.UseWeather = false
	} else {
		if c.LogDBURI != "" {
			c.UseDBLog = true
		}
		if c.PushoverAppToken != "" {
			c.UsePushover = true
		}
		if c.WeatherApiKey != "" {
			c.UseWeather = true
		}
	}
	return err
}
