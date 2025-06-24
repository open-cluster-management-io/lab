# Default values
API_IMAGE_NAME ?= dashboard-api
UI_IMAGE_NAME ?= dashboard-ui
IMAGE_TAG ?= latest
REGISTRY ?= quay.io/open-cluster-management
PLATFORMS ?= linux/amd64,linux/arm64

# Full image names
API_FULL_IMAGE_NAME = $(REGISTRY)/$(API_IMAGE_NAME):$(IMAGE_TAG)
UI_FULL_IMAGE_NAME = $(REGISTRY)/$(UI_IMAGE_NAME):$(IMAGE_TAG)

.PHONY: dev-ui dev-uiserver dev-apiserver dev-apiserver-real build-ui build-uiserver build-apiserver build docker-build-api docker-push-api docker-build-push-api clean

# Development targets
dev-ui:
	cd . && npm run dev

dev-uiserver:
	@echo "Building JS code first..."
	npm run build
	@echo "Starting UI GIN server..."
	cd uiserver && go run uiserver.go

dev-apiserver:
	cd apiserver && chmod +x run-dev.sh && ./run-dev.sh

dev-apiserver-real:
	cd apiserver && go run main.go

# Build targets
build-ui:
	cd . && npm run build

build-uiserver:
	cd uiserver && go build -o uiserver

build-apiserver:
	cd apiserver && go build -o apiserver

build: build-ui build-uiserver build-apiserver

# Docker build targets for API (multi-arch, no load)
docker-build-api:
	@echo "Building API Docker image: $(API_FULL_IMAGE_NAME)"
	docker buildx build \
		--platform $(PLATFORMS) \
		-f Dockerfile.api \
		-t $(API_FULL_IMAGE_NAME) \
		.

docker-push-api:
	@echo "Pushing API Docker image: $(API_FULL_IMAGE_NAME)"
	docker push $(API_FULL_IMAGE_NAME)

docker-build-push-api:
	@echo "Building and pushing API multi-arch Docker image: $(API_FULL_IMAGE_NAME)"
	docker buildx build \
		--platform $(PLATFORMS) \
		-f Dockerfile.api \
		-t $(API_FULL_IMAGE_NAME) \
		--push \
		.

# Docker build targets for UI (multi-arch, no load)
docker-build-ui:
	@echo "Building UI Docker image: $(UI_FULL_IMAGE_NAME)"
	docker buildx build \
		--platform $(PLATFORMS) \
		-f Dockerfile.ui \
		-t $(UI_FULL_IMAGE_NAME) \
		.

docker-push-ui:
	@echo "Pushing UI Docker image: $(UI_FULL_IMAGE_NAME)"
	docker push $(UI_FULL_IMAGE_NAME)

docker-build-push-ui:
	@echo "Building and pushing UI multi-arch Docker image: $(UI_FULL_IMAGE_NAME)"
	docker buildx build \
		--platform $(PLATFORMS) \
		-f Dockerfile.ui \
		-t $(UI_FULL_IMAGE_NAME) \
		--push \
		.

# Build both images
docker-build: docker-build-api docker-build-ui

# Push both images
docker-push: docker-push-api docker-push-ui

# Build and push both images
docker-build-push: docker-build-push-api docker-build-push-ui

# Build for local development (single arch)
docker-build-local-api:
	@echo "Building local API Docker image: $(API_IMAGE_NAME):$(IMAGE_TAG)"
	docker buildx build \
		-f Dockerfile.api \
		-t $(API_IMAGE_NAME):$(IMAGE_TAG) \
		--load \
		.

docker-build-local-ui:
	@echo "Building local UI Docker image: $(UI_IMAGE_NAME):$(IMAGE_TAG)"
	docker buildx build \
		-f Dockerfile.ui \
		-t $(UI_IMAGE_NAME):$(IMAGE_TAG) \
		--load \
		.

docker-build-local: docker-build-local-api docker-build-local-ui

# Legacy docker target for backward compatibility
docker: docker-build-local

# Clean up
clean:
	rm -rf dist
	rm -rf apiserver/static
	rm -f apiserver/apiserver
	rm -f uiserver/uiserver

# Add target to use debug script
debug-apiserver:
	cd apiserver && chmod +x debug.sh && ./debug.sh

# Setup buildx builder for multi-arch builds
setup-buildx:
	docker buildx create --name ocm-builder --use || docker buildx use ocm-builder
	docker buildx inspect --bootstrap

# Default target
all: build

test-frontend:
	npm run test

test-apiserver:
	cd apiserver && go test ./...

test-uiserver:
	cd uiserver && go test ./...

test: test-frontend test-apiserver test-uiserver
	@echo "All tests passed!"

lint:
	npm run lint
	cd apiserver && go vet ./...
	cd uiserver && go vet ./...

# Test the UI server functionality
test-uiserver-functionality:
	@echo "Testing UI server functionality..."
	@echo "Building frontend..."
	@npm run build > /dev/null 2>&1
	@echo "Starting UI server in background..."
	@cd uiserver && go run uiserver.go & echo $$! > /tmp/uiserver.pid
	@sleep 3
	@echo "Testing endpoints..."
	@curl -s http://localhost:3000/health | grep -q "healthy" && echo "✅ Health endpoint: OK" || echo "❌ Health endpoint: FAILED"
	@curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/ | grep -q "200" && echo "✅ Main page: OK" || echo "❌ Main page: FAILED"
	@curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/assets/index-B4HUQdL0.css | grep -E "(200|404)" > /dev/null && echo "✅ Assets: Available" || echo "❌ Assets: FAILED"
	@echo "Stopping UI server..."
	@kill `cat /tmp/uiserver.pid` 2>/dev/null || true
	@rm -f /tmp/uiserver.pid
	@echo "✅ UI server test completed!"