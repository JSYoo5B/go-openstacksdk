# Portpairs

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/create_sfc_port_pair`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.WithCreateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.WithCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.WithCreateHeaders`

- Create sends POST /v2.0/sfc/port_pairs with {"port_pair": attrs} holding every given declared attribute (Python attribute names or Body keys, id included, unknown names dropped), accepts 200-399, decodes the port_pair member or root object over the sent values, and returns 400 and above with Create operation context.
  Tests: `network/v2/extensions/sfc/portpairs/api_test.go:TestPortPairCreatePostsEnvelope`, `network/v2/extensions/sfc/portpairs/api_test.go:TestPortPairRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Python sends no microversion for these networking-sfc resources; Go sends the selected client's microversion as OpenStack-API-Version: network <version> through Gophercloud when one is configured.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent.
- Go JSON-encodes the create body with sorted keys instead of the Python attribute order, ignores attribute-name case folding, and rejects two aliases of one Body key with different values before HTTP where Python keeps an arbitrary one.

### `python:network/v2/get_sfc_port_pair`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.WithGetHeaders`

- Get sends GET /v2.0/sfc/port_pairs/{id}, decodes the port_pair member or root object with the Python Body descriptor conversions and the project_id to tenant_id alias, fails non-object JSON with the accepted response as proof and returns 404 with Get operation context.
  Tests: `network/v2/extensions/sfc/portpairs/api_test.go:TestPortPairGetDecodesDescriptorsAndRejects`, `network/v2/extensions/sfc/portpairs/api_test.go:TestPortPairRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Python sends no microversion for these networking-sfc resources; Go sends the selected client's microversion as OpenStack-API-Version: network <version> through Gophercloud when one is configured.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent.

### `python:network/v2/find_sfc_port_pair`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.WithFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.WithFindQuery`

- Find sends GET /v2.0/sfc/port_pairs/{name_or_id} with every find query name as query first and, after a clean 400, 403 or 404, lists /v2.0/sfc/port_pairs with the query plus name=name_or_id (unless the query names one) matching exact ID or name; a missing resource is nil by default or resource.NotFoundError with WithFindIgnoreMissing(false), duplicates return resource.AmbiguousError and other GET failures do not fall back.
  Tests: `network/v2/extensions/sfc/portpairs/api_test.go:TestPortPairFindFallsBackToNameList`, `network/v2/extensions/sfc/portpairs/api_test.go:TestPortPairRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Python sends no microversion for these networking-sfc resources; Go sends the selected client's microversion as OpenStack-API-Version: network <version> through Gophercloud when one is configured.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent.
- Python find_* emits deprecation warnings for ignore_missing=True; Go emits none.
- A duplicate exact match returns resource.AmbiguousError instead of openstack DuplicateResource; a missing resource with ignore_missing false returns resource.NotFoundError instead of NotFoundException.
- Go rejects an id find query and Resource control names before HTTP where Python raises TypeError or forwards them, and ignores all_projects as Python's list does.

### `python:network/v2/sfc_port_pairs`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.API.All`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.WithListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.WithListName`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.WithListProjectID`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.WithListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.WithListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.WithListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.WithListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.WithListHeaders`

- List sends GET /v2.0/sfc/port_pairs with Accept: application/json and the QueryParameters names (plus limit and marker) as Requests-encoded server query, filters the other Body attributes locally with Python dict-subset semantics, ignores other names, follows the port_pairs_links next link (including a /v2.0-prefixed relative link) and limit-driven marker continuation until an empty page, honors max_items and paginated=False, and returns page failures with List operation context.
  Tests: `network/v2/extensions/sfc/portpairs/api_test.go:TestPortPairListQueryFiltersAndPaging`, `network/v2/extensions/sfc/portpairs/api_test.go:TestPortPairRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Python sends no microversion for these networking-sfc resources; Go sends the selected client's microversion as OpenStack-API-Version: network <version> through Gophercloud when one is configured.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent.
- Python returns a generator of Resources; Go returns a lazy iter.Seq2 (and All) whose local Body filters compare JSON values exactly, so booleans and numbers stay distinct unlike Python True == 1.

### `python:network/v2/update_sfc_port_pair`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.WithUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.WithUpdateHeaders`

- Update sends PUT /v2.0/sfc/port_pairs/{id} with {"port_pair": attrs} holding only the given declared attributes except id, sends nothing when no declared attribute is given, decodes the response like Get and returns 409 with Update operation context.
  Tests: `network/v2/extensions/sfc/portpairs/api_test.go:TestPortPairUpdatePutsDirtyAttributes`, `network/v2/extensions/sfc/portpairs/api_test.go:TestPortPairRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Python sends no microversion for these networking-sfc resources; Go sends the selected client's microversion as OpenStack-API-Version: network <version> through Gophercloud when one is configured.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent.
- Python update with an id attribute raises TypeError for a duplicate id keyword; Go ignores an id field and uses the ID argument.

### `python:network/v2/delete_sfc_port_pair`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.WithDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs.WithDeleteHeaders`

- Delete sends DELETE /v2.0/sfc/port_pairs/{id}, accepts 200-399, turns a clean 404 into nil by default as ignore_missing=True, returns the 404 with WithDeleteIgnoreMissing(false) and always returns other statuses such as 409 with Delete operation context.
  Tests: `network/v2/extensions/sfc/portpairs/api_test.go:TestPortPairDeleteIgnoresMissingByDefault`, `network/v2/extensions/sfc/portpairs/api_test.go:TestPortPairRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Python sends no microversion for these networking-sfc resources; Go sends the selected client's microversion as OpenStack-API-Version: network <version> through Gophercloud when one is configured.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent.

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9500-L9629)
- [`sfc_port_pair.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/sfc_port_pair.py)
