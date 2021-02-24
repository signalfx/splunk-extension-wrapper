#!/bin/bash

function _panic() {
  >&2 echo "$1"
  exit 1
}

[[ -z "$PROFILE" ]] && _panic "Error: PROFILE not defined."
[[ -z "$LAYER_NAME" ]] && _panic "Error: LAYER_NAME not defined."
[[ -z "$REGIONS" ]] && _panic "Error: REGIONS not defined."
[[ -z "$VERSIONS_FILE" ]] && _panic "Error: VERSIONS_FILE not defined."

echo "Setting permission for '${LAYER_NAME}' layer versions..."
echo "AWS profile: ${PROFILE}"
echo "Regions: ${REGIONS}"
echo "Versions file: ${VERSIONS_FILE}"

for region in ${REGIONS}; do
  echo "Making the layer available publicly in ${region} region..."

  LATEST_VERSION=$(grep ${region} ${VERSIONS_FILE} | cut -f8 -d:)

  echo "The latest version: ${LATEST_VERSION}"

  AWS_PROFILE=$PROFILE aws lambda add-layer-version-permission \
    --layer-name "${LAYER_NAME}" \
    --version-number "${LATEST_VERSION}" \
    --action lambda:GetLayerVersion \
    --statement-id any-account \
    --principal "*" \
    --output text \
    --region "${region}" \
    --no-cli-pager ||
    _panic "Can't set permission for ${LAYER_NAME}:${LATEST_VERSION}"
done

echo "Layer publication finished"
