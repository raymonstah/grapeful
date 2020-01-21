#!/bin/bash

set -eu

ROOT="$(cd $(dirname $0)/.. && pwd)"
TARGET="${ROOT}/target"

echo "creating zip files..."
echo "#"

# create target directory if does not exist
mkdir -p ${TARGET}

# iterate all subdirectories of functions directory and create zip files from Go source code
# add to target directory
if [[ -d ${ROOT}/backend/functions ]] ; then
  for dir in "${ROOT}"/backend/functions/*
  do
    fn=$(basename "${dir}")
    if [ "$(ls -A $dir)" ]; then
      echo "#"
      echo "zipping... " $fn
      # build function
      (cd "${TARGET}"; GOOS=linux go build -o "${fn}" "${ROOT}/backend/functions/${fn}"/*.go)
      # zip function
      (cd "${TARGET}"; zip -r "${TARGET}/${fn}.zip" "${fn}")
      # remove executable
      (cd "${TARGET}"; rm "${TARGET}/${fn}")
    else
     echo "skipping folder $dir, ...empty"
    fi
  done
fi