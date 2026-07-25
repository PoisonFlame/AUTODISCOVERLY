# AI Usage

This repo's initial implementation was written by an AI coding agent
([Claude Code](https://claude.com/claude-code), Anthropic, model Claude
Sonnet 5) operating under direct human direction, rather than written by
hand and later assisted by AI. This file discloses the scope and limits of
that involvement.

## What the AI did

- Proposed the architecture (Go, stdlib-only HTTP, YAML config with
  domain-defaults-and-overrides) after the human specified the goal,
  constraints (Docker image size, multi-domain, TLS delegated to a reverse
  proxy), and language choice.
- Wrote all Go source, tests, the Dockerfile, docker-compose example, and
  documentation.
- Ran the build and full test suite, built the Docker image, and exercised
  every endpoint with curl against a running container to verify behavior
  end-to-end, rather than only relying on unit tests.
- Ran a dependency/stdlib vulnerability scan (`govulncheck`) and a
  secret-pattern scan across the working tree and full git history, and
  fixed what it found (stale Go base image with known stdlib CVEs; missing
  HTTP server timeouts).
- When asked whether the SOAP and Autodiscover v2 JSON implementations
  could be checked against Microsoft's docs rather than left as best-effort
  guesses, fetched Microsoft Learn/MSDN/open-spec pages directly and found
  the first draft of both was wrong in real, not cosmetic, ways — see
  below.

## What the human did

- Set the goal, constraints, and product decisions: MXroute as the mail
  host, Go as the language, multi-domain support from day one, TLS
  terminated by an existing reverse proxy rather than in-container.
- Reviewed and approved the implementation plan before code was written.
- Made explicit calls on ambiguous design points when asked, e.g. how
  `display_name` should be sourced (auto-derived from the email address
  rather than a server-side lookup or fully static label).
- Requested the security review this file's sibling changes came from, and
  requested this disclosure file.
- Asked whether the SOAP/v2 JSON best-effort implementations could be
  verified against Microsoft's docs, and offered to check them personally
  if needed — the AI did the lookup itself instead.
- Reviewed the repo structure and is the one deploying and operating it.

## A concrete example of AI-written code being wrong, and how it got caught

The first draft of two endpoints was built from general pattern-matching
against related Microsoft conventions rather than a confirmed source, and
both turned out to be wrong in substantive ways once checked:

- **Autodiscover v2 JSON** (`internal/handlers/v2json.go`): the original
  version invented `Protocol=IMAP`/`SMTP` query values returning
  `{"Server", "Port", "SSL", "Username"}`. Real Autodiscover v2 has no
  IMAP/SMTP protocol value at all — it's a pointer service that resolves a
  requested protocol (`ActiveSync`, `Ews`, `Rest`, `AutodiscoverV1`, etc.)
  to a single `{"Protocol", "Url"}` service-endpoint URL. No real Outlook
  client would ever have queried the endpoint the original version
  implemented. Confirmed via [Icewolf's Autodiscover V2 JSON writeup](https://blog.icewolf.ch/archive/2020/12/09/autodiscover-v2-json-requests/)
  and corroborating community documentation of observed client behavior.
  Rewritten to only answer `Protocol=AutodiscoverV1`, pointing back at this
  server's own POX endpoint.
- **SOAP GetUserSettings** (`internal/handlers/soap.go`): the original
  version used invented field names (`InternalImapServer`,
  `InternalImapPort`, `InternalImapSSL`, ...) as flat `Name`/`Value` pairs.
  The real `UserSettingName` enum has no such fields; the actual names are
  `InternalImap4Connections` / `ExternalImap4Connections` /
  `InternalSmtpConnections` / `ExternalSmtpConnections`, each an
  `xsi:type="ProtocolConnectionCollectionSetting"` wrapping a
  `ProtocolConnections` list of `<Hostname>`/`<Port>`/`<EncryptionMethod>`
  entries. Confirmed against Microsoft's
  [`UserSettingName` enum reference](https://learn.microsoft.com/en-us/dotnet/api/microsoft.exchange.webservices.autodiscover.usersettingname),
  the [`GetUserSettings` SOAP operation reference](https://learn.microsoft.com/en-us/exchange/client-developer/web-service-reference/getusersettings-operation-soap)
  (which supplied a real example response used to check envelope/namespace
  structure), the [`ProtocolConnection` element reference](https://learn.microsoft.com/en-us/exchange/client-developer/web-service-reference/protocolconnection-soap),
  and the [`[MS-OXWSADISC]` full XML schema](https://learn.microsoft.com/en-us/openspecs/exchange_server_protocols/ms-oxwsadisc/9095809b-f1b1-41a0-bfbe-9d947a9229c7)
  (the authoritative wire-format XSD, used to confirm the exact
  `ProtocolConnectionCollectionSetting`/`ProtocolConnections`/
  `ProtocolConnection` complex-type nesting). Rewritten to match.

Both are unit-tested against the corrected shapes and were curl-verified
against a running container after the fix. POX remains the primary,
best-established mechanism either way — these two are secondary paths for
clients that try them first, and are now schema-accurate rather than merely
plausible-looking.

## Provenance in git history

Commits authored by the agent carry `Co-Authored-By: Claude Sonnet 5
<noreply@anthropic.com>` rather than being attributed solely to the human,
so the history itself reflects who wrote what.
