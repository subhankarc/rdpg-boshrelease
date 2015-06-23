#!/bin/bash

_bosh() {
  bosh -n -t ${bosh_target} -u ${bosh_username} -p ${bosh_password} $@
}

set -e

_bosh delete deployment ${bosh_deployment_name} --force || echo "Continuing..."
_bosh create release
set +e
_bosh upload release --rebase || echo "Continuing..."
set -e

./rdpg-dev manifest warden

_bosh deploy
