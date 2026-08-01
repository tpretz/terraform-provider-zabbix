# Implementation contract for new resources

Read this before adding a resource. Every rule here comes from a bug that was
found by running acceptance tests against a real Zabbix 7.0.26 server.

## Layout

For a Zabbix API object `foo`:

- `go-zabbix-api/foo.go` — the typed API client: struct, slice type, and
  `FoosGet` / `FooGetByID` / `FoosCreate` / `FoosUpdate` / `FoosDeleteByIds`.
- `provider/resource_foo.go` — `resourceFoo()` (+ `dataFoo()` if a lookup by
  name makes sense).
- `provider/resource_foo_test.go` — acceptance tests.

Do **not** edit `provider/provider.go`; report the registration lines instead.

Name every package-level helper so it cannot collide: `fooBuildObject`,
`fooFlattenBars`, not `buildObject` / `flattenBars`.

## Client conventions

Follow `go-zabbix-api/host_group.go` and `token.go`.

```go
func (api *API) FoosGet(params Params) (res Foos, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	err = api.CallWithErrorParse("foo.get", params, &res)
	return
}
```

Create returns IDs in `result["fooids"]`; write them back onto the structs.
Delete takes a plain `[]string` of IDs.

### JSON tags — the rules that actually bite

1. **`omitempty` is case sensitive.** `encoding/json` only honours the exact
   lowercase string. `json:"x,omitEmpty"` silently serializes always. This was a
   real bug in `item.go`.
2. **Read-only fields must never be sent.** Anything Zabbix populates via a
   `select...` parameter (`hosts`, `parentTemplates`, `discoveryRule`,
   `lastaccess`, `created_at`) needs `omitempty`, or `json:"-"` if the provider
   never reads it. Zabbix rejects unknown parameters outright with
   `unexpected parameter "x"`.
3. **Immutable fields must be cleared on update.** Zabbix rejects `hostid`,
   `userid`, `ruleid` and friends on `*.update` even when the value is
   unchanged. Give them `omitempty` and zero them in the update handler:
   ```go
   obj.UserID = "" // not updatable, token.update rejects it
   ```
4. Fields removed in newer Zabbix versions must stop being sent. Version-gate
   with `api.Config.Version >= 60200` or drop them with `json:"-"` and a comment
   naming the version that removed them.

## Schema conventions

- Explicit `Create` / `Read` / `Update` / `Delete`, plus
  `Importer: &schema.ResourceImporter{State: schema.ImportStatePassthrough}`.
- Snake_case keys matching Zabbix field names where sensible.
- Map Zabbix's numeric enums to readable strings with a lookup pair, as
  `HINV_LOOKUP` / `HINV_LOOKUP_REV` do in `resource_host.go`. Validate with
  `validation.StringInSlice(FOO_ARR, false)`.
- Mark secrets `Sensitive: true`.

### Idempotency — the single most common failure

If Zabbix computes or defaults a value server side, the field **must** be
`Computed: true`, otherwise Terraform reports a permanent diff and the
acceptance test fails with "After applying this step, the plan was not empty".

Real examples that were broken:
- `template.name` defaults to `host` server side.
- `graph.ymin_itemid` / `ymax_itemid` come back as `"0"` when unset.

For a field the user may set but Zabbix will otherwise fill in, use
`Optional: true, Computed: true` and no `Default`.

Conversely, if Zabbix *requires* a field that is logically optional, give the
schema a `Default` matching Zabbix's own default — do not leave it empty and
rely on `omitempty`, which produces `the parameter "x" is missing` (this was the
preprocessing `error_handler` bug).

## Tests

Put them in `provider/resource_foo_test.go`, named `TestAccFoo...`.

```go
func TestAccFoo(t *testing.T) {
	id := resource.UniqueId()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{Config: cfg(id, "first"),  Check: ...},
			{Config: cfg(id, "second"), Check: ...},  // exercises Update
		},
	})
}
```

Requirements:
- Always include a **second step that changes a field**. Create-only tests miss
  the entire class of "field rejected on update" bugs.
- Use `resource.UniqueId()` in every name so parallel runs do not collide.
- The framework asserts an empty plan after each step, which is what catches
  missing `Computed`.

Acceptance tests **must** be run with `-v` or the SDK refuses to start them.

```shell
export ZABBIX_URL=http://zabbix-web:8080/api_jsonrpc.php
export ZABBIX_USER=Admin ZABBIX_PASS=zabbix TF_ACC=1
go test ./provider/... -v -run TestAccFoo -timeout 10m
```

## Verify the API contract before writing Go

Do not trust the documentation about which parameters are accepted. Probe the
live server first:

```shell
Z=http://zabbix-web:8080/api_jsonrpc.php
TOK=<token from /workspace/env.txt>
curl -s -X POST $Z -H 'Content-Type: application/json-rpc' \
  -H "Authorization: Bearer $TOK" \
  -d '{"jsonrpc":"2.0","method":"foo.create","params":{...},"id":1}'
```

Check create, then update with the same body, then delete. The update call is
where immutable-parameter rejections show up. Clean up whatever you create.

## Definition of done

- `go build ./...` and `go vet ./...` clean.
- `gofmt -l` reports nothing for files you touched.
- Your `TestAccFoo*` tests pass against 7.0.26.
- No resources left behind on the server.
- Registration lines for `provider.go` reported, not applied.
