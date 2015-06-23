#!/bin/bash

set -e

# change to root of bosh release
DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd $DIR/../..

cat > ~/.bosh_config << EOF
---
aliases:
  target:
    bosh-lite: ${bosh_target}
auth:
  ${bosh_target}:
    username: ${bosh_username}
    password: ${bosh_password}
EOF
bosh target ${bosh_target}

cat > config/private.yml << EOF
---
blobstore:
  s3:
    access_key_id: ${aws_access_key_id}
    secret_access_key: ${aws_secret_access_key}
EOF

_bosh() {
  bosh -n $@
}

set -e

VERSION=$(cat ../version/number)
if [ -z "$VERSION" ]; then
  echo "missing version number"
  exit 1
fi

git config --global user.email "ci@localhost"
git config --global user.name "CI Bot"

git merge --no-edit ${promotion_branch}

bosh target ${BOSH_TARGET}

bosh -n create release --final --with-tarball --version "$VERSION"

git add -A
git commit -m "release v${VERSION}"