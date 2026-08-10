# Changelog

## Unreleased

- Added authenticated server removal with complete deletion of metrics, inventory, checks, and process data.
- Added search and client-side pagination to the Network Checks section.
- Added cross-platform running-process inventory and persistent process availability monitoring.
- Displayed monitored processes in a table and added explicit confirmation dialogs for server and process-monitor deletion.
- Rebuilt the Windows Agent Manager as a modern, responsive .NET 8 WPF application with a system tray and single-instance behavior.
- Added console-free C# Windows service and elevated control applications while retaining the Go collection engine.
- Reworked connection changes with secure temporary requests, health checks, detailed enrollment errors, and rollback.
- Updated the Windows installer and release workflow for self-contained .NET executables.

## [0.4.0-alpha.1] - 2026-08-04

### Added
- Device hardware and operating-system inventory endpoint.
- Windows and Linux inventory collection for manufacturer, model, OS build, CPU topology, total RAM, BIOS, latest detected update, and GPUs.
- Compact CPU, OS update, and total/used RAM information on overview cards.
- Detailed Hardware & Operating System section on the device page.
- Daily inventory refresh with an immediate scan on agent startup.


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
