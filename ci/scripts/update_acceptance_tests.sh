#!/bin/bash

set -ex

if [[ "${subtree_repo}X" == "X" ]]; then
  echo "REQUIRE: \$subtree_repo to match to src/xyz in bosh release"
  exit 1
fi

pushd ${subtree_repo}
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
  --prefix src/${subtree_repo} \
  ${subtree_repo_url} \
  ${subtree_repo_branch} \
  --squash -m "Bump tests: $commit_message"
