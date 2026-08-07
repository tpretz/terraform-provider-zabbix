resource "zabbix_templategroup" "applications" {
  name = "Templates/Applications"
}

resource "zabbix_template" "example" {
  host        = "example-application"
  name        = "Example application"
  description = "Managed by Terraform"
  groups      = [zabbix_templategroup.applications.id]

  macro {
    name  = "{$APP.PORT}"
    value = "8080"
  }
}
