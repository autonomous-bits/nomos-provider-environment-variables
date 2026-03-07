# Data Model: Wildcard Path Spread Operator

**Phase**: 1 — Design & Contracts  
**Branch**: `002-wildcard-path-spread`  
**Date**: 2026-03-07

---

## Entities

### WildcardPathRequest

A `FetchRequest` whose `Path` slice ends with the literal segment `"*"`.

| Field | Type | Description |
|-------|------|-------------|
| `Path` | `[]string` | One or more segments; the last MUST be `"*"` |
| `Path[0..n-2]` | `[]string` | Namespace segments used to construct the match prefix |
| `Path[n-1]` | `string` | Always `"*"` — the wildcard operator |

**Validation rules:**
- Any path segment equal to `"*"` that is NOT the last segment → `InvalidArgument` error.
- A path of `["*"]` (root wildcard) is valid.
- An empty path (`[]`) remains invalid (existing rule, unchanged).

---

### PathPrefix

The resolved environment variable name prefix used for wildcard matching. Built from the namespace segments preceding `*`.

| Attribute | Value |
|-----------|-------|
| Source | `Resolver.BuildPrefix(path[:len(path)-1])` |
| Case | Determined by provider `case_transform` config (same as single-key lookups) |
| Separator | Determined by provider `separator` config |
| Trailing separator | Always appended when namespace is non-empty (ensures full-segment boundary matching) |
| Provider prefix | Prepended when `prefix_mode == "prepend"` and `prefix != ""` |

**Examples** (separator `_`, case `upper`, provider prefix `MYAPP_`):

| Path | PathPrefix | Matches |
|------|-----------|---------|
| `["*"]` | `"MYAPP_"` | All vars starting with `MYAPP_` |
| `["database", "*"]` | `"MYAPP_DATABASE_"` | `MYAPP_DATABASE_HOST`, `MYAPP_DATABASE_PORT`, … |
| `["app", "db", "*"]` | `"MYAPP_APP_DB_"` | `MYAPP_APP_DB_HOST`, … |

**Examples** (separator `_`, case `upper`, no provider prefix):

| Path | PathPrefix | Matches |
|------|-----------|---------|
| `["*"]` | `""` | All env vars |
| `["database", "*"]` | `"DATABASE_"` | `DATABASE_HOST`, `DATABASE_PORT`, … |

---

### SpreadResult

The in-memory representation of a wildcard query result before serialisation.

| Field | Type | Description |
|-------|------|-------------|
| `entries` | `map[string]interface{}` | Key: raw suffix of original env var name (PathPrefix stripped, no case transform). Value: string or type-converted value. |

**Key derivation:** Given env var `MYAPP_DATABASE_HOST=localhost` and `PathPrefix = "MYAPP_DATABASE_"`:
```
key = "HOST"   // raw suffix, original casing, no transformation
```

**Empty result:** An empty `map[string]interface{}` is valid and maps to an empty `structpb.Struct`. Not an error.

**Key collision:** If two env vars produce the same key after prefix stripping (operator misconfiguration), the surviving value is non-deterministic (`os.Environ()` order). The provider does not detect or report this.

---

### WildcardError

A gRPC error returned when the wildcard operator is misused.

| Scenario | gRPC Code | Message |
|----------|-----------|---------|
| `*` in non-terminal position | `InvalidArgument` | `"wildcard operator '*' is only valid at the terminal position of a path; found at index %d"` |
| Multiple `*` segments in non-terminal positions | `InvalidArgument` | Same — reports the first offending index |

---

## State Transitions

No new provider state transitions. Wildcard requests are handled entirely within the existing `StateReady` processing path — the same pre-condition check applies as for single-key `Fetch` calls.

---

## Package Changes

| Package | Change | Reason |
|---------|--------|--------|
| `internal/resolver` | Add `BuildPrefix(path []string) (string, error)` to `Resolver` | Construct the env var prefix from namespace segments |
| `internal/fetcher` | Add `FetchAll(matchPrefix string) (map[string]string, error)` to `Fetcher` | Enumerate and filter env vars by prefix |
| `internal/provider` | Update `Fetch` handler in `fetch.go` to detect and route wildcard paths | Protocol entry point |
| `internal/provider` | Reuse `convertValue` helper per entry in the spread result | Apply type conversion consistently |
