SHELL := /bin/bash

COMPOSE_FILE := docker-compose.test.yml
COMPOSE      ?= docker compose -f $(COMPOSE_FILE)

# --- version matrix ---------------------------------------------------------
#
# ADDING A VERSION is two edits: one entry here and one three-line block in
# $(COMPOSE_FILE). Every per-version target below is generated from VERSIONS.
#
#   VERSIONS  every version the harness knows about
#   GATED     the release-gating subset — `make testacc`
#             8.0 is excluded on purpose: it tracks the `ubuntu-trunk` nightly
#             and must never block a release. `make testall` adds it.
VERSIONS := 60 70 74 80
GATED    := 60 70 74

ZBX_PORT_60 := 8060
ZBX_PORT_70 := 8070
ZBX_PORT_74 := 8074
ZBX_PORT_80 := 8080

# --- how the test suite reaches Zabbix --------------------------------------
#
# By default the suite runs on the host and talks to the published localhost
# ports, so no container is needed to develop. Inside the devcontainer the
# stacks are on the same compose network instead and are reachable by service
# name — devcontainer.json sets ZBX_IN_CONTAINER=1 to switch to that.
ZBX_HOST ?= localhost

ifdef ZBX_IN_CONTAINER
zbx_url = http://zabbix-web-$(1):8080/api_jsonrpc.php
else
zbx_url = http://$(ZBX_HOST):$(ZBX_PORT_$(1))/api_jsonrpc.php
endif

# --- test configuration -----------------------------------------------------

ZABBIX_USER  ?= Admin
ZABBIX_PASS  ?= zabbix
TEST_TIMEOUT ?= 60m
# TEST=<regex> narrows to a single test; TESTARGS passes extra `go test` flags.
TEST         ?=
TESTARGS     ?=

export TF_ACC := 1
export TF_ACC_STATE_LINEAGE := 1

# --- locating the terraform binary ------------------------------------------
#
# terraform-plugin-testing shells out to terraform from a temp working
# directory under $TMPDIR, not from the repo. Version managers that resolve by
# walking up from $PWD (asdf, mise) therefore never see this repo's
# .tool-versions and fall back to the global config — which, if it does not pin
# terraform, fails with
#
#   cannot run Terraform provider tests: error calling terraform version
#   command: exit status 126 / No version is set for command terraform
#
# That looks exactly like a provider bug and is not one. Resolving the absolute
# path here and handing it over via TF_ACC_TERRAFORM_PATH sidesteps the problem
# entirely: the harness runs that binary directly, wherever it is invoked from.
# Falls back to whatever is on PATH when asdf is not in use.
TERRAFORM_BIN ?= $(shell asdf which terraform 2>/dev/null || command -v terraform)
export TF_ACC_TERRAFORM_PATH := $(TERRAFORM_BIN)
export ZABBIX_USER
export ZABBIX_PASS

# Acceptance run for one version. Logs land in provider/acc-<ver>.log so a
# four-version run leaves four separately readable logs.
define run_acc
	@echo "==> zabbix $(1): $(call zbx_url,$(1))"
	ZABBIX_URL=$(call zbx_url,$(1)) \
	TF_ACC_LOG_PATH=$(CURDIR)/provider/acc-$(1).log \
	go test -v -count=1 -timeout $(TEST_TIMEOUT) \
		$(if $(TEST),-run '$(TEST)',) $(TESTARGS) ./provider
endef

.DEFAULT_GOAL := help

# --- build / unit tests -----------------------------------------------------

.PHONY: build
build: ## Build the provider plugin
	go build ./...

.PHONY: vet
vet: ## go vet
	go vet ./...

.PHONY: test
test: ## Unit tests only (no Zabbix server needed)
	TF_ACC= go test ./provider/

# --- test environment -------------------------------------------------------

.PHONY: testenv-up
testenv-up: ## Bring up all Zabbix stacks and wait until their APIs answer
	$(COMPOSE) up -d --wait

.PHONY: testenv-down
testenv-down: ## Tear down all stacks and delete their database volumes
	$(COMPOSE) down -v --remove-orphans

.PHONY: testenv-logs
testenv-logs: ## Follow logs from all stacks
	$(COMPOSE) logs -f --tail=100

.PHONY: testenv-status
testenv-status: ## Show container/health status
	$(COMPOSE) ps

.PHONY: testenv-pull
testenv-pull: ## Pull/refresh all images (picks up a new ubuntu-trunk build)
	$(COMPOSE) pull

.PHONY: testenv-verify
testenv-verify: ## Ask every running stack for its apiinfo.version
	@for v in $(VERSIONS); do \
		url=$$($(MAKE) -s print-url-$$v); \
		printf '%-4s %-46s ' "$$v" "$$url"; \
		curl -s --max-time 10 -X POST -H 'Content-Type: application/json-rpc' \
			-d '{"jsonrpc":"2.0","method":"apiinfo.version","params":{},"id":1}' \
			"$$url" || true; \
		echo; \
	done

# --- acceptance tests -------------------------------------------------------

.PHONY: testacc
testacc: ## Acceptance tests on the release-gating versions (6.0/7.0/7.4)
	@rc=0; for v in $(GATED); do $(MAKE) test$$v || rc=1; done; exit $$rc

.PHONY: testall
testall: ## testacc plus Zabbix 8.0 (trunk, reported but never gating)
	@rc=0; for v in $(GATED); do $(MAKE) test$$v || rc=1; done; \
	$(MAKE) test80 || echo ">>> zabbix 8.0 (ubuntu-trunk) FAILED - non-blocking, investigate but do not gate on it"; \
	exit $$rc

VER ?= 74

.PHONY: test-one
test-one: ## Run a single test: make test-one TEST=TestAccResourceHost VER=74
	@test -n "$(TEST)" || { echo "usage: make test-one TEST=TestAccResourceHost [VER=74]"; exit 1; }
	@$(MAKE) test$(VER) TEST='$(TEST)'

.PHONY: cleanlogs
cleanlogs: ## Remove per-version acceptance logs
	rm -f provider/acc-*.log

# --- generated per-version targets ------------------------------------------

define VERSION_RULES
.PHONY: test$(1) testenv-up-$(1) testenv-down-$(1) testenv-logs-$(1) print-url-$(1)

test$(1): ## Acceptance tests against Zabbix $(1)
	$$(call run_acc,$(1))

testenv-up-$(1): ## Bring up only the Zabbix $(1) stack
	$$(COMPOSE) up -d --wait zabbix-web-$(1)

testenv-down-$(1): ## Tear down only the Zabbix $(1) stack
	$$(COMPOSE) rm -sfv zabbix-web-$(1) zabbix-server-$(1) zabbix-db-$(1)

testenv-logs-$(1): ## Follow logs for the Zabbix $(1) stack
	$$(COMPOSE) logs -f --tail=100 zabbix-db-$(1) zabbix-server-$(1) zabbix-web-$(1)

print-url-$(1):
	@echo '$$(call zbx_url,$(1))'
endef

$(foreach v,$(VERSIONS),$(eval $(call VERSION_RULES,$(v))))

# --- help -------------------------------------------------------------------

.PHONY: help
help: ## Show this help
	@echo "terraform-provider-zabbix - versions: $(VERSIONS) (gating: $(GATED))"
	@echo
	@grep -hE '^[a-zA-Z_0-9-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo
	@echo "  per-version:         test<VER>, testenv-up-<VER>, testenv-down-<VER>, testenv-logs-<VER>"
	@echo "  e.g.                 make testenv-up-74 && make test-one TEST=TestAccResourceHost VER=74"
