# New repository starter (getskillpack org)

Use this when creating a new public repository under [getskillpack](https://github.com/getskillpack).

1. Copy everything under [`templates/repo-starter/`](../templates/repo-starter/) to the root of the new repository (so `.github/` and `README.md` land at the top level).
2. Replace `{project-name}` and the description placeholder in `README.md` with real content. Keep the root README **English-only** — see [ORG_README_POLICY.md](./ORG_README_POLICY.md).
3. Extend `.github/workflows/ci.yml` with language-specific jobs (Go, Node, etc.) as the project grows. The default job enforces the org README policy and is safe for any stack.
