# Testing

The provider is tested against **live Zabbix servers**, one stack per supported
version. Everything runs with plain `docker compose` and `make` — no devcontainer
required.

| Version | Role | Web UI | Image |
|---|---|---|---|
| 6.0 LTS | gating (version floor) | http://localhost:8060/ | `ubuntu-6.0.48` |
| 7.0 LTS | gating | http://localhost:8070/ | `ubuntu-7.0.29` |
| 7.4 | gating (current release) | http://localhost:8074/ | `ubuntu-7.4.13` |
| 8.0 | **early warning, non-blocking** | http://localhost:8080/ | `ubuntu-trunk` |

Credentials for every stack: `Admin` / `zabbix`.

## Prerequisites

- Docker with Compose v2 (`docker compose`, not `docker-compose`)
- Go — `.tool-versions` pins **golang 1.25.12**
- **A `terraform` binary.** The acceptance-test driver shells out to it.
  `.tool-versions` pins **terraform 1.8.5**, and the `Makefile` resolves the
  absolute path itself (`asdf which terraform`, falling back to `command -v
  terraform`) and hands it over as `TF_ACC_TERRAFORM_PATH`. That is deliberate:
  the harness runs from a temp directory under `$TMPDIR`, so a version manager
  that resolves by walking up from `$PWD` never sees this repo's
  `.tool-versions` and falls back to a global config that may not pin
  terraform at all — which fails with `exit status 126` before any Zabbix call
  is made and looks exactly like a provider bug. No `PATH` juggling is needed;
  if you are not using a version manager, any `terraform` on `PATH` is used.

## Quick start

```bash
make testenv-up        # bring up all four stacks, wait until their APIs answer
make testenv-verify    # print each stack's apiinfo.version
make testacc           # acceptance tests on 6.0 / 7.0 / 7.4
make testenv-down      # tear down and delete the database volumes
```

Four full Zabbix stacks is a lot of laptop RAM. To work against just one:

```bash
make testenv-up-74
make test-one TEST=TestAccResourceHost VER=74
make testenv-down-74
```

`make` with no target lists everything.

## Day-to-day iteration

`test-one` is the one to reach for:

```bash
make test-one TEST=TestAccResourceHost VER=74   # single test, single version
make test-one TEST='TestAccResourceItem.*' VER=70
```

`TEST` is a Go test regex (`go test -run`). `VER` defaults to `74`.

Whole-version runs, and the two aggregates:

```bash
make test60 / test70 / test74 / test80   # one version, whole suite
make testacc                             # 6.0 + 7.0 + 7.4 — the release gate
make testall                             # testacc plus 8.0 (reported, never gating)
```

`testacc` runs every gated version even if an earlier one fails, then exits
non-zero — so one command gives you the whole cross-version failure picture
rather than stopping at the first break.

Each run writes `provider/acc-<ver>.log` (`make cleanlogs` to remove them).

Useful knobs: `TEST_TIMEOUT` (default `60m`), `TESTARGS` (extra `go test` flags,
e.g. `TESTARGS=-failfast`), `ZBX_HOST` (default `localhost`).

Unit tests need no server at all: `make test`.

## About the Zabbix 8.0 stack

Zabbix 8.0 LTS is not released yet. No `zabbix/zabbix-server-pgsql:*8.0*` tag
exists, so the 8.0 stack tracks **`ubuntu-trunk`**, the nightly build of the 8.0
pre-release line. It currently reports `8.0.0`.

That tag is a **moving target**: it changes under you between runs and can break
upstream for reasons unrelated to this provider. Its purpose is to surface 8.0
API breakage months before 8.0 ships.

**Treat 8.0 failures as signal to investigate, never as a release gate.** This is
why `make testacc` excludes it and only `make testall` runs it.

On 8.0 GA: pin both 8.0 images to `ubuntu-8.0-<patch>` in
`docker-compose.test.yml`, move `80` from `VERSIONS` into `GATED` in the
`Makefile`, and drop the caveats above.

## Adding a Zabbix version

Deliberately a two-line change — the previous harness rotted because it was four
hand-duplicated 40-line stacks.

1. Append one three-line block to `services:` in `docker-compose.test.yml`
   (copy the pattern in the header comment; the shared behaviour lives in the
   `x-*` YAML anchors, so a version entry is only image tags, port and links).
2. Add the version to `VERSIONS` in the `Makefile` and give it a `ZBX_PORT_<ver>`.

All per-version targets (`test<ver>`, `testenv-up-<ver>`, `testenv-down-<ver>`,
`testenv-logs-<ver>`) are generated from `VERSIONS` — there is nothing else to
write.

## How the harness is built

Each version gets its **own PostgreSQL container** plus `zabbix-server-pgsql`
and `zabbix-web-nginx-pgsql`. Nothing is shared between versions, so one stack
cannot corrupt another and stacks can be started and destroyed independently.
Image tags are pinned to explicit patch versions (8.0/trunk excepted) so a rerun
months from now is the same environment.

### Healthchecks wait on the API, not the web root

The frontend serves HTTP **before** the Zabbix schema finishes loading. A
`curl -f http://localhost:8080/` healthcheck — what this harness used to use —
therefore reports *healthy* while the database is still empty, which is the
classic cause of flaky first-run acceptance tests. Verified against a
schema-less database:

```
curl -f http://localhost:8080/     -> HTTP 200                    (healthy, but unusable)
apiinfo.version                    -> {"error":{... "the table \"dbversion\" was not found"}}
```

So each web container's healthcheck POSTs an `apiinfo.version` JSON-RPC call to
`api_jsonrpc.php` and requires a `"result"` in the response. `make testenv-up`
uses `docker compose up --wait`, so it does not return until every stack is
genuinely answering API calls.

## Troubleshooting

**Port already in use** — 8060/8070/8074/8080 must be free; 8080 is the usual
collision. Change the `ports:` entry for that version and its `ZBX_PORT_<ver>`
in the `Makefile`.

**A stack won't go healthy** — `make testenv-logs-74` (or `make testenv-status`
for a health summary). First boot has to load the schema; the healthcheck allows
a 90s grace period plus 90 retries.

**Tests fail with stale objects from an aborted run** — reset just that version:
`make testenv-down-74 && make testenv-up-74`.

**Refresh the 8.0 nightly** — `make testenv-pull` then recreate the stack.

## Running inside the devcontainer

`.devcontainer/` still works and no longer duplicates the stacks:
`devcontainer.json` merges `.devcontainer/docker-compose.yml` (the dev container)
with the repo-root `docker-compose.test.yml` (the stacks) into one compose
project. It sets `ZBX_IN_CONTAINER=1`, which makes the `Makefile` address the
stacks by compose service name (`http://zabbix-web-74:8080/api_jsonrpc.php`)
instead of via published localhost ports. All the `make` targets above behave
identically.

## Current status

**Green on all four versions.** 173 tests in `./provider`, 143 of them
acceptance, roughly 355-365s per version:

| Version | Result | Skips |
|---|---|---|
| 6.0.48 | pass | 3 — the `zabbix_templategroup` tests, gated to 6.2+ |
| 7.0.29 | pass | 1 |
| 7.4.13 | pass | 1 |
| 8.0-trunk | pass | 1 |

The remaining skip on every version is version-bound behaviour guarded by a
`SkipFunc`, not a failure.

Getting here was PLAN.md phases 0-3 plus phase 8. The baseline this file used to
record — 7.4 and 8.0 failing wholesale on `Invalid parameter "/": unexpected
parameter "auth"`, 6.0/7.0 failing on pre-6.0 fixtures and un-swept objects — is
history: bearer auth landed in Phase 2a, the pre-6.0 code paths and fixtures were
deleted in Phase 2b, and every resource now has a `CheckDestroy` and a sweeper.

8.0 passing is welcome but is **not** a guarantee: `ubuntu-trunk` moves under you
(see below), so treat a green 8.0 as a snapshot rather than a commitment.
