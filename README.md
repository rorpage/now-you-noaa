# Now You NOAA

A NOAA weather alert notifier for home servers. It polls the NOAA Weather API for active alerts in the areas you configure, then fires a single webhook notification per alert. No alert goes unnoticed; no notification repeats.

It is a single Go binary with no runtime dependencies. Drop it on any Linux or macOS machine, point it at a config file, and schedule it with cron or systemd.

## Features

- Monitors any U.S. state or NOAA forecast zone for active weather alerts
- Filter by event type (e.g. Tornado Warning, Severe Thunderstorm Warning) and severity
- One notification per alert -- no duplicates, even across restarts
- Built-in Slack and Discord payload presets; custom Go template support for any other platform
- Webhook URL configurable via environment variable or config file
- Custom HTTP headers supported (for auth tokens and other per-platform requirements)
- Persistent state in a plain JSON file; old entries are pruned automatically
- Single static binary, no runtime dependencies, no Docker required

## Installation

### Download a pre-built binary (recommended)

Download the appropriate binary for your platform from the [Releases](https://github.com/rorpage/now-you-noaa/releases) page:

| Platform | File |
|---|---|
| Linux x86-64 | `now-you-noaa-linux-amd64` |
| Linux ARM64 (Raspberry Pi, etc.) | `now-you-noaa-linux-arm64` |
| macOS Intel | `now-you-noaa-darwin-amd64` |
| macOS Apple Silicon | `now-you-noaa-darwin-arm64` |
| Windows x86-64 | `now-you-noaa-windows-amd64.exe` |

```bash
# Example: Linux x86-64
curl -fsSL https://github.com/rorpage/now-you-noaa/releases/latest/download/now-you-noaa-linux-amd64 \
  -o /usr/local/bin/now-you-noaa
chmod +x /usr/local/bin/now-you-noaa
```

### Build from source

Requires Go 1.22+.

```bash
git clone https://github.com/rorpage/now-you-noaa.git
cd now-you-noaa
CGO_ENABLED=0 go build -ldflags="-s -w" -o now-you-noaa .
sudo mv now-you-noaa /usr/local/bin/
```

## Configuration

Create a config file at `/etc/now-you-noaa/config.json`:

```bash
sudo mkdir -p /etc/now-you-noaa
sudo nano /etc/now-you-noaa/config.json
```

```json
{
  "areas": ["IN"],
  "eventTypes": [
    "Tornado Warning",
    "Severe Thunderstorm Warning",
    "Flash Flood Warning"
  ]
}
```

See `config.example.json` for a more complete example with all supported fields.

### Config fields

| Field | Required | Default | Description |
|---|---|---|---|
| `areas` | See note | -- | Array of state abbreviations to monitor (e.g. `["IN", "OH"]`) |
| `zones` | See note | -- | Array of NOAA zone codes to monitor (e.g. `["INZ032", "INZ050"]`) |
| `eventTypes` | No | all | Array of event type strings to match (e.g. `["Tornado Warning"]`). Empty means all events. |
| `severity` | No | all | Array of severity levels to match: `Extreme`, `Severe`, `Moderate`, `Minor`. Empty means all. |
| `userAgent` | No | built-in | User-Agent header sent to NOAA API. Override to identify your instance. |
| `notificationUrl` | See note | -- | Webhook URL to POST alerts to |
| `notificationMethod` | No | `POST` | HTTP method for notifications |
| `notificationHeaders` | No | -- | Extra headers (e.g. `{"Authorization": "Bearer ..."}`) |
| `notificationType` | No | `webhook` | Payload format: `webhook`, `slack`, `discord`, or `template` |
| `notificationTemplate` | If `template` | -- | Go template string used when `notificationType` is `template` |
| `stateFilePath` | No | `/var/lib/now-you-noaa/state.json` | Where to persist notification state |
| `pruneAfterDays` | No | `7` | How many days to keep state entries before pruning |

**Areas or zones:** At least one `area` or `zone` is required. Both can be configured simultaneously; duplicate alerts are suppressed.

**Notification URL:** Set via the `NOTIFICATION_URL` environment variable (preferred, keeps it out of the config file) or as `notificationUrl` in the config. The env var takes precedence.

## Environment Variables

| Variable | Description |
|---|---|
| `NOTIFICATION_URL` | Webhook URL (overrides `notificationUrl` in config) |
| `CONFIG_FILE` | Path to config file (default: `/etc/now-you-noaa/config.json`) |
| `STATE_FILE` | Path to state file (default: `/var/lib/now-you-noaa/state.json`) |

## Finding Your Zone Codes

NOAA divides the U.S. into forecast zones. Zone codes follow the pattern `SSZnnn` where `SS` is the state abbreviation and `nnn` is a three-digit number (e.g. `INZ032` is the Marion County, Indiana zone).

To find zone codes for your area:

1. Visit [alerts.weather.gov](https://alerts.weather.gov/) and look at your local NWS office
2. Query the API directly: `curl "https://api.weather.gov/zones?area=IN&type=forecast"`
3. Use the state-level `areas` filter instead -- it covers all zones in the state

## Common Event Types

These are the most commonly watched NOAA event types. Event type strings are case-sensitive in config.

| Event Type | Severity |
|---|---|
| Tornado Warning | Extreme |
| Tornado Watch | Severe |
| Severe Thunderstorm Warning | Severe |
| Severe Thunderstorm Watch | Moderate |
| Flash Flood Warning | Severe |
| Flash Flood Watch | Moderate |
| Flood Warning | Moderate |
| Winter Storm Warning | Severe |
| Blizzard Warning | Extreme |
| Ice Storm Warning | Severe |
| High Wind Warning | Severe |
| Excessive Heat Warning | Extreme |
| Special Weather Statement | -- |

Leave `eventTypes` empty to receive all active alerts for your configured areas.

## Notification Payload

Each alert is an HTTP POST with `Content-Type: application/json`:

```json
{
  "alert": {
    "id": "https://api.weather.gov/alerts/urn:oid:2.49.0.1.840.0.abc123",
    "event": "Tornado Warning",
    "headline": "Tornado Warning issued June 16 at 2:15PM EDT until 2:45 PM EDT",
    "severity": "Extreme",
    "urgency": "Immediate",
    "certainty": "Observed",
    "areaDesc": "Marion; Hamilton",
    "senderName": "NWS Indianapolis IN",
    "sent": "2026-06-16T18:15:00+00:00",
    "effective": "2026-06-16T18:15:00+00:00",
    "onset": "2026-06-16T18:15:00+00:00",
    "expires": "2026-06-16T18:45:00+00:00",
    "ends": "",
    "description": "...",
    "instruction": "TAKE COVER NOW!"
  },
  "summary": "Tornado Warning issued June 16 at 2:15PM EDT until 2:45 PM EDT"
}
```

### Notification types

The `notificationType` field controls the outgoing payload shape:

| Type | Payload sent |
|---|---|
| `webhook` (default) | Full JSON object (see [Notification Payload](#notification-payload)) |
| `slack` | `{"text": "<summary>"}` -- ready for a Slack incoming webhook URL |
| `discord` | `{"content": "<summary>"}` -- ready for a Discord webhook URL |
| `template` | Output of your Go template, rendered against the payload data |

**Slack:**

```json
{
  "notificationUrl": "https://hooks.slack.com/services/...",
  "notificationType": "slack"
}
```

**Discord:**

```json
{
  "notificationUrl": "https://discord.com/api/webhooks/...",
  "notificationType": "discord"
}
```

**Custom template:**

Set `notificationType` to `"template"` and provide a `notificationTemplate` string. The template is rendered with Go's [`text/template`](https://pkg.go.dev/text/template) and has access to all payload fields:

| Variable | Description |
|---|---|
| `{{.Summary}}` | Pre-built summary string (the alert headline, or event + area) |
| `{{.Alert.Event}}` | Event type (e.g. `Tornado Warning`) |
| `{{.Alert.Headline}}` | Alert headline text |
| `{{.Alert.Severity}}` | Severity level (`Extreme`, `Severe`, `Moderate`, `Minor`) |
| `{{.Alert.Urgency}}` | Urgency level (`Immediate`, `Expected`, `Future`) |
| `{{.Alert.AreaDesc}}` | Affected area description |
| `{{.Alert.SenderName}}` | Issuing NWS office (e.g. `NWS Indianapolis IN`) |
| `{{.Alert.Expires}}` | Expiration time (RFC 3339) |
| `{{.Alert.Instruction}}` | Safety instructions |
| `{{.Alert.Description}}` | Full alert description |

Example -- ntfy.sh with a plain-text title:

```json
{
  "notificationUrl": "https://ntfy.sh/my-weather-alerts",
  "notificationType": "template",
  "notificationTemplate": "{\"topic\": \"my-weather-alerts\", \"message\": \"{{.Summary}}\"}"
}
```

## Scheduling

The binary is a one-shot job: it runs, checks for new alerts, and exits.

### systemd timer (recommended)

systemd timers have proper log capture via `journalctl`, survive reboots cleanly with `Persistent=true`, run as a dedicated non-root user, and are standard on any modern Linux distro. Ready-to-use files are in `deploy/systemd/`.

**Quick install:**

```bash
sudo bash deploy/systemd/install.sh
```

The script downloads the latest binary from GitHub Releases (or builds from source if Go is available), creates a `now-you-noaa` system user, sets up `/etc/now-you-noaa/` and `/var/lib/now-you-noaa/`, and enables the timer. Then:

```bash
# Set your notification URL
sudo nano /etc/now-you-noaa/env

# Copy your config
sudo cp config.json /etc/now-you-noaa/config.json
sudo chown root:now-you-noaa /etc/now-you-noaa/config.json

# Start
sudo systemctl start now-you-noaa.timer
```

**Useful commands:**

```bash
systemctl status now-you-noaa.timer     # next scheduled run
systemctl start now-you-noaa.service    # run immediately
journalctl -u now-you-noaa -f           # follow logs
```

**Changing the schedule:** Edit `/etc/systemd/system/now-you-noaa.timer`, update `OnCalendar`, then:

```bash
sudo systemctl daemon-reload && sudo systemctl restart now-you-noaa.timer
```

Common values:

```ini
OnCalendar=*:0/5     # every 5 minutes (default)
OnCalendar=*:0/10    # every 10 minutes
OnCalendar=minutely  # every minute
```

### cron

Add a line to your crontab with `crontab -e`:

```cron
*/5 * * * * NOTIFICATION_URL=https://ntfy.sh/my-weather-alerts /usr/local/bin/now-you-noaa
```

Or if you prefer an env file:

```cron
*/5 * * * * env $(cat /etc/now-you-noaa/env | xargs) /usr/local/bin/now-you-noaa
```

Logs go to syslog (`journalctl -t now-you-noaa` or `/var/log/syslog`).

## How It Works

1. Load config from `CONFIG_FILE` (default: `/etc/now-you-noaa/config.json`)
2. Load notification state from `STATE_FILE`, pruning entries older than `pruneAfterDays`
3. Fetch active alerts from the NOAA API for all configured `areas` and `zones`
4. Filter alerts by `eventTypes` and `severity` (if configured)
5. For each matching alert, check whether a notification was already sent
6. If not, POST the notification payload to the configured URL and record the alert ID in state
7. Save state to disk

The state file is the single source of truth for idempotency. As long as it persists across runs, no alert will ever trigger more than one notification.

## Versioning

Releases are created automatically on every push to `main`. The version follows CalVer: `YYYY.MM.DD.N` where N is the GitHub Actions run number (e.g. `2026.06.16.1`). No manual tagging required.

The version is embedded in the binary at build time:

```bash
now-you-noaa --version
# 2026.06.16.1
```

When running locally from source, `--version` prints `dev`.

The latest release is always available at `https://github.com/rorpage/now-you-noaa/releases/latest`.
