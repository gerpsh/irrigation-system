package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gregdel/pushover"
)

type LogEntry struct {
	Type      string
	Timestamp time.Time
	Message   string
}

func (le *LogEntry) String() string {
	tstr := le.Timestamp.Format("2006-01-02 15:04:05")
	return fmt.Sprintf("%v - %v", tstr, le.Message)
}

// format watering event log message
func FormatEventMessage(cw *CurrentWeather, duration string, valve string, name string) string {
	if cw == nil {
		return fmt.Sprintf("Water on Valve %v (%v) Event: %v", valve, name, duration)
	} else {
		msg := fmt.Sprintf("Valve: %v (%v) || Temp: %v || Humidity: %v || Condition: %v || Water Type: %v", valve, name, cw.Temp, cw.Humidity, cw.Condition.Text, duration)
		return msg
	}
}

// log event in location defined in config
func (v *Valve) LogEvent(c *Config, cw *CurrentWeather, duration string) error {
	le := LogEntry{
		Type:      "event",
		Timestamp: time.Now(),
		Message:   FormatEventMessage(cw, duration, v.ID, v.Name),
	}
	if !c.UseDBLog {
		file, err := os.OpenFile(c.EventLogFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return fmt.Errorf("could not open event log file: %v", err)
		}
		defer file.Close()
		_, err = file.Write([]byte(le.String() + "\n"))
		if err != nil {
			log.Println("could not log event")
			return fmt.Errorf("could not log event to file: %v", err)
		}
		return nil
	} else {
		err := c.LogDB.Ping()
		if err != nil {
			return fmt.Errorf("could not connect to log database: %v", err)
		}
		_, err = c.LogDB.Exec("INSERT INTO events (eventtime, message) VALUES ($1, $2)", le.Timestamp, le.Message)
		if err != nil {
			return fmt.Errorf("could not insert event into log db: %v", err)
		}
	}
	if c.UsePushover {
		err := PushNotif(c, &le)
		if err != nil {
			return fmt.Errorf("could not send push notification: %v", err)
		}
	}
	return nil
}

// log error in location defined in config
func LogError(c *Config, e error) error {
	le := LogEntry{
		Type:      "error",
		Timestamp: time.Now(),
		Message:   fmt.Sprint(e),
	}
	if !c.UseDBLog {
		file, err := os.OpenFile(c.EventLogFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return fmt.Errorf("could not open error log file: %v", err)
		}
		defer file.Close()
		_, err = file.Write([]byte(le.String() + "\n"))
		if err != nil {
			return fmt.Errorf("could not log error to file: %v", err)
		}
	} else {
		_, err := c.LogDB.Exec("INSERT INTO errors (errortime, message) VALUES ($1, $2)", le.Timestamp, le.Message)
		if err != nil {
			return fmt.Errorf("could not insert error into log db: %v", err)
		}
	}
	if c.UsePushover {
		err := PushNotif(c, &le)
		if err != nil {
			return fmt.Errorf("could not send push notification: %v", err)
		}
	}
	return nil
}

// Send push notif request to Pushover
func PushNotif(c *Config, le *LogEntry) error {
	app := pushover.New(c.PushoverAppToken)
	recipients := make([]*pushover.Recipient, 0)
	for _, uk := range c.PushoverUserKeys {
		recipient := pushover.NewRecipient(uk)
		recipients = append(recipients, recipient)
	}

	msg := pushover.NewMessageWithTitle(le.String(), fmt.Sprintf("Irrigation %v", le.Type))
	for _, r := range recipients {
		_, err := app.SendMessage(msg, r)
		if err != nil {
			return fmt.Errorf("could not send push notif to recipient: %v", err)
		}
	}
	return nil
}
