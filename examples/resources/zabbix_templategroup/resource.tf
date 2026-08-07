# template groups were split out of host groups in Zabbix 6.2
resource "zabbix_templategroup" "applications" {
  name = "Templates/Applications"
}

resource "zabbix_template" "example" {
  host   = "example-template"
  groups = [zabbix_templategroup.applications.id]
}
