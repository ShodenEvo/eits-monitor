# EITS Native Monitoring Agent v0.3.0

Native outbound-only monitoring agent for the EITS Monitor server.

## Supported builds

- Windows x64
- Linux x64
- Linux ARM64

## Metrics

- CPU utilization
- Physical memory utilization
- Local disk utilization
- Uptime
- Linux network byte counters
- Server-configured TCP checks
- Server-configured UDP checks

The Windows v0.3 collector reports network counters as zero. Windows interface counters will be added with the process/service monitoring revision.

## Security

- No inbound listening port
- Unique per-device credential after enrollment
- Enrollment token removed from `config.json` after successful enrollment
- TLS 1.2 minimum for HTTPS
- Plain HTTP must be explicitly enabled for trusted development networks
- Bounded local offline queue

## Windows installation

Copy the `windows` package to the target machine, open an elevated PowerShell prompt in that folder, and run:

```powershell
Set-ExecutionPolicy -Scope Process Bypass
.\install-agent.ps1 `
  -ServerUrl "http://10.0.0.4:8088" `
  -EnrollmentToken "YOUR_ENROLLMENT_TOKEN" `
  -DeviceName "WINDOWS-PC01" `
  -AllowInsecureHttp
```

The installer creates a SYSTEM startup task named `EITS Monitoring Agent`. This avoids an external service-wrapper dependency while keeping the agent fully unattended. Files are installed under:

```text
C:\Program Files\EITS Agent\eits-agent.exe
C:\ProgramData\EITS\Agent\config.json
C:\ProgramData\EITS\Agent\identity.json
C:\ProgramData\EITS\Agent\logs\eits-agent.log
```

Diagnostics:

```powershell
& 'C:\Program Files\EITS Agent\eits-agent.exe' diagnostics
Get-Content 'C:\ProgramData\EITS\Agent\logs\eits-agent.log' -Tail 100
```

Uninstall:

```powershell
.\uninstall-agent.ps1
```

## Linux installation

```bash
sudo ./install-agent.sh \
  http://10.0.0.4:8088 \
  YOUR_ENROLLMENT_TOKEN \
  SERVER-LINUX01 \
  --allow-insecure-http
```

Status and diagnostics:

```bash
systemctl status eits-agent
journalctl -u eits-agent -f
sudo /usr/local/bin/eits-agent diagnostics
```

## Production HTTPS

Use an HTTPS URL and omit `--allow-insecure-http`. Do not use `skip_tls_verify` with a public or production server.
