# gRPC Contract: Wildcard Path Spread Operator

**Feature**: Wildcard Path Spread Operator  
**Date**: 2026-03-07  
**Protocol Version**: v1 (no new RPCs; extends existing `Fetch` semantics)

---

## Overview

This document specifies the extended `Fetch` RPC behaviour introduced by the wildcard spread operator. The base gRPC service contract is unchanged:

```protobuf
service ProviderService {
  rpc Fetch(FetchRequest) returns (FetchResponse);
  // Init, Info, Health, Shutdown — unchanged
}
```

No new protobuf messages or RPC methods are added. All changes are behavioural extensions to the existing `Fetch` RPC.

---

## Fetch RPC — Wildcard Extensions

### Trigger Condition

A `FetchRequest` is treated as a **wildcard request** when its `path` slice is non-empty and the **last element is the literal string `"*"`**.

```
FetchRequest { path: ["*"] }                         // root wildcard
FetchRequest { path: ["database", "*"] }             // prefix-scoped wildcard
FetchRequest { path: ["app", "database", "*"] }      // nested wildcard
```

---

### Validation

Before processing, the provider validates the path for wildcard placement:

| Condition | gRPC Status | Message |
|-----------|-------------|---------|
| `"*"` at any position except the last | `INVALID_ARGUMENT` | `wildcard operator '*' is only valid at the terminal position of a path; found at index <N>` |
| Empty path (existing rule, unchanged) | `INVALID_ARGUMENT` | `path cannot be empty` |
| Empty path segment (existing rule, unchanged) | `INVALID_ARGUMENT` | `path[<N>] cannot be empty string` |

---

### Successful Response

On a valid wildcard request, the provider returns a single unary `FetchResponse`. The `value` field is a `google.protobuf.Struct` where **each top-level field is one matched environment variable**:

```protobuf
FetchResponse {
  value: Struct {
    fields: {
      "HOST": Value { string_value: "localhost" },
      "PORT": Value { number_value: 5432 },
      "URL":  Value { string_value: "postgres://localhost" }
    }
  }
}
```

**Key format:** Raw suffix of the original environment variable name after the full match prefix is stripped. No case transformation is applied to keys.

**Value format:** Type-converted using the same rules as single-key `Fetch` (controlled by `enable_type_conversion` and `enable_json_parsing` config flags).

**Empty result:** When no environment variables match the resolved prefix, the response is:

```protobuf
FetchResponse {
  value: Struct { fields: {} }   // empty struct — not an error
}
```

---

### Prefix Resolution

The match prefix is constructed as follows:

```
namespace_segments = path[0 : len(path)-1]   // all segments before "*"

if len(namespace_segments) == 0:
    // Root wildcard
    if prefix_mode == "prepend" AND prefix != "":
        match_prefix = prefix                // e.g., "MYAPP_"
    else:
        match_prefix = ""                    // matches all variables
else:
    // Non-root wildcard
    transformed = apply_case(namespace_segments, case_transform)
    joined      = join(transformed, separator)
    if prefix_mode == "prepend" AND prefix != "":
        match_prefix = prefix + joined + separator  // e.g., "MYAPP_DATABASE_"
    else:
        match_prefix = joined + separator           // e.g., "DATABASE_"

strip_prefix = match_prefix   // same string stripped from every matched key
```

**Full-segment boundary:** The trailing `separator` is appended to `joined` to ensure that only variables in the requested namespace are matched (e.g., `DATABASE_` matches `DATABASE_HOST` but not `DATABASEX_HOST`).

---

### Prefix Mode Interaction

| `prefix_mode` | `prefix` | Root wildcard | Prefixed wildcard |
|---|---|---|---|
| `prepend` | `"MYAPP_"` | Returns all vars starting with `MYAPP_`, keys stripped of `MYAPP_` | Returns vars starting with `MYAPP_<JOINED>_`, keys stripped of full prefix |
| `prepend` | `""` | Returns all vars, no stripping | Returns vars starting with `<JOINED>_`, keys stripped of `<JOINED>_` |
| `filter_only` | `"MYAPP_"` | Returns all vars; keys not stripped (no prefix prepended) | Returns vars starting with `<JOINED>_`; keys stripped of `<JOINED>_` |

---

### Error Responses

| Condition | gRPC Code | Notes |
|-----------|-----------|-------|
| `"*"` in non-terminal position | `INVALID_ARGUMENT` | First offending index reported |
| Provider not initialized | `FAILED_PRECONDITION` | Existing behaviour, unchanged |
| Type conversion failure on a value | `INVALID_ARGUMENT` | Existing behaviour per value |

---

### Provider Configuration (unchanged)

No new configuration fields. Wildcard behaviour is governed entirely by the existing configuration:

| Field | Effect on Wildcard |
|-------|--------------------|
| `separator` | Used to join namespace segments when building the match prefix |
| `case_transform` | Applied to each namespace segment when building the match prefix |
| `prefix` | Combined with path-derived prefix per `prefix_mode` |
| `prefix_mode` | `"prepend"` or `"filter_only"` — see Prefix Mode Interaction table above |
| `enable_type_conversion` | Applied to each matched value before serialisation |
| `enable_json_parsing` | Applied to each matched value before serialisation |

---

## Differentiation from Single-Key Fetch

| Aspect | Single-key `Fetch` | Wildcard `Fetch` |
|--------|-------------------|-----------------|
| Path terminal | Any non-`"*"` string | `"*"` |
| Response struct | `{"value": <single value>}` | `{"KEY1": val1, "KEY2": val2, …}` |
| Not found | `NOT_FOUND` error | Empty struct (no error) |
| Key in response | N/A (only `"value"`) | Raw env var suffix, no case transform |
