IMAGE_REGISTRY ?= quay.io/open-cluster-management
IMAGE_REPO ?= headlamp-ocm
IMAGE_TAG ?= latest
IMAGE_NAME = $(IMAGE_REGISTRY)/$(IMAGE_REPO):$(IMAGE_TAG)
PLATFORMS ?= linux/amd64,linux/arm64

.PHONY: check-diff
check-diff:
	npm run format
	npm run lint
	@if [ -n "$$(git diff --name-only)" ]; then \
		echo "Diff detected after formatting/linting:"; \
		git diff --name-only; \
		exit 1; \
	fi

.PHONY: test-unit
test-unit:
	npm run test

.PHONY: test
test: test-unit

.PHONY: test-e2e
test-e2e:
	@echo "No e2e tests configured yet"

.PHONY: build
build:
	npm run build

.PHONY: images
images:
	docker buildx build \
		--platform $(PLATFORMS) \
		-f build/Dockerfile.base \
		-t $(IMAGE_NAME) \
		--push \
		.

# buildx --push in the images target handles push and multi-arch manifests.
# These targets are retained as no-ops for CI compatibility.

.PHONY: image-push
image-push:
	@echo "Handled by buildx --push in images target"

.PHONY: image-manifest
image-manifest:
	@echo "Handled by buildx --platform in images target"

.PHONY: image-manifest-annotate
image-manifest-annotate:
	@echo "Handled by buildx --platform in images target"

.PHONY: image-manifest-push
image-manifest-push:
	@echo "Handled by buildx --platform in images target"
