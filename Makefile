.PHONY: build-backend build-frontend build deploy clean dev-deploy dev-url

# Variables
BACKEND_IMAGE := kubepulse-backend
FRONTEND_IMAGE := kubepulse-frontend
VERSION := latest
DOCKER_REGISTRY := ghcr.io/miketomlin19

# Build the backend Docker image
build-backend:
	cd backend && docker build -t $(DOCKER_REGISTRY)/$(BACKEND_IMAGE):$(VERSION) . && docker push $(DOCKER_REGISTRY)/$(BACKEND_IMAGE):$(VERSION)

# Build the frontend Docker image
build-frontend:
	cd frontend && docker build -t $(DOCKER_REGISTRY)/$(FRONTEND_IMAGE):$(VERSION) . && docker push $(DOCKER_REGISTRY)/$(FRONTEND_IMAGE):$(VERSION)

# Build all images
build: build-backend build-frontend

# Deploy to Kubernetes
deploy:
	kubectl apply -k k8s/base

# Deploy development environment
dev-deploy:
	kubectl apply -k k8s/overlays/dev

# Get the development URLs
dev-url:
	@echo "Backend WebSocket URL: ws://localhost:$$(kubectl get svc kubepulse-backend -o jsonpath='{.spec.ports[0].nodePort}')/ws"
	@echo "Frontend URL: http://localhost:$$(kubectl get svc kubepulse-frontend -o jsonpath='{.spec.ports[0].nodePort}')"

# Clean up Kubernetes resources
clean:
	kubectl delete -k k8s/base

# Run backend locally (for development)
run-backend:
	cd backend && go run cmd/server/main.go

# Run frontend locally (for development)
run-frontend:
	cd frontend && npm start

# Install dependencies
deps:
	cd backend && go mod tidy
	cd frontend && npm install

.DEFAULT_GOAL := build