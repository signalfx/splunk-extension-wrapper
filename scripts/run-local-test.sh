#!/bin/bash

function _panic() {
  >&2 echo "$1"
  exit 1
}

INPUT_FILE=$1
export ZIP_NAME=bin/extension.zip
FUNCTION_PATH=bin/test/function.zip

[[ -z "$INPUT_FILE" ]] && _panic "Error: INPUT_FILE not defined."
[[ -f "$INPUT_FILE" ]] || _panic "Error: $INPUT_FILE doesn't exist"

[[ -f "$ZIP_NAME" ]] || _panic "Error: the file with the extension doesn't exist ($ZIP_NAME)"

[[ -f "$FUNCTION_PATH" ]] || _panic "Error: the file with the function doesn't exist ($FUNCTION_PATH)"

[[ -z "$PROFILE" ]] && _panic "Error: PROFILE not defined."
[[ -z "$SPLUNK_REALM" ]] && _panic "Error: SPLUNK_REALM not defined."
[[ -z "$SPLUNK_ACCESS_TOKEN" ]] && _panic "Error: SPLUNK_ACCESS_TOKEN not defined."

echo "Input file: ${INPUT_FILE}"

export REGIONS=$(cat "$INPUT_FILE" | cut -d, -f2 | uniq | tr '\n' ' ')
echo "Regions: $REGIONS"

export LAYER_NAME=$(whoami)-extension-layer-test
echo "Layer name: $LAYER_NAME"

echo "AWS profile: ${PROFILE}"
echo "Splunk realm: ${SPLUNK_REALM}"
echo -n "Splunk ingest token: "
echo "${SPLUNK_ACCESS_TOKEN}" | sed 's/./\*/g'

TMP=$(mktemp -d)
echo "Test dir: $TMP"

export VERSIONS_FILE=$(mktemp)
echo "Versions file: ${VERSIONS_FILE}"

export AWS_PROFILE=$PROFILE

#########################
# create layer versions

$(dirname $0)/add-layer-version.sh

#########################
# prepare request files - they'll be used to create and delete a function

export FUNCTION_ZIP=$(base64 -i "$FUNCTION_PATH")

cat "$INPUT_FILE" | cut -d, -f1,2 | uniq | \
while IFS=, read -r FUNCTION_NAME reg; do
  export FUNCTION_LAYER=$(grep "$reg" "$VERSIONS_FILE")

  dir=${TMP}/${reg}/${FUNCTION_NAME}; mkdir -p "$dir" || _panic "Can't create a tmp dir for $FUNCTION_NAME function"

  export FUNCTION_NAME=$(whoami)-$FUNCTION_NAME

  for template in test/*.json.template; do
    cat "$template" | envsubst > ${dir}/$(basename -s ".template" "$template")
  done
done


#########################
# create functions

cat "$INPUT_FILE" | cut -d, -f1,2 | uniq | \
while IFS=, read -r fn reg; do
  echo "Creating function $fn in $reg"

  aws lambda create-function \
    --region $reg \
    --cli-input-json file://${TMP}/${reg}/${fn}/add-test-function.json \
    --query "FunctionArn" --output text \
    --no-cli-pager ||
      _panic "Can't create the function '$fn' in $reg"
done


function cleanup() {
  trap - INT TERM
  kill $(jobs -p); wait

  #########################
  # delete functions

  cat "$INPUT_FILE" | cut -d, -f1,2 | uniq | \
  while IFS=, read -r fn reg; do
    echo "Deleting function $fn in $reg"

    aws lambda delete-function \
      --region $reg \
      --cli-input-json file://${TMP}/${reg}/${fn}/delete-test-function.json \
      --no-cli-pager ||
        _panic "Can't delete the function '$fn' in $reg"
  done

  #########################
  # delete layer versions

  $(dirname $0)/delete-layer-version.sh

  exit 0
}

trap cleanup INT TERM

#########################
# run tests

while IFS=, read -r fn reg times invoke_delay; do
  (REGION=$reg FUNCTION_NAME=$(whoami)-$fn SKIP_CREATE=true INVOKE_DELAY=$invoke_delay \
    $(dirname $0)/run-test-in-region.sh "$times" > /dev/null; echo $fn in $reg is done) &
done < "$INPUT_FILE"

echo "Waiting for scenarios to finish..."
wait

cleanup
