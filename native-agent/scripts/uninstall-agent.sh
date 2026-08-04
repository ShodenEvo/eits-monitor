#!/usr/bin/env bash
set -euo pipefail
if [[ $EUID -ne 0 ]]; then echo "Run with sudo." >&2; exit 1; fi
systemctl disable --now eits-agent 2>/dev/null || true
rm -f /etc/systemd/system/eits-agent.service /usr/local/bin/eits-agent
systemctl daemon-reload
printf 'Program removed. Configuration/state retained under /etc/eits-agent and /var/lib/eits-agent.\n'
