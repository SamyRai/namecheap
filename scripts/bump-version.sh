#!/bin/bash
# Bump version script for semantic versioning

set -e

VERSION_TYPE="${1:-patch}"

if [[ ! "$VERSION_TYPE" =~ ^(major|minor|patch)$ ]]; then
    echo "Error: Version type must be 'major', 'minor', or 'patch'"
    exit 1
fi

# Get current version
CURRENT_VERSION=$(grep 'Version = ' pkg/version/version.go | sed 's/.*Version = "\(.*\)"/\1/')

if [ -z "$CURRENT_VERSION" ]; then
    echo "Error: Could not find current version in pkg/version/version.go"
    exit 1
fi

# Parse version parts
IFS='.' read -r -a PARTS <<< "$CURRENT_VERSION"
MAJOR="${PARTS[0]}"
MINOR="${PARTS[1]}"
PATCH="${PARTS[2]}"

# Calculate new version
case "$VERSION_TYPE" in
    major)
        MAJOR=$((MAJOR + 1))
        MINOR=0
        PATCH=0
        ;;
    minor)
        MINOR=$((MINOR + 1))
        PATCH=0
        ;;
    patch)
        PATCH=$((PATCH + 1))
        ;;
esac

NEW_VERSION="$MAJOR.$MINOR.$PATCH"

echo "Current version: $CURRENT_VERSION"
echo "New version: $NEW_VERSION"
echo ""

# Update version in code
sed -i.bak "s/Version = \".*\"/Version = \"$NEW_VERSION\"/" pkg/version/version.go
rm -f pkg/version/version.go.bak

echo "✅ Version updated to $NEW_VERSION in pkg/version/version.go"
echo ""
echo "Next steps:"
echo "  1. Review the changes: git diff pkg/version/version.go"
echo "  2. Commit: git commit -m 'chore: bump version to $NEW_VERSION'"
echo "  3. Tag: git tag -a v$NEW_VERSION -m 'Release v$NEW_VERSION'"
echo "  4. Push: git push origin master && git push origin v$NEW_VERSION"

