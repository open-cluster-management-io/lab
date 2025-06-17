# If you update this file, please follow:
# https://www.thapaliya.com/en/writings/well-documented-makefiles/

include makelib/common.mk

IMAGE ?= quay.io/open-cluster-management/fleetconfig-controller:$(TAG)

##@ Dev / CI Targets

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/main.go

##@ Generation Targets

helm-doc-gen: helmdoc ## Generate helm chart README.md
	readme-generator -v charts/fleetconfig-controller/values.yaml -r charts/fleetconfig-controller/README.md

.PHONY: manifests
manifests: kustomize controller-gen ## Generate CustomResourceDefinition and WebhookConfiguration objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	$(KUSTOMIZE) build config/crd -o charts/fleetconfig-controller/crds/fleetconfig.open-cluster-management.io-crds.yaml
	./hack/install_crds.sh

##@ Testing Targets

.PHONY: test
test: gocovmerge test-unit test-e2e ## Run tests.
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

.PHONY: test-e2e  # Run e2e tests in the top-level test directory.
test-e2e: kind kubectl ginkgo
	@mkdir -p $(COVER_DIR)/e2e
	rm -rf $(COVER_DIR)/e2e/*
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
install-crds: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

.PHONY: uninstall-crds
uninstall-crds: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -
