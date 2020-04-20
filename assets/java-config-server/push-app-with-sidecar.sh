#!/usr/bin/env bash

set -ex

DORA_PATH="../java-dora"
DORA_MANIFEST_PATH="../java-config-server/dora_manifest.yml"
JAVA_DORA_JAR="build/libs/java-dora-0.0.1-SNAPSHOT.jar"
JAVA_CONFIG_JAR="build/libs/java-config-server-0.0.1-SNAPSHOT.jar"

function clean() {
    ./gradlew clean
}

function build_config_server() {
    ./gradlew build
    cp build/libs/*.jar "${DORA_PATH}/build/libs"
}

function push_dora_with_sidecars() {
    pushd ${DORA_PATH} > /dev/null
        ./gradlew build
        zip "${JAVA_DORA_JAR}" -u "${JAVA_CONFIG_JAR}"
        cf create-app java-dora
        cf apply-manifest -f "${DORA_MANIFEST_PATH}"
        cf push java-dora -p build/libs/java-dora-0.0.1-SNAPSHOT.jar
    popd > /dev/null
}

function main() {
    clean
    build_config_server
    push_dora_with_sidecars
}

main
