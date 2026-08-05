# Native Windows implementation status

Implemented in source:

- Windows Service conversion for the monitoring agent.
- Desktop manager EXE with service state, registration state, server/device details, last activity, service controls, logs, dashboard, and update launcher.
- Native updater EXE with website manifest, SHA-256 verification, staging, backup, replacement, restart verification, and rollback.
- Single interactive Inno Setup EXE that collects enrollment settings, writes configuration, installs and starts the service, and starts the manager.
- GitHub Actions release build for the Windows installer and Linux agent package.

The generated Windows binaries must be built by the GitHub Actions Windows runner or a Windows development machine. They were not executed in this Linux workspace.
