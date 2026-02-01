# Provider Development Standards

Last updated: 2026-02-01

This document defines the authoritative standards for implementing Nomos external providers. It covers the gRPC protocol contract, reserved fields, provider lifecycle, and architectural constraints that all provider implementations must follow.

## Table of Contents

- [Overview](#overview)
- [Reserved Fields and Metadata](#reserved-fields-and-metadata)
- [Provider Lifecycle and Instance Management](#provider-lifecycle-and-instance-management)
- [gRPC Protocol Contract](#grpc-protocol-contract)
- [Provider Implementation Requirements](#provider-implementation-requirements)
- [Data Flow and Responsibilities](#data-flow-and-responsibilities)
- [Type System and Value Constraints](#type-system-and-value-constraints)
- [Error Handling Standards](#error-handling-standards)
- [Security and Validation](#security-and-validation)
- [Testing Requirements](#testing-requirements)
- [References](#references)

## Overview

### What is a Nomos Provider?

A Nomos provider is a standalone executable that:
- Implements the `nomos.provider.v1.ProviderService` gRPC contract
- Runs as a subprocess managed by the Nomos CLI
- Fetches configuration data from external sources
- Returns structured data compatible with Nomos value types

### Protocol Dependency

All providers **MUST** implement the gRPC service contract defined in:
```
github.com/autonomous-bits/nomos/libs/provider-proto
```

Add this dependency to your provider:
```bash
go get github.com/autonomous-bits/nomos/libs/provider-proto@v0.1.0
```

## Reserved Fields and Metadata

### Critical: Reserved Provider Metadata Fields

The following three fields are **RESERVED** and have special meaning in the Nomos provider ecosystem. Provider implementations **MUST NOT** use these field names for any provider-specific configuration or data:

#### 1. **Alias** (Reserved)

**Definition**: The instance name for a provider as declared in `.csl` source files.

**Managed by**: Nomos CLI and compiler
**Source**: `.csl` source declaration (e.g., `source file as configs { ... }`)
**Passed to provider**: Via `InitRequest.alias`
**Provider responsibility**: Store for logging and diagnostics only
**Provider constraint**: **MUST NOT** use `alias` as a configuration key

```csl
// Example .csl declaration
source file as configs {     // "configs" is the alias
  version = "1.0.0"
  config = {
    directory = "./data"
  }
}
```

The CLI passes the alias (`"configs"`) to the provider during initialization:
```protobuf
message InitRequest {
  string alias = 1;          // Managed by CLI, not configurable by provider
  // ...
}
```

**Why it's reserved**: The CLI uses the alias to uniquely identify and manage provider instances. Each alias results in one provider subprocess.

#### 2. **Type** (Reserved)

**Definition**: The provider implementation type identifier (e.g., `file`, `http`, `vault`).

**Managed by**: Provider implementation
**Source**: `InfoResponse.type` returned by the provider
**Used by**: CLI for provider resolution, logging, and binary discovery
**Provider responsibility**: Return a stable, unique identifier in `Info()` RPC
**Provider constraint**: **MUST NOT** use `type` as a configuration key

```go
func (p *Provider) Info(ctx context.Context, req *providerv1.InfoRequest) (*providerv1.InfoResponse, error) {
    return &providerv1.InfoResponse{
        Alias:   p.alias,
        Version: "1.0.0",
        Type:    "file",        // Required: unique type identifier
    }, nil
}
```

**Why it's reserved**: The CLI uses the type to locate provider binaries in the installation directory (`.nomos/providers/{type}/{version}/...`). The type must match the binary naming convention.

#### 3. **Version** (Reserved)

**Definition**: The semantic version of the provider implementation.

**Managed by**: Provider implementation and `.csl` source declarations
**Source**: 
  - Declared in `.csl`: `version = "1.0.0"`
  - Returned by provider: `InfoResponse.version`
**Used by**: CLI for provider resolution, compatibility checks, and binary discovery
**Provider responsibility**: Return the build version in `Info()` RPC
**Provider constraint**: **MUST NOT** use `version` as a configuration key

```csl
// Declared in .csl (authoritative)
source file as configs {
  version = "1.0.0"          // Required: semantic version
  config = {
    directory = "./data"
  }
}
```

```go
// Returned by provider (must match binary version)
func (p *Provider) Info(ctx context.Context, req *providerv1.InfoRequest) (*providerv1.InfoResponse, error) {
    return &providerv1.InfoResponse{
        Alias:   p.alias,
        Version: "1.0.0",       // Must match build version
        Type:    "file",
    }, nil
}
```

**Why it's reserved**: The CLI uses the version to locate the correct provider binary and ensure compatibility. Versions follow semantic versioning (semver).

### Reserved Field Summary

| Field | Managed By | Source | Provider Access | Configurable |
|-------|-----------|--------|----------------|--------------|
| **Alias** | CLI | `.csl` declaration | Read-only (via `InitRequest`) | ❌ No |
| **Type** | Provider | `Info()` response | Provider-defined | ❌ No |
| **Version** | Provider + `.csl` | `.csl` + `Info()` response | Provider-defined | ❌ No |

### Configuration Guidelines

Provider-specific configuration **MUST** avoid these reserved field names. Use descriptive, domain-specific names:

✅ **Correct**:
```csl
source file as configs {
  version = "1.0.0"          // Reserved field (required)
  config = {
    directory = "./data"     // Provider-specific config
    format = "yaml"          // Provider-specific config
    recursive = true         // Provider-specific config
  }
}
```

❌ **Incorrect**:
```csl
source custom as data {
  version = "1.0.0"
  config = {
    alias = "my-alias"       // ❌ WRONG: 'alias' is reserved
    type = "custom"          // ❌ WRONG: 'type' is reserved
    version = "2.0.0"        // ❌ WRONG: 'version' is reserved
  }
}
```

## Provider Lifecycle and Instance Management

### Critical: One Provider Instance Per Alias

**Architectural Constraint**: Providers **MUST NOT** implement multi-instance tracking or manage multiple provider instances within a single process.

**Why**: The Nomos CLI follows a **one subprocess per alias** model:
- Each unique alias in `.csl` files results in exactly one provider subprocess
- Each subprocess handles requests for one logical provider instance
- Multiple aliases of the same type result in multiple processes

### Instance Management Responsibilities

| Responsibility | Owner | Implementation |
|---------------|-------|----------------|
| **Instance tracking** | CLI | Manages subprocess per alias |
| **Alias identification** | CLI | Passes alias via `InitRequest` |
| **Process lifecycle** | CLI | Starts, monitors, stops processes |
| **Configuration scoping** | Provider | Stores one config (from `Init`) |
| **State isolation** | Provider | Single instance state only |

### Example: Multiple Instances

```csl
// config.csl

// First instance: alias = "dev_configs"
source file as dev_configs {
  version = "1.0.0"
  config = {
    directory = "./configs/dev"
  }
}

// Second instance: alias = "prod_configs"
source file as prod_configs {
  version = "1.0.0"
  config = {
    directory = "./configs/prod"
  }
}

// CLI behavior:
// - Starts TWO subprocesses (one per alias)
// - Each subprocess receives different InitRequest:
//   - Process 1: InitRequest { alias: "dev_configs", config: { directory: "./configs/dev" } }
//   - Process 2: InitRequest { alias: "prod_configs", config: { directory: "./configs/prod" } }
```

### Provider Implementation Pattern

✅ **Correct** (single instance):
```go
type Provider struct {
    providerv1.UnimplementedProviderServiceServer
    
    // Single instance state
    alias   string                 // From InitRequest
    config  *Config               // From InitRequest
    client  *http.Client          // Single client for this instance
}

func (p *Provider) Init(ctx context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
    // Store the single instance configuration
    p.alias = req.Alias
    
    config, err := ParseConfig(req.Config.AsMap())
    if err != nil {
        return nil, status.Errorf(codes.InvalidArgument, "invalid config: %v", err)
    }
    p.config = config
    
    // Initialize single instance resources
    p.client = &http.Client{Timeout: 30 * time.Second}
    
    return &providerv1.InitResponse{}, nil
}
```

❌ **Incorrect** (multi-instance tracking):
```go
type Provider struct {
    providerv1.UnimplementedProviderServiceServer
    
    // ❌ WRONG: Don't track multiple instances in one process
    instances map[string]*Instance
    mu        sync.RWMutex
}

func (p *Provider) Init(ctx context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
    // ❌ WRONG: Don't create instance maps
    p.mu.Lock()
    defer p.mu.Unlock()
    
    p.instances[req.Alias] = &Instance{...}
    return &providerv1.InitResponse{}, nil
}
```

### Process Lifecycle

```
┌─────────────────────────────────────────────────────────────────┐
│ Nomos CLI (nomos build)                                         │
│                                                                  │
│  Discovers: alias="dev_configs", type="file", version="1.0.0"   │
│                                                                  │
│  1. Locate binary: .nomos/providers/file/1.0.0/darwin-arm64/    │
│  2. Start subprocess with random port                           │
│  3. Read PORT=50051 from stdout                                 │
│  4. Establish gRPC connection                                   │
│  5. Call Init(alias="dev_configs", config={...})                │
│  6. On each import: Call Fetch(path=[...])                      │
│  7. On completion: Call Shutdown()                              │
│  8. Terminate subprocess                                        │
└─────────────────────────────────────────────────────────────────┘
                            │
                            │ gRPC over localhost
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│ Provider Subprocess (file provider)                             │
│                                                                  │
│  - Single instance for alias "dev_configs"                      │
│  - Stores config: { directory: "./configs/dev" }                │
│  - Serves Fetch requests for this instance only                 │
│  - No knowledge of other aliases/instances                      │
└─────────────────────────────────────────────────────────────────┘
```

### Key Takeaways

1. **One subprocess per alias**: CLI creates separate processes for each alias
2. **No instance tracking**: Providers manage only their own single instance
3. **Alias identification**: CLI uses alias to route requests to correct subprocess
4. **State isolation**: Each subprocess has isolated configuration and state

## gRPC Protocol Contract

### Service Definition

All providers **MUST** implement this gRPC service from `libs/provider-proto`:

```protobuf
service ProviderService {
  rpc Init(InitRequest) returns (InitResponse);
  rpc Fetch(FetchRequest) returns (FetchResponse);
  rpc Info(InfoRequest) returns (InfoResponse);
  rpc Health(HealthRequest) returns (HealthResponse);
  rpc Shutdown(ShutdownRequest) returns (ShutdownResponse);
}
```

### RPC Method Requirements

| RPC | Required | Idempotent | Called When | Purpose |
|-----|----------|------------|-------------|---------|
| **Init** | ✅ Yes | ✅ Yes | Once per instance | Initialize provider with config |
| **Fetch** | ✅ Yes | ✅ Yes | Multiple times | Retrieve data at path |
| **Info** | ✅ Yes | ✅ Yes | Once at startup | Return provider metadata |
| **Health** | ✅ Yes | ✅ Yes | Before first fetch | Verify operational status |
| **Shutdown** | ⚠️ Best effort | ✅ Yes | At build completion | Cleanup resources |

### Init RPC

**Purpose**: Initialize the provider with instance-specific configuration.

**Called**: Exactly once per provider instance (subprocess) before any `Fetch` calls.

**Request**:
```protobuf
message InitRequest {
  string alias = 1;                    // Instance identifier (e.g., "dev_configs")
  google.protobuf.Struct config = 2;   // Provider-specific configuration
  string source_file_path = 3;         // Absolute path to .csl file
  
  reserved 4 to 10;                    // Reserved for future use
}
```

**Response**:
```protobuf
message InitResponse {
  reserved 1 to 10;                    // Reserved for future use
}
```

**Provider Requirements**:
1. **MUST** validate all required configuration keys
2. **MUST** return `InvalidArgument` error for invalid/missing config
3. **MUST** return `FailedPrecondition` if dependencies unavailable
4. **MUST** store alias for logging (not configuration)
5. **SHOULD** initialize connections to external resources
6. **SHOULD** be idempotent (multiple calls with same config = same result)

**Example**:
```go
func (p *Provider) Init(ctx context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
    p.alias = req.Alias
    
    // Parse provider-specific config
    config, err := ParseConfig(req.Config.AsMap())
    if err != nil {
        return nil, status.Errorf(codes.InvalidArgument, "invalid config: %v", err)
    }
    p.config = config
    
    // Initialize resources
    if err := p.initialize(); err != nil {
        return nil, status.Errorf(codes.FailedPrecondition, "init failed: %v", err)
    }
    
    log.Printf("[%s] initialized successfully", p.alias)
    return &providerv1.InitResponse{}, nil
}
```

### Fetch RPC

**Purpose**: Retrieve configuration data at the specified path.

**Called**: Multiple times during compilation for each import reference.

**Critical**: The CLI passes **only the path** to the provider, **not the alias**. The provider already knows its alias from the `Init` call.

**Request**:
```protobuf
message FetchRequest {
  repeated string path = 1;            // Path segments (e.g., ["database", "prod"])
  
  reserved 2 to 10;                    // Reserved for future use
}
```

**Response**:
```protobuf
message FetchResponse {
  google.protobuf.Struct value = 1;    // Structured data
  
  reserved 2 to 10;                    // Reserved for future use
}
```

**Path Semantics**:
- Path is an ordered array of string segments
- Interpretation is provider-specific
- Empty path is provider-defined (may be root or error)

**Provider Requirements**:
1. **MUST** return `NotFound` if path does not exist
2. **MUST** return `InvalidArgument` for malformed paths
3. **MUST** return `FailedPrecondition` if `Init` not called
4. **MUST** return structured data compatible with Nomos types
5. **MUST** be thread-safe (concurrent `Fetch` calls are allowed)
6. **SHOULD** be idempotent
7. **SHOULD** implement caching for expensive fetches

**Example**:
```go
func (p *Provider) Fetch(ctx context.Context, req *providerv1.FetchRequest) (*providerv1.FetchResponse, error) {
    if p.config == nil {
        return nil, status.Error(codes.FailedPrecondition, "provider not initialized")
    }
    
    // Path interpretation: provider-specific
    // Note: req does NOT contain alias - provider already knows its alias from Init
    data, err := p.fetcher.FetchData(ctx, req.Path)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            return nil, status.Errorf(codes.NotFound, "path not found: %v", req.Path)
        }
        return nil, status.Errorf(codes.Internal, "fetch failed: %v", err)
    }
    
    // Convert to protobuf Struct
    value, err := structpb.NewValue(data)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "value conversion failed: %v", err)
    }
    
    return &providerv1.FetchResponse{
        Value: value.GetStructValue(),
    }, nil
}
```

### Info RPC

**Purpose**: Return provider metadata.

**Called**: Once during provider process startup (before or after `Init`).

**Request**:
```protobuf
message InfoRequest {
  reserved 1 to 10;                    // Reserved for future use
}
```

**Response**:
```protobuf
message InfoResponse {
  string alias = 1;                    // Provider instance name (from Init)
  string version = 2;                  // Provider implementation version (semver)
  string type = 3;                     // Provider type identifier
  
  reserved 4 to 10;                    // Reserved for future use
}
```

**Provider Requirements**:
1. **MUST** return valid semantic version (e.g., "1.0.0")
2. **MUST** return stable type identifier matching binary name
3. **SHOULD** return alias if `Init` has been called
4. **MAY** return empty alias if `Init` not yet called
5. **SHOULD** include build metadata in version (e.g., "1.0.0+abc123")

**Example**:
```go
const (
    ProviderType    = "file"
    ProviderVersion = "1.0.0"  // Set via build flags
)

func (p *Provider) Info(ctx context.Context, req *providerv1.InfoRequest) (*providerv1.InfoResponse, error) {
    return &providerv1.InfoResponse{
        Alias:   p.alias,           // May be empty before Init
        Version: ProviderVersion,   // Injected at build time
        Type:    ProviderType,      // Matches binary naming
    }, nil
}
```

### Health RPC

**Purpose**: Check provider operational status.

**Called**: After connection established, before first `Fetch`.

**Request**:
```protobuf
message HealthRequest {
  reserved 1 to 10;                    // Reserved for future use
}
```

**Response**:
```protobuf
message HealthResponse {
  enum Status {
    STATUS_UNSPECIFIED = 0;            // Unknown state
    STATUS_OK = 1;                     // Healthy and ready
    STATUS_DEGRADED = 2;               // Operational but impaired
    STATUS_STARTING = 3;               // Still initializing
  }
  
  Status status = 1;                   // Current health state
  string message = 2;                  // Optional diagnostic message
  
  reserved 3 to 10;                    // Reserved for future use
}
```

**Provider Requirements**:
1. **MUST** return `STATUS_OK` when fully operational
2. **MUST** return `STATUS_DEGRADED` for partial functionality
3. **SHOULD** return `STATUS_STARTING` during long initialization
4. **SHOULD** include actionable diagnostic messages
5. **SHOULD** verify external dependency availability

**Example**:
```go
func (p *Provider) Health(ctx context.Context, req *providerv1.HealthRequest) (*providerv1.HealthResponse, error) {
    // Check external dependencies
    if err := p.checkConnectivity(); err != nil {
        return &providerv1.HealthResponse{
            Status:  providerv1.HealthResponse_STATUS_DEGRADED,
            Message: fmt.Sprintf("connectivity check failed: %v", err),
        }, nil
    }
    
    return &providerv1.HealthResponse{
        Status:  providerv1.HealthResponse_STATUS_OK,
        Message: "provider is healthy",
    }, nil
}
```

### Shutdown RPC

**Purpose**: Gracefully shutdown the provider.

**Called**: At the end of compilation or when CLI exits.

**Note**: This is a **best-effort** call. The CLI **MAY** forcefully terminate the process if shutdown takes too long (recommended timeout: 5 seconds).

**Request**:
```protobuf
message ShutdownRequest {
  reserved 1 to 10;                    // Reserved for future use
}
```

**Response**:
```protobuf
message ShutdownResponse {
  reserved 1 to 10;                    // Reserved for future use
}
```

**Provider Requirements**:
1. **MUST** close open connections and file handles
2. **MUST** release acquired resources
3. **MUST** complete within 5 seconds
4. **SHOULD** be idempotent
5. **SHOULD** flush buffered data if applicable
6. **MAY** ignore errors during cleanup

**Example**:
```go
func (p *Provider) Shutdown(ctx context.Context, req *providerv1.ShutdownRequest) (*providerv1.ShutdownResponse, error) {
    log.Printf("[%s] shutting down", p.alias)
    
    // Close connections (best effort)
    if p.httpClient != nil {
        p.httpClient.CloseIdleConnections()
    }
    
    if p.dbConn != nil {
        _ = p.dbConn.Close() // Ignore errors during shutdown
    }
    
    return &providerv1.ShutdownResponse{}, nil
}
```

## Provider Implementation Requirements

### Process Communication

**Port Discovery**: Providers **MUST** communicate their listening port to the CLI via stdout.

**Requirements**:
1. Listen on `127.0.0.1:0` (random available port)
2. Print `PORT={port_number}\n` to stdout as the **first line**
3. Flush stdout immediately after printing port
4. Use stderr for all subsequent logging

**Example**:
```go
func main() {
    // Listen on random available port
    lis, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }
    
    // Print port to stdout (MUST be first output)
    addr := lis.Addr().(*net.TCPAddr)
    fmt.Printf("PORT=%d\n", addr.Port)
    os.Stdout.Sync()  // CRITICAL: Flush stdout
    
    // Redirect logs to stderr
    log.SetOutput(os.Stderr)
    
    // Start gRPC server
    server := grpc.NewServer()
    providerv1.RegisterProviderServiceServer(server, NewProvider())
    
    log.Printf("Provider listening on %s", lis.Addr())
    if err := server.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}
```

### Binary Naming Convention

Provider binaries **MUST** follow this naming pattern for GitHub releases:

```
nomos-provider-{type}-{version}-{os}-{arch}[.exe]
```

Examples:
- `nomos-provider-file-1.0.0-darwin-arm64`
- `nomos-provider-file-1.0.0-darwin-amd64`
- `nomos-provider-file-1.0.0-linux-amd64`
- `nomos-provider-http-2.1.0-windows-amd64.exe`

**Platform Support** (minimum):
- `darwin/arm64` (Apple Silicon)
- `darwin/amd64` (Intel Mac)
- `linux/amd64` (Linux x86_64)
- `linux/arm64` (Linux ARM64)

### Installation Layout

The CLI installs providers in this structure:

```
.nomos/
  providers/
    {type}/
      {version}/
        {os}-{arch}/
          provider              # Executable (chmod 755)
          CHECKSUM              # Optional: SHA256 checksum
```

Example:
```
.nomos/
  providers/
    file/
      1.0.0/
        darwin-arm64/
          provider
```

### Build Integration

**Recommended**: Use `ldflags` to inject version at build time:

```go
// main.go
package main

var (
    version = "dev"      // Overridden by build flags
    commit  = "unknown"
    date    = "unknown"
)

func (p *Provider) Info(ctx context.Context, req *providerv1.InfoRequest) (*providerv1.InfoResponse, error) {
    return &providerv1.InfoResponse{
        Alias:   p.alias,
        Version: version,  // Use injected version
        Type:    "file",
    }, nil
}
```

Build command:
```bash
go build -ldflags "\
  -X main.version=1.0.0 \
  -X main.commit=$(git rev-parse --short HEAD) \
  -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o nomos-provider-file-1.0.0-darwin-arm64 \
  ./cmd/provider
```

## Data Flow and Responsibilities

### Responsibility Matrix

| Task | CLI Responsibility | Provider Responsibility |
|------|-------------------|------------------------|
| **Alias management** | Track alias-to-subprocess mapping | Store alias for logging only |
| **Type resolution** | Match type to binary path | Return stable type identifier |
| **Version management** | Enforce version from `.csl` and lock file | Return build version in `Info` |
| **Process lifecycle** | Start, monitor, stop subprocess | Handle Init/Fetch/Shutdown RPCs |
| **Path resolution** | Construct path from import expressions | Interpret path semantics |
| **Data fetching** | Call `Fetch(path)` RPC | Retrieve data from source |
| **Configuration** | Pass config from `.csl` to `Init` | Parse and validate config |
| **Error handling** | Surface gRPC errors to user | Return appropriate gRPC status codes |

### Data Flow Example

```csl
// config.csl
source file as configs {
  version = "1.0.0"
  config = {
    directory = "./data"
  }
}

app = import configs["database"]["production"]
```

**Flow**:

1. **CLI**: Parses `.csl`, discovers alias `"configs"`, type `"file"`, version `"1.0.0"`
2. **CLI**: Locates binary: `.nomos/providers/file/1.0.0/darwin-arm64/provider`
3. **CLI**: Starts subprocess
4. **Provider**: Prints `PORT=50051` to stdout
5. **CLI**: Connects to `localhost:50051` via gRPC
6. **CLI**: Calls `Init(alias="configs", config={directory: "./data"})`
7. **Provider**: Validates config, initializes, returns success
8. **CLI**: Calls `Fetch(path=["database", "production"])`
   - **NOTE**: Path only, no alias passed to Fetch
9. **Provider**: Reads `./data/database/production.csl` (or similar), returns data
10. **CLI**: Merges data into compilation
11. **CLI**: Calls `Shutdown()` when compilation completes
12. **Provider**: Cleans up, exits
13. **CLI**: Terminates subprocess if still running

## Type System and Value Constraints

### Supported Nomos Types

Provider `Fetch` responses **MUST** return data compatible with Nomos types:

| Nomos Type | Go Type | Protobuf Representation |
|-----------|---------|------------------------|
| **String** | `string` | `google.protobuf.Value` (string) |
| **Number** | `int64`, `float64` | `google.protobuf.Value` (number) |
| **Boolean** | `bool` | `google.protobuf.Value` (bool) |
| **List** | `[]interface{}` | `google.protobuf.ListValue` |
| **Map** | `map[string]interface{}` | `google.protobuf.Struct` |
| **Null** | `nil` | `google.protobuf.Value` (null) |

### Value Conversion

**Use `structpb.NewValue()` for type-safe conversion**:

```go
func (p *Provider) Fetch(ctx context.Context, req *providerv1.FetchRequest) (*providerv1.FetchResponse, error) {
    // Fetch your data
    data := map[string]interface{}{
        "database": map[string]interface{}{
            "host": "localhost",
            "port": 5432,
            "ssl": true,
            "pools": []interface{}{10, 20, 30},
        },
    }
    
    // Convert to protobuf Value
    value, err := structpb.NewValue(data)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "conversion failed: %v", err)
    }
    
    // Return Struct (not Value)
    return &providerv1.FetchResponse{
        Value: value.GetStructValue(),
    }, nil
}
```

### Value Constraints

**Requirements**:
1. **MUST** return `google.protobuf.Struct` (map[string]interface{}) from `Fetch`
2. **MUST NOT** return unsupported types (functions, channels, pointers to structs)
3. **MUST** return JSON-compatible values
4. **SHOULD** avoid deeply nested structures (>10 levels)
5. **SHOULD** avoid large values (>10MB)

## Error Handling Standards

### gRPC Status Codes

Use appropriate status codes for different error conditions:

| Status Code | When to Use | Example |
|------------|-------------|---------|
| `OK` | Success (implicit) | (No error returned) |
| `InvalidArgument` | Invalid request parameters | Path format invalid, config missing required field |
| `NotFound` | Resource not found | Path does not exist in data source |
| `FailedPrecondition` | Operation out of order | `Fetch` called before `Init` |
| `PermissionDenied` | Authorization failure | Insufficient permissions to access resource |
| `Unavailable` | Temporary failure | External service unreachable |
| `DeadlineExceeded` | Operation timeout | Fetch took too long |
| `Internal` | Unexpected error | Bug in provider implementation |

### Error Messages

**Requirements**:
1. **MUST** include actionable context in error messages
2. **MUST** avoid leaking sensitive information (passwords, tokens)
3. **SHOULD** include resource identifiers (path, config key)
4. **SHOULD** suggest remediation when possible

**Example**:
```go
// ✅ Good error messages
return nil, status.Errorf(codes.InvalidArgument, 
    "config missing required field 'directory': got %v", req.Config.AsMap())

return nil, status.Errorf(codes.NotFound, 
    "path not found: %v (searched in directory: %s)", req.Path, p.config.Directory)

return nil, status.Errorf(codes.PermissionDenied, 
    "insufficient permissions to read file: %s", filePath)

// ❌ Bad error messages
return nil, status.Error(codes.Internal, "error")  // Not actionable
return nil, status.Errorf(codes.Internal, "failed to connect to %s with password %s", 
    host, password)  // Leaks sensitive data
```

## Security and Validation

### Input Validation

**Requirements**:
1. **MUST** validate all `InitRequest.config` fields
2. **MUST** validate `FetchRequest.path` to prevent directory traversal
3. **MUST** sanitize paths before filesystem operations
4. **MUST NOT** execute arbitrary code from config
5. **SHOULD** use allowlists instead of denylists

**Example**:
```go
func (p *Provider) Fetch(ctx context.Context, req *providerv1.FetchRequest) (*providerv1.FetchResponse, error) {
    // Validate path segments
    for _, segment := range req.Path {
        if segment == "" || segment == "." || segment == ".." {
            return nil, status.Errorf(codes.InvalidArgument, "invalid path segment: %q", segment)
        }
        if strings.Contains(segment, "/") || strings.Contains(segment, "\\") {
            return nil, status.Errorf(codes.InvalidArgument, "path separator in segment: %q", segment)
        }
    }
    
    // Safe to join
    relativePath := filepath.Join(req.Path...)
    fullPath := filepath.Join(p.config.Directory, relativePath)
    
    // Ensure path is within allowed directory
    if !strings.HasPrefix(fullPath, p.config.Directory) {
        return nil, status.Error(codes.PermissionDenied, "path traversal detected")
    }
    
    // Safe to read
    data, err := os.ReadFile(fullPath)
    // ...
}
```

### Execution Permissions

**Requirements**:
- Provider binaries **MUST** have execute permissions (chmod 755)
- Provider binaries **MUST NOT** be world-writable
- CLI **MUST** verify binary permissions before execution

### Checksum Verification

**Requirements**:
- Provider releases **SHOULD** include SHA256 checksums
- CLI **MUST** verify checksums on download
- Checksums **MUST** use SHA256 algorithm

**Format** (SHA256SUMS file):
```
a1b2c3d4... nomos-provider-file-1.0.0-darwin-arm64
b2c3d4e5... nomos-provider-file-1.0.0-linux-amd64
```

## Testing Requirements

### Unit Tests

**Requirements**:
1. **MUST** test all RPC methods
2. **MUST** test error conditions
3. **MUST** test configuration parsing
4. **MUST** achieve >80% code coverage

**Example**:
```go
func TestProvider_Init(t *testing.T) {
    tests := []struct {
        name    string
        config  map[string]interface{}
        wantErr codes.Code
    }{
        {
            name:    "valid config",
            config:  map[string]interface{}{"directory": "./data"},
            wantErr: codes.OK,
        },
        {
            name:    "missing directory",
            config:  map[string]interface{}{},
            wantErr: codes.InvalidArgument,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            p := NewProvider()
            config, _ := structpb.NewStruct(tt.config)
            req := &providerv1.InitRequest{
                Alias:  "test",
                Config: config,
            }
            
            _, err := p.Init(context.Background(), req)
            
            if tt.wantErr == codes.OK {
                assert.NoError(t, err)
            } else {
                assert.Error(t, err)
                st, ok := status.FromError(err)
                assert.True(t, ok)
                assert.Equal(t, tt.wantErr, st.Code())
            }
        })
    }
}
```

### Integration Tests

**Requirements**:
1. **MUST** test against live gRPC server
2. **MUST** test full RPC lifecycle (Init → Fetch → Shutdown)
3. **SHOULD** test concurrent Fetch calls
4. **SHOULD** test with example `.csl` configurations

### Contract Tests

**Requirements**:
1. **MUST** verify compliance with protocol buffer contract
2. **MUST** verify all required fields are populated
3. **MUST** verify error codes match specifications

## References

### Protocol Definition

- **Protocol Buffers**: `/libs/provider-proto/proto/nomos/provider/v1/provider.proto`
- **Go Generated Code**: `github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1`

### Documentation

- [Provider Authoring Guide](./provider-authoring-guide.md) - Practical implementation guide
- [External Providers Architecture](../architecture/nomos-external-providers-feature-breakdown.md) - Architecture decisions
- [Terraform Providers Overview](./terraform-providers-overview.md) - Comparison with Terraform

### External Resources

- [Protocol Buffers](https://protobuf.dev/) - Google's data serialization
- [gRPC Go](https://grpc.io/docs/languages/go/) - gRPC for Go
- [gRPC Status Codes](https://grpc.io/docs/guides/status-codes/) - Error code semantics
- [Semantic Versioning](https://semver.org/) - Version numbering

---

**Document Version**: 1.0.0  
**Last Updated**: 2026-02-01  
**Status**: Authoritative
