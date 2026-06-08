#!/usr/bin/env sh
set -eu

SERVICE_NAME="${1:-acme233-auto-renew}"
CONFIG_PATH="${2:-$(pwd)/config_acme.json}"
BINARY_PATH="${3:-$(pwd)/acme233}"
UNIT_PATH="/etc/systemd/system/${SERVICE_NAME}.service"
WORK_DIR="$(dirname "$CONFIG_PATH")"

if [ ! -f "$CONFIG_PATH" ]; then
	echo "config file not found: $CONFIG_PATH" >&2
	exit 1
fi
if [ ! -f "$BINARY_PATH" ]; then
	echo "binary not found: $BINARY_PATH" >&2
	exit 1
fi

if [ "$(id -u)" -ne 0 ]; then
	SUDO="sudo"
else
	SUDO=""
fi

tmp_file="$(mktemp)"
trap 'rm -f "$tmp_file"' EXIT

cat >"$tmp_file" <<EOF
[Unit]
Description=acme233 automatic renewal service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=$WORK_DIR
ExecStart=$BINARY_PATH auto-renew -config $CONFIG_PATH
Restart=always
RestartSec=30

[Install]
WantedBy=multi-user.target
EOF

$SUDO cp "$tmp_file" "$UNIT_PATH"
$SUDO systemctl daemon-reload
$SUDO systemctl enable --now "$SERVICE_NAME"

echo "Installed and started $SERVICE_NAME"
echo "Check status with: systemctl status $SERVICE_NAME"