#!/bin/bash

function _panic() {
  >&2 echo "$1"
  exit 1
}

[[ -z "$PROFILE" ]] && _panic "Error: PROFILE not defined."
[[ -z "$REGION" ]] && _panic "Error: REGION not defined."
[[ -z "$FUNCTION_NAME" ]] && _panic "Error: FUNCTION_NAME not defined."

echo "AWS profile: ${PROFILE}"
echo "Region: ${REGION}"
echo "Function name: ${FUNCTION_NAME}"

export AWS_PROFILE=$PROFILE AWS_DEFAULT_REGION=$REGION

echo "Sending request to create a function"
cat bin/test/add-test-function.json

aws lambda create-function \
  --cli-input-json file://bin/test/add-test-function.json \
  --no-cli-pager ||
    _panic "Can't create the function"

response_file=$(mktemp)

for i in {1..5}; do
  echo "Calling the function ${i} time"

  aws lambda invoke \
    --function-name "${FUNCTION_NAME}" \
    --payload '{}' \
    --no-cli-pager \
    ${response_file} ||
      _panic "Can't invoke the function: $(cat ${response_file})"
done

echo "Sending request to delete the function"
cat bin/test/delete-test-function.json

aws lambda delete-function \
  --cli-input-json file://bin/test/delete-test-function.json \
  --no-cli-pager ||
    _panic "Can't remove the function"

echo "Done."
