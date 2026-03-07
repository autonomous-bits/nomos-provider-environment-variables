# Research: Wildcard Path Spread Operator

**Phase**: 0 — Outline & Research  
**Branch**: `002-wildcard-path-spread`  
**Date**: 2026-03-07

---

## Decision 1: Wildcard Detection Location

**Question:** Where in the call stack should `*` detection and routing live?

**Decision:** In `internal/provider/fetch.go` — the gRPC `Fetch` handler.

**Rationale:** The existing handler already branches on path length (single-segment vs. multi-segment) at the top of the function. Wildcard detection is a third branch at the same level, keeping all path-routing logic co-located. Pushing detection into the resolver or fetcher would leak protocol concerns into lower layers.

**Alternatives considered:**
- Resolver layer: rejected — the resolver's responsibility is name transformation, not protocol routing.
- New gRPC method: rejected — the constitution requires strict contract fidelity; adding an RPC would be a MINOR version bump and is unnecessary since `Fetch` accepts arbitrary paths.

---

## Decision 2: Prefix Construction for Wildcard Matching

**Question:** How is the env var name prefix built from the path segments preceding `*`?

**Decision:** Reuse the existing `TransformSegments` + separator join from `resolver`, adding a new `BuildPrefix(path []string) (string, error)` method on `Resolver`. The result is the transformed, joined segments with a trailing separator appended — giving full-segment boundary matching (e.g., `["database"]` → `"DATABASE_"`).

**Algorithm:**
```
pathSegments = req.Path[0 : len(req.Path)-1]   // all before *

if len(pathSegments) == 0:
    // Root wildcard
    if prefixMode == "prepend" && prefix != "":
        matchPrefix = prefix           // e.g., "MYAPP_"
    else:
        matchPrefix = ""               // all vars

else:
    // Non-root wildcard
    transformedJoin = join(transform(pathSegments), separator)
    if prefixMode == "prepend" && prefix != "":
        matchPrefix = prefix + transformedJoin + separator  // "MYAPP_DATABASE_"
    else:
        matchPrefix = transformedJoin + separator           // "DATABASE_"

stripPrefix = matchPrefix   // same string stripped from each matched key
```

**Rationale:** Adding a trailing separator ensures `DATABASE_HOST` matches but `DATABASEX_HOST` does not. Reusing `TransformSegments` guarantees consistency with single-key lookups (FR-008).

**Alternatives considered:**
- No trailing separator: rejected — would match unintended variables with longer prefixes (e.g., `DATABASE2_HOST`).
- Separate prefix builder outside `Resolver`: rejected — would duplicate transformation logic.

---

## Decision 3: Environment Variable Enumeration

**Question:** How does the fetcher enumerate all env vars matching a given prefix?

**Decision:** Add `FetchAll(matchPrefix string) (map[string]string, error)` to `Fetcher`. It calls `os.Environ()` and iterates with `strings.HasPrefix` + `strings.Cut("=")`. The returned map keys are the raw suffix after stripping `matchPrefix` (FR-004 — no case conversion on keys).

**Rationale:** `os.Environ()` is the only correct, portable way to enumerate all environment variables in Go. There is no need for a cache here since the result is per-request and the variable set can change between requests.

**Alternatives considered:**
- Iterating the existing `sync.Map` cache: rejected — the cache is populated lazily on individual lookups; it will not contain all variables on a wildcard request.
- Reading `/proc/self/environ` on Linux: rejected — not portable; `os.Environ()` is cross-platform.

---

## Decision 4: FetchResponse Shape for Wildcard Results

**Question:** How is the spread map returned in the existing `FetchResponse` protobuf message?

**Decision:** The existing `FetchResponse.Value` field is `google.protobuf.Struct`. For wildcard results, the struct fields ARE the map entries directly (i.e., the struct is the map). This differs from the single-value response which wraps the result as `{"value": protoVal}`.

**Rationale:** `google.protobuf.Struct` is exactly a `map<string, Value>` — it is the natural wire representation for a key-value collection. The protocol already supports this; no schema changes are needed. Callers can distinguish a wildcard response from a single-value response by the presence of multiple top-level struct fields vs. the single `"value"` key.

**Alternatives considered:**
- Wrapping in `{"value": {map}}`: rejected — introduces an unnecessary extra level of nesting that callers must unwrap for every wildcard result.
- Streaming responses: rejected — user clarification Q3 chose unary response.

---

## Decision 5: `filter_only` Prefix Mode for Wildcard

**Question:** How does `filter_only` prefix mode interact with wildcard operations?

**Decision:** In `filter_only` mode, the provider prefix is NOT prepended to the path. For wildcard operations, the path-derived prefix is used as the `matchPrefix`. An additional safety check validates that the derived prefix starts with the configured provider prefix; if not, the provider returns an empty collection (consistent with how `filter_only` silently excludes non-prefixed vars in single-key lookups).

**Rationale:** In `filter_only` mode, the user is expected to include the prefix in the path itself (e.g., `env["MYAPP_DATABASE"]["*"]`). This is consistent with existing `filter_only` single-key behaviour. No new semantics are required.

**Alternatives considered:**
- Combining filter_only prefix with path prefix: rejected — would change `filter_only` semantics and break existing single-key behaviour.

---

## Decision 6: Error for Non-Terminal Wildcard

**Question:** What gRPC status code is used for a wildcard in non-terminal position?

**Decision:** `codes.InvalidArgument` with message: `"wildcard operator '*' is only valid at the terminal position of a path; found at index %d"`.

**Rationale:** This is a client-side input error, matching the existing pattern for `InvalidArgument` used for path validation in the current `Fetch` handler (FR-006).

---

## Decision 7: Type Conversion in Wildcard Results

**Question:** Does the existing type conversion pipeline apply to each value in a wildcard result?

**Decision:** Yes. If `enable_type_conversion` or `enable_json_parsing` is configured, the existing `convertValue` helper is called on each string value in the spread result before assembly into the struct. This is identical to the single-value path (spec Assumption: type conversion applies per-value).

**Rationale:** Reusing `convertValue` requires no new logic and ensures consistent behaviour across single and wildcard paths.

---

## Resolved NEEDS CLARIFICATION Items

_None. All clarifications were obtained during the `/speckit.clarify` session on 2026-03-07._
