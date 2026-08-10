# Windows implementation status

Implemented in source:

- .NET 8 Windows Service host supervising the portable Go collection engine.
- Responsive WPF manager with automatic status refresh, system tray, single-instance behavior, registration details, logs, dashboard, and asynchronous native service controls.
- Console-free elevated helper with protected connection requests, health checks, enrollment verification, and rollback.
- Native updater EXE with website manifest, SHA-256 verification, staging, backup, replacement, restart verification, and rollback.
- Single interactive Inno Setup EXE that collects enrollment settings, writes configuration, installs and starts the service, and starts the manager.
- GitHub Actions release build for the Windows installer and Linux agent package.

The Agent Manager shows update management as a disabled future feature. The reserved updater is not launched from the manager in this release.

The generated Windows binaries are built by the GitHub Actions Windows runner or a Windows development machine. The current development build was compiled and smoke-tested on the Scriptcase-Dev workstation.
