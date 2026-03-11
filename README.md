# Irrigation System

A weather-aware irrigation controller for Raspberry Pi. It drives solenoid valves via GPIO on a configurable schedule, using forecast and historical weather data from [WeatherAPI.com](https://www.weatherapi.com/) to skip watering when rain is expected or temperatures are too cold, and to add extra watering during heat waves.

## How It Works

The system runs as a single long-lived process that checks the clock every second. When the current time matches a configured **timepoint** for a valve, it decides whether to water:

- **Primary timepoints** water by default, *unless*:
  - Combined past + forecast precipitation in the lookback/lookahead window meets or exceeds the rain threshold
  - The current temperature is below the cold threshold (frost risk)
- **Secondary timepoints** do *not* water by default, *unless*:
  - The current temperature is at or above the hot threshold
  - Combined precipitation is below the rain threshold

Without weather enabled, primary timepoints always water and secondary timepoints never fire.

## Prerequisites

- **Hardware**: Raspberry Pi with GPIO, relay HAT, 12V DC solenoid valves, and a 12V power supply. See [HARDWARE.md](HARDWARE.md) for the full parts list, wiring guide, and plumbing instructions.
- **Go**: 1.22.3+
- **WeatherAPI.com API key** (free tier works) if using weather-based decisions
- **PostgreSQL** (optional) for event/error logging
- **Pushover account** (optional) for push notifications

## Configuration

All configuration lives in a single `config.json` file. Copy the sample to get started:

```bash
cp config_sample.json config.json
```

### Config Fields

| Field | Type | Description |
|---|---|---|
| `use_db_log` | bool | Enable PostgreSQL logging |
| `event_log_file` | string | Path to the file-based event log |
| `error_log_file` | string | Path to the file-based error log |
| `log_db_uri` | string | PostgreSQL connection string (required when `use_db_log` is true) |
| `event_table` | string | Name of the events table in PostgreSQL |
| `error_table` | string | Name of the errors table in PostgreSQL |
| `use_pushover` | bool | Enable Pushover push notifications |
| `pushover_user_keys` | string[] | Pushover user keys to notify |
| `pushover_app_token` | string | Pushover application token |
| `valves` | Valve[] | Array of valve definitions (see below) |
| `use_weather` | bool | Enable weather-based watering decisions |
| `weather_api_key` | string | WeatherAPI.com API key |
| `location` | string | Location for weather queries (e.g. a US zip code) |
| `weather_forecast_url` | string | WeatherAPI forecast endpoint template |
| `weather_history_url` | string | WeatherAPI history endpoint template |
| `rain_lookback` | int | Hours to look back for past precipitation |
| `rain_lookahead` | int | Hours to look ahead for forecast precipitation |
| `rain_threshold` | float | Combined past + forecast precipitation (mm) that triggers a skip |
| `hot_threshold` | float | Temperature (F) at or above which secondary timepoints activate |
| `cold_threshold` | float | Temperature (F) below which primary timepoints are skipped |
| `check_online_url` | string | URL used to verify internet connectivity before API calls |

### Valve Configuration

Each valve represents a physical solenoid connected to a GPIO pin:

| Field | Type | Description |
|---|---|---|
| `id` | string | Unique identifier for the valve |
| `name` | string | Human-readable name (e.g. "blueberries") |
| `pin` | int | GPIO pin number (BCM numbering) |
| `timepoints` | Timepoint[] | Scheduled watering windows |

### Timepoint Configuration

| Field | Type | Description |
|---|---|---|
| `days` | int[] | Days of the week to run (0 = Sunday, 6 = Saturday) |
| `hour` | int | Hour to trigger (0-23) |
| `minute` | int | Minute to trigger (0-59) |
| `type` | string | `"primary"` (default-on) or `"secondary"` (default-off, hot-weather only) |
| `duration` | int | How long to open the valve, in seconds |

### Example

```json
{
    "use_db_log": false,
    "event_log_file": "/var/log/irrigation/events.log",
    "error_log_file": "/var/log/irrigation/errors.log",
    "use_pushover": false,
    "valves": [
        {
            "id": "1",
            "name": "blueberries",
            "pin": 26,
            "timepoints": [
                {
                    "days": [0,1,2,3,4,5,6],
                    "hour": 7,
                    "minute": 1,
                    "type": "primary",
                    "duration": 75
                },
                {
                    "days": [0,1,2,3,4,5,6],
                    "hour": 3,
                    "minute": 30,
                    "type": "secondary",
                    "duration": 30
                }
            ]
        }
    ],
    "use_weather": true,
    "weather_api_key": "<your_key>",
    "location": "19130",
    "weather_forecast_url": "https://api.weatherapi.com/v1/forecast.json?key=%v&q=%v&days=2&aqi=no&alerts=no",
    "weather_history_url": "https://api.weatherapi.com/v1/history.json?key=%v&q=%v&dt={}",
    "rain_lookback": 6,
    "rain_lookahead": 6,
    "rain_threshold": 10.0,
    "hot_threshold": 75.0,
    "cold_threshold": 40.0,
    "check_online_url": "https://www.google.com/"
}
```

## Building

```bash
make build
```

This compiles the binary to `./irrigation-system`.

To run the tests:

```bash
make test
```

## Database Setup (Optional)

If using PostgreSQL for logging (`use_db_log: true`), create the database and tables using the provided schema:

```bash
psql -h <host> -U <user> -f log_db.schema
```

This creates the `irrigationlog` database with `events` and `errors` tables.

## Deployment

The system is designed to run as a systemd service on a Raspberry Pi. A setup script is provided to automate the process.

### Quick Setup

1. Copy `config_sample.json` to your desired install directory and edit it:

```bash
mkdir -p /opt/irrigation-system
cp config_sample.json /opt/irrigation-system/config.json
# edit /opt/irrigation-system/config.json with your settings
```

2. Run the setup script, passing the install directory:

```bash
sudo ./setup.sh /opt/irrigation-system
```

The script will:
- Build the Go binary into the install directory
- Create parent directories for the configured log files
- Verify GPIO device availability
- Install, enable, and start a systemd service (`irrigation.service`)
- Install and enable a daily restart timer (`irrigation-restart.timer`) that restarts the service at 2:00 AM to guard against hangs

### Manual Setup

If you prefer to set things up by hand:

**Build the binary:**

```bash
make build
```

**Create the config:**

Place `config.json` in the install directory. The binary accepts an optional config path argument (defaults to `./config.json`):

```bash
./irrigation-system /path/to/config.json
```

**Create a systemd service** at `/etc/systemd/system/irrigation.service`:

```ini
[Unit]
Description=Irrigation System
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/opt/irrigation-system/irrigation-system /opt/irrigation-system/config.json
WorkingDirectory=/opt/irrigation-system
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
```

**Enable and start:**

```bash
sudo systemctl daemon-reload
sudo systemctl enable irrigation.service
sudo systemctl start irrigation.service
```

### Daily Restart Timer

The setup script installs a systemd timer that restarts the service once a day at 2:00 AM. This guards against potential process hangs. To check the timer status:

```bash
systemctl status irrigation-restart.timer
systemctl list-timers irrigation-restart.timer
```

### Managing the Service

The Makefile provides shortcuts for common operations:

```bash
make restart   # rebuild and restart the service
make status    # check service status
```
