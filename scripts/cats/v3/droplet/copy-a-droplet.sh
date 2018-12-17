#!/bin/sh

set -exu

CF_API_ENDPOINT=$(cf api | grep -i "api endpoint" | awk '{print $3}')
DOMAIN="${CF_API_ENDPOINT:12}"
appName=origapp
destAppName=destapp
spaceName=space
SPACE_GUID=$(cf space $spaceName --guid)

# the test deletes origapp later on, so this 'if true/false' block isn't very useful

if true ; then

# delete app
cf delete -f $appName || true
cf delete -f $destAppName || true

# create app (src-app)
args=$(printf '{"name":"%s", "relationships": {"space": {"data": {"guid": "%s"}}}}' $appName $SPACE_GUID)
APP_GUID=$(cf curl /v3/apps -X POST -d "$args" | tee /dev/tty | jq -r .guid)
echo $APP_GUID

# create destapp
args=$(printf '{"name":"%s", "relationships": {"space": {"data": {"guid": "%s"}}}}' $destAppName $SPACE_GUID)
DEST_APP_GUID=$(cf curl /v3/apps -X POST -d "$args" | tee /dev/tty | jq -r .guid)
echo $DEST_APP_GUID

# load dora for app
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
  "PROCESSING_UPLOAD") echo "build $PACKAGE_GUID is PROCESSING_UPLOAD..." ;;
  *) echo "Unexpected state: $state" ; exit 1 ;;
  esac
  sleep 2
done

# stage app
stageBody="$(printf '{"lifecycle": {"type": "buildpack", "data": {"buildpacks": ["ruby_buildpack"] } }, "package": { "guid" : "%s"}}' $PACKAGE_GUID)"
BUILD_GUID=$(cf curl /v3/builds -X POST -d "$stageBody" | tee /dev/tty | jq -r .guid)
echo $BUILD_GUID

while : ; do
  state=$(cf curl /v3/builds/$BUILD_GUID | jq -r '.state')
  case $state in
  "FAILED") echo "Failed to build the build" ; exit 1 ;;
  "STAGED") break ;;
  "STAGING") echo "build $BUILD_GUID is ${state}..." ;;
  *) echo "Unexpected state: $state" ; exit 1 ;;
  esac

  sleep 5
done
DROPLET_GUID=$(cf curl /v3/builds/$BUILD_GUID | jq -r '.droplet.guid')

else
  APP_GUID=690a2dc1-4fd5-4ec4-91d0-b14f3f52133f
  DEST_APP_GUID=949c7214-eda7-4bd2-a84e-3ad67e65308a
  PACKAGE_GUID=daf14d59-64b5-4865-839b-fe7a4376361d
  DROPLET_GUID=4b2c053f-a63a-4be9-ac13-11f59791ff1e
fi # end if

# copy src-app droplet to dest-app's list of droplets

copyRequestBody="$(printf '{"relationships": {"app": {"data": {"guid":"%s"}}}}' $DEST_APP_GUID)"
COPIED_DROPLET_GUID=$(cf curl /v3/droplets?source_guid=$DROPLET_GUID -X POST -d "$copyRequestBody" | jq -r '.guid')

while : ; do
  state=$(cf curl /v3/droplets/$COPIED_DROPLET_GUID | tee /dev/tty | jq -r '.state')
  case $state in
  "FAILED") echo "Failed to build the build" ; exit 1 ;;
  "STAGED") break ;;
  "COPYING") echo "droplet-copying is ${state}..." ;;
  *) echo "Unexpected state: $state" ; exit 1 ;;
  esac

  sleep 2
done

exit 1

url=$(cf curl -v /v3/apps/$APP_GUID -X DELETE | tee /dev/tty  | grep Location: | sed 's/Location.*capi.land//' | tr -d '\r')
while : ; do
  state=$(cf curl $url | jq -r '.state')
  case $state in
  "FAILED") echo "Failed to stage the package" ; exit 1 ;;
  "COMPLETE") break ;;
  "PROCESSING") echo "deleting app $appName is $state..." ;;
  *) echo "Unexpected state: $state" ; exit 1 ;;
  esac
  sleep 2
done

# set dest-app's current-droplet to the copied-droplet, scale processes to 256MB

appUpdateBody="$(printf '{"data": {"guid": "%s"}}' $COPIED_DROPLET_GUID)"
cf curl /v3/apps/$DEST_APP_GUID/relationships/current_droplet -X PATCH -d "$appUpdateBody"
for ptype in $(cf curl /v3/apps/$DEST_APP_GUID/processes | jq -r '.resources[].type' ) ; do
  cf curl /v3/apps/$DEST_APP_GUID/processes/$ptype/actions/scale -d '{"memory_in_mb":"256"}' -X POST
done

webProcessGUID=$(cf curl /v3/apps/$DEST_APP_GUID/processes?types=web | jq -r .guid)
workerProcessGUID=$(cf curl /v3/apps/$DEST_APP_GUID/processes?types=worker | jq -r .guid)

if [ -z "$webProcessGUID" ] ; then
  echo "empty webProcessGUID" ; exit 1
fi
if [ -z "$workerProcessGUID" ] ; then
  echo "empty workerProcessGUID" ; exit 1
fi

# assign a route to dest-app

cf create-route space $DOMAIN -n $destAppName
ROUTE_GUID=$(cf curl /v2/routes?q=host:$destAppName | jq -r '.resources[].metadata.guid' | head -1)
cf curl /v2/routes/$ROUTE_GUID/apps/$DEST_APP_GUID -X PUT

# start dest-app
cf curl /v3/apps/$DEST_APP_GUID/actions/start -X POST

# verify dest-app works
curl "${destAppName}.${DOMAIN}"
