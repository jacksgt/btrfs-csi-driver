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

# Run sanity tests (requires Btrfs)
.PHONY: test-sanity
test-sanity:
	$(GOTEST) -v -tags=btrfs ./internal/driver -run TestSanity

# Run all tests including sanity tests
.PHONY: test-all
test-all:
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

# Get logs from Kubernetes
.PHONY: logs-csi
logs-csi:
	kubectl logs -n kube-system -l app=btrfs-csi-driver -c csi-provisioner --tail=-1
	kubectl logs -n kube-system -l app=btrfs-csi-driver -c btrfs-csi-driver --tail=-1
	kubectl logs -n kube-system -l app=btrfs-csi-driver -c node-driver-registrar --tail=-1
	kubectl logs -n kube-system -l app=btrfs-csi-driver -c csi-resizer --tail=-1

# Deploy test resources
.PHONY: deploy-test
deploy-test:
	kubectl apply -f test/kubernetes/

# Clean up test resources
.PHONY: clean-test
clean-test:
	kubectl delete -f test/kubernetes/ --ignore-not-found=true

# Test volume expansion
.PHONY: test-expansion
test-expansion:
	@echo "Testing volume expansion..."
	kubectl apply -f test/kubernetes/test-volume-expansion.yaml
	@echo "Expanding PVC from 1Gi to 2Gi..."
	kubectl patch pvc btrfs-expansion-test-pvc -p '{"spec":{"resources":{"requests":{"storage":"2Gi"}}}}'
	@echo "Waiting for expansion to complete..."
	kubectl wait --for=condition=FileSystemResizePending pvc/btrfs-expansion-test-pvc --timeout=60s || true
	@echo "Checking PVC status..."
	kubectl get pvc btrfs-expansion-test-pvc

# Check if Btrfs is available on nodes
.PHONY: check-btrfs
check-btrfs:
	@echo "Checking Btrfs support on nodes..."
	@kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' | while read node; do \
		echo "Checking node: $$node"; \
		kubectl debug node/$$node -it --image=alpine:3.18 -- chroot /host btrfs version || echo "Btrfs not available on $$node"; \
	done

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

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build        - Build the Go binary"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run tests"
	@echo "  test-sanity  - Run CSI sanity tests (requires Btrfs)"
	@echo "  test-all     - Run all tests including sanity tests"
	@echo "  deps         - Download and tidy dependencies"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-push  - Push Docker image to registry"
	@echo "  deploy-csi   - Deploy CSI driver to Kubernetes"
	@echo "  undeploy-csi - Remove CSI driver from Kubernetes"
	@echo "  deploy-test        - Deploy test PVC and Pod"
	@echo "  deploy-expansion-test - Deploy volume expansion test resources"
	@echo "  test-expansion     - Test volume expansion functionality"
	@echo "  clean-test         - Remove test resources"
	@echo "  check-btrfs        - Check Btrfs support on nodes"
	@echo "  status             - Show driver status"
	@echo "  all                - Build and deploy everything"
	@echo "  help               - Show this help"