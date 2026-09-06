# Deploying Rhythms on Proxmox (LXC)

## Build

Build directly inside the LXC (it already has Go 1.27 — no cross-compile toolchain needed since the sqlite driver is pure Go):

```sh
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o rhythms ./cmd/rhythms
```

The `rhythms` binary embeds all templates, static assets, and migrations — nothing else needs to ship with it.

## Install

```sh
useradd --system --home /opt/rhythms --shell /usr/sbin/nologin rhythms
mkdir -p /opt/rhythms/data
cp rhythms /opt/rhythms/rhythms
chown -R rhythms:rhythms /opt/rhythms

cp deploy/rhythms.service /etc/systemd/system/rhythms.service
systemctl daemon-reload
systemctl enable --now rhythms
journalctl -u rhythms -f
```

## Configuration

All settings are environment variables (or equivalent flags — see `internal/config/config.go`):

| Variable | Default | Notes |
|---|---|---|
| `RHYTHMS_ADDR` | `:8080` | listen address |
| `RHYTHMS_DB` | `data/rhythms.db` | SQLite file path |
| `RHYTHMS_SECURE_COOKIES` | `true` | **set to `false` only if serving plain HTTP on your LAN** — if you put a TLS-terminating reverse proxy (e.g. Caddy, or Tailscale serve) in front of this, set it back to `true` |
| `RHYTHMS_VAPID_SUBSCRIBER` | `mailto:admin@localhost` | contact URI sent in the Web Push VAPID JWT — set to a real `mailto:` or `https:` address you control |
| `RHYTHMS_DEBUG` | `false` | enables `POST /debug/test-push` for verifying push notifications end to end — leave off in normal operation |

The VAPID keypair is generated once on first run and stored in the `app_config` table inside the SQLite file itself, so it survives restarts and travels with your backups.

## Reaching it from your phone

- On your home Wi-Fi: browse straight to `http://<lxc-ip>:8080`, then use the browser's "Add to Home Screen" / install prompt.
- From outside your home network: put it behind whatever you already use for remote access (Tailscale, a reverse proxy with a real TLS cert, etc.) rather than exposing the LXC directly — that's outside the scope of this app.

## Backups

The entire application state lives in one file: the SQLite database at `RHYTHMS_DB` (plus its `-wal`/`-shm` sidecar files while the server is running). Stopping the service and copying that file is a complete backup.
