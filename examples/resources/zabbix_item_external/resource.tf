data "zabbix_host" "example" {
  host = "server-1.example.com"
}

resource "zabbix_item_external" "cert_expiry" {
  hostid    = data.zabbix_host.example.id
  name      = "TLS certificate days remaining"
  key       = "check_cert[{HOST.CONN},443]"
  valuetype = "unsigned"
  delay     = "1h"
}
