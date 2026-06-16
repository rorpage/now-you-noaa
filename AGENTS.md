# AGENTS.md

Context for AI agents (Claude Code, Copilot, etc.) working in this repository.

## What this project does

Now You NOAA is a one-shot Go binary for home servers. It queries the NOAA Weather API for active weather alerts in configured U.S. states or forecast zones, finds alerts not yet notified, and sends a single webhook notification per alert. Idempotency is maintained via a JSON state file. It runs directly on Linux or macOS -- no Docker, no runtime dependencies.

## Repository layout

```
main.go       -- entry point; orchestrates config, NOAA fetch, notify, state
config.go     -- config types, loading from JSON + env var overrides
noaa.go       -- NOAA API fetch, response parsing, alert filtering
notifier.go   -- builds notification payload (webhook/slack/discord/template), POSTs to webhook URL
state.go      -- reads/writes/prunes the state file

go.mod                -- module definition; no external dependencies
config.example.json   -- copy to config.json (gitignored) to run locally

.github/workflows/
  publish.yml   -- on push to main: cross-compiles binaries for 5 platforms, creates GitHub Release

deploy/
  systemd/
    now-you-noaa.service  -- oneshot unit; runs binary as now-you-noaa user
    now-you-noaa.timer    -- fires every 5 minutes, Persistent=true
    env.example           -- template for /etc/now-you-noaa/env (holds NOTIFICATION_URL)
    install.sh            -- downloads binary (or builds from source), creates user/dirs, enables timer
```

## Key design decisions

- **Single static binary, no runtime dependencies.** Go's standard library handles HTTP and JSON. CGO is disabled so the binary runs on any Linux/macOS without glibc or other shared libraries.
- **No Docker.** This project targets home server use. Docker adds path, permission, and user complexity that the Go binary approach eliminates entirely.
- **One-shot execution.** The binary runs, checks for alerts, and exits. Scheduling is the caller's responsibility (cron or systemd timer).
- **State file for idempotency.** `state.json` records notified alert IDs with timestamps. Entries older than `pruneAfterDays` (default 7) are pruned on each run. If a notification POST fails, the alert ID is not recorded, so it will be retried on the next run.
- **Config-file-first, env-var override.** `NOTIFICATION_URL`, `CONFIG_FILE`, and `STATE_FILE` env vars override their config file equivalents. Keep the notification URL in an env var to avoid committing secrets.
- **Client-side filtering.** `eventTypes` and `severity` filters are applied after fetching from the NOAA API rather than via query parameters. This keeps the fetch logic simple and ensures consistent behavior regardless of API parameter support changes.
- **Deduplication across areas and zones.** When both `areas` and `zones` are configured, separate API calls are made and results are merged by alert ID before filtering and notification.
- **User-Agent compliance.** NOAA's API guidelines request that clients identify themselves via User-Agent. The default user agent identifies this project. Override with the `userAgent` config field.

## Config fields

| Field | Required | Default | Description |
|---|---|---|---|
| `areas` | See note | -- | Array of state abbreviations to monitor (e.g. `["IN", "OH"]`) |
| `zones` | See note | -- | Array of NOAA zone codes (e.g. `["INZ032"]`) |
| `eventTypes` | No | all | Alert event type strings to match. Empty = all events. |
| `severity` | No | all | Severity levels to match: `Extreme`, `Severe`, `Moderate`, `Minor`. Empty = all. |
| `userAgent` | No | built-in | User-Agent header for NOAA API requests |
| `notificationUrl` | See note | -- | Webhook URL to POST alerts to |
| `notificationMethod` | No | `POST` | HTTP method for notifications |
| `notificationHeaders` | No | -- | Extra headers |
| `notificationType` | No | `webhook` | Payload format: `webhook`, `slack`, `discord`, or `template` |
| `notificationTemplate` | If `template` | -- | Go template string |
| `stateFilePath` | No | `/var/lib/now-you-noaa/state.json` | Where to persist notification state |
| `pruneAfterDays` | No | `7` | Days to keep state entries before pruning |

At least one of `areas` or `zones` is required. Notification URL is required (via env var or config).

## Default file paths

| Purpose | Default |
|---|---|
| Config | `/etc/now-you-noaa/config.json` |
| State | `/var/lib/now-you-noaa/state.json` |

## NOAA API

Base URL: `https://api.weather.gov/alerts/active`

Query parameters used:
- `status=Actual` -- always set; excludes Test, Exercise, System, Draft alerts
- `area=IN,OH` -- comma-separated state abbreviations (built from `areas` config)
- `zone=INZ032` -- comma-separated zone codes (built from `zones` config)

The response is a GeoJSON FeatureCollection. Each feature's `id` field (a URL like `https://api.weather.gov/alerts/urn:oid:...`) is used as the unique alert identifier for state tracking. The `properties` object contains all alert metadata.

Key `properties` fields:
- `event` -- event type string (e.g. `"Tornado Warning"`)
- `headline` -- short human-readable summary (nullable)
- `severity` -- `Extreme`, `Severe`, `Moderate`, `Minor`, or `Unknown`
- `urgency` -- `Immediate`, `Expected`, `Future`, `Past`, or `Unknown`
- `certainty` -- `Observed`, `Likely`, `Possible`, `Unlikely`, or `Unknown`
- `areaDesc` -- human-readable area description
- `senderName` -- issuing NWS office
- `sent`, `effective`, `onset` -- ISO 8601 timestamps
- `expires` -- when the alert expires
- `ends` -- when the hazard ends (nullable; distinct from expiry)
- `description` -- full alert text (nullable)
- `instruction` -- safety instructions (nullable)

NOAA requires a descriptive User-Agent header. The default is `now-you-noaa (https://github.com/rorpage/now-you-noaa)`.

## Notification payload shape

```go
type alertResult struct {
    ID          string `json:"id"`
    Event       string `json:"event"`
    Headline    string `json:"headline"`
    Severity    string `json:"severity"`
    Urgency     string `json:"urgency"`
    Certainty   string `json:"certainty"`
    AreaDesc    string `json:"areaDesc"`
    SenderName  string `json:"senderName"`
    Sent        string `json:"sent"`
    Effective   string `json:"effective"`
    Onset       string `json:"onset"`
    Expires     string `json:"expires"`
    Ends        string `json:"ends"`
    Description string `json:"description"`
    Instruction string `json:"instruction"`
}

type notificationPayload struct {
    Alert   alertResult `json:"alert"`
    Summary string      `json:"summary"`
}
```

`Summary` is the alert's `Headline` if non-empty; otherwise `"<Event>: <AreaDesc>"`.

## Development workflow

```bash
go build ./...              # compile
go vet ./...                # static analysis
go build -o now-you-noaa . && CONFIG_FILE=config.json NOTIFICATION_URL=http://localhost:3001 ./now-you-noaa
```

To test locally without a real webhook, run a listener in another terminal:

```bash
python3 -m http.server 3001
```

To inspect live NOAA data without running the full binary:

```bash
curl -s "https://api.weather.gov/alerts/active?status=Actual&area=IN" | jq '.features[].properties.event'
```

## Versioning

Releases use CalVer: `YYYY.MM.DD.N` where N is `github.run_number`. The version is embedded at build time via `-ldflags "-X main.version=..."` into the `version` variable in `main.go`. Running from source prints `dev`. No manual tagging; every push to main creates a release.

## GitHub Actions

`.github/workflows/publish.yml` triggers on every push to `main`. It computes the CalVer string, cross-compiles for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, and windows/amd64 (with the version embedded via ldflags), then creates a GitHub Release tagged with the CalVer string using `softprops/action-gh-release` with `make_latest: true`.

## Scheduling

Two scheduling options are documented in README.md:
- **systemd timer** -- recommended; dedicated user; logs via `journalctl`; deploy files in `deploy/systemd/`
- **cron** -- simpler; one crontab line; logs go to syslog

When changing the default schedule (currently every 5 minutes), update `OnCalendar` in `deploy/systemd/now-you-noaa.timer` and the examples in README.md.

## Style conventions

- Standard Go idioms; run `go vet` before committing
- No external dependencies -- standard library only
- Log lines are prefixed with `[module]` (e.g. `[noaa]`, `[notify]`, `[state]`, `[config]`)
- Do not write comments that explain what the code does -- only add one when the WHY is non-obvious
- No em dashes anywhere in code or documentation

## Files to keep updated

When making changes, keep README.md and AGENTS.md in sync:
- New config fields -> Config fields table in README.md and this file
- New env vars -> Environment Variables table in README.md and this file
- Architectural changes -> Key design decisions section in this file
