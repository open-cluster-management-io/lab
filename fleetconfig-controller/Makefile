# If you update this file, please follow:
# https://www.thapaliya.com/en/writings/well-documented-makefiles/

include makelib/common.mk

IMAGEBUILDER_VERSION ?= 1.2.16
IMAGE_BUILD_EXTRA_FLAGS ?= --build-arg OS=linux --build-arg ARCH=$(GOARCH)

include $(addprefix ./vendor/github.com/openshift/build-machinery-go/make/, \
	targets/openshift/images.mk \
)

IMAGE_REGISTRY ?= quay.io/open-cluster-management
IMAGE_TAG ?= latest
IMAGE_FLAVOURS ?= \
	fleetconfig-controller:./build/Dockerfile.base \
	fleetconfig-controller-eks:./build/Dockerfile.eks \
	fleetconfig-controller-gke:./build/Dockerfile.gke

# Generate build targets for each variant
$(foreach flavour,$(IMAGE_FLAVOURS),\
	$(eval IMAGE_REPO := $(word 1,$(subst :, ,$(flavour)))) \
	$(eval DOCKERFILE := $(word 2,$(subst :, ,$(flavour)))) \
	$(eval $(call build-image,$(IMAGE_REPO),$(IMAGE_REGISTRY)/$(IMAGE_REPO):$(IMAGE_TAG),$(DOCKERFILE),.)) \
)

.PHONY: image-push
image-push:
	$(foreach flavour,$(IMAGE_FLAVOURS),\
		$(eval IMAGE_REPO := $(word 1,$(subst :, ,$(flavour)))) \
		docker push $(IMAGE_REGISTRY)/$(IMAGE_REPO):$(IMAGE_TAG); \
	)

.PHONY: image-manifest
image-manifest:
	$(foreach flavour,$(IMAGE_FLAVOURS),\
		$(eval IMAGE_REPO := $(word 1,$(subst :, ,$(flavour)))) \
		$(eval IMAGE := $(IMAGE_REGISTRY)/$(IMAGE_REPO):$(IMAGE_TAG)) \
		docker manifest create $(IMAGE) $(IMAGE)-amd64 --amd64 $(IMAGE)-arm64 --arm64; \
	)

.PHONY: image-manifest-annotate
image-manifest-annotate:
	$(foreach flavour,$(IMAGE_FLAVOURS),\
		$(eval IMAGE_REPO := $(word 1,$(subst :, ,$(flavour)))) \
		$(eval IMAGE := $(IMAGE_REGISTRY)/$(IMAGE_REPO):$(IMAGE_TAG)) \
		docker manifest annotate $(IMAGE) --arch amd64 $(IMAGE)-amd64; \
		docker manifest annotate $(IMAGE) --arch arm64 $(IMAGE)-arm64; \
	)

##@ Generation Targets

.PHONY: generate
generate: controller-gen ## Generate controller API code with controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

helm-doc-gen: helmdoc ## Generate helm chart README.md
	readme-generator -v charts/fleetconfig-controller/values.yaml -r charts/fleetconfig-controller/README.md

.PHONY: manifests
manifests: kustomize controller-gen ## Generate CustomResourceDefinition and WebhookConfiguration objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	$(KUSTOMIZE) build config/crd -o charts/fleetconfig-controller/crds/fleetconfig.open-cluster-management.io-crds.yaml
	./hack/install_crds.sh

##@ Testing Targets

COVER_DIR=_build/cov

coverage: ## Show global test coverage
	go tool cover -func $(COVER_DIR)/coverage.out

coverage-html: ## Open global test coverage report in your browser
	go tool cover -html $(COVER_DIR)/coverage.out

.PHONY: test
test: gocovmerge test-unit test-e2e ## Run all tests.
	$(GOCOVMERGE) $(COVER_DIR)/unit/*.out $(COVER_DIR)/e2e/*.out > $(COVER_DIR)/coverage.out.tmp
	# Omit test code from coverage report
	cat $(COVER_DIR)/coverage.out.tmp | grep -vE 'test' > $(COVER_DIR)/coverage.out
	go tool cover -func=$(COVER_DIR)/coverage.out -o $(COVER_DIR)/cover.func
	go tool cover -html=$(COVER_DIR)/coverage.out -o $(COVER_DIR)/cover.html
	go tool cover -func=$(COVER_DIR)/coverage.out | grep total
	cp $(COVER_DIR)/coverage.out cover.out

.PHONY: test-unit
test-unit: manifests generate fmt vet envtest ## Run unit tests.
	@mkdir -p $(COVER_DIR)/unit
	rm -rf $(COVER_DIR)/unit/*
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" \
		go test -v \
		-coverpkg=./... \
		-coverprofile=cover.out \
		$(shell go list ./... | grep -v '/test/e2e')

.PHONY: test-e2e
test-e2e: kind kubectl ginkgo support-bundle ## Run e2e tests in the top-level test directory.
	@mkdir -p $(COVER_DIR)/e2e
	rm -rf $(COVER_DIR)/e2e/*
	@set -a; source ./hack/.versions.env; set +a; \
	$(GINKGO) run -vv \
		--cover \
		--coverpkg=./... \
		--label-filter="fleetconfig" \
		--output-dir=$(COVER_DIR)/e2e \
		--timeout 20m \
		./test/e2e/

##@ Build / Deploy Targets

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install-crds
install-crds: manifests kustomize ## Install CRDs.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

.PHONY: uninstall-crds
uninstall-crds: manifests kustomize ## Uninstall CRDs. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -
