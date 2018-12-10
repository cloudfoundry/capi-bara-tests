#!/bin/sh

set -x

NUM_INSTANCES=1
cf delete -f dora
cf push dora -b ruby_buildpack -p ../cf-acceptance-tests/assets/dora/ -m 128MB -i $NUM_INSTANCES -d legend-chill.capi.land -v
curl dora.legend-chill.capi.land
cf push dora -b staticfile_buildpack -p ../cf-acceptance-tests/assets/staticfile/ -m 128MB -i $NUM_INSTANCES -d legend-chill.capi.land -v
for x in {1..10} ; do
  curl dora.legend-chill.capi.land
  sleep 1
done
