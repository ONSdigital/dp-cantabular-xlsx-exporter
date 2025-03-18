#!/bin/bash -eux

cwd=$(pwd)

pushd $cwd/dp-cantabular-xlsx-exporter
  go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.6
  make lint
popd
