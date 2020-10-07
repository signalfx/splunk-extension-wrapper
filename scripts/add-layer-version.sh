#!/bin/bash

function _panic() {
  >&2 echo "$1"
  exit 1
}

[[ -z "$PROFILE" ]] && _panic "Error: PROFILE not defined."
[[ -z "$LAYER_NAME" ]] && _panic "Error: LAYER_NAME not defined."
[[ -z "$ZIP_NAME" ]] && _panic "Error: ZIP_NAME not defined."
[[ -z "$REGIONS" ]] && _panic "Error: REGIONS not defined."

DESCRIPTION="The SignalFx Lambda Extension Layer provides customers with a simplified runtime-independent interface to collect high-resolution, low-latency metrics on Lambda Function execution."

echo "Adding '${LAYER_NAME}' layer versions..."
echo "AWS profile: ${PROFILE}"
echo "Zip file: ${ZIP_NAME}"
echo "Regions:  ${REGIONS}"

for region in ${REGIONS}; do
  echo "Adding the layer in ${region} region..."

  AWS_PROFILE=$PROFILE aws lambda publish-layer-version \
    --layer-name "${LAYER_NAME}" \
    --description "${DESCRIPTION}" \
    --license-info "Apache-2.0" \
    --zip-file "fileb://${ZIP_NAME}" \
    --region "${region}" ||
    _panic "Stopping script execution due to aws-cli error"
done

echo "The layer added to all supported regions"

