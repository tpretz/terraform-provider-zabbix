# active proxy: the proxy connects to the server
resource "zabbix_proxy" "active" {
  name              = "proxy-eu-west"
  description       = "managed by terraform"
  allowed_addresses = "10.0.0.1,10.0.0.2"

  tls_accept       = "psk"
  tls_psk_identity = "proxy-eu-west"
  tls_psk          = "b7e3b0e9a94e4f2a8d1c6f5b3a2e9d8c7b6a5f4e3d2c1b0a9f8e7d6c5b4a3f2e"
}

# passive proxy: the server connects to the proxy
resource "zabbix_proxy" "passive" {
  name           = "proxy-dmz"
  operating_mode = "passive"
  address        = "proxy-dmz.example.com"
  port           = "10051"
}

resource "zabbix_hostgroup" "example" {
  name = "Remote sites"
}

resource "zabbix_host" "example" {
  host    = "server-1.example.com"
  groups  = [zabbix_hostgroup.example.id]
  proxyid = zabbix_proxy.active.id

  interface {
    type = "agent"
    ip   = "10.0.0.10"
  }
}
