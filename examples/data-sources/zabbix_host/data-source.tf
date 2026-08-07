data "zabbix_host" "example" {
  host = "server-1.example.com"
}

output "host_id" {
  value = data.zabbix_host.example.id
}
