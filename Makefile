.PHONY: help build run test clean deps migrate

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

deps: ## Install dependencies
	go mod download
	go mod tidy

build: ## Build the application
	go build -o bin/server cmd/server/main.go

run: ## Run the application
	go run cmd/server/main.go

test: ## Run tests
	go test -v ./...

clean: ## Clean build artifacts
	rm -rf bin/

migrate-up: ## Run database migrations up
	@echo "Database migrations are auto-applied on startup"

migrate-down: ## Run database migrations down
	@echo "Manual migration rollback not implemented yet"

docker-build: ## Build Docker image
	docker build -t onedrive-storage:latest -f docker/Dockerfile .

docker-run: ## Run Docker container
	docker-compose -f docker/docker-compose.yaml up

lint: ## Run linter
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping"; \
	fi

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...
