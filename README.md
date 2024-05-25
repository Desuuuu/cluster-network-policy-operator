# ClusterNetworkPolicy Operator

## Overview

The operator watches `ClusterNetworkPolicy` resources, which are scoped to the
cluster, and creates corresponding `NetworkPolicy` resources in the configured
namespaces. The `NetworkPolicy` resources are kept in-sync by the operator, any
manual change will be overwritten.

In case of a conflict with a `NetworkPolicy` that is not managed by the
operator, it is left as-is and an error is logged. This
behavior can be modified by setting the `networking.desuuuu.com/conflict-policy`
annotation to `replace` on the `NetworkPolicy`.

By default, the operator is configured to ignore its own namespace as well as
`kube-*` namespaces, meaning it will never execute any operation in these
namespaces. This is configurable through CLI arguments.

## Installation

### Using Helm

Installing using Helm is documented in the [Helm chart README](helm/README.md).

### Using kubectl

```
kubectl create namespace cluster-network-policy-operator

kubectl apply -f https://github.com/Desuuuu/cluster-network-policy-operator/releases/latest/download/networking.desuuuu.com_clusternetworkpolicies.yaml

kubectl apply -f https://github.com/Desuuuu/cluster-network-policy-operator/releases/latest/download/cluster-network-policy-operator.yaml
```

## Usage

```yaml
apiVersion: networking.desuuuu.com/v1
kind: ClusterNetworkPolicy
metadata:
  name: my-network-policy
spec:
  labels:
    my-label: value
  annotations:
    my-annotation: value
  namespaceSelector:
    matchLabels:
      namespace-label: value
  podSelector:
    matchLabels:
      role: db
  policyTypes:
  - Egress
  egress:
  - to:
    - ipBlock:
        cidr: "10.0.0.0/24"
    ports:
    - protocol: TCP
      port: 5978
```

The `spec` field of `ClusterNetworkPolicy` mirrors the `spec` field of
`NetworkPolicy`, with the addition of the following optional fields:

* `labels` - Labels to apply to the `NetworkPolicy` resources.
* `annotations` - Annotations to apply to the `NetworkPolicy` resources.
* `namespaceSelector` - Label selector to further restrict in which namespaces
the `NetworkPolicy` resources are created.

Please note that `namespaceSelector` cannot be used to target a namespace that
is ignored by the operator.
