# Windows agent application

The Windows release is a single installer executable:

`EITS-Agent-Setup-vX.Y.Z.exe`

The installer asks for the EITS server URL, enrollment token, device name, and whether insecure HTTP is allowed. The Windows integration is implemented in C#/.NET 8, with the portable collection engine retained in Go. It installs:

- `EITSAgent`, a genuine automatic Windows service running the monitoring collector.
- `Eits.Agent.Manager.exe`, a responsive WPF desktop and system-tray interface for status, registration, logs, dashboard access, and elevated service controls.
- `Eits.Agent.Service.exe`, the Windows Service host for the Go collection engine.
- `Eits.Agent.Control.exe`, a console-free elevated helper for native service control and protected configuration changes.
- `eits-agent-engine.exe`, the portable Go collection and server-transport engine.
- `eits-agent-updater.exe`, a privileged updater with checksum verification, backup, and rollback.

Configuration, identity, queue, and logs are stored under `C:\ProgramData\EITS\Agent`. Closing the manager does not stop monitoring because collection runs in the Windows service.

The manager refreshes automatically every 10 seconds. Starting, stopping, or restarting the service requests Windows administrator approval only when required. Agent update controls remain disabled until the update workflow is released.

Service operations run outside the interface thread through the Windows Service API and have a 25-second timeout. No PowerShell or console window is used; Windows still displays its standard UAC consent prompt. **Change connection** accepts a new server URL, enrollment token, and device name. The token is passed through a short-lived request file instead of command-line arguments. The helper checks server health, backs up the configuration and identity, restarts the service, waits for enrollment, and rolls back on failure. Reconnecting may leave the previous device record on the old server for an administrator to remove.

Before changing protected files, the Manager checks `<server URL>/api/health`. When using the default Compose web port on a trusted HTTP network, include it in the address, for example `http://192.0.2.10:8088`, and enable **Allow insecure HTTP**.

The Manager provides a notification-area icon. Closing its window hides the Manager to the system tray without stopping monitoring. The tray menu can open or exit the Manager and start, stop, or restart the service.

## Future update hosting

The reserved updater implementation checks this manifest by default. It is not currently exposed as an Agent Manager option:

`https://monitor.example.com/downloads/agent/windows/update.json`

Example:

```json
{
  "version": "0.5.0-alpha.1",
  "url": "https://monitor.example.com/downloads/agent/windows/eits-agent-update-v0.5.0-alpha.1.zip",
  "sha256": "REPLACE_WITH_PACKAGE_SHA256",
  "mandatory": false,
  "release_notes": "Windows service, manager interface, and native updater."
}
```

The update ZIP must contain `eits-agent-service.exe`, `eits-agent-manager.exe`, and `eits-agent-updater.exe` at its root.
