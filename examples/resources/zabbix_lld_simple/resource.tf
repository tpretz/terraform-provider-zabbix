data "zabbix_host" "example" {
  host = "server-1.example.com"
}

resource "zabbix_lld_simple" "vmware_vms" {
  hostid = data.zabbix_host.example.id
  name   = "Virtual machine discovery"
  key    = "vmware.vm.discovery[{$VMWARE.URL}]"
  delay  = "1h"
}
