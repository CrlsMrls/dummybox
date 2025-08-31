#!/bin/bash

# Test script to verify context cancellation issues in log and memory endpoints

echo "Building dummybox..."
go build -o dummybox .

echo "Starting server in background..."
./dummybox --port=8082 --auth-token=test-token &
SERVER_PID=$!

# Wait for server to start
sleep 2

echo "=== Testing /log endpoint context cancellation ==="
echo "1. Testing interval logging (should continue after HTTP request completes):"
curl -s -w "\nHTTP request completed in %{time_total}s\n" "http://localhost:8082/log?level=info&interval=2&duration=10&token=test-token" | jq .

echo ""
echo "Waiting 15 seconds to see if interval logging continues..."
echo "Look for log entries every 2 seconds in server output above."
sleep 15

echo ""
echo "=== Testing /memory endpoint context cancellation ==="
echo "2. Testing memory allocation (should hold memory for full duration):"
curl -s -w "\nHTTP request completed in %{time_total}s\n" "http://localhost:8082/memory?size=100&duration=8&token=test-token" | jq .

echo ""
echo "3. Checking memory usage immediately after request:"
ps aux | grep -E '[d]ummybox' | head -1

echo ""
echo "Waiting 10 seconds to see if memory gets deallocated due to context cancellation..."
sleep 10

echo ""
echo "4. Checking memory usage after wait (should have been deallocated by timeout, not context cancellation):"
ps aux | grep -E '[d]ummybox' | head -1

echo ""
echo "Stopping server..."
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null

echo "Test completed!"
echo ""
echo "Expected behavior:"
echo "- Log entries should appear every 2 seconds for 10 seconds total"
echo "- Memory should be held for 8 seconds, then deallocated by timeout"
echo "- No 'context cancellation' messages should appear"
