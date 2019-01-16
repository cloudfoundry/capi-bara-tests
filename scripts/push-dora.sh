#!/bin/sh

set -ex

cd $HOME/go/src/github.com/cloudfoundry/cf-acceptance-tests/assets/dora

perl -pi -e 's/(Hi.*Dora.*\#)(\d+)/$1 . (int($2) + 1)/e' dora.rb

CF_API_ENDPOINT=$(cf api | grep -i "api endpoint" | awk '{print $3}')

APP_NAME=dora
SPACE_GUID=$(cf space `cf target | tail -n 1 | awk '{print $2}'` --guid)
APP_GUID=$(cf curl /v3/apps -X POST -d "$(printf '{"name": "%s", "relationships": {"space": {"data": {"guid": "%s"}}}}' "$APP_NAME" "$SPACE_GUID")" | tee /dev/tty | jq -r .guid)

cf curl -X PATCH /v3/apps/$APP_GUID/features/revisions -d '{"enabled": true}'

PACKAGE_GUID=$(cf curl /v3/packages -X POST -d "$(printf '{"type":"bits", "relationships": {"app": {"data": {"guid": "%s"}}}}' "$APP_GUID")" | tee /dev/tty | jq -r .guid)

zip -r my-app-v2.zip * # ( -x *.zip if there are old zip files hanging around)

curl -k "$CF_API_ENDPOINT/v3/packages/$PACKAGE_GUID/upload" -F bits=@"my-app-v2.zip" -H "Authorization: $(cf oauth-token | grep bearer)"

# Wait for the package to go from PROCESSING_UPLOAD to READY
while : ; do state=$(cf curl /v3/packages/$PACKAGE_GUID | tee /dev/tty | jq -r .state) ; if [ "$state" == "READY" ] ; then break; fi; sleep 5 ; done

rm my-app-v2.zip  # clean up after yaself

BUILD_GUID=$(cf curl /v3/builds -X POST -d "$(printf '{ "package": { "guid": "%s" }}' "$PACKAGE_GUID")" | tee /dev/tty | jq -r .guid)

# Wait for staging to complete

while : ; do state=$(cf curl /v3/builds/$BUILD_GUID | tee /dev/tty | jq -r .state); if [ "$state" != "STAGING" ] ; then break; fi; sleep 2 ; done

DROPLET_GUID=$(cf curl /v3/builds/$BUILD_GUID | jq -r '.droplet.guid')

cf curl /v3/apps/$APP_GUID/relationships/current_droplet -X PATCH -d "$(printf '{"data": {"guid": "%s"}}' "$DROPLET_GUID")"

echo "revisions before starting"
cf curl /v3/apps/$APP_GUID/revisions | jq .
read x

cf curl -X POST /v3/apps/$APP_GUID/actions/start


echo "revisions after starting"
cf curl /v3/apps/$APP_GUID/revisions | jq .
read x

perl -pi -e 's/(Hi.*Dora.*\#)(\d+)/$1 . (int($2) + 1)/e' dora.rb


PACKAGE2_GUID=$(cf curl /v3/packages -X POST -d "$(printf '{"type":"bits", "relationships": {"app": {"data": {"guid": "%s"}}}}' "$APP_GUID")" | tee /dev/tty | jq -r .guid)

zip -r my-app-v2.zip * # ( -x *.zip if there are old zip files hanging around)

curl -k "$CF_API_ENDPOINT/v3/packages/$PACKAGE2_GUID/upload" -F bits=@"my-app-v2.zip" -H "Authorization: $(cf oauth-token | grep bearer)"

# Wait for the package to go from PROCESSING_UPLOAD to READY
while : ; do state=$(cf curl /v3/packages/$PACKAGE2_GUID | tee /dev/tty | jq -r .state) ; if [ "$state" == "READY" ] ; then break; fi; sleep 5 ; done

rm my-app-v2.zip  # clean up after yaself

BUILD2_GUID=$(cf curl /v3/builds -X POST -d "$(printf '{ "package": { "guid": "%s" }}' "$PACKAGE2_GUID")" | tee /dev/tty | jq -r .guid)

# Wait for staging to complete

while : ; do state=$(cf curl /v3/builds/$BUILD2_GUID | tee /dev/tty | jq -r .state); if [ "$state" != "STAGING" ] ; then break; fi; sleep 2 ; done

DROPLET2_GUID=$(cf curl /v3/builds/$BUILD2_GUID | jq -r '.droplet.guid')



cf curl /v3/apps/$APP_GUID/relationships/current_droplet -X PATCH -d "$(printf '{"data": {"guid": "%s"}}' "$DROPLET2_GUID")"

echo "revisions before starting droplet2"
cf curl /v3/apps/$APP_GUID/revisions | jq .
read x

cf curl -X POST /v3/apps/$APP_GUID/actions/start


echo "revisions after starting droplet2"
cf curl /v3/apps/$APP_GUID/revisions | jq .

