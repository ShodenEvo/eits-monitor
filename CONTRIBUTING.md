# Contributing

Thank you for helping improve EITS Monitor.

## Development workflow

1. Fork the repository.
2. Create a branch from `main`.
3. Keep changes focused and include tests where practical.
4. Run the relevant build checks.
5. Open a pull request using the provided template.

## Local checks

### Native agent

```bash
cd native-agent
gofmt -w .
go vet ./...
go test ./...
```

### Docker host agent

```bash
cd agent
gofmt -w .
go vet ./...
go test ./...
```

### API

```bash
python -m py_compile server/app/main.py
```

### Web

```bash
cd web
npm install
npm run build
```

### Full stack

```bash
docker compose config
docker compose build
```

## Coding principles

- Keep the agent outbound-only.
- Avoid remote command execution features.
- Preserve compatibility with Windows and Linux.
- Prefer explicit database migrations and safe data retention.
- Document security implications and operational trade-offs.
