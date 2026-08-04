# Agent Guide

## Docker host agent

The Docker agent is included in `docker-compose.yml`. It monitors the Linux host running the EITS Monitor stack.

## Native agent

The native agent source is under `native-agent/`.

### Supported platforms

- Windows AMD64
- Linux AMD64
- Linux ARM64

### Enrollment

The installer receives the server URL, temporary enrollment token, and display name. After successful enrollment, the token is replaced by a unique identity and credential stored locally.

### Communication

- Outbound API requests only
- Config refresh before metric collection
- Bounded local queue during outages
- Automatic retry
- TLS verification enabled by default

### Runtime files

Windows:

```text
C:\Program Files\EITS Agent\eits-agent.exe
C:\ProgramData\EITS\Agent\config.json
C:\ProgramData\EITS\Agent\identity.json
C:\ProgramData\EITS\Agent\metrics.queue
C:\ProgramData\EITS\Agent\logs\eits-agent.log
```

Linux:

```text
/usr/local/bin/eits-agent
/etc/eits-agent/config.json
/var/lib/eits-agent/identity.json
/var/lib/eits-agent/metrics.queue
/var/log/eits-agent/eits-agent.log
```
