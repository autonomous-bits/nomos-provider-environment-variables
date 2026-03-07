# Implementation Plan: Wildcard Path Spread Operator

**Branch**: `002-wildcard-path-spread` | **Date**: 2026-03-07 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `specs/002-wildcard-path-spread/spec.md`

## Summary

Add wildcard spread operator support to the `Fetch` RPC: when the terminal path segment is `"*"`, the provider returns all environment variables whose names begin with the prefix derived from the preceding path segments, as a single unary `FetchResponse` containing a `structpb.Struct` map. Three new pieces of logic are added — (1) wildcard detection and validation in the existing `Fetch` handler, (2) `BuildPrefix` on `Resolver` to construct the match prefix from namespace segments, and (3) `FetchAll` on `Fetcher` to enumerate env vars by prefix — with zero new dependencies and no protocol schema changes.

## Technical Context

**Language/Version**: Go 1.26 (go.mod) — constitution requires Go 1.25+
**Primary Dependencies**: `google.golang.org/grpc`, `google.golang.org/protobuf`, `github.com/autonomous-bits/nomos/libs/provider-proto` — all existing; no new dependencies
**Storage**: N/A — environment variables are process-level state; `os.Environ()` is the data source
**Testing**: Go standard `testing` package, `go test -race`, existing integration test framework in `tests/integration/`
**Target Platform**: Linux/macOS (darwin/arm64, darwin/amd64, linux/amd64) — binary portability per constitution
**Project Type**: single
**Performance Goals**: Wildcard response time no worse than the cumulative time of equivalent individual single-key lookups for the same variables (SC-002; no superlinear per-result overhead)
**Constraints**: `*` in non-terminal path position must return `InvalidArgument`; empty wildcard result must return empty struct, not an error; no new gRPC methods or proto changes
**Scale/Scope**: Single provider subprocess; env var count is typically hundreds; `os.Environ()` is O(n) per wildcard call

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. gRPC Contract Fidelity | PASS | No new RPCs; extends `Fetch` semantics only. `FetchResponse.Value` (Struct) already supports map payloads. |
| II. Binary Portability | PASS | No new external dependencies; `os.Environ()` is stdlib. Cross-platform builds unchanged. |
| III. Integration Testing | GATE | New integration tests required: root wildcard, prefixed wildcard, nested wildcard, invalid placement, empty result, prefix+wildcard combination. Must follow TDD (tests written first). |
| IV. Process Isolation & Clean Lifecycle | PASS | No changes to startup, shutdown, or signal handling. |
| V. Observability | GATE | Wildcard operations must log: resolved match prefix (DEBUG), result count (DEBUG), validation errors (WARN/ERROR). |

**Post-design re-check**: Both gates (III, V) are satisfied by the design — the implementation plan explicitly includes integration test tasks and observability logging. No violations require justification.

## Project Structure

### Documentation (this feature)

```text
specs/002-wildcard-path-spread/
├── plan.md                      <- this file
├── research.md                  <- Phase 0 (complete)
├── data-model.md                <- Phase 1 (complete)
├── quickstart.md                <- Phase 1 (complete)
├── contracts/
│   └── wildcard-fetch.md        <- Phase 1 (complete)
└── tasks.md                     <- Phase 2 (/speckit.tasks - not yet created)
```

### Source Code (repository root)

```text
internal/
├── resolver/
│   └── resolver.go          <- add BuildPrefix method
├── fetcher/
│   └── fetcher.go           <- add FetchAll method
└── provider/
    └── fetch.go             <- add wildcard detection + routing

tests/
├── unit/
│   ├── resolver_test.go     <- add BuildPrefix unit tests
│   ├── fetcher_test.go      <- add FetchAll unit tests
│   └── fetcher_unix_test.go <- FetchAll platform tests if needed
└── integration/
    └── grpc_test.go         <- add wildcard Fetch integration tests
```

**Structure Decision**: Single project. All new logic extends existing packages in `internal/`. No new packages or files created. Tests follow the established pattern of `tests/unit/` for unit tests and `tests/integration/` for gRPC contract tests.

## Implementation Design

### 1. `internal/resolver` — `BuildPrefix`

Add method to `Resolver`:

```go
// BuildPrefix constructs the environment variable name prefix used for wildcard
// matching from the namespace segments preceding "*" in the path.
// Returns the match prefix (may be empty for root-level wildcards with no
// provider prefix). The trailing separator is appended when namespace segments
// are non-empty to ensure full-segment boundary matching.
func (r *Resolver) BuildPrefix(namespacePath []string) (string, error)
```

**Algorithm**:

- If `namespacePath` is empty (root wildcard):
  - `prefixMode == "prepend"` and `prefix != ""` → return `r.prefix`
  - Otherwise → return `""`
- If `namespacePath` is non-empty:
  - Validate no empty segments (reuses `ErrEmptySegment`)
  - `transformed = TransformSegments(namespacePath, r.caseTransform)`
  - `joined = strings.Join(transformed, r.separator)`
  - `prefixMode == "prepend"` and `prefix != ""` → return `r.prefix + joined + r.separator`
  - Otherwise → return `joined + r.separator`

---

### 2. `internal/fetcher` — `FetchAll`

Add method to `Fetcher`:

```go
// FetchAll returns all environment variables whose names begin with matchPrefix.
// Keys in the returned map are the raw suffix after stripping matchPrefix (no
// case transformation applied). Values are raw string values.
// Returns an empty map (not an error) when no variables match.
func (f *Fetcher) FetchAll(matchPrefix string) (map[string]string, error)
```

**Algorithm**:

- `envs = os.Environ()` — snapshot of all env vars
- For each `"KEY=VALUE"` entry:
  - `key, val, _ = strings.Cut(entry, "=")`
  - If `matchPrefix == ""` OR `strings.HasPrefix(key, matchPrefix)`:
    - `relKey = key[len(matchPrefix):]` — raw suffix
    - Skip if `relKey == ""` (exact-prefix match with no sub-key)
    - Skip with WARN log if `len(val) > MaxValueSize`
    - `result[relKey] = val`
- Return `result, nil`

No caching — `FetchAll` always calls `os.Environ()` for a current snapshot.

---

### 3. `internal/provider/fetch.go` — Wildcard Routing

**New validation pass** (inserted after existing empty-path check):

```go
for i, seg := range req.Path {
    if seg == "*" && i != len(req.Path)-1 {
        return nil, status.Errorf(codes.InvalidArgument,
            "wildcard operator '*' is only valid at the terminal position of a path; found at index %d", i)
    }
}
```

**Wildcard dispatch** (before existing single-segment / multi-segment branch):

```go
if req.Path[len(req.Path)-1] == "*" {
    return p.fetchWildcard(ctx, req.Path)
}
```

**New `fetchWildcard` helper**:

```go
func (p *Provider) fetchWildcard(_ context.Context, path []string) (*pb.FetchResponse, error) {
    namespacePath := path[:len(path)-1]

    matchPrefix, err := p.resolver.BuildPrefix(namespacePath)
    // ... log + return InvalidArgument on error

    p.logger.Debug("wildcard fetch: matchPrefix=%q path=%v", matchPrefix, path)

    rawEntries, err := p.fetcher.FetchAll(matchPrefix)
    // ... log + return Internal on error

    p.logger.Debug("wildcard fetch: %d variables matched prefix %q", len(rawEntries), matchPrefix)

    fields := make(map[string]interface{}, len(rawEntries))
    for key, val := range rawEntries {
        converted, err := p.maybeConvert(val)
        // ... log + return InvalidArgument on error
        fields[key] = converted
    }

    resultStruct, err := structpb.NewStruct(fields)
    // ... log + return Internal on error

    return &pb.FetchResponse{Value: resultStruct}, nil
}
```

`maybeConvert` is a thin wrapper calling `p.convertValue(val)` when type conversion is configured, otherwise returning `val` as-is.

---

## Complexity Tracking

> No constitution violations — no entries required.
