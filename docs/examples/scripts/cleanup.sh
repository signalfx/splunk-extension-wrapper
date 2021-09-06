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

if (aws lambda get-function --function-name "${BUFFERED_FUNCTION_NAME}" --no-cli-pager > /dev/null 2> /dev/null); then
  aws lambda delete-function --no-cli-pager \
    --function-name "${BUFFERED_FUNCTION_NAME}" ||
      _panic "Can't delete the ${BUFFERED_FUNCTION_NAME} function"

  echo "The ${BUFFERED_FUNCTION_NAME} function was deleted"
fi

if (aws lambda get-function --function-name "${REAL_TIME_FUNCTION_NAME}" --no-cli-pager > /dev/null 2> /dev/null); then
  aws lambda delete-function --no-cli-pager \
    --function-name "${REAL_TIME_FUNCTION_NAME}" ||
      _panic "Can't delete the ${REAL_TIME_FUNCTION_NAME} function"

  echo "The ${REAL_TIME_FUNCTION_NAME} function was deleted"
fi

aws iam detach-role-policy --no-cli-pager \
  --role-name "${ROLE_NAME}" \
  --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole

aws iam delete-role \
  --role-name "${ROLE_NAME}" \
  --no-cli-pager ||
  _panic "Can't delete the ${ROLE_NAME} role"

echo "The ${ROLE_NAME} role was deleted"
