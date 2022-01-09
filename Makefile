

testacc:
	TF_ACC=1 ZABBIX_URL=http://zabbix-web-40:8080/api_jsonrpc.php ZABBIX_USER=Admin ZABBIX_PASS=zabbix go test ./provider
	TEST_GT5=1 TF_ACC=1 ZABBIX_URL=http://zabbix-web-50:8080/api_jsonrpc.php ZABBIX_USER=Admin ZABBIX_PASS=zabbix go test ./provider
	TEST_GT5=1 TF_ACC=1 ZABBIX_URL=http://zabbix-web-54:8080/api_jsonrpc.php ZABBIX_USER=Admin ZABBIX_PASS=zabbix go test ./provider