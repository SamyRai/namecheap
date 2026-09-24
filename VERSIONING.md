# Semantic Versioning

This project follows [Semantic Versioning](https://semver.org/) (SemVer) for version management.

## Version Format

Versions follow the format: `MAJOR.MINOR.PATCH` (e.g., `1.2.3`)

- **MAJOR** version: Incremented for incompatible API changes
- **MINOR** version: Incremented for backwards-compatible functionality additions
- **PATCH** version: Incremented for backwards-compatible bug fixes

## Current Status

This project is currently in **pre-1.0.0** status, meaning:
- The API is not considered stable
- Breaking changes may occur between minor versions
- Use at your own risk and responsibility

## Version Management

There is no version literal stored in source. `pkg/version.Version` defaults
to `"dev"` and is only ever overridden at build time via `-ldflags "-X
go.glpx.pro/zonekit/pkg/version.Version=..."` (see the `Makefile`'s `VERSION`
variable and `.github/workflows/release.yml`), so the binary's version always
reflects what actually built it instead of a hand-maintained string that can
drift from reality.

### Automatic Version Bumping

Use GitHub Actions workflow to cut a release:

1. Go to Actions → Version Management
2. Click "Run workflow"
3. Select version type (patch, minor, or major)
4. The workflow calculates the new version from the latest existing tag and
   pushes a new `vX.Y.Z` tag — no source file is edited or committed.

### Manual Version Bumping

Create and push a tag directly:
```bash
git tag -a v0.2.0 -m "Release v0.2.0"
git push origin v0.2.0
```

### Version Information

Check the current version:
```bash
./zonekit --version
```

Or programmatically:
```go
import "go.glpx.pro/zonekit/pkg/version"

fmt.Println(version.Version)
fmt.Println(version.String())
fmt.Println(version.FullString())
```

## Release Process

1. **Create Tag**: Tag the release (automated via the Version Management
   workflow, or manually)
2. **Update CHANGELOG**: Document changes in CHANGELOG.md
3. **GitHub Release**: Pushing the tag triggers the release workflow, which:
   - Builds binaries for all platforms with `-ldflags -X
     go.glpx.pro/zonekit/pkg/version.Version=<tag>` (plus commit and build
     date)
   - Creates a GitHub release
   - Uploads artifacts
   - Generates release notes

## Pre-Release Versions

Pre-release versions can be indicated with suffixes:
- `0.1.0-alpha.1` - Alpha release
- `0.1.0-beta.1` - Beta release
- `0.1.0-rc.1` - Release candidate

## Version History

- `v0.1.0` - Initial release (2025-11-22)

