# Release Guide

This document describes how to create releases for ARC using the automated release workflow.

## Overview

ARC uses [GoReleaser](https://goreleaser.com/) with GitHub Actions to automate the release process. The workflow builds cross-platform binaries, generates changelogs, and publishes GitHub releases.

## Supported Platforms

Releases include binaries for:
- **Linux**: amd64, arm64
- **macOS**: amd64 (Intel), arm64 (Apple Silicon)
- **Windows**: amd64

## Release Process

### 1. Prepare for Release

Before creating a release:

1. Ensure all changes are merged to the appropriate branch
2. Update version-related documentation if needed
3. Review recent commits for changelog

### 2. Trigger Release Workflow

#### Via GitHub UI

1. Go to **Actions** → **cd-release**
2. Click **Run workflow**
3. Fill in the parameters:
   - **version**: Semantic version (e.g., `v1.0.0`, `v1.1.0-beta.1`)
   - **draft**: Create as draft for review (recommended)
   - **prerelease**: Mark as prerelease for beta/RC versions
   - **create_tag**: Auto-create git tag if it doesn't exist
   - **environment**: Target environment (default: `prd`)
4. Click **Run workflow**

#### Version Format

Follow [Semantic Versioning](https://semver.org/):

```
vMAJOR.MINOR.PATCH[-PRERELEASE]
```

Examples:
- `v1.0.0` - Stable release
- `v1.1.0` - New features
- `v1.0.1` - Bug fixes
- `v1.2.0-beta.1` - Beta release
- `v2.0.0-rc.1` - Release candidate

### 3. Monitor Workflow

1. Watch the workflow execution in GitHub Actions
2. Check for any errors in the build process
3. Review the generated artifacts

### 4. Verify Release

After the workflow completes:

1. Go to **Releases** page
2. Review the release notes and changelog
3. Download and test binaries for your platform
4. Verify checksums match

### 5. Publish Release

If created as draft:

1. Edit the release
2. Review the auto-generated changelog
3. Add any additional release notes
4. Click **Publish release**

## Release Artifacts

Each release includes:

- **Binaries**: Pre-compiled for all supported platforms
- **Archives**: `.tar.gz` (Linux/macOS), `.zip` (Windows)
- **Checksums**: SHA256 checksums file
- **Source code**: Automatic GitHub archive

## Changelog Generation

The changelog is automatically generated from git commits:

- **Features**: Commits starting with `feat:`
- **Bug Fixes**: Commits starting with `fix:`
- **Security**: Commits starting with `security:`
- **Performance**: Commits starting with `perf:`

Excluded from changelog:
- `docs:`, `test:`, `chore:`, `ci:`, `refactor:`, `style:`
- Merge commits

## Manual Release (Advanced)

If needed, you can create releases manually:

```bash
# Install tools
aqua install

# Create and push tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# Run GoReleaser
export GITHUB_TOKEN="your-github-token"
goreleaser release --clean
```

## Troubleshooting

### Tag Already Exists

If the tag already exists:
- Set `create_tag: false` in the workflow
- Or delete the existing tag: `git tag -d v1.0.0 && git push origin :refs/tags/v1.0.0`

### Build Failures

Common issues:
- **Go version mismatch**: Check `go_version` input matches go.mod
- **Dependency errors**: Ensure `go.sum` is up to date
- **Test failures**: All tests must pass before release

### Release Not Published

If using draft mode:
- Draft releases are not visible to the public
- Edit and publish the draft manually

## Security

### GitHub Token

The workflow uses `secrets.GITHUB_TOKEN`:
- Automatically provided by GitHub Actions
- Has write access to create releases and tags
- Scoped to the repository only

### Signing (Optional)

To enable GPG signing of releases:

1. Add GPG key to secrets
2. Uncomment the `sign` section in `.goreleaser.yaml`
3. Update the workflow to pass the GPG fingerprint

## References

- [GoReleaser Documentation](https://goreleaser.com/)
- [Semantic Versioning](https://semver.org/)
- [GitHub Releases](https://docs.github.com/en/repositories/releasing-projects-on-github)
