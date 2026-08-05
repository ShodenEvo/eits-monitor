# Windows agent application

The Windows release is a single installer executable:

`EITS-Agent-Setup-vX.Y.Z.exe`

The installer asks for the EITS server URL, enrollment token, device name, and whether insecure HTTP is allowed. It then installs:

- `EITSAgent`, a genuine automatic Windows service running the monitoring collector.
- `eits-agent-manager.exe`, a desktop interface for status, logs, service controls, dashboard access, and updates.
- `eits-agent-updater.exe`, a privileged updater with checksum verification, backup, and rollback.

Configuration, identity, queue, and logs are stored under `C:\ProgramData\EITS\Agent`. Closing the manager does not stop monitoring because collection runs in the Windows service.

## Update hosting

The updater checks this manifest by default:

`https://eits.myds.me/downloads/agent/windows/update.json`

Example:

```json
{
  "version": "0.5.0-alpha.1",
  "url": "https://eits.myds.me/downloads/agent/windows/eits-agent-update-v0.5.0-alpha.1.zip",
  "sha256": "REPLACE_WITH_PACKAGE_SHA256",
  "mandatory": false,
  "release_notes": "Windows service, manager interface, and native updater."
}
```

The update ZIP must contain `eits-agent-service.exe`, `eits-agent-manager.exe`, and `eits-agent-updater.exe` at its root.
