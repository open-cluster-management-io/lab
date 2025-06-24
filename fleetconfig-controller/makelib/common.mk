# If you update this file, please follow:
# https://www.thapaliya.com/en/writings/well-documented-makefiles/

.DEFAULT_GOAL := help

# See https://stackoverflow.com/questions/11958626/make-file-warning-overriding-commands-for-target
%: %-default
	@ true

TIME   = `date +%H:%M:%S`

BLUE   := $(shell printf "\033[34m")
YELLOW := $(shell printf "\033[33m")
RED    := $(shell printf "\033[31m")
GREEN  := $(shell printf "\033[32m")
CNone  := $(shell printf "\033[0m")

INFO = echo ${TIME} ${BLUE}[INFO]${CNone}
OK   = echo ${TIME} ${GREEN}[ OK ]${CNone}
ERR  = echo ${TIME} ${RED}[ ERR ]${CNone} "error:"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

GOARCH ?= $(shell go env GOARCH)
GOOS ?= $(shell go env GOOS)
GOBUILD_ENV = CGO_ENABLED=0

##@ Help Targets
help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[0m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Dev / CI Targets

check-diff: reviewable ## Execute auto-gen code commands and ensure branch is clean
	git --no-pager diff
	git diff --quiet || ($(ERR) please run 'make reviewable' to include all changes && false)
	@$(OK) branch is clean

reviewable-default: fmt vet lint generate manifests helm-doc-gen ## Ensure code is ready for review
	go mod tidy
	go mod vendor
	@$(INFO) PR is ready for review

fmt-default: ## Run go fmt against code
	@$(INFO) go fmt
	@go fmt ./...

vet: ## Run go vet against code
	@$(INFO) go vet
	@go vet $(shell go list ./...)

lint: golangci-lint ## Run golangci-lint against code
	@$(INFO) lint
	@$(GOLANGCI_LINT) run --fix --verbose

##@ Tool Targets

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

export PATH := $(PATH):$(LOCALBIN)

.PHONY: helmdoc
helmdoc: ## Install readme-generator-for-helm
ifeq (, $(shell which readme-generator))
	@{ \
	set -e ;\
	echo 'installing readme-generator-for-helm' ;\
	npm install -g @bitnami/readme-generator-for-helm ;\
	}
else
	@$(OK) readme-generator-for-helm is already installed
HELMDOC=$(shell which readme-generator)
endif

KIND_VERSION ?= 0.27.0
.PHONY: kind
kind:
	@command -v kind >/dev/null 2>&1 || { \
		echo "kind not found, downloading..."; \
		curl -Lo $(LOCALBIN)/kind https://github.com/kubernetes-sigs/kind/releases/download/v$(KIND_VERSION)/kind-$(GOOS)-$(GOARCH); \
		chmod +x $(LOCALBIN)/kind; \
	}

KUBECTL_VERSION ?= 1.31.9
KUBECTL ?= kubectl
.PHONY: kubectl
kubectl:
	@command -v kubectl >/dev/null 2>&1 || { \
		echo "kubectl not found, downloading..."; \
		curl -Lo $(LOCALBIN)/kubectl https://dl.k8s.io/release/v$(KUBECTL_VERSION)/bin/$(GOOS)/$(GOARCH)/kubectl; \
		chmod +x $(LOCALBIN)/kubectl; \
	}

SUPPORT_BUNDLE_VERSION ?= 0.119.0
SUPPORT_BUNDLE ?= support-bundle
.PHONY: support-bundle
support-bundle:
	@command -v support-bundle >/dev/null 2>&1 || { \
		echo "support-bundle not found, downloading..."; \
		curl -Lo $(LOCALBIN)/support-bundle.tar.gz https://github.com/replicatedhq/troubleshoot/releases/download/v$(SUPPORT_BUNDLE_VERSION)/support-bundle_$(GOOS)_$(GOARCH).tar.gz; \
		tar -xzf $(LOCALBIN)/support-bundle.tar.gz -C $(LOCALBIN) support-bundle; \
		chmod +x $(LOCALBIN)/support-bundle; \
		rm $(LOCALBIN)/support-bundle.tar.gz; \
	}

CONTROLLER_TOOLS_VERSION ?= v0.17.0
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen-$(CONTROLLER_TOOLS_VERSION)
.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Install controller-gen
$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

ENVTEST_VERSION ?= release-0.19
ENVTEST ?= $(LOCALBIN)/setup-envtest-$(ENVTEST_VERSION)
ENVTEST_K8S_VERSION := $(shell jq -r '.envTestK8sVersion' ../hack/test-config.json 2>/dev/null)
.PHONY: envtest
envtest: $(ENVTEST) ## Install setup-envtest locally if necessary
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

GINKGO_VERSION ?= v2.23.4	
GINKGO ?= $(LOCALBIN)/ginkgo-$(GINKGO_VERSION)
.PHONY: ginkgo
ginkgo: $(GINKGO) ## Install ginkgo locally if necessary
$(GINKGO): $(LOCALBIN)
	$(call go-install-tool,$(GINKGO),github.com/onsi/ginkgo/v2/ginkgo,$(GINKGO_VERSION))

GOCOVMERGE_VERSION ?= latest
GOCOVMERGE ?= $(LOCALBIN)/gocovmerge-$(GOCOVMERGE_VERSION)
.PHONY: gocovmerge
gocovmerge: $(GOCOVMERGE) ## Install gocovmerge locally if necessary
$(GOCOVMERGE): $(LOCALBIN)
	$(call go-install-tool,$(GOCOVMERGE),github.com/wadey/gocovmerge,${GOCOVMERGE_VERSION})

GOLANGCI_LINT_VERSION ?= v2.0.2
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint-$(GOLANGCI_LINT_VERSION)
.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Install golangci-lint locally if necessary
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,${GOLANGCI_LINT_VERSION})

KUSTOMIZE_VERSION ?= v5.4.1
KUSTOMIZE = $(LOCALBIN)/kustomize-$(KUSTOMIZE_VERSION)
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Install kustomize locally if necessary
$(KUSTOMIZE): $(LOCALBIN)
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

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
