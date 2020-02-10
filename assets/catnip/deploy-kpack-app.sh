#!/usr/bin/env bash
set -x
guid=$(cf v3-create-package $1 | grep "package guid" | awk '{print $NF}')
echo "created package"
echo $guid

cf curl /v3/builds -X POST -d "{\"package\": {\"guid\":\"$guid\"},  \"lifecycle\": { \"type\": \"kpack\", \"data\": {} } }"
