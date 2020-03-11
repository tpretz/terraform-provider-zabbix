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

resource "zabbix_item_internal" "a" {
  hostid = data.zabbix_host.test.id
  key = "zabbix[hosts]"
  name = "internal_test"
  valuetype = "float"
}

resource "zabbix_item_snmp" "a" {
  hostid = zabbix_template.a.id
  key = "snmptesta"
  name = "snmp test a"
  valuetype = "float"
  snmp_version = 2
  snmp_oid = "DISMAN-EVENT-MIB::sysUpTimeInstance"
}

resource "zabbix_item_snmp" "b" {
  hostid = zabbix_template.a.id
  key = "snmptestb"
  name = "snmp test b"
  valuetype = "float"
  snmp_version = 3
  snmp_oid = "DISMAN-EVENT-MIB::sysUpTimeInstance"
}

resource "zabbix_item_simple" "ping" {
  hostid = zabbix_template.a.id
  key = "icmpping[,,,,]"
  name = "Ping"
  valuetype = "float"
}

resource "zabbix_item_agent" "a" {
  hostid = zabbix_template.a.id
  key = "agent.hostname"
  name = "Hostname"
  valuetype = "text"
}

resource "zabbix_trigger" "ping" {
  description = "Ping"
  expression = "{${zabbix_template.a.host}:${zabbix_item_simple.ping.key}.last()}=1"
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
  host = "host.example.com"
  groups = [zabbix_hostgroup.a.id, data.zabbix_hostgroup.a.id]
  templates = [zabbix_template.a.id, data.zabbix_template.a.id]
  
  interface {
    dns = "eth0.host.example.com"
    type = "snmp"
  }
  interface {
    dns = "eth0.host.example.com"
    main = true
  }
  
  macro {
    name = "{$SOMETHING}"
    value = "ELSE"
  }
}
