data "zabbix_host" "example" {
  host = "server-1.example.com"
}

resource "zabbix_lld_external" "mounts" {
  hostid = data.zabbix_host.example.id
  name   = "Mount point discovery"
  key    = "discover_mounts[{HOST.CONN}]"
  delay  = "1h"
}
