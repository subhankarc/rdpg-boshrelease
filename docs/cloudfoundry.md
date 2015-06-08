# Cloud Foundry

First we make sure we are logged into an org/space in CF, for develoment targeting
bosh-lite,
```sh
cf api --skip-ssl-validation https://api.10.244.0.34.xip.io
cf auth admin admin
cf create-org $USER
cf target -o $USER
cf create-space rdpg
cf target -s rdpg
```

We now register the rdpg service broker, this example is in development environment,
```sh
CF_TRACE=true cf create-service-broker rdpg cfadmin cfadmin http://10.244.2.2:8888
```

Next we allow access to the new service,
```sh
cf enable-service-access rdpg -o $USER
cf service-access
cf marketplace
```

Now we can create a service instance,
```sh
CF_TRACE=true cf create-service rdpg small rdpg-test-1
cf services
cf bind --help
cf bind-service
```

Now we deploy an application to our cloud foundry for testing,
```sh
git clone https://github.com/wayneeseguin/cf-rdpg-app
cd cf-rdpg-app
cf push cf-rdpg
```

Now we bind the application to the service we created above,
```sh
cf bind-service cf-rdpg rdpg-test-1
cf restage cf-rdpg-app // App should try to connect to database now.
cf logs cf-rdpg-app --recent
```

Visit [The UI in the browser](http://cf-rdpg.10.244.0.34.xip.io) and examine
the status.

