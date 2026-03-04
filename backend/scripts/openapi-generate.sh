#!/usr/bin/env bash
# Merge schemas, bundle OpenAPI spec, generate code, then cleanup

set -e

# Change to script directory (backend/)
cd "$(dirname "$0")/.." || exit 1

SCHEMAS_FILE="internal/openapi/components/schemas.yml"
OPENAPI_SRC="internal/openapi/openapi.yml"
OPENAPI_BUNDLE=$(mktemp -t openapi-bundle-XXXXXX.yaml)

# Cleanup function to remove temporary files
cleanup() {
    echo "Cleaning up temporary files..."
    rm -f "$SCHEMAS_FILE"
    rm -f "$OPENAPI_BUNDLE"
}

# Register cleanup function to run on exit (success or failure)
trap cleanup EXIT

# Merge schema files and fix route references
echo "Merging schema files and fixing references..."
python3 scripts/merge-schemas.py

# Bundle OpenAPI spec to temporary file
echo "Bundling OpenAPI spec..."
mkdir -p internal/openapi
command -v npx >/dev/null 2>&1 || (echo "Error: npx (Node.js) required. Install Node.js or run: npm install -g @redocly/cli"; exit 1)
npx --yes @redocly/cli bundle "$OPENAPI_SRC" -o "$OPENAPI_BUNDLE" --ext yaml

# Generate code from bundled file
echo "Generating OpenAPI code..."
oapi-codegen -config codegen/oapi-codegen-types.yml "$OPENAPI_BUNDLE"
oapi-codegen -config codegen/oapi-codegen-server.yml "$OPENAPI_BUNDLE"
oapi-codegen -config codegen/oapi-codegen-spec.yml "$OPENAPI_BUNDLE"
oapi-codegen -config codegen/oapi-codegen-client.yml "$OPENAPI_BUNDLE"

echo "✓ OpenAPI code generation completed"
