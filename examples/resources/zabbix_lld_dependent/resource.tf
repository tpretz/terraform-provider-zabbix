data "zabbix_host" "example" {
  host = "server-1.example.com"
}

resource "zabbix_item_trapper" "raw_inventory" {
  hostid    = data.zabbix_host.example.id
  name      = "Inventory (raw)"
  key       = "inventory.raw"
  valuetype = "text"
}

# a dependent discovery rule is driven by its master item, so delay must be 0
resource "zabbix_lld_dependent" "disks" {
  hostid        = data.zabbix_host.example.id
  master_itemid = zabbix_item_trapper.raw_inventory.id
  name          = "Disk discovery"
  key           = "disk.discovery"
  delay         = "0"

  preprocessor {
    type   = "12"
    params = ["$.disks"]
  }
}
