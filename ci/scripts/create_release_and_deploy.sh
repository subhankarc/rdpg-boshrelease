#!/bin/bash

# change to root of bosh release
DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd $DIR/../../..

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

_bosh() {
  bosh -n $@
}

set -e

_bosh delete deployment ${bosh_deployment_name} --force || echo "Continuing..."
_bosh create release
set +e
_bosh upload release --rebase || echo "Continuing..."
set -e

echo "running: rdpg-dev manifest warden"
DEBUG=true ./rdpg-dev manifest warden

_bosh deploy
