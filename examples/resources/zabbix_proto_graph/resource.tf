data "zabbix_host" "example" {
  host = "server-1.example.com"
}

resource "zabbix_lld_agent" "interfaces" {
  hostid = data.zabbix_host.example.id
  name   = "Network interface discovery"
  key    = "net.if.discovery"
  delay  = "1h"
}

resource "zabbix_proto_item_agent" "if_in" {
  hostid    = data.zabbix_host.example.id
  ruleid    = zabbix_lld_agent.interfaces.id
  name      = "Interface {#IFNAME}: bytes received"
  key       = "net.if.in[{#IFNAME}]"
  valuetype = "unsigned"
}

resource "zabbix_proto_graph" "traffic" {
  name   = "Interface {#IFNAME}: traffic"
  width  = "900"
  height = "200"

  item {
    itemid = zabbix_proto_item_agent.if_in.id
    color  = "1A7C11"
  }
}
