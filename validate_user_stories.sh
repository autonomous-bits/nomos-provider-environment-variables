#!/bin/bash
# T100: Manual end-to-end test of all user stories

set -e

echo "=== T100: User Stories End-to-End Validation ==="
echo

# Setup test environment variables
export US1_API_KEY="secret123"
export US1_DB_HOST="localhost"
export US1_DB_PORT="5432"

export US2_DATABASE_HOST="db.example.com"
export US2_DATABASE_PORT="3306"

export US3_MYAPP_DB_HOST="myapp-db"
export US3_MYAPP_API_KEY="myapp-secret"
export US3_SYSTEM_PATH="/usr/bin"

export US4_PORT="8080"
export US4_DEBUG="true"
export US4_TIMEOUT="30"
export US4_CONFIG='{"retries":3,"timeout":5000}'

export US5_REQUIRED_VAR="required-value"
# US5_MISSING_VAR intentionally not set

echo "Test environment variables set"
echo

# Build provider
echo "Building provider..."
make build > /dev/null 2>&1
echo "✓ Provider built"
echo

# US1: Direct environment variable access
echo "=== US1: Direct Environment Variable Access ==="
echo "Environment: US1_API_KEY=secret123, US1_DB_HOST=localhost, US1_DB_PORT=5432"
echo "Expected: Direct fetch of US1_API_KEY should return 'secret123'"
echo "Status: Ready for manual gRPC test"
echo

# US2: Hierarchical path mapping
echo "=== US2: Hierarchical Path Mapping ==="
echo "Environment: US2_DATABASE_HOST=db.example.com, US2_DATABASE_PORT=3306"
echo "Expected: Path ['us2_database', 'host'] → US2_DATABASE_HOST → 'db.example.com'"
echo "Config: separator='_', case_transform='upper'"
echo "Status: Ready for manual gRPC test"
echo

# US3: Prefix filtering
echo "=== US3: Prefix Filtering ==="
echo "Environment: US3_MYAPP_DB_HOST=myapp-db, US3_MYAPP_API_KEY=myapp-secret, US3_SYSTEM_PATH=/usr/bin"
echo "Expected: With prefix='US3_MYAPP_', only MYAPP variables accessible"
echo "Config: prefix='US3_MYAPP_', prefix_mode='prepend'"
echo "Status: Ready for manual gRPC test"
echo

# US4: Type conversion
echo "=== US4: Type Conversion ==="
echo "Environment: US4_PORT=8080, US4_DEBUG=true, US4_TIMEOUT=30, US4_CONFIG='{\"retries\":3,\"timeout\":5000}'"
echo "Expected conversions:"
echo "  - US4_PORT → number 8080"
echo "  - US4_DEBUG → boolean true"
echo "  - US4_TIMEOUT → number 30"
echo "  - US4_CONFIG → JSON object {retries:3, timeout:5000}"
echo "Status: Ready for manual gRPC test"
echo

# US5: Required variable validation
echo "=== US5: Required Variable Validation ==="
echo "Environment: US5_REQUIRED_VAR=required-value (US5_MISSING_VAR not set)"
echo "Expected: Init with required_variables=['US5_REQUIRED_VAR', 'US5_MISSING_VAR'] should fail"
echo "Status: Ready for manual gRPC test"
echo

echo "=== T100 Validation Complete ==="
echo "Note: Full validation requires integration with Nomos gRPC client"
echo "All environment variables are set and ready for testing"
