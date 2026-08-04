# Troubleshooting

## Agent receives 403 during enrollment

Verify that the token passed to the installer exactly matches `AGENT_ENROLLMENT_TOKEN` inside the API container:

```bash
docker compose exec api printenv AGENT_ENROLLMENT_TOKEN
```

Recreate the API after changing `.env`:

```bash
docker compose up -d --force-recreate api
```

## Agent metrics return 422

Use an up-to-date agent and inspect the complete agent log. Current agents include the API response body for failed requests.

## A network check is not visible immediately

Refresh to the current release. Device details use optimistic updates and periodic API refresh.

## TCP check against the Docker host fails

Inside a container, `127.0.0.1` points to that container. Use the host LAN address or configure a host gateway alias.

## UDP check appears up without an application response

UDP has no handshake. Enable **Require response** and configure a payload understood by the target service. For binary protocols, wait for a protocol-specific probe implementation.

## Deleting a check returns a database constraint error

Use the current API version. It deletes dependent check results transactionally and applies `ON DELETE CASCADE` to the foreign key.
