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

  # Send mutating API calls one at a time. Defaults to true and should stay
  # there: Zabbix has races on concurrent writes, and one of them silently
  # drops a template's triggers. Reads are never serialized, so plan and
  # refresh are unaffected. Set this to false only if you are certain your
  # configuration cannot race and you need the speed.
  serialize = true
}
