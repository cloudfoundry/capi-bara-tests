#!/bin/sh

set -x
TARGET=almond-hide

NUM_INSTANCES=1
cf delete -f dora
cf push dora -b ruby_buildpack -p ../cf-acceptance-tests/assets/dora/ -m 128MB -i $NUM_INSTANCES -d ${TARGET}.capi.land -v
curl dora.${TARGET}.capi.land
cf push dora -b staticfile_buildpack -p ../cf-acceptance-tests/assets/staticfile/ -m 128MB -i $NUM_INSTANCES -d ${TARGET}.capi.land -v
for x in {1..10} ; do
  curl dora.${TARGET}.capi.land
  sleep 1
done
