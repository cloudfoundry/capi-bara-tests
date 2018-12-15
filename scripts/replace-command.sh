#!/bin/sh

set -exu

CF_API_ENDPOINT=$(cf api | grep -i "api endpoint" | awk '{print $3}')
DOMAIN="${CF_API_ENDPOINT:12}"
appName=app2
spaceName=space
SPACE_GUID=$(cf space $spaceName --guid)
envVars='{"foo": "bar"}'

manifestToApply='
applications:
- name: "$appName"
  processes:
  - type: web
    command: manifest-command.sh
'

nullCommandManifest='
applications:
- name: "$appName"
  processes:
  - type: web
    command: null
'

# Start creating things

if false ; then

cf delete -f $appName || true
args=$(printf '{"name":"%s", "relationships": {"space": {"data": {"guid": "%s"}}}, "environment_variables": {"foo":"bar"} }' $appName $SPACE_GUID)

APP_GUID=$(cf curl /v3/apps -X POST -d "$args" | tee /dev/tty | jq -r .guid)
echo $APP_GUID

PACKAGE_GUID=$(cf curl /v3/packages -X POST -d "$(printf '{"relationships": {"app": {"data": {"guid": "%s"}}}, "type": "bits"}' $APP_GUID)" | tee /dev/tty | jq -r .guid)

echo $PACKAGE_GUID

if [ ! -f my-app.zip ] ; then
  D1=$PWD
  cd $HOME/go/src/github.com/cloudfoundry/cf-acceptance-tests/assets/dora
  zip -r $D1/my-app.zip .
  cd $D1
fi

curl -k "$CF_API_ENDPOINT/v3/packages/$PACKAGE_GUID/upload" -F bits=@"my-app.zip" -H "Authorization: $(cf oauth-token | grep bearer)"

while : ; do
  state=$(cf curl /v3/packages/$PACKAGE_GUID | jq -r '.state')
  case $state in
  "FAILED") echo "Failed to stage the package" ; exit 1 ;;
  "READY") break ;;
  "PROCESSING_UPLOAD") echo PROCESSING_UPLOAD... ;;
  *) echo "Unexpected state: $state" ; exit 1 ;;
  esac
  sleep 0.5
done

stageBody="$(printf '{"lifecycle": {"type": "buildpack", "data": {"buildpacks": ["ruby_buildpack"] } }, "package": { "guid" : "%s"}}' $PACKAGE_GUID)"
BUILD_GUID=$(cf curl /v3/builds -X POST -d "$stageBody" | tee /dev/tty | jq -r .guid)
echo $BUILD_GUID

while : ; do
  state=$(cf curl /v3/builds/$BUILD_GUID | jq -r '.state')
  case $state in
  "FAILED") echo "Failed to build the build" ; exit 1 ;;
  "STAGED") break ;;
  "STAGING") echo ${state}... ;;
  *) echo "Unexpected state: $state" ; exit 1 ;;
  esac

  sleep 2
done

DROPLET_GUID=$(cf curl /v3/builds/$BUILD_GUID | jq -r '.droplet.guid')

else
APP_GUID=b1ea50f5-04b8-4050-b1f7-abbd7e12dce6
DROPLET_GUID=8efc962d-61f3-4b56-9c7c-7a10df71ad55

fi  # if true/false

# cf curl /v3/apps/$APP_GUID/relationships/current_droplet -X PATCH -d "$(printf '{"data": {"guid": "%s"}}' "$DROPLET_GUID")"

url=$(cf curl /v3/apps/$APP_GUID/actions/apply_manifest -X POST -H "Content-Type: application/x-yaml" -d "$manifestToApply" -i | tee /dev/tty  | grep Location: | sed 's/Location.*capi.land//' | tr -d '\r')

while : ; do
  state=$(cf curl $url | jq -r '.state')
  case $state in
  "FAILED") echo "Failed to stage the package" ; exit 1 ;;
  "COMPLETE") break ;;
  "PROCESSING") echo $state... ;;
  *) echo "Unexpected state: $state" ; exit 1 ;;
  esac
  sleep 0.5
done

PROCESS_GUID="$(cf curl /v3/apps/$APP_GUID/processes?types=web  | jq -r '.resources[0].guid')"
PROCESS_COMMAND=$(cf curl /v3/processes/$PROCESS_GUID | jq -r '.command')

#echo 'press return to set process to null'
#read x

cf curl /v3/processes/$PROCESS_GUID -X PATCH -d '{"command": null}' -i

echo final process-command:
cf curl /v3/processes/$PROCESS_GUID | jq -r '.command'
