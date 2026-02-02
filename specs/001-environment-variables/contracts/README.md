# gRPC Contract: Environment Variables Provider

**Feature**: Environment Variables Provider  
**Date**: 2026-02-01  
**Protocol Version**: v1

## Overview

The environment variables provider implements the standard Nomos provider gRPC contract defined in:

```
github.com/autonomous-bits/nomos/libs/provider-proto/proto/nomos/provider/v1/provider.proto
```

This document specifies provider-specific configuration and behavior details.

## Service Contract

```protobuf
service ProviderService {
  rpc Init(InitRequest) returns (InitResponse);
  rpc Fetch(FetchRequest) returns (FetchResponse);
  rpc Info(InfoRequest) returns (InfoResponse);
  rpc Health(HealthRequest) returns (HealthResponse);
  rpc Shutdown(ShutdownRequest) returns (ShutdownResponse);
}
```

**Reference**: See `provider-development-standards.md` for complete protocol specification.

---

## Provider-Specific Configuration

The `InitRequest.config` field accepts the following provider-specific configuration:

### Configuration Schema (protobuf Struct)

```json
{
  "separator": "_",
  "case_transform": "upper",
  "prefix": "",
  "prefix_mode": "prepend",
  "required_variables": [],
  "enable_type_conversion": true,
  "enable_json_parsing": true
}
```

### Field Specifications

#### `separator` (optional, default: `"_"`)

- **Type**: `string`
- **Description**: Character used to join path segments when transforming to environment variable names
- **Valid Values**: Any single character (e.g., `"_"`, `"-"`, `"."`)
- **Example**: With separator `"_"`, path `["database", "host"]` becomes `"database_host"` (before case transform)

#### `case_transform` (optional, default: `"upper"`)

- **Type**: `string`
- **Description**: Case transformation applied to each path segment
- **Valid Values**: `"upper"`, `"lower"`, `"preserve"`
- **Behavior**:
  - `"upper"`: Convert all segments to uppercase (e.g., `"database"` → `"DATABASE"`)
  - `"lower"`: Convert all segments to lowercase (e.g., `"Database"` → `"database"`)
  - `"preserve"`: Keep original case (e.g., `"DataBase"` → `"DataBase"`)

#### `prefix` (optional, default: `""`)

- **Type**: `string`
- **Description**: Prefix string for filtering/prepending environment variables
- **Behavior**: See `prefix_mode` for details
- **Example**: `"MYAPP_"` to scope to variables like `MYAPP_DATABASE_HOST`

#### `prefix_mode` (optional, default: `"prepend"`)

- **Type**: `string`
- **Description**: Controls how the prefix is applied
- **Valid Values**: `"prepend"`, `"filter_only"`
- **Behavior**:
  - `"prepend"`: Transform path first, then prepend prefix to create variable name
    - Example: `prefix="APP_"`, path `["db", "host"]` → transform → `"DB_HOST"` → prepend → `"APP_DB_HOST"`
  - `"filter_only"`: Only expose variables with the prefix, but don't automatically prepend
    - Example: `prefix="APP_"`, path `["APP_DB_HOST"]` → `"APP_DB_HOST"` (user must include prefix in path)

#### `required_variables` (optional, default: `[]`)

- **Type**: `array of string`
- **Description**: List of environment variable names that must exist before Init succeeds
- **Validation**: Checked using original environment variable names (after prefix prepending if `prefix_mode="prepend"`)
- **Example**: `["DATABASE_URL", "API_KEY"]`
- **Behavior**: Init returns `InvalidArgument` error if any required variable is missing

#### `enable_type_conversion` (optional, default: `true`)

- **Type**: `boolean`
- **Description**: Whether to automatically convert string values to appropriate types
- **Behavior**:
  - `true`: Attempt conversion with precedence: Number → Boolean → String
  - `false`: Return all values as strings
- **Type Detection**:
  - **Number**: Valid integer or float (e.g., `"1234"`, `"3.14"`)
  - **Boolean**: `"true"`, `"false"`, `"yes"`, `"no"` (case-insensitive)

#### `enable_json_parsing` (optional, default: `true`)

- **Type**: `boolean`
- **Description**: Whether to parse JSON-formatted environment variable values
- **Behavior**:
  - `true`: Parse values starting with `{` or `[` as JSON
  - `false`: Return all values as strings, even if they look like JSON
- **Error Handling**: Returns `InvalidArgument` error if JSON parsing fails (malformed JSON)

---

## RPC Method Behaviors

### Init

**Request**: `InitRequest`
```protobuf
message InitRequest {
  string alias = 1;                    // Provider instance name
  google.protobuf.Struct config = 2;   // Provider-specific config (see above)
  string source_file_path = 3;         // Absolute path to .csl file
}
```

**Response**: `InitResponse`
```protobuf
message InitResponse {
  // Empty - no provider-specific fields
}
```

**Provider Behavior**:
1. Parse and validate config fields (see Configuration Schema above)
2. Check required variables exist in environment
3. Return `InvalidArgument` if validation fails
4. Transition to READY state if validation succeeds

**Error Codes**:
- `InvalidArgument`: Invalid config field, missing required variable
- `Internal`: Unexpected error during initialization

---

### Fetch

**Request**: `FetchRequest`
```protobuf
message FetchRequest {
  repeated string path = 1;            // Path segments (e.g., ["database", "host"])
}
```

**Response**: `FetchResponse`
```protobuf
message FetchResponse {
  google.protobuf.Struct value = 1;    // Environment variable value (typed)
}
```

**Provider Behavior**:
1. Validate path (non-empty, no empty segments)
2. Transform path to environment variable name using config rules
3. Check cache for variable name
4. If cache miss: fetch from `os.Getenv()`, apply type conversion, store in cache
5. Convert cached value to `protobuf.Struct`
6. Return value

**Path Transformation Algorithm**:
```
1. Validate path (must have ≥1 segment, no empty strings)
2. Apply case transform to each segment
3. Join segments with separator
4. If prefix is set and prefix_mode="prepend": prepend prefix
5. Result: environment variable name
```

**Type Conversion Algorithm** (if `enable_type_conversion=true`):
```
1. Try numeric conversion (strconv.ParseInt, strconv.ParseFloat)
   → If success: return number
2. Try boolean conversion (true/false/yes/no, case-insensitive)
   → If success: return boolean
3. If enable_json_parsing=true and value starts with { or [:
   → Parse JSON
   → If success: return object/array
   → If failure: return InvalidArgument error
4. Return original string
```

**Error Codes**:
- `NotFound`: Environment variable does not exist
- `InvalidArgument`: Malformed path, JSON parsing failure, value size exceeds 1MB
- `FailedPrecondition`: Fetch called before Init
- `Internal`: Unexpected error

---

### Info

**Request**: `InfoRequest` (empty)

**Response**: `InfoResponse`
```protobuf
message InfoResponse {
  string alias = 1;      // From InitRequest (empty if not yet initialized)
  string version = 2;    // Provider version (e.g., "1.0.0")
  string type = 3;       // Provider type: "environment-variables"
}
```

**Provider Behavior**:
- Return static type: `"environment-variables"`
- Return build version (injected via ldflags)
- Return alias if Init has been called

---

### Health

**Request**: `HealthRequest` (empty)

**Response**: `HealthResponse`
```protobuf
message HealthResponse {
  enum Status {
    STATUS_UNSPECIFIED = 0;
    STATUS_OK = 1;
    STATUS_DEGRADED = 2;
    STATUS_STARTING = 3;
  }
  
  Status status = 1;
  string message = 2;
}
```

**Provider Behavior**:
- Return `STATUS_DEGRADED` if state is UNINITIALIZED
- Return `STATUS_OK` if state is READY
- Return `STATUS_DEGRADED` if state is SHUTTING_DOWN
- Message includes current state and any issues

---

### Shutdown

**Request**: `ShutdownRequest` (empty)

**Response**: `ShutdownResponse` (empty)

**Provider Behavior**:
1. Transition to SHUTTING_DOWN state
2. Clear cache (free memory)
3. Stop background goroutines (if any)
4. Transition to STOPPED state
5. Return success

**Error Codes**: None - best effort (errors are logged but not returned)

---

## Configuration Examples

### Minimal Configuration

```json
{
  "version": "1.0.0"
}
```

All defaults apply:
- `separator="_"`
- `case_transform="upper"`
- Type conversion enabled
- JSON parsing enabled

**Usage in CSL**:
```csl
source environment-variables as env {
  version = "1.0.0"
}

// Fetch DATABASE_HOST
database_host = import env["database"]["host"]
```

---

### Custom Separator and Case

```json
{
  "version": "1.0.0",
  "config": {
    "separator": "-",
    "case_transform": "lower"
  }
}
```

**Usage**:
```csl
source environment-variables as env {
  version = "1.0.0"
  config = {
    separator = "-"
    case_transform = "lower"
  }
}

// Fetch database-host (lowercase with dash separator)
database_host = import env["database"]["host"]
```

---

### Prefix Filtering with Auto-Prepend

```json
{
  "version": "1.0.0",
  "config": {
    "prefix": "MYAPP_",
    "prefix_mode": "prepend"
  }
}
```

**Usage**:
```csl
source environment-variables as env {
  version = "1.0.0"
  config = {
    prefix = "MYAPP_"
    prefix_mode = "prepend"
  }
}

// Fetch MYAPP_DATABASE_HOST (prefix automatically prepended)
database_host = import env["database"]["host"]
```

---

### Required Variables Validation

```json
{
  "version": "1.0.0",
  "config": {
    "required_variables": ["DATABASE_URL", "API_KEY", "SECRET_KEY"]
  }
}
```

**Behavior**:
- Init fails immediately if any of these variables are missing
- Error message lists all missing variables

---

### Disable Type Conversion

```json
{
  "version": "1.0.0",
  "config": {
    "enable_type_conversion": false,
    "enable_json_parsing": false
  }
}
```

**Behavior**:
- All values returned as strings
- Useful for strict string-only configurations

---

## Error Response Examples

### Missing Required Variable

```json
{
  "code": "InvalidArgument",
  "message": "required environment variables missing: API_KEY, SECRET_KEY"
}
```

### Environment Variable Not Found

```json
{
  "code": "NotFound",
  "message": "environment variable not found: DATABASE_HOST"
}
```

### Malformed Path

```json
{
  "code": "InvalidArgument",
  "message": "path[2] cannot be empty string"
}
```

### JSON Parsing Failure

```json
{
  "code": "InvalidArgument",
  "message": "failed to parse JSON value for CONFIG: invalid character '}' looking for beginning of object key string"
}
```

### Value Size Exceeded

```json
{
  "code": "InvalidArgument",
  "message": "environment variable value exceeds maximum size of 1048576 bytes (got 2000000 bytes)"
}
```

---

## Implementation Reference

See `data-model.md` for detailed entity definitions and transformations.

**Key Files**:
- `internal/config/`: Configuration parsing and validation
- `internal/resolver/`: Path-to-variable-name transformation
- `internal/fetcher/`: Environment variable fetching and type conversion
- `internal/provider/`: gRPC service implementation
