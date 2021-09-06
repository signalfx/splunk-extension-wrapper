#!/bin/bash

# Copyright Splunk Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

. $(dirname "$0")/common.sh

if ! (aws iam get-role --role-name "${ROLE_NAME}" > /dev/null 2> /dev/null); then
  aws iam create-role --no-cli-pager \
    --role-name "${ROLE_NAME}" \
    --assume-role-policy-document file://$(dirname "$0")/trust-policy.json > /dev/null ||
      _panic "Can't create the ${ROLE_NAME} role"

  aws iam attach-role-policy --no-cli-pager \
    --role-name "${ROLE_NAME}" \
    --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole ||
      _panic "Can't attach the basic execution role to the ${ROLE_NAME} role"

  echo "The ${ROLE_NAME} role was created"
  echo "Waiting for a while for the changes to propagate across all regions"

  sleep 10
fi

export ROLE_ARN=$(aws iam get-role --no-cli-pager \
  --role-name "${ROLE_NAME}" \
  --query "Role.Arn" --output text)

echo "The role arn is: ${ROLE_ARN}"

cat $(dirname "$0")/add-test-function.json.template |
  FUNCTION_NAME="${BUFFERED_FUNCTION_NAME}" \
  FAST_INGEST=false \
  envsubst | xargs -0 aws lambda create-function \
    --no-cli-pager --cli-input-json > /dev/null ||
      _panic "Can't create the ${BUFFERED_FUNCTION_NAME} function"

echo "The ${BUFFERED_FUNCTION_NAME} function was created"

cat $(dirname "$0")/add-test-function.json.template |
  FUNCTION_NAME="${REAL_TIME_FUNCTION_NAME}" \
  FAST_INGEST=true \
  envsubst | xargs -0 aws lambda create-function \
    --no-cli-pager --cli-input-json > /dev/null ||
      _panic "Can't create the ${REAL_TIME_FUNCTION_NAME} function"

echo "The ${REAL_TIME_FUNCTION_NAME} function was created"
