# Makefile for Btrfs CSI Driver

# Variables
REGISTRY ?= localhost:5000
IMAGE_NAME ?= btrfs-csi
IMAGE_TAG ?= latest
IMAGE ?= $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build the binary
.PHONY: build
build:
	$(GOBUILD) -o bin/btrfs-csi -v ./main.go

# Clean build artifacts
.PHONY: clean
clean:
	$(GOCLEAN)
	rm -rf bin/

# Run tests
.PHONY: test
test:
	$(GOTEST) -v ./...

# Download dependencies
.PHONY: deps
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Build Docker image
.PHONY: docker-build
docker-build:
	docker build -t $(IMAGE) .

# Copy Docker image to k3s
.PHONY: docker-copy
docker-copy: docker-build
	docker save $(IMAGE) | k3s ctr images import -

# Push Docker image
.PHONY: docker-push
docker-push:
	docker push $(IMAGE)

# Deploy to Kubernetes
.PHONY: deploy-csi
deploy-csi:
	kubectl apply -f deploy/kubernetes/

# Undeploy from Kubernetes
.PHONY: undeploy-csi
undeploy-csi:
	kubectl delete -f deploy/kubernetes/ --ignore-not-found=true

# Deploy test resources
.PHONY: deploy-test
deploy-test:
	kubectl apply -f test/kubernetes/

# Clean up test resources
.PHONY: clean-test
clean-test:
	kubectl delete -f test/kubernetes/ --ignore-not-found=true

# Check if Btrfs is available on nodes
.PHONY: check-btrfs
check-btrfs:
	@echo "Checking Btrfs support on nodes..."
	@kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' | while read node; do \
		echo "Checking node: $$node"; \
		kubectl debug node/$$node -it --image=alpine:3.18 -- chroot /host btrfs version || echo "Btrfs not available on $$node"; \
	done

# Show logs from the driver
.PHONY: logs
logs:
	kubectl logs -n kube-system -l app=btrfs-csi-driver -c btrfs-csi-driver --tail=100

# Show driver status
.PHONY: status
status:
	@echo "=== CSI Driver Status ==="
	kubectl get csidriver btrfs.csi.k8s.io
	@echo ""
	@echo "=== DaemonSet Status ==="
	kubectl get daemonset -n kube-system btrfs-csi-driver
	@echo ""
	@echo "=== Pod Status ==="
	kubectl get pods -n kube-system -l app=btrfs-csi-driver
	@echo ""
	@echo "=== StorageClass ==="
	kubectl get storageclass btrfs-local

# All-in-one build and deploy
.PHONY: all
all: deps build docker-build deploy

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build        - Build the Go binary"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run tests"
	@echo "  deps         - Download and tidy dependencies"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-push  - Push Docker image to registry"
	@echo "  deploy       - Deploy to Kubernetes"
	@echo "  undeploy     - Remove from Kubernetes"
	@echo "  deploy-test  - Deploy test PVC and Pod"
	@echo "  clean-test   - Remove test resources"
	@echo "  check-btrfs  - Check Btrfs support on nodes"
	@echo "  logs         - Show driver logs"
	@echo "  status       - Show driver status"
	@echo "  all          - Build and deploy everything"
	@echo "  help         - Show this help"