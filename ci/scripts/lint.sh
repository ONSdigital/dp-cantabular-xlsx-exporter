#!/bin/bash -eux

cwd=$(pwd)

pushd $cwd/dp-cantabular-xlsx-exporter
  make lint
popd
