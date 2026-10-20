# Portpairgroups

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/create_sfc_port_pair_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.WithCreateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.WithCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.WithCreateHeaders`

- Create sends POST /v2.0/sfc/port_pair_groups with {"port_pair_group": attrs} holding every given declared attribute (Python attribute names or Body keys, id included, unknown names dropped), accepts 200-399, decodes the port_pair_group member or root object over the sent values, and returns 400 and above with Create operation context.
  Tests: `network/v2/extensions/sfc/portpairgroups/api_test.go:TestPortPairGroupCreatePostsEnvelope`, `network/v2/extensions/sfc/portpairgroups/api_test.go:TestPortPairGroupRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Python sends no microversion for these networking-sfc resources; Go sends the selected client's microversion as OpenStack-API-Version: network <version> through Gophercloud when one is configured.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent.
- Go JSON-encodes the create body with sorted keys instead of the Python attribute order, ignores attribute-name case folding, and rejects two aliases of one Body key with different values before HTTP where Python keeps an arbitrary one.

### `python:network/v2/get_sfc_port_pair_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.WithGetHeaders`

- Get sends GET /v2.0/sfc/port_pair_groups/{id}, decodes the port_pair_group member or root object with the Python Body descriptor conversions and the project_id to tenant_id alias, fails non-object JSON with the accepted response as proof and returns 404 with Get operation context.
  Tests: `network/v2/extensions/sfc/portpairgroups/api_test.go:TestPortPairGroupGetDecodesDescriptorsAndRejects`, `network/v2/extensions/sfc/portpairgroups/api_test.go:TestPortPairGroupRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Python sends no microversion for these networking-sfc resources; Go sends the selected client's microversion as OpenStack-API-Version: network <version> through Gophercloud when one is configured.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent.

### `python:network/v2/find_sfc_port_pair_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.WithFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.WithFindQuery`

- Find sends GET /v2.0/sfc/port_pair_groups/{name_or_id} with every find query name as query first and, after a clean 400, 403 or 404, lists /v2.0/sfc/port_pair_groups with the query plus name=name_or_id (unless the query names one) matching exact ID or name; a missing resource is nil by default or resource.NotFoundError with WithFindIgnoreMissing(false), duplicates return resource.AmbiguousError and other GET failures do not fall back.
  Tests: `network/v2/extensions/sfc/portpairgroups/api_test.go:TestPortPairGroupFindFallsBackToNameList`, `network/v2/extensions/sfc/portpairgroups/api_test.go:TestPortPairGroupRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Python sends no microversion for these networking-sfc resources; Go sends the selected client's microversion as OpenStack-API-Version: network <version> through Gophercloud when one is configured.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent.
- Python find_* emits deprecation warnings for ignore_missing=True; Go emits none.
- A duplicate exact match returns resource.AmbiguousError instead of openstack DuplicateResource; a missing resource with ignore_missing false returns resource.NotFoundError instead of NotFoundException.
- Go rejects an id find query and Resource control names before HTTP where Python raises TypeError or forwards them, and ignores all_projects as Python's list does.

### `python:network/v2/sfc_port_pair_groups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.API.All`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.WithListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.WithListName`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.WithListProjectID`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.WithListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.WithListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.WithListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.WithListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.WithListHeaders`

- List sends GET /v2.0/sfc/port_pair_groups with Accept: application/json and the QueryParameters names (plus limit and marker) as Requests-encoded server query, filters the other Body attributes locally with Python dict-subset semantics, ignores other names, follows the port_pair_groups_links next link (including a /v2.0-prefixed relative link) and limit-driven marker continuation until an empty page, honors max_items and paginated=False, and returns page failures with List operation context.
  Tests: `network/v2/extensions/sfc/portpairgroups/api_test.go:TestPortPairGroupListQueryFiltersAndPaging`, `network/v2/extensions/sfc/portpairgroups/api_test.go:TestPortPairGroupRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Python sends no microversion for these networking-sfc resources; Go sends the selected client's microversion as OpenStack-API-Version: network <version> through Gophercloud when one is configured.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent.
- Python returns a generator of Resources; Go returns a lazy iter.Seq2 (and All) whose local Body filters compare JSON values exactly, so booleans and numbers stay distinct unlike Python True == 1.

### `python:network/v2/update_sfc_port_pair_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.WithUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.WithUpdateHeaders`

- Update sends PUT /v2.0/sfc/port_pair_groups/{id} with {"port_pair_group": attrs} holding only the given declared attributes except id, sends nothing when no declared attribute is given, decodes the response like Get and returns 409 with Update operation context.
  Tests: `network/v2/extensions/sfc/portpairgroups/api_test.go:TestPortPairGroupUpdatePutsDirtyAttributes`, `network/v2/extensions/sfc/portpairgroups/api_test.go:TestPortPairGroupRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Python sends no microversion for these networking-sfc resources; Go sends the selected client's microversion as OpenStack-API-Version: network <version> through Gophercloud when one is configured.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent.
- Python update with an id attribute raises TypeError for a duplicate id keyword; Go ignores an id field and uses the ID argument.

### `python:network/v2/delete_sfc_port_pair_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.WithDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups.WithDeleteHeaders`

- Delete sends DELETE /v2.0/sfc/port_pair_groups/{id}, accepts 200-399, turns a clean 404 into nil by default as ignore_missing=True, returns the 404 with WithDeleteIgnoreMissing(false) and always returns other statuses such as 409 with Delete operation context.
  Tests: `network/v2/extensions/sfc/portpairgroups/api_test.go:TestPortPairGroupDeleteIgnoresMissingByDefault`, `network/v2/extensions/sfc/portpairgroups/api_test.go:TestPortPairGroupRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Python sends no microversion for these networking-sfc resources; Go sends the selected client's microversion as OpenStack-API-Version: network <version> through Gophercloud when one is configured.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
ppg = conn.network.create_sfc_port_pair_group(
    name="fw-group", port_pairs=[pp.id],
    port_pair_group_parameters={"lb_fields": ["ip_src"]})
conn.network.update_sfc_port_pair_group(ppg, port_pairs=[pp.id, pp2.id])
conn.network.delete_sfc_port_pair_group(ppg)
```

```go
api := portpairgroups.New(networkClient)
ppg, err := api.Create(ctx,
    portpairgroups.WithCreateField("name", "fw-group"),
    portpairgroups.WithCreateField("port_pairs", []string{pp.ID}),
    portpairgroups.WithCreateField("port_pair_group_parameters", map[string]any{"lb_fields": []string{"ip_src"}}))
if err != nil { return err }
_, err = api.Update(ctx, ppg.ID, portpairgroups.WithUpdateField("port_pairs", []string{pp.ID, pp2.ID}))
if err != nil { return err }
err = api.Delete(ctx, ppg.ID)
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9630-L9767)
- [`sfc_port_pair_group.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/sfc_port_pair_group.py)
