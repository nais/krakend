# KrakenD Operator for Kubernetes

Kubernetes operator for installing and managing [KrakenD](https://www.krakend.io/) - an open-source API Gateway - in Kubernetes namespaces.

## Overview

The KrakenD Kubernetes Operator simplifies the deployment and management of the KrakenD API Gateway and its configurations within Kubernetes namespaces.

## Features

- **Automated Deployment**: Install and manage KrakenD API Gateway using custom resources.
- **Configuration Management**: Simplify API endpoint configurations with sane defaults.
- **Custom Resources**: Utilize `Krakend` and `ApiEndpoints` custom resources for deployment and configuration.

## Getting Started

### Prerequisites

- A Kubernetes cluster (local or remote)
- `kubectl` configured to interact with your cluster
- [Helm](https://helm.sh/) installed

### Installation

1. **Install Custom Resource Definitions (CRDs)**:

  ```sh
  kubectl apply -k config/crd/
  ```

2. **Deploy the Operator**:

  ```sh
  make deploy IMG=<your-registry>/krakend:latest
  ```

### Usage

#### Deploying KrakenD

Create a `Krakend` resource to deploy KrakenD in your namespace:

```yaml
apiVersion: krakend.nais.io/v1
kind: Krakend
metadata:
  name: my-namespace
  namespace: my-namespace
spec:
  ingress:
  enabled: true
  className: your-ingress-class
  annotations: {}
  hosts:
    - host: my-namespace.nais.io
      paths:
      - path: /
        pathType: Prefix
  authProviders:
  - name: some-jwt-auth-provider
    alg: RS256
    jwkUrl: https://the-jwk-url
    issuer: https://the-jwt-issuer
  deployment:
  replicaCount: 2
  image:
    tag: 3.0.2
  resources:
    limits:
    cpu: 100m
    memory: 128Mi
    requests:
    cpu: 100m
    memory: 128Mi
```

Apply the resource:

```sh
kubectl apply -f <your-krakend-resource.yaml>
```

#### Configuring API Endpoints

Create an `ApiEndpoints` resource to define your API endpoints:

```yaml
apiVersion: krakend.nais.io/v1
kind: ApiEndpoints
metadata:
  name: app1
  namespace: my-namespace
spec:
  appName: app1
  auth:
  name: some-jwt-auth-provider
  cache: true
  debug: true
  audience:
    - "audience1"
  scopes:
    - "scope1"
  rateLimit:
  maxRate: 10
  clientMaxRate: 0
  strategy: ip
  capacity: 0
  clientCapacity: 0
  endpoints:
  - path: /app1/somesecurestuff
    method: GET
    backendHost: http://app1
    backendPath: /somesecurestuff
  - path: /anotherapp
    method: GET
    backendHost: https://anotherapp.nais.io
    backendPath: /
  openEndpoints:
  - path: /doc
    method: GET
    backendHost: http://app1
    backendPath: /doc
```

Apply the resource:

```sh
kubectl apply -f <your-apiendpoints-resource.yaml>
```

#### Generated KrakenD config

The operator runs KrakenD CE 3.x, from our GAR mirror of the official image (`krakend/Dockerfile`, signed and with SBOM). For each `Krakend` resource it writes the full
`krakend.json` (`version: 4`) to the ConfigMap `<name>-krakend-v3`, built from the `Krakend` resource and all
`ApiEndpoints` in the namespace. Prometheus metrics (OpenTelemetry) are exposed on port 9091.

`spec.deployment.image` is only used if the tag is 3.x or newer, otherwise the operator default image is used.

## Development

### Running Locally

1. **Install Sample Custom Resources**:

  ```sh
  kubectl apply -k config/samples/
  ```

2. **Build and Push Docker Image**:

  ```sh
  make docker-build docker-push IMG=<your-registry>/krakend:latest
  ```

3. **Deploy the Controller**:

  ```sh
  make deploy IMG=<your-registry>/krakend:latest
  ```

### Uninstalling

To remove the CRDs and the controller:

```sh
make uninstall
make undeploy
```

## Contributing

Contributions are welcome! Please refer to the [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

## Additional Resources

- [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)
- [KrakenD Documentation](https://www.krakend.io/docs/)

