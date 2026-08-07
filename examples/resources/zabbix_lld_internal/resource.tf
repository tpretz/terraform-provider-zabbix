data "zabbix_host" "example" {
  host = "server-1.example.com"
}

resource "zabbix_lld_internal" "internal" {
  hostid = data.zabbix_host.example.id
  name   = "Internal discovery"
  key    = "zabbix[host,,items]"
  delay  = "1h"
}
