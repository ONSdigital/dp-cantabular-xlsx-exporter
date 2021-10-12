#!/bin/bash -eux

pushd dp-cantabular-xlsx-exporter
  make build
  cp build/dp-cantabular-xlsx-exporter Dockerfile.concourse ../build
popd
