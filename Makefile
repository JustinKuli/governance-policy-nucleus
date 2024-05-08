ROOTDIR ?= $(CURDIR)

## Location to install dependencies to
LOCAL_BIN ?= $(ROOTDIR)/bin
$(LOCAL_BIN):
	mkdir -p $(LOCAL_BIN)

# Keep an existing GOPATH, make a private one if it is undefined
GOPATH_DEFAULT := $(ROOTDIR)/.go
export GOPATH ?= $(GOPATH_DEFAULT)
GOBIN_DEFAULT := $(GOPATH)/bin
export GOBIN ?= $(GOBIN_DEFAULT)

# Set PATH so that locally installed things will be used first
export PATH=$(LOCAL_BIN):$(GOBIN):$(shell echo $$PATH)

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# go-install will 'go install' any package $1 to LOCAL_BIN
# Note: this replaces `go-get-tool`.
go-install = @set -e ; mkdir -p $(LOCAL_BIN) ; GOBIN=$(LOCAL_BIN) go install $(1)

# Define local utilities before other targets so they work correctly
# Note: this pattern of variables, paths, and target names allows users to
#  override the version used, and helps Make by not using PHONY targets.
# To 'refresh' versions, remove the local bin directory.

CONTROLLER_GEN_VERSION ?= v0.15.0 # https://github.com/kubernetes-sigs/controller-tools/releases/latest
CONTROLLER_GEN ?= $(LOCAL_BIN)/controller-gen
$(CONTROLLER_GEN): $(LOCAL_BIN)
	$(call go-install,sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION))

ENVTEST ?= $(LOCAL_BIN)/setup-envtest
$(ENVTEST): $(LOCAL_BIN)
	$(call go-install,sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)

KUSTOMIZE_VERSION ?= v5.4.1 # https://github.com/kubernetes-sigs/kustomize/releases/latest
KUSTOMIZE ?= $(LOCAL_BIN)/kustomize
$(KUSTOMIZE): $(LOCAL_BIN)
	$(call go-install,sigs.k8s.io/kustomize/kustomize/v5@$(KUSTOMIZE_VERSION))

GOLANGCI_VERSION ?= v1.58.0 # https://github.com/golangci/golangci-lint/releases/latest
GOLANGCI ?= $(LOCAL_BIN)/golangci-lint
$(GOLANGCI): $(LOCAL_BIN)
	$(call go-install,github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_VERSION))

# To change this version, adjust it in the go.mod file
# https://github.com/onsi/ginkgo/releases/latest
GINKGO_VERSION := $(shell awk '/github.com\/onsi\/ginkgo\/v2/ {print $$2}' go.mod)
GINKGO ?= $(LOCAL_BIN)/ginkgo
$(GINKGO): $(LOCAL_BIN)
	$(call go-install,github.com/onsi/ginkgo/v2/ginkgo@$(GINKGO_VERSION))

.PHONY: manifests
manifests: $(CONTROLLER_GEN) $(KUSTOMIZE) ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths=".;./api/..." \
	  output:crd:artifacts:config=config/crd/bases
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./test/fakepolicy/..." \
	  output:crd:artifacts:config=test/fakepolicy/config/crd/bases \
	  output:rbac:artifacts:config=test/fakepolicy/config/rbac
	$(KUSTOMIZE) build ./test/fakepolicy/config/default > ./test/fakepolicy/config/deploy.yaml

.PHONY: generate
generate: $(CONTROLLER_GEN) ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

# Note: this target is not used by Github Actions. Instead, each linter is run 
# separately to automatically decorate the code with the linting errors.
# Note: this target will fail if yamllint is not installed.
.PHONY: lint
lint: $(GOLANGCI)
	$(GOLANGCI) run
	yamllint -c $(ROOTDIR)/.yamllint.yaml .

ENVTEST_K8S_VERSION ?= 1.29
.PHONY: test
test: manifests generate $(GINKGO) $(ENVTEST) ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" $(GINKGO) \
	  --coverpkg=./... --covermode=count --coverprofile=cover.out ./...

.PHONY: fuzz-test
fuzz-test:
	go test ./api/v1beta1 -fuzz=FuzzMatchesExcludeAll -fuzztime=20s
	go test ./api/v1beta1 -fuzz=FuzzMatchesIncludeAll -fuzztime=20s
