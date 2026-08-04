#!/usr/bin/env bash
set -euo pipefail
if [[ $EUID -ne 0 ]]; then echo "Run with sudo." >&2; exit 1; fi
if [[ $# -lt 2 ]]; then echo "Usage: sudo ./install-agent.sh SERVER_URL ENROLLMENT_TOKEN [DEVICE_NAME] [--allow-insecure-http]"; exit 2; fi
SERVER_URL="$1"; TOKEN="$2"; NAME="${3:-$(hostname)}"; ALLOW=false
[[ "${4:-}" == "--allow-insecure-http" ]] && ALLOW=true
ARCH=$(uname -m); case "$ARCH" in x86_64) BIN=eits-agent-linux-amd64;; aarch64|arm64) BIN=eits-agent-linux-arm64;; *) echo "Unsupported architecture: $ARCH"; exit 1;; esac
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
install -m 0755 "$ROOT/dist/$BIN" /usr/local/bin/eits-agent
install -d -m 0750 /etc/eits-agent /var/lib/eits-agent /var/log/eits-agent
cat >/etc/eits-agent/config.json <<JSON
{
  "server_url": "$SERVER_URL",
  "enrollment_token": "$TOKEN",
  "device_name": "$NAME",
  "collection_interval_seconds": 30,
  "request_timeout_seconds": 15,
  "allow_insecure_http": $ALLOW,
  "skip_tls_verify": false,
  "state_directory": "/var/lib/eits-agent",
  "log_directory": "/var/log/eits-agent",
  "queue": {"enabled": true, "maximum_records": 2880},
  "logging": {"level": "info", "maximum_size_mb": 10, "maximum_files": 5}
}
JSON
chmod 0600 /etc/eits-agent/config.json
install -m 0644 "$ROOT/scripts/eits-agent.service" /etc/systemd/system/eits-agent.service
systemctl daemon-reload
systemctl enable --now eits-agent
systemctl --no-pager status eits-agent
