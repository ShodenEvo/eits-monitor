# Changelog

All notable changes to EITS Monitor are documented here.

## [0.3.0-alpha.1] - 2026-08-04

### Added

- Docker Compose server stack with PostgreSQL, FastAPI, React/Nginx, and Docker host agent
- Native Go agent for Windows AMD64, Linux AMD64, and Linux ARM64
- CPU, memory, disk, uptime, and basic network monitoring
- TCP and UDP endpoint checks
- Device enrollment and per-agent credentials
- Responsive desktop and mobile dashboard
- Configurable disk warning and critical thresholds
- Offline metric queue in the native agent
- Windows and Linux installation scripts

### Fixed

- Empty Go result slices no longer serialize as invalid `null` payloads
- Port-check deletion now removes dependent historical results safely
- Device details refresh after adding or deleting checks
- Frontend chart date formatting handles optional Recharts labels safely
- Windows installer resolves its packaged executable correctly

### Security

- Enrollment tokens are used only for initial registration
- Agents use unique credentials after enrollment
- Agent communication is outbound-only
