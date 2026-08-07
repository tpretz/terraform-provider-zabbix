data "zabbix_host" "example" {
  host = "server-1.example.com"
}

resource "zabbix_lld_snmp" "interfaces" {
  hostid   = data.zabbix_host.example.id
  name     = "SNMP interface discovery"
  key      = "if.discovery"
  snmp_oid = "discovery[{#IFNAME},1.3.6.1.2.1.31.1.1.1.1]"
  delay    = "1h"

  condition {
    macro    = "{#IFNAME}"
    value    = "^lo$"
    operator = "notmatch"
  }
}
