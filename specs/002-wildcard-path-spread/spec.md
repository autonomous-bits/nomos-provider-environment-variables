# Feature Specification: Wildcard Path Spread Operator

**Feature Branch**: `002-wildcard-path-spread`  
**Created**: 2026-03-07  
**Status**: Draft  
**Input**: User description: "I want the provider to be able to handle the spread operator (*) when a path is sent through the path could contain the `*` operator at the end of the path. When the spread operator is encountered at the root of the path it should iterate through all the variables and return them, if it is after the root it should only load the subset of environment variables and so on."

## Clarifications

### Session 2026-03-07

- Q: Should wildcard access have additional restrictions beyond the existing provider-level prefix, or should the prefix alone serve as the security scope boundary? → A: Wildcard access is governed solely by the existing provider-level prefix. No additional restriction inside the provider. Callers are responsible for configuring a restrictive prefix.
- Q: What case convention is used for returned key names in a spread result after the path prefix is stripped? → A: Preserve original env var casing — the returned key is the raw suffix from the original environment variable name with no transformation applied.
- Q: How is the wildcard collection surfaced to the caller in the gRPC protocol response? → A: Unary response with a map field — a single response message whose value field is a `map<string, Value>` (or equivalent structured value type).
- Q: What is the defined behaviour when two environment variables produce the same relative key after prefix stripping? → A: Behaviour is non-deterministic (one value survives, which one is undefined). This is considered operator misconfiguration and is documented rather than enforced at runtime.
- Q: What is the measurable performance target for wildcard queries (SC-002)? → A: Wildcard response time must be no worse than the time taken to perform the equivalent set of individual single-key lookups for the same variables (no superlinear overhead per result).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Root-Level Wildcard Retrieval (Priority: P1)

A developer wants to retrieve all environment variables at once by using the wildcard operator `*` at the root of the path. Instead of fetching variables one by one, they issue a single request with `env["*"]` and receive a collection of all available environment variables as key-value pairs.

**Why this priority**: This is the foundational capability of the spread operator. It delivers immediate value for use cases like debugging, introspection, or bulk configuration loading. It is the simplest form of the feature and establishes the behaviour contract for all other wildcard variants.

**Independent Test**: Can be fully tested by setting multiple environment variables (e.g., `API_KEY=secret`, `DB_HOST=localhost`), requesting `env["*"]`, and verifying the response contains all expected key-value pairs as a collection.

**Acceptance Scenarios**:

1. **Given** environment variables `API_KEY=secret` and `DB_HOST=localhost` are set, **When** the user requests `env["*"]`, **Then** the provider returns a collection containing all available environment variables including `API_KEY=secret` and `DB_HOST=localhost`.
2. **Given** no environment variables are set in the provider's configured scope, **When** the user requests `env["*"]`, **Then** the provider returns an empty collection without error.
3. **Given** a provider configured with prefix `MYAPP_` and variables `MYAPP_KEY=value1` and `SYSTEM_PATH=/usr/bin` are set, **When** the user requests `env["*"]`, **Then** only the prefix-filtered variables are returned (`KEY=value1`), with the configured prefix stripped from the key names.

---

### User Story 2 - Prefix-Scoped Wildcard Retrieval (Priority: P2)

A developer using hierarchical path addressing wants to retrieve all environment variables under a specific namespace by placing `*` after one or more path segments. For example, `env["database"]["*"]` retrieves all variables that map to the `DATABASE_` namespace (e.g., `DATABASE_HOST`, `DATABASE_PORT`), returning them as a collection keyed by the relative variable name.

**Why this priority**: This enables bulk retrieval of a logical grouping of variables without needing to know every variable name upfront, which is useful for dynamic configuration loading and reduces individual fetch boilerplate.

**Independent Test**: Can be tested by setting variables `DATABASE_HOST=localhost`, `DATABASE_PORT=5432`, and `APP_KEY=value`, requesting `env["database"]["*"]`, and verifying only `DATABASE_*` variables are returned with keys `HOST` and `PORT`.

**Acceptance Scenarios**:

1. **Given** environment variables `DATABASE_HOST=localhost`, `DATABASE_PORT=5432`, and `APP_KEY=value` are set, with the provider configured with uppercase-underscore path translation, **When** the user requests `env["database"]["*"]`, **Then** the provider returns a collection containing `HOST=localhost` and `PORT=5432`, excluding `APP_KEY`.
2. **Given** environment variable `DATABASE_URL=postgres://localhost` is set, **When** the user requests `env["database"]["*"]`, **Then** the returned collection contains `URL=postgres://localhost` keyed by the path-relative name (`URL`), not the full environment variable name.
3. **Given** no environment variables with the resolved prefix `DATABASE_` exist, **When** the user requests `env["database"]["*"]`, **Then** the provider returns an empty collection without error.

---

### User Story 3 - Deeply Nested Wildcard Retrieval (Priority: P3)

A developer needs to retrieve all environment variables within a nested namespace hierarchy. For example, `env["app"]["database"]["*"]` retrieves all variables mapping to `APP_DATABASE_*` (e.g., `APP_DATABASE_HOST`, `APP_DATABASE_PORT`), excluding variables at sibling namespaces such as `APP_CACHE_*`.

**Why this priority**: This extends wildcard support to deeper path hierarchies, enabling fine-grained subset retrieval for complex configurations. It follows naturally from prefix-scoped wildcards.

**Independent Test**: Can be tested by setting variables `APP_DATABASE_HOST=localhost`, `APP_DATABASE_PORT=5432`, and `APP_CACHE_HOST=redis`, requesting `env["app"]["database"]["*"]`, and verifying only the `APP_DATABASE_*` variables are returned.

**Acceptance Scenarios**:

1. **Given** environment variables `APP_DATABASE_HOST=localhost`, `APP_DATABASE_PORT=5432`, and `APP_CACHE_HOST=redis` are set, **When** the user requests `env["app"]["database"]["*"]`, **Then** the provider returns a collection containing `HOST=localhost` and `PORT=5432`, excluding `APP_CACHE_HOST`.
2. **Given** a deeply nested path `env["service"]["db"]["replica"]["*"]`, **When** the request is made, **Then** only variables matching the fully resolved prefix `SERVICE_DB_REPLICA_` are returned, with that full prefix stripped from all returned key names.

---

### User Story 4 - Wildcard Position Validation (Priority: P4)

A developer accidentally places `*` in a non-terminal position in the path (e.g., `env["*"]["host"]`). The provider detects this invalid usage and returns a clear error message indicating that `*` is only valid as the final segment of a path, preventing undefined or partial results.

**Why this priority**: This enforces a clear, predictable contract for the feature and protects users from inadvertently receiving incorrect results due to misuse of the wildcard operator.

**Independent Test**: Can be tested by sending a path with `*` in a non-terminal position and verifying the provider returns a descriptive error rather than any result.

**Acceptance Scenarios**:

1. **Given** a request path where `*` appears before additional segments (e.g., path `["*", "host"]`), **When** the provider processes the request, **Then** it returns an error clearly stating that the wildcard operator is only valid at the terminal position of a path.
2. **Given** a request path with `*` appearing multiple times (e.g., `["database", "*", "*"]`), **When** the provider processes the request, **Then** it returns an error indicating invalid wildcard placement rather than any partial result.

---

### Edge Cases

- What happens when `*` is the only segment and no variables exist in scope? → Provider returns an empty collection without error.
- What happens when thousands of environment variables match a root-level wildcard? → Provider returns all matching variables; callers are responsible for appropriate scoping to manage result size.
- What happens when the path prefix before `*` combines with a configured provider-level prefix (e.g., configured prefix `MYAPP_`, path `env["other"]["*"]`)? → The two prefixes are combined (`MYAPP_OTHER_`), and only variables matching that combined prefix are returned.
- What happens when type conversion is in effect and a wildcard result is returned? → Type conversion applies to each individual value within the returned collection in the same way as for single-key lookups.
- What happens when two environment variables produce the same relative key after prefix stripping (e.g., ambiguous prefix scope)? → Behaviour is non-deterministic: one value survives in the map and which one is unspecified. This is an operator misconfiguration concern, not a provider error condition.
- Can `*` be used as a literal variable name? → No. `*` is exclusively reserved as the wildcard operator and is never treated as a literal key name.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The provider MUST accept a path request where the terminal (last) segment is the wildcard operator `*`.
- **FR-002**: When `*` is the sole path segment (root-level spread), the provider MUST return all environment variables available within its configured scope as a collection of key-value pairs.
- **FR-003**: When `*` follows one or more path segments, the provider MUST return only the environment variables whose resolved names begin with the prefix constructed from the preceding path segments using the provider's configured translation rules (case conversion, separator).
- **FR-004**: The provider MUST strip the matched path prefix from the returned key names, returning only the raw suffix from the original environment variable name with no case transformation applied (e.g., `DATABASE_HOST` requested via `env["database"]["*"]` is returned with key `HOST`; `database_host` requested via `env["database"]["*"]` is returned with key `host`).
- **FR-005**: When a provider-level prefix is configured with `prefix_mode=prepend`, any wildcard operation MUST combine the provider prefix with any path-derived prefix before resolving matching variables, and the full combined prefix MUST be stripped from all returned key names. When `prefix_mode=filter_only`, the provider prefix is not prepended; the path-derived prefix alone is used as the match prefix (see FR-010).
- **FR-006**: The provider MUST reject any path request that contains `*` in a non-terminal position and MUST return a descriptive error message indicating the correct usage.
- **FR-007**: When a wildcard path resolves to no matching variables, the provider MUST return an empty collection without error.
- **FR-008**: Path segment translation rules (case conversion and separator) configured for the provider MUST apply consistently when constructing the prefix used for wildcard resolution, matching the behaviour of single-key path lookups.
- **FR-009**: A wildcard path request MUST be fulfilled via a single unary gRPC response whose value field is a structured map of string keys to values (`map<string, Value>`), where each entry is one matched environment variable with its raw-suffix key and its (optionally type-converted) value.
- **FR-010**: When `prefix_mode=filter_only` and a wildcard path is issued, the provider MUST use only the path-derived prefix as the match prefix (the provider-level prefix is not prepended). If the path-derived prefix does not begin with the configured provider prefix — indicating the request is outside the configured scope — the provider MUST return an empty collection without error, consistent with the `filter_only` single-key filtering behaviour.

### Key Entities

- **Wildcard Path Request**: A path request where the final segment is `*`, used to retrieve all variables whose resolved names match the prefix formed by the preceding path segments.
- **Path Prefix**: The resolved environment variable name fragment constructed from all path segments preceding `*`, built using the provider's configured path translation rules.
- **Spread Result**: A collection of key-value pairs returned in response to a wildcard path request, delivered as a single unary gRPC response whose value field is a structured map (`map<string, Value>`). Each key is the raw suffix of the original environment variable name after the full matched prefix is stripped, with no case transformation applied. Each value is the (optionally type-converted) variable value.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Developers can retrieve all variables within a namespace using a single wildcard request, reducing the number of individual fetch operations to one regardless of how many variables exist in that namespace.
- **SC-002**: Wildcard query response time MUST be no worse than the cumulative time required to perform an equivalent set of individual single-key lookups for the same matched variables (i.e., no superlinear per-result overhead is introduced by the wildcard evaluation path).
- **SC-003**: 100% of path requests with `*` in a non-terminal position are rejected with a descriptive error, with no partial or undefined results returned under any circumstance.
- **SC-004**: Empty-namespace wildcard requests return an empty collection in 100% of cases, with no errors or unexpected behaviour caused by the absence of matching variables.
- **SC-005**: Wildcard results correctly apply existing path translation rules and configured prefix scoping in 100% of cases, ensuring no variables outside the configured scope are included in any spread result.

## Assumptions

- The `*` character is exclusively reserved as the wildcard operator in path requests and cannot be used as a literal environment variable name.
- The provider-level prefix is the **sole access control mechanism** for wildcard operations. No additional allow-listing or wildcard-specific restriction is applied inside the provider. Operators are responsible for configuring a sufficiently restrictive prefix to limit variable exposure. Broader authorization concerns (e.g., per-caller access control) are the responsibility of the Nomos compiler layer, not the provider.
- The returned collection uses relative key names (prefix stripped) rather than full environment variable names, consistent with how individual path-based lookups expose variable names after translation in feature 001.
- The spread operator supports only terminal wildcard matching. Glob-style patterns such as mid-path wildcards (e.g., `env["db*"]`) are out of scope for this feature.
- Type conversion (as specified in feature 001, User Story 4) applies to each individual value within a spread result in the same way as for individual key lookups.
- Wildcard results are returned as a single unary gRPC response containing a structured map value (`map<string, Value>`), not as a streaming sequence of individual responses. An empty wildcard result is represented as an empty map, not as an error or absence of response.
- If two environment variables within the resolved scope produce the same relative key after prefix stripping, the surviving value in the returned map is non-deterministic. This is treated as operator misconfiguration; the provider does not detect or report this condition.
- The order of key-value pairs in the returned collection is non-deterministic, consistent with standard environment variable enumeration behaviour.
- Path translation rules (uppercase/lowercase, separator character) are already established by feature 001 and apply unchanged to wildcard prefix construction.
