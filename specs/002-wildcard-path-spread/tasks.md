---
description: "Development tasks for wildcard path spread operator"
---

# Tasks: Wildcard Path Spread Operator

**Input**: Design documents from `specs/002-wildcard-path-spread/`
**Prerequisites**: plan.md ✓, spec.md ✓, data-model.md ✓, contracts/wildcard-fetch.md ✓, research.md ✓, quickstart.md ✓

**Tests**: TDD required per constitution Principle III (GATE) — tests written first, verified to fail before implementation

**Organization**: Tasks grouped by user story to enable independent implementation and testing. All changes extend existing packages in `internal/`; no new packages or files are created in source.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4)
- Include exact file paths in descriptions

---

## Phase 1: Setup

**Purpose**: Verify project baseline is clean before adding wildcard functionality.

- [X] T001 Verify project builds and all existing tests pass: `go test -race ./...` from repository root

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Implement `FetchAll` on `Fetcher` — the data enumeration primitive consumed by every wildcard user story. Fully unit-testable in isolation with no dependency on `BuildPrefix` or the provider layer.

**⚠️ CRITICAL**: `FetchAll` must be implemented and passing before `fetchWildcard` can be wired up in Phase 3.

- [X] T002 [P] Write `FetchAll` unit tests (TDD: prefix="" returns all vars; matchPrefix filters by `strings.HasPrefix`; no match returns empty map; `MaxValueSize` violation logs WARN and skips entry — `MaxValueSize` = 1 MB, defined in `internal/fetcher/fetcher.go`) in `tests/unit/fetcher_test.go`
- [X] T003 Implement `FetchAll(matchPrefix string) (map[string]string, error)` on `Fetcher` (snapshot `os.Environ()`, `strings.Cut("=")`, `strings.HasPrefix` filter, strip `matchPrefix` from key, skip exact-match keys where `relKey == ""`, WARN log and skip values exceeding `MaxValueSize` — 1 MB cap defined in `internal/fetcher/fetcher.go`) in `internal/fetcher/fetcher.go`

**Checkpoint**: `go test -race ./tests/unit/ -run TestFetchAll` passes — wildcard user story implementation can now begin.

---

## Phase 3: User Story 1 — Root-Level Wildcard Retrieval (Priority: P1) 🎯 MVP

**Goal**: A single `Fetch` call with `path: ["*"]` returns all environment variables in the provider's configured scope as a `structpb.Struct` map of key-value pairs.

**Independent Test**: Set `API_KEY=secret` and `DB_HOST=localhost`, call `Fetch(path: ["*"])`, verify the response struct contains `API_KEY=secret` and `DB_HOST=localhost`. Also verify an empty in-scope result returns an empty struct without error.

### Tests for User Story 1 (TDD — write FIRST, verify they FAIL before T006/T007)

- [X] T004 [P] [US1] Write `BuildPrefix` unit tests for root wildcard (empty `namespacePath` → `""` when no provider prefix; empty `namespacePath` + `prefix_mode=prepend` + `prefix="MYAPP_"` → `"MYAPP_"`) in `tests/unit/resolver_test.go`
- [X] T005 [P] [US1] Write root wildcard integration tests (spec US1-AC1: all in-scope vars returned; US1-AC2: empty in-scope returns empty struct, no error (SC-004); US1-AC3: provider `prefix="MYAPP_"` scopes and strips results) in `tests/integration/grpc_test.go`

### Implementation for User Story 1

- [X] T006 [US1] Implement `BuildPrefix(namespacePath []string) (string, error)` on `Resolver` (root case: empty `namespacePath` → return `r.prefix` if `prefixMode=="prepend" && prefix!=""`, else `""`; non-root case placeholder for T010) in `internal/resolver/resolver.go`
- [X] T007 [US1] Add wildcard validation loop after existing empty-path check (iterate `req.Path`; if `seg=="*"` and `i != len-1` return `codes.InvalidArgument` with message `"wildcard operator '*' is only valid at the terminal position of a path; found at index %d"`); add wildcard dispatch branch (`if req.Path[len-1] == "*"` call `p.fetchWildcard`); implement `fetchWildcard` helper (extract `namespacePath = path[:len-1]`; call `p.resolver.BuildPrefix`; log DEBUG match prefix; call `p.fetcher.FetchAll`; log DEBUG result count; call `p.maybeConvert` per entry; assemble `structpb.NewStruct`; return `FetchResponse`) in `internal/provider/fetch.go`

**Checkpoint**: `Fetch(path: ["*"])` returns all in-scope env vars as a struct map — US1 independently functional and testable.

---

## Phase 4: User Story 2 — Prefix-Scoped Wildcard Retrieval (Priority: P2)

**Goal**: `Fetch(path: ["database", "*"])` returns only `DATABASE_*` environment variables with keys stripped of the `DATABASE_` prefix (e.g., `HOST`, `PORT`), honouring the provider's case transform and separator configuration.

**Independent Test**: Set `DATABASE_HOST=localhost`, `DATABASE_PORT=5432`, `APP_KEY=value`, call `Fetch(path: ["database", "*"])`, verify response contains only `HOST=localhost` and `PORT=5432` — `APP_KEY` is excluded.

### Tests for User Story 2 (TDD — write FIRST, verify they FAIL before T010)

- [X] T008 [P] [US2] Write `BuildPrefix` unit tests for single-segment namespace (`["database"]` + `case=upper` + `sep="_"` → `"DATABASE_"`; with `prefix="MYAPP_"` → `"MYAPP_DATABASE_"`; empty segment → `ErrEmptySegment`) in `tests/unit/resolver_test.go`
- [X] T009 [P] [US2] Write prefix-scoped wildcard integration tests (spec US2-AC1: single-segment namespace, sibling exclusion; US2-AC2: returned key is relative suffix not full name; US2-AC3: empty `DATABASE_` result returns empty struct without error) in `tests/integration/grpc_test.go`

### Implementation for User Story 2

- [X] T010 [US2] Extend `BuildPrefix` for non-empty `namespacePath` (validate no empty segments using existing `ErrEmptySegment`; `transformed = TransformSegments(namespacePath, r.caseTransform)`; `joined = strings.Join(transformed, r.separator)`; return `r.prefix + joined + r.separator` if `prefixMode=="prepend" && prefix!=""`, else `joined + r.separator`) in `internal/resolver/resolver.go`

**Checkpoint**: `Fetch(path: ["database", "*"])` returns only `DATABASE_*` vars with stripped keys — US2 independently functional.

---

## Phase 5: User Story 3 — Deeply Nested Wildcard Retrieval (Priority: P3)

**Goal**: `Fetch(path: ["app", "database", "*"])` returns only `APP_DATABASE_*` variables, excluding sibling namespaces such as `APP_CACHE_*`. The fully combined prefix is stripped from all returned keys.

**Independent Test**: Set `APP_DATABASE_HOST=localhost`, `APP_DATABASE_PORT=5432`, `APP_CACHE_HOST=redis`, call `Fetch(path: ["app", "database", "*"])`, verify response contains only `HOST=localhost` and `PORT=5432` — `APP_CACHE_HOST` is excluded.

### Tests for User Story 3 (TDD — write FIRST, then run to verify pass-through from T010)

- [X] T011 [P] [US3] Write `BuildPrefix` unit tests for multi-segment namespaces (`["app", "database"]` → `"APP_DATABASE_"`; `["service", "db", "replica"]` → `"SERVICE_DB_REPLICA_"`; with provider prefix → full combined prefix) in `tests/unit/resolver_test.go`
- [X] T012 [P] [US3] Write nested wildcard integration tests (spec US3-AC1: `APP_DATABASE_*` returned, `APP_CACHE_*` excluded; US3-AC2: deeply nested path strips full combined prefix from returned keys) in `tests/integration/grpc_test.go`

### Implementation for User Story 3

- [X] T013 [US3] Run T011 unit tests against the `BuildPrefix` implementation from T010; if all pass (N-segment join handled by `strings.Join`) close as verified; if any test fails, fix the edge case in `internal/resolver/resolver.go` and re-run until green

**Checkpoint**: `Fetch(path: ["app", "database", "*"])` returns only `APP_DATABASE_*` vars — US3 independently functional.

---

## Phase 6: User Story 4 — Wildcard Position Validation (Priority: P4)

**Goal**: Any path containing `*` in a non-terminal position is rejected with `INVALID_ARGUMENT` and a message that identifies the offending index. No partial result is ever returned.

**Independent Test**: Send `Fetch(path: ["*", "host"])`, verify gRPC status `INVALID_ARGUMENT` with message `"wildcard operator '*' is only valid at the terminal position of a path; found at index 0"`.

### Tests for User Story 4 (TDD — write FIRST, verify covered by T007 validation loop)

- [X] T014 [P] [US4] Write non-terminal wildcard validation integration tests (spec US4-AC1: `["*", "host"]` → `INVALID_ARGUMENT` at index 0; US4-AC2: `["database", "*", "*"]` → `INVALID_ARGUMENT` at index 1; SC-003: no partial result returned in any case) in `tests/integration/grpc_test.go`

### Implementation for User Story 4

> **Note**: The validation loop was implemented as part of T007 (US1 scope). T015 confirms all T014 scenarios pass and fixes any gaps.

- [X] T015 [US4] Verify all T014 integration test cases pass against the T007 validation loop; fix error message formatting or missing index detection if any scenario fails; confirm WARN/ERROR log lines are emitted for `InvalidArgument` and `Internal` error paths within `fetchWildcard` (constitution Principle V) in `internal/provider/fetch.go`

**Checkpoint**: All non-terminal wildcard paths return `INVALID_ARGUMENT` — US4 independently functional, SC-003 satisfied.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Validate cross-cutting requirements — type conversion, `filter_only` prefix mode, provider-prefix combination, performance baseline, and quickstart scenario confirmation.

- [X] T016 [P] Add type conversion wildcard integration test (configure `enable_type_conversion: true`; verify numeric string value `"5432"` in wildcard result is returned as number `5432`; spec assumption: type conversion applies per value) in `tests/integration/grpc_test.go`
- [X] T017 [P] Add combined provider-prefix + path-prefix wildcard integration test (configured `prefix="MYAPP_"` + `prefix_mode=prepend` + `path=["database","*"]` → only `MYAPP_DATABASE_*` vars returned; full `MYAPP_DATABASE_` prefix stripped from keys, spec FR-005 and data-model PathPrefix examples) in `tests/integration/grpc_test.go`
- [X] T020 [P] Add `filter_only` prefix mode wildcard integration tests (contracts/wildcard-fetch.md Prefix Mode Interaction table — root wildcard returns all vars without stripping; prefixed wildcard uses path-derived prefix only; safety check: path-derived prefix outside configured scope returns empty collection, research Decision 5; spec FR-010) in `tests/integration/grpc_test.go`
- [X] T018 Run full test suite with race detection and verify coverage ≥ 80%: `go test -race -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | grep -E 'total|provider|fetcher|resolver'`; confirm no package falls below 80% threshold (constitution Principle III)
- [X] T019 Run quickstart validation to confirm all six quickstart.md scenarios work end-to-end: `./scripts/validate-quickstart.sh`
- [X] T021 Run performance validation to confirm SC-002 (wildcard response time ≤ cumulative individual lookups): `./validate_performance.sh`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 (clean baseline) — blocks US1 implementation (`FetchAll` must exist before `fetchWildcard`)
- **US1 (Phase 3)**: Depends on Phase 2 (`FetchAll` + passing unit tests)
- **US2 (Phase 4)**: Depends on US1 (`fetchWildcard` scaffold established, `BuildPrefix` root-path exists)
- **US3 (Phase 5)**: Depends on US2 (`BuildPrefix` non-empty namespace handles N segments via `strings.Join`)
- **US4 (Phase 6)**: Depends on US1 (validation loop implemented in T007)
- **Polish (Phase 7)**: Depends on all user story phases complete

### User Story Dependencies

- **US1 (P1)**: Depends only on Foundational — no story dependencies. MVP delivery target: complete T001–T007.
- **US2 (P2)**: Depends on US1 (`fetchWildcard` in place); T010 extends `BuildPrefix` to handle non-empty namespace.
- **US3 (P3)**: Implementation falls through from US2 — `strings.Join` handles N segments. Phase 5 primarily adds test coverage.
- **US4 (P4)**: Validation implemented in T007 (US1); Phase 6 writes tests and confirms correctness.

> **Parallel opportunity between stories**: After US1 is complete, US2 (T008, T009, T010), US3 (T011, T012), and US4 (T014) test-writing tasks can proceed in parallel by separate developers or agents, coordinating merges into `grpc_test.go`.

### Within Each User Story

1. Tests MUST be written (and confirmed to fail) before implementation tasks
2. `BuildPrefix` (resolver layer) before `fetchWildcard` (provider layer)
3. `FetchAll` (foundational) before `fetchWildcard` (US1)
4. Story checkpoint verified before moving to next priority phase

### Parallel Opportunities

- T002 and T001 are independent (T001 is a read-only verification)
- T004 [P] and T005 [P] within US1 — different files (`resolver_test.go` vs `grpc_test.go`), no dependency between them
- T008 [P] and T009 [P] within US2 — different files
- T011 [P] and T012 [P] within US3 — different files
- T014 [P] within US4 — independent write
- T016 [P], T017 [P], and T020 [P] within Polish — all three add to `grpc_test.go` (coordinate to avoid merge conflicts)

---

## Parallel Example: User Story 1

```bash
# Phase 2: implement FetchAll
go test -race ./tests/unit/ -run TestFetchAll  # T002: confirm tests exist and fail
# implement T003 (FetchAll)
go test -race ./tests/unit/ -run TestFetchAll  # verify T003 passes

# Phase 3: write tests in parallel (T004 and T005)
# Terminal 1: edit tests/unit/resolver_test.go  (T004)
# Terminal 2: edit tests/integration/grpc_test.go (T005)

go test -race ./tests/unit/ -run TestBuildPrefix       # confirm T004 fails
go test -race ./tests/integration/ -run TestWildcard   # confirm T005 fails

# Implement T006 (BuildPrefix root case)
go test -race ./tests/unit/ -run TestBuildPrefix       # verify T006 passes

# Implement T007 (fetchWildcard + dispatch + validation)
go test -race ./tests/integration/ -run TestWildcard   # verify T005 passes
go test -race ./tests/integration/ -run TestWildcard   # US1 checkpoint ✅
```

---

## Implementation Strategy

**MVP Scope**: US1 (Phase 3) alone delivers the core wildcard capability. After T001–T007, `Fetch(path: ["*"])` works against a running provider. This is the minimum shippable increment.

**Incremental Delivery**:
1. **After US1** (T001–T007): Root wildcard functional; provider-prefix scoping respected; observability logging in place
2. **After US2** (T008–T010): Namespace prefix scoping works; single-segment path correct
3. **After US3** (T011–T013): Deep hierarchy confirmed; multi-segment path coverage
4. **After US4** (T014–T015): Input validation hardened; SC-003 satisfied
5. **After Polish** (T016–T021): Type conversion, combined-prefix, `filter_only` mode, race-safety, coverage gate, quickstart, and performance (SC-002) confirmed

---

## Summary

| Metric | Value |
|--------|-------|
| Total tasks | 21 |
| Foundational tasks | 2 (T002–T003) |
| US1 tasks | 4 (T004–T007) |
| US2 tasks | 3 (T008–T010) |
| US3 tasks | 3 (T011–T013) |
| US4 tasks | 2 (T014–T015) |
| Polish tasks | 6 (T016, T017, T018, T019, T020, T021) |
| Tasks marked [P] (parallelizable) | 9 |
| MVP scope | T001–T007 (7 tasks) |

update