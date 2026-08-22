# terraform-provider-zabbix

A [Terraform](https://terraform.io) provider for [Zabbix](https://www.zabbix.com).

Manage hosts, host groups, templates, template groups, proxies, items, item
prototypes, low-level discovery rules, triggers and graphs as code, against the
Zabbix JSON-RPC API.

> **Upgrading from `v0.17.0`? Read [MIGRATING.md](./MIGRATING.md) first.**
> `v2.0.0` is deliberately a breaking release: the minimum Zabbix version moved
> to 6.0, three resources and a data source were removed, and four collections
> became sets. Every change is either an edit to your `.tf` files or a
> `terraform state` operation — nothing needs your Zabbix objects to be
> recreated.

## Documentation

The **reference documentation lives in [`docs/`](./docs)** and is published to
the Terraform Registry. It is generated from the provider schema by
[`tfplugindocs`](https://github.com/hashicorp/terraform-plugin-docs) — see
[DEVELOPMENT.md](./DEVELOPMENT.md#documentation) before editing it by hand.

| | |
|---|---|
| [`docs/index.md`](./docs/index.md) | provider configuration |
| [`docs/resources/`](./docs/resources) | one page per resource |
| [`docs/data-sources/`](./docs/data-sources) | one page per data source |
| [CHANGELOG.md](./CHANGELOG.md) | what changed in each release |
| [MIGRATING.md](./MIGRATING.md) | `v0.17.0` → `v2.0.0` upgrade guide |
| [CONTRIBUTING.md](./CONTRIBUTING.md) | the patterns to follow, and the traps |
| [DEVELOPMENT.md](./DEVELOPMENT.md) | building, testing, generating the docs |
| [TESTING.md](./TESTING.md) | the live multi-version acceptance harness |
| [MAINTAINING.md](./MAINTAINING.md) · [RELEASING.md](./RELEASING.md) | new-Zabbix-version runbook, and cutting a release |
| [PLAN.md](./PLAN.md) · [API-COVERAGE.md](./API-COVERAGE.md) | roadmap and the API gap checklist |

## Zabbix version support

| Tier | Versions | Commitment |
|---|---|---|
| **Supported** — CI-gated, release-blocking | 6.0 LTS, 7.0 LTS, 7.4 | the full acceptance suite must pass before a release is cut |
| **Early warning** — non-blocking | 8.0, via the `ubuntu-trunk` nightly | in the test matrix, reported but never gating; promoted to release-blocking on GA |
| **Dropped** | 4.0, 5.0, 5.4 | the code paths are deleted, not merely untested |

The floor tracks Zabbix's own limited-support window: **6.0 leaves limited
support on 2027-02-28, at which point the floor moves to 7.0** and the 6.0 stack
is dropped from the matrix.

The provider detects the server version at configure time (`apiinfo.version`,
unauthenticated) and adapts: bearer-token auth on 6.4+, `selectHostGroups` /
`selectTemplateGroups` on 7.2+, the rewritten proxy model on 7.0+, and so on.
You do not configure the version — but a resource or attribute that does not
exist on your server is refused rather than silently dropped, because from
Zabbix 7.0 an unknown property is a hard API error.

### Minimum Zabbix version per resource

Everything is available on 6.0 unless listed here.

| Resource / attribute | Minimum |
|---|---|
| `zabbix_templategroup`, `data.zabbix_templategroup` | **6.2** |
| `zabbix_template.groups` means *template* group ids | **6.2** (host group ids on 6.0/6.1 — see [MIGRATING.md §5](./MIGRATING.md#5-zabbix_templategroups-now-means-template-groups)) |
| `zabbix_template.vendor_name`, `.vendor_version` | **6.4** |
| `zabbix_template.readme`, `.wizard_ready` | **7.4** |

`zabbix_proxy` and `zabbix_host`'s proxy assignment are modelled on the 7.0
object (`proxyid` + `monitored_by`) and translated back to the pre-7.0
(`proxy_hostid`) shape automatically, so they work unchanged from 6.0.

## Installation

```hcl
terraform {
  required_providers {
    zabbix = {
      source  = "tpretz/zabbix"
      version = "~> 2.0"
    }
  }
}
```

## Provider configuration

```hcl
provider "zabbix" {
  url = "https://zabbix.example.com/api_jsonrpc.php"

  # either username + password ...
  username = "Admin"
  password = "zabbix"

  # ... or an API token (Zabbix 5.4+)
  # token = "..."
}
```

Every argument has an environment-variable fallback — `ZABBIX_URL`,
`ZABBIX_USER`, `ZABBIX_PASS`, `ZABBIX_TOKEN` — so credentials need not be
written into configuration. See [`docs/index.md`](./docs/index.md) for the full
schema, including `tls_insecure` and `serialize`.

## Importing

Every resource supports `terraform import` using the Zabbix numeric id:

```console
$ terraform import zabbix_host.example 10634
```

Two attributes cannot be imported, because the Zabbix API never returns them:
`zabbix_proxy.tls_psk_identity` / `.tls_psk`, and the same pair on
`zabbix_host`. They are write-only on the server.

## Status

The provider is feature-complete for the objects listed in
[`docs/resources/`](./docs/resources) and is tested against live 6.0, 7.0, 7.4
and 8.0 servers on every change. It does **not** yet cover users, user groups,
roles, actions, media types, maintenance windows, value maps, host prototypes,
web scenarios, services, SLAs, network discovery, or dashboards — that backlog
is enumerated in [PLAN.md § Phase 4](./PLAN.md#phase-4--feature-completeness)
and [API-COVERAGE.md](./API-COVERAGE.md).

## Contributing

Start with [CONTRIBUTING.md](./CONTRIBUTING.md) — the item/prototype/LLD triad,
version gating, the collection rules and the traps that have actually bitten —
and [DEVELOPMENT.md](./DEVELOPMENT.md) for build and documentation mechanics. In
short: `make build`, `make test` for unit tests, and
`make testenv-up && make testacc` for the live acceptance suite across every
supported Zabbix version.

## License

[MIT](./LICENSE).
