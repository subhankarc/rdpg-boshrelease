# Acceptance Tests

## Overview
Acceptance tests help to assure that recent changes result in rdpg clusters are in a good state.  They are also used by the Concourse build pipeline to create releases for deployments to other deployment pipelines.  The tests are written for the Ginkgo spec and thus the order of execution is random.

## Running Acceptance Tests
Make any changes you would like to rdpg-boshrelease/src/rdpg-acceptance-tests.  From the root folder of rdpg-boshrelease run the following:

```bash
bosh create release --force && bosh upload release
bosh -n deploy
bosh run errand acceptance_tests
```

Assuming the errand ran successfully, the tail of the output should appear similarly to:
```
Errand `acceptance_tests' completed successfully (exit code 0)
```

If the errand wasn't successful, scroll up in the output and start reviewing which tests passed and which failed.

## Current Tests
The tests are defined in rdpg-boshrelease/src/rdpg-acceptance-tests/rdpg-service/* folders with each folder representing logical groups of tests:
 - broker: contains Service Broker availability tests
 - postgres: validates schemas and tables in the rdpg database on each node
 - postgres-consul: checks warden deployments that the correct number of nodes are available

### broker
 - Prompts for Basic Auth creds when they aren't provided
 - Does not accept bad Basic Auth creds
 - Accepts valid Basic Auth creds

### postgres
 - **Check Schemas Exist** - This test validates that all nodes in all clusters have the following schemas created in the rdpg database: `bdr, rdpg, cfsb, tasks, backups, metrics, audit`. If any are missing, the bootstrapping process for the rdpg daemon was not successful.  Look at the logs at `/var/vcap/sys/log/rdpgd-{manager,service}/rdpg-{manager,service}.log` and search for errors.
 - **Check cfsb Tables Exist** - This test validates that `cfsb.services`, `cfsb.plans`, `cfsb.instances`, `cfsb.bindings`, and `cfsb.credentials` tables exist in the rdgp database for every management and service cluster node.
 - **Check rdpg Tables Exist** - This test validates that `rdpg.confi`g, `rdpg.consul_watch_notifications` and `rdpg.events` tables exist in the rdgp database for every management and service cluster node.
 - **Check tasks Tables Exist** - This test validates that `tasks.tasks` and `tasks.schedules` tables exist on every management and service cluster node in the rdpg database
 - **Check Instance Counts** - Every service cluster is supposed to pre-allocate bdr replicated databases and report the existence of these databases to the management cluster.  Meta information about each of thee databases is stored in the rdpg database in `cfsb.instances`. This check validates that each service cluster has created a default minimum (20) of user databases *(hint: user databases names all start with 'd')*, that all nodes in a cluster have the same number of databases and finally that the management cluster matches the sum of all the service clusters' available user databases. Note that when this test is run immediately following a new deployment it may fail the test until all of the databases have been created the first time.  Wait a few minutes and run the test again and only then if the failure persists should you be worried.
 - **Check Scheduled Tasks Exist** - Tests that each service cluster has at least 3 scheduled tasks (the default) with a role of All or Service and that each cluster in the node has the same number of active scheduled tasks.  For the management cluster there are at least 4 default tasks and the count is compared across all nodes in the cluster.
 - **Check for Missed Scheduled Tasks** - Checks for any enabled task which is eligible to be scheduled has been skipped for more than twice the duration.  This validates jobs are being rescheduled correctly and are firing.
 - **Check for databases known to cfsb.instances but don't exist** - These are databases which do not currently or never have existed within postgres.  If all nodes in a service cluster report that a particular database fails to exist either the database was deleted and the entry in cfsb.instances was not updated correctly to denote it's retirement or an unknown bug during the initial database creation. If at least one node in the service cluster has the database created but the other nodes did not, something failed with the bdr database join function in the "PrecreateDatabases" scheduled task for that service cluster.
 - **Check for databases which exist and aren't known to cfsb.instances** - These are databases which aren't being managed by the rdpg daemon (but likely should be).  This could be the result of the database being restored manually from another service cluster but not registered with cfsb.instances.  Every effort should be made to determine if the database is a user database and added back to cfsb.instances so that scheduled database maintenance can be performed on it (including backups).

### postgres-consul
 - **Check Node Counts** - For deployments of rdpg to warden, the deployment manifest defines there to be 1 management cluster with 3 nodes, and two service clusters each with 2 nodes.  When consul is queried for services matching "rdpgmc, rdpgsc1, rdpgsc2" the number of nodes returned are compared against the good known values and also validates that a connection to consul can be made.
