data "zabbix_host" "example" {
  host = "server-1.example.com"
}

resource "zabbix_lld_agent" "interfaces" {
  hostid = data.zabbix_host.example.id
  name   = "Network interface discovery"
  key    = "net.if.discovery"
  delay  = "1h"
}

resource "zabbix_proto_item_http" "endpoint_health" {
  hostid    = data.zabbix_host.example.id
  ruleid    = zabbix_lld_agent.interfaces.id
  name      = "Endpoint {#IFNAME}: health"
  key       = "endpoint.health[{#IFNAME}]"
  url       = "https://app.example.com/{#IFNAME}/healthz"
  valuetype = "text"
  delay     = "1m"
}
