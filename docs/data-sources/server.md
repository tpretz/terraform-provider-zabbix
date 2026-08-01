# zabbix_server (Data Source)

Returns information about the connected Zabbix server.

## Example Usage

```hcl
data "zabbix_server" "current" {}

output "zabbix_version" {
  value = data.zabbix_server.current.version
}
```

## Attributes Reference

| Name    | Type   | Description                          |
|---------|--------|--------------------------------------|
| version | String | Zabbix server version (e.g. `7.0.1`) |
