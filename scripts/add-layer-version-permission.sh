#!/bin/bash

function _panic() {
  >&2 echo "$1"
  exit 1
}

[[ -z "$PROFILE" ]] && _panic "Error: PROFILE not defined."
[[ -z "$LAYER_NAME" ]] && _panic "Error: LAYER_NAME not defined."
[[ -z "$REGIONS" ]] && _panic "Error: REGIONS not defined."

echo "Setting permission for '${LAYER_NAME}' layer versions..."
echo "AWS profile: ${PROFILE}"
echo "Regions:  ${REGIONS}"

for region in ${REGIONS}; do
  echo "Making the layer available publicly in ${region} region..."

  LATEST_VERSION=$(AWS_PROFILE=$PROFILE aws lambda list-layer-versions \
    --layer-name "${LAYER_NAME}" \
    --region "${region}" \
    --max-items 1 \
    --query "LayerVersions[0].Version") ||
    _panic "Can't find ARN for the layer in ${region} region"

  echo "The latest version: ${LATEST_VERSION}"

  AWS_PROFILE=$PROFILE aws lambda add-layer-version-permission \
    --layer-name "${LAYER_NAME}" \
    --version-number "${LATEST_VERSION}" \
    --action lambda:GetLayerVersion \
    --statement-id any-account \
    --principal "*" \
    --output text \
    --region "${region}"
    --no-cli-pager ||
    _panic "Can't set permission for ${LAYER_NAME}:${LATEST_VERSION}"
done

echo "Layer publication finished"
