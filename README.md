# AutoDiscoverly

A small, self-hosted Autodiscover/Autoconfig service for IMAP/SMTP mail providers
(e.g. [MXroute](https://mxroute.com)). Point it at your domains and mail clients
— Outlook (desktop and mobile), Thunderbird, K-9/Fair Email, etc. — can be set up
with just an email address and password, no manual server/port entry.

Single Go binary, single YAML config, single Docker image (~10-15MB).

This project's initial implementation was AI-written under human direction —
see [AI_USAGE.md](AI_USAGE.md) for what that means in practice.

## Endpoints

| Client | Mechanism | Endpoint |
|---|---|---|
| Outlook (desktop & mobile) | POX Autodiscover | `POST /autodiscover/autodiscover.xml` |
| Outlook (older/edge cases) | SOAP GetUserSettings | `POST /autodiscover/autodiscover.xml` (same URL, detected by request body) |
| Outlook (newer) | Autodiscover v2 (JSON) | `GET /autodiscover/autodiscover.json?Email=...&Protocol=AutodiscoverV1` |
| Thunderbird / others | Mozilla Autoconfig | `GET /mail/config-v1.1.xml?emailaddress=...` and `GET /.well-known/autoconfig/mail/config-v1.1.xml` |
| Orchestrator | Liveness | `GET /health` |

Every handler resolves the target domain from the **email address in the
request** (payload or query string), not from the `Host` header. That means
a single catch-all vhost in your reverse proxy, pointed at this one
container, is enough for every domain you configure below — you don't need
per-domain routing rules, just DNS + certs (see below).

## Configuration

Copy [`config.example.yaml`](config.example.yaml) and edit it:

```yaml
server:
  listen: ":8080"

defaults:
  imap:
    hostname: mail.mxrouting.net
    port: 993
    encryption: SSL        # SSL | STARTTLS | none
    username_format: email # email | local-part
  smtp:
    hostname: mail.mxrouting.net
    port: 465
    encryption: SSL
    username_format: email

domains:
  example.com: {}
  another-example.org:
    display_name: "Another Example Support"
    imap:
      port: 143
      encryption: STARTTLS   # per-field override, hostname still inherited from defaults
```

Any `imap`/`smtp` field left unset on a domain falls back to `defaults`. A
domain with no overrides at all can just be `example.com: {}`.

`display_name` is optional. Leave it unset and each mailbox gets its own
name auto-derived from the email address at request time (e.g.
`jane.doe@example.com` → "Jane Doe") — there's no server-side source of a
mailbox's real name to query, so this is a best-effort heuristic, not a
lookup. Set `display_name` explicitly only if you want every mailbox on a
domain to show the same branded label instead (e.g. a shared support
inbox).

The config path defaults to `/etc/autodiscoverly/config.yaml`, override with
the `CONFIG_PATH` env var.

## Running it

```sh
docker build -t autodiscoverly .
docker run -d \
  -p 127.0.0.1:8080:8080 \
  -v "$(pwd)/config.yaml:/etc/autodiscoverly/config.yaml:ro" \
  autodiscoverly
```

Or with `docker-compose.yml` (put your real config at `./config.yaml` first):

```sh
docker compose up -d
```

Put your existing reverse proxy (Traefik/Caddy/nginx) in front of it for TLS.

## DNS records you need, per domain

- `autodiscover.<domain>` → your reverse proxy (Outlook uses this).
- `autoconfig.<domain>` → your reverse proxy (Thunderbird uses this) — **or**
  skip this record entirely and rely on the `/.well-known/autoconfig/...`
  path served on `<domain>` itself, which Thunderbird also checks.
- Certificates covering whichever of the above hostnames you use.

## A caveat worth knowing about

Even with working self-hosted Autodiscover, Outlook Mobile may still probe
Microsoft's own cloud Autodiscover service as part of how it looks up
non-Microsoft domains. That's client-side behavior no self-hosted server can
fully suppress — it's not a bug in this project, just a heads up if you go
looking at your web server logs.

## Protocol fidelity

All three Outlook mechanisms are implemented against Microsoft's published
schemas, not guessed — see [AI_USAGE.md](AI_USAGE.md) for the exact sources
and what was cross-checked:

- **POX** (`/autodiscover/autodiscover.xml` with an `<Autodiscover>` body)
  is the well-established mechanism Outlook uses for plain IMAP/SMTP
  accounts and is the one to rely on above the other two.
- **Autodiscover v2 JSON** does *not* have an IMAP/SMTP protocol value in
  Microsoft's real implementation — it's a pointer service that resolves a
  requested "next-level protocol" (ActiveSync/EWS/REST/etc.) to a single
  service URL. This server only answers `Protocol=AutodiscoverV1`, pointing
  the client back at its own POX endpoint; every other protocol value
  correctly 404s, since we have no Exchange-native services to point to.
- **SOAP GetUserSettings** returns `InternalImap4Connections` /
  `ExternalImap4Connections` / `*SmtpConnections` as
  `ProtocolConnectionCollectionSetting`-typed `UserSetting` entries (the
  real `xsi:type` and element nesting Exchange servers use), not
  the made-up flat field names an earlier draft used.

If a real device still disagrees with any of the above, the fix is almost
always in `internal/handlers/*.go` — file an issue/PR with the client's
actual request/response.

## Development

```sh
go test ./...
gofmt -l .
go vet ./...
```

No local Go toolchain? Run the same commands in a container:

```sh
docker run --rm -v "$(pwd)":/src -w /src golang:1.25-alpine \
  sh -c "go test ./... && gofmt -l . && go vet ./..."
```

## Project layout

```
cmd/autodiscoverly/    main.go — config load, server wiring, graceful shutdown
internal/config/       YAML parsing + structural validation
internal/mailconfig/   domain -> effective IMAP/SMTP settings (defaults + overrides merge)
internal/handlers/     one file per protocol (pox.go, soap.go, v2json.go, autoconfig.go, health.go)
internal/server/       route registration
```
