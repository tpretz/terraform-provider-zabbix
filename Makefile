
export TF_ACC=1
export TF_ACC_LOG_PATH=acc.log
export TF_ACC_STATE_LINEAGE=1
export ZABBIX_USER=Admin
export ZABBIX_PASS=zabbix

.PHONY: testacc
testacc: cleanlog test40 test50 test54 test60

.PHONY: cleanlog
cleanlog:
	rm provider/acc.log || true

.PHONY: test40
test40:
	ZABBIX_URL=http://zabbix-web-40:8080/api_jsonrpc.php go test -v ./provider

.PHONY: test50
test50:
	ZABBIX_URL=http://zabbix-web-50:8080/api_jsonrpc.php go test -v ./provider

.PHONY: test54
test54:
	ZABBIX_URL=http://zabbix-web-54:8080/api_jsonrpc.php go test -v ./provider

.PHONY: test60
test60:
	ZABBIX_URL=http://zabbix-web-60:8080/api_jsonrpc.php go test -v ./provider
