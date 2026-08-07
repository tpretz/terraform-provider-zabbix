data "zabbix_host" "example" {
  host = "server-1.example.com"
}

resource "zabbix_item_http" "healthcheck" {
  hostid    = data.zabbix_host.example.id
  name      = "Application health"
  key       = "app.health"
  url       = "https://app.example.com/healthz"
  valuetype = "text"
  delay     = "30s"

  request_method = "get"
  status_codes   = "200"
  timeout        = "5s"

  headers = {
    Accept = "application/json"
  }
}
