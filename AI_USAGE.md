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
- Reviewed the repo structure and is the one deploying and operating it.

## Known limits, disclosed rather than hidden

A few implementation details are explicitly best-effort because they
depend on undocumented or thinly-documented third-party behavior the AI
could not verify against a real device during development:

- The Autodiscover v2 JSON response shape (`internal/handlers/v2json.go`)
  is modeled on POX's fields, not copied from a confirmed Microsoft schema.
- The SOAP `GetUserSettings` response (`internal/handlers/soap.go`) uses a
  plausible but unverified set of `UserSetting` names for plain IMAP/SMTP,
  since Microsoft's public docs focus on Exchange-native settings.

Both are called out in [README.md](README.md) as fallback paths, with POX
as the verified primary mechanism. If a real client disagrees with either,
treat it as a bug to fix against that client's actual request/response, not
as a sign the rest of the service is unreliable.

## Provenance in git history

Commits authored by the agent carry `Co-Authored-By: Claude Sonnet 5
<noreply@anthropic.com>` rather than being attributed solely to the human,
so the history itself reflects who wrote what.
