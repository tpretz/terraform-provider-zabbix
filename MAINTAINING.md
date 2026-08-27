# Maintaining

The routine, calendar-driven half of this project: what runs on its own, what a
new Zabbix release costs, and when a version gets dropped.

Companion documents: [CONTRIBUTING.md](./CONTRIBUTING.md) for how a change is
made, [DEVELOPMENT.md](./DEVELOPMENT.md) for build and docs mechanics,
[TESTING.md](./TESTING.md) for the acceptance harness,
[RELEASING.md](./RELEASING.md) for cutting a release.

---

## What is automated

| | What | State |
|---|---|---|
| `.github/workflows/ci.yml` | build, vet, `go test ./provider/`, `gofmt`, toolchain pins, on push and PR | **disabled** until v2.0.0 — see RELEASING.md |
| `.github/workflows/nightly.yml` | full acceptance matrix, 03:17 UTC, opens an issue on failure | **disabled** until v2.0.0 |
| `.github/dependabot.yml` | weekly Go module and GitHub Actions updates | inert until `v2` is the default branch |
| `.github/workflows/release.yml` | goreleaser on a `v2.*` tag | **disabled** until v2.0.0 |

Everything is deliberately inert on the `v2` branch. Turning it on is a
checklist in [RELEASING.md](./RELEASING.md), done once.

Two things `ci.yml` does **not** do, and should: it runs `go test ./provider/`
rather than `go test ./...`, so `internal/zabbix`'s credential-redaction tests
never run in CI; and it does not run `make docs-check`, so a schema
`Description` change can be merged with `docs/` left stale. Both are covered by
the pre-flight checklist in RELEASING.md § 0 in the meantime.

### Reading a nightly failure

The nightly opens **one** issue labelled `nightly-failure` and comments on it
thereafter, so a persistent break does not produce a new issue every night.
Close it when the run goes green; the next failure opens a fresh one.

- A failure on **6.0, 7.0 or 7.4** is release-gating. It either blocks the next
  release or gets reverted.
- A failure on **8.0 alone opens nothing**. That job is `continue-on-error` and
  tracks the moving `ubuntu-trunk` image; read it, do not gate on it. It exists
  to give months of warning before an API change lands in a release.
- Each job uploads `provider/acc-<ver>.log` as an artifact, and dumps the
  container logs for its stack on failure.

Reproduce locally with the version that failed:

```bash
make testenv-up-74 && make test74
make test-one TEST=TestAccResourceHost VER=74     # then narrow
```

### Known upkeep: the Node 20 action deprecation

Every CI run currently prints:

> Node.js 20 is deprecated. The following actions target Node.js 20 but are being
> forced to run on Node.js 24: `actions/checkout@v4`, `actions/setup-go@v5`

A warning, not a failure — GitHub is running them on Node 24 regardless, and the
first v2.0.0 CI run was green with it. The fix is a major bump on both, across all
three workflows:

| Action | Now | Used by |
|---|---|---|
| `actions/checkout` | `@v4` | ci, release, nightly |
| `actions/setup-go` | `@v5` | ci, release, nightly |

Do it as one change and let CI prove it, rather than during a release — `release.yml`
is the one workflow with no cheap way to rehearse, so a broken bump there is found by
a failed publish. `actions/upload-artifact@v4` in the nightly is unaffected;
`crazy-max/ghaction-import-gpg@v6` and `goreleaser/goreleaser-action@v6` are current.

Dependabot is configured for `github-actions` (see below), so it will raise these
itself once it starts running — which is only after `v2` becomes the default branch,
since Dependabot reads its config from there.

### Reviewing a Dependabot pull request

Two ecosystems, both weekly and grouped, so a routine week is one or two pull
requests. What to check:

1. **CI green.** For Go module updates that is the whole review: the provider
   has no dependency it uses lightly, and the acceptance matrix is what would
   catch a behavioural change. Run the nightly manually
   (`workflow_dispatch`) before merging a `terraform-plugin-*` bump.
2. **`make check-toolchain` did not fail.** If it did, the bump raised
   `go.mod`'s `go` directive past the `golang` pin in `.tool-versions`. Raise
   the pin to match **in the same merge** — do not merge the bump alone and fix
   it after. See [the toolchain pins](#the-toolchain-pins) below.
3. **No Go version was written into a workflow file.** Workflows read
   `go-version-file: go.mod`. A literal there is a pin nothing checks.
4. **Add a `CHANGELOG.md` entry** only when the bump is user-visible — an SDK
   minor that changes protocol behaviour is; a transitive patch is not.

### The toolchain pins

The Go version is pinned in three places and they must stay consistent:

| Where | What it means | Who moves it |
|---|---|---|
| `go.mod` — `go 1.25.8` | the floor advertised to anyone building from source; must be ≥ every dependency's own directive | a dependency bump, often automatically |
| `.tool-versions` — `golang 1.25.12` | what we actually build, test and generate docs with; must be ≥ the directive | a human |
| `ci.yml`, `nightly.yml` | `go-version-file: go.mod` — derived, never edited | nobody |

`make check-toolchain` fails when the directive overtakes the pin, and CI runs
it before it even installs Go. Without it the failure is silent: with
`GOTOOLCHAIN=auto`, Go downloads whatever the directive asks for and the
`.tool-versions` pin stops describing anything.

Terraform is pinned once, in `.tool-versions`. The `Makefile` resolves the
binary and hands it to the harness as `TF_ACC_TERRAFORM_PATH`; the nightly reads
the same line with `awk`. Do not hardcode it anywhere else.

---

## Runbook: a new Zabbix version

Zabbix ships a minor release roughly every four months and an LTS every two
years. The harness was built so that adding one is cheap — **a `VERSIONS` entry
plus a three-line compose block** — and the expensive part is reading the API
changelog, which no amount of tooling removes.

Do steps 1–5 as soon as a version reaches beta or its first `ubuntu-trunk`
build. Do steps 6–8 at GA.

### 1. Add the stack

Two edits, per [TESTING.md § Adding a Zabbix version](./TESTING.md#adding-a-zabbix-version):

- one three-line block in `docker-compose.test.yml`'s `services:` — the shared
  behaviour is in the `x-*` YAML anchors, so a version entry is only image tags,
  a published port and links. Pin an explicit patch tag
  (`ubuntu-7.4.13`), never `latest`; a pre-GA line has no tag but
  `ubuntu-trunk`, which is a moving target and must be treated as such.
- one entry in the `Makefile`'s `VERSIONS`, plus its `ZBX_PORT_<ver>`. Leave it
  **out** of `GATED` for now.

Every per-version target (`test<ver>`, `testenv-up-<ver>`, `testenv-down-<ver>`,
`testenv-logs-<ver>`) is generated from `VERSIONS`. There is nothing else to
write.

```bash
make testenv-up-76 && make testenv-verify     # confirm it answers apiinfo.version
```

### 2. Run the matrix against it

```bash
make test76
```

A wholesale failure is almost always one cause — that is the shape every
version break in this provider has taken. Read the first error, not the count.
The 7.2 `auth` removal failed all 61 tests identically; the 6.2 template-group
split accounted for every remaining failure on three versions at once.

### 3. Read the upstream API changes page

<https://www.zabbix.com/documentation/X.Y/en/manual/api/changes> — the single
most valuable half hour in this runbook, and the only step that finds the
changes the tests *cannot* see. Zabbix's `.get` methods ignore unknown
parameters on every version, so a removed `select*` parameter is a **silent
wrong answer**, not an error: the resource reads back empty and the acceptance
suite may well still pass.

For each entry in the changes page, grep the tree for the property name:

```bash
grep -rn 'selectGroups\|proxy_hostid' internal/zabbix/ provider/
```

Triage into: (a) removed or renamed properties the provider sends or selects —
must be gated; (b) new properties worth exposing — Phase 4 work, not release
blocking; (c) new validation strictness — usually shows up as a hard error in
step 2 already.

Record what you found in `API-COVERAGE.md`, whether or not it needed a change.
"Read the 7.6 changes page, nothing affects us" is worth writing down; it is
the difference between a checked version and an untested one.

### 4. Gate the deltas

Add a named constant to `internal/zabbix/base.go` — `V80`, and so on — and gate
on `api.Config.Version >= zabbix.V80`. Never a bare integer, never a version
string. The encoding is `major*10000 + minor*100 + patch`, so **8.0 is
`80000` and 7.6 is `70600`**; see
[CONTRIBUTING.md § Version gating](./CONTRIBUTING.md#version-gating), which
explains why a gate of `80000` for 8.0 is right and `78000` is a version that
can never exist.

Keep the old path. A gate is not a migration: every supported version has to
keep working.

### 5. Add it to the nightly

A new entry in `.github/workflows/nightly.yml`'s matrix, with
`gating: false` while it is pre-GA. `continue-on-error` follows that flag.

### 6. Promote it at GA

- Pin the compose images to the released patch tag, replacing `ubuntu-trunk`.
- Move the version from `VERSIONS`-only into `GATED` in the `Makefile`, so
  `make testacc` covers it.
- Flip `gating: true` in the nightly matrix.
- Full matrix must be green before the promotion is committed. A version does
  not become release-gating with a known failure — resolve it or say in writing
  why it is acceptable.

### 7. Update the support tables

They are in more places than you will remember. All of them:

| File | What |
|---|---|
| `README.md` | version support policy table |
| `templates/index.md.tmpl` | the same table for the Registry landing page — then `make docs` |
| `TESTING.md` | the stack table, the 8.0 caveat section, the current-status table |
| `API-COVERAGE.md` § 6 | the Zabbix support-lifecycle table |
| `PLAN.md` § Version support policy | tiers and commitments |
| `CLAUDE.md` | the decisions table and the version-gate table |
| `DEVELOPMENT.md` | the version-gate constant table |
| `CHANGELOG.md` | an entry under `Unreleased` — a support change is user-visible |

`make docs-check` catches the generated half; nothing catches the rest, which
is why they are listed here.

### 8. Drop the version that fell out of support

The floor tracks Zabbix's own **limited support** window, so a version leaves
when Zabbix stops supporting it, not when it becomes inconvenient.

| Version | Released | Full support ends | Limited support ends |
|---|---|---|---|
| 6.0 LTS | 2022-02-08 | 2025-02-28 | **2027-02-28** |
| 7.0 LTS | 2024-06-04 | 2027-06-30 | 2029-06-30 |
| 7.4 | 2025-07-01 | until 8.0 LTS | Q4 2026 |
| 8.0 LTS | Q3 2026 | Q3 2029 | Q3 2031 |

Raising the floor is a **breaking change and needs a major version**, because
gated code is deleted rather than merely untested — that is the deal this
project made with itself in PLAN.md, and it is what makes the codebase stay
small. The procedure, which Phase 2b is the worked example of:

1. Remove the version from `VERSIONS`, `GATED`, the compose file and the
   nightly matrix.
2. Delete every gate below the new floor and the path it guarded — `grep -rn
   'zabbix\.V6' provider/ internal/` — plus the schema attributes that only
   existed for it. Phase 2b removed 1396 lines this way.
3. Delete the acceptance-test `SkipFunc`s that referenced those gates.
4. Write the `MIGRATING.md` section: what was removed, and the before/after.
5. Note it in `CHANGELOG.md` under a new major version.

The next one due is **6.0 on 2027-02-28**, which makes 7.0 the floor and
removes the `V62` and `V64` gates: template groups and bearer auth stop being
conditional.

---

## First exercise: promoting 8.0 at GA (Q3 2026)

Steps 1–5 are already done — the 8.0 stack has been in the matrix since Phase 1
and has been green throughout. What is left at GA:

```
[ ] docker-compose.test.yml: zabbix/*:ubuntu-trunk -> ubuntu-8.0-<patch>, both images
[ ] Makefile: move 80 from VERSIONS-only into GATED
[ ] nightly.yml: the 8.0 matrix entry gets gating: true
[ ] make testenv-down-80 && make testenv-up-80 && make test80 — green on the pinned tag
[ ] read https://www.zabbix.com/documentation/8.0/en/manual/api/changes end to end
    against a released version rather than trunk; gate anything new
[ ] TESTING.md: delete "About the Zabbix 8.0 stack" and the trunk caveats
[ ] the other seven support tables in step 7
[ ] CHANGELOG.md: "8.0 is now release-gating" under Unreleased
```

Two things to know going in. `ubuntu-trunk` moves under you, so a green 8.0
today is a snapshot and not a commitment — re-run against the pinned tag before
believing it. And 8.0 has already found one real bug in this provider: it
returns a graph's items in a different order than 6.0/7.x, which is what forced
the `TypeList`→`TypeSet` conversion. Expect the pinned build to find something
trunk did not.
