#!/bin/bash

set -ex

pushd acceptance-tests
commit_hash=$(git rev-parse HEAD)
commit_message=$(git log --oneline | head -n1)
subtree_repo_url=$(git config remote.origin.url)
subtree_repo_branch="master"
popd

pushd rdpg-boshrelease
git subtree pull \
  --prefix src/rdpg-acceptance-tests \
  ${subtree_repo_url} \
  ${subtree_repo_branch} \
  --squash


echo "Checking for changes in $(pwd)..."
if [[ "$(git status -s)X" != "X" ]]; then
  git add . --all
  git commit -m "Bump tests: $commit_message"
fi
