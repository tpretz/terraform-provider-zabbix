# terraform-provider-zabbix — revival & maintenance plan

Goal: feature-complete, documented, fully tested provider supporting Zabbix 6.0 LTS,
7.0 LTS and 7.4, on a footing that makes routine maintenance cheap.

Companion document: [API-COVERAGE.md](./API-COVERAGE.md) — the gap checklist.

## Decisions

| Decision | Choice |
|---|---|
| Minimum supported Zabbix | **6.0 LTS** — 4.0, 5.0 and 5.4 dropped |
| API client | **Merged into this repo** as `internal/zabbix`; submodule retired |
| Breaking changes | **Batched into one major release** (v2.0.0) with a migration guide |
| Where the work happens | A new **`v2`** branch — no legacy branch or release is touched |

### Branch and release strategy

All work in this plan happens on a new **`v2`** branch, cut from `testenv`.

- `master` (last released `v0.17.0`) and `testenv` are **frozen**. No commits, no
  backports, no re-tagging. Existing `v0.x` releases stay published and unchanged.
- Nothing here ships as a `v0.x` patch, including the 7.2 auth fix. Users on current
  Zabbix wait for v2.0.0 rather than getting a partial fix on the old line.
- The first release off this branch is **`v2.0.0`**, cut only when phases 0–3 are
  complete and the 6.0/7.0/7.4 matrix is green. No pre-1.0 or interim tags in between —
  if something needs publishing before then, use a prerelease tag (`v2.0.0-rc1`).
- v1.x is deliberately skipped so the branch name, the major version, and the Terraform
  Registry version all read the same.
- Once `v2.0.0` is out, `v2` becomes the default branch and `master` is left as an
  archive of the old line.

Two mechanical notes:

- **No Go module path change.** Go's `/vN` suffix rule applies to modules consumed as
  libraries; a provider is a binary and is not imported. Upstream providers
  (`terraform-provider-aws` is at v6) keep an unsuffixed path — do the same.
- **CI and release workflows must be branch-scoped** so a push to `v2` cannot publish
  over the `v0.x` line. Tag-triggered release stays `v*`, but gate the job on the tag
  being `v2.*` and the commit being an ancestor of `v2`.

### Version support policy

| Tier | Versions | Commitment |
|---|---|---|
| Supported (CI-gated, release-blocking) | 6.0 LTS, 7.0 LTS, 7.4 | full acceptance suite must pass |
| Early-warning (non-blocking) | 8.0 LTS via `ubuntu-trunk` | in the matrix from Phase 1; promoted to release-blocking on GA (Q3 2026) |
| Dropped | 4.0, 5.0, 5.4 | code paths deleted, not just untested |

The floor tracks Zabbix's own limited-support window. 6.0 leaves limited support
2027-02-28, at which point the floor moves to 7.0.

### Consequences of the 6.0 floor

Dropping below-6.0 support is a deletion exercise, and a large one — do it early, in
Phase 2b, so every later phase works against a smaller codebase:

- **Legacy SNMP item model** — `SNMPv1Agent`/`SNMPv2Agent`/`SNMPv3Agent` (types 1/4/6)
  collapsed into `SNMPAgent` (20) at 5.0. Delete the pre-5.0 branches in
  `resource_snmp_common.go:182,208,235,259` and `resource_host.go:472,728`.
- **Applications** — removed at 5.4. Delete `zabbix_application` resource + data source,
  `internal/zabbix/application.go`, `selectApplications` in `common_item.go:275`, the
  `applications` field on every item schema, and `ItemsGetByApplicationId`.
- **Aggregate items** — removed at 6.0. Delete `zabbix_item_aggregate`,
  `zabbix_proto_item_aggregate` and `resource_aggregate_common.go`.
- **v4 inventory/tag compat** — the v4-legacy handling in `host.go` that `CLAUDE.md`
  currently tells you to preserve. Delete it, and update `CLAUDE.md` to match.
- **Version gates** — every `Config.Version < 60000` comparison becomes dead. Roughly
  30 sites across provider and tests.

Gates that **remain** (6.0 is below all of these):
`>= 60200` template groups · `>= 60400` bearer auth, template vendor fields ·
`>= 70000` proxy model, `monitored_by`, LLD header arrays, browser items ·
`>= 70200` `selectHostGroups`/`selectTemplateGroups` · `>= 70400` LLD rule prototypes.

Introduce named constants (`zabbix.V62`, `V64`, `V70`, `V72`, `V74`) rather than bare
integers so gates stay greppable.

### SDK

Stay on `terraform-plugin-sdk/v2` (bump 2.10.1 → 2.40.x). Migrating to
`terraform-plugin-framework` is a full rewrite of ~50 resources and buys little for a
CRUD-over-JSON-RPC provider. Revisit only if protocol v6 features are needed.

---

## The unit of work

Phases 0–3 are enumerated task by task below. Phase 4 is not, and deliberately so: it is
35 repetitions of one well-defined shape. Rather than write out 250 checkboxes, the
shape is defined once here, and Phase 4 lists the instances.

**Definition of done for a new resource** (8 steps, `S1`–`S8`):

| | Step | Detail |
|---|---|---|
| S1 | Client struct + CRUD | `internal/zabbix/<obj>.go` — struct with Zabbix JSON tags, `<Obj>sGet/Create/Update/Delete` wrapping `CallWithErrorParse` |
| S2 | Version gates | named constants; old path retained for every supported version where the object or field differs |
| S3 | Resource schema | `provider/resource_<obj>.go` — schema with a `Description` on every field, `ValidateFunc`s, lookup tables via the `_LOOKUP`/`_REV`/`_ARR` idiom |
| S4 | CRUD funcs + registration | Create/Read/Update/Delete + `ImportStatePassthrough`; register in `provider.go` |
| S5 | Data source | where a lookup-by-name is useful (not all objects need one) |
| S6 | Acceptance test | create → update → re-read, plus `ImportState`/`ImportStateVerify`, plus a `SkipFunc` for version-bound behaviour |
| S7 | Sweeper | `resource.AddTestSweepers` entry so aborted runs self-clean |
| S8 | Docs + example | schema descriptions drive `tfplugindocs`; hand-written intro in `templates/`, runnable HCL in `examples/` |

A resource that is version-bound (proxy, connector, browser items) costs roughly 1.5×
because S2 and S6 double up. An item backend type costs less than a standalone object —
S1/S3 are mostly free, since `common_item.go` supplies the machinery and the work is one
`resource_<type>_common.go` plus three registrations.

**Estimated task volume:**

| Phase | Enumerated tasks | Real sub-tasks (est.) |
|---|---|---|
| 0 — toolchain | 9 | ~15 |
| 1 — test infrastructure | 10 | ~30 |
| 2 — 7.x correctness + purge | 8 | ~25 |
| 3 — fix + test existing | 15 | ~60 (mostly test files) |
| **v2.0.0 — release gate** | — | — |
| 4 — feature completeness | 35 instances (+2 deferred) | ~250 (35 × S1–S8) |
| 5 — documentation | 7 | ~55 (per-resource descriptions) |
| 6 — maintenance posture | 5 | ~10 |

Critical path to v2.0.0 is phases 0–3: **42 enumerated tasks, ~130 sub-tasks.**
Everything after is incremental `v2.x`.

Phase 4 is the long tail by construction and is not meant to complete before v2.0.0.

---

## Phase 0 — Toolchain, repo structure, CI baseline

Nothing else is safe to do until the build and CI are modern.

- [x] **Cut the `v2` branch** from `testenv` and do everything below on it. `master` and
      `testenv` are frozen from this point.
- [x] **Merge `go-zabbix-api` into `internal/zabbix`.** Copy the submodule contents in
      with history (`git subtree add` or a filtered merge), rewrite the import path
      `github.com/tpretz/go-zabbix-api` → `github.com/tpretz/terraform-provider-zabbix/internal/zabbix`,
      delete `.gitmodules` and the `replace` directive, and remove the
      `sed -i '/^replace/d' go.mod` step from `release.yml`. Archive the upstream repo
      with a pointer to this one.
- [x] `go.mod`: `go 1.11` → `go 1.23.0`; dropped the `github.com/hashicorp/terraform v0.12.23`
      dependency; `go mod tidy` pruned the indirect block (net −580 lines of `go.sum`)
- [x] Bump `terraform-plugin-sdk/v2` 2.10.1 → **v2.37.0** (not 2.40.1 — see below)
- [ ] **Toolchain decision: raise Go to 1.25.x and finish the SDK bump to 2.40.1.**
      The SDK's own `go` directive gates this and 2.40.1 is not reachable from Go 1.23:

      | SDK | requires |
      |---|---|
      | v2.36.1 | go 1.22.0 |
      | **v2.37.0** | **go 1.23.0** ← current ceiling |
      | v2.38.0–v2.39.0 | go 1.24.0 |
      | v2.40.0 | go 1.25.0 |
      | v2.40.1 | go 1.25.8 |

      The installed toolchain is go1.23.4, so v2.40.1 needs a two-minor jump, not a near
      miss. Deferred rather than forcing a `GOTOOLCHAIN` download mid-task. Do this when
      pinning the CI Go version — that is where the toolchain floor gets set anyway — and
      the SDK bump is then a one-line `go mod edit`. A 2026 release should not ship on a
      Feb-2025 SDK.
      Note the `toolchain go1.23.4` line `go mod tidy` auto-inserts was stripped
      deliberately: it would impose a 1.23.4 floor on anyone running an older 1.23.x.
- [x] `.github/workflows/ci.yml`: `go build ./...`, `go vet ./...`, `go test ./provider/`,
      `gofmt -l`, on push + PR
- [x] `release.yml`: `actions/checkout@v4`, `setup-go@v5` (version from `go.mod`),
      `goreleaser-action@v6`, `--clean` instead of the removed `--rm-dist`
- [x] `.goreleaser.yml`: drop `386`/`arm` where unused, `changelog.skip` → `changelog.disable`
- [x] Add `golangci-lint` with a conservative ruleset
- [x] Update `CLAUDE.md`: no submodule, new version floor, no v4 compat carve-out

**Exit criteria:** green CI on push; `goreleaser build --snapshot` succeeds locally;
`git submodule status` returns nothing.

---

## Phase 1 — Test infrastructure  ✅ DONE

**This comes first deliberately.** Every later phase is a change to API-facing behaviour,
and none of it is verifiable without live servers for each supported version. Standing
the harness up before touching provider code means every subsequent commit — the bearer
auth fix, the legacy purge, each new resource — is validated against all four targets as
it lands, rather than in a big-bang test run at the end.

The harness has no dependency on the provider code, so nothing blocks it.

The current one only works inside the devcontainer and targets four EOL versions.

- [ ] **Standalone `docker-compose.test.yml` at repo root**, runnable with plain
      `docker compose` — not coupled to `.devcontainer`. One stack per version:
      6.0, 7.0, 7.4, **8.0**.
- [ ] **Zabbix 8.0 stack from day one.** 8.0 LTS is due Q3 2026 and no
      `zabbix/zabbix-server-pgsql:*8.0*` tag is published yet, but `ubuntu-trunk` is
      built nightly (last push 2026-07-30) and is the 8.0 pre-release line. Add the stack
      now pointing at `ubuntu-trunk`, mark it **non-blocking** in CI, and flip it to a
      pinned `ubuntu-8.0-*` tag on GA. This surfaces 8.0 API breakage months before it
      ships instead of after.
      Note trunk is a moving target — treat 8.0 failures as signal to investigate, never
      as a release gate, until the tag is pinned.
- [ ] Switch to **PostgreSQL** (`zabbix-server-pgsql` / `zabbix-web-nginx-pgsql`) with one
      DB container per version. The current single shared MySQL 8.0 with
      `mysql_native_password` is fragile and exists mainly for the 4.0/5.0 images being
      dropped. Pin image tags to explicit patch versions for reproducibility.
- [ ] Per-version healthchecks that wait on `api_jsonrpc.php` answering `apiinfo.version`,
      not just the web root — the frontend responds before the DB schema is loaded, which
      is the classic flaky-start cause.
- [ ] **Makefile rework:**
      - `make testenv-up` / `testenv-down` / `testenv-logs` (+ `testenv-up-60` etc. to
        bring up a single stack — four full Zabbix stacks is a lot of laptop RAM)
      - `make test60 test70 test74 test80`; `make testacc` runs the three gated versions,
        `make testall` adds 8.0
      - `make test-one TEST=TestAccResourceHost VER=74` for single-test iteration, which
        is what day-to-day development will actually use
      - `ZABBIX_URL` resolved from a per-version localhost port so the suite runs outside
        the container too
      - per-version logs (`provider/acc-<ver>.log`)
- [ ] **Version-add must be a one-liner.** Parameterise the compose stacks (YAML anchors
      or a small template) so adding 8.2 later is a config entry, not a copy-pasted 40-line
      block. The current file is four hand-duplicated stacks; that is why it rotted.
- [ ] **CI acceptance workflow:** GH Actions matrix over 6.0/7.0/7.4 as release-gating,
      8.0 with `continue-on-error: true`. Compose brought up as a step. Nightly schedule
      plus manual dispatch — acceptance runs are too slow for every PR.
- [ ] **`CheckDestroy` on every resource test.** There is currently **zero** `CheckDestroy`
      in the suite. This is a correctness gap rather than hygiene: a test can pass while
      leaving its objects on the server, so a broken `Delete` path goes undetected. Every
      resource has a delete path and none is verified.
- [ ] **Sweepers** (`resource.AddTestSweepers`) as a *recovery* path, for runs killed
      mid-flight where `CheckDestroy` never runs. Keyed off the consistent `test-*` prefix.
- [ ] ~~Unique naming helper~~ — **dropped after review.** Each version has its own
      dedicated Postgres container and tests run sequentially (no `t.Parallel`), so
      cross-version and intra-run collisions are impossible. Consistent `test-*` names are
      better anyway, because sweepers need a predictable prefix. The random-suffix
      convention comes from cloud providers sharing one account across many CI jobs — that
      topology does not exist here.
- [ ] Devcontainer updated to reference the new compose file rather than duplicating it.

**Exit criteria:** `make testenv-up` brings up all four versions from a clean checkout on
a laptop and in CI, each answering `apiinfo.version` with the expected number, and
`make testacc` *runs* to completion against each.

The suite will **fail** at this point — that is expected and is the point. 7.x fails on
bearer auth, 6.0 fails on the pre-6.0 fixtures. Capture that per-version failure list as
the baseline; it becomes the concrete worklist for Phase 2 and the measure of progress
through it.

---

## Phase 2 — Correctness on 7.x, and the legacy purge  ✅ DONE

Two halves, done together because they touch the same files: fix what's broken above
6.0, delete what only existed below it.

### 2a — Unbreak current Zabbix

The provider cannot talk to a 7.2+ server at all today.

- [ ] **Bearer auth.** In `internal/zabbix/base.go`, send `Authorization: Bearer <token>`
      when `Config.Version >= 60400`; keep the `auth` body property below that (6.0/6.2
      still need it). Drop `auth` from the request struct on the new path entirely.
- [ ] **`apiinfo.version` must stay unauthenticated** — verify the probe in `NewAPI`
      runs before any credential is attached.
- [ ] **`selectGroups` removal.** Replace with `selectHostGroups` (host) /
      `selectTemplateGroups` (template) on `>= 70200`; keep `selectGroups` below.
      Sites: `resource_host.go:596,625`, `resource_template.go:126,153`.
- [ ] **Host ↔ proxy.** `proxy_hostid` → `proxyid` on `>= 70000`, and set `monitored_by`
      (0 server / 1 proxy / 2 proxy group) accordingly. `internal/zabbix/host.go:63`.
- [ ] **LLD HTTP headers/query fields** → array-of-`{name,value}` on `>= 70000`.
- [ ] **Preprocessing** `params` mandatory for "check for not supported value" on `>= 70000`.
- [x] **`ruleid` on `itemprototype.update`** — a fourth 7.2 hard-error parameter,
      alongside `auth`, `selectGroups` and `proxy_hostid`. Missed in the original audit
      because it is an object property on the update path rather than a request
      parameter; found later by the first prototype acceptance test. Fixed in
      `prepItemsUpdate`.
- [ ] **Strict-validation audit.** Grep every `zabbix.Params{...}` literal in provider and
      client and check each key against the 7.4 reference. 7.2 made unknown parameters a
      hard error rather than ignoring them, so this needs to be exhaustive, not spot-checked.
- [ ] Add `Script` (21) and `Browser` (22) to the `ItemType` enum.

### 2b — Delete pre-6.0 support  ✅ DONE

Per "Consequences of the 6.0 floor" above: legacy SNMP model, applications, aggregate
items, v4 inventory/tag compat, and all `< 60000` gates. Remove the corresponding
version-conditional branches from the existing tests at the same time
(`resource_host_test.go`, `resource_item_snmp_test.go`, `resource_item_agent_test.go`
and friends all carry `< 50000` / `>= 50400` `SkipFunc`s).

**Exit criteria:** the Phase 1 baseline failure list is empty for 6.0, 7.0 and 7.4 — the
existing acceptance suite passes end-to-end on all three. No `Config.Version` comparison
below `60200` remains in the tree. 8.0/trunk failures are triaged and recorded, not
necessarily fixed.

---

## Phase 3 — Correct and test what already exists

Close the gaps in current resources before adding new ones.

### 3a — resource corrections

- [x] `zabbix_templategroup` resource + data source; teach `zabbix_template` to use
      template groups on `>= 60200` and host groups below. **Breaking** — migration guide.
- [x] `zabbix_proxy` **resource** (currently data source only), modelled on 7.0
      (`name`, `operating_mode`, `address`, `port`, `allowed_addresses`, TLS fields) with
      pre-7.0 translation; update the data source likewise.
- [x] `zabbix_host`: `templates_clear`, proxy assignment (`monitored_by`/`proxyid`), full inventory fields,
      IPMI settings, TLS connect/accept + PSK, `templates_clear` on removal.
- [x] `zabbix_template`: `uuid`, `vendor_name`/`vendor_version` (6.4+), `readme` and
      `wizard_ready` (7.4+).
- [x] `zabbix_trigger`: audit against the 7.4 object — `event_name`, `manual_close`,
      `correlation_mode`/`correlation_tag`, `opdata`, dependencies.
### 3b — state migration for the breaking changes

Removing attributes from a schema does not remove them from users' state files. Without
upgraders, a `v0.17.0` state hits "Invalid address to set" / unknown-attribute errors on
first plan under v2.

- [ ] Bump `SchemaVersion` on every item resource; add a `StateUpgraders` entry that
      strips `applications` from prior state.
- [x] `SchemaVersion` + upgrader on `zabbix_template` for the host-group →
      template-group `groups` transition on 6.2+.
- [ ] For the removed resources (`zabbix_application`, `zabbix_item_aggregate`,
      `zabbix_proto_item_aggregate`) there is no upgrade path — document
      `terraform state rm` in `MIGRATING.md`.
- [ ] Test the upgraders with a checked-in `v0.17.0` state fixture.

### 3c — fill the test gaps

Grouped by what's actually missing (see [API-COVERAGE.md §4](./API-COVERAGE.md)):

- [x] **Nine empty stubs** — all filled (`resource_trigger_test.go`,
      `resource_proxy_test.go`, and the seven `resource_lld_*_test.go`), each with
      `CheckDestroy` and an `ImportState`/`ImportStateVerify` step. Writing them
      surfaced two real bugs: `zabbix_lld_dependent` could never be created
      (delay defaulted to 3600, Zabbix requires 0), and PSK attributes cannot
      round-trip on import because `proxy.get` never returns them.
- [x] **No test file at all** — `item_http` / `proto_item_http` / `lld_http`,
      `proto_trigger`, `proto_graph` — all covered
- [x] **Prototype coverage** — all ten `zabbix_proto_item_*` resources covered, each
      asserting `ruleid` via `TestCheckResourceAttrPair` so it is proven to survive the
      `selectDiscoveryRule` round trip. This found that **item prototypes could not be
      updated at all on Zabbix 7.2+** — `itemprototype.update` was sent the create-only
      `ruleid`, which 7.2 made a hard error.
- [x] **Data source coverage** — `host`, `hostgroup`, `template`, `proxy` and
      `templategroup` all covered
- [x] **Import tests** — every resource test has `ImportState`/`ImportStateVerify`.
      Only one exclusion exists suite-wide: proxy `tls_psk_identity`/`tls_psk`, which
      `proxy.get` never returns on any version.
- [ ] **Migrate to `terraform-plugin-testing`** — `helper/resource`'s test harness is
      deprecated in SDKv2 in favour of the standalone
      `github.com/hashicorp/terraform-plugin-testing` module. Do this before writing
      ~30 new test files, not after.

**Exit criteria:** every registered resource and data source has an acceptance test
including an import step; state upgraders verified against a v0.17.0 fixture; all three
supported versions green.

---

## v2.0.0 — the breaking release

Cut from the `v2` branch, once phases 0–3 are done and the 6.0/7.0/7.4 matrix is green.
Everything breaking lands in this one release:

- Minimum Zabbix version is now 6.0
- `zabbix_application` and its data source removed → migrate to item tags
- `zabbix_item_aggregate` / `zabbix_proto_item_aggregate` removed → migrate to
  calculated items with aggregate functions
- `zabbix_template.groups` references template groups on 6.2+ → `zabbix_templategroup`
- `applications` attribute removed from every item schema

Ship with a `MIGRATING.md` covering each of the five, with before/after HCL, framed as a
`v0.17.0 → v2.0.0` upgrade.

Post-release, `v2` becomes the default branch and `master` is archived.

---

## Phase 4 — Feature completeness

Ordered by user value; see [API-COVERAGE.md §2](./API-COVERAGE.md) for the full list.
This is the post-2.0 backlog — additive, parallelisable, no longer on the critical path,
and shipped as ordinary `v2.x` minor releases.

Each row below is one work item following S1–S8 from "The unit of work". Columns flag the
steps that carry unusual cost: **DS** = data source also needed, **VG** = version-gated
behaviour, **∼** = relative size.

### 4a — missing item backend types

One `resource_<type>_common.go` plus three registrations each, following
`resource_snmp_common.go` as the fullest example. Cheaper than a standalone object:
S1/S3 are largely supplied by `common_item.go`.

| # | Type | Resources | VG | ∼ |
|---|---|---|---|---|
| 11 | `db_monitor` | item, proto, lld | | M |
| 12 | `ipmi` | item, proto | | S |
| 13 | `ssh` | item, proto, lld | | M |
| 14 | `telnet` | item, proto, lld | | M |
| 16 | `jmx` | item, proto, lld | | M |
| 21 | `script` | item, proto, lld | | M |
| 22 | `browser` | item, proto, lld | 7.0+ | M |

Shared prerequisite: `authtype`/`username`/`password`/`privatekey`/`publickey` schema
fragment for SSH, and a `params`-carrying fragment reused by db_monitor/ssh/telnet/
script/browser.

### 4b — tier 1 objects

| Object | Resource | DS | VG | ∼ |
|---|---|---|---|---|
| user | `zabbix_user` | ✔ | 7.4 strict `user.get` validation | L |
| usergroup | `zabbix_usergroup` | ✔ | 7.2 removed `rights`/`userids`/`selectRights` | L |
| role | `zabbix_role` | | 6.0+ | M |
| action | `zabbix_action` | | 6.0 `update_operations` rename | XL — five action types, nested op/condition/filter blocks |
| mediatype | `zabbix_mediatype` | | 7.2 dropped `content_type`/`exec_params`; 7.4 OAuth fields | L |
| maintenance | `zabbix_maintenance` | | 7.2 removed `groupids`/`hostids` | L — timeperiod blocks |
| valuemap | `zabbix_valuemap` | | host/template scoped since 5.4 | M |
| hostprototype | `zabbix_host_prototype` | | | L — the significant remaining LLD gap |
| httptest | `zabbix_web_scenario` | | | L — step blocks |
| token | `zabbix_token` | | 5.4+ | S |
| proxygroup | `zabbix_proxygroup` | ✔ | 7.0+ only | M |
| usermacro (global) | `zabbix_global_macro` | | | S |

### 4c — tier 2 objects

| Object | Resource | VG | ∼ |
|---|---|---|---|
| script | `zabbix_script` | | M |
| service | `zabbix_service` | 6.0 hierarchy rewrite | L |
| sla | `zabbix_sla` | 6.0+ | M |
| drule + dcheck | `zabbix_discovery_rule` | | L — network discovery, distinct from LLD |
| autoregistration | `zabbix_autoregistration` | | S — singleton |
| correlation | `zabbix_correlation` | | M |
| regexp | `zabbix_regexp` | | S |
| connector | `zabbix_connector` | 7.0+ only | M |
| discoveryruleprototype | `zabbix_lld_rule_prototype` | **7.4+ only** | M |

### 4d — singletons and read-only data sources

| Object | Form | ∼ |
|---|---|---|
| settings | `zabbix_settings` (singleton resource) | M |
| housekeeping | `zabbix_housekeeping` (singleton) | S |
| authentication | `zabbix_authentication` (singleton) | M |
| iconmap | `zabbix_iconmap` | S |
| image | `zabbix_image` | S |
| event / problem | data sources only | M |
| usermacro | data source | S |

### 4e — deliberately out of scope

Recorded so the coverage matrix isn't misread as an infinite backlog:
`alert`, `history`, `trend`, `auditlog`, `task`, `hanode`, `dhost`, `dservice`,
`graphitem`, `module`, `report`, `map`, `userdirectory`, `mfa` — either read-only
telemetry with no config-as-code value, or operational state Terraform should not own.
`configuration.import`/`export` is a possible future `zabbix_template_import` resource
but is a poor fit for Terraform's diff model; revisit only on demand.

### 4f — deferred: dashboards

`zabbix_dashboard` and `zabbix_template_dashboard` are **deferred to a later 2.x minor**,
not dropped.

Rationale: dashboards are the single largest item in the plan and the one whose API
churns most per release — widget field renames landed in 7.0 (`plaintext` → `itemhistory`,
`str.str.index1.index2` → `str.index1.str.index2`, `x` range 0–23 → 0–71), 7.2
(`dataover` removed, `tophosts.style` → `layout`, clock sizing fields dropped) and 7.4
(`itemcard` added). Modelling every widget type means re-doing that work every release,
which is exactly the treadmill the maintenance goal is meant to avoid.

When it is picked up, ship the escape hatch first:

```hcl
widget {
  type   = "itemhistory"
  x      = 0
  y      = 0
  fields = jsonencode({ ... })   # passed through verbatim
}
```

An opaque `fields` blob is version-agnostic and costs nothing to maintain. Strongly-typed
blocks for the handful of widgets people actually manage as code can be layered on later
without a breaking change.

---

## Phase 5 — Documentation

- [ ] Adopt **`terraform-plugin-docs`** (`tfplugindocs generate`) — the 38 files under
      `docs/` are hand-written and will drift. Hand-written prose moves into `templates/`
      and per-resource `examples/`.
- [ ] Every schema field gets a `Description` (tfplugindocs renders these). This is the
      bulk of the work; do it incrementally, one resource at a time.
- [ ] Split the 49 KB `README.md`: short overview + dev/contributing guide; resource
      reference moves to generated docs.
- [ ] Document the version support policy and per-resource minimum Zabbix version.
- [ ] Populate `examples/` — currently one directory. `tfplugindocs` pulls
      `examples/resources/<name>/resource.tf` and `examples/data-sources/<name>/data-source.tf`
      into generated pages, and `import.sh` for the import section. One per resource.
- [ ] Decide the fate of `utils/template2terraform` — the standalone Python XML→HCL
      converter. Either bring it into CI (test it, document it, note the Python 3
      dependency) or move it to a separate repo. Right now it is undocumented and untested
      and will rot.
- [ ] Verify Terraform Registry publication (docs layout, `index.md`, category, logo).

---

## Phase 6 — Routine maintenance posture

- [ ] `CHANGELOG.md` (keep-a-changelog) + `.changie.yaml` or equivalent
- [ ] Renovate/Dependabot for Go modules and GitHub Actions
- [ ] Nightly acceptance run against supported versions; issue opened automatically on failure
- [ ] A **new-Zabbix-release runbook**: add compose stack → run matrix → read the upstream
      `manual/api/changes` page → gate deltas → update support table → drop the version
      that fell out of limited support. For 8.0 the first two steps are already done in
      Phase 1 — the runbook's first real exercise is promoting the trunk stack to a pinned
      `ubuntu-8.0-*` tag and making it release-blocking on GA.
- [ ] `CONTRIBUTING.md` documenting the item triad pattern and the version-gate idiom

---

## Sequencing

```
branch v2 ◀── cut from testenv

Phase 0        Phase 1          Phase 2            Phase 3
toolchain ──▶  test harness ──▶ 7.x correctness ──▶ fix + test ──▶ v2.0.0
               6.0 7.0 7.4 8.0  + legacy purge      existing         │
                    │                                                │
                    └──── validates everything downstream ───────────┤
                                                                     │
                              Phase 4 ──▶ Phase 5 ──▶ Phase 6 ◀──────┘
                              features    docs        maintenance
```

**The harness comes first and everything else is measured against it.** Phase 1 has no
dependency on provider code, so it can be built immediately; from that point every
change in phases 2–6 is validated against all four Zabbix versions as it lands. The
per-version failure list captured at the end of Phase 1 is the worklist for Phase 2 and
the progress measure through it.

Phases 0–3 are one campaign on the critical path — the provider is non-functional on
current Zabbix until Phase 2 ships. Phase 4 onward is the routine-maintenance backlog,
released as incremental `v2.x` minors.

Bringing 8.0 (via `ubuntu-trunk`) into the matrix now rather than on GA means the
Phase 6 new-release runbook gets exercised continuously instead of once, and 8.0 API
breakage surfaces while there is still time to respond to it.
