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
  over the `v0.x` line. `RELEASING.md` tightens this further than originally planned: the
  trigger itself becomes `tags: ['v2.*']` **and** the job keeps a guard. Belt and braces,
  because the trigger pattern is the half that gets widened by accident.

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

**Definition of done for a new resource** (9 steps, `S1`–`S9`):

| | Step | Detail |
|---|---|---|
| S1 | Client struct + CRUD | `internal/zabbix/<obj>.go` — struct with Zabbix JSON tags, `<Obj>sGet/Create/Update/Delete` wrapping `CallWithErrorParse` |
| S2 | Version gates | named constants; old path retained for every supported version where the object or field differs |
| S3 | Resource schema | `provider/resource_<obj>.go` — schema with a `Description` on every field, `ValidateFunc`s, lookup tables via the `_LOOKUP`/`_REV`/`_ARR` idiom |
| S4 | CRUD funcs + registration | Create/Read/Update/Delete + `ImportStatePassthrough`; register in `provider.go` |
| S5 | Data source | where a lookup-by-name is useful (not all objects need one) |
| S6 | Acceptance test | create → update → re-read, plus `ImportState`/`ImportStateVerify`, plus a `SkipFunc` for version-bound behaviour. Every collection attribute additionally meets `C1`–`C7` below |
| S7 | Sweeper | `resource.AddTestSweepers` entry so aborted runs self-clean |
| S8 | Docs + example | schema descriptions drive `tfplugindocs`; hand-written intro in `templates/`, runnable HCL in `examples/` |
| S9 | Minimum and reverse | applied from its Required set alone, and every optional attribute that was set has been unset again — see `S9` below |
| U1–U4 | Changed in life | every settable attribute changed on an existing resource and asserted against a server re-read; the step asserts it was an *update*, not a replace; every create-only attribute is `ForceNew`, and nothing else is — see `U1`–`U4` below |

**Definition of done for the minimum and the way back** (`S9`).

`S1`–`S8` and `C1`–`C7` between them ask *"does this attribute behave correctly when
set?"*. `E1`–`E6` ask *"what happens when things go wrong?"*. None of them asks the two
questions a user asks on day one and on day thirty:

> **Does the resource work when I set only what is required?**
> **Can what I set be unset?**

Those two have produced **six** defects between them, every one of which survived eight
phases of structured testing, because every fixture in the suite was written by somebody
who already knew the answer: `zabbix_lld_trapper` uncreatable without an explicit
`delay = "0"`; a `preprocessor` block failing on create when `error_handler` was omitted;
`zabbix_host` refusing a host with no interface; HTTP `posts`/`proxy` and trigger `url`
settable but not clearable; HTTP `status_codes`/`timeout` the same; the four SNMPv3
credentials the same, and not even expressible as empty. A resource is not tested until
it has been created from its documented minimum and returned from every optional value
it can hold.

| | Case | What it must do |
|---|---|---|
| S9a | **the documented minimum** | a configuration setting *only* the attributes marked `Required` — what someone following the generated docs writes — applies and then plans clean. Needing more than that is a finding, not a fixture detail: either the attribute should be `Required`, or it needs a working default |
| S9b | **each optional block at its own minimum** | S9a omits optional blocks entirely and so never reaches the defaults declared *inside* one — which is exactly where the `error_handler` defect lived. Every optional block gets the same treatment, carrying only the attributes the block itself marks `Required` |
| S9c | **every `Default:` against a live server** | the empty plan after S9a/S9b is the check: a default the server rewrites, or rejects, shows up there and nowhere else. `""` is legal for some Zabbix properties and rejected outright for others, so probe — do not reason from the documentation |
| S9d | **set, then unset** | every optional scalar that can hold an empty value is set to a non-default, then returned to empty, and the plan after the clear must be empty. Zabbix reads an absent property as "leave as is", so a clear that does not reach the server leaves a diff that reapplies forever |
| S9e | **the clear reaching the server** | where the property is merged rather than replaced — host `inventory`, interface `details` — assert against a re-read, as `C6` does for collections. State is written by the provider's own read, so a dropped clear still looks right in state |

S9d is the scalar twin of `C6`, and it has the same cause. The mechanical form of the
bug is an `omitempty` struct tag in `internal/zabbix/` on a property the user is allowed
to empty — see [CONTRIBUTING.md § "The `omitempty` trap"](./CONTRIBUTING.md#the-omitempty-trap)
for which tag a new field should carry. It is not the only form: `d.GetOk` reports `""`
as "not set" and will drop the key just as thoroughly, which is how the host inventory
fields were unclearable while every struct tag involved was correct.

Two traps worth naming, because both hid a defect behind something that looked fine:

- **A default hides the empty value.** An attribute with no default reaches `""` by being
  omitted, so an existing fixture may stumble over it. An attribute *with* a default only
  reaches `""` if the user writes it out, and no fixture ever does — which is why
  `status_codes` and the SNMPv3 credentials outlasted the ones without defaults.
- **A `ValidateFunc` can make a legal value unreachable.** `StringIsNotWhiteSpace` on an
  optional attribute forbids the empty string outright. Before adding one, check that
  Zabbix agrees the value is invalid; four interface attributes carried it for values
  Zabbix accepts, so the plugin rejected a configuration the server would have taken.

**Definition of done for a collection attribute** (`C1`–`C7`).

Most of the provider's schema is collections — sets of ids, blocks of interfaces, items,
conditions, tags, macros, preprocessing steps. Every collection bug found so far hid
behind a fixture that used **exactly one element**: the 8.0 graph reordering, the LLD
formula ids the provider echoed back to a server that rejects them, and a whole class of
silently-dropped edits that only a multi-element set can expose. One element cannot
distinguish a set from a list, cannot show an ordering assumption, and cannot show an
identity assumption. A collection is not tested until it has been tested plural.

| | Case | What it must do |
|---|---|---|
| C1 | **none** | attribute omitted entirely (where it is optional) — creates, plans clean, and imports back empty |
| C2 | **one** | the trivial case |
| C3 | **many** | three elements where the server may reorder, two otherwise. At least two must be *of the same kind* — two SNMP interfaces, two conditions on one macro — so that element identity is proven to come from content and not from position |
| C4 | **reordered** | the same elements rewritten in a different order, as a `PlanOnly: true` step. For a **set** this must plan empty. For a **list** the opposite is the assertion: the reorder must produce a diff and the new order must survive the round trip, because that is what claiming `TypeList` means |
| C5 | **edit one of many** | change one attribute of one element and assert the *others* are untouched — and, where the object has a server-assigned id, that the edited element kept it rather than being silently recreated |
| C6 | **remove one, then all** | N → N-1 → 0. The removal has to be shown reaching the server, not merely leaving state; Zabbix's update calls replace collections wholesale and an omitted element is a deletion |
| C7 | **import at full size** | `ImportStateVerify` with the collection at its largest. This is the only check that the flatten function and the set hash agree — a mismatch there is invisible in every other step |

Assert set elements by content (`TestCheckTypeSetElemNestedAttrs`,
`TestCheckTypeSetElemAttrPair`), never by index: a set's indices in test state are
positional artefacts of the shim and mean nothing.

Two schema rules that fall out of the same place, and that C3–C5 are what catch:

- **A `TypeSet` hash must cover every user-settable attribute of the element.**
  `helper/schema`'s `diffSet` returns early when the old and new hash-code lists match
  and never compares the elements themselves, so an attribute left out of the hash can
  never be seen to change — the edit is *silently discarded*, with an empty plan.
  Exclude only server-assigned ids, which config does not have and which would otherwise
  replace every element on every plan (`hashElementExcept` in `provider/utils.go`).
- **`TypeList` is a claim that order is semantic**, and C4 is where that claim gets
  audited against the server. See CLAUDE.md § "Collections".

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
| 4 — feature completeness | 35 instances (+2 deferred) | ~250 (35 × S1–S9) |
| 5 — documentation | 7 | ~55 (per-resource descriptions) |
| 6 — maintenance posture | 5 | ~10 |
| 7 — collection test backfill | 8 | ~40 |
| 8 — edge cases and failure modes | 6 | ~50 |

Critical path to v2.0.0 is phases 0–3: **42 enumerated tasks, ~130 sub-tasks.**
Everything after is incremental `v2.x`.

Phase 4 is the long tail by construction and is not meant to complete before v2.0.0.

---

## Phase 0 — Toolchain, repo structure, CI baseline  ✅ DONE

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
- [x] **Toolchain decision: raise Go to 1.25.x and finish the SDK bump to 2.40.1.**
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

- [x] **Standalone `docker-compose.test.yml` at repo root**, runnable with plain
      `docker compose` — not coupled to `.devcontainer`. One stack per version:
      6.0, 7.0, 7.4, **8.0**.
- [x] **Zabbix 8.0 stack from day one.** 8.0 LTS is due Q3 2026 and no
      `zabbix/zabbix-server-pgsql:*8.0*` tag is published yet, but `ubuntu-trunk` is
      built nightly (last push 2026-07-30) and is the 8.0 pre-release line. Add the stack
      now pointing at `ubuntu-trunk`, mark it **non-blocking** in CI, and flip it to a
      pinned `ubuntu-8.0-*` tag on GA. This surfaces 8.0 API breakage months before it
      ships instead of after.
      Note trunk is a moving target — treat 8.0 failures as signal to investigate, never
      as a release gate, until the tag is pinned.
- [x] Switch to **PostgreSQL** (`zabbix-server-pgsql` / `zabbix-web-nginx-pgsql`) with one
      DB container per version. The current single shared MySQL 8.0 with
      `mysql_native_password` is fragile and exists mainly for the 4.0/5.0 images being
      dropped. Pin image tags to explicit patch versions for reproducibility.
- [x] Per-version healthchecks that wait on `api_jsonrpc.php` answering `apiinfo.version`,
      not just the web root — the frontend responds before the DB schema is loaded, which
      is the classic flaky-start cause.
- [x] **Makefile rework:**
      - `make testenv-up` / `testenv-down` / `testenv-logs` (+ `testenv-up-60` etc. to
        bring up a single stack — four full Zabbix stacks is a lot of laptop RAM)
      - `make test60 test70 test74 test80`; `make testacc` runs the three gated versions,
        `make testall` adds 8.0
      - `make test-one TEST=TestAccResourceHost VER=74` for single-test iteration, which
        is what day-to-day development will actually use
      - `ZABBIX_URL` resolved from a per-version localhost port so the suite runs outside
        the container too
      - per-version logs (`provider/acc-<ver>.log`)
- [x] **Version-add must be a one-liner.** Parameterise the compose stacks (YAML anchors
      or a small template) so adding 8.2 later is a config entry, not a copy-pasted 40-line
      block. The current file is four hand-duplicated stacks; that is why it rotted.
- [x] **CI acceptance workflow:** GH Actions matrix over 6.0/7.0/7.4 as release-gating,
      8.0 with `continue-on-error: true`. Compose brought up as a step. Nightly schedule
      plus manual dispatch — acceptance runs are too slow for every PR.
- [x] **`CheckDestroy` on every resource test.** There is currently **zero** `CheckDestroy`
      in the suite. This is a correctness gap rather than hygiene: a test can pass while
      leaving its objects on the server, so a broken `Delete` path goes undetected. Every
      resource has a delete path and none is verified.
- [x] **Sweepers** (`resource.AddTestSweepers`) as a *recovery* path, for runs killed
      mid-flight where `CheckDestroy` never runs. Keyed off the consistent `test-*` prefix.
- [x] ~~Unique naming helper~~ — **dropped after review.** Each version has its own
      dedicated Postgres container and tests run sequentially (no `t.Parallel`), so
      cross-version and intra-run collisions are impossible. Consistent `test-*` names are
      better anyway, because sweepers need a predictable prefix. The random-suffix
      convention comes from cloud providers sharing one account across many CI jobs — that
      topology does not exist here.
- [x] Devcontainer updated to reference the new compose file rather than duplicating it.

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

- [x] **Bearer auth.** In `internal/zabbix/base.go`, send `Authorization: Bearer <token>`
      when `Config.Version >= 60400`; keep the `auth` body property below that (6.0/6.2
      still need it). Drop `auth` from the request struct on the new path entirely.
- [x] **`apiinfo.version` must stay unauthenticated** — verify the probe in `NewAPI`
      runs before any credential is attached.
- [x] **`selectGroups` removal.** Replace with `selectHostGroups` (host) /
      `selectTemplateGroups` (template) on `>= 70200`; keep `selectGroups` below.
      Sites: `resource_host.go:596,625`, `resource_template.go:126,153`.
- [x] **Host ↔ proxy.** `proxy_hostid` → `proxyid` on `>= 70000`, and set `monitored_by`
      (0 server / 1 proxy / 2 proxy group) accordingly. `internal/zabbix/host.go:63`.
- [x] **LLD HTTP headers/query fields** → array-of-`{name,value}` on `>= 70000`.
- [x] **Preprocessing** `params` mandatory for "check for not supported value" on `>= 70000`.
- [x] **`ruleid` on `itemprototype.update`** — a fourth 7.2 hard-error parameter,
      alongside `auth`, `selectGroups` and `proxy_hostid`. Missed in the original audit
      because it is an object property on the update path rather than a request
      parameter; found later by the first prototype acceptance test. Fixed in
      `prepItemsUpdate`.
- [x] **Strict-validation audit.** Grep every `zabbix.Params{...}` literal in provider and
      client and check each key against the 7.4 reference. 7.2 made unknown parameters a
      hard error rather than ignoring them, so this needs to be exhaustive, not spot-checked.
- [x] Add `Script` (21) and `Browser` (22) to the `ItemType` enum.

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

## Phase 3 — Correct and test what already exists  ✅ DONE

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

- [x] ~~Bump `SchemaVersion` on every item resource; add a `StateUpgraders` entry that
      strips `applications` from prior state.~~ **Not needed — verified, not assumed.**
      `helper/schema`'s `GRPCProviderServer.removeAttributes` (`grpc_provider.go:516`)
      deletes every key the current schema no longer declares, *after* any upgraders run
      and *before* decoding, whether or not the resource declares one. The flatmap path
      does the same by only iterating declared attributes. Twenty `SchemaVersion` bumps
      would each have been handed state the key was already gone from.
      `TestV0StateFixtureItemsDropRemovedAttributes` pins this with a negative control
      (decoding the fixture straight into the v2 schema must fail), so it breaks loudly
      if a future SDK stops doing it. Same applies to the legacy SNMP item attributes.
      What no upgrader can fix is the *config* — `applications = [...]` in HCL is a hard
      error and needs a hand edit. `MIGRATING.md` § 3 covers it.
- [x] `SchemaVersion` + upgrader on `zabbix_template` for the host-group →
      template-group `groups` transition on 6.2+.
- [x] For the removed resources (`zabbix_application`, `zabbix_item_aggregate`,
      `zabbix_proto_item_aggregate`) there is no upgrade path — document
      `terraform state rm` in `MIGRATING.md`.
- [x] Test the upgraders with a checked-in `v0.17.0` state fixture —
      `provider/testdata/v0.17.0-state.json` and `-flatmap.json`, driven through the real
      `UpgradeResourceState` handler. Flatmap is covered too, which is the path the
      TypeSet upgraders exist for and which nothing had exercised.

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
- [x] **Migrate to `terraform-plugin-testing`** — done at **v1.13.3**, the newest release
      whose `go` directive (1.23.0) builds on the pinned go1.23.4 toolchain; v1.14+ needs
      Go 1.24/1.25, the same wall the SDK bump hit. Test-only — the provider stays on
      `terraform-plugin-sdk/v2`. Nothing the suite used was renamed or dropped.
      `TestCase.Providers` still exists there but is deprecated, so the 48 call sites moved
      to `ProviderFactories` rather than leaving the suite on a deprecated field. Pulls
      `terraform-plugin-go` v0.27.0 → v0.28.0. Original note follows:
      ~~**Migrate to `terraform-plugin-testing`** — `helper/resource`'s test harness is
      deprecated in SDKv2 in favour of the standalone
      `github.com/hashicorp/terraform-plugin-testing` module. Do this before writing
      ~30 new test files, not after.~~

**Exit criteria:** every registered resource and data source has an acceptance test
including an import step; state upgraders verified against a v0.17.0 fixture; all three
supported versions green.

---

## v2.0.0 — the breaking release  ⬅ **EXIT CRITERIA MET**

Phases 0–3 are complete and the matrix is green on 6.0.48, 7.0.29, 7.4.13 and
8.0-trunk with 61 acceptance tests. The only Phase 0 item still open is the deferred
Go 1.25 / SDK 2.40.1 toolchain bump, which is not release-blocking.

Cut from the `v2` branch.
Everything breaking lands in this one release:

- Minimum Zabbix version is now 6.0
- `graph.item`, `host.interface` and the LLD filter `condition` block are **sets, not
  lists** — server return order is not stable across Zabbix versions. Sets cannot be
  indexed: use `one(...)` instead of `interface[0]`
- `zabbix_application` and its data source removed → migrate to item tags
- `zabbix_item_aggregate` / `zabbix_proto_item_aggregate` removed → migrate to
  calculated items with aggregate functions
- `zabbix_template.groups` references template groups on 6.2+ → `zabbix_templategroup`
- `applications` attribute removed from every item schema

`MIGRATING.md` is written. Ship with it covering each of the five, with before/after HCL, framed as a
`v0.17.0 → v2.0.0` upgrade.

Post-release, `v2` becomes the default branch and `master` is archived.

---

## Phase 4 — Feature completeness

Ordered by user value; see [API-COVERAGE.md §2](./API-COVERAGE.md) for the full list.
This is the post-2.0 backlog — additive, parallelisable, no longer on the critical path,
and shipped as ordinary `v2.x` minor releases.

Each row below is one work item following S1–S9 from "The unit of work". Columns flag the
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

### `U1`–`U4`: an attribute is not covered until it has been changed in life

Create and destroy exercise two thirds of what a user does. Editing an existing
resource is the most common of the three and was the least tested.

| | Case | What it must do |
|---|---|---|
| U1 | **changed in life** | every settable attribute changed on an existing resource, asserted against a **server re-read**, not merely Terraform state |
| U2 | **and it was an update** | `plancheck.ExpectResourceAction(...Update)` on the step. Without it a wrongly-`ForceNew` attribute passes: Terraform destroys and recreates, the end state matches, the test is green, and the user has silently lost their item's history |
| U3 | **create-only is `ForceNew`** | every attribute Zabbix refuses to update is `ForceNew` and asserts replacement |
| U4 | **nothing else is `ForceNew`** | probed against live servers, never inferred. Replacing an item **discards its history**, so a needless `ForceNew` is silent data loss rather than an inconvenience |

`U3` is how `hostid` (create-only from 7.0) and prototype `ruleid` (create-only from 7.2)
should have been found. Both were found by accident instead, and `ruleid` had made
**every** `zabbix_proto_item_*` resource un-updatable on current Zabbix.

Probe results across all four versions: `item.update` and `discoveryrule.update` reject
`hostid` on 6.0–8.0; `itemprototype.update` rejects `ruleid` from 7.0 and — worse —
*accepts and silently ignores* it on 6.0, so without `ForceNew` Terraform would report a
move that never happened. All three existing flags are correct.

Enforcement lives in `provider/acc_update_test.go`: the coverage map, the `ForceNew` set
and the exemptions are each checked against the live schema, so an attribute added
without update coverage fails the build. **Exempt by name with a reason, never by
omission** — an attribute quietly falling out of a list is exactly how `ruleid` stayed
unnoticed.

---

## Phase 8 — Edge cases and failure modes  ✅ DONE

Phases 3 and 7 answer "does the happy path work, and does it work plural?". Neither
asks what happens when things go *wrong*, and that is where the remaining risk sits.
Audited state at the time of writing:

| Dimension | Coverage |
|---|---|
| every registered resource and data source appears in a test | ✅ 37/37 |
| version gates | covered incidentally by running the four-version matrix — adequate |
| negative / error paths | ✅ closed — was 4 `ExpectError` suite-wide |
| drift / out-of-band deletion | ✅ closed — was none; all 37 resource types now covered |
| ForceNew | ✅ closed — all three fields assert replacement |
| provider configuration | ✅ closed — `token`, `serialize` and `tls_insecure` all exercised |

- [x] **`E1` — drift.** For every resource: delete the object via the API behind
      Terraform's back, then assert the next plan is non-empty and re-applies cleanly.
      This is the highest-value gap. A `Read` that fails to `d.SetId("")` on a missing
      object leaves the user permanently stuck, needing a manual `terraform state rm`,
      and it fails *silently* — several resources do handle it, none is verified.
- [x] **`E2` — negative paths.** `ExpectError` on the validation that actually matters:
      every `_LOOKUP_ARR` enum, `dependent` LLD rejecting a non-zero `delay`,
      `zabbix_templategroup` below 6.2, proxy active/passive mode mismatches, a
      `custom` LLD `evaltype` with a malformed formula.
- [x] **`E3` — ForceNew.** Change each `ForceNew` field and assert the resource is
      *replaced*, not updated in place or silently ignored.
- [x] **`E4` — provider configuration.** `token` auth end to end (it sets `api.Auth`
      directly and skips `Login()`, so it exercises a different path from
      username/password, and Phase 2a rewrote exactly that code). Also `serialize`, and
      `tls_insecure`. Token auth was spot-checked by hand against 7.4 and works; it is
      still unexercised by the suite.
- [x] **`E5` — data source not-found.** Looking up something that does not exist should
      produce a clear error, not a panic or an empty resource. Note the `zabbix_template`
      data source panicked the provider on *every* read until Phase 3c; a not-found test
      is the cheapest guard against that class.
- [x] **`E6` — scalar boundaries.** Empty strings where optional, unicode in names and
      descriptions, and the macro/tag values Zabbix treats specially.

Sequence `E1` first: it is the only one whose absence can leave a user unable to
recover.

## Phase 5 — Documentation  ✅ DONE

- [x] Adopt **`terraform-plugin-docs`** (`tfplugindocs generate`) — the 38 files under
      `docs/` are hand-written and will drift. Hand-written prose moves into `templates/`
      and per-resource `examples/`.
- [x] Every schema field gets a `Description` (tfplugindocs renders these). This is the
      bulk of the work; do it incrementally, one resource at a time.
- [x] Split the 49 KB `README.md`: short overview + dev/contributing guide; resource
      reference moves to generated docs.
- [x] Document the version support policy and per-resource minimum Zabbix version.
- [x] Populate `examples/` — currently one directory. `tfplugindocs` pulls
      `examples/resources/<name>/resource.tf` and `examples/data-sources/<name>/data-source.tf`
      into generated pages, and `import.sh` for the import section. One per resource.
- [x] Decide the fate of `utils/template2terraform` — the standalone Python XML→HCL
      converter. Either bring it into CI (test it, document it, note the Python 3
      dependency) or move it to a separate repo. Right now it is undocumented and untested
      and will rot.
- [x] Verify Terraform Registry publication (docs layout, `index.md`, category, logo).

---

## Phase 6 — Routine maintenance posture  ✅ DONE

- [x] `CHANGELOG.md` (keep-a-changelog) + `.changie.yaml` or equivalent
- [x] Renovate/Dependabot for Go modules and GitHub Actions
- [x] Nightly acceptance run against supported versions; issue opened automatically on failure
- [x] A **new-Zabbix-release runbook**: add compose stack → run matrix → read the upstream
      `manual/api/changes` page → gate deltas → update support table → drop the version
      that fell out of limited support. For 8.0 the first two steps are already done in
      Phase 1 — the runbook's first real exercise is promoting the trunk stack to a pinned
      `ubuntu-8.0-*` tag and making it release-blocking on GA.
- [x] `CONTRIBUTING.md` documenting the item triad pattern and the version-gate idiom
- [x] **`RELEASING.md`** — not called for by this phase, and the real gap. Nothing
      described how to cut v2.0.0 itself: push `v2`, enable Actions, remove the
      `refs/heads/v2` guards, scope `release.yml` to `v2.*` tags, tag, make `v2` the
      default branch. Note Dependabot also only activates once `v2` is default, since it
      reads its config from the default branch.
- [x] `goreleaser build --snapshot` — Phase 0's last exit criterion, finally run. Eight
      targets, config validates, binaries named `terraform-provider-zabbix_v<version>` as
      the Registry requires, and the `go mod tidy` before-hook leaves `go.mod`/`go.sum`
      byte-identical.

---

## Phase 7 — Collection test backfill  ✅ DONE

Apply `C1`–`C7` (see "The unit of work") to every collection already in the schema. This
is a backfill: the rules were written after the 8.0 graph regression showed what a
one-element fixture cannot see, and most of the suite predates them.

Audit as of the TypeSet conversion — cardinalities are the maximum any single test step
reaches, across the whole suite:

| Area | Attribute | Now | Missing |
|---|---|---|---|
| graph, proto_graph | `item` (set) | 1, 2, 3; reorder; edit-one-of-many; import | C6 — an item is never removed from a graph. `zabbix_graph` itself has no import step (only `zabbix_proto_graph` does) |
| host | `interface` (set) | 1–4 incl. two SNMP; reorder; edit-one-of-many; remove-one; import | complete (C1/C6-to-zero are N/A: `interface` is Required, Min 1) |
| lld_* | `condition` (set) | 1, 2, 3; reorder; edit-one-of-many; remove-one; `evaltype = "custom"`; import | C6 — never returns to zero conditions |
| item_*, proto_item_*, lld_* | `preprocessor` (**list**) | max 3, and only in `item_agent` + `proto_item_snmp` | **C4 — no test reorders preprocessing steps.** This is the one collection whose order genuinely is semantic, and nothing proves the provider preserves it. Also C6, and 8 of 10 item types never exercise it at all |
| host, template | `macro` (set) | 1, 2; remove-all | C4, C5 — never reordered, never edited one-of-several |
| host, trigger, item_*, proto_* | `tag` (set) | 2 on host, 1 everywhere else; host covers edit-one and remove-all | C3 on every resource but host; C4 everywhere |
| trigger, proto_trigger | `dependencies` (set of ids) | 1, on `trigger` only | C2 on `proto_trigger`, which never sets one at all; C3, C6 on both — a trigger never depends on two triggers, and a dependency is never removed |
| host, template | `templates` (set of ids) | 1 | C3, C6 — and `existingTemplateIds`/`templates_clear`, which exists precisely to survive a template being destroyed in the same apply, is only ever exercised with one template |
| host, template | `groups` (set of ids) | 1 (2 once) | C3, C6 — a group is never removed from a host |
| item_http, proto_item_http, lld_http | `headers` (map) | 1, 2 | C6 — headers never emptied |
| host | `inventory` (list, single block) | 0, 1, changed | complete — not a collection, a single nested block |

Tasks:

- [x] `preprocessor` reorder test first — it is the only list left claiming semantic
      order, and the claim is currently unverified on any version
- [x] C3 + C4 for `tag` on one item resource and on `trigger`; the item triad shares
      `common_tag.go`, so one item type covers the machinery for all ten
- [x] C3 + C6 for `dependencies`, `templates`, `groups`
- [x] C6 for `item`, `condition`, `headers`
- [x] import step for `zabbix_graph`
- [x] Where a collection's coverage comes from shared machinery (`common_tag.go`,
      `common_macro.go`, `common_lld.go`, `common_item.go`), test it thoroughly **once**
      and reference that from the others rather than copying eleven near-identical
      fixtures. Note where that decision is made, so a later reader does not read the
      absence as an oversight.
- [x] Mirror `C1`–`C7` into CLAUDE.md § "Testing expectations" — the rules are only
      useful if they are read before the test is written, and CLAUDE.md is what gets read
- [ ] **Standing rule, not a task:** fold the checklist into the S1–S9 review for every Phase 4 resource, so the
      backlog stops growing while it is being paid down

**Exit criteria:** every row in the table above reads "complete", and any deliberate
exception is written down next to the collection it applies to.

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

Phase 7 is not a stage in that sequence: it is a standing quality bar. The `C1`–`C7`
checklist applies to new work from now on, and the backfill table is worked through
alongside phases 4–6 rather than gating them.

Phases 0–3 are one campaign on the critical path — the provider is non-functional on
current Zabbix until Phase 2 ships. Phase 4 onward is the routine-maintenance backlog,
released as incremental `v2.x` minors.

Bringing 8.0 (via `ubuntu-trunk`) into the matrix now rather than on GA means the
Phase 6 new-release runbook gets exercised continuously instead of once, and 8.0 API
breakage surfaces while there is still time to respond to it.
