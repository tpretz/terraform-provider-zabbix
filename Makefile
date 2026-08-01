PKG     = ./...
BINARY  = terraform-provider-zabbix
VERSION ?= 1.0.0
NAMESPACE ?= local
OS_ARCH ?= linux_amd64

# Both Terraform and OpenTofu search ~/.terraform.d/plugins with no CLI
# configuration at all. They differ only in the default registry hostname, so
# install under both to cover either tool.
PLUGIN_ROOT = $(HOME)/.terraform.d/plugins
TF_DIR   = $(PLUGIN_ROOT)/registry.terraform.io/$(NAMESPACE)/zabbix/$(VERSION)/$(OS_ARCH)
TOFU_DIR = $(PLUGIN_ROOT)/registry.opentofu.org/$(NAMESPACE)/zabbix/$(VERSION)/$(OS_ARCH)

.PHONY: build vet lint install testacc test60 test70

build:
	go build $(PKG)

vet:
	go vet $(PKG)

lint:
	golangci-lint run $(PKG)

# Install locally for both Terraform and OpenTofu.
# Reference it as:
#   terraform { required_providers { zabbix = { source = "$(NAMESPACE)/zabbix", version = "$(VERSION)" } } }
install:
	mkdir -p $(TF_DIR) $(TOFU_DIR)
	go build -o $(TF_DIR)/$(BINARY)_v$(VERSION) .
	cp $(TF_DIR)/$(BINARY)_v$(VERSION) $(TOFU_DIR)/$(BINARY)_v$(VERSION)
	@echo "installed $(NAMESPACE)/zabbix $(VERSION) for terraform and opentofu"

# Run the acceptance tests against an already-running Zabbix.
# Requires ZABBIX_URL, ZABBIX_USER and ZABBIX_PASS to be set, e.g:
#   ZABBIX_URL=http://localhost:8070/api_jsonrpc.php \
#   ZABBIX_USER=Admin ZABBIX_PASS=zabbix make testacc
testacc:
	TF_ACC=1 go test ./provider/... -v -timeout 30m

test60:
	docker compose -f docker/docker-compose.yml up -d
	@echo "Waiting for Zabbix 6.0 to become ready..."
	@sleep 60
	ZABBIX_URL=http://localhost:8081/api_jsonrpc.php \
	ZABBIX_USER=Admin \
	ZABBIX_PASS=zabbix \
	TF_ACC=1 \
	go test ./provider/... -v -timeout 30m
	docker compose -f docker/docker-compose.yml down

test70:
	docker compose -f docker-compose.zabbix70.yml up -d
	@echo "Waiting for Zabbix 7.0 to become ready..."
	@sleep 90
	ZABBIX_URL=http://localhost:8070/api_jsonrpc.php \
	ZABBIX_USER=Admin \
	ZABBIX_PASS=zabbix \
	TF_ACC=1 \
	go test ./provider/... -v -timeout 30m
	docker compose -f docker-compose.zabbix70.yml down
