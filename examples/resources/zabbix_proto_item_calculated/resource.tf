data "zabbix_host" "example" {
  host = "server-1.example.com"
}

resource "zabbix_lld_agent" "interfaces" {
  hostid = data.zabbix_host.example.id
  name   = "Network interface discovery"
  key    = "net.if.discovery"
  delay  = "1h"
}

resource "zabbix_proto_item_calculated" "total" {
  hostid    = data.zabbix_host.example.id
  ruleid    = zabbix_lld_agent.interfaces.id
  name      = "Interface {#IFNAME}: total throughput"
  key       = "net.if.total[{#IFNAME}]"
  formula   = "last(//net.if.in[{#IFNAME}])+last(//net.if.out[{#IFNAME}])"
  valuetype = "float"
  delay     = "1m"
}
