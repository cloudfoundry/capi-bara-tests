#!/usr/bin/env bash

DORA_PATH="../dora"
DORA_MANIFEST_PATH="../golang-config-server/dora_manifest.yml"

function clean() {
    rm "${DORA_PATH}/config-server"
    rm config-server
}

function build_config_server() {
    GOOS=linux GOARCH=amd64 go build -o config-server .
    cp config-server ${DORA_PATH}
}

function push_dora_with_sidecars() {
    pushd ${DORA_PATH} > /dev/null
      cf create-app dora
      cf apply-manifest -f ${DORA_MANIFEST_PATH}
      cf push dora
    popd > /dev/null
}

function main() {
    clean
    build_config_server
    push_dora_with_sidecars
}

main
