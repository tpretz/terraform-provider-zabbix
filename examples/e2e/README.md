# End-to-end verification config

A single configuration exercising every resource family the provider supports:
template groups, host groups, templates with macros, four item types, item
preprocessing, tags, triggers, graphs with multiple items, LLD rules with item
prototypes, hosts with inventory/macros/template linkage, and all data sources.

Each `*_lookup_matches` output asserts that a data source resolved to the same
object the corresponding resource created, so a successful apply with all
outputs `true` is a meaningful smoke test.

## Running it

Build the provider into a local plugin mirror and point the CLI at it:

```shell
PLUGIN_DIR=$PWD/plugins/registry.opentofu.org/local/zabbix/1.0.0/linux_amd64
mkdir -p "$PLUGIN_DIR"
(cd ../.. && go build -o "$PLUGIN_DIR/terraform-provider-zabbix_v1.0.0" .)

cat > tofurc <<'CFG'
provider_installation {
  filesystem_mirror {
    path    = "PLUGINS_PATH"
    include = ["local/*"]
  }
  direct {
    exclude = ["local/*"]
  }
}
CFG
sed -i "s|PLUGINS_PATH|$PWD/plugins|" tofurc

export TF_CLI_CONFIG_FILE=$PWD/tofurc
export ZABBIX_URL=http://localhost:8070/api_jsonrpc.php
export ZABBIX_USER=Admin
export ZABBIX_PASS=zabbix

tofu init
tofu apply -auto-approve
tofu plan -detailed-exitcode   # must report no changes (exit 0)
tofu destroy -auto-approve
```

The `tofu plan -detailed-exitcode` step is the important one: it fails with
exit code 2 if any resource is not idempotent.
