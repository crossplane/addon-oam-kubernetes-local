
# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests
test: fmt vet
	go test -v $(shell go list ./... | grep -v e2e-test) -coverprofile cover.out

# Build controller binary
controller: fmt vet
	go build -o bin/controller main.go

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	cd config/manager && kustomize edit set image controller=${IMG}
	kustomize build config/default | kubectl apply -f -

# Delete controller in the configured Kubernetes cluster in ~/.kube/config
clean: manifests
	kustomize build config/default | kubectl delete -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths="./..."

# Build the docker image
docker-build:
	docker build . -t $(IMG)

# Push the docker image
docker-push:
	docker push ${IMG}

# load docker image to the kind cluster
kind-load:
	kind load docker-image $(IMG) || { echo >&2 "kind not installed or error loading image: $(IMG)"; exit 1; }

e2e-setup:
	kubectl create namespace cert-manager
	kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v0.14.0/cert-manager.yaml
	kubectl create namespace crossplane-system
	helm repo add crossplane-master https://charts.crossplane.io/master/
	helm install crossplane --namespace crossplane-system crossplane-master/crossplane --version 0.9.0-rc --wait
	kubectl wait --for=condition=Ready pod -l app.kubernetes.io/name=webhook -n cert-manager --timeout=300s
	kubectl wait --for=condition=Ready pod -l app.kubernetes.io/instance=cert-manager -n cert-manager --timeout=300s
	kubectl wait --for=condition=Ready pod -l app=crossplane -n crossplane-system --timeout=300s

e2e-cleanup:
	helm uninstall crossplane --namespace crossplane-system
	kubectl delete namespace crossplane-system --wait
	kubectl delete -f https://github.com/jetstack/cert-manager/releases/download/v0.14.0/cert-manager.yaml --wait

e2e-test: docker-build kind-load
	kubectl create namespace oam-system
	helm install e2e ./charts/oam-core-resources/ -n oam-system --set image.repository=$(IMG) --wait \
		|| { echo >&2 "helm install timeout"; \
		kubectl logs `kubectl get pods -n oam-system -l "app.kubernetes.io/name=oam-core-resources,app.kubernetes.io/instance=e2e" -o jsonpath="{.items[0].metadata.name}"` -c e2e; \
		helm uninstall e2e -n oam-system; exit 1;}
	kubectl wait --for=condition=Ready pod -l app.kubernetes.io/name=oam-core-resources -n oam-system --timeout=300s
	ginkgo -v ./e2e-test/
	helm uninstall e2e -n oam-system
	kubectl delete namespace oam-system --wait

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.4 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif