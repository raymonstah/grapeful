#!/bin/bash

set -eu

ROOT="$(cd $(dirname $0)/.. && pwd)"

(cd ${ROOT}/backend; go test -cover ./...)
