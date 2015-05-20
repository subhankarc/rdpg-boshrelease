# rdpg Project Specific Notes

## Release Directory

mkdir stemcells # we are going to download the stemcell locally
curl -sL https://bosh.io/d/stemcells/bosh-warden-boshlite-centos-go_agent  -o stemcells/bosh-warden-boshlite-centos-go_agent.tgz

./prepare stemcell warden # uploads the stemcell
./prepare manifest warden # only first time, also does the stemcell
./prepare dev # generates a new release and uploads it
bosh -n deploy

SSH To a BOSH Deployed VM: bosh ssh {vm name} {vm index}
bosh ssh rdpg 0
bosh ssh rdpg 1
bosh ssh rdpg 2
bosh ssh rdpg 3
bosh ssh rdpg 4

Remove old VM ssh keys when destroyed
vim ~/.ssh/known_hosts # delete the 10.* lines at the bottom

## On a VM

head -n 100 /var/vcap/sys/log/pgbdr/pgbdr.log # first 100 lines
tail -n 100 /var/vcap/sys/log/pgbdr/pgbdr.log # last 100 lines
tail -f /var/vcap/sys/log/pgbdr/pgbdr.log # follow
/var/vcap/bosh/bin/monit restart consul # restart consul : {start|stop|restart}

psql -U vcap pgbdr
consul members

## In PostgreSQL
 -- Display Extensions loaded in the current database
\dx
-- See all connected nodes in master-master replication for the current database
select * from bdr.bdr_nodes; 

## Consul
consul members # show all members in the consul cluster

## Tuning based on machine sizing and usage

For a reference starting point read the 
[pgtune blog post](http://leopard.in.ua/2014/03/24/pgtune-for-postgresql/) 
and use the [website tool](http://pgtune.leopard.in.ua)

