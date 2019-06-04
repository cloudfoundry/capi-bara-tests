#!/bin/sh

set -ex

perl -pi -e 's/(Hi.*Dora.*\#)(\d+)/$1 . (int($2) + 1)/e' dora.rb

CF_API_ENDPOINT=$(cf api | grep -i "api endpoint" | awk '{print $3}')

APP_GUID=$(cf app dora --guid  2>/dev/null)

cf curl /v3/apps/$APP_GUID/features/revisions -X PATCH -d '{"enabled": true }'

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

cf curl /v3/deployments -X POST -d "$(printf '{ "droplet": { "guid": "%s"}, "relationships":{ "app": { "data": { "guid": "%s" }}}, "metadata":{"labels":{"target_completion_rate": "0.6"}}}' $DROPLET_GUID $APP_GUID)"

set +x

for x in {1..10000} ; do curl "dora.${CF_API_ENDPOINT#*.}"; sleep 0.5; done
