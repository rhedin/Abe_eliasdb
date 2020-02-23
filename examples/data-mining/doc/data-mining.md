EliasDB Data Mining Example
==
This example demonstrates a more complex application which uses the cluster feature and EQL and GraphQL for data queries.

The idea of the application is to provide a platform for datamining with 3 components for presentation, collection and storage of data.

The tutorial assumes you have downloaded EliasDB, extracted and build it.
It also assumes that you have a running docker environment with the `docker` and `docker-compose` commands being available. This tutorial will only work in unix-like environments.

For this tutorial please execute "build.sh" in the subdirectory: examples/data-mining.




After starting EliasDB point your browser to:
```
https://localhost:9090
```

The generated default key and certificate for https are self-signed which should give a security warning in the browser. After accepting you should see a login prompt. Enter the credentials for the default user elias:
```
Username: elias
Password: elias
```

The browser should display the chat application after clicking `Login`. Open a second window and write some chat messages. You can see that both windows update immediately. This is done with GraphQL subscriptions.
