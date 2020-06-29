module github.com/crossplane/oam-controllers

go 1.13

require (
	github.com/crossplane/crossplane-runtime v0.8.0
	github.com/crossplane/oam-kubernetes-runtime v0.0.3
	github.com/go-logr/logr v0.1.0
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	github.com/pkg/errors v0.9.1
	gomodules.xyz/jsonpatch/v2 v2.0.1
	k8s.io/api v0.18.3
	k8s.io/apimachinery v0.18.3
	k8s.io/client-go v0.18.3
	k8s.io/kube-openapi v0.0.0-20200410145947-61e04a5be9a6
	k8s.io/kubectl v0.18.3
	sigs.k8s.io/controller-runtime v0.6.0
)
