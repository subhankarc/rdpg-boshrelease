#!/bin/bash

bosh -n -t ${bosh_target} -u ${bosh_username} -p ${bosh_password} \
  delete deployment ${bosh_deployment_name} --force
