# Automatic updates

Agent packages include a separate updater. Supported modes are `disabled`, `notify`, and `automatic`; channels are `stable` and `prerelease`. The updater checks GitHub Releases, validates the release asset SHA-256 digest, preserves configuration and identity, and rolls back the executable if restart fails.

Windows installs `EITS Agent Update` as a daily SYSTEM scheduled task. Linux installs a systemd oneshot service and timer. The server updater runs only on the Docker host, creates an environment/commit/database backup, rebuilds the stack, tests `/api/health`, and rolls back the Git commit when the health check fails.

Server configuration is supplied through `/etc/eits-monitor/update.env`; agent configuration through `/etc/eits-agent/update.env` on Linux or scheduled-task arguments on Windows.
