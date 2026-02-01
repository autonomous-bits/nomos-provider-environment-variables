# Nomos Provider: Environment Variables

A high-performance Nomos provider that enables configuration values to be pulled from environment variables.

## Overview

The environment variables provider implements the Nomos `ProviderService` gRPC contract, running as a subprocess to fetch configuration values from the system environment. It supports:

- **Direct variable access**: Fetch environment variables by name
- **Hierarchical path mapping**: Transform paths like `["database"]["host"]` to `DATABASE_HOST`
- **Prefix filtering**: Scope to variables with a specific prefix (e.g., `MYAPP_`)
- **Automatic type conversion**: Convert strings to numbers, booleans, and JSON objects
- **Required variable validation**: Fail fast if critical variables are missing
- **High performance**: Sub-millisecond fetch latency (57.6µs average)
- **Thread-safe**: Handles 10,000+ concurrent requests safely

## Features

✨ **Key Features**:
- Integrate with Twelve-Factor App methodology
- Seamless Docker/Kubernetes integration
- Cross-platform support (macOS, Linux, Windows)
- Type-safe value conversion
- Production-ready with >80% test coverage
- Zero data races (validated with Go race detector)

## Installation

### Option 1: Via Nomos CLI (Recommended)

The Nomos CLI automatically downloads the provider when it encounters a source declaration:

```bash
# The provider will be auto-downloaded on first use
nomos build
```

### Option 2: Manual Download

Download the latest binary for your platform from [GitHub Releases](https://github.com/autonomous-bits/nomos-provider-environment-variables/releases):

```bash
# macOS (Apple Silicon)
curl -L -o nomos-provider-environment-variables \
  https://github.com/autonomous-bits/nomos-provider-environment-variables/releases/latest/download/nomos-provider-environment-variables-darwin-arm64

# macOS (Intel)
curl -L -o nomos-provider-environment-variables \
  https://github.com/autonomous-bits/nomos-provider-environment-variables/releases/latest/download/nomos-provider-environment-variables-darwin-amd64

# Linux (x86_64)
curl -L -o nomos-provider-environment-variables \
  https://github.com/autonomous-bits/nomos-provider-environment-variables/releases/latest/download/nomos-provider-environment-variables-linux-amd64

# Linux (ARM64)
curl -L -o nomos-provider-environment-variables \
  https://github.com/autonomous-bits/nomos-provider-environment-variables/releases/latest/download/nomos-provider-environment-variables-linux-arm64

# Windows (x86_64)
curl -L -o nomos-provider-environment-variables.exe \
  https://github.com/autonomous-bits/nomos-provider-environment-variables/releases/latest/download/nomos-provider-environment-variables-windows-amd64.exe

# Make executable (Unix-like systems)
chmod +x nomos-provider-environment-variables
```

### Option 3: From Source

Requires Go 1.21 or later:

```bash
git clone https://github.com/autonomous-bits/nomos-provider-environment-variables.git
cd nomos-provider-environment-variables
make build
```

## Usage Examples

### User Story 1: Basic Environment Variable Fetching

Inject runtime-specific configuration values without hardcoding them.

**Environment Setup**:
```bash
export DATABASE_URL="postgres://localhost:5432/mydb"
export API_KEY="secret-key-12345"
export DEBUG="true"
```

**CSL Configuration**:
```csl
source environment-variables as env {
  version = "1.0.0"
}

config = {
  database_url = import env["DATABASE_URL"]
  api_key = import env["API_KEY"]
  debug = import env["DEBUG"]
}
```

**Output** (types auto-converted):
```json
{
  "config": {
    "database_url": "postgres://localhost:5432/mydb",
    "api_key": "secret-key-12345",
    "debug": true
  }
}
```

---

### User Story 2: Hierarchical Path Mapping

Organize configuration hierarchically in CSL while respecting flat environment variable conventions.

**Environment Setup**:
```bash
export DATABASE_HOST="localhost"
export DATABASE_PORT="5432"
export DATABASE_USER="admin"
export REDIS_HOST="cache.example.com"
```

**CSL Configuration**:
```csl
source environment-variables as env {
  version = "1.0.0"
  config = {
    separator = "_"
    case_transform = "upper"
  }
}

# Paths are transformed: ["database"]["host"] → DATABASE_HOST
database = {
  host = import env["database"]["host"]
  port = import env["database"]["port"]
  user = import env["database"]["user"]
}

redis = {
  host = import env["redis"]["host"]
}
```

**Output**:
```json
{
  "database": {
    "host": "localhost",
    "port": 5432,
    "user": "admin"
  },
  "redis": {
    "host": "cache.example.com"
  }
}
```

---

### User Story 3: Prefix-Based Filtering

Scope the provider to specific namespaced variables for security and organization.

**Environment Setup**:
```bash
export MYAPP_DATABASE_URL="postgres://localhost"
export MYAPP_API_KEY="secret123"
export SYSTEM_PATH="/usr/bin"  # Will not be accessible
```

**CSL Configuration**:
```csl
source environment-variables as env {
  version = "1.0.0"
  config = {
    prefix = "MYAPP_"
    prefix_mode = "prepend"
  }
}

# Prefix is automatically prepended
config = {
  database_url = import env["database"]["url"]  # → MYAPP_DATABASE_URL
  api_key = import env["api"]["key"]            # → MYAPP_API_KEY
}

# This would fail: SYSTEM_PATH doesn't have MYAPP_ prefix
# system_path = import env["SYSTEM_PATH"]
```

**Benefits**: Prevents accidental access to system variables, provides namespace isolation.

---

### User Story 4: Type Conversion

Automatically convert string environment variables to appropriate types.

**Environment Setup**:
```bash
export PORT="8080"
export ENABLE_DEBUG="true"
export MAX_RETRIES="3"
export TIMEOUT="30.5"
export RETRY_CONFIG='{"max_retries":3,"backoff_ms":100}'
```

**CSL Configuration**:
```csl
source environment-variables as env {
  version = "1.0.0"
  config = {
    enable_type_conversion = true
    enable_json_parsing = true
  }
}

config = {
  port = import env["PORT"]                    # → 8080 (number)
  debug = import env["ENABLE_DEBUG"]           # → true (boolean)
  max_retries = import env["MAX_RETRIES"]      # → 3 (number)
  timeout = import env["TIMEOUT"]              # → 30.5 (float)
  retry_config = import env["RETRY_CONFIG"]    # → {max_retries: 3, backoff_ms: 100} (object)
}
```

**Type Detection**:
- Numbers: `"123"` → `123`, `"3.14"` → `3.14`
- Booleans: `"true"`, `"false"`, `"yes"`, `"no"` (case-insensitive)
- JSON: Values starting with `{` or `[` are parsed as structured data

---

### User Story 5: Required Variable Validation

Fail fast during initialization if critical environment variables are missing.

**Environment Setup**:
```bash
export API_KEY="secret123"
# DATABASE_URL is intentionally missing
```

**CSL Configuration**:
```csl
source environment-variables as env {
  version = "1.0.0"
  config = {
    required_variables = ["API_KEY", "DATABASE_URL", "SECRET_KEY"]
  }
}

config = {
  api_key = import env["API_KEY"]
}
```

**Result**: Provider initialization fails immediately with clear error:
```
Error: Provider initialization failed
required environment variables missing: DATABASE_URL, SECRET_KEY
```

**Benefits**: Catch configuration errors at compile-time instead of runtime.

---

## Configuration Reference

### Complete Configuration Options

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `separator` | string | `"_"` | Character used to join path segments when resolving variable names |
| `case_transform` | string | `"upper"` | Case conversion for variable names: `"upper"`, `"lower"`, or `"preserve"` |
| `prefix` | string | `""` | Prefix for filtering or prepending to variable names |
| `prefix_mode` | string | `"prepend"` | Prefix behavior: `"prepend"` (auto-add prefix) or `"filter_only"` (explicit prefix required) |
| `required_variables` | array | `[]` | List of environment variables that must exist at initialization |
| `enable_type_conversion` | boolean | `true` | Automatically convert strings to numbers and booleans |
| `enable_json_parsing` | boolean | `true` | Parse JSON-formatted string values into structured data |

### Minimal Configuration

```csl
source environment-variables as env {
  version = "1.0.0"
}
```

Uses all default settings - suitable for most use cases.

### Full Configuration Example

```csl
source environment-variables as env {
  version = "1.0.0"
  config = {
    separator = "_"
    case_transform = "upper"
    prefix = "MYAPP_"
    prefix_mode = "prepend"
    required_variables = ["MYAPP_DATABASE_URL", "MYAPP_API_KEY"]
    enable_type_conversion = true
    enable_json_parsing = true
  }
}
```

## Performance Characteristics

Based on comprehensive benchmarks ([tests/integration/PERFORMANCE_BENCHMARKS.md](tests/integration/PERFORMANCE_BENCHMARKS.md)):

| Metric | Performance | Success Criteria | Status |
|--------|-------------|------------------|---------|
| **Fetch Latency** | 57.6µs (0.0576ms) | <10ms | ✅ **174x better** |
| **Concurrent Throughput** | 78,931 ops/s | 10,000 concurrent requests | ✅ **Passed** |
| **Data Race Safety** | Zero races detected | Zero races | ✅ **Validated** |
| **Value Consistency** | 100% consistent | 100% consistent | ✅ **Passed** |

**Hardware**: Apple M2, Go 1.22, macOS

### Performance by Operation

- String fetch: ~68µs
- Number fetch: ~68µs
- Boolean fetch: ~66µs
- JSON parsing: ~72µs
- Path resolution: ~64µs

All operations complete well under 10ms target.

## Build Instructions

### Prerequisites

- Go 1.21 or later
- make (optional, but recommended)
- golangci-lint (for linting)

### Basic Build

```bash
# Build for current platform
make build

# Or use go directly
go build -o nomos-provider-environment-variables ./cmd/provider
```

### Development Workflow

```bash
# Run all checks (lint + test + build)
make dev

# Individual commands
make lint           # Run linters
make test           # Run unit tests
make test-integration  # Run integration tests
make coverage       # Generate coverage report
```

### Cross-Platform Compilation

Build binaries for all supported platforms:

```bash
make cross-compile
```

This creates binaries in the `dist/` directory:
- `nomos-provider-environment-variables-<version>-darwin-amd64`
- `nomos-provider-environment-variables-<version>-darwin-arm64`
- `nomos-provider-environment-variables-<version>-linux-amd64`
- `nomos-provider-environment-variables-<version>-linux-arm64`
- `nomos-provider-environment-variables-<version>-windows-amd64.exe`

SHA256 checksums are automatically generated in `dist/SHA256SUMS`.

### Version Injection

The build automatically injects version information from Git tags:

```bash
# Build with version from git describe
make build

# Or manually specify version
VERSION=1.0.0 make build
```

## Testing

### Run All Tests

```bash
# Unit tests with race detection
make test

# Integration tests
make test-integration

# All tests with coverage
make coverage
```

### Test Coverage

Current coverage: **>80%**

View coverage report:
```bash
make coverage
open coverage.html
```

### Performance Benchmarks

```bash
# Run fetch response time benchmarks
go test -tags=integration -bench=BenchmarkFetchResponseTime -benchmem ./tests/integration/

# Run concurrent safety benchmarks
go test -tags=integration -bench=BenchmarkConcurrentFetches -benchmem ./tests/integration/

# Validate zero data races (critical!)
go test -tags=integration -bench=BenchmarkConcurrentFetches_DataRaceValidation -race ./tests/integration/
```

See [tests/integration/PERFORMANCE_BENCHMARKS.md](tests/integration/PERFORMANCE_BENCHMARKS.md) for detailed benchmark documentation.

### Quickstart Validation

Validate all examples from the quickstart guide:

```bash
./scripts/validate-quickstart.sh
```

This script validates all usage patterns documented in the quickstart.

## Troubleshooting

### Error: "environment variable not found"

**Symptom**:
```
Error: environment variable not found: DATABASE_URL
```

**Cause**: The environment variable is not set in the current shell/environment.

**Solution**:
1. Verify the variable is set: `echo $DATABASE_URL`
2. Set it if missing: `export DATABASE_URL="your-value"`
3. Check for typos in the variable name (case-sensitive on Unix/Linux)

---

### Error: "required environment variables missing"

**Symptom**:
```
Error: Provider initialization failed
required environment variables missing: API_KEY, SECRET_KEY
```

**Cause**: One or more required variables (specified in `required_variables`) are not set.

**Solution**:
1. Set all required variables: `export API_KEY="..." SECRET_KEY="..."`
2. Remove variables from `required_variables` if they're optional
3. Check the exact variable names (including any prefix)

---

### Error: "provider not found"

**Symptom**:
```
Error: provider not found: environment-variables version 1.0.0
```

**Cause**: The provider binary is not installed or not in the expected location.

**Solution**:
1. The Nomos CLI should auto-download on first use
2. Check installation: `ls .nomos/providers/environment-variables/1.0.0/`
3. Manual installation: See "Installation" section above
4. Verify binary has execute permissions: `chmod +x .nomos/providers/.../provider`

---

### Error: "failed to parse JSON value"

**Symptom**:
```
Error: failed to parse JSON value for CONFIG: invalid character '}' ...
```

**Cause**: The environment variable value starts with `{` or `[` but is malformed JSON.

**Solution**:
1. Fix the JSON syntax in the environment variable
2. Escape special characters if needed
3. Disable JSON parsing if you want the raw string:
   ```csl
   config = {
     enable_json_parsing = false
   }
   ```

---

### Values are strings instead of numbers/booleans

**Symptom**: `PORT="8080"` is imported as string `"8080"` instead of number `8080`.

**Cause**: Type conversion is disabled.

**Solution**:
Enable type conversion (it's on by default):
```csl
config = {
  enable_type_conversion = true
}
```

---

### Case sensitivity issues on Windows vs Unix

**Symptom**: Import works on Windows but fails on Linux (or vice versa).

**Cause**: Environment variable names are case-sensitive on Unix/Linux but case-insensitive on Windows.

**Solution**:
- **Best practice**: Use uppercase variable names for cross-platform compatibility
- On Unix: Ensure exact case match (`PATH` ≠ `path`)
- On Windows: `PATH` and `path` refer to the same variable

---

### Provider hangs or doesn't respond

**Symptom**: Provider starts but doesn't respond to gRPC requests.

**Cause**: Port conflict or network configuration issue.

**Solution**:
1. Check provider startup output for `PROVIDER_PORT=<port>`
2. Verify no other process is using that port: `lsof -i :<port>`
3. Check firewall settings allow localhost gRPC connections

---

## Documentation

- **[Quickstart Guide](specs/001-environment-variables/quickstart.md)** - 5-minute tutorial and common patterns
- **[Specification](specs/001-environment-variables/spec.md)** - Feature requirements and user scenarios
- **[Implementation Plan](specs/001-environment-variables/plan.md)** - Technical design and architecture
- **[Performance Benchmarks](tests/integration/PERFORMANCE_BENCHMARKS.md)** - Detailed performance analysis
- **[Provider Development Standards](docs/provider-development-standards.md)** - gRPC contract details

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines.

### Quick Start for Contributors

```bash
# Clone repository
git clone https://github.com/autonomous-bits/nomos-provider-environment-variables.git
cd nomos-provider-environment-variables

# Install development tools
make install-tools

# Run development workflow
make dev

# Submit PR with conventional commits
git commit -m "feat: add new feature"
```

## License

See [LICENSE](LICENSE) file for details.

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history following [Keep a Changelog](https://keepachangelog.com/) format.

## Support

- **Issues**: [GitHub Issues](https://github.com/autonomous-bits/nomos-provider-environment-variables/issues)
- **Discussions**: [GitHub Discussions](https://github.com/autonomous-bits/nomos-provider-environment-variables/discussions)
- **Security**: For security issues, see SECURITY.md