data "zabbix_host" "example" {
  host = "server-1.example.com"
}

resource "zabbix_item_trapper" "batch_duration" {
  hostid    = data.zabbix_host.example.id
  name      = "Nightly batch duration"
  key       = "batch.duration"
  valuetype = "float"
}
