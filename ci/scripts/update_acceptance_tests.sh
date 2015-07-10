#!/bin/bash

set -ex

pushd acceptance-tests
commit_hash=$(git rev-parse HEAD)
commit_message=$(git log --oneline | head -n1)
subtree_repo_url=$(git config remote.origin.url)
subtree_repo_branch="master"
popd

git config --global user.email "concourse-bot@starkandwayne.com"
git config --global user.name "Concourse Bot"

pushd rdpg-boshrelease
git checkout master # see http://stackoverflow.com/a/18608538/36170
git subtree pull \
  --prefix src/rdpg-acceptance-tests \
  ${subtree_repo_url} \
  ${subtree_repo_branch} \
  --squash -m "Bump tests: $commit_message"
