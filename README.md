# EITS Monitor

> [!WARNING]
> **Alpha software.** EITS Monitor is currently intended for home labs, test environments, and community evaluation. Breaking changes may occur, and the project has not undergone an external security audit.

EITS Monitor is a lightweight, self-hosted infrastructure monitoring platform with a responsive web dashboard and cross-platform Go agents. It monitors host resources, disk capacity, uptime, and configurable TCP/UDP endpoints without exposing inbound agent ports.

## Current features

- Docker Compose deployment for the server stack
- FastAPI backend and PostgreSQL database
- React + TypeScript responsive dashboard
- Docker-host monitoring agent
- Native Windows, Linux AMD64, and Linux ARM64 agents
- CPU, memory, disk, uptime, and basic network metrics
- Configurable disk warning and critical thresholds
- TCP and UDP endpoint checks
- Per-device enrollment credentials
- Metric history and device health states
- Smartphone-friendly interface

## Screenshots

![Device monitoring details](docs/screenshots/device-details.png)

![TCP and UDP network checks](docs/screenshots/network-checks.png)

## Architecture

```text
Native Windows/Linux Agents ─┐
                             ├── HTTPS/HTTP API ── FastAPI ── PostgreSQL
Docker Host Agent ───────────┘                         │
                                                      └── React/Nginx UI
```

Agents initiate outbound communication only. They enroll once using a temporary enrollment token and then use a unique device credential.

## Quick start

### Requirements

- Linux host with Docker Engine and Docker Compose v2
- 2 GB RAM minimum for a small test deployment
- A modern browser

### 1. Configure

```bash
cp .env.example .env
```

Generate secrets:

```bash
openssl rand -hex 64
openssl rand -hex 32
openssl rand -hex 24
```

Edit `.env` and replace every placeholder value.

### 2. Start

```bash
docker compose up -d --build
```

Open:

```text
http://SERVER-IP:8088
```

### 3. Check status

```bash
docker compose ps
docker compose logs --tail=100 api agent web
```

## Native agents

Source and installers are under [`native-agent/`](native-agent/).

Supported build targets:

- Windows AMD64
- Linux AMD64
- Linux ARM64

Build all targets:

```bash
cd native-agent
./scripts/build.sh
```

PowerShell:

```powershell
cd native-agent
.\scripts\build.ps1
```

Release binaries should be attached to GitHub Releases rather than committed to the repository.

### Windows installation

Run PowerShell as Administrator:

```powershell
Set-ExecutionPolicy -Scope Process Bypass

.\install-agent.ps1 `
  -ServerUrl "https://monitor.example.com" `
  -EnrollmentToken "YOUR_ENROLLMENT_TOKEN" `
  -DeviceName "SERVER-WIN01"
```

For isolated HTTP testing only, add:

```powershell
-AllowInsecureHttp
```

### Linux installation

```bash
sudo ./install-agent.sh \
  https://monitor.example.com \
  YOUR_ENROLLMENT_TOKEN \
  SERVER-LINUX01
```

## TCP and UDP checks

TCP checks confirm that a connection can be established.

Generic UDP checks have inherent limitations because UDP has no connection handshake. A UDP send without a response can confirm only that the datagram was sent without an immediate local error. Enable **Require response** when the service returns a known response. Protocol-specific DNS, NTP, and SNMP probes are planned.

## Security notes

- Use HTTPS for production deployments.
- Rotate enrollment tokens periodically.
- Never commit `.env`, agent identity files, database dumps, or private keys.
- The Docker host agent mounts host resources read-only and should be treated as trusted infrastructure.
- EITS Monitor does not provide arbitrary remote-command execution.

Read [`SECURITY.md`](SECURITY.md) before exposing the application outside a trusted network.

## Documentation

- [Architecture](docs/architecture.md)
- [Agent guide](docs/agents.md)
- [Troubleshooting](docs/troubleshooting.md)
- [Roadmap](ROADMAP.md)
- [Contributing](CONTRIBUTING.md)

## Release status

Current version: **v0.3.0-alpha.1**

See [`CHANGELOG.md`](CHANGELOG.md).

## License

Licensed under the Apache License 2.0. See [`LICENSE`](LICENSE).
