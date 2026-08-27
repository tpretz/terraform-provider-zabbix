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

# --- toolchain pin consistency ----------------------------------------------
#
# The Go version is pinned in three places and they have to stay consistent:
#
#   go.mod `go 1.25.8`          the floor we advertise -- the minimum the SDK
#                               needs, and what a consumer building from source
#                               must have
#   .tool-versions `golang`     what we actually build, test and generate docs
#                               with, and so >= the directive
#   ci.yml/nightly.yml          `go-version-file: go.mod`, so CI follows the
#                               directive automatically and is never a third
#                               number to edit
#
# A dependency bot can move the first without the other two: raising the go
# directive is a routine side effect of a terraform-plugin-sdk bump. With
# GOTOOLCHAIN=auto that does not fail loudly -- Go silently downloads a newer
# toolchain and the .tool-versions pin quietly stops being what anything runs.
# This target makes it fail loudly instead, and CI runs it.
.PHONY: check-toolchain
check-toolchain: ## Fail if .tool-versions' Go pin is older than go.mod's directive
	@modgo=$$(awk '$$1=="go" && $$2 ~ /^[0-9]/ {print $$2; exit}' go.mod); \
	pingo=$$(awk '$$1=="golang" {print $$2; exit}' .tool-versions); \
	test -n "$$modgo" || { echo "no go directive found in go.mod"; exit 1; }; \
	test -n "$$pingo" || { echo "no golang pin found in .tool-versions"; exit 1; }; \
	awk -v a="$$modgo" -v b="$$pingo" 'BEGIN { \
		na = split(a, A, "."); nb = split(b, B, "."); \
		for (i = 1; i <= 3; i++) { \
			x = (i <= na ? A[i] + 0 : 0); y = (i <= nb ? B[i] + 0 : 0); \
			if (x > y) exit 1; if (x < y) exit 0 } exit 0 }' || { \
		echo; \
		echo ">>> go.mod requires go $$modgo but .tool-versions pins golang $$pingo."; \
		echo ">>> Raise the .tool-versions pin to at least $$modgo and commit both."; \
		exit 1; \
	}; \
	echo "toolchain pins consistent: go.mod $$modgo <= .tool-versions $$pingo"

# --- documentation ----------------------------------------------------------
#
# docs/ is generated from three inputs: the Description on every schema
# attribute, the prose templates in templates/, and the runnable HCL in
# examples/. Never hand-edit a file under docs/ -- the next `make docs` will
# throw the edit away. If a generated page reads badly, the fix is almost
# always a better schema Description, which reaches `terraform providers
# schema -json` and editor tooling too.
#
# Pinned rather than floating: tfplugindocs output changes between releases,
# so an unpinned version would make `docs-check` fail on somebody else's
# machine for no reason. v0.25.0 needs go >= 1.25.8, which .tool-versions
# satisfies (1.25.12); v0.24.0 and below are the fallback for an older
# toolchain, at the cost of a slightly different rendering.
TFPLUGINDOCS_VERSION ?= v0.25.0
TFPLUGINDOCS         := go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@$(TFPLUGINDOCS_VERSION)

# tfplugindocs runs `terraform init` from a temp directory, so it hits exactly
# the version-manager shim problem TF_ACC_TERRAFORM_PATH solves for the test
# harness -- but it has no equivalent flag, so the resolved binary's directory
# goes on PATH ahead of the shim instead.
DOCS_PATH := $(dir $(TERRAFORM_BIN)):$(PATH)

.PHONY: docs
docs: ## Regenerate docs/ from the provider schema, templates/ and examples/
	PATH="$(DOCS_PATH)" $(TFPLUGINDOCS) generate --provider-name zabbix --rendered-provider-name Zabbix

.PHONY: docs-check
docs-check: ## Fail if docs/ is out of date with the schema (CI gate)
	@$(MAKE) docs
	@git diff --quiet --exit-code -- docs/ || { \
		echo; \
		echo ">>> docs/ is out of date. Run 'make docs' and commit the result."; \
		git --no-pager diff --stat -- docs/; \
		exit 1; \
	}
	@echo "docs/ is up to date"

.PHONY: docs-validate
docs-validate: ## Check docs/ against the Terraform Registry's layout rules
	PATH="$(DOCS_PATH)" $(TFPLUGINDOCS) validate --provider-name zabbix

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

.PHONY: sweep
sweep: ## Delete leftover test-* objects from every stack (recovers an aborted run)
	@rc=0; for v in $(VERSIONS); do \
		printf '==> sweeping zabbix %-4s ' "$$v"; \
		ZABBIX_URL=$$($(MAKE) -s print-url-$$v) \
			go test ./provider/ -sweep=all >/dev/null 2>&1 \
			&& echo ok || { echo FAILED; rc=1; }; \
	done; exit $$rc

# --- the release gate -------------------------------------------------------
#
# One target for RELEASING.md step 0, so a full check is one command and one
# walk-away rather than a dozen prompts. Takes roughly 30 minutes: the local
# checks are seconds, the four acceptance runs are ~6.5 minutes each.
#
# Sweeps before each version rather than once at the start, because a failure
# part-way through leaves objects behind that would fail the next version with
# "already exists" and send you hunting a bug that is not there.
#
# 8.0 runs but never gates, matching the support policy. Its result is printed
# at the end either way.
.PHONY: release-gate
release-gate: ## Everything RELEASING.md step 0 checks, in one run (~30 min)
	@echo "=== local checks ==="
	@$(MAKE) --no-print-directory check-toolchain
	@$(MAKE) --no-print-directory build
	@$(MAKE) --no-print-directory vet
	@out=$$(gofmt -l .); test -z "$$out" || { echo "gofmt: $$out"; exit 1; }; echo "gofmt: clean"
	@go test ./... -count=1 || exit 1
	@$(MAKE) --no-print-directory docs-check
	@echo
	@echo "=== acceptance matrix ==="
	@rc=0; fail=""; \
	for v in $(GATED); do \
		ZABBIX_URL=$$($(MAKE) -s print-url-$$v) go test ./provider/ -sweep=all >/dev/null 2>&1; \
		if $(MAKE) --no-print-directory test$$v >/dev/null 2>&1; then \
			echo "zabbix $$v: PASS"; \
		else \
			echo "zabbix $$v: FAIL  (see provider/acc-$$v.log)"; rc=1; fail="$$fail $$v"; \
		fi; \
	done; \
	ZABBIX_URL=$$($(MAKE) -s print-url-80) go test ./provider/ -sweep=all >/dev/null 2>&1; \
	if $(MAKE) --no-print-directory test80 >/dev/null 2>&1; then \
		echo "zabbix 80: PASS  (non-gating)"; \
	else \
		echo "zabbix 80: FAIL  (non-gating - investigate, do not block on it)"; \
	fi; \
	echo; \
	if [ $$rc -eq 0 ]; then \
		echo "RELEASE GATE: PASS - gating versions $(GATED) all green"; \
	else \
		echo "RELEASE GATE: FAIL -$$fail"; \
	fi; \
	exit $$rc

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
