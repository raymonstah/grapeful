#!/bin/bash

set -eu

GOPATH=${GOPATH:=${HOME}/go}
ROOT="$(cd $(dirname $0)/.. && pwd)/backend"

if [[ ! -f ${GOPATH}/bin/golint ]] ; then
  (cd ${ROOT} && go get golang.org/x/lint/golint)
fi

cd ${ROOT} && ${GOPATH}/bin/golint -set_exit_status ./...
