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

func FormatEventMessage(cw *CurrentWeather, duration string) string {
	if cw == nil {
		return fmt.Sprintf("Water Event: %v", duration)
	} else {
		msg := fmt.Sprintf("Temp: %v || Humidity: %v || Condition: %v || Water Type: %v", cw.Temp, cw.Humidity, cw.Condition.Text, duration)
		return msg
	}
}

func LogEvent(c *Config, cw *CurrentWeather, duration string) error {
	le := LogEntry{
		Type:      "event",
		Timestamp: time.Now(),
		Message:   FormatEventMessage(cw, duration),
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
		_, err := c.LogDB.Exec("INSERT INTO events (eventtime, message) VALUES ($1, $2)", le.Timestamp, le.Message)
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
