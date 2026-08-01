terraform {
  required_providers {
    zabbix = {
      # After `make install`: use "local/zabbix"
      # From the Terraform/OpenTofu registry: use "registry.terraform.io/tpretz/zabbix"
      source  = "local/zabbix"
      version = "1.0.0"
    }
  }
}

# Option 1: API token (required when the account uses MFA)
provider "zabbix" {
  url       = "http://zabbix.example.com/api_jsonrpc.php"
  api_token = var.zabbix_api_token
}

# Option 2: username + password
# provider "zabbix" {
#   url      = "http://zabbix.example.com/api_jsonrpc.php"
#   username = var.zabbix_user
#   password = var.zabbix_password
# }
#
# All arguments can also be set with environment variables:
#   ZABBIX_URL, ZABBIX_API_TOKEN, ZABBIX_USER (or ZABBIX_USERNAME),
#   ZABBIX_PASS (or ZABBIX_PASSWORD)
