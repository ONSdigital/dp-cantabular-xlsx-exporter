#!/bin/bash -eux

export cwd=$(pwd)

pushd $cwd/dp-cantabular-xlsx-exporter
  make audit
popd