# Public ecosystem (getskillpack)

Overview of **public** repositories under [getskillpack](https://github.com/getskillpack). Engineering direction: **Go 1.22+** for the registry service, CLI, and package-manager core (no TypeScript for those components).

## Active surface

| Repository | Role |
|------------|------|
| [registry](https://github.com/getskillpack/registry) | HTTP API and reference server for publishing and downloading skill packages. Canonical contract: [registry-api.md](./registry-api.md). |
| [cli](https://github.com/getskillpack/cli) | `skillget` CLI and **Go** package-manager core (version resolution, lockfile, registry client, installs). |

## Retired

The historical **`getskillpack/skillget-manager`** repository (TypeScript prototype) has been **removed** from GitHub. It must not be linked from org indexes or docs as an active dependency. Replace any former references with **`cli`** for client and package-manager work.

## New public repositories

- Root **`README.md`** must be **English-only** — [ORG_README_POLICY.md](./ORG_README_POLICY.md).
- Copy [`templates/repo-starter/`](../templates/repo-starter/) for a baseline that includes README policy checks in CI — [REPO_TEMPLATE.md](./REPO_TEMPLATE.md).
