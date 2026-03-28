# Security

## Supported versions

We patch security issues in the **default branch** (`main`) of this repository. Deployments should track a tagged release or a known commit; see `registry -version` and `GET /version` on a running instance.

## Reporting a vulnerability

Please **do not** file a public issue for an undisclosed security vulnerability.

- Prefer **[GitHub private vulnerability reporting](https://github.com/getskillpack/registry/security/advisories/new)** for this repository (if enabled for the org).
- Alternatively, contact the **getskillpack** maintainers through the channel your organization uses for security reports.

Include: affected component (e.g. registry HTTP API, file store), steps to reproduce, and impact assessment if known.

## Operational hardening

Operators should follow [docs/PUBLIC_REGISTRY_RUNBOOK.md](docs/PUBLIC_REGISTRY_RUNBOOK.md): secrets (`REGISTRY_WRITE_TOKEN`, optional `REGISTRY_READ_TOKEN`), TLS at the edge, rate limits, and metrics as appropriate.
