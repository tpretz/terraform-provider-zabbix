data "zabbix_host" "example" {
  host = "server-1.example.com"
}

resource "zabbix_lld_agent" "interfaces" {
  hostid = data.zabbix_host.example.id
  name   = "Network interface discovery"
  key    = "net.if.discovery"
  delay  = "1h"
}

resource "zabbix_proto_item_trapper" "raw" {
  hostid    = data.zabbix_host.example.id
  ruleid    = zabbix_lld_agent.interfaces.id
  name      = "Interface {#IFNAME}: raw"
  key       = "iface.raw[{#IFNAME}]"
  valuetype = "text"
}

resource "zabbix_proto_item_dependent" "errors" {
  hostid        = data.zabbix_host.example.id
  ruleid        = zabbix_lld_agent.interfaces.id
  master_itemid = zabbix_proto_item_trapper.raw.id
  name          = "Interface {#IFNAME}: errors"
  key           = "iface.errors[{#IFNAME}]"
  valuetype     = "unsigned"

  preprocessor {
    type   = "jsonpath"
    params = ["$.errors"]
  }
}
