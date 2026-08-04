# GitHub Repository Setup

## Recommended repository settings

Repository name:

```text
eits-monitor
```

Description:

```text
A lightweight, self-hosted server and device monitoring platform with cross-platform Go agents, FastAPI, React, PostgreSQL, TCP/UDP checks, disk thresholds, and a responsive dashboard.
```

Visibility:

```text
Public
```

Do not initialize the GitHub repository with a README, `.gitignore`, or license because this package already contains them.

Topics:

```text
monitoring
server-monitoring
self-hosted
golang
fastapi
react
postgresql
docker
windows-agent
linux-agent
homelab
infrastructure-monitoring
```

## Publish using the GitHub website and Git

1. Create an empty public repository named `eits-monitor`.
2. Copy its HTTPS repository URL.
3. Run the commands below from this package directory.

```bash
git init
git branch -M main
git add .
git status
git commit -m "Initial public alpha release"
git remote add origin https://github.com/YOUR_GITHUB_USERNAME/eits-monitor.git
git push -u origin main
```

## Publish using GitHub CLI

Log in once:

```bash
gh auth login
```

Then run from this directory:

```bash
git init
git branch -M main
git add .
git commit -m "Initial public alpha release"
gh repo create eits-monitor \
  --public \
  --source=. \
  --remote=origin \
  --description "A lightweight, self-hosted server and device monitoring platform with cross-platform Go agents, FastAPI, React, PostgreSQL, TCP/UDP checks, disk thresholds, and a responsive dashboard." \
  --push
```

## Create the first alpha release

After CI succeeds:

```bash
git tag -a v0.3.0-alpha.1 -m "EITS Monitor v0.3.0-alpha.1"
git push origin v0.3.0-alpha.1
```

The release workflow builds and attaches Windows AMD64, Linux AMD64, and Linux ARM64 agents with SHA-256 checksums.

## Recommended GitHub page configuration

After the first push:

1. Open **About** and add the description and topics above.
2. Open **Settings → General → Features** and keep Issues enabled.
3. Open **Settings → Code security and analysis** and enable Dependabot alerts and secret scanning when available.
4. Open **Settings → Branches** and protect `main` after the first successful CI run.
5. Require pull requests and the CI status checks before merging.
6. Open **Security → Advisories** to confirm private vulnerability reporting is available.
7. Create these labels: `bug`, `enhancement`, `documentation`, `security`, `agent`, `backend`, `frontend`, `good first issue`.

## Suggested first GitHub issues

- Add process monitoring to native agents
- Add Windows Service monitoring
- Add systemd service monitoring
- Add Docker container monitoring
- Implement persistent alert lifecycle
- Add email and webhook notifications
- Introduce Alembic database migrations
- Add HTTP/HTTPS and TLS certificate probes
- Add protocol-aware DNS, NTP, and SNMP checks
- Add backup and restore documentation
