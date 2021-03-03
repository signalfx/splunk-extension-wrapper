#!/bin/bash

function _panic() {
  >&2 echo "$1"
  exit 1
}

[[ -z "$PROFILE" ]] && _panic "Error: PROFILE not defined."
[[ -z "$LAYER_NAME" ]] && _panic "Error: LAYER_NAME not defined."
[[ -z "$REGIONS" ]] && _panic "Error: REGIONS not defined."
[[ -z "$VERSIONS_FILE" ]] && _panic "Error: VERSIONS_FILE not defined."

echo "Deleting '${LAYER_NAME}' layer version..."
echo "AWS profile: ${PROFILE}"
echo "Regions: ${REGIONS}"
echo "Versions file: ${VERSIONS_FILE})"

export AWS_PROFILE=$PROFILE

for region in ${REGIONS}; do
  layer_version=$(grep ${region} ${VERSIONS_FILE} | cut -f8 -d:)

  echo "Deleting the layer version ${layer_version} from ${region} region..."

  aws lambda delete-layer-version \
    --layer-name "${LAYER_NAME}" \
    --version-number "${layer_version}" \
    --region "${region}" \
    --no-cli-pager ||
    _panic "Stopping script execution due to aws-cli error"
done

echo "The layer version is deleted"

