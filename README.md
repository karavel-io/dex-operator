# Dex Operator

> :warning: **THIS SOFTWARE IS ALPHA** Do not use in production, expect bugs and other issues.  
> On the other hand, you are very much encouraged to try it in test environments, and report any bug or DX issues to the
> [issue tracker](https://github.com/karavel-io/dex-operator/issues) if you want to help us improve the project!

A [Kubernetes Operator] for [Dex]. It can deploy and manage Dex instances as well as
declaratively configure client applications.

## Deploy

TODO

### Params

```bash
Usage of /manager:
  -health-probe-bind-address string
    	The address the probe endpoint binds to. (default ":8081")
  -kubeconfig string
    	Paths to a kubeconfig. Only required if out-of-cluster.
  -leader-elect
    	Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.
  -metrics-bind-address string
    	The address the metric endpoint binds to. (default ":8080")
  -zap-devel
    	Development Mode defaults(encoder=consoleEncoder,logLevel=Debug,stackTraceLevel=Warn). Production Mode defaults(encoder=jsonEncoder,logLevel=Info,stackTraceLevel=Error) (default true)
  -zap-encoder value
    	Zap log encoding (one of 'json' or 'console')
  -zap-log-level value
    	Zap Level to configure the verbosity of logging. Can be one of 'debug', 'info', 'error', or any integer value > 0 which corresponds to custom debug levels of increasing verbosity
  -zap-stacktrace-level value
    	Zap Level at and above which stacktraces are captured (one of 'info', 'error').
```

## Dex

Instances are managed using the `Dex` resources. `Dex` is a [Custom Resource] that is used to configure
a Dex server. The operator will create the necessary objects (`Deployment`, `Service`, `ConfigMap` etc) to
deploy and configure the Dex instance based on the configuration provided in the `Dex` object.

Here is an example `Dex` object that configures a mock connector. 
You can find more examples in the [config/samples](./config/samples) folder.

```yaml
apiVersion: dex.karavel.io/v1alpha1
kind: Dex
metadata:
  name: dex
  namespace: dex
spec:
  publicURL: https://dex.example.com
  replicas: 1
  connectors:
    - type: mockCallback
      id: mock
      name: Example
```

### Custom Image

By default, the operator will deploy the latest official Dex container image available at `quay.io/dexidp/dex:latest`.
If you wish to pin a specific image tag, or wish to use a different Dex image, you can override it by setting the `image` field.

**WARNING** Using an image other than `quay.io/dexidp/dex` is unsupported.

```yaml
apiVersion: dex.karavel.io/v1alpha1
kind: Dex
metadata:
  name: dex
  namespace: dex
spec:
  # rest of the configuration omitted
  image: quay.io/dexidp/dex:v2.26.0
```

### Environment Variables

Dex supports [reading connectors config values from environment variables](https://github.com/dexidp/dex/blob/master/examples/config-dev.yaml#L123).
To inject those values into the container from Kubernetes `ConfigMap` or `Secret` objects use the `envFrom` field on your `Dex` manifest.

```yaml
apiVersion: dex.karavel.io/v1alpha1
kind: Dex
metadata:
  name: dex
  namespace: dex
spec:
  # rest of the configuration omitted
  envFrom:
    - configMapRef:
        name: dex-configmap
    - secretRef:
        name: dex-secret
```

### Exposing instances

#### Using Ingresses

By default the `Dex` object will generate an `Ingress` resource to expose the public URL and allow HTTP
traffic to flow in. The gRPC API and metrics endpoint will NOT be exposed for security reasons.

You can customize how the `Ingress` resource is configured via the `ingress` field.

```yaml
apiVersion: dex.karavel.io/v1alpha1
kind: Dex
metadata:
  name: dex
  namespace: dex
spec:
  # rest of the configuration omitted
  ingress:
    enabled: true
    labels:
      example-label: value
    annotations:
      example-annotation: value
    tlsEnabled: true
    tlsSecretName: custom-secret
```

### Scaling instances

The number of replicas can be tweaked by changing the `replicas` field. By default it is set to `1`.

The `Dex` object supports the [scale subresource] to be able to scale the underlying `Deployment` object
using `kubectl`. This also makes it compatible with automated scaling solutions like the [Horizontal Pod Autoscaler].

`kubectl scale dex my-dex-instance --namespace dex --replicas 5`

## DexClient

`DexClient` objects are OAuth 2.0 clients that are registered on a Dex instance. Applications typically include 
these objects in their deployment manifests.

A `DexClient` targets a specific `Dex` instance via the `instanceRef` field. If the two objects live in the same
namespace then the `namespace` field can be omitted.

A `Secret` named `dex-$NAME-credentials` will be created in the same namespace as the `DexClient` object, where
`$NAME` is the `DexClient` `metadata.name` field.

It will contain two keys, `clientId` and `clientSecret`. For client marked `public: true` the `clientSecret` field
will be absent. These are the OAuth 2.0 client_id and client_secret values for the newly created client.

Applications should be reading these values directly from the `Secret` (i.e. by mounting them into environment variables)
instead of hard-coding them, as they may be rotated and refreshed by the operator.

```yaml
apiVersion: dex.karavel.io/v1alpha1
kind: DexClient
metadata:
  name: example
  namespace: default
spec:
  name: Example
  public: false
  redirectUris:
    - https://example.com/oauth/callback
  instanceRef:
    name: dex
    namespace: dex
```

The generated `Secret` will look like this:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: dex-example-credentials
  namespace: default
type: Opaque
data:
  clientId: ZGVmYXVsdC1leGFtcGxl
  clientSecret: d2hhdCBhcmUgeW91IGxvb2tpbmcgZm9yIGV4YWN0bHk/IDsp
```
## Local build

A local environment to test it is provided using [Kind].

```bash
make kind-start   # start the local cluster

make install-cert-manager      # install cert-manager

make manifests docker-load  # build and load the docker image into Kind

make deploy       # deploy the operator
```

Sample resources can be found in [config/samples](./config/samples)

`kubectl apply -f config/samples`

[Kubernetes Operator]: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/
[Dex]: https://dexidp.io
[Kind]: https://kind.sigs.k8s.io/
[Custom Resource]: https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/
[scale subresource]: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#scale-subresource
[Horizontal Pod Autoscaler]: https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/
