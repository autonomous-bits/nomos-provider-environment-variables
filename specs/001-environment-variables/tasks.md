---
description: "Development tasks for environment variables provider"
---

# Tasks: Environment Variables Provider

**Input**: Design documents from `/specs/001-environment-variables/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Following TDD principles per constitution - tests written first, validated to fail, then implementation

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Current Status

**✅ Completed Phases:**
- **Phase 1 (Setup)**: 5/6 tasks complete (83%)
- **Phase 2 (Foundational)**: 9/9 tasks complete (100%) ✓
- **Phase 3 (User Story 1 - MVP)**: 17/17 tasks complete (100%) ✓
- **Phase 8 (Polish - Partial)**: 5/15 tasks complete (33%)

**🎯 MVP Status: COMPLETE** - Basic environment variable fetching is fully functional!

**📊 Overall Progress**: 36/105 tasks complete (34%)

---

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [x] T001 Initialize Go module with go.mod and required dependencies (grpc, protobuf, provider-proto)
- [x] T002 Create project directory structure (cmd/, internal/, tests/, docs/)
- [ ] T003 [P] Configure .golangci.yml linter configuration
- [x] T004 [P] Create Makefile with build, test, lint, and cross-compile targets
- [x] T005 [P] Create README.md with provider overview and installation instructions
- [x] T006 [P] Create CHANGELOG.md following Keep a Changelog format

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T007 Create structured logger in internal/logger/logger.go with ERROR, WARN, INFO, DEBUG levels
- [x] T008 Create Config struct in internal/config/config.go with all configuration fields
- [x] T009 Implement config validation in internal/config/config.go (ValidateConfig function)
- [x] T010 Implement protobuf Struct parser in internal/config/parser.go (ParseConfig function)
- [x] T011 Create main.go entry point with gRPC server setup in cmd/provider/main.go
- [x] T012 Implement PORT announcement to stdout in cmd/provider/main.go
- [x] T013 Implement signal handling (SIGTERM, SIGINT) for graceful shutdown in cmd/provider/main.go
- [x] T014 Create base Provider struct with state fields in internal/provider/provider.go
- [x] T015 Register ProviderService with gRPC server in cmd/provider/main.go

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Basic Environment Variable Fetching (Priority: P1) 🎯 MVP

**Goal**: Enable direct environment variable access via single-segment paths, returning values as strings

**Independent Test**: Set `export API_KEY=secret123`, configure provider with minimal config, fetch `env["API_KEY"]`, verify value returned

### Tests for User Story 1 (TDD - Write First)

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [x] T016 [P] [US1] Integration test for Init RPC with minimal config in tests/integration/grpc_test.go
- [x] T017 [P] [US1] Integration test for successful Fetch RPC (variable exists) in tests/integration/grpc_test.go
- [x] T018 [P] [US1] Integration test for failed Fetch RPC (variable not found) in tests/integration/grpc_test.go
- [x] T019 [P] [US1] Integration test for Info RPC returning type and version in tests/integration/grpc_test.go
- [x] T020 [P] [US1] Integration test for Health RPC state transitions in tests/integration/lifecycle_test.go
- [x] T021 [P] [US1] Unit test for Config validation (valid minimal config) in internal/config/config_test.go
- [x] T022 [P] [US1] Unit test for environment variable lookup (exists, not exists, empty) in internal/fetcher/fetcher_test.go

### Implementation for User Story 1

- [x] T023 [US1] Implement Init RPC handler in internal/provider/init.go (parse config, validate, store alias)
- [x] T024 [US1] Implement Info RPC handler in internal/provider/info.go (return type="environment-variables", version, alias)
- [x] T025 [US1] Implement Health RPC handler in internal/provider/health.go (return STATUS_DEGRADED before init, STATUS_OK after)
- [x] T026 [US1] Create basic Fetcher in internal/fetcher/fetcher.go with direct os.Getenv lookup (without caching - caching added in T068)
- [x] T027 [US1] Implement Fetch RPC handler in internal/provider/fetch.go (single-segment path only, no transformation)
- [x] T028 [US1] Add NotFound error handling for missing variables in internal/provider/fetch.go
- [x] T029 [US1] Implement value to protobuf Struct conversion in internal/provider/fetch.go
- [x] T030 [US1] Implement Shutdown RPC handler in internal/provider/shutdown.go (transition to STOPPED state)
- [x] T031 [US1] Add path validation (non-empty, no empty segments) in internal/provider/fetch.go
- [x] T032 [US1] Add FailedPrecondition error if Fetch called before Init in internal/provider/fetch.go

**Checkpoint**: At this point, User Story 1 should be fully functional - direct variable access works

---

## Phase 4: User Story 2 - Path-Based Variable Name Mapping (Priority: P2)

**Goal**: Transform hierarchical paths to environment variable names using configurable separator and case conversion

**Independent Test**: Set `export DATABASE_HOST=localhost`, configure separator="_" and case_transform="upper", fetch `env["database"]["host"]`, verify resolves to `DATABASE_HOST` and returns `"localhost"`

### Tests for User Story 2 (TDD - Write First)

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T033 [P] [US2] Unit test for path transformation (uppercase + underscore) in tests/unit/resolver_test.go
- [ ] T034 [P] [US2] Unit test for path transformation (lowercase + underscore) in tests/unit/resolver_test.go
- [ ] T035 [P] [US2] Unit test for path transformation (preserve case + custom separator) in tests/unit/resolver_test.go
- [ ] T036 [P] [US2] Unit test for case transformation functions in tests/unit/transform_test.go
- [ ] T037 [P] [US2] Integration test for multi-segment path resolution in tests/integration/grpc_test.go
- [ ] T038 [P] [US2] Unit test for Config validation (invalid case_transform, invalid separator) in tests/unit/config_test.go

### Implementation for User Story 2

- [ ] T039 [P] [US2] Implement case transformation functions in internal/resolver/transform.go (ApplyUpper, ApplyLower, ApplyPreserve)
- [ ] T040 [P] [US2] Implement path segment joining with separator in internal/resolver/resolver.go
- [ ] T041 [US2] Implement full path-to-variable-name resolution in internal/resolver/resolver.go (ResolvePath function)
- [ ] T042 [US2] Update Config validation to check case_transform and separator in internal/config/config.go
- [ ] T043 [US2] Update Fetch handler to use resolver for multi-segment paths in internal/provider/fetch.go
- [ ] T044 [US2] Add support for both direct access and transformed access in internal/provider/fetch.go

**Checkpoint**: At this point, User Stories 1 AND 2 should both work - direct and hierarchical access both functional

---

## Phase 5: User Story 3 - Prefix-Based Filtering (Priority: P3)

**Goal**: Scope provider to variables with a specific prefix, with configurable prepend vs filter-only modes

**Independent Test**: Set `export MYAPP_KEY=value1` and `export OTHER_KEY=value2`, configure prefix="MYAPP_", fetch both, verify only MYAPP_KEY is accessible

### Tests for User Story 3 (TDD - Write First)

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T045 [P] [US3] Unit test for prefix prepending (prepend mode) in tests/unit/prefix_test.go
- [ ] T046 [P] [US3] Unit test for prefix filtering (filter_only mode) in tests/unit/prefix_test.go
- [ ] T047 [P] [US3] Unit test for prefix with path transformation in tests/unit/resolver_test.go
- [ ] T048 [P] [US3] Integration test for prefix + prepend mode in tests/integration/grpc_test.go
- [ ] T049 [P] [US3] Integration test for prefix + filter_only mode in tests/integration/grpc_test.go
- [ ] T050 [P] [US3] Unit test for Config validation (invalid prefix_mode) in tests/unit/config_test.go

### Implementation for User Story 3

- [ ] T051 [P] [US3] Implement prefix prepending logic in internal/resolver/prefix.go (PrependPrefix function)
- [ ] T052 [P] [US3] Implement prefix filtering logic in internal/resolver/prefix.go (FilterByPrefix function)
- [ ] T053 [US3] Update Config validation to check prefix_mode in internal/config/config.go
- [ ] T054 [US3] Integrate prefix logic into ResolvePath in internal/resolver/resolver.go
- [ ] T055 [US3] Update Fetch handler to respect prefix_mode configuration in internal/provider/fetch.go
- [ ] T056 [US3] Add NotFound error for filtered-out variables in internal/provider/fetch.go

**Checkpoint**: All basic features complete - direct access, path mapping, and prefix filtering all work independently

---

## Phase 6: User Story 4 - Type Conversion from String Values (Priority: P4)

**Goal**: Automatically convert environment variable strings to numbers, booleans, and JSON objects based on content patterns

**Independent Test**: Set `export PORT=8080`, fetch `env["PORT"]`, verify returns number 8080 not string "8080"

### Tests for User Story 4 (TDD - Write First)

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T057 [P] [US4] Unit test for numeric conversion (integers and floats) in tests/unit/converter_test.go
- [ ] T058 [P] [US4] Unit test for boolean conversion (true/false/yes/no) in tests/unit/converter_test.go
- [ ] T059 [P] [US4] Unit test for JSON parsing (objects and arrays) in tests/unit/converter_test.go
- [ ] T060 [P] [US4] Unit test for conversion precedence (number before boolean) in tests/unit/converter_test.go
- [ ] T061 [P] [US4] Unit test for JSON parsing errors in tests/unit/converter_test.go
- [ ] T062 [P] [US4] Integration test for type-converted values in Fetch response in tests/integration/grpc_test.go
- [ ] T063 [P] [US4] Unit test for empty string handling in tests/unit/converter_test.go

### Implementation for User Story 4

- [ ] T064 [P] [US4] Implement numeric conversion in internal/fetcher/converter.go (TryNumeric function)
- [ ] T065 [P] [US4] Implement boolean conversion in internal/fetcher/converter.go (TryBoolean function)
- [ ] T066 [P] [US4] Implement JSON parsing in internal/fetcher/converter.go (TryJSON function with 100-level depth limit per FR-030)
- [ ] T067 [US4] Implement type conversion coordinator in internal/fetcher/converter.go (ConvertValue function: check JSON prefix first, then apply number/boolean/string precedence per FR-016)
- [ ] T068 [US4] Add caching to Fetcher with sync.Map in internal/fetcher/fetcher.go (depends on T026; cache lifecycle per FR-031: persist until Shutdown, clear on Shutdown)
- [ ] T069 [US4] Update Fetch handler to apply type conversion when enabled in internal/provider/fetch.go
- [ ] T070 [US4] Add InvalidArgument error for JSON parsing failures in internal/provider/fetch.go
- [ ] T071 [US4] Implement value size validation (1MB limit) in internal/fetcher/fetcher.go

**Checkpoint**: Type conversion working - numbers, booleans, and JSON all converted correctly

---

## Phase 7: User Story 5 - Required Variable Validation (Priority: P5)

**Goal**: Validate required variables exist during Init and fail fast with clear error messages listing missing variables

**Independent Test**: Configure required_variables=["API_KEY", "SECRET"], omit SECRET, verify Init fails immediately with "required environment variables missing: SECRET"

### Tests for User Story 5 (TDD - Write First)

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T072 [P] [US5] Unit test for single required variable validation (exists) in tests/unit/validator_test.go
- [ ] T073 [P] [US5] Unit test for single required variable validation (missing) in tests/unit/validator_test.go
- [ ] T074 [P] [US5] Unit test for multiple required variables validation in tests/unit/validator_test.go
- [ ] T075 [P] [US5] Integration test for Init failure with missing required variables in tests/integration/grpc_test.go
- [ ] T076 [P] [US5] Unit test for required variables with prefix in tests/unit/validator_test.go

### Implementation for User Story 5

- [ ] T077 [US5] Implement required variable validation in internal/fetcher/validator.go (ValidateRequired function)
- [ ] T078 [US5] Update Init handler to call validator during initialization in internal/provider/init.go
- [ ] T079 [US5] Format error messages to list all missing variables in internal/fetcher/validator.go
- [ ] T080 [US5] Add InvalidArgument error code for missing required variables in internal/provider/init.go

**Checkpoint**: All user stories complete - provider fully functional with all P1-P5 features

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [x] T081 [P] Add comprehensive logging for all RPC methods in internal/provider/
- [x] T082 [P] Add version injection via ldflags in Makefile
- [ ] T083 [P] Create integration test fixtures in tests/integration/fixtures/
- [ ] T084 [P] Add test for concurrent Fetch calls (thread safety) in tests/integration/grpc_test.go
- [ ] T085 [P] Add test for platform-native case sensitivity in tests/unit/fetcher_test.go
- [ ] T086 [P] Add test for special characters in variable names (dots, dashes) in tests/unit/fetcher_test.go
- [ ] T087 [P] Add test for Unicode/UTF-8 variable names and values in tests/unit/fetcher_test.go
- [ ] T088 [P] Add test for Windows case-insensitivity behavior in tests/integration/grpc_test.go (skip on Unix)
- [ ] T089 [P] Add benchmark test for SC-002 (fetch response <10ms) in tests/integration/performance_test.go
- [ ] T090 [P] Add benchmark test for SC-006 (concurrent fetches) in tests/integration/performance_test.go
- [ ] T091 [P] Create quickstart validation script to test examples from quickstart.md
- [ ] T092 [P] Update README.md with build instructions and usage examples
- [ ] T093 [P] Add GitHub Actions workflow for CI (test, lint, build)
- [x] T094 [P] Create cross-platform build script for releases (darwin/amd64, darwin/arm64, linux/amd64, linux/arm64, windows/amd64)
- [x] T095 [P] Generate SHA256SUMS file for release binaries
- [x] T096 Run full test suite and verify >80% coverage
- [x] T097 Run golangci-lint and fix all issues
- [ ] T098 Test graceful shutdown with SIGTERM and SIGINT
- [ ] T099 Verify binary has no external runtime dependencies (ldd check)
- [ ] T100 Manual end-to-end test of all user stories
- [ ] T101 Validate SC-001 (environment variable import within 30 seconds)
- [ ] T102 Validate SC-003 (initialization fails within 2 seconds for missing required vars)
- [ ] T103 Validate SC-004 (hierarchical path mapping works in 100% of use cases)
- [ ] T104 Validate SC-005 (type conversion accuracy ≥95%)
- [ ] T105 Validate SC-007 (cross-platform behavior identical)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-7)**: All depend on Foundational phase completion
  - User stories can then proceed in priority order: US1 → US2 → US3 → US4 → US5
  - Or in parallel if team capacity allows (after Foundational complete)
- **Polish (Phase 8)**: Depends on all desired user stories being complete

### User Story Dependencies

- **US1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **US2 (P2)**: Depends on US1 (extends Fetch handler and resolver) - Builds on basic variable access
- **US3 (P3)**: Depends on US2 (adds prefix to resolver) - Builds on path transformation
- **US4 (P4)**: Depends on US1 (adds conversion to fetcher) - Can work in parallel with US2/US3 if staffed
- **US5 (P5)**: Depends on US1 (adds validation to Init) - Can work in parallel with US2/US3/US4 if staffed

### Within Each User Story

1. **Tests FIRST** (TDD): Write all tests for the story, ensure they FAIL
2. **Models/Core Logic**: Implement core functionality (resolver, converter, validator)
3. **Integration**: Wire into RPC handlers
4. **Validation**: Run tests, ensure they PASS
5. **Checkpoint**: Verify story works independently before moving to next

### Parallel Opportunities per Story

**User Story 1**:
```
Parallel: T016, T017, T018, T019, T020, T021, T022 (all tests)
Then: T023, T024, T025 (different RPC handlers)
Then: T026 (fetcher)
Then: T027-T032 (fetch handler updates)
```

**User Story 2**:
```
Parallel: T033, T034, T035, T036, T037, T038 (all tests)
Parallel: T039, T040 (transform and resolver core)
Then: T041-T044 (integration)
```

**User Story 3**:
```
Parallel: T045, T046, T047, T048, T049, T050 (all tests)
Parallel: T051, T052 (prefix logic)
Then: T053-T056 (integration)
```

**User Story 4**:
```
Parallel: T057, T058, T059, T060, T061, T062, T063 (all tests)
Parallel: T064, T065, T066 (conversion functions)
Then: T067-T071 (integration and caching)
```

**User Story 5**:
```
Parallel: T072, T073, T074, T075, T076 (all tests)
Then: T077-T080 (validation implementation)
```

**Polish (Phase 8)**:
```
Parallel: T081, T082, T083, T084, T085, T086, T087, T088, T089, T090 (all independent)
Then: T091-T095 (final validation)
```

---

## Parallel Example: User Story 1 Tests

```bash
# Launch all tests for User Story 1 together:
go test -run TestInitRPC tests/integration/grpc_test.go &
go test -run TestFetchSuccess tests/integration/grpc_test.go &
go test -run TestFetchNotFound tests/integration/grpc_test.go &
go test -run TestInfoRPC tests/integration/grpc_test.go &
go test -run TestHealthTransitions tests/integration/lifecycle_test.go &
go test -run TestConfigValidation tests/unit/config_test.go &
go test -run TestEnvLookup tests/unit/fetcher_test.go &
wait
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T006)
2. Complete Phase 2: Foundational (T007-T015) ← CRITICAL GATE
3. Complete Phase 3: User Story 1 (T016-T032)
4. **STOP and VALIDATE**: 
   - Set environment variables
   - Start provider
   - Verify Init, Info, Health, Fetch, Shutdown all work
   - Test with Nomos CLI if available
5. **MVP READY**: Provider can fetch environment variables directly

### Incremental Delivery

1. **MVP (US1)**: Direct environment variable access - ship it! ✅
2. **v0.2 (US1 + US2)**: Add hierarchical path mapping - ship it! ✅
3. **v0.3 (US1 + US2 + US3)**: Add prefix filtering - ship it! ✅
4. **v0.4 (US1-US4)**: Add type conversion - ship it! ✅
5. **v1.0 (US1-US5)**: Add required variables validation - ship it! ✅

Each increment is independently valuable and deployable.

### Full Parallel Strategy (4 Developers)

After Foundational complete:
- **Developer A**: US1 (T016-T032) - 2 days
- **Developer B**: US2 (T033-T044) - starts after US1 day 1 - 2 days
- **Developer C**: US4 (T057-T071) - starts after US1 day 1 - 2 days
- **Developer D**: US5 (T072-T080) - starts after US1 day 1 - 1 day

Then:
- **Developer A + B**: US3 (T045-T056) - 1 day
- **Developer C + D**: Polish (T081-T095) - 2 days

**Total time**: ~5 days with 4 developers vs ~10 days with 1 developer

---

## Task Count Summary

- **Phase 1 (Setup)**: 5/6 tasks complete (83%)
- **Phase 2 (Foundational)**: 9/9 tasks complete (100%) ✓ BLOCKING PHASE COMPLETE
- **Phase 3 (US1 - MVP)**: 17/17 tasks complete (100%) ✓ MVP READY
- **Phase 4 (US2)**: 0/12 tasks complete (0%)
- **Phase 5 (US3)**: 0/12 tasks complete (0%)
- **Phase 6 (US4)**: 0/15 tasks complete (0%)
- **Phase 7 (US5)**: 0/9 tasks complete (0%)
- **Phase 8 (Polish)**: 5/15 tasks complete (33%)

**Total**: 36/105 tasks complete (34%)

**Test Coverage**: 7/7 US1 test tasks complete (100%) - All MVP tests passing

---

## Notes

- **TDD Strictly Enforced**: All test tasks must be completed and failing before starting implementation tasks
- **[P] tasks**: Different files, can run in parallel
- **[Story] labels**: Maps task to user story for traceability
- **File paths**: All paths specified per project structure in plan.md
- **Constitution compliance**: Integration tests for gRPC contract, >80% coverage, clean lifecycle
- **Independent stories**: Each story verifiable standalone at its checkpoint
- **Commit strategy**: Commit after each task or logical group for incremental progress
- **Validation**: Stop at each checkpoint to independently test the story before proceeding
