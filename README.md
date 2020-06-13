# OAM Controllers

> **👷 WARNNING:** This repo is designed to be used by **OAM workloads/traits/scopes developers.**

## What's inside?

This repo contains Kubernetes controllers for core OAM [workloads](https://github.com/oam-dev/spec/blob/master/3.workload.md#core-workload)/[traits](https://github.com/oam-dev/spec/blob/master/6.traits.md#core-traits)/scopes. 

Workloads
- [ContainerizedWorkload](https://github.com/crossplane/addon-oam-kubernetes-local/tree/79a8c2e5695a757aa06247058912b4354e1c6d09/pkg/controller/core/workloads/containerizedworkload)

Traits
- [ManualScalerTrait](https://github.com/crossplane/addon-oam-kubernetes-local/tree/79a8c2e5695a757aa06247058912b4354e1c6d09/pkg/controller/core/traits/manualscalertrait)

## Prerequisites

- Kubernetes v1.16+
- Helm 3
- [Crossplane](https://github.com/crossplane/crossplane) v0.11+ installed

## (Optional) In case you don't know how to install Crossplane

Here's an easy three steps version:
```console
kubectl create namespace crossplane-system
helm repo add crossplane-alpha https://charts.crossplane.io/alpha
helm install crossplane --namespace crossplane-system crossplane-alpha/crossplane
```

Please feel free to ignore this warning which is caused by helm v2 compatibility issue:
```
manifest_sorter.go:192: info: skipping unknown hook: "crd-install"
```

## Install OAM controllers

#### Clone this repo

```console
git clone git@github.com:crossplane/addon-oam-kubernetes-local.git
cd ./addon-oam-kubernetes-local
```

#### (Optional) Enable webhook and install cert manager

The OAM `ManualScalerTrait` controller includes a sample webhook to [validate the manual scalar trait](https://github.com/crossplane/addon-oam-kubernetes-local/blob/757d1922a5266e775b1f131af7da4fb6cbc1a037/pkg/webhooks/manualscalertrait_webhook.go). This webhook [is disabled by default](https://github.com/crossplane/addon-oam-kubernetes-local/blob/42e82c49fb679df4e295802b4727c25faaad4d24/charts/oam-core-resources/values.yaml#L6) in its helm chart.

You need to install a cert-manager to provide self-signed certifications if you want to play with the webhooks.

```console
kubectl create namespace cert-manager
kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v0.14.0/cert-manager.yaml
```
> For more detailed instructions of cert manager please check [Cert-manager docs](https://cert-manager.io/docs/installation/kubernetes/).

#### Install controllers

```console
kubectl create namespace oam-system
helm install controller -n oam-system ./charts/oam-core-resources/ 
```

## Verify

* Apply a sample application configuration

```console
kubectl apply -f examples/containerized-workload/
```

* Verify that the application is running
You can check the status and events from the applicationconfiguration object   
```console
kubectl describe applicationconfigurations.core.oam.dev example-appconfig
Status:
  Conditions:
    Last Transition Time:  2020-06-12T21:18:40Z
    Reason:                Successfully reconciled resource
    Status:                True
    Type:                  Synced
  Workloads:
    Component Name:  example-component
    Traits:
      Trait Ref:
        API Version:  core.oam.dev/v1alpha2
        Kind:         ManualScalerTrait
        Name:         example-appconfig-trait
    Workload Ref:
      API Version:  core.oam.dev/v1alpha2
      Kind:         ContainerizedWorkload
      Name:         example-appconfig-workload
Events:
  Type    Reason                 Age                    From                                       Message
  ----    ------                 ----                   ----                                       -------
  Normal  RenderedComponents     48s (x220 over 3h35m)  oam/applicationconfiguration.core.oam.dev  Successfully rendered components
  Normal  Deployment created     30s (x2 over 30s)      ContainerizedWorkload                      Successfully server side patched a deployment
  Normal  Service created        30s (x2 over 30s)      ContainerizedWorkload                      Successfully applied a service
  Normal  Manual scalar applied  30s                    ManualScalarTrait                          Successfully scaled a resource
```

You should also see a deployment looking like below
```console
kubectl get deployments
NAME                                    READY   UP-TO-DATE   AVAILABLE   AGE
example-appconfig-workload-deployment   3/3   3           3              28s
```

And a service looking like below
```console
kubectl get services
AME                                             TYPE       CLUSTER-IP     EXTERNAL-IP   PORT(S)    AGE
example-appconfig-workload-deployment-service   NodePort   10.96.78.215   <none>        8080/TCP   28s
```
