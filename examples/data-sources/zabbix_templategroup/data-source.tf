# requires Zabbix 6.2 or later
data "zabbix_templategroup" "applications" {
  name = "Templates/Applications"
}

resource "zabbix_template" "example" {
  host   = "example-template"
  groups = [data.zabbix_templategroup.applications.id]
}
