First install multipass.

You'll need to decide which network adapter to use for the bridge.

```multipass networks```

```multipass set local.bridged-network=eth0```

``` multipass launch -vv -n cups --cloud-init ./cloud-config.yaml --network bridged```

You can now access the cups web interface from http://{bridgedip}:631
It should have full IPP/Bonjour network discovery