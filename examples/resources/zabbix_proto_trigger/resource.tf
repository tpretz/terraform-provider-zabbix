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

# Build the expression from references rather than repeating the host and key as
# literals. Terraform sees zabbix_proto_item_agent.if_errors in the interpolation
# and orders the trigger prototype after the item prototype on its own, so no
# depends_on is needed.
#
# The {#IFNAME} macro is part of the item prototype's key, so referencing the key
# carries it across -- there is nothing to keep in step by hand.
resource "zabbix_proto_trigger" "if_errors" {
  name       = "Errors on interface {#IFNAME}"
  expression = "last(/${data.zabbix_host.example.host}/${zabbix_proto_item_agent.if_errors.key})>0"
  priority   = "warn"
}
