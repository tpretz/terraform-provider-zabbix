terraform {
  required_providers {
    zabbix = {
      source  = "tpretz/zabbix"
      version = "~> 2.0"
    }
  }
}

provider "zabbix" {
  url = "https://zabbix.example.com/api_jsonrpc.php"

  # Authenticate with a username and password ...
  username = "Admin"
  password = "zabbix"

  # ... or with an API token, in which case username and password are unused.
  # token = "..."

  # Every argument above has an environment fallback, so credentials need not
  # be written into configuration at all:
  #
  #   ZABBIX_URL   (or ZABBIX_SERVER_URL)
  #   ZABBIX_USER  (or ZABBIX_USERNAME)
  #   ZABBIX_PASS  (or ZABBIX_PASSWORD)
  #   ZABBIX_TOKEN

  # Skip TLS certificate verification. Testing only.
  tls_insecure = false

  # Serialise API calls. Zabbix has known races on concurrent writes to the
  # same object; enable this if you see intermittent API errors under
  # parallelism.
  serialize = false
}
