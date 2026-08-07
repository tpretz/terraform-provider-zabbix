data "zabbix_host" "example" {
  host = "server-1.example.com"
}

resource "zabbix_lld_http" "services" {
  hostid = data.zabbix_host.example.id
  name   = "Service discovery"
  key    = "service.discovery"
  url    = "https://app.example.com/services"
  delay  = "1h"

  headers = {
    Accept = "application/json"
  }

  macro {
    macro = "{#SERVICE}"
    path  = "$.name"
  }
}
