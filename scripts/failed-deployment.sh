#!/bin/sh

set -ex

# Important: run `cf push dora -i 10` before running this script

cd "$(dirname $0)"/../assets

CF_API_ENDPOINT=$(cf api | grep -i "api endpoint" | awk '{print $3}')
APP_GUID=$(cf app dora --guid  2>/dev/null)

PACKAGE_GUID=$(cf curl /v3/packages -X POST -d "$(printf '{"type":"bits", "relationships": {"app": {"data": {"guid": "%s"}}}}' "$APP_GUID")" | tee /dev/tty | jq -r .guid)

pushd bad-dora
  zip -r ../my-app-v2.zip * # ( -x *.zip if there are old zip files hanging around)
popd

curl -k "$CF_API_ENDPOINT/v3/packages/$PACKAGE_GUID/upload" -F bits=@"my-app-v2.zip" -H "Authorization: $(cf oauth-token | grep bearer)"

# Wait for the package to go from PROCESSING_UPLOAD to READY
while : ; do state=$(cf curl /v3/packages/$PACKAGE_GUID | tee /dev/tty | jq -r .state) ; if [ "$state" == "READY" ] ; then break; fi; sleep 5 ; done

rm my-app-v2.zip  # clean up after yaself

BUILD_GUID=$(cf curl /v3/builds -X POST -d "$(printf '{ "package": { "guid": "%s" }}' "$PACKAGE_GUID")" | tee /dev/tty | jq -r .guid)

# Wait for staging to complete

while :;  do
   state=$(cf curl /v3/builds/$BUILD_GUID | tee /dev/tty | jq -r .state)
   if [ "$state" != "STAGING" ] ; then
      break
   fi
   sleep 2
done

DROPLET_GUID=$(cf curl /v3/builds/$BUILD_GUID | jq -r '.droplet.guid')

if [[ $DROPLET_GUID == "null" ]] ; then
   echo "Failed to build"
   exit 1
fi

cf curl /v3/apps/$APP_GUID/relationships/current_droplet -X PATCH -d "$(printf '{"data": {"guid": "%s"}}' "$DROPLET_GUID")"

DEPLOYMENT_GUID=$(cf curl /v3/deployments -d "$(printf '{ "relationships":{ "app": { "data": { "guid": "%s" }}}}' $APP_GUID)" | tee /dev/tty | jq -r .guid)

timeout 130s watch "cf curl /v3/deployments/$DEPLOYMENT_GUID | jq -r .state"

PREVIOUS_DROPLET_GUID=$(cf curl /v3/apps/$APPGUID/droplets?states=STAGED | jq -r '.resources[-2].guid')
# Hopefully this droplet is good
cf curl /v3/apps/$APP_GUID/relationships/current_droplet -X PATCH -d "$(printf '{"data": {"guid": "%s"}}' "$PREVIOUS_DROPLET_GUID")"

NEW_DEPLOYMENT_GUID=$(cf curl /v3/deployments -d "$(printf '{ "relationships":{ "app": { "data": { "guid": "%s" }}}}' $APP_GUID)" | tee /dev/tty | jq -r .guid)

timeout 130s watch bash -c "echo DEPLOYMENT_GUID; cf curl /v3/deployments/$DEPLOYMENT_GUID | jq -r .state ; "echo NEW_DEPLOYMENT_GUID; cf curl /v3/deployments/$NEW_DEPLOYMENT_GUID | jq -r .state"

