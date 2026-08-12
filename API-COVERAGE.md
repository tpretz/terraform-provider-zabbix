# Zabbix API coverage matrix

Baseline: Zabbix 7.4 API reference (59 documented API objects).
Provider state as of commit `88f2ff7` (phases 0-2 complete, phase 3 in progress).
Verified against live 6.0.48 / 7.0.29 / 7.4.13 / 8.0-trunk servers.

Target support floor is **Zabbix 6.0 LTS** (see [PLAN.md](./PLAN.md)) — anything marked
💀 below is removed from the API at or before 6.0 and is scheduled for deletion, not
repair.

Legend: ✅ full · 🟡 partial · ❌ none · 💀 dead (API removed upstream)

## 1. Objects the provider covers

| API object | Resource | Data source | State | Notes |
|---|---|---|---|---|
| host | `zabbix_host` | `zabbix_host` | ✅ | 7.x fixed; 70 inventory fields, IPMI, TLS/PSK all covered |
| hostgroup | `zabbix_hostgroup` | `zabbix_hostgroup` | ✅ | |
| template | `zabbix_template` | `zabbix_template` | ✅ | 7.x fixed; template groups with state upgrader; `uuid`, `vendor_*` (6.4+), `readme`/`wizard_ready` (7.4+) |
| item | 10 of 17 types | ❌ | 🟡 | all tested; see §3 for missing backend types |
| item prototype | 10 of 17 types | ❌ | 🟡 | all tested; see §3 |
| LLD rule (`discoveryrule`) | 8 types | ❌ | 🟡 | see §3 |
| trigger | `zabbix_trigger` | ❌ | ✅ | field audit done: `event_name`, `opdata`, `manual_close`, correlation fields, dependencies |
| trigger prototype | `zabbix_proto_trigger` | ❌ | ✅ | |
| graph | `zabbix_graph` | ❌ | ✅ | |
| graph prototype | `zabbix_proto_graph` | ❌ | ✅ | |
| proxy | `zabbix_proxy` | `zabbix_proxy` | ✅ | resource + data source, 7.0 model translated for 6.0–8.0 |
| host interface | inline in `zabbix_host` | — | 🟡 | not separately addressable; SNMPv3 details partial |
| user macro | inline (host/template) | — | 🟡 | no **global** macro resource |
| templategroup | `zabbix_templategroup` | `zabbix_templategroup` | ✅ | 6.2+; below that templates use host groups |

## 2. Objects with no coverage at all

Grouped by value for a Terraform user.

### Tier 1 — commonly managed as code, high demand
| API object | Proposed resource |
|---|---|
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

`Script` (21) and `Browser` (22) now exist in the `internal/zabbix/item.go` enum but have no Terraform resources yet. Aggregate (8) and the legacy SNMP v1/v2/v3 types (1/4/6) were deleted with pre-6.0 support.

## 4. Acceptance test coverage

**Every version passes**: 6.0.48, 7.0.29, 7.4.13, 8.0-trunk.

Landed:
- **All nine empty stubs filled** — `zabbix_trigger`, `zabbix_proxy`, and the seven `zabbix_lld_*`
- **Import coverage** — `ImportState`/`ImportStateVerify` on every test that has one; this found that proxy PSK attributes cannot round-trip (`proxy.get` never returns them, so they need `ImportStateVerifyIgnore`)
- **`CheckDestroy`** on 17+ tests via a shared helper, verified to genuinely fail when a delete is broken
- **Sweepers** for every object type with dependency ordering, plus `TestMain`

**46 acceptance tests pass on every version.** The Phase 3 exit criterion is *almost*
met: every registered resource and data source has an acceptance test, and all but one
include an import step — **`zabbix_graph` has no `ImportState` step** (`zabbix_proto_graph`
does). Tracked in Phase 7. Suite-wide there is exactly one `ImportStateVerifyIgnore`: proxy
`tls_psk_identity`/`tls_psk`, which `proxy.get` never returns on any version.

Still open: the `terraform-plugin-testing` migration, and a `v0.17.0` state fixture for
the template state upgrader.

**One known failure on 8.0-trunk only** (non-blocking by policy): `TestAccResourceGraph`
and `TestAccResourceProtoGraph`. `graph.get` returns `gitems` in a different order than
6.0/7.x, and `item` is a `TypeList` whose order must match config. Sorting by `sortorder`
is not the fix — it breaks 7.4, because config order need not match `sortorder`. Needs
either a `TypeSet` migration or config-order matching on read. Must be resolved before
8.0 becomes release-gating on GA.

Preprocessing step types are `PREPROC_LOOKUP` (items) and `LLD_PREPROC_LOOKUP`
(discovery rules). The two lists genuinely differ: `matches_regex` (14) is accepted on an
item by 6.0 but rejected on a discovery rule until 7.0, so they carry separate gate maps
rather than sharing one.

Bugs found by writing these tests — five so far, all invisible behind missing coverage:
- **Every `zabbix_proto_item_*` was impossible to update on Zabbix 7.2+.** `itemprototype.update` was sent the create-only `ruleid`; 7.2 made unknown parameters a hard error, so any change to any item prototype failed. Item prototypes were effectively write-once on both current Zabbix releases.
- **`post_type` defaulted to `"body"` across the whole http triad** — not a valid value (raw/json/xml); copied from `retrieve_mode` where `"body"` is valid. It mapped to `""`, Zabbix applied `raw`, and the item read back as `raw` against a config saying `body`: a permanent, unappliable diff on every http item.
- **The `zabbix_template` data source panicked the provider on every read** — shared `templateRead` calls `d.Set("templates", ...)`, absent from the data source schema, and SDKv2 `Set` panics on an undeclared key.
- `zabbix_lld_dependent` **could never be created** — `delay` defaulted to 3600, Zabbix requires 0
- `templates_clear` could reference an already-deleted template; 6.0 tolerated it, 7.0+ rejects it

## 5. Version-compatibility breakages — all resolved

Every row below was fixed in phases 2a/2b/3a and is verified against live servers.

| Symptom | Cause | Resolution |
|---|---|---|
| All API calls rejected on ≥7.2 | `auth` sent as a JSON-RPC body property | `Authorization: Bearer` at `>= V64`, body property below |
| `host`/`template` read silently wrong on ≥7.2 | `selectGroups` removed | `selectHostGroups`/`selectTemplateGroups` at `>= V72` |
| `item` read | `selectApplications` | deleted with applications |
| Host proxy assignment wrong on ≥7.0 | `proxy_hostid` → `proxyid` | plus `monitored_by` |
| Proxy data source wrong on ≥7.0 | proxy object rewritten | full resource + data source translating both models |
| Template groups on ≥6.2 | templates left host groups | `zabbix_templategroup` + state upgrader |
| `zabbix_application`, `zabbix_item_aggregate` | removed upstream | deleted |
| LLD HTTP headers/query fields on ≥7.0 | object → array | translated at `>= V70` |
| Preprocessing "not supported" on ≥7.0 | `params` mandatory | handled |

Two findings that contradicted the original assumptions:

- **Strict validation arrived in 7.0, not 7.2, and is per-method.** `item.create`, `itemprototype.create` and `discoveryrule.create` reject unknown properties from 7.0; `host.create` and `graph.create` are lenient even on 8.0. `.get` methods ignore unknown params on *all* versions, so a stale `selectGroups` was a **silent wrong answer**, not an error.
- **`hostid` became create-only in 7.0** — every item update on 7.x would have failed.

## 6. Zabbix support lifecycle (drives the test matrix)

| Version | Released | Full support ends | Limited support ends |
|---|---|---|---|
| 6.0 LTS | 2022-02-08 | 2025-02-28 | 2027-02-28 |
| 7.0 LTS | 2024-06-04 | 2027-06-30 | 2029-06-30 |
| 7.4 | 2025-07-01 | until 8.0 LTS | Q4 2026 |
| 8.0 LTS | Q3 2026 (imminent) | Q3 2029 | Q3 2031 |

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
