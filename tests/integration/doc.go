// Package integration contains integration tests for the environment-variables provider.
//
// These tests require the full provider implementation including gRPC server,
// and test end-to-end workflows. They are tagged with 'integration' and must
// be explicitly run using:
//
//	go test -tags=integration ./tests/integration/...
//
// Integration tests verify:
//   - gRPC service implementation and contracts
//   - Full request/response lifecycle
//   - Multi-segment path resolution
//   - Prefix handling (prepend and filter_only modes)
//   - Type conversion in fetch responses
//   - Error handling and status codes
package integration
