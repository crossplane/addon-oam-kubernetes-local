# oam-kubernetes-implementation

This OAM Kubernetes implementation allows an application developer/operator to compose a cloud-native application following the
 OAM v1alpha2 spec and run it on a kubernetes cluster.  

## Examples

The functionality of this addon can be demonstrated with the following steps:

* Install cert manager
```
kubectl create namespace cert-manager

kubectl apply --validate=false -f https://github.com/jetstack/cert-manager/releases/download/v0.14.0/cert-manager.yaml
```
More detailed instructions for cert manager can be found in the [Cert-manager docs](https://cert-manager.io/docs/installation/kubernetes/).

* Install OAM Application Controller
```
kubectl create namespace crossplane-system

helm repo add crossplane-master https://charts.crossplane.io/master/

version=$(helm search repo crossplane --devel | awk '$1 == "crossplane-master/crossplane" {print $2}')

helm install crossplane --namespace crossplane-system crossplane-master/crossplane --version $version --devel
```

More detailed instructions can be found in the [Crossplane docs]( https://crossplane.io/docs/v0.8/install-crossplane.html).

* Install OAM Core workload and trait controllers

```
git clone git@github.com:crossplane/addon-oam-kubernetes-local.git

make uninstall

make docker-build IMG=controller:v1

make deploy IMG=controller:v1
```

* Apply the sample application config

```
kubectl apply -f config/samples/sample_application_config.yaml
```

*. Verify that the application is running

```
kubectl get secret k8scluster --template={{.data.kubeconfig}} | base64 --decode > remote.kubeconfig
kubectl --kubeconfig=remote.kubeconfig get deployments
kubectl --kubeconfig=remote.kubeconfig get services
```
