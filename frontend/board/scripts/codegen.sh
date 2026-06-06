#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BACKEND_DIR="$SCRIPT_DIR/../../../backend"
# Use the redocly-bundled YAML (single self-contained file, no dangling $refs).
BUNDLED_YAML="$BACKEND_DIR/internal/openapi/openapi.yaml"
MERGED_SCHEMAS="$BACKEND_DIR/internal/openapi/components/schemas.yml"
OUT="$SCRIPT_DIR/../src/shared/api/schema.d.ts"

cleanup() {
  rm -f "$BUNDLED_YAML" "$MERGED_SCHEMAS"
}
trap cleanup EXIT

echo "Bundling OpenAPI spec (running make merge-schemas + openapi-bundle)..."
(cd "$BACKEND_DIR" && make openapi-bundle)

echo "Generating TypeScript types..."
bunx openapi-typescript "$BUNDLED_YAML" -o "$OUT"

echo "Done -> src/shared/api/schema.d.ts"
