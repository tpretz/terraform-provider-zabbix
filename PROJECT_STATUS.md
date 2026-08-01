# Project Status — terraform-provider-zabbix

Fork of [tpretz/terraform-provider-zabbix](https://github.com/tpretz/terraform-provider-zabbix)
extended to support Zabbix 7.0 LTS, API token authentication (MFA-safe), and
significantly wider resource coverage.

**Last verified:** 2026-07-31 against Zabbix 7.0.26 / OpenTofu 1.12.3

---

## Verification summary

| Check | Result |
|-------|--------|
| `go build ./...` | ✅ clean |
| `go vet ./...` | ✅ clean |
| `gofmt` | ✅ clean |
| Unit tests | ✅ 6/6 pass |
| Acceptance tests | ✅ 41/41 pass (against live Zabbix 7.0.26) |
| E2E — base (11 resources) | ✅ apply → plan empty → destroy clean |
| E2E — extended (19 resources) | ✅ apply → plan empty → destroy clean |
| E2E — MFA path (token only) | ✅ plan + apply with `ZABBIX_API_TOKEN` only, no login |
| Import (56 resources) | ✅ all pass with empty plan after import |
| `goreleaser check` | ✅ passes |
| `goreleaser build --snapshot` | ✅ produces all platform artifacts |

---

## Resources — 56 total

### From upstream tpretz (fixed for Zabbix 7.0)

| Resource | Notes |
|----------|-------|
| `zabbix_host` | Added tags, inventory, proxyid (7.0), template linkage |
| `zabbix_hostgroup` | — |
| `zabbix_template` | Fixed name idempotency; groups now require template group IDs on 7.0 |
| `zabbix_application` | Deprecated since Zabbix 5.4, kept for compatibility |
| `zabbix_trigger` / `zabbix_proto_trigger` | — |
| `zabbix_graph` / `zabbix_proto_graph` | Fixed: update was completely broken (missing graphid) |
| `zabbix_item_agent/snmp/http/trapper/simple/external/internal/dependent/calculated/aggregate/snmptrap` | Fixed: delta, hosts, hostid-on-update, error_handler |
| `zabbix_proto_item_*` (same types) | Same fixes |
| `zabbix_lld_agent/snmp/http/trapper/simple/external/internal/dependent` | Fixed: delay for trapper/dependent (must be 0), formulaid gating, hostid-on-update |

### New in this fork

| Resource | Domain |
|----------|--------|
| `zabbix_templategroup` + `data.zabbix_templategroup` | Required for templates on Zabbix ≥ 6.2 |
| `zabbix_template_link` | Declarative template item management |
| `zabbix_token` | API token lifecycle; **import refused by design** |
| `zabbix_user` | Full user management |
| `zabbix_usergroup` + `data.zabbix_usergroup` | With per-group permissions |
| `zabbix_role` + `data.zabbix_role` | UI/API/action access control |
| `zabbix_mediatype` | email, script, webhook |
| `zabbix_script` | Custom scripts and webhooks |
| `zabbix_action` | trigger/discovery/autoregistration/internal/service eventsources |
| `zabbix_proxy` (resource) | Zabbix 7.0 shape (`operating_mode`, not `status`) |
| `zabbix_proxygroup` | New in Zabbix 7.0 |
| `zabbix_maintenance` | one_time / daily / weekly / monthly periods |
| `zabbix_valuemap` + `data.zabbix_valuemap` | Per-host/template value maps |
| `zabbix_global_macro` | Global macros (different API methods from host-level) |
| `zabbix_regexp` | Regular expression library |
| `zabbix_service` | Business service tree |
| `zabbix_sla` | SLA definitions |
| `zabbix_httptest` | Web scenarios |

### Data sources — 11 total

`zabbix_server`, `zabbix_host`, `zabbix_hostgroup`, `zabbix_templategroup`,
`zabbix_template`, `zabbix_proxy`, `zabbix_usergroup`, `zabbix_role`,
`zabbix_valuemap`, `zabbix_token` (metadata only — no secret),
`zabbix_application`

---

## Authentication

### API token (MFA-safe)

```hcl
provider "zabbix" {
  url       = "http://zabbix.example.com/api_jsonrpc.php"
  api_token = var.zabbix_api_token   # or env ZABBIX_API_TOKEN
}
```

Credentials are sent in `Authorization: Bearer` header. No `user.login` call
is made. Validated on provider configure — bad token fails early.

### Username + password

```hcl
provider "zabbix" {
  url      = "http://zabbix.example.com/api_jsonrpc.php"
  username = var.zabbix_user     # or env ZABBIX_USER
  password = var.zabbix_password # or env ZABBIX_PASS
}
```

---

## Using locally

### Requirements

- Go 1.22+ (for building)
- OpenTofu ≥ 1.0 or Terraform ≥ 1.0

### Install

```bash
git clone https://github.com/YOUR_FORK/terraform-provider-zabbix
cd terraform-provider-zabbix
make install
```

Installs to `~/.terraform.d/plugins/registry.{terraform,opentofu}.io/local/zabbix/1.0.0/`.
No CLI configuration needed.

### Reference in config

```hcl
terraform {
  required_providers {
    zabbix = {
      source  = "local/zabbix"
      version = "1.0.0"
    }
  }
}
```

### Quick example

```bash
cd examples/quickstart
export ZABBIX_URL=http://your-zabbix/api_jsonrpc.php
export ZABBIX_USER=Admin ZABBIX_PASS=zabbix
tofu init && tofu apply
```

Creates: host group, template group, template + items + trigger + LLD,
monitored host, webhook media type, action, maintenance window, user, token.

---

## Release pipeline

`.github/workflows/release.yml` triggers on tag `v*`:

1. `actions/setup-go` reads version from `go.mod` (≥ 1.22)
2. Import GPG key from `GPG_PRIVATE_KEY` + `PASSPHRASE` secrets
3. goreleaser v2 builds all platforms, signs checksums, attaches manifest
4. GitHub Release created automatically

Both registries read that release:
- **Terraform Registry** — claim namespace at `registry.terraform.io` with your GitHub account
- **OpenTofu Registry** — submit issue at `github.com/opentofu/registry`

Required secrets in the GitHub repo:
| Secret | Value |
|--------|-------|
| `GPG_PRIVATE_KEY` | `gpg --armor --export-secret-keys FINGERPRINT` |
| `PASSPHRASE` | GPG key passphrase |

---

## Known limitations

| Limitation | Detail |
|------------|--------|
| `zabbix_token` import refused | By design — Zabbix returns secret once; silent import would lose it. Use `data.zabbix_token` to reference metadata |
| `zabbix_user` import | `passwd` empty after import; first apply sets it |
| Destroy with own token | Config managing its own token/user cannot self-destroy. Use separate credentials |
| Tested Zabbix versions | 7.0.26 only. Compatibility code for 5.x/6.x retained but unverified |

---

## Not implemented (exists in Zabbix 7.0 API)

`zabbix_dashboard`, `zabbix_templatedashboard`, `zabbix_connector`,
`zabbix_correlation`, `zabbix_iconmap`, `zabbix_map`, `zabbix_host_prototype`,
`zabbix_drule` (network discovery), `zabbix_userdirectory`, `zabbix_mfa`,
`zabbix_report`

---

## File structure (changes vs upstream)

```
.github/workflows/release.yml   updated: Go from go.mod, goreleaser v2
.goreleaser.yml                  rewritten for v2 format + registry manifest
terraform-registry-manifest.json new: required by Terraform/OpenTofu registry
Makefile                         new: build/install/test targets
CHANGELOG.md                     new: full change log
CONTRIBUTING_RESOURCES.md        new: conventions for adding resources
docker-compose.zabbix70.yml      new: Zabbix 7.0 for acceptance tests

go-zabbix-api/                   vendored, local replace
  base.go          token auth (Bearer header), apiinfo.version guard
  token.go         new: Token CRUD
  maintenance.go   new: Maintenance CRUD
  template_group.go new: TemplateGroup CRUD
  item.go          fixed: delta, hosts, hostid omitempty, discoveryRule typo
  lld.go           fixed: hostid omitempty
  action.go        fixed: eventsource omitempty
  script.go        fixed: scope omitempty
  + mediatype.go, user.go, usergroup.go, role.go, managed_proxy.go,
    proxygroup.go, valuemap.go, regexp.go, service.go, sla.go, httptest.go,
    global_macro.go (all new)

provider/
  provider.go      api_token field, optional username/password, token auth path
  resource_token.go        new + import refused
  resource_user.go         new
  resource_usergroup.go    new
  resource_role.go         new
  resource_mediatype.go    new
  resource_script.go       new
  resource_action.go       new
  resource_managed_proxy.go new
  resource_proxygroup.go   new
  resource_maintenance.go  new
  resource_valuemap.go     new
  resource_global_macro.go new
  resource_regexp.go       new
  resource_service.go      new
  resource_sla.go          new
  resource_httptest.go     new
  resource_templategroup.go new
  resource_template_link.go new (ported from nzolot)
  data_server.go           new
  common_item.go           fixed: hostid/ruleid cleared on update, error_handler default
  common_lld.go            fixed: delay per type, hostid cleared on update,
                                  formulaid gated on custom evaltype
  resource_graph.go        fixed: GraphID on update, ymin/ymax_itemid Computed
  resource_template.go     fixed: name Computed
  resource_host.go         fixed: proxyid for 7.0, tags
  resource_*_common.go     fixed: LLD delay per type (agent/http/snmp/etc)

examples/
  provider/provider.tf     updated to current format
  quickstart/main.tf       new: complete working example
  e2e/main.tf              new: base E2E (11 resources)
  e2e-extended/main.tf     new: extended E2E (19 resources)
```
