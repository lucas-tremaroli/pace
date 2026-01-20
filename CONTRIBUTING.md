# Contributing to Pace

## Requirements

- Go 1.21+

## Building

```bash
make build    # Build the binary to bin/pace
make install  # Build and install to $HOME/go/bin
make clean    # Remove build artifacts
```

## Reporting Issues

If you encounter any bugs or have feature requests, please open an [issue](https://github.com/lucas-tremaroli/pace/issues).

## Contributing Code

1. Fork the repository.
2. Create a new branch for your feature or bugfix.
3. Make your changes and commit them with clear messages.
4. Test your changes with `make test`.
5. Push your changes to your fork.
6. Open a pull request against the `main` branch of this repository.

## Releasing

Releases are automated via [GoReleaser](https://goreleaser.com/) and triggered by Git tags.

### Creating a Release

1. Ensure all changes are merged to `main`
2. Create and push a version tag:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```
3. The GitHub Actions workflow will automatically:
   - Build binaries for Linux and macOS (amd64/arm64)
   - Create a GitHub release with artifacts
   - Update the Homebrew formula

### Version Format

Follow [Semantic Versioning](https://semver.org/):
- `vMAJOR.MINOR.PATCH` (e.g., `v1.2.3`)
- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

### Testing a Release Locally

```bash
goreleaser release --snapshot --clean
```
