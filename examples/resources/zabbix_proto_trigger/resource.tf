data "zabbix_host" "example" {
  host = "server-1.example.com"
}

resource "zabbix_lld_agent" "interfaces" {
  hostid = data.zabbix_host.example.id
  name   = "Network interface discovery"
  key    = "net.if.discovery"
  delay  = "1h"
}

resource "zabbix_proto_item_agent" "if_errors" {
  hostid    = data.zabbix_host.example.id
  ruleid    = zabbix_lld_agent.interfaces.id
  name      = "Interface {#IFNAME}: inbound errors"
  key       = "net.if.in.errors[{#IFNAME}]"
  valuetype = "unsigned"
}

resource "zabbix_proto_trigger" "if_errors" {
  name       = "Errors on interface {#IFNAME}"
  expression = "last(/server-1.example.com/net.if.in.errors[{#IFNAME}])>0"
  priority   = "warn"

  depends_on = [zabbix_proto_item_agent.if_errors]
}
