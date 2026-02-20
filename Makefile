
export TF_ACC=1
# Where Terraform SDK writes debug logs during acceptance tests.
# Default to /tmp so running as root/non-root (or in CI) doesn't trip over repo permissions.
TF_ACC_LOG_PATH ?= /tmp/terraform-provider-zabbix-acc.log
export TF_ACC_LOG_PATH
export TF_ACC_STATE_LINEAGE=1
export ZABBIX_USER=Admin
export ZABBIX_PASS=zabbix

# Default to host-run acceptance tests (CI-friendly). Override if you run tests from inside the compose network.
ZABBIX_HOST ?= localhost

.PHONY: testacc
testacc: cleanlog test40 test50 test54 test60

.PHONY: cleanlog
cleanlog:
	rm -f "$(TF_ACC_LOG_PATH)"
	touch "$(TF_ACC_LOG_PATH)"
	chmod 666 "$(TF_ACC_LOG_PATH)" || true

.PHONY: test40
test40:
	ZABBIX_URL=http://$(ZABBIX_HOST):8040/api_jsonrpc.php go test -v -failfast ./provider

.PHONY: test50
test50:
	ZABBIX_URL=http://$(ZABBIX_HOST):8050/api_jsonrpc.php go test -v -failfast ./provider

.PHONY: test54
test54:
	ZABBIX_URL=http://$(ZABBIX_HOST):8054/api_jsonrpc.php go test -v -failfast ./provider

.PHONY: test60
test60:
	ZABBIX_URL=http://$(ZABBIX_HOST):8060/api_jsonrpc.php go test -v -failfast ./provider
