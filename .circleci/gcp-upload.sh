#!/usr/bin/env bash
set -euo pipefail

declare -a arches=("darwin_amd64" "windows_amd64" "linux_amd64")

chmod +x bin/*/fw*

# Confirm that version is set?
cli_version=$(bin/linux_amd64/fw version | grep Version | tr -s ' ' | cut -d ' ' -f 3)
if [ "${cli_version}" != "${CIRCLE_TAG}" ]; then
    echo "CLI Version: ${cli_version} does not match tag: ${CIRCLE_TAG}!"
    exit 1
fi

# Configure GCloud
echo $GCLOUD_SERVICE_KEY > ${HOME}/gcloud-service-key.json
gcloud auth activate-service-account --key-file=${HOME}/gcloud-service-key.json
gcloud config set project "${GCLOUD_PROJECT}"

# Zip Artifacts
mkdir -p zips/
(
    cd bin/
    for arch in "${arches[@]}"
    do
        zip -r ../zips/fw-$arch.zip $arch
    done
)

# Upload artifacts
(
    cd zips/
    gsutil cp *.zip "gs://flywheel-dist/cli/${CIRCLE_TAG}"
    gsutil acl ch -u AllUsers:R "gs://flywheel-dist/cli/${CIRCLE_TAG}/*"
)
