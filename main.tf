provider "zabbix" { 
  username = "Admin"
  password = "zabbix"
  url = "http://127.0.0.1:8081/api_jsonrpc.php"
}

data "zabbix_host" "test" {
  host = "Zabbix server"
}

resource "zabbix_item_trapper" "a" {
  hostid = data.zabbix_host.test.id
  key = "abc_def"
  name = "ABC DEF"
  valuetype = 1
}

resource "zabbix_trigger" "b" {
  description = "test trigger"
  expression = "{${data.zabbix_host.test.host}:${zabbix_item_trapper.a.key}.nodata(120)}=1"
}
