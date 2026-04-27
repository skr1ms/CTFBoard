#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BACKEND_DIR="$SCRIPT_DIR/../../../backend"
# Use the redocly-bundled YAML (single self-contained file, no dangling $refs).
# Regenerate it if missing or the source spec is newer.
BUNDLED_YAML="$BACKEND_DIR/internal/openapi/openapi.yaml"
SPEC="$BACKEND_DIR/internal/openapi/openapi.yml"
OUT="$SCRIPT_DIR/../src/shared/api/schema.d.ts"

if [ ! -f "$BUNDLED_YAML" ] || [ "$SPEC" -nt "$BUNDLED_YAML" ]; then
  echo "Bundling OpenAPI spec (running make merge-schemas + openapi-bundle)..."
  (cd "$BACKEND_DIR" && make openapi-bundle)
fi

echo "Generating TypeScript types..."
bunx openapi-typescript "$BUNDLED_YAML" -o "$OUT"

echo "Done -> src/shared/api/schema.d.ts"
