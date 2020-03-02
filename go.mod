module github.com/tpretz/terraform-provider-zabbix

require (
	github.com/hashicorp/terraform-plugin-sdk v1.7.0
	github.com/tpretz/go-zabbix-api v0.3.2-0.20200123085336-cb1c1795df50
)

replace github.com/tpretz/go-zabbix-api => ../go-zabbix-api
