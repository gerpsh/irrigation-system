#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVICE_NAME="irrigation.service"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}"
RESTART_SERVICE_NAME="irrigation-restart.service"
RESTART_SERVICE_FILE="/etc/systemd/system/${RESTART_SERVICE_NAME}"
RESTART_TIMER_NAME="irrigation-restart.timer"
RESTART_TIMER_FILE="/etc/systemd/system/${RESTART_TIMER_NAME}"
BINARY_NAME="irrigation-system"

usage() {
    echo "Usage: sudo $0 <install-directory>"
    echo ""
    echo "Sets up the irrigation system as a systemd service on a Raspberry Pi."
    echo ""
    echo "  <install-directory>  Absolute path where the binary and config.json reside."
    echo "                       config.json must already exist in this directory."
    echo ""
    echo "This script must be run as root."
    exit 1
}

log() {
    echo "[setup] $*"
}

fail() {
    echo "[setup] ERROR: $*" >&2
    exit 1
}

# --- Validate inputs ---

if [[ $# -lt 1 ]]; then
    usage
fi

INSTALL_DIR="$1"

if [[ $EUID -ne 0 ]]; then
    fail "This script must be run as root (use sudo)."
fi

if [[ ! -d "$INSTALL_DIR" ]]; then
    fail "Install directory does not exist: ${INSTALL_DIR}"
fi

CONFIG_PATH="${INSTALL_DIR}/config.json"
if [[ ! -f "$CONFIG_PATH" ]]; then
    fail "config.json not found in ${INSTALL_DIR}. Create it before running setup (see config_sample.json)."
fi

if ! command -v go &>/dev/null; then
    fail "Go is not installed or not on PATH. Install Go 1.22+ before running setup."
fi

log "Install directory: ${INSTALL_DIR}"
log "Config path:       ${CONFIG_PATH}"

# --- Build the binary ---

log "Building binary..."
(cd "$SCRIPT_DIR" && go build -o "${INSTALL_DIR}/${BINARY_NAME}" main.go config.go log.go water.go weather.go)
log "Binary built at ${INSTALL_DIR}/${BINARY_NAME}"

# --- Create log directories ---

parse_json_field() {
    local field="$1"
    local file="$2"
    python3 -c "import json,sys; print(json.load(open('${file}')).get('${field}',''))" 2>/dev/null
}

EVENT_LOG_FILE="$(parse_json_field event_log_file "$CONFIG_PATH")"
ERROR_LOG_FILE="$(parse_json_field error_log_file "$CONFIG_PATH")"

if [[ -n "$EVENT_LOG_FILE" ]]; then
    EVENT_LOG_DIR="$(dirname "$EVENT_LOG_FILE")"
    log "Creating event log directory: ${EVENT_LOG_DIR}"
    mkdir -p "$EVENT_LOG_DIR"
fi

if [[ -n "$ERROR_LOG_FILE" ]]; then
    ERROR_LOG_DIR="$(dirname "$ERROR_LOG_FILE")"
    log "Creating error log directory: ${ERROR_LOG_DIR}"
    mkdir -p "$ERROR_LOG_DIR"
fi

# --- Verify GPIO access ---

if [[ -e /dev/gpiomem ]]; then
    log "GPIO device /dev/gpiomem found."
elif [[ -e /dev/mem ]]; then
    log "GPIO device /dev/gpiomem not found, but /dev/mem is available (requires root)."
else
    log "WARNING: Neither /dev/gpiomem nor /dev/mem found. GPIO may not work on this device."
fi

# --- Install systemd service ---

log "Installing systemd service to ${SERVICE_FILE}..."

cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=Irrigation System
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${INSTALL_DIR}/${BINARY_NAME} ${CONFIG_PATH}
WorkingDirectory=${INSTALL_DIR}
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# --- Install daily restart timer ---

log "Installing restart service to ${RESTART_SERVICE_FILE}..."

cat > "$RESTART_SERVICE_FILE" <<EOF
[Unit]
Description=Restart Irrigation System

[Service]
Type=oneshot
ExecStart=/usr/bin/systemctl restart ${SERVICE_NAME}
EOF

log "Installing restart timer to ${RESTART_TIMER_FILE}..."

cat > "$RESTART_TIMER_FILE" <<EOF
[Unit]
Description=Daily restart of Irrigation System

[Timer]
OnCalendar=*-*-* 02:00:00
Persistent=true

[Install]
WantedBy=timers.target
EOF

# --- Enable and start everything ---

log "Reloading systemd daemon..."
systemctl daemon-reload

log "Enabling ${SERVICE_NAME}..."
systemctl enable "$SERVICE_NAME"

log "Starting ${SERVICE_NAME}..."
systemctl start "$SERVICE_NAME"

log "Enabling ${RESTART_TIMER_NAME}..."
systemctl enable "$RESTART_TIMER_NAME"

log "Starting ${RESTART_TIMER_NAME}..."
systemctl start "$RESTART_TIMER_NAME"

echo ""
log "Setup complete. Service status:"
echo ""
systemctl status "$SERVICE_NAME" --no-pager || true
echo ""
log "Restart timer status:"
echo ""
systemctl status "$RESTART_TIMER_NAME" --no-pager || true
