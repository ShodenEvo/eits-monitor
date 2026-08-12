# Changelog

## Unreleased

- Escalate device health from port, monitored-process, disk, and reporting failures with visible overview explanations.
- Simplify network checks to device-local port monitoring by removing host entry and host display from the dashboard.
- Allow fresh Docker agent builds when the dependency-free Go module has no generated `go.sum` file.
- Correct Linux disk reporting by reading the host mount namespace and excluding container bind mounts and virtual filesystems.
- Add threshold-aware disk colors, a CPU percentage suffix, and pagination for monitored processes.
- Use consistent CPU and memory labels with percentage units displayed alongside both values.

## [0.5.0-alpha.12] - 2026-08-11

### Fixed

- Corrected main-window card alignment and Quick Access content overflow at different window sizes.
- Increased the connection-dialog working area for high-DPI Windows display scaling.
- Kept accent-button text readable across normal, hover, pressed, and disabled states.
- Rechecked exclusive access to the service executable after shutdown and immediately before replacement.
- Added bounded cleanup of lingering EITS service, engine, manager, and updater processes during upgrades.

## [0.5.0-alpha.11] - 2026-08-11

### Fixed

- Stop and verify the existing `EITSAgent` service before Setup replaces application files.
- Abort upgrades with the actual service-control error when Windows cannot stop the service.
- Prevent Windows Restart Manager from reopening old agent processes during an upgrade.
- Restart and verify the updated service only after all new binaries are installed.

## [0.5.0-alpha.10] - 2026-08-11

### Changed

- Redesigned the Windows Agent Manager as a futuristic dark command center with responsive status, uplink, telemetry, operations, and quick-access panels.
- Restyled the connection workflow to match the new agent interface.
- Added an original EITS monitoring shield icon to the application window, executable, system tray, shortcuts, and installer.

## [0.5.0-alpha.9] - 2026-08-11

### Fixed

- Prompt for a fresh server and enrollment token when an incomplete installation has a configuration file but no agent identity.
- Preserve connection settings automatically only when the existing agent has completed enrollment.

## [0.5.0-alpha.8] - 2026-08-11

### Fixed

- Replaced the installer PowerShell service-registration step with the native control helper.
- Made installation fail with a useful message when Windows service creation or startup fails.
- Created the ProgramData log directory and initial installation log before service startup.
- Embedded the package version in Windows and Linux collection-engine builds.

### Documentation

- Refreshed the dashboard screenshots.
- Release notes are now uploaded from a Markdown file so GitHub renders line breaks correctly.

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
