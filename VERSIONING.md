# Semantic Versioning

This project follows [Semantic Versioning](https://semver.org/) (SemVer) for version management.

## Version Format

Versions follow the format: `MAJOR.MINOR.PATCH` (e.g., `1.2.3`)

- **MAJOR** version: Incremented for incompatible API changes
- **MINOR** version: Incremented for backwards-compatible functionality additions
- **PATCH** version: Incremented for backwards-compatible bug fixes

## Current Status

**Current Version: 0.1.0**

This project is currently in **pre-1.0.0** status, meaning:
- The API is not considered stable
- Breaking changes may occur between minor versions
- Use at your own risk and responsibility

## Version Management

### Automatic Version Bumping

Use GitHub Actions workflow to bump versions:

1. Go to Actions → Version Management
2. Click "Run workflow"
3. Select version type (patch, minor, or major)
4. The workflow will:
   - Calculate the new version
   - Update `pkg/version/version.go`
   - Create a git commit
   - Create and push a git tag

### Manual Version Bumping

1. Update `Version` in `pkg/version/version.go`
2. Commit the change
3. Create and push a tag:
   ```bash
   git tag -a v0.2.0 -m "Release v0.2.0"
   git push origin v0.2.0
   ```

### Version Information

Check the current version:
```bash
./namecheap-dns --version
```

Or programmatically:
```go
import "namecheap-dns-manager/pkg/version"

fmt.Println(version.Version)
fmt.Println(version.String())
fmt.Println(version.FullString())
```

## Release Process

1. **Update Version**: Bump version using workflow or manually
2. **Update CHANGELOG**: Document changes in CHANGELOG.md
3. **Create Tag**: Tag the release (automated or manual)
4. **GitHub Release**: The release workflow will automatically:
   - Build binaries for all platforms
   - Create a GitHub release
   - Upload artifacts
   - Generate release notes

## Pre-Release Versions

Pre-release versions can be indicated with suffixes:
- `0.1.0-alpha.1` - Alpha release
- `0.1.0-beta.1` - Beta release
- `0.1.0-rc.1` - Release candidate

## Version History

- `v0.1.0` - Initial release (2025-11-22)

