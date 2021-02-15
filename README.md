# Dex Operator

A [Kubernetes Operator] for [Dex]. It can deploy and manage Dex instances as well as
declaratively configure client applications.

## Local build

A local environment to test it is provided using [Kind].

```bash
make kind-start   # start the local cluster

make install      # install the CRDs and cert-manager

make docker-load  # build and load the docker image into Kind

make deploy       # deploy the operator
```

Sample resources can be found in [config/samples](./config/samples)

`kubectl apply -f config/samples`

[Kubernetes Operator]: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/
[Dex]: https://dexidp.io
[Kind]: https://kind.sigs.k8s.io/
