#!/bin/bash
# Copyright Splunk Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/../.." && pwd )"

echo "==> Starting OTel Collector..."
cd "$SCRIPT_DIR"
docker-compose up -d

echo "==> Waiting for collector to be ready..."
sleep 3

# Check if collector is running
if ! docker-compose ps | grep -q "Up"; then
    echo "ERROR: Collector failed to start"
    docker-compose logs
    exit 1
fi

echo "==> Cleaning up old metrics file..."
rm -f /tmp/otel-metrics.json

echo "==> Running integration tests..."
cd "$PROJECT_ROOT"
go test -tags=integration ./test/integration/... -v -timeout 30s

TEST_EXIT_CODE=$?

echo "==> Stopping collector..."
cd "$SCRIPT_DIR"
docker-compose down

if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo "==> Integration tests PASSED ✓"
else
    echo "==> Integration tests FAILED ✗"
fi

exit $TEST_EXIT_CODE

