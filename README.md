# Rhythms

A self-hosted habit tracker — a from-scratch rewrite of [uHabits](https://github.com/iSoron/uhabits) in Go, HTMX, and SQLite. Single binary, single SQLite file, no accounts, runs on your own hardware.

Ports uHabits' actual algorithms (interval-snapping auto-fill, streak detection, EMA scoring) from its source rather than reimplementing them from a description, so the numbers behave the way the original app's do.

## Features

- Boolean and numeric habits, with arbitrary frequency ("3 times per 7 days", not just daily/weekly)
- Streak and score (EMA) tracking, with SVG charts (score line, best-streaks, calendar heatmap, day-of-week breakdown) — no client-side charting library
- SKIP/UNKNOWN checkmark states, drag-to-reorder, archive, per-habit CSV export
- A `/today` due-list, installable as a PWA, with Web Push reminders
- One-click, guaranteed-consistent SQLite backup download

## Quick start

### Docker Compose (recommended)

```sh
docker compose up -d
```

Rhythms is now at `http://localhost:8080`, with its data in a named Docker volume (`rhythms-data`).

### Docker

```sh
docker build -t rhythms .
docker run -d -p 8080:8080 -v rhythms-data:/data --name rhythms rhythms
```

### From source

Requires Go 1.24+.

```sh
make build
./bin/rhythms
```

### systemd (bare-metal Linux, no Docker)

See [`deploy/rhythms.service`](deploy/rhythms.service) for a unit file and install steps.

## Configuration

All configuration is environment variables — there's no config file.

| Variable | Default | Purpose |
|---|---|---|
| `RHYTHMS_ADDR` | `:8080` | Address/port to listen on |
| `RHYTHMS_DB_PATH` | `rhythms.db` | Path to the SQLite database file |
| `RHYTHMS_BASIC_AUTH_USER` | *(unset)* | Username for HTTP basic auth |
| `RHYTHMS_BASIC_AUTH_PASS` | *(unset)* | Password for HTTP basic auth |
| `RHYTHMS_SKIP_ENABLED` | `true` | Whether the checkmark can reach SKIP/UNKNOWN states |
| `RHYTHMS_VAPID_SUBJECT` | `mailto:admin@localhost` | Contact URL/email sent to push services with each notification (set this to your own — required by the Web Push spec) |

Basic auth is off by default, which is fine for a device only reachable on your own LAN. If Rhythms is exposed to the wider internet, set `RHYTHMS_BASIC_AUTH_USER`/`_PASS` (or put it behind a reverse proxy that handles auth).

Web Push needs a real `RHYTHMS_VAPID_SUBJECT` to be a good citizen toward push services, but reminders work without setting it — the default is just a placeholder contact.

## Reminders and notifications

Set a reminder time (and which days) when creating or editing a habit. To actually receive them:

1. Open `/today` in a browser.
2. Click **Enable reminders on this device**, and allow the notification permission prompt.

Rhythms polls once a minute for reminders that are due, not yet completed, and not already sent that day, and pushes a browser notification. This requires HTTPS (or `localhost`) — browsers won't allow Web Push over plain HTTP on a real hostname, so a reverse proxy with TLS (Caddy, nginx, Tailscale, etc.) is required for reminders to work outside of local testing.

## Backup and restore

**Backup**: visit `/backup` to download a consistent snapshot of the whole database (uses SQLite's `VACUUM INTO`, so it's safe to do this while the app is running).

**Restore**: stop the app, replace the database file at `RHYTHMS_DB_PATH` with the backup, and start it again. There's no in-app restore flow — this is a deliberate scope decision, since a live-restore endpoint on a running database is a meaningfully riskier feature than a download button.

**CSV export**: each habit's detail page links to a CSV export of its full history, in the same `Date,Value,Notes` shape uHabits itself uses.

## Development

```sh
make build   # build ./bin/rhythms
make run     # build and run
make test    # go test ./...
make tidy    # go mod tidy
```

Project layout:

```
cmd/rhythms/       main() — wiring, config, server + scheduler startup
internal/domain/   pure logic: habit/entry model, frequency & streak & score algorithms
internal/store/    SQLite persistence (implements the domain package's repo interfaces)
internal/web/      HTTP handlers, routing, HTML templates rendering
internal/charts/   server-rendered SVG charts
internal/reminder/ the Web Push reminder scheduler
web/templates/     html/template files
web/static/        CSS, vendored JS (htmx, SortableJS), service worker, icons
```

Migrations live in `internal/store/migrations/` and run automatically on startup — there's no separate migrate step.

## Releasing

Cross-compiled binaries (linux/darwin, amd64/arm64) via [goreleaser](https://goreleaser.com):

```sh
goreleaser build --snapshot --clean   # local test build, no publishing
goreleaser release --clean            # real release — needs a git tag and a configured remote
```

## What's deliberately different from uHabits

- No Android home-screen widgets (no web equivalent) — replaced by the installable PWA `/today` view.
- Web Push instead of native OS notifications, layered on top of the simpler in-app `/today` list.
- Drag-reorder uses a small vendored JS library (SortableJS) — the one place this project isn't pure HTMX, since drag-and-drop isn't something HTMX covers on its own.
- Charts are server-rendered SVG, not an interactive JS charting library — no hover tooltips, but zero client-side chart bundle.
- Backup/restore is a raw SQLite file (matching uHabits' own approach), not a structured export/import format.
- Single shared user, optional HTTP basic auth — no accounts, sessions, or per-user data.
