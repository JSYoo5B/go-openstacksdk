# Serviceproviders

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:identity/v3/create_service_provider`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.WithCreateField`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.WithCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.WithCreateHeaders`

- Create sends PUT OS-FEDERATION/service_providers/{id} with only the declared attributes (Python names and Body keys, is_enabled aliasing enabled) under service_provider and without id or name, keeps explicit null, drops unknown attributes, accepts every status 200-399, overlays the response object on the request seed and wraps 4xx/5xx with Create operation context.
  Tests: `identity/v3/serviceproviders/api_test.go:TestServiceProviderCreatePutsMemberWithoutID`, `identity/v3/serviceproviders/api_test.go:TestServiceProviderRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Python takes the member ID from the id or name attribute and fails before HTTP without one; Go takes it as an explicit argument and ignores id and name fields.
- Python keeps an arbitrary value when two aliases of one Body key (for example is_enabled and enabled) are given; Go rejects conflicting alias values before HTTP.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent (Keystone v3 has no microversion).

### `python:identity/v3/delete_service_provider`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.WithDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.WithDeleteHeaders`

- Delete sends DELETE OS-FEDERATION/service_providers/{id}, accepts 200-399, turns a clean 404 into nil by default as ignore_missing=True, returns the 404 with WithDeleteIgnoreMissing(false) and always returns other statuses such as 409 with Delete operation context.
  Tests: `identity/v3/serviceproviders/api_test.go:TestServiceProviderDeleteIgnoresMissingByDefault`, `identity/v3/serviceproviders/api_test.go:TestServiceProviderRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent (Keystone v3 has no microversion).

### `python:identity/v3/find_service_provider`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.WithFindIgnoreMissing`

- Find sends GET OS-FEDERATION/service_providers/{name_or_id} first and, after a clean 400, 403 or 404, lists the whole unfiltered collection matching the exact id (name is the id Body key); a missing resource is nil by default or resource.NotFoundError with WithFindIgnoreMissing(false), duplicates fail and other GET failures do not fall back.
  Tests: `identity/v3/serviceproviders/api_test.go:TestServiceProviderFindFallsBackToList`, `identity/v3/serviceproviders/api_test.go:TestServiceProviderRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Python find_* emits deprecation warnings for ignore_missing=True (and find_identity_provider itself is deprecated); Go emits none.
- A duplicate exact match returns resource.AmbiguousError instead of openstack DuplicateResource; a missing resource with ignore_missing false returns resource.NotFoundError instead of NotFoundException.
- Go follows Keystone's links {"next": url} dictionary, which the pinned Python paginator ignores, as the other SDK-owned identity lists do; top-level next, HTTP Link and limit-driven marker continuation match Python.

### `python:identity/v3/get_service_provider`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.WithGetHeaders`

- Get sends GET OS-FEDERATION/service_providers/{id}, accepts every status 200-399, decodes the service_provider envelope or a root object into typed fields plus raw Body/Wire views, fails a non-object JSON body with resource.ResponseError and returns 404 as a native error with Get operation context.
  Tests: `identity/v3/serviceproviders/api_test.go:TestServiceProviderGetDecodesAndRejects`, `identity/v3/serviceproviders/api_test.go:TestServiceProviderRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent (Keystone v3 has no microversion).

### `python:identity/v3/service_providers`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.API.All`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.WithListOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.WithListID`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.WithListEnabled`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.WithListLimit`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.WithListMarker`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.WithListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.WithListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.WithListHeaders`

- List sends GET OS-FEDERATION/service_providers with Accept: application/json, Requests-encoded server query id and is_enabled/enabled (client name preferred over server name), filters name, auth_url, description, relay_state_prefix and sp_url locally after the raw row cap, ignores unknown names, follows limit-driven marker continuation until an empty page, honors max_items as a limit hint and paginated=False, and wraps errors with List operation context.
  Tests: `identity/v3/serviceproviders/api_test.go:TestServiceProviderListQueryFiltersAndPaging`, `identity/v3/serviceproviders/api_test.go:TestServiceProviderRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Go follows Keystone's links {"next": url} dictionary, which the pinned Python paginator ignores, as the other SDK-owned identity lists do; top-level next, HTTP Link and limit-driven marker continuation match Python.
- Local Body filters compare JSON values and distinguish booleans from numbers; Python compares type-converted attribute values (True == 1, int/bool conversion).
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent (Keystone v3 has no microversion).
- Go returns a lazy iterator (All collects it); Python returns a generator of Resources.

### `python:identity/v3/update_service_provider`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.WithUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders.WithUpdateHeaders`

- Update sends PATCH OS-FEDERATION/service_providers/{id} with only the given declared attributes under service_provider, never id or name, sends nothing when no declared attribute is given, overlays the response on the seed and wraps errors with Update operation context.
  Tests: `identity/v3/serviceproviders/api_test.go:TestServiceProviderUpdatePatchesDirtyAttributes`, `identity/v3/serviceproviders/api_test.go:TestServiceProviderRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Python update with name= rewrites the id Body key and therefore the request path; Go always targets the ID argument and ignores id and name attributes.
- Python keeps an arbitrary value when two aliases of one Body key (for example is_enabled and enabled) are given; Go rejects conflicting alias values before HTTP.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent (Keystone v3 has no microversion).

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L3218-L3340)
- [`service_provider.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/service_provider.py)
