#!/bin/bash -eux

# Run component tests in docker compose defined in features/compose folder
pushd dp-cantabular-xlsx-exporter/features/compose
  COMPONENT_TEST_USE_LOG_FILE=false docker-compose up --abort-on-container-exit
  e=$?
popd

# Cat the component-test output file and remove it so log output can
# be seen in Concourse
pushd dp-cantabular-xlsx-exporter
  cat component-output.txt && rm component-output.txt
popd

# Show message to prevent any confusion by 'ERRO 0' output
echo "please ignore error codes 0, like so: ERRO[xxxx] 0, as error code 0 means that there was no error"

# exit with the same code returned by docker compose
exit $e
