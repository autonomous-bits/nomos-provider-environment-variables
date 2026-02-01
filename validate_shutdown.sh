#!/bin/bash
# T098: Test graceful shutdown with SIGTERM and SIGINT

set -e

echo "=== T098: Graceful Shutdown Validation ==="
echo

# Build the provider
echo "Building provider..."
make build > /dev/null 2>&1

# Test 1: SIGTERM
echo "Test 1: Testing SIGTERM signal..."
./nomos-provider-environment-variables > /tmp/provider_sigterm.log 2>&1 &
PID=$!
sleep 1
echo "  Provider PID: $PID"
kill -TERM $PID
wait $PID
EXIT_CODE=$?
echo "  Exit code: $EXIT_CODE"
if [ $EXIT_CODE -eq 0 ]; then
    echo "  ✓ SIGTERM: Provider exited cleanly with status 0"
else
    echo "  ✗ SIGTERM: Provider exited with non-zero status: $EXIT_CODE"
fi

# Test 2: SIGINT
echo
echo "Test 2: Testing SIGINT signal (Ctrl+C simulation)..."
./nomos-provider-environment-variables > /tmp/provider_sigint.log 2>&1 &
PID=$!
sleep 1
echo "  Provider PID: $PID"
kill -INT $PID
wait $PID
EXIT_CODE=$?
echo "  Exit code: $EXIT_CODE"
if [ $EXIT_CODE -eq 0 ]; then
    echo "  ✓ SIGINT: Provider exited cleanly with status 0"
else
    echo "  ✗ SIGINT: Provider exited with non-zero status: $EXIT_CODE"
fi

# Test 3: Verify PORT output
echo
echo "Test 3: Verifying PORT output..."
timeout 2 ./nomos-provider-environment-variables 2>/dev/null | head -1 > /tmp/provider_port.log || true
PORT_LINE=$(cat /tmp/provider_port.log)
if [[ $PORT_LINE =~ ^PORT=[0-9]+$ ]]; then
    echo "  ✓ PORT output: $PORT_LINE"
else
    echo "  ✗ PORT output not found or malformed: $PORT_LINE"
fi

echo
echo "=== T098 Validation Complete ==="
