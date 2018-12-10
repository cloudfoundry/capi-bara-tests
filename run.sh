#!/bin/sh

cf push -v BARA-1-APP-417c453c5c304989 -b ruby_buildpack -m 128MB -i 2 -p assets/dora -d legend-chill.capi.land
