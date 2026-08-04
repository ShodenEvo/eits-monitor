# Architecture

## Components

### Web interface

React and TypeScript are built into static assets and served by Nginx. Nginx proxies `/api` requests to FastAPI.

### API

FastAPI handles authentication, enrollment, metrics ingestion, device configuration, thresholds, and network-check management.

### Database

PostgreSQL stores users, sessions, devices, credentials, metrics, disk samples, network checks, and check results.

### Docker host agent

The Compose agent monitors the Linux Docker host using host PID visibility and read-only host filesystem mounts.

### Native agent

The native Go agent runs on Windows or Linux, enrolls with the central server, retrieves assigned checks, and pushes metrics. It does not expose an inbound port.

## Trust boundaries

- Browser sessions are separate from agent credentials.
- Enrollment tokens create identities but are not used for normal metric uploads.
- Each agent receives a unique secret.
- Agents should communicate over HTTPS outside isolated test networks.
- The Docker host agent has broad read visibility and must be treated as trusted infrastructure.
