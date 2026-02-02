# Feature Specification: Environment Variables Provider

**Feature Branch**: `001-environment-variables`  
**Created**: 2026-02-01  
**Status**: Draft  
**Input**: User description: "Create a new Nomos provider that enables the Nomos compiler to pull configuration values from environment variables. Users will define environment variables in their system, then reference them in CSL files by setting up this provider as a source with an alias."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Basic Environment Variable Fetching (Priority: P1)

A developer needs to inject runtime-specific configuration values (like API keys, database URLs, or feature flags) into their Nomos configuration without hardcoding them in CSL files. They set environment variables in their shell or CI/CD system, declare an environment-variables provider source in their CSL file with an alias, and import values using the alias.

**Why this priority**: This is the core MVP functionality that delivers immediate value. Without this, the provider has no purpose. Developers can start using environment variables for configuration immediately.

**Independent Test**: Can be fully tested by setting environment variables (e.g., `export DATABASE_URL=postgres://localhost`), creating a CSL file with an env provider source, importing a single variable, and verifying the compiler retrieves the correct value.

**Acceptance Scenarios**:

1. **Given** environment variable `API_KEY=secret123` is set, and a CSL file declares `source environment-variables as env { version = "1.0.0" }`, **When** the user imports `env["API_KEY"]`, **Then** the compiler receives the value `"secret123"`.

2. **Given** multiple environment variables are set (`DB_HOST=localhost`, `DB_PORT=5432`), **When** the user imports `env["DB_HOST"]` and `env["DB_PORT"]` in separate references, **Then** each import resolves to the correct corresponding value.

3. **Given** an environment variable is not set, **When** the user attempts to import it via `env["MISSING_VAR"]`, **Then** the provider returns a "not found" error and the compilation fails with a clear message.

---

### User Story 2 - Path-Based Variable Name Mapping (Priority: P2)

A developer wants to organize configuration hierarchically in CSL (e.g., `database.host`, `database.port`) but environment variables are typically flat with naming conventions like `DATABASE_HOST`, `DATABASE_PORT`. They need the provider to translate hierarchical paths to flat environment variable names using a configurable separator and case conversion.

**Why this priority**: This enables idiomatic CSL usage while respecting environment variable naming conventions. It's the next logical step after basic fetching.

**Independent Test**: Can be tested by setting environment variables with conventional naming (e.g., `DATABASE_HOST=localhost`), configuring the provider with a separator and case convention, importing using a path like `env["database"]["host"]`, and verifying the provider correctly resolves it to `DATABASE_HOST`.

**Acceptance Scenarios**:

1. **Given** environment variable `DATABASE_HOST=localhost` is set, and provider config specifies uppercase conversion with underscore separator, **When** the user imports `env["database"]["host"]`, **Then** the provider resolves it to `DATABASE_HOST` and returns `"localhost"`.

2. **Given** environment variable `app_api_timeout=30` is set, and provider config specifies lowercase conversion with underscore separator, **When** the user imports `env["app"]["api"]["timeout"]`, **Then** the provider resolves it to `app_api_timeout` and returns `"30"`.

3. **Given** environment variable `DB_USER=admin` is set, **When** the user imports `env["db"]["user"]` with default uppercase-underscore convention, **Then** the provider returns `"admin"`.

---

### User Story 3 - Prefix-Based Filtering (Priority: P3)

A developer working in an environment with hundreds of environment variables wants to scope the provider to only expose variables with a specific prefix (e.g., `MYAPP_`). This prevents accidental access to system or unrelated environment variables and provides namespace isolation.

**Why this priority**: This adds organizational and security value but isn't essential for basic functionality. It's useful for production deployments with many environment variables.

**Independent Test**: Can be tested by setting multiple environment variables (e.g., `MYAPP_KEY=value1`, `OTHER_KEY=value2`), configuring the provider with prefix `MYAPP_`, attempting to import both variables, and verifying only the prefixed variable is accessible.

**Acceptance Scenarios**:

1. **Given** environment variables `MYAPP_DB_HOST=localhost` and `SYSTEM_PATH=/usr/bin` are set, and provider config specifies prefix `MYAPP_`, **When** the user imports `env["MYAPP_DB_HOST"]`, **Then** the value `"localhost"` is returned.

2. **Given** the same environment variables and prefix configuration, **When** the user attempts to import `env["SYSTEM_PATH"]`, **Then** the provider returns "not found" error.

3. **Given** prefix `MYAPP_` is configured, and variable `MYAPP_VERSION=2.0` is set, **When** the user imports using the path `env["VERSION"]` (without prefix in the path), **Then** the provider automatically prepends `MYAPP_` to resolve to `MYAPP_VERSION` and returns `"2.0"`.

---

### User Story 4 - Type Conversion from String Values (Priority: P4)

Environment variables are always strings, but developers need to use them as numbers, booleans, or structured data in their configuration. The provider automatically detects and converts string values to appropriate types (number, boolean) based on content patterns, and optionally parses JSON-formatted strings into structured maps or lists.

**Why this priority**: This enhances developer experience by eliminating manual type conversion in CSL. It's not critical for MVP but makes the provider much more ergonomic.

**Independent Test**: Can be tested by setting environment variables with different type patterns (e.g., `PORT=8080`, `DEBUG=true`, `CONFIG={"key":"value"}`), importing them, and verifying the returned data types match expectations (number, boolean, map).

**Acceptance Scenarios**:

1. **Given** environment variable `PORT=8080` is set, **When** the user imports `env["PORT"]`, **Then** the provider returns the numeric value `8080` (not the string `"8080"`).

2. **Given** environment variable `ENABLE_FEATURE=true` is set, **When** the user imports `env["ENABLE_FEATURE"]`, **Then** the provider returns the boolean value `true`.

3. **Given** environment variable `CONFIG={"timeout":30,"retries":3}` is set, **When** the user imports `env["CONFIG"]`, **Then** the provider parses the JSON and returns a structured map `{timeout: 30, retries: 3}`.

4. **Given** environment variable `EMPTY_VAR=` is set (empty string), **When** the user imports `env["EMPTY_VAR"]`, **Then** the provider returns an empty string `""` (not null).

---

### User Story 5 - Required Variable Validation (Priority: P5)

A developer wants to fail fast during compilation if critical environment variables are missing, rather than encountering errors at runtime. They configure a list of required variables in the provider config, and the provider validates their presence during initialization.

**Why this priority**: This improves error handling and developer feedback but can be worked around by checking for errors at import time. It's nice-to-have for production-ready deployments.

**Independent Test**: Can be tested by configuring the provider with a list of required variables, intentionally omitting one, and verifying the provider initialization fails immediately with a clear error message listing the missing variable.

**Acceptance Scenarios**:

1. **Given** provider config specifies required variables `["API_KEY", "DATABASE_URL"]`, and both are set, **When** the provider initializes, **Then** initialization succeeds without errors.

2. **Given** provider config specifies required variable `API_KEY`, but it is not set, **When** the provider initializes, **Then** initialization fails with error "required environment variable missing: API_KEY".

3. **Given** provider config specifies required variables `["VAR1", "VAR2"]`, and neither is set, **When** the provider initializes, **Then** initialization fails with error listing all missing variables: "required environment variables missing: VAR1, VAR2".

---

### Edge Cases

- **Empty environment variable values**: What happens when an environment variable is set but empty (e.g., `VAR=`)? Should return empty string, not null or error.
- **Variable name case sensitivity**: Environment variable names follow platform-native behavior: case-sensitive on Unix/Linux (where `PATH` and `path` are different variables) and case-insensitive on Windows (where they are the same). CSL files using different cases for the same variable name may behave differently across platforms.
- **Special characters in variable names**: What happens with environment variables containing dots, dashes, or other special characters (e.g., `MY.VAR.NAME`)? Should be supported as-is for direct access.
- **Circular reference in JSON values**: If a JSON-formatted environment variable contains circular references, the provider should return a JSON parsing error with details.
- **Very large environment variable values**: What happens when an environment variable contains megabytes of data? Should impose a reasonable size limit (e.g., 1MB) and return an error if exceeded.
- **Unicode and multibyte characters**: Environment variables with non-ASCII characters should be supported if the system encoding supports them.
- **Path with empty segments**: What happens when a path contains empty strings (e.g., `env[""]["key"]`)? Should return "invalid path" error.
- **Concurrent fetches**: Multiple imports of the same variable should be safe and return consistent cached values, not re-read from the environment on every fetch.

## Requirements *(mandatory)*

### Functional Requirements

#### Core Provider Contract

- **FR-001**: Provider MUST implement the Nomos `nomos.provider.v1.ProviderService` gRPC contract, including all required RPCs: `Init`, `Fetch`, `Info`, `Health`, and `Shutdown`.

- **FR-002**: Provider MUST communicate its listening port to the Nomos CLI by printing `PORT={port_number}` to stdout as the first line of output.

- **FR-003**: Provider MUST use the reserved field `type` with the value `"environment-variables"` in the `Info` RPC response.

- **FR-004**: Provider MUST accept and store the `alias` from `InitRequest` for logging and diagnostics, but MUST NOT use it as a configuration key.

- **FR-005**: Provider MUST follow semantic versioning for the `version` field returned in the `Info` RPC response.

#### Environment Variable Fetching

- **FR-006**: Provider MUST fetch environment variable values from the process environment where the provider subprocess is running.

- **FR-006a**: Provider MUST use platform-native case sensitivity for environment variable lookups (case-sensitive on Unix/Linux, case-insensitive on Windows).

- **FR-007**: Provider MUST return a gRPC `NotFound` status code when a requested environment variable does not exist.

- **FR-008**: Provider MUST support empty string values for environment variables and distinguish them from unset variables.

- **FR-009**: Provider MUST cache environment variable values after the first read and return consistent cached values for subsequent `Fetch` calls (environment variables are read-only during provider lifetime).

#### Path Resolution

- **FR-010**: Provider MUST support direct environment variable access using a single-segment path (e.g., path `["API_KEY"]` maps to environment variable `API_KEY`).

- **FR-011**: Provider MUST support path-to-variable-name mapping for multi-segment paths, converting hierarchical paths to flat environment variable names using a configurable separator pattern (default: uppercase with underscore, e.g., `["database"]["host"]` → `DATABASE_HOST`).

- **FR-012**: Provider configuration MUST allow specifying the path-to-name conversion strategy with options for case transformation (uppercase, lowercase, preserve) and separator character (underscore, dash, dot).

#### Prefix Filtering

- **FR-013**: Provider configuration MUST allow specifying an optional prefix string to filter environment variables (e.g., `prefix: "MYAPP_"`).

- **FR-014**: When a prefix is configured, provider MUST only expose environment variables that start with the specified prefix.

- **FR-015**: Provider configuration MUST allow specifying a `prefix_mode` option with values `"prepend"` (default) or `"filter_only"` to control prefix behavior. In `"prepend"` mode, the prefix is automatically prepended to the transformed variable name. In `"filter_only"` mode, the prefix is only used for filtering and must be included in the path by the user.

#### Type Conversion

- **FR-016**: Provider MUST automatically convert environment variable string values to appropriate types using this precedence order: attempt numeric conversion first, then boolean conversion, then return as string. Numeric strings (integers and floats) convert to numbers. Boolean strings (`true`, `false`, `yes`, `no`) convert to booleans only if numeric conversion fails. JSON parsing (FR-017) is evaluated independently based on value prefix, not within this type conversion precedence.

- **FR-017**: Provider MUST support parsing JSON-formatted environment variable values into structured data (maps and lists) when the value starts with `{` or `[`. JSON parsing takes precedence over simple type conversion when the prefix condition is met.

- **FR-018**: Provider MUST return the original string value if simple type conversion (number/boolean) fails. Provider MUST return a gRPC `InvalidArgument` error with parsing details if JSON parsing fails for a value starting with `{` or `[`.

#### Validation and Error Handling

- **FR-019**: Provider configuration MAY allow specifying a list of required environment variables that must be present.

- **FR-020**: When required variables are configured, provider MUST validate their presence during the `Init` RPC BEFORE accepting the configuration and BEFORE setting the provider state to initialized. If any required variables are missing, the provider MUST return a gRPC `InvalidArgument` status code including the list of missing variable names in the error message and MUST remain in the uninitialized state. Required variables MUST be specified using their original environment variable names as they appear in the system environment (e.g., `["DATABASE_HOST", "API_KEY"]`), not transformed path representations.

- **FR-021**: Provider MUST return a gRPC `InvalidArgument` status code for malformed paths (e.g., paths with empty segments).

- **FR-022**: Provider MUST return a gRPC `FailedPrecondition` status code if `Fetch` is called before `Init`.

- **FR-023**: Provider MUST include actionable error messages that specify which environment variable or path caused the error.

#### Configuration Schema

- **FR-024**: Provider MUST accept configuration via the `InitRequest.config` field as a structured map.

- **FR-025**: Provider configuration MUST NOT use reserved field names: `alias`, `type`, or `version`.

- **FR-026**: Provider configuration fields MUST include:
  - `separator` (optional, default: `"_"`): Character to use when joining path segments
  - `case_transform` (optional, default: `"upper"`): One of `"upper"`, `"lower"`, `"preserve"`
  - `prefix` (optional, default: `""`): Filter prefix for environment variables
  - `prefix_mode` (optional, default: `"prepend"`): One of `"prepend"` or `"filter_only"` to control whether prefix is automatically added to transformed variable names
  - `required_variables` (optional, default: `[]`): List of environment variable names that must be present
  - `enable_type_conversion` (optional, default: `true`): Whether to automatically convert types
  - `enable_json_parsing` (optional, default: `true`): Whether to parse JSON-formatted values

#### Performance and Safety

- **FR-027**: Provider MUST be thread-safe and support concurrent `Fetch` calls without data races.

- **FR-028**: Provider MUST complete `Shutdown` RPC within 5 seconds and clean up any resources.

- **FR-029**: Provider MUST impose a maximum size limit of 1MB for environment variable values and return a gRPC `InvalidArgument` error if exceeded.

- **FR-030**: Provider MUST limit JSON parsing depth to a maximum of 100 nested levels and return a gRPC `InvalidArgument` error if exceeded to prevent stack overflow from circular or deeply nested structures.

- **FR-031**: Provider MUST cache environment variable values in memory after the first successful fetch. Cached values MUST remain consistent across all subsequent `Fetch` calls until the provider is shut down via the `Shutdown` RPC. Cache MUST be cleared on `Shutdown` and MUST NOT persist across `Init` calls if the provider is reinitialized.

### Key Entities

- **Environment Variable**: A name-value pair defined in the operating system process environment. The name is a case-sensitive string, and the value is always a string. Represents external configuration retrieved by the provider.

- **Path**: An ordered array of string segments representing a hierarchical location in the configuration namespace (e.g., `["database", "prod", "host"]`). Used by the Nomos compiler to reference configuration values and mapped by the provider to flat environment variable names.

- **Provider Configuration**: A structured map passed during initialization that controls provider behavior. Includes options for path resolution strategy, prefix filtering, type conversion, and validation rules.

- **Cached Value**: An environment variable value stored in provider memory after the first fetch. Ensures consistent values across multiple fetches and avoids repeated environment reads.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Developers can successfully import environment variable values into CSL files within 30 seconds of setting up the provider source declaration.

- **SC-002**: Provider responds to fetch requests for environment variables in under 10 milliseconds (cached values).

- **SC-003**: Provider initialization fails with clear error messages within 2 seconds when required environment variables are missing, eliminating runtime configuration errors.

- **SC-004**: Developers can use hierarchical path syntax (e.g., `env["database"]["host"]`) that maps to conventional environment variable names (e.g., `DATABASE_HOST`) without manual string manipulation in 100% of use cases.

- **SC-005**: Provider correctly converts numeric and boolean environment variable strings to appropriate types in 95% of common patterns (integers, floats, true/false, yes/no, 1/0).

- **SC-006**: Provider handles concurrent fetches of the same environment variable safely with zero data races or inconsistencies across 10,000 concurrent requests.

- **SC-007**: Provider initialization and all fetch operations complete successfully on all supported platforms (macOS Intel/ARM, Linux x86_64/ARM64, Windows x86_64) with identical behavior.

## Clarifications

### Session 2026-02-01

- Q: When using prefix filtering with path-based variable mapping, how should the prefix interact with the path transformation? → A: Make it configurable via a new config option `prefix_mode: "prepend"` or `"filter_only"`
- Q: How should the provider handle environment variable name case sensitivity across platforms? → A: Use platform-native behavior (case-sensitive on Unix, case-insensitive on Windows) and document platform differences
- Q: When type conversion is enabled and a value could match multiple type patterns, what should the precedence order be? → A: Number → Boolean → String (numerical values take priority; "1" becomes number 1, not boolean true)
- Q: When JSON parsing fails for a value that starts with `{` or `[`, what should the provider do? → A: Return an error with JSON parsing failure details (helps users catch malformed JSON early)
- Q: When specifying required variables in the configuration, should they reference the original environment variable names or the transformed path representations? → A: Use original environment variable names (e.g., `required_variables: ["DATABASE_HOST", "API_KEY"]`)
