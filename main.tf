provider "zabbix" { 
  username = "Admin"
  password = "zabbix"
  url = "http://127.0.0.1:8081/api_jsonrpc.php"
}

data "zabbix_host" "test" {
  host = "Zabbix server"
}

data "zabbix_hostgroup" "a" {
  name = "Hypervisors"
}

data "zabbix_template" "a" {
  host = "Template App FTP Service"
}

resource "zabbix_hostgroup" "a" {
  name = "test group"
}

resource "zabbix_item_trapper" "a" {
  hostid = data.zabbix_host.test.id
  key = "abc_def"
  name = "ABC DEF"
  valuetype = "float"
}

resource "zabbix_item_http" "a" {
  hostid = zabbix_host.a.id
  key = "http_one"
  name = "http one"
  valuetype = "text"

  url = "http://google.com"
  interfaceid = zabbix_host.a.interface[0].id
  verify_host = true

  preprocessor {
    type = "5"
    params = "^test$\nbob"
  }
}

resource "zabbix_trigger" "b" {
  description = "test trigger"
  expression = "{${data.zabbix_host.test.host}:${zabbix_item_trapper.a.key}.nodata(120)}=1"
  dependencies = [zabbix_trigger.c.id]
}
resource "zabbix_trigger" "c" {
  description = "test trigger"
  expression = "{${data.zabbix_host.test.host}:${zabbix_item_trapper.a.key}.nodata(240)}=1"
}

resource "zabbix_template" "a" {
  groups = [zabbix_hostgroup.a.id]
  host = "example template"
  name = "visible name"

  macro {
    name = "{$SOMETHING}"
    value = "GOOD"
  }
}

resource "zabbix_host" "a" {
  host = "test.isp.dev"
  groups = [zabbix_hostgroup.a.id, data.zabbix_hostgroup.a.id]
  templates = [zabbix_template.a.id, data.zabbix_template.a.id]
  
  interface {
    dns = "test.isp.cdev"
    type = "snmp"
  }
  interface {
    dns = "test.isp.adev"
    main = true
  }
  
  macro {
    name = "{$SOMETHING}"
    value = "ELSE"
  }
}
