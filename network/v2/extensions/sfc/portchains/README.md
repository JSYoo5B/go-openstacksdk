# Portchains

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/create_sfc_port_chain`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.WithCreateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.WithCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.WithCreateHeaders`

- Create sends POST /v2.0/sfc/port_chains with {"port_chain": attrs} holding every given declared attribute (Python attribute names or Body keys, id included, unknown names dropped), accepts 200-399, decodes the port_chain member or root object over the sent values, and returns 400 and above with Create operation context.
  Tests: `network/v2/extensions/sfc/portchains/api_test.go:TestPortChainCreatePostsEnvelope`, `network/v2/extensions/sfc/portchains/api_test.go:TestPortChainRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Python sends no microversion for these networking-sfc resources; Go sends the selected client's microversion as OpenStack-API-Version: network <version> through Gophercloud when one is configured.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent.
- Go JSON-encodes the create body with sorted keys instead of the Python attribute order, ignores attribute-name case folding, and rejects two aliases of one Body key with different values before HTTP where Python keeps an arbitrary one.

### `python:network/v2/get_sfc_port_chain`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.WithGetHeaders`

- Get sends GET /v2.0/sfc/port_chains/{id}, decodes the port_chain member or root object with the Python Body descriptor conversions and the project_id to tenant_id alias, fails non-object JSON with the accepted response as proof and returns 404 with Get operation context.
  Tests: `network/v2/extensions/sfc/portchains/api_test.go:TestPortChainGetDecodesDescriptorsAndRejects`, `network/v2/extensions/sfc/portchains/api_test.go:TestPortChainRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Python sends no microversion for these networking-sfc resources; Go sends the selected client's microversion as OpenStack-API-Version: network <version> through Gophercloud when one is configured.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent.

### `python:network/v2/find_sfc_port_chain`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.WithFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.WithFindQuery`

- Find sends GET /v2.0/sfc/port_chains/{name_or_id} with every find query name as query first and, after a clean 400, 403 or 404, lists /v2.0/sfc/port_chains with the query plus name=name_or_id (unless the query names one) matching exact ID or name; a missing resource is nil by default or resource.NotFoundError with WithFindIgnoreMissing(false), duplicates return resource.AmbiguousError and other GET failures do not fall back.
  Tests: `network/v2/extensions/sfc/portchains/api_test.go:TestPortChainFindFallsBackToNameList`, `network/v2/extensions/sfc/portchains/api_test.go:TestPortChainRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Python sends no microversion for these networking-sfc resources; Go sends the selected client's microversion as OpenStack-API-Version: network <version> through Gophercloud when one is configured.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent.
- Python find_* emits deprecation warnings for ignore_missing=True; Go emits none.
- A duplicate exact match returns resource.AmbiguousError instead of openstack DuplicateResource; a missing resource with ignore_missing false returns resource.NotFoundError instead of NotFoundException.
- Go rejects an id find query and Resource control names before HTTP where Python raises TypeError or forwards them, and ignores all_projects as Python's list does.

### `python:network/v2/sfc_port_chains`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.API.All`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.WithListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.WithListName`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.WithListProjectID`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.WithListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.WithListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.WithListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.WithListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.WithListHeaders`

- List sends GET /v2.0/sfc/port_chains with Accept: application/json and the QueryParameters names (plus limit and marker) as Requests-encoded server query, filters the other Body attributes locally with Python dict-subset semantics, ignores other names, follows the port_chains_links next link (including a /v2.0-prefixed relative link) and limit-driven marker continuation until an empty page, honors max_items and paginated=False, and returns page failures with List operation context.
  Tests: `network/v2/extensions/sfc/portchains/api_test.go:TestPortChainListQueryFiltersAndPaging`, `network/v2/extensions/sfc/portchains/api_test.go:TestPortChainRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Python sends no microversion for these networking-sfc resources; Go sends the selected client's microversion as OpenStack-API-Version: network <version> through Gophercloud when one is configured.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent.
- Python returns a generator of Resources; Go returns a lazy iter.Seq2 (and All) whose local Body filters compare JSON values exactly, so booleans and numbers stay distinct unlike Python True == 1.

### `python:network/v2/update_sfc_port_chain`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.WithUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.WithUpdateHeaders`

- Update sends PUT /v2.0/sfc/port_chains/{id} with {"port_chain": attrs} holding only the given declared attributes except id, sends nothing when no declared attribute is given, decodes the response like Get and returns 409 with Update operation context.
  Tests: `network/v2/extensions/sfc/portchains/api_test.go:TestPortChainUpdatePutsDirtyAttributes`, `network/v2/extensions/sfc/portchains/api_test.go:TestPortChainRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Python sends no microversion for these networking-sfc resources; Go sends the selected client's microversion as OpenStack-API-Version: network <version> through Gophercloud when one is configured.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent.
- Python update with an id attribute raises TypeError for a duplicate id keyword; Go ignores an id field and uses the ID argument.

### `python:network/v2/delete_sfc_port_chain`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.WithDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains.WithDeleteHeaders`

- Delete sends DELETE /v2.0/sfc/port_chains/{id}, accepts 200-399, turns a clean 404 into nil by default as ignore_missing=True, returns the 404 with WithDeleteIgnoreMissing(false) and always returns other statuses such as 409 with Delete operation context.
  Tests: `network/v2/extensions/sfc/portchains/api_test.go:TestPortChainDeleteIgnoresMissingByDefault`, `network/v2/extensions/sfc/portchains/api_test.go:TestPortChainRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Python sends no microversion for these networking-sfc resources; Go sends the selected client's microversion as OpenStack-API-Version: network <version> through Gophercloud when one is configured.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
pc = conn.network.create_sfc_port_chain(
    name="web-chain", port_pair_groups=[ppg.id], flow_classifiers=[fc.id],
    chain_parameters={"correlation": "mpls", "symmetric": False})
conn.network.update_sfc_port_chain(pc, flow_classifiers=[fc.id, fc2.id])
conn.network.delete_sfc_port_chain(pc)
```

```go
api := portchains.New(networkClient)
pc, err := api.Create(ctx,
    portchains.WithCreateField("name", "web-chain"),
    portchains.WithCreateField("port_pair_groups", []string{ppg.ID}),
    portchains.WithCreateField("flow_classifiers", []string{fc.ID}),
    portchains.WithCreateField("chain_parameters", map[string]any{"correlation": "mpls", "symmetric": false}))
if err != nil { return err }
_, err = api.Update(ctx, pc.ID, portchains.WithUpdateField("flow_classifiers", []string{fc.ID, fc2.ID}))
if err != nil { return err }
err = api.Delete(ctx, pc.ID)
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9367-L9499)
- [`sfc_port_chain.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/sfc_port_chain.py)
