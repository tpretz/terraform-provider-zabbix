# Zabbix API coverage matrix

Baseline: Zabbix 7.4 API reference (59 documented API objects).
Provider state as of commit `8a9169f` (last touched 2023-05-09).

Target support floor is **Zabbix 6.0 LTS** (see [PLAN.md](./PLAN.md)) — anything marked
💀 below is removed from the API at or before 6.0 and is scheduled for deletion, not
repair.

Legend: ✅ full · 🟡 partial · ❌ none · 💀 dead (API removed upstream)

## 1. Objects the provider covers

| API object | Resource | Data source | State | Notes |
|---|---|---|---|---|
| host | `zabbix_host` | `zabbix_host` | 🟡 | `selectGroups` breaks on ≥7.2; `proxy_hostid`/`monitored_by` not handled for ≥7.0; no `inventory` beyond mode; no IPMI/TLS fields |
| hostgroup | `zabbix_hostgroup` | `zabbix_hostgroup` | ✅ | |
| template | `zabbix_template` | `zabbix_template` | 🟡 | `selectGroups` breaks on ≥7.2; `groups` must reference **template groups** on ≥6.2; no `uuid`/`vendor_name`/`vendor_version` |
| item | 11 of 18 types | ❌ | 🟡 | see §3 |
| item prototype | 11 of 18 types | ❌ | 🟡 | see §3 |
| LLD rule (`discoveryrule`) | 8 types | ❌ | 🟡 | see §3 |
| trigger | `zabbix_trigger` | ❌ | 🟡 | no acceptance test; no `event_name`, `manual_close`, correlation fields audit |
| trigger prototype | `zabbix_proto_trigger` | ❌ | 🟡 | no acceptance test |
| graph | `zabbix_graph` | ❌ | ✅ | |
| graph prototype | `zabbix_proto_graph` | ❌ | 🟡 | no acceptance test |
| proxy | ❌ | `zabbix_proxy` | 🟡 | **data source only**; not updated for the 7.0 proxy redesign |
| host interface | inline in `zabbix_host` | — | 🟡 | not separately addressable; SNMPv3 details partial |
| user macro | inline (host/template) | — | 🟡 | no **global** macro resource |
| application | `zabbix_application` | `zabbix_application` | 💀 | API removed in Zabbix 5.4 |

## 2. Objects with no coverage at all

Grouped by value for a Terraform user.

### Tier 1 — commonly managed as code, high demand
| API object | Proposed resource |
|---|---|
| templategroup | `zabbix_templategroup` (+ data source) — **required** for template management on ≥6.2 |
| proxy | `zabbix_proxy` (resource; 7.0 model) |
| proxygroup | `zabbix_proxygroup` (7.0+) |
| user | `zabbix_user` (+ data source) |
| usergroup | `zabbix_usergroup` (+ data source) |
| role | `zabbix_role` (6.0+) |
| action | `zabbix_action` (trigger/service/discovery/autoregistration/internal) |
| mediatype | `zabbix_mediatype` |
| maintenance | `zabbix_maintenance` |
| valuemap | `zabbix_valuemap` (host/template scoped since 5.4) |
| hostprototype | `zabbix_host_prototype` — notable LLD gap |
| httptest | `zabbix_web_scenario` |
| token | `zabbix_token` (5.4+) |

### Tier 2 — useful, lower frequency
| API object | Proposed resource |
|---|---|
| script | `zabbix_script` |
| service | `zabbix_service` (6.0 model) |
| sla | `zabbix_sla` (6.0+) |
| drule / dcheck | `zabbix_discovery_rule` (network discovery — distinct from LLD) |
| autoregistration | `zabbix_autoregistration` (singleton) |
| correlation | `zabbix_correlation` |
| regexp | `zabbix_regexp` |
| connector | `zabbix_connector` (7.0+) |
| dashboard | `zabbix_dashboard` |
| templatedashboard | `zabbix_template_dashboard` |
| lldruleprototype (`discoveryruleprototype`) | `zabbix_lld_rule_prototype` (**7.4+ only**) |

### Tier 3 — singletons / global config
`settings`, `housekeeping`, `authentication`, `iconmap`, `image`, `module`, `userdirectory` (7.0+), `mfa` (7.0+), `report`.

### Tier 4 — read-only, best exposed as data sources (or skipped)
`event`, `problem`, `alert`, `history`, `trend`, `auditlog`, `task`, `hanode`, `dhost`, `dservice`, `graphitem`, `configuration` (import/export), `map`.

## 3. Item / prototype / LLD type coverage

Zabbix 7.4 item types and what the triad covers:

| # | Type | `zabbix_item_*` | `zabbix_proto_item_*` | `zabbix_lld_*` |
|---|---|---|---|---|
| 0/7 | Zabbix agent (passive/active) | ✅ | ✅ | ✅ |
| 2 | Zabbix trapper | ✅ | ✅ | ✅ |
| 3 | Simple check | ✅ | ✅ | ✅ |
| 5 | Zabbix internal | ✅ | ✅ | ✅ |
| 8 | Zabbix aggregate | 💀 | 💀 | — |
| 9 | Web item | n/a (read-only) | n/a | n/a |
| 10 | External check | ✅ | ✅ | ✅ |
| 11 | Database monitor | ❌ | ❌ | ❌ |
| 12 | IPMI agent | ❌ | ❌ | — |
| 13 | SSH agent | ❌ | ❌ | ❌ |
| 14 | TELNET agent | ❌ | ❌ | ❌ |
| 15 | Calculated | ✅ | ✅ | — |
| 16 | JMX agent | ❌ | ❌ | ❌ |
| 17 | SNMP trap | ✅ | ✅ | — |
| 18 | Dependent item | ✅ | ✅ | ✅ |
| 19 | HTTP agent | ✅ | ✅ | ✅ |
| 20 | SNMP agent | ✅ | ✅ | ✅ |
| 21 | Script | ❌ | ❌ | ❌ |
| 22 | Browser (7.0+) | ❌ | ❌ | ❌ |

Client-side enums in `go-zabbix-api/item.go` stop at `SNMPAgent = 20`; `Script` (21) and `Browser` (22) are absent.

## 4. Acceptance test coverage

15 `TestAcc*` functions exist. Gaps:

**Empty stubs** (file contains only `package provider`):
`resource_trigger_test.go`, `resource_proxy_test.go`, and every `resource_lld_*_test.go`
(agent, dependent, external, internal, simple, snmp, trapper).

**No test file at all:** `zabbix_item_http` / `zabbix_proto_item_http` / `zabbix_lld_http`, `zabbix_proto_trigger`, `zabbix_proto_graph`.

**No prototype coverage:** every `zabbix_proto_item_*` resource is untested.

**No data source coverage:** none of `zabbix_host`, `zabbix_application`, `zabbix_proxy`, `zabbix_hostgroup`, `zabbix_template` has a test.

**No import coverage:** every resource declares `ImportStatePassthrough` but no test uses `ImportState: true`.

**No sweepers:** failed runs leak objects into the test server.

## 5. Version-compatibility breakages (verified against docs)

| Symptom | Cause | Affected versions |
|---|---|---|
| All API calls rejected | `auth` sent as a JSON-RPC body property (`base.go:26`); removed upstream, replaced by `Authorization: Bearer` (available 6.4+) | **≥7.2** |
| `host`/`template` read fails | `"selectGroups"` (`resource_host.go:596,625`; `resource_template.go:126,153`) removed | **≥7.2** |
| `item` read may fail | `"selectApplications"` (`common_item.go:275`) sent unconditionally | ≥5.4 — *verify empirically* |
| Host proxy assignment wrong | `proxy_hostid` (`host.go:63`) renamed `proxyid`; `monitored_by` now required when using a proxy | **≥7.0** |
| Proxy data source wrong | `host`→`name`, `status`→`operating_mode`, `interface`→`address`/`port`, `proxy_address`→`allowed_addresses` | **≥7.0** |
| Template groups | templates live in template groups, not host groups | **≥6.2** |
| `zabbix_application` | applications removed | **≥5.4** |
| `zabbix_item_aggregate` | aggregate item type removed (use calculated + aggregate functions) | **≥6.0** |
| LLD HTTP headers/query fields | name-indexed object → array of `{name,value}` | **≥7.0** |
| Preprocessing "check for not supported value" | `params` now mandatory | **≥7.0** |

## 6. Zabbix support lifecycle (drives the test matrix)

| Version | Released | Full support ends | Limited support ends |
|---|---|---|---|
| 6.0 LTS | 2022-02-08 | 2025-02-28 | 2027-02-28 |
| 7.0 LTS | 2024-06-04 | 2027-06-30 | 2029-06-30 |
| 7.4 | 2025-07-01 | until 8.0 LTS | Q4 2026 |
| 8.0 LTS | Q3 2026 (imminent) | Q3 2029 | Q3 2031 |

4.0, 5.0, 5.4, 6.2, 6.4, 7.2 are all past end of limited support. The current
`Makefile` targets 4.0/5.0/5.4/6.0 — i.e. **every** target except 6.0 is EOL, and
the two versions Zabbix considers current (7.0, 7.4) are untested.

Target matrix: **6.0, 7.0, 7.4** release-gating, plus **8.0** non-blocking via the
nightly `ubuntu-trunk` images from day one (no `*8.0*` tag is published yet as of
2026-08-02), promoted to release-gating on GA. The floor moves to 7.0 when 6.0 leaves
limited support on 2027-02-28.

## 7. Sources

- [API reference index (7.4)](https://www.zabbix.com/documentation/current/en/manual/api/reference)
- [API changes 6.4 → 7.0](https://www.zabbix.com/documentation/7.0/en/manual/api/changes)
- [API changes 7.0 → 7.2](https://www.zabbix.com/documentation/7.2/en/manual/api/changes)
- [API changes 7.2 → 7.4](https://www.zabbix.com/documentation/current/en/manual/api/changes)
- [API changes 5.4 → 6.0](https://www.zabbix.com/documentation/6.0/en/manual/api/changes_5.4_-_6.0)
- [Item object (7.4)](https://www.zabbix.com/documentation/current/en/manual/api/reference/item/object)
- [Authentication (7.4)](https://www.zabbix.com/documentation/7.4/en/manual/api)
- [Life cycle & release policy](https://www.zabbix.com/life_cycle_and_release_policy)
