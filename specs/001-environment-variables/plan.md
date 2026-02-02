# Implementation Plan: Environment Variables Provider

**Branch**: `001-environment-variables` | **Date**: 2026-02-01 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-environment-variables/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Create a Nomos provider that enables the Nomos compiler to pull configuration values from environment variables. The provider implements the `nomos.provider.v1.ProviderService` gRPC contract, runs as a subprocess, and fetches values from the system environment with support for path-based variable name mapping, prefix filtering, automatic type conversion, and validation.

## Technical Context

**Language/Version**: Go 1.25.6  
**Primary Dependencies**: 
  - `google.golang.org/grpc` - gRPC server implementation
  - `google.golang.org/protobuf` - Protocol buffer serialization for structured data
  - `github.com/autonomous-bits/nomos/libs/provider-proto` - gRPC contract definitions
  
**Storage**: Process environment (read-only, cached in memory after first fetch)  
**Testing**: Go standard `testing` package with table-driven tests, integration tests for gRPC contract  
**Target Platform**: Cross-platform binaries (darwin/arm64, darwin/amd64, linux/amd64, linux/arm64, windows/amd64)  
**Project Type**: Single standalone binary (provider subprocess)  
**Performance Goals**: <10ms fetch response time (cached values), <2s initialization  
**Constraints**: 
  - Zero external runtime dependencies (statically linked binary)
  - Platform-native case sensitivity (Unix: case-sensitive, Windows: case-insensitive)
  - Maximum environment variable value size: 1MB
  - Process isolation (loopback-only gRPC, OS-assigned port)
  
**Scale/Scope**: 
  - Designed for hundreds of environment variables
  - Concurrent fetch support (thread-safe caching)
  - Single instance per provider alias (CLI manages multiple instances)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. gRPC Contract Fidelity ✅ COMPLIANT

- ✅ All RPC methods (Init, Fetch, Info, Health, Shutdown) will be implemented
- ✅ Init validates configuration (required_variables, case_transform, separator, prefix)
- ✅ Fetch returns protobuf Struct with environment variable values
- ✅ Info returns accurate version, type (`"environment-variables"`), and alias metadata
- ✅ Health reports DEGRADED before init, OK after successful init
- ✅ Shutdown performs cleanup (clears cache, releases resources)
- ✅ Port announcement follows `PORT=<port>` format to stdout

**No violations**

### II. Binary Portability ✅ COMPLIANT

- ✅ Cross-platform builds planned: darwin/arm64, darwin/amd64, linux/amd64, linux/arm64, windows/amd64
- ✅ Statically-linked binary (Go's default for providers)
- ✅ Binary naming: `nomos-provider-environment-variables-<version>-<os>-<arch>`
- ✅ SHA256 checksums will be published with releases
- ✅ Zero external runtime dependencies (stdlib + gRPC only)
- ✅ GitHub Releases for distribution

**No violations**

### III. Integration Testing (NON-NEGOTIABLE) ✅ COMPLIANT

- ✅ TDD approach: tests written first, validated to fail, then implementation
- ✅ Integration tests will cover all RPC methods (Init, Fetch, Info, Health, Shutdown)
- ✅ Error paths tested: missing env vars, invalid config, malformed paths
- ✅ State transitions tested: uninitialized → initialized → shutdown
- ✅ Coverage target: >80% for service and parser code
- ✅ Real environment variable fixtures (no mocks for OS environment)
- ✅ Red-Green-Refactor cycle enforced

**No violations**

### IV. Process Isolation & Clean Lifecycle ✅ COMPLIANT

- ✅ Startup prints `PORT=<port>` as first line to stdout
- ✅ Operational logs go to stderr
- ✅ Graceful shutdown for SIGTERM and SIGINT
- ✅ Shutdown cleans up resources (cache cleared, goroutines stopped)
- ✅ Binds to `127.0.0.1` (loopback only)
- ✅ OS-assigned port (listen on `:0`)
- ✅ Clean exit with status code 0 on success

**No violations**

### V. Observability & Debugging ✅ COMPLIANT

- ✅ Structured logging to stderr (timestamp, level, message, context)
- ✅ Log levels: ERROR, WARN, INFO, DEBUG
- ✅ ERROR logs include context and stack traces where applicable
- ✅ gRPC errors include descriptive messages (not just codes)
- ✅ Environment variable names logged in errors (absolute references)
- ✅ Version and provider type logged on startup
- ✅ Health status transitions logged

**No violations**

### Overall Status: ✅ ALL PRINCIPLES SATISFIED

No complexity justification required. The environment variables provider is a straightforward implementation of the standard Nomos provider contract with no architectural deviations.

## Project Structure

### Documentation (this feature)

```text
specs/001-environment-variables/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
│   └── provider.proto   # gRPC service definition (reference)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
cmd/
└── provider/
    └── main.go          # Entry point: gRPC server, port announcement, signal handling

internal/
├── provider/
│   ├── provider.go      # ProviderService implementation
│   ├── init.go          # Init RPC handler
│   ├── fetch.go         # Fetch RPC handler (path resolution, caching)
│   ├── info.go          # Info RPC handler
│   ├── health.go        # Health RPC handler
│   └── shutdown.go      # Shutdown RPC handler
├── config/
│   ├── config.go        # Configuration struct and validation
│   └── parser.go        # Parse protobuf Struct to Config
├── resolver/
│   ├── resolver.go      # Path-to-variable-name resolution
│   ├── transform.go     # Case transformation and separator logic
│   └── prefix.go        # Prefix filtering and prepending
├── fetcher/
│   ├── fetcher.go       # Environment variable fetcher with caching
│   ├── converter.go     # Type conversion (string → number/bool/JSON)
│   └── validator.go     # Required variable validation
└── logger/
    └── logger.go        # Structured logging wrapper

tests/
├── integration/
│   ├── grpc_test.go     # Full gRPC contract integration tests
│   ├── lifecycle_test.go # Init → Fetch → Shutdown state tests
│   └── fixtures/        # Test environment variable setups
├── unit/
│   ├── resolver_test.go # Path resolution unit tests
│   ├── converter_test.go # Type conversion unit tests
│   └── config_test.go   # Configuration parsing unit tests
└── testdata/
    └── configs/         # Sample provider configurations

docs/
├── provider-development-standards.md # Already exists
└── quickstart.md                     # Phase 1 output

go.mod                   # Go module definition
go.sum                   # Dependency checksums
Makefile                 # Build, test, lint targets
.golangci.yml            # Linter configuration
README.md                # Provider overview and usage
CHANGELOG.md             # Version history (Keep a Changelog format)
```

**Structure Decision**: Single project structure (Option 1 from template). This is a standalone Go binary implementing a gRPC service, following standard Go conventions with `cmd/` for entry points and `internal/` for private implementation code. The provider is self-contained with no frontend/backend split or mobile targets.

## Complexity Tracking

> **No violations to justify** - All constitution principles satisfied without deviations.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
