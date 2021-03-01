#!/bin/bash

function _panic() {
  >&2 echo "$1"
  exit 1
}

[[ -z "$PROFILE" ]] && _panic "Error: PROFILE not defined."
[[ -z "$LAYER_NAME" ]] && _panic "Error: LAYER_NAME not defined."
[[ -z "$REGIONS" ]] && _panic "Error: REGIONS not defined."

echo "Listing layer '${LAYER_NAME}' versions..."
echo "AWS profile: ${PROFILE}"
echo "Regions:  ${REGIONS}"

export AWS_PROFILE=$PROFILE

for region in ${REGIONS}; do
  LATEST_VERSION=$(aws lambda list-layer-versions \
    --layer-name "${LAYER_NAME}" \
    --region "${region}" \
    --max-items 1 \
    --query "LayerVersions[0].LayerVersionArn" | sed 's/^"//;s/"$//;s/^null$/layer not found/') ||
    _panic "Can't find ARN for the layer in ${region} region"

  echo "${region}: ${LATEST_VERSION}"
done

echo "Done."
