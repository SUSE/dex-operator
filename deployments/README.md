# `dex-operator` deployment

This folder contains a all-in-one manifest file that you can use in order to deploy the `dex`
operator in your cluster.

You can deploy it in your cluster with the following command:

```
# kubectl apply -f https://raw.githubusercontent.com/kubic-project/dex-operator/master/deployments/dex-operator-full.yaml
```

This all-in-one manifest will create the `dex-controller` service account inside the `kube-system`
namespace, the `dex-operator` related CRD's and the dex operator manager as a stateful set.

## Updating the `dex-operator` all-in-one manifest

If you have performed any changes that could have visibility in the all-in-one manifest, you need
to regenerate it. In order to do this, you have to run:

```
# make manifests
```
