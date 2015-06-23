#!/bin/bash

bosh -t ${bosh_target} -u ${bosh_username} -p ${bosh_password} \
  deployments
