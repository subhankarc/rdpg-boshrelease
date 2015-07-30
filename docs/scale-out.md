# Scale Out

Scaling out allows adding new service clusters to the existing deployment. The new requests for user databases will be distributed among the existing and new service clusters. On one hand, scaling out allows user databases span among more servers; on the other hand, it enables RDPG to flexibly increase user database capacity.

## Modify the Manifest

Assume we already have two service clusters and we want to add the third one. The steps to modify ```rdpg-warden-manifest.yml``` are as follows.

Copy and paste service cluster two job block.

Modify name from ```rdpgsc2``` to ```rdpgsc3```:

  ```
    - instances: 2
      name: rdpgsc3
  ```

Modify Network ips to what you use here, in this example we are modifing ```10.244.2.22``` and ```10.244.2.26``` into ```10.244.2.30``` and ```10.244.2.34```.

``` 
    networks:
  - default:
    - dns
    - gateway
    name: rdpg
    static_ips:
    - 10.244.2.30
    - 10.244.2.34
```

Modify the cluster_name for rdpgd_service in properties from ```rdpgsc2``` into ```rdpgsc3```.

```
    rdpgd_service:
      cluster_name: rdpgsc3
```

Modify networks fields as follows.

```
   - cloud_properties:
      name: random
    range: 10.244.2.28/30
    reserved:
    - 10.244.2.29
    static:
    - 10.244.2.30
   - cloud_properties:
      name: random
    range: 10.244.2.32/30
    reserved:
    - 10.244.2.33
    static:
    - 10.244.2.34 
```

Increase resources_pools size accordingly, in this example it is 2 since we add one service cluster.

``` 
    resource_pools:
     - cloud_properties:
       name: random
     name: rdpg
     network: rdpg
     size: 9
```

## Testing 

Run ```bosh vms```, you should be able to see the new service clusters nodes running.

Go to ```10.244.2.2:8500``` in your favorite browser, you should be able to see the new service cluster master node and replica node show up. Also the ```postgres```, ```haproxy``` and ```pgbouncer``` services should have increased number of passing.

Connect to management administrative database, you should be able to see pre-created databases in ```cfsb.instances``` table from the new added service cluster.



