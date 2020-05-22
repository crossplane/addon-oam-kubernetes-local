# oam-kubernetes-implementation

This OAM Kubernetes implementation allows an application developer/operator to compose a cloud-native application following the
 OAM v1alpha2 spec and run it on a kubernetes cluster.  

## Prerequisites

1. go version 1.12+
2. Kubernetes version v1.17+ with KubeCtl configured
3. Helm 3.0+ 


## Getting started

The functionality of this addon can be demonstrated with the following steps:

* Install cert manager
```
kubectl create namespace cert-manager

kubectl apply --validate=false -f https://github.com/jetstack/cert-manager/releases/download/v0.15.0/cert-manager.yaml
```
More detailed instructions for cert manager can be found in the [Cert-manager docs](https://cert-manager.io/docs/installation/kubernetes/).

* Install OAM Application Controller
```
kubectl create namespace crossplane-system

helm repo add crossplane-alpha https://charts.crossplane.io/alpha

helm install crossplane --namespace crossplane-system crossplane-alpha/crossplane
```

More detailed instructions can be found in the [Crossplane docs]( https://crossplane.io/docs/v0.8/install-crossplane.html).

* Install OAM Core workload and trait controllers

```
git clone git@github.com:crossplane/addon-oam-kubernetes-local.git

kubectl create namespace oam-system

helm install controller -n oam-system ./charts/oam-core-resources/ 
```

* Apply the sample application config

```
kubectl apply -f examples/containerized-workload/
```

* Verify that the application is running
You should see a deployment looking like below
```
kubectl get deployments
NAME                                    READY   UP-TO-DATE   AVAILABLE   AGE
example-appconfig-workload-deployment   10/10   10           10          8m11s
```

And a service looking like below
```
kubectl get services
AME                                            TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)    AGE
example-appconfig-workload-deployment-service   ClusterIP   10.96.78.215   <none>        8080/TCP   8m28s
```
