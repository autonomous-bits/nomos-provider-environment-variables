# Quickstart: Environment Variables Provider

**Provider Type**: `environment-variables`  
**Version**: 1.0.0  
**Status**: Draft

## What is the Environment Variables Provider?

The environment variables provider enables you to pull configuration values from your system's environment variables into Nomos CSL files. Instead of hardcoding sensitive data like API keys or database URLs, you can reference them from environment variables that are set in your shell, CI/CD pipelines, or deployment environments.

## When to Use This Provider

✅ **Good use cases**:
- Injecting secrets (API keys, passwords, tokens)
- Environment-specific configuration (dev vs prod database URLs)
- CI/CD integration (build numbers, deployment targets)
- Docker/Kubernetes deployments (container environment variables)
- Twelve-Factor App configuration

❌ **Not recommended for**:
- Large configuration files (use `file` provider instead)
- Structured hierarchical configs (use `file` provider with YAML/JSON)
- Frequently changing values (environment variables are read-only during provider lifetime)

## Installation

The environment variables provider is distributed as a prebuilt binary for multiple platforms.

### Option 1: Install via Nomos CLI (Recommended)

```bash
# The Nomos CLI will automatically download the provider
# when it encounters a source declaration in your .csl files
nomos build
```

The CLI downloads providers to `.nomos/providers/` in your workspace.

### Option 2: Manual Download

```bash
# Download from GitHub Releases
VERSION=1.0.0
OS=darwin  # or linux, windows
ARCH=arm64 # or amd64

curl -L -o nomos-provider-environment-variables \
  https://github.com/autonomous-bits/nomos-provider-environment-variables/releases/download/v${VERSION}/nomos-provider-environment-variables-${VERSION}-${OS}-${ARCH}

# Verify checksum
curl -L -O \
  https://github.com/autonomous-bits/nomos-provider-environment-variables/releases/download/v${VERSION}/SHA256SUMS

sha256sum -c SHA256SUMS --ignore-missing

# Install to Nomos providers directory
mkdir -p .nomos/providers/environment-variables/${VERSION}/${OS}-${ARCH}
mv nomos-provider-environment-variables .nomos/providers/environment-variables/${VERSION}/${OS}-${ARCH}/provider
chmod +x .nomos/providers/environment-variables/${VERSION}/${OS}-${ARCH}/provider
```

## 5-Minute Tutorial

### Step 1: Set Environment Variables

```bash
export DATABASE_URL="postgres://localhost:5432/mydb"
export API_KEY="secret-key-12345"
export DEBUG="true"
export MAX_CONNECTIONS="100"
```

### Step 2: Create a CSL File

Create `config.csl`:

```csl
// Declare the environment variables provider
source environment-variables as env {
  version = "1.0.0"
}

// Import values from environment variables
config = {
  database_url = import env["DATABASE_URL"]
  api_key = import env["API_KEY"]
  debug = import env["DEBUG"]
  max_connections = import env["MAX_CONNECTIONS"]
}
```

### Step 3: Compile

```bash
nomos build -f config.csl
```

### Step 4: Verify Output

Check the compiled output:

```json
{
  "config": {
    "database_url": "postgres://localhost:5432/mydb",
    "api_key": "secret-key-12345",
    "debug": true,
    "max_connections": 100
  }
}
```

Note that values are automatically converted to appropriate types (boolean for `"true"`, number for `"100"`).

## Common Patterns

### Pattern 1: Direct Variable Access

Most straightforward approach - reference environment variables by their exact name.

**Environment**:
```bash
export DATABASE_HOST="localhost"
export DATABASE_PORT="5432"
```

**CSL**:
```csl
source environment-variables as env {
  version = "1.0.0"
}

database = {
  host = import env["DATABASE_HOST"]
  port = import env["DATABASE_PORT"]
}
```

---

### Pattern 2: Hierarchical Path Mapping

Use hierarchical paths in CSL that map to flat environment variable names.

**Environment**:
```bash
export DATABASE_HOST="localhost"
export DATABASE_PORT="5432"
export REDIS_HOST="cache.example.com"
```

**CSL**:
```csl
source environment-variables as env {
  version = "1.0.0"
  config = {
    separator = "_"
    case_transform = "upper"
  }
}

// env["database"]["host"] → DATABASE_HOST
database = {
  host = import env["database"]["host"]
  port = import env["database"]["port"]
}

// env["redis"]["host"] → REDIS_HOST
redis = {
  host = import env["redis"]["host"]
}
```

**How it works**: Path `["database"]["host"]` is transformed to:
1. Join with separator `_`: `"database_host"`
2. Apply uppercase: `"DATABASE_HOST"`
3. Fetch from environment: `os.Getenv("DATABASE_HOST")`

---

### Pattern 3: Prefix-Based Namespacing

Scope the provider to only expose variables with a specific prefix.

**Environment**:
```bash
export MYAPP_DATABASE_URL="postgres://localhost"
export MYAPP_API_KEY="secret123"
export SYSTEM_PATH="/usr/bin"  # Not exposed
```

**CSL**:
```csl
source environment-variables as env {
  version = "1.0.0"
  config = {
    prefix = "MYAPP_"
    prefix_mode = "prepend"
  }
}

// env["database"]["url"] → MYAPP_DATABASE_URL
config = {
  database_url = import env["database"]["url"]
  api_key = import env["api"]["key"]
}

// This would fail: SYSTEM_PATH doesn't have prefix
// system_path = import env["SYSTEM_PATH"]
```

**Benefits**:
- Prevents accidental access to system variables
- Namespace isolation for multi-app environments
- Cleaner CSL paths (prefix is implicit)

---

### Pattern 4: Required Variables Validation

Fail fast if critical environment variables are missing.

**Environment**:
```bash
export API_KEY="secret123"
# DATABASE_URL is missing!
```

**CSL**:
```csl
source environment-variables as env {
  version = "1.0.0"
  config = {
    required_variables = ["API_KEY", "DATABASE_URL", "SECRET_KEY"]
  }
}

// Provider initialization will fail before any imports are evaluated
config = {
  api_key = import env["API_KEY"]
}
```

**Error Message**:
```
Error: Provider initialization failed
required environment variables missing: DATABASE_URL, SECRET_KEY
```

**When to use**: Production deployments where missing config should prevent startup.

---

### Pattern 5: Type Conversion

Automatically convert string values to numbers, booleans, and JSON objects.

**Environment**:
```bash
export PORT="8080"
export ENABLE_DEBUG="true"
export RETRY_CONFIG='{"max_retries":3,"backoff_ms":100}'
```

**CSL**:
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
  retry = import env["RETRY_CONFIG"]           # → {max_retries: 3, backoff_ms: 100} (object)
}
```

**Type Detection**:
- **Numeric**: `"123"` → `123`, `"3.14"` → `3.14`
- **Boolean**: `"true"`, `"false"`, `"yes"`, `"no"` (case-insensitive)
- **JSON**: Values starting with `{` or `[` are parsed as JSON

**Disable type conversion** if you need all values as strings:
```csl
config = {
  enable_type_conversion = false
  enable_json_parsing = false
}
```

---

### Pattern 6: Multi-Environment Setup

Use different prefixes for different environments.

**Development**:
```bash
export DEV_DATABASE_URL="postgres://localhost/dev"
export DEV_API_BASE="http://localhost:3000"
```

**Production**:
```bash
export PROD_DATABASE_URL="postgres://prod-db.example.com/prod"
export PROD_API_BASE="https://api.example.com"
```

**CSL**:
```csl
// Development configuration
source environment-variables as dev_env {
  version = "1.0.0"
  config = {
    prefix = "DEV_"
    prefix_mode = "prepend"
  }
}

// Production configuration
source environment-variables as prod_env {
  version = "1.0.0"
  config = {
    prefix = "PROD_"
    prefix_mode = "prepend"
  }
}

// Use the appropriate environment
use_dev = import dev_env["DATABASE_URL"]  # → DEV_DATABASE_URL
use_prod = import prod_env["DATABASE_URL"] # → PROD_DATABASE_URL
```

## Configuration Reference

### Quick Reference Table

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `separator` | string | `"_"` | Character to join path segments |
| `case_transform` | string | `"upper"` | Case conversion: `"upper"`, `"lower"`, `"preserve"` |
| `prefix` | string | `""` | Prefix for filtering/prepending variables |
| `prefix_mode` | string | `"prepend"` | Prefix behavior: `"prepend"` or `"filter_only"` |
| `required_variables` | array | `[]` | List of variables that must exist |
| `enable_type_conversion` | boolean | `true` | Auto-convert to numbers/booleans |
| `enable_json_parsing` | boolean | `true` | Parse JSON-formatted values |

### Minimal Configuration

```csl
source environment-variables as env {
  version = "1.0.0"
}
```

All defaults apply. Suitable for most use cases.

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

### Error: "path[1] cannot be empty string"

**Symptom**:
```
Error: path[1] cannot be empty string
```

**Cause**: You used an empty segment in the path, like `env[""]["key"]` or `env["key"][""]`.

**Solution**:
- Use non-empty path segments: `env["valid"]["key"]`
- For single-segment paths, use: `env["KEY"]`

---

### Error: "failed to parse JSON value"

**Symptom**:
```
Error: failed to parse JSON value for CONFIG: invalid character '}' looking for beginning of object key string
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
1. Enable type conversion (it's on by default):
   ```csl
   config = {
     enable_type_conversion = true
   }
   ```
2. Or manually convert in CSL if you prefer explicit control

---

### Case sensitivity issues on Windows vs Unix

**Symptom**: Import works on Windows but fails on Linux (or vice versa).

**Cause**: Environment variable names are case-sensitive on Unix/Linux but case-insensitive on Windows.

**Solution**:
- **Best practice**: Use uppercase variable names for cross-platform compatibility
- On Unix, ensure exact case match: `PATH` ≠ `path`
- On Windows, `PATH` and `path` refer to the same variable

---

### Provider not found by Nomos CLI

**Symptom**:
```
Error: provider not found: environment-variables version 1.0.0
```

**Cause**: The provider binary is not installed or not in the expected location.

**Solution**:
1. The CLI should auto-download on first use
2. Check installation: `ls .nomos/providers/environment-variables/1.0.0/`
3. Manual installation: See "Installation" section above
4. Verify binary has execute permissions: `chmod +x .nomos/providers/...`

## Advanced Topics

### Custom Separator for Dot Notation

If your environment uses dots in variable names:

**Environment**:
```bash
export database.host="localhost"
export database.port="5432"
```

**CSL**:
```csl
source environment-variables as env {
  version = "1.0.0"
  config = {
    separator = "."
    case_transform = "preserve"
  }
}

database = {
  host = import env["database"]["host"]  # → database.host
}
```

### Preserving Original Case

For case-sensitive applications:

**Environment**:
```bash
export MyAppKey="value123"
export myAppUrl="https://example.com"
```

**CSL**:
```csl
source environment-variables as env {
  version = "1.0.0"
  config = {
    case_transform = "preserve"
  }
}

config = {
  key = import env["MyAppKey"]    # Exact case match
  url = import env["myAppUrl"]
}
```

### Filter-Only Prefix Mode

When you want filtering but explicit variable names:

**Environment**:
```bash
export APP_DATABASE_URL="postgres://localhost"
export SYSTEM_PATH="/usr/bin"
```

**CSL**:
```csl
source environment-variables as env {
  version = "1.0.0"
  config = {
    prefix = "APP_"
    prefix_mode = "filter_only"
  }
}

config = {
  # Must include prefix explicitly in path
  database_url = import env["APP_DATABASE_URL"]
  
  # This would fail - SYSTEM_PATH doesn't match prefix
  # system_path = import env["SYSTEM_PATH"]
}
```

## Next Steps

- **Learn More**: Read `docs/provider-development-standards.md` for protocol details
- **Examples**: See `tests/integration/` for comprehensive usage examples
- **API Reference**: See `specs/001-environment-variables/contracts/` for gRPC contract
- **Report Issues**: https://github.com/autonomous-bits/nomos-provider-environment-variables/issues

## FAQ

**Q: Can I change environment variables after the provider starts?**  
A: No. Environment variables are read once and cached. Restart the Nomos build to pick up changes.

**Q: What's the maximum size for an environment variable value?**  
A: 1MB. Larger values will return an error.

**Q: Are environment variables validated or sanitized?**  
A: No. The provider returns raw values from the environment. Validate/sanitize in your application.

**Q: Can I use this provider in Docker/Kubernetes?**  
A: Yes! It works seamlessly with container environment variables.

**Q: Does this work on Windows?**  
A: Yes. Note that Windows environment variables are case-insensitive (unlike Unix/Linux).

**Q: How do I handle secrets securely?**  
A: Use your platform's secret management (AWS Secrets Manager, Kubernetes Secrets, etc.) to inject environment variables. Never commit secrets to source control.
