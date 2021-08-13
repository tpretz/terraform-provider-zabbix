provider "zabbix" {
  # Required
  username = "<api_user>"
  password = "<api_password>"
  url = "http://example.com/api_jsonrpc.php"
  
  # Optional

  # Disable TLS verfication (false by default)
  tls_insecure = true

  # Serialize Zabbix API calls (false by default)
  # Note: race conditions have been observed, enable this if required
  serialize = true
}

