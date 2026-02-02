# Data Model: Environment Variables Provider

**Feature**: Environment Variables Provider  
**Date**: 2026-02-01  
**Status**: Complete

## Overview

This document defines the data structures, entities, and their relationships for the environment variables provider. All entities are designed for in-memory operation with no persistent storage requirements.

## Core Entities

### 1. Provider Configuration

Represents the validated configuration passed during the Init RPC.

**Source**: `InitRequest.config` (protobuf Struct)  
**Lifecycle**: Created during Init, immutable thereafter  
**Storage**: In-memory field in Provider struct

**Attributes**:

| Field | Type | Default | Required | Description |
|-------|------|---------|----------|-------------|
| `separator` | string | `"_"` | No | Character used to join path segments (e.g., `_`, `-`, `.`) |
| `case_transform` | string | `"upper"` | No | Case transformation: `"upper"`, `"lower"`, or `"preserve"` |
| `prefix` | string | `""` | No | Filter prefix for environment variables (e.g., `"MYAPP_"`) |
| `prefix_mode` | string | `"prepend"` | No | Prefix behavior: `"prepend"` or `"filter_only"` |
| `required_variables` | []string | `[]` | No | List of environment variable names that must exist |
| `enable_type_conversion` | bool | `true` | No | Whether to convert strings to numbers/booleans |
| `enable_json_parsing` | bool | `true` | No | Whether to parse JSON-formatted values |

**Validation Rules**:
- `case_transform` must be one of: `upper`, `lower`, `preserve`
- `prefix_mode` must be one of: `prepend`, `filter_only`
- `separator` must be a single character (length 1)
- `required_variables` entries must be non-empty strings

**Example (Go struct)**:
```go
type Config struct {
    Separator            string
    CaseTransform        string
    Prefix               string
    PrefixMode           string
    RequiredVariables    []string
    EnableTypeConversion bool
    EnableJSONParsing    bool
}
```

---

### 2. Path

Represents a hierarchical reference to a configuration value in CSL.

**Source**: `FetchRequest.path` (repeated string)  
**Lifecycle**: Received per Fetch RPC, not stored  
**Example**: `["database", "host"]` (from CSL: `env["database"]["host"]`)

**Attributes**:

| Field | Type | Description |
|-------|------|-------------|
| `segments` | []string | Ordered array of path components |

**Constraints**:
- Must have at least 1 segment (empty paths are invalid)
- Segments cannot be empty strings
- Platform-specific max length (typically 255 chars total)

**Transformation**:
- Path segments are joined using the configured separator
- Case transformation is applied to each segment
- Result is the environment variable name to lookup

**Example Transformations**:

| Path | Separator | Case Transform | Prefix | Prefix Mode | Result Variable |
|------|-----------|----------------|--------|-------------|-----------------|
| `["API_KEY"]` | `_` | `upper` | `""` | `prepend` | `API_KEY` |
| `["database", "host"]` | `_` | `upper` | `""` | `prepend` | `DATABASE_HOST` |
| `["db", "user"]` | `_` | `upper` | `APP_` | `prepend` | `APP_DB_USER` |
| `["version"]` | `_` | `lower` | `myapp_` | `prepend` | `myapp_version` |
| `["APP_KEY"]` | `_` | `preserve` | `APP_` | `filter_only` | `APP_KEY` |

---

### 3. Environment Variable

Represents a name-value pair from the process environment.

**Source**: Operating system process environment (`os.Getenv`, `os.LookupEnv`)  
**Lifecycle**: Read from environment, never modified  
**Case Sensitivity**: Platform-native (Unix: sensitive, Windows: insensitive)

**Attributes**:

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Variable name (case-sensitive on Unix) |
| `value` | string | Variable value (always a string in OS environment) |
| `exists` | bool | Whether the variable is set (distinguishes from empty value) |

**Constraints**:
- Names follow platform conventions (typically uppercase on Unix)
- Values are unbounded by OS, but provider enforces 1MB limit (FR-029)
- Empty values (`VAR=`) are distinct from unset variables

**Special Cases**:
- **Empty value**: `exists=true, value=""`
- **Unset variable**: `exists=false, value=""`
- **Windows case-insensitive**: `PATH` and `path` resolve to same variable
- **Unix case-sensitive**: `PATH` and `path` are different variables

---

### 4. Cached Value

Represents a transformed and typed value stored in the provider's cache.

**Source**: Environment variable after transformation and type conversion  
**Lifecycle**: Created on first fetch, retained until shutdown  
**Storage**: In-memory `sync.Map` keyed by variable name

**Attributes**:

| Field | Type | Description |
|-------|------|-------------|
| `variable_name` | string | Original environment variable name used as cache key |
| `raw_value` | string | Untransformed string value from environment |
| `typed_value` | interface{} | Value after type conversion (number, bool, map, list, or string) |
| `type` | string | Detected type: `"string"`, `"number"`, `"boolean"`, `"object"`, `"array"` |

**Type Conversion Rules** (when `enable_type_conversion=true`):

| Raw Value | Conversion Priority | Result Type | Result Value |
|-----------|-------------------|-------------|--------------|
| `"1234"` | Number | `number` | `int64(1234)` |
| `"3.14"` | Number | `number` | `float64(3.14)` |
| `"true"` | Boolean (after number fails) | `boolean` | `true` |
| `"false"` | Boolean | `boolean` | `false` |
| `"yes"` | Boolean | `boolean` | `true` |
| `"no"` | Boolean | `boolean` | `false` |
| `"{"key":"value"}"` | JSON | `object` | `map[string]interface{}{"key": "value"}` |
| `"[1,2,3]"` | JSON | `array` | `[]interface{}{1, 2, 3}` |
| `"hello"` | None | `string` | `"hello"` |
| `""` | None | `string` | `""` |

**JSON Parsing** (when `enable_json_parsing=true`):
- Triggered only if value starts with `{` or `[`
- Uses `encoding/json.Unmarshal`
- Returns `InvalidArgument` error if parsing fails
- Result is converted to `structpb.Value` for protobuf compatibility

**Example (Go struct)**:
```go
type CachedValue struct {
    VariableName string
    RawValue     string
    TypedValue   interface{}
    Type         string
}
```

---

### 5. Transformation Rule

Represents the logic for converting paths to environment variable names.

**Lifecycle**: Derived from Config, applied during each Fetch  
**Not Stored**: Behavior is implemented in resolver functions

**Logic Flow**:

```
1. Receive path: ["database", "host"]
2. Apply case transform to each segment:
   - upper: "database" → "DATABASE", "host" → "HOST"
   - lower: "database" → "database", "host" → "host"
   - preserve: no change
3. Join segments with separator:
   - separator="_": ["DATABASE", "HOST"] → "DATABASE_HOST"
4. Apply prefix logic:
   - prefix="APP_", prefix_mode="prepend": "DATABASE_HOST" → "APP_DATABASE_HOST"
   - prefix="APP_", prefix_mode="filter_only": "DATABASE_HOST" (no change, but filtered)
5. Result: "APP_DATABASE_HOST" (or error if not found/filtered out)
```

**Pseudo-code**:
```go
func ResolvePath(path []string, config Config) (string, error) {
    // Validate path
    if len(path) == 0 {
        return "", ErrEmptyPath
    }
    
    // Transform each segment
    transformed := make([]string, len(path))
    for i, segment := range path {
        transformed[i] = ApplyCaseTransform(segment, config.CaseTransform)
    }
    
    // Join with separator
    varName := strings.Join(transformed, config.Separator)
    
    // Apply prefix
    if config.Prefix != "" && config.PrefixMode == "prepend" {
        varName = config.Prefix + varName
    }
    
    return varName, nil
}
```

---

## Entity Relationships

```
┌─────────────────────────────────────────────────────────────┐
│ Provider Process                                            │
│                                                             │
│  ┌─────────────────┐                                        │
│  │ Provider        │                                        │
│  │ ├─ alias        │                                        │
│  │ ├─ config ──────┼───> Config                            │
│  │ └─ cache ───────┼───> sync.Map<varName, CachedValue>    │
│  └─────────────────┘                                        │
│          │                                                   │
│          │ on Fetch RPC                                     │
│          ▼                                                   │
│  ┌─────────────────┐                                        │
│  │ FetchRequest    │                                        │
│  │ └─ path ────────┼───> Path ["database", "host"]         │
│  └─────────────────┘                                        │
│          │                                                   │
│          │ transform using Config                           │
│          ▼                                                   │
│  ┌─────────────────┐                                        │
│  │ Variable Name   │                                        │
│  │ "DATABASE_HOST" │                                        │
│  └─────────────────┘                                        │
│          │                                                   │
│          │ check cache                                      │
│          ▼                                                   │
│  ┌─────────────────┐     cache miss                         │
│  │ Cache Lookup    │ ────────────────> os.Getenv()         │
│  └─────────────────┘                  │                     │
│          │                             │                     │
│          │ cache hit                   │                     │
│          ▼                             ▼                     │
│  ┌─────────────────┐           ┌──────────────────┐        │
│  │ CachedValue     │           │ Environment      │        │
│  │ ├─ variable_name│ <─────────│ Variable         │        │
│  │ ├─ raw_value    │   store   │ └─ value: string │        │
│  │ ├─ typed_value  │           └──────────────────┘        │
│  │ └─ type         │                                        │
│  └─────────────────┘                                        │
│          │                                                   │
│          │ convert to protobuf                              │
│          ▼                                                   │
│  ┌─────────────────┐                                        │
│  │ FetchResponse   │                                        │
│  │ └─ value: Struct│───> protobuf Struct                   │
│  └─────────────────┘                                        │
└─────────────────────────────────────────────────────────────┘
```

**Relationship Summary**:

1. **Provider → Config**: 1:1 (one config per provider instance)
2. **Provider → Cache**: 1:1 (one cache per provider instance)
3. **Cache → CachedValue**: 1:N (multiple cached values, keyed by variable name)
4. **Path → Variable Name**: 1:1 (deterministic transformation per fetch)
5. **Variable Name → Environment Variable**: 1:1 (direct lookup in OS environment)
6. **Environment Variable → CachedValue**: 1:1 (one cached value per variable)

---

## State Transitions

### Provider Lifecycle States

```
┌─────────────────┐
│ UNINITIALIZED   │ (Provider created, no config)
└────────┬────────┘
         │ Init RPC
         ▼
┌─────────────────┐
│  INITIALIZING   │ (Validating config, checking required vars)
└────────┬────────┘
         │ Success
         ▼
┌─────────────────┐
│   READY         │ (Accepting Fetch requests)
└────────┬────────┘
         │ Shutdown RPC
         ▼
┌─────────────────┐
│ SHUTTING_DOWN   │ (Clearing cache, stopping goroutines)
└────────┬────────┘
         │ Complete
         ▼
┌─────────────────┐
│   STOPPED       │ (Process exits)
└─────────────────┘
```

**Allowed Transitions**:
- `UNINITIALIZED → INITIALIZING`: Via Init RPC
- `INITIALIZING → READY`: On successful config validation
- `INITIALIZING → UNINITIALIZED`: On failed initialization (error returned)
- `READY → SHUTTING_DOWN`: Via Shutdown RPC
- `SHUTTING_DOWN → STOPPED`: After cleanup complete

**RPC Method Constraints**:
- `Init`: Only allowed in `UNINITIALIZED` state
- `Fetch`: Only allowed in `READY` state (returns `FailedPrecondition` otherwise)
- `Info`: Allowed in any state
- `Health`: Allowed in any state (returns `DEGRADED` if not `READY`)
- `Shutdown`: Allowed in `READY` state

---

## Data Validation

### Config Validation (during Init)

```go
func ValidateConfig(c Config) error {
    // Case transform validation
    validCaseTransforms := map[string]bool{
        "upper": true, "lower": true, "preserve": true,
    }
    if !validCaseTransforms[c.CaseTransform] {
        return fmt.Errorf("invalid case_transform: %s (must be upper, lower, or preserve)", c.CaseTransform)
    }
    
    // Prefix mode validation
    validPrefixModes := map[string]bool{
        "prepend": true, "filter_only": true,
    }
    if !validPrefixModes[c.PrefixMode] {
        return fmt.Errorf("invalid prefix_mode: %s (must be prepend or filter_only)", c.PrefixMode)
    }
    
    // Separator validation
    if len(c.Separator) != 1 {
        return fmt.Errorf("separator must be a single character, got: %q", c.Separator)
    }
    
    // Required variables validation
    for i, varName := range c.RequiredVariables {
        if varName == "" {
            return fmt.Errorf("required_variables[%d] is empty", i)
        }
        
        _, exists := os.LookupEnv(varName)
        if !exists {
            return fmt.Errorf("required environment variable missing: %s", varName)
        }
    }
    
    return nil
}
```

### Path Validation (during Fetch)

```go
func ValidatePath(path []string) error {
    if len(path) == 0 {
        return fmt.Errorf("path cannot be empty")
    }
    
    for i, segment := range path {
        if segment == "" {
            return fmt.Errorf("path[%d] cannot be empty string", i)
        }
    }
    
    return nil
}
```

### Value Size Validation (during Fetch)

```go
const MaxValueSize = 1 * 1024 * 1024 // 1MB

func ValidateValueSize(value string) error {
    if len(value) > MaxValueSize {
        return fmt.Errorf("environment variable value exceeds maximum size of %d bytes (got %d bytes)", MaxValueSize, len(value))
    }
    return nil
}
```

---

## Implementation Notes

### Thread Safety

- **Config**: Immutable after Init (no locking needed)
- **Cache**: Uses `sync.Map` for concurrent access
- **Provider state**: Atomic state transitions using `sync/atomic` or mutex

### Memory Considerations

- **Cache growth**: Unbounded (grows with unique variable accesses)
- **Mitigation**: Prefix filtering limits scope, 1MB max per value
- **Typical usage**: <100 variables cached (~10MB total)

### Performance Characteristics

| Operation | Complexity | Typical Time |
|-----------|-----------|--------------|
| Path transformation | O(n) segments | <1μs |
| Cache lookup | O(1) | <100ns |
| Environment lookup (cache miss) | O(1) | <1μs |
| Type conversion | O(n) value length | <100μs |
| JSON parsing | O(n) value length | <1ms |
| **Total fetch (cached)** | **O(1)** | **<10ms** |

---

## Summary

This data model supports the environment variables provider with:
- **Flexible configuration** for path transformation and type conversion
- **Efficient caching** for fast repeated fetches
- **Clear state transitions** for predictable lifecycle management
- **Validation at boundaries** (Init and Fetch) to fail fast
- **Thread-safe operations** for concurrent requests

All entities are designed for in-memory operation with zero persistence requirements, aligning with the read-only nature of environment variables.
