# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.2] - 2026-02-02

### Changed
- Maintenance release

## [1.0.0] - 2026-02-02

### Added
- Initial implementation of environment variables provider
- Support for direct environment variable access
- Hierarchical path-to-variable-name mapping with configurable separator and case transformation
- Prefix-based filtering with prepend and filter-only modes
- Automatic type conversion (string to number/boolean/JSON)
- Required variable validation during initialization
- Thread-safe caching for improved performance
- gRPC service implementation (Init, Fetch, Info, Health, Shutdown)
- Cross-platform support (macOS, Linux, Windows)
- Comprehensive test suite with >80% coverage
- GitHub Actions CI/CD workflow for automated testing, linting, and builds
- Quickstart validation script (`scripts/validate-quickstart.sh`)
- Comprehensive README with usage examples for all 5 user stories
- Performance benchmarks documentation
- Build instructions and troubleshooting guide

### Fixed
- Added explicit stdout flush for PORT announcement to resolve CLI hang issue

[Unreleased]: https://github.com/autonomous-bits/nomos-provider-environment-variables/compare/v0.1.2...HEAD
[0.1.2]: https://github.com/autonomous-bits/nomos-provider-environment-variables/compare/v1.0.0...v0.1.2
[1.0.0]: https://github.com/autonomous-bits/nomos-provider-environment-variables/releases/tag/v1.0.0
