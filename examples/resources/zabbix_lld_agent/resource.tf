data "zabbix_host" "example" {
  host = "server-1.example.com"
}

resource "zabbix_lld_agent" "interfaces" {
  hostid   = data.zabbix_host.example.id
  name     = "Network interface discovery"
  key      = "net.if.discovery"
  delay    = "1h"
  lifetime = "30d"

  # skip loopback and virtual interfaces
  condition {
    macro    = "{#IFNAME}"
    value    = "^(lo|veth|docker)"
    operator = "notmatch"
  }
}
