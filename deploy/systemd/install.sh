#!/usr/bin/env bash
# Installs game-over-man as a systemd timer running every 10 minutes.
# Run as root: sudo ./install.sh
set -euo pipefail

UNIT_DIR=/etc/systemd/system
DATA_DIR=/var/lib/game-over-man
CONFIG_DIR=/etc/game-over-man
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [ "$EUID" -ne 0 ]; then
  echo "Please run as root: sudo $0"
  exit 1
fi

if ! command -v docker &>/dev/null; then
  echo "Docker not found. Install it first: https://docs.docker.com/engine/install/"
  exit 1
fi

echo "Creating directories..."
mkdir -p "$DATA_DIR" "$CONFIG_DIR"

echo "Installing unit files..."
install -m 644 "$SCRIPT_DIR/game-over-man.service" "$UNIT_DIR/"
install -m 644 "$SCRIPT_DIR/game-over-man.timer"   "$UNIT_DIR/"

if [ ! -f "$CONFIG_DIR/env" ]; then
  install -m 600 "$SCRIPT_DIR/env.example" "$CONFIG_DIR/env"
  echo ""
  echo "  -> Edit $CONFIG_DIR/env with your NOTIFICATION_URL before starting."
fi

if [ ! -f "$CONFIG_DIR/config.json" ]; then
  echo "  -> Copy your config.json to $CONFIG_DIR/config.json before starting."
fi

echo ""
echo "Reloading systemd..."
systemctl daemon-reload

echo "Enabling timer..."
systemctl enable game-over-man.timer

echo ""
echo "Once your config.json and env file are in place, start with:"
echo "  sudo systemctl start game-over-man.timer"
echo ""
echo "Useful commands:"
echo "  systemctl status game-over-man.timer   -- next scheduled run"
echo "  systemctl start game-over-man.service  -- run immediately"
echo "  journalctl -u game-over-man -f         -- follow logs"
