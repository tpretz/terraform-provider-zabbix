data "zabbix_host" "example" {
  host = "server-1.example.com"
}

resource "zabbix_item_trapper" "raw_json" {
  hostid    = data.zabbix_host.example.id
  name      = "Application metrics (raw)"
  key       = "app.metrics.raw"
  valuetype = "text"
}

resource "zabbix_item_dependent" "queue_depth" {
  hostid        = data.zabbix_host.example.id
  master_itemid = zabbix_item_trapper.raw_json.id
  name          = "Queue depth"
  key           = "app.queue.depth"
  valuetype     = "unsigned"

  preprocessor {
    type   = "jsonpath"
    params = ["$.queue.depth"]
  }
}
