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
- Go
- **A `terraform` binary on `PATH`.** The SDK's acceptance-test driver shells out
  to it. If it is missing — or is an inactive version-manager shim, e.g. `asdf`
  with no version selected — every acceptance test fails immediately with
  `Error setting test config: exit status 126` before any Zabbix call is made.
  That error means your local toolchain, not the provider. Check with
  `terraform version`.

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

## Current status — baseline

The acceptance suite does **not** pass yet. That is expected: the harness landed
before the provider fixes it exists to verify. Baseline from the first full run
(2026-08-03, provider at `4eb71cf`+):

| Version | Result | Cause |
|---|---|---|
| 6.0 | 12 fail / 3 acc pass | `DBEXECUTE_ERROR`, plus `Host group "test-group" already exists` — fixture collisions from aborted runs |
| 7.0 | 11 fail / 4 acc pass | same profile as 6.0 |
| 7.4 | 13 fail / 2 acc pass | **all** fail on `Invalid parameter "/": unexpected parameter "auth"` |
| 8.0 | fails | same `auth` error as 7.4 |

Two distinct root causes, matching PLAN.md:

1. **7.2+ auth.** The client sends the token as a JSON-RPC `auth` body property,
   which 7.2 removed in favour of an `Authorization: Bearer` header. Since 7.2
   also made unknown request parameters a hard error, *every* call fails. This is
   a single fix that should clear the whole 7.4 and 8.0 columns at once
   (PLAN.md Phase 2a).
2. **Fixtures.** 6.0/7.0 failures are pre-6.0 test fixtures plus fixed fixture
   names with no sweepers, so an aborted run poisons the next one — the
   `AddTestSweepers` and unique-naming items still open in PLAN.md Phase 1.

Note the harness itself is not implicated in any of these: all four stacks come
up healthy and answer `apiinfo.version` correctly.
