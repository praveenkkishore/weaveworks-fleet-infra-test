# Makefile for Cato Terraform Go SDK

# Configuration
export CATO_TOKEN ?= R=eu1|K=D5ACD90B5FFC9EE916E04192AD048C70
export CATO_ACCOUNT_ID ?= 17957

# Default site configuration
SITE_NAME ?= Praveen-IPsec-BGP-Site
PUBLIC_IP ?= 1.1.1.1
BGP_IP ?= 169.254.200.1
BGP_ASN ?= 65100
PSK ?= praveen_infoblox
NETWORK ?= 10.201.1.0/24

# Binary output
BINARY_NAME = cato-terraform
BINARY_DIR = bin

.PHONY: help
help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

.PHONY: deps
deps: ## Download Go dependencies
	@echo "→ Downloading dependencies..."
	go mod download
	go mod tidy

.PHONY: build
build: deps ## Build the binary
	@echo "→ Building $(BINARY_NAME)..."
	@mkdir -p ../$(BINARY_DIR)
	go build -o ../$(BINARY_DIR)/$(BINARY_NAME) ./cmd/cato-terraform

.PHONY: run
run: ## Run the application (create resources)
	@echo "→ Running Cato Terraform executor..."
	@echo "  CATO_TOKEN: $(shell echo $$CATO_TOKEN | cut -c1-20)..."
	@echo "  CATO_ACCOUNT_ID: $(CATO_ACCOUNT_ID)"
	@echo ""
	go run cmd/cato-terraform/main.go \
		--site-name="$(SITE_NAME)" \
		--public-ip="$(PUBLIC_IP)" \
		--bgp-ip="$(BGP_IP)" \
		--bgp-asn=$(BGP_ASN) \
		--psk="$(PSK)" \
		--network="$(NETWORK)"

.PHONY: create
create: run ## Alias for 'run' - create resources

.PHONY: destroy
destroy: ## Destroy resources
	@echo "→ Destroying Cato resources..."
	go run cmd/cato-terraform/main.go --destroy

.PHONY: run-binary
run-binary: build ## Build and run the binary
	@echo "→ Running $(BINARY_NAME) binary..."
	../$(BINARY_DIR)/$(BINARY_NAME) \
		--site-name="$(SITE_NAME)" \
		--public-ip="$(PUBLIC_IP)" \
		--bgp-ip="$(BGP_IP)" \
		--bgp-asn=$(BGP_ASN) \
		--psk="$(PSK)" \
		--network="$(NETWORK)"

.PHONY: test
test: ## Run tests
	@echo "→ Running tests..."
	go test -v ./...

.PHONY: fmt
fmt: ## Format Go code
	@echo "→ Formatting code..."
	go fmt ./...

.PHONY: lint
lint: ## Run linter (requires golangci-lint)
	@echo "→ Running linter..."
	golangci-lint run

.PHONY: clean
clean: ## Clean build artifacts
	@echo "→ Cleaning..."
	rm -rf ../$(BINARY_DIR)
	go clean

.PHONY: env
env: ## Show environment variables
	@echo "Environment Configuration:"
	@echo "  CATO_TOKEN: $(shell echo $$CATO_TOKEN | cut -c1-20)..."
	@echo "  CATO_ACCOUNT_ID: $(CATO_ACCOUNT_ID)"
	@echo "  SITE_NAME: $(SITE_NAME)"
	@echo "  PUBLIC_IP: $(PUBLIC_IP)"
	@echo "  BGP_IP: $(BGP_IP)"
	@echo "  BGP_ASN: $(BGP_ASN)"
	@echo "  PSK: $(PSK)"
	@echo "  NETWORK: $(NETWORK)"

.PHONY: custom
custom: ## Run with custom parameters (use: make custom SITE_NAME="My-Site" BGP_IP="169.254.201.1")
	@echo "→ Running with custom parameters..."
	go run cmd/cato-terraform/main.go \
		--site-name="$(SITE_NAME)" \
		--public-ip="$(PUBLIC_IP)" \
		--bgp-ip="$(BGP_IP)" \
		--bgp-asn=$(BGP_ASN) \
		--psk="$(PSK)" \
		--network="$(NETWORK)"

# Example: Create a different site
.PHONY: example-site2
example-site2: ## Example: Create second site with different IP
	@echo "→ Creating second site example..."
	go run cmd/cato-terraform/main.go \
		--site-name="Praveen-IPsec-BGP-Site-2" \
		--public-ip="2.2.2.2" \
		--bgp-ip="169.254.201.1" \
		--bgp-asn=65101 \
		--psk="praveen_infoblox_2" \
		--network="10.202.1.0/24"
