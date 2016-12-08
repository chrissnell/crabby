#!/bin/bash -e

if [ -z "${CRABBY_CONFIG}" ]; then
  echo The env var CRABBY_CONFIG needs to be defined. 
  echo This variable points crabby towards its config file.
  echo This image accepts a volume, /config, that you can
  echo use for passing in a config file externally.
  echo Exiting...
  exit 1
fi

if [ -n ${CRABBY_START_DELAY} ]; then
  echo "Delaying Crabby startup by ${CRABBY_START_DELAY} seconds..."
  sleep ${CRABBY_START_DELAY}
fi

# Use gosu to drop privileges
exec gosu nobody /crabby -config=$CRABBY_CONFIG
