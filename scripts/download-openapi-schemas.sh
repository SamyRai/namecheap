#!/bin/bash
# Script to download official OpenAPI schemas from DNS providers

set -e

PROVIDER_DIR="pkg/dns/provider"
TEMP_DIR="/tmp/zonekit-openapi"

mkdir -p "$TEMP_DIR"

echo "Downloading official OpenAPI schemas..."

# Cloudflare
echo "📥 Downloading Cloudflare OpenAPI spec..."
CLOUDFLARE_URL="https://raw.githubusercontent.com/cloudflare/api-schemas/main/openapi.yaml"
CLOUDFLARE_OUTPUT="$PROVIDER_DIR/cloudflare/openapi.yaml"

if curl -s -L -f "$CLOUDFLARE_URL" -o "$CLOUDFLARE_OUTPUT"; then
    CLOUDFLARE_SIZE=$(ls -lh "$CLOUDFLARE_OUTPUT" | awk '{print $5}')
    echo "✅ Cloudflare: Downloaded ($CLOUDFLARE_SIZE)"
    echo "   Source: $CLOUDFLARE_URL"
    echo "   Saved to: $CLOUDFLARE_OUTPUT"
else
    echo "❌ Cloudflare: Failed to download"
fi

echo ""
echo "📝 Note: The Cloudflare spec is large (~16MB) and includes all services."
echo "   For DNS-only usage, consider extracting just DNS-related paths."
echo ""
echo "✅ Download complete!"
echo ""
echo "To extract DNS-only paths (optional):"
echo "  yq eval '.paths | with_entries(select(.key | contains(\"dns\")))' \\"
echo "    $CLOUDFLARE_OUTPUT > $PROVIDER_DIR/cloudflare/openapi-dns-only.yaml"

