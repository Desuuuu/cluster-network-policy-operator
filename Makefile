REPOSITORY ?= ghcr.io/desuuuu/cluster-network-policy-operator

HELM_REPOSITORY ?= ghcr.io/desuuuu/helm-charts
HELM_RELEASE ?= cluster-network-policy-operator
HELM_NAMESPACE ?= cluster-network-policy-operator

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.29.0

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: generate
generate: controller-gen helm-docs ## Run controller-gen and helm-docs.
	rm -f ./helm/crds/*.yaml ./helm/templates/generated/*.yaml
	$(CONTROLLER_GEN) paths="./..." object:headerFile="./hack/boilerplate.go.txt" crd rbac:roleName=__ROLE_NAME__ output:crd:artifacts:config=./helm/crds output:rbac:artifacts:config=./helm/templates/generated
	sed -i -r -z 's/^(\s|-)*//' ./helm/crds/*.yaml ./helm/templates/generated/*.yaml
	sed -i 's/name: __ROLE_NAME__/{{- include "cluster-network-policy-operator.clusterRoleMetadata" . | nindent 2 }}/' ./helm/templates/generated/*.yaml
	$(HELM_DOCS) -c ./helm -s file --ignore-non-descriptions

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./... -coverprofile ./cover.out

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter.
	$(GOLANGCI_LINT) run ./...

##@ Build

.PHONY: build
build: generate fmt vet ## Build the manager.
	go build -o ./bin/manager ./cmd/manager

.PHONY: run
run: generate fmt vet ## Run the manager.
	go run ./cmd/manager

.PHONY: ko-build
ko-build: generate fmt vet ko ## Build and push image.
	$(call require-env,TAG)
	KO_DOCKER_REPO=$(REPOSITORY) $(KO) build ./cmd/manager --bare --tags=$(TAG) --tag-only --sbom=cyclonedx

.PHONY: ko-build-local
ko-build-local: generate fmt vet ko ## Build image locally.
	$(call require-env,TAG)
	KO_DOCKER_REPO=$(REPOSITORY) $(KO) build ./cmd/manager --bare --tags=$(TAG) --tag-only --local

.PHONY: ko-login
ko-login: ko ## Log into a remote registry.
	$(call require-env,REGISTRY)
	$(call require-env,USERNAME)
	$(call require-env,PASSWORD)
	@$(KO) login -u "$(USERNAME)" -p "$(PASSWORD)" "$(REGISTRY)"

##@ Helm

.PHONY: helm-template
helm-template: generate ## Generate YAML manifests from Helm chart.
	$(call require-env,TAG)
	helm template $(HELM_RELEASE) ./helm --namespace $(HELM_NAMESPACE) --skip-crds --set image.tag=$(TAG) > $${HELM_TEMPLATE:-"/dev/stdout"}

.PHONY: helm-check
helm-check: generate kubeconform ## Validate Helm chart templates.
	$(call require-env,TAG)
	helm template $(HELM_RELEASE) ./helm --namespace $(HELM_NAMESPACE) --skip-crds --set image.tag=$(TAG) | $(KUBECONFORM) -summary

.PHONY: helm-package
helm-package: generate ## Package and push Helm chart.
	$(call require-env,TAG)
	helm package ./helm -u --version $${TAG#v} --app-version $(TAG) -d ./dist
	helm push ./dist/cluster-network-policy-operator-$${TAG#v}.tgz oci://$(HELM_REPOSITORY)

.PHONY: helm-login
helm-login: ## Log into a remote registry.
	$(call require-env,REGISTRY)
	$(call require-env,USERNAME)
	$(call require-env,PASSWORD)
	helm registry login "$(REGISTRY)" -u "$(USERNAME)" -p "$(PASSWORD)"

##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen-$(CONTROLLER_TOOLS_VERSION)
ENVTEST ?= $(LOCALBIN)/setup-envtest-$(ENVTEST_VERSION)
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint-$(GOLANGCI_LINT_VERSION)
KO ?= $(LOCALBIN)/ko-$(KO_VERSION)
KUBECONFORM ?= $(LOCALBIN)/kubeconform-$(KUBECONFORM_VERSION)
HELM_DOCS ?= $(LOCALBIN)/helm-docs-$(HELM_DOCS_VERSION)

## Tool Versions
CONTROLLER_TOOLS_VERSION ?= v0.15.0
ENVTEST_VERSION ?= release-0.18
GOLANGCI_LINT_VERSION ?= v1.58.1
KO_VERSION ?= v0.15.2
KUBECONFORM_VERSION ?= v0.6.6
HELM_DOCS_VERSION ?= v1.13.1

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally.
$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

.PHONY: ko
ko: $(KO) ## Download ko locally.
$(KO): $(LOCALBIN)
	$(call go-install-tool,$(KO),github.com/google/ko,$(KO_VERSION))

.PHONY: kubeconform
kubeconform: $(KUBECONFORM) ## Download kubeconform locally.
$(KUBECONFORM): $(LOCALBIN)
	$(call go-install-tool,$(KUBECONFORM),github.com/yannh/kubeconform/cmd/kubeconform,$(KUBECONFORM_VERSION))

.PHONY: helm-docs
helm-docs: $(HELM_DOCS) ## Download helm-docs locally.
$(HELM_DOCS): $(LOCALBIN)
	$(call go-install-tool,$(HELM_DOCS),github.com/norwoodj/helm-docs/cmd/helm-docs,$(HELM_DOCS_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary (ideally with version)
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f $(1) ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv "$$(echo "$(1)" | sed "s/-$(3)$$//")" $(1) ;\
}
endef

define require-env
@[ -n "$($(1))" ] || { \
echo "$(1) must be set" >&2 ;\
exit 1 ;\
}
endef
