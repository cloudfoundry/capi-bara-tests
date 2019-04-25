#!/usr/bin/env bash

DORA_PATH="../java-dora"
DORA_MANIFEST_PATH="../java-config-server/dora_manifest.yml"

function clean() {
    ./gradlew clean
}

function build_config_server() {
    ./gradlew build
    cp java-config-server/build/libs/*.jar ${DORA_PATH}
}

function push_dora_with_sidecars() {
    pushd ${DORA_PATH} > /dev/null
      cf v3-create-app java-dora
      cf v3-apply-manifest -f ${DORA_MANIFEST_PATH}
      cf v3-push java-dora
    popd > /dev/null
}

function main() {
    clean
    build_config_server
    push_dora_with_sidecars
}

main