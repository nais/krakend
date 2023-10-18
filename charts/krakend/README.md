# krakend

This is a wrapper chart for [krakend](https://www.krakend.io/) which is used by the krakend-operator to install krakend instances in a namespace.

## Upgrading

Bump the version of the krakend dependency in `Chart.yaml` and run `helm dependency update .` from within this folder.

Run the following command to make sure the operator is using the new version:

```bash
make krakend-package
```

## Changing the krakend chart configuration

Do your updates in the values.yaml/templates and run the following command from the root of this repo:

```bash
make krakend-package
```