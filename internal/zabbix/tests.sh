#!/bin/bash
set -ueo pipefail

echo "Run zabbix checks"

# Check zabbix version
ZABBIX_VERSION=$(curl -s -X POST -H 'Content-Type: application/json-rpc' -d '{"jsonrpc":"2.0","method":"apiinfo.version","id":1,"auth":null,"params":{}}'  "$TEST_ZABBIX_URL" | jq -c -r .result)
if ! [[ "$ZABBIX_VERSION" == "$TEST_ZABBIX_VERSION"* ]]; then 
    echo "Zabbix server version wrong version"
    exit 1
fi

# Check login
ZABBIX_TOKEN=$(curl -s -X POST -H 'Content-Type: application/json-rpc' -d "{\"jsonrpc\":\"2.0\",\"method\":\"user.login\",\"id\":1,\"params\":{\"user\":\"$TEST_ZABBIX_USER\",\"password\":\"$TEST_ZABBIX_PASSWORD\"}}" "$TEST_ZABBIX_URL" | jq -c -r .result)
if [[ "$ZABBIX_VERSION" == "null" ]]; then 
    echo "Zabbix login failed"
    exit 2
fi

# Check request
ZABBIX_USERID=$(curl -s -X POST -H 'Content-Type: application/json-rpc' -d "{\"jsonrpc\":\"2.0\",\"method\":\"user.get\",\"id\":1,\"auth\":\"${ZABBIX_TOKEN}\",\"params\":{}}" "$TEST_ZABBIX_URL" | jq -r -c '.result[] | select(.name=="Zabbix") | .userid')
if [[ $ZABBIX_USERID -ne 1 ]]; then
    echo "Zabbix default user does not match id 1"
    exit 3
fi

echo "Success"
exit 0
