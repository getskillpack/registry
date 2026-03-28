#!/usr/bin/env bash
# E2E: local registry (this repo) + skillget CLI.
# Requires: go, curl, python3; skillget on PATH, or SKILLGET_SRC pointing at cli repo root.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TMP="$(mktemp -d)"
trap '[[ -n "${REG_PID:-}" ]] && kill "${REG_PID}" 2>/dev/null || true; rm -rf "${TMP}"' EXIT

pick_port() {
	python3 -c "import socket; s=socket.socket(); s.bind(('127.0.0.1',0)); print(s.getsockname()[1]); s.close()"
}

PORT="$(pick_port)"
TOKEN="e2e-smoke-token"
SKILL_NAME="e2e-smoke-skill"
SKILL_VER="1.0.0"
BASE_URL="http://127.0.0.1:${PORT}"
API_V1="${BASE_URL}/api/v1"

export REGISTRY_DATA_DIR="${TMP}/data"
export REGISTRY_LISTEN_ADDR="127.0.0.1:${PORT}"
export REGISTRY_WRITE_TOKEN="${TOKEN}"
export REGISTRY_PUBLIC_URL="${BASE_URL}"

(cd "${ROOT}" && go build -o "${TMP}/registry" ./cmd/registry)
"${TMP}/registry" &
REG_PID=$!

for _ in $(seq 1 50); do
	if curl -sf "${BASE_URL}/healthz" >/dev/null; then
		break
	fi
	sleep 0.1
done
curl -sf "${BASE_URL}/healthz" >/dev/null

mkdir -p "${TMP}/pack"
echo "e2e smoke" >"${TMP}/pack/README.md"
tar -czf "${TMP}/archive.tar.gz" -C "${TMP}/pack" .

MANIFEST="$(printf '{"name":"%s","version":"%s","description":"e2e smoke skill","author":"registry-ci"}' "${SKILL_NAME}" "${SKILL_VER}")"

code="$(curl -sS -X POST "${API_V1}/skills" \
	-H "Authorization: Bearer ${TOKEN}" \
	-F "manifest=${MANIFEST}" \
	-F "archive=@${TMP}/archive.tar.gz;filename=skill.tar.gz;type=application/gzip" \
	-o /dev/null -w "%{http_code}")"
[[ "${code}" == "201" ]]

SKILLGET_BIN="${SKILLGET_BIN:-}"
if [[ -z "${SKILLGET_BIN}" ]]; then
	if command -v skillget >/dev/null 2>&1; then
		SKILLGET_BIN="$(command -v skillget)"
	fi
fi
if [[ -z "${SKILLGET_BIN}" && -n "${SKILLGET_SRC:-}" ]]; then
	(cd "${SKILLGET_SRC}" && go build -o "${TMP}/skillget" ./cmd/skillget)
	SKILLGET_BIN="${TMP}/skillget"
fi
if [[ -z "${SKILLGET_BIN}" ]]; then
	echo "e2e-smoke: set SKILLGET_SRC to getskillpack/cli repo root, or install skillget on PATH, or set SKILLGET_BIN" >&2
	exit 1
fi

WORKDIR="${TMP}/consumer"
mkdir -p "${WORKDIR}"

export SKILLGET_REGISTRY_URL="${API_V1}"
(
	cd "${WORKDIR}"
	"${SKILLGET_BIN}" search "${SKILL_NAME}" | tee "${TMP}/search.out"
	grep -q "${SKILL_NAME}" "${TMP}/search.out"
)

(
	cd "${WORKDIR}"
	"${SKILLGET_BIN}" install "${SKILL_NAME}@${SKILL_VER}" | tee "${TMP}/install.out"
	grep -q "Wrote " "${TMP}/install.out"
	grep -q "skills.lock" "${TMP}/install.out"
)

ARCH_EXPECT="${WORKDIR}/.skillget/skills/${SKILL_NAME}/${SKILL_VER}/${SKILL_NAME}-${SKILL_VER}.tar.gz"
[[ -f "${ARCH_EXPECT}" ]]

echo "e2e-smoke: OK (registry + skillget install ${SKILL_NAME}@${SKILL_VER})"
