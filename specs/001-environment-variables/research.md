# Technical Research: Environment Variables Provider

**Feature**: Environment Variables Provider  
**Date**: 2026-02-01  
**Status**: Complete

## Overview

This document captures the technical research and decisions made for implementing the environment variables provider. All items from the Technical Context have been researched and resolved.

## Research Items

### 1. Path-to-Variable-Name Transformation Strategy

**Decision**: Configurable transformation with separator and case options

**Rationale**:
- Environment variables follow platform conventions (typically `UPPER_CASE` on Unix, varies on Windows)
- CSL imports use hierarchical paths like `env["database"]["host"]`
- Need flexible mapping between these two representations

**Implementation Approach**:
- Join path segments using configurable separator (default: `_`)
- Apply case transformation: `upper`, `lower`, or `preserve` (default: `upper`)
- Example: `["database"]["host"]` → join with `_` → `database_host` → uppercase → `DATABASE_HOST`

**Alternatives Considered**:
1. **Fixed uppercase-underscore convention** - Rejected: Not flexible for environments using lowercase conventions
2. **Regex-based mapping** - Rejected: Too complex, hard to debug
3. **Bidirectional mapping file** - Rejected: Adds configuration overhead, defeats simplicity goal

**Libraries/Tooling**: Go stdlib `strings` package (no external dependencies needed)

---

### 2. Prefix Filtering and Prepending

**Decision**: Dual-mode prefix support (`prepend` vs `filter_only`)

**Rationale**:
- Users want namespace isolation to avoid accidental system variable access
- Some users want automatic prefix prepending for ergonomics
- Others want explicit control over variable names

**Implementation Approach**:
- `prefix`: Optional string (e.g., `MYAPP_`)
- `prefix_mode`: `prepend` (default) or `filter_only`
- In `prepend` mode: transform path first, then prepend prefix
- In `filter_only` mode: only filter, user must include prefix in path

**Alternatives Considered**:
1. **Always prepend** - Rejected: Users clarified they want control (clarification Q1)
2. **Only filter, never prepend** - Rejected: Reduces ergonomics for common use cases
3. **Prepend before transformation** - Rejected: Less predictable composition

**Libraries/Tooling**: Go stdlib `strings.HasPrefix`, `strings.TrimPrefix`

---

### 3. Type Conversion Precedence

**Decision**: Number → Boolean → String

**Rationale**:
- Per clarification Q3, numerical values take priority
- Prevents ambiguity: `"1"` becomes number `1`, not boolean `true`
- Matches common developer expectations (numbers are more common than booleans in config)

**Implementation Approach**:
1. Attempt numeric parsing (`strconv.ParseInt`, `strconv.ParseFloat`)
2. If numeric parsing fails, check boolean patterns (`true`, `false`, `yes`, `no`)
3. If both fail, return as string

**Alternatives Considered**:
1. **Boolean first** - Rejected: User specified number priority (clarification Q3)
2. **Only unambiguous conversions** - Rejected: Lower ergonomics, requires manual conversion
3. **Configurable precedence** - Rejected: Adds complexity without clear use case

**Libraries/Tooling**: Go stdlib `strconv` package

---

### 4. JSON Parsing Strategy

**Decision**: Strict JSON parsing with errors on failure

**Rationale**:
- Per clarification Q4, return errors for malformed JSON to catch issues early
- Prevents silent failures where users expect structured data but get strings
- Values starting with `{` or `[` signal clear intent to use JSON

**Implementation Approach**:
- Detect JSON intent: value starts with `{` or `[`
- Use `encoding/json.Unmarshal` to parse into `interface{}`
- Convert to `structpb.Value` for protobuf compatibility
- Return `InvalidArgument` error with parsing details on failure

**Alternatives Considered**:
1. **Silent fallback to string** - Rejected: User specified errors preferred (clarification Q4)
2. **Log warning but return string** - Rejected: Errors are more actionable
3. **Third-party JSON library (jsoniter)** - Rejected: Stdlib sufficient, avoids dependencies

**Libraries/Tooling**: Go stdlib `encoding/json`

---

### 5. Environment Variable Caching

**Decision**: Read-once, cache-forever strategy

**Rationale**:
- Environment variables don't change during process lifetime
- Multiple imports of same variable should return consistent values
- Performance: avoid repeated `os.Getenv` calls

**Implementation Approach**:
- `sync.Map` for thread-safe caching
- On first fetch: read from `os.Getenv`, apply transformations, cache result
- Subsequent fetches: return cached value
- Clear cache on `Shutdown` RPC

**Alternatives Considered**:
1. **No caching** - Rejected: Inefficient, violates <10ms performance goal
2. **TTL-based cache** - Rejected: Unnecessary complexity, env vars don't change
3. **LRU cache** - Rejected: All variables likely to be accessed, no eviction needed

**Libraries/Tooling**: Go stdlib `sync.Map`

---

### 6. Cross-Platform Case Sensitivity

**Decision**: Platform-native behavior

**Rationale**:
- Per clarification Q2, use platform-native case sensitivity
- Unix/Linux: case-sensitive (`PATH` ≠ `path`)
- Windows: case-insensitive (`PATH` == `path`)
- Matches developer expectations for each platform

**Implementation Approach**:
- Use Go's `os.Getenv` directly (respects platform behavior)
- Document platform differences in provider docs
- Recommend uppercase conventions for cross-platform CSL files

**Alternatives Considered**:
1. **Always case-sensitive** - Rejected: User specified platform-native (clarification Q2)
2. **Always case-insensitive** - Rejected: Breaks Unix semantics
3. **Configurable case sensitivity** - Rejected: Adds complexity, rare need

**Libraries/Tooling**: Go stdlib `os` package

---

### 7. Required Variables Validation

**Decision**: Validate original environment variable names at Init

**Rationale**:
- Per clarification Q5, use original var names (e.g., `["DATABASE_HOST"]`)
- Validation happens once during Init, fails fast
- Clear error messages list all missing variables

**Implementation Approach**:
- Config field: `required_variables []string`
- During Init: iterate list, check `os.LookupEnv` for each
- Collect all missing variables
- Return `InvalidArgument` with formatted error message

**Alternatives Considered**:
1. **Validate transformed paths** - Rejected: User specified original names (clarification Q5)
2. **Validate on first fetch** - Rejected: Fails late, less clear errors
3. **Support both formats** - Rejected: Adds complexity, ambiguous semantics

**Libraries/Tooling**: Go stdlib `os.LookupEnv`

---

### 8. Structured Logging

**Decision**: Custom structured logger wrapping Go stdlib

**Rationale**:
- Constitution requires structured logging to stderr
- Need timestamp, level, message, context fields
- Avoid heavy external dependencies (zerolog, zap)

**Implementation Approach**:
- Custom logger type with methods: `Error()`, `Warn()`, `Info()`, `Debug()`
- Output format: `timestamp level message field1=value1 field2=value2`
- Use `log.New(os.Stderr, ...)` from stdlib
- Log level controlled by environment variable or config

**Alternatives Considered**:
1. **zerolog** - Rejected: Adds dependency, overkill for provider needs
2. **zap** - Rejected: Performance benefits not needed, complex API
3. **Plain log.Printf** - Rejected: Doesn't meet structured logging requirement

**Libraries/Tooling**: Go stdlib `log` package

---

### 9. gRPC Server Configuration

**Decision**: Standard gRPC server with graceful shutdown

**Rationale**:
- Constitution requires loopback-only binding, OS-assigned port
- Need clean shutdown for SIGTERM/SIGINT
- Port announcement to stdout on startup

**Implementation Approach**:
- Listen on `127.0.0.1:0` (random port)
- Extract port from `net.Listener` → print `PORT={port}` to stdout
- Redirect logs to stderr after port announcement
- Implement signal handling (context cancellation) for graceful shutdown
- Call `grpcServer.GracefulStop()` on shutdown

**Alternatives Considered**:
1. **Configurable port** - Rejected: Constitution requires OS-assigned
2. **Listen on all interfaces** - Rejected: Security risk, violates constitution
3. **No graceful shutdown** - Rejected: Violates constitution principle IV

**Libraries/Tooling**: 
- `google.golang.org/grpc` 
- Go stdlib `net`, `os/signal`, `context`

---

### 10. Error Handling and Status Codes

**Decision**: Standard gRPC status codes with detailed messages

**Rationale**:
- Align with provider development standards (FR-023)
- Clear mapping of errors to gRPC codes
- Include actionable context (variable names, paths)

**Implementation Approach**:
- `NotFound`: Environment variable doesn't exist
- `InvalidArgument`: Malformed path, invalid config, JSON parse error, missing required vars
- `FailedPrecondition`: Fetch called before Init
- `Internal`: Unexpected errors (bugs)
- All errors include variable names, paths, or config keys

**Alternatives Considered**:
1. **Always return Internal** - Rejected: Loses error semantics
2. **Custom error codes** - Rejected: Standard codes are well-understood
3. **Minimal error messages** - Rejected: Violates actionable error requirement

**Libraries/Tooling**: `google.golang.org/grpc/status`, `google.golang.org/grpc/codes`

---

## Summary of Decisions

| Area | Decision | Key Dependency |
|------|----------|---------------|
| Path transformation | Configurable separator + case | `strings` (stdlib) |
| Prefix handling | Dual-mode (prepend/filter_only) | `strings` (stdlib) |
| Type conversion | Number → Boolean → String | `strconv` (stdlib) |
| JSON parsing | Strict with errors | `encoding/json` (stdlib) |
| Caching | Read-once, cache-forever | `sync.Map` (stdlib) |
| Case sensitivity | Platform-native | `os.Getenv` (stdlib) |
| Required vars | Validate original names at Init | `os.LookupEnv` (stdlib) |
| Logging | Custom structured logger | `log` (stdlib) |
| gRPC server | Loopback, OS port, graceful shutdown | `grpc`, `net`, `signal` |
| Error handling | Standard gRPC codes + context | `grpc/status` |

## External Dependencies Analysis

**Direct Dependencies** (from go.mod):
- `google.golang.org/grpc` - gRPC server framework
- `google.golang.org/protobuf` - Protocol buffer runtime
- `github.com/autonomous-bits/nomos/libs/provider-proto` - Contract definitions

**Notable**: All other functionality uses Go standard library. Zero external dependencies for core logic.

## Open Questions

None - all technical decisions resolved through specification clarifications.
