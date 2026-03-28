#!/usr/bin/env bash
# Собирает tar.gz примера из examples/hello-skill (только SKILL.md в корне архива).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SRC="$ROOT/examples/hello-skill"
OUT_DIR="$ROOT/examples/dist"
mkdir -p "$OUT_DIR"
OUT="$OUT_DIR/hello-skill-0.1.0.tar.gz"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
cp "$SRC/SKILL.md" "$TMP/SKILL.md"
# Детерминированный tarball для воспроизводимых checksum (GNU tar)
export SOURCE_DATE_EPOCH=1700000000
tar --sort=name --mtime="@${SOURCE_DATE_EPOCH}" --owner=0 --group=0 --numeric-owner \
  -czf "$OUT" -C "$TMP" SKILL.md
echo "Wrote $OUT"
sha256sum "$OUT"
