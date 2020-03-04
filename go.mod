module github.com/tpretz/terraform-provider-zabbix

go 1.14

require (
	github.com/hashicorp/terraform-plugin-sdk v1.7.0
	github.com/hashicorp/tf-sdk-migrator v1.1.0 // indirect
	github.com/tpretz/go-zabbix-api v0.4.0
)

//replace github.com/tpretz/go-zabbix-api => ../go-zabbix-api
