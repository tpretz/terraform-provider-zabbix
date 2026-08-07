data "zabbix_host" "example" {
  host = "server-1.example.com"
}

resource "zabbix_lld_agent" "interfaces" {
  hostid = data.zabbix_host.example.id
  name   = "Network interface discovery"
  key    = "net.if.discovery"
  delay  = "1h"
}

resource "zabbix_proto_item_snmp" "if_in_octets" {
  hostid    = data.zabbix_host.example.id
  ruleid    = zabbix_lld_agent.interfaces.id
  name      = "Interface {#IFNAME}: bytes received"
  key       = "ifHCInOctets[{#SNMPINDEX}]"
  snmp_oid  = "1.3.6.1.2.1.31.1.1.1.6.{#SNMPINDEX}"
  valuetype = "unsigned"
  delay     = "1m"
}
