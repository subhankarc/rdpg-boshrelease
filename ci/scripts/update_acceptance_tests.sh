#!/bin/bash

set -ex

pushd acceptance-tests
commit_hash=$(git rev-parse HEAD)
commit_message=$(git log --oneline | head -n1)
popd

pushd rdpg-boshrelease
git submodule update --init
cd src/rdpg-acceptance-tests
git pull origin master

echo "Checking for changes in $(pwd)..."
git checkout $commit_hash
cd ../..
if [[ "$(git status -s)X" != "X" ]]; then
  git add . --all
  git commit -m "Bump tests: $commit_message"
fi
