# Security Policy

## Supported versions

| Version | Supported |
|---|---|
| 0.3.x alpha | Yes |
| Earlier versions | No |

## Reporting a vulnerability

Do not report security vulnerabilities through a public GitHub issue.

Use the repository's **Security** tab and submit a private vulnerability report through GitHub Security Advisories. Include affected versions, reproduction steps, impact, and any proposed mitigation.

## Deployment guidance

- EITS Monitor is alpha software and has not undergone an external security audit.
- Use HTTPS for any deployment beyond an isolated test network.
- Treat enrollment tokens, session secrets, database credentials, and agent identities as secrets.
- Rotate the enrollment token after broad deployment or accidental disclosure.
- Do not publish `.env`, database backups, logs containing credentials, or agent identity files.
- Restrict access to the Docker socket and host filesystem.
- The Docker host agent uses read-only host mounts but remains a privileged monitoring component.
- EITS Monitor intentionally does not implement arbitrary remote command execution.
