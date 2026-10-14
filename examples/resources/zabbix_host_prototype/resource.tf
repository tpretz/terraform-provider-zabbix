data "zabbix_host" "example" {
  host = "vcenter.example.com"
}

data "zabbix_hostgroup" "virtual_machines" {
  name = "Virtual machines"
}

# Zabbix's own VMware monitoring discovers virtual machines with the
# agent-polled "vmware.vm.discovery" key; this fixture uses a trapper rule
# instead, since this provider has no zabbix_lld_vmware resource yet.
resource "zabbix_lld_trapper" "vms" {
  hostid = data.zabbix_host.example.id
  name   = "Virtual machine discovery"
  key    = "vmware.vm.discovery"
  delay  = "0"
}

# host, group_prototype and tag follow the Zabbix API reference's own
# hostprototype.create example:
# https://www.zabbix.com/documentation/current/en/manual/api/reference/hostprototype/create
resource "zabbix_host_prototype" "vm" {
  ruleid = zabbix_lld_trapper.vms.id
  host   = "{#VM.NAME}"
  group  = [data.zabbix_hostgroup.virtual_machines.id]

  group_prototype {
    name = "{#HV.NAME}"
  }

  tag {
    key   = "datacenter"
    value = "{#DATACENTER.NAME}"
  }
}
