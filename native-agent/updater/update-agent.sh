#!/usr/bin/env bash
set -euo pipefail
MODE=${EITS_UPDATE_MODE:-automatic}; CHANNEL=${EITS_UPDATE_CHANNEL:-prerelease}; REPO=${EITS_UPDATE_REPOSITORY:-ShodenEvo/eits-monitor}
[[ "$MODE" == disabled ]] && exit 0
BIN=/usr/local/bin/eits-agent; LOG=/var/log/eits-agent/update.log; mkdir -p /var/log/eits-agent
log(){ printf '%s %s\n' "$(date -Is)" "$*" >>"$LOG"; }
api="https://api.github.com/repos/$REPO/releases?per_page=20"
headers=(-H 'Accept: application/vnd.github+json' -H 'X-GitHub-Api-Version: 2022-11-28'); [[ -n "${GITHUB_TOKEN:-}" ]] && headers+=(-H "Authorization: Bearer $GITHUB_TOKEN")
json=$(curl -fsSL "${headers[@]}" "$api")
python3 - "$CHANNEL" > /tmp/eits-release.$$ <<'PY' <<<"$json"
import json,sys
channel=sys.argv[1]; rs=[r for r in json.load(sys.stdin) if not r['draft']]
if channel=='stable': rs=[r for r in rs if not r['prerelease']]
if not rs: raise SystemExit(2)
r=rs[0]; a=next((a for a in r['assets'] if a['name'].startswith('eits-agent-linux-v') and a['name'].endswith('.tar.gz')),None)
if not a: raise SystemExit(3)
print(r['tag_name'].lstrip('v')); print(a['browser_download_url']); print((a.get('digest') or '').removeprefix('sha256:'))
PY
mapfile -t meta </tmp/eits-release.$$; rm -f /tmp/eits-release.$$
latest=${meta[0]}; url=${meta[1]}; expected=${meta[2]:-}; current=$($BIN version)
[[ "$latest" == "$current" ]] && { log "Already current: $current"; exit 0; }
log "Update available $current -> $latest"; [[ "$MODE" == notify ]] && exit 10
tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
pkg="$tmp/agent.tar.gz"; curl -fsSL "$url" -o "$pkg"
[[ -n "$expected" ]] && echo "$expected  $pkg" | sha256sum -c -
tar -xzf "$pkg" -C "$tmp"
arch=$(uname -m); case "$arch" in x86_64) name=eits-agent-linux-amd64;; aarch64|arm64) name=eits-agent-linux-arm64;; *) exit 4;; esac
new=$(find "$tmp" -type f -name "$name" | head -1); [[ -n "$new" ]]
[[ "$($new version)" == "$latest" ]]
cp "$BIN" "$BIN.bak"; systemctl stop eits-agent; install -m 0755 "$new" "$BIN"
if systemctl start eits-agent && sleep 8 && systemctl is-active --quiet eits-agent; then rm -f "$BIN.bak"; log "Updated successfully to $latest"; else cp "$BIN.bak" "$BIN"; systemctl restart eits-agent; log "ERROR: rolled back update"; exit 1; fi
