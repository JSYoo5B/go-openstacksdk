# Identityproviders

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:identity/v3/create_identity_provider`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.WithCreateField`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.WithCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.WithCreateHeaders`

- Create sends PUT OS-FEDERATION/identity_providers/{id} with only the declared attributes (Python names and Body keys, is_enabled aliasing enabled) under identity_provider and without id or name, keeps explicit null, drops unknown attributes, accepts every status 200-399, overlays the response object on the request seed and wraps 4xx/5xx with Create operation context.
  Tests: `identity/v3/identityproviders/api_test.go:TestIdentityProviderCreatePutsMemberWithoutID`, `identity/v3/identityproviders/api_test.go:TestIdentityProviderRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Python takes the member ID from the id or name attribute and fails before HTTP without one; Go takes it as an explicit argument and ignores id and name fields.
- Python keeps an arbitrary value when two aliases of one Body key (for example is_enabled and enabled) are given; Go rejects conflicting alias values before HTTP.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent (Keystone v3 has no microversion).

### `python:identity/v3/delete_identity_provider`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.WithDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.WithDeleteHeaders`

- Delete sends DELETE OS-FEDERATION/identity_providers/{id}, accepts 200-399, turns a clean 404 into nil by default as ignore_missing=True, returns the 404 with WithDeleteIgnoreMissing(false) and always returns other statuses such as 409 with Delete operation context.
  Tests: `identity/v3/identityproviders/api_test.go:TestIdentityProviderDeleteIgnoresMissingByDefault`, `identity/v3/identityproviders/api_test.go:TestIdentityProviderRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent (Keystone v3 has no microversion).

### `python:identity/v3/find_identity_provider`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.WithFindIgnoreMissing`

- Find sends GET OS-FEDERATION/identity_providers/{name_or_id} first and, after a clean 400, 403 or 404, lists the whole unfiltered collection matching the exact id (name is the id Body key); a missing resource is nil by default or resource.NotFoundError with WithFindIgnoreMissing(false), duplicates fail and other GET failures do not fall back.
  Tests: `identity/v3/identityproviders/api_test.go:TestIdentityProviderFindFallsBackToList`, `identity/v3/identityproviders/api_test.go:TestIdentityProviderRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Python find_* emits deprecation warnings for ignore_missing=True (and find_identity_provider itself is deprecated); Go emits none.
- A duplicate exact match returns resource.AmbiguousError instead of openstack DuplicateResource; a missing resource with ignore_missing false returns resource.NotFoundError instead of NotFoundException.
- Go follows Keystone's links {"next": url} dictionary, which the pinned Python paginator ignores, as the other SDK-owned identity lists do; top-level next, HTTP Link and limit-driven marker continuation match Python.

### `python:identity/v3/get_identity_provider`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.WithGetHeaders`

- Get sends GET OS-FEDERATION/identity_providers/{id}, accepts every status 200-399, decodes the identity_provider envelope or a root object into typed fields plus raw Body/Wire views, fails a non-object JSON body with resource.ResponseError and returns 404 as a native error with Get operation context.
  Tests: `identity/v3/identityproviders/api_test.go:TestIdentityProviderGetDecodesAndRejects`, `identity/v3/identityproviders/api_test.go:TestIdentityProviderRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent (Keystone v3 has no microversion).

### `python:identity/v3/identity_providers`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.API.All`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.WithListOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.WithListID`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.WithListEnabled`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.WithListLimit`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.WithListMarker`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.WithListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.WithListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.WithListHeaders`

- List sends GET OS-FEDERATION/identity_providers with Accept: application/json, Requests-encoded server query id and is_enabled/enabled (client name preferred over server name), filters name, description, domain_id, remote_ids and authorization_ttl locally after the raw row cap, ignores unknown names, follows limit-driven marker continuation until an empty page, honors max_items as a limit hint and paginated=False, and wraps errors with List operation context.
  Tests: `identity/v3/identityproviders/api_test.go:TestIdentityProviderListQueryFiltersAndPaging`, `identity/v3/identityproviders/api_test.go:TestIdentityProviderRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Go follows Keystone's links {"next": url} dictionary, which the pinned Python paginator ignores, as the other SDK-owned identity lists do; top-level next, HTTP Link and limit-driven marker continuation match Python.
- Local Body filters compare JSON values and distinguish booleans from numbers; Python compares type-converted attribute values (True == 1, int/bool conversion).
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent (Keystone v3 has no microversion).
- Go returns a lazy iterator (All collects it); Python returns a generator of Resources.

### `python:identity/v3/update_identity_provider`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.WithUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders.WithUpdateHeaders`

- Update sends PATCH OS-FEDERATION/identity_providers/{id} with only the given declared attributes under identity_provider, never id or name, sends nothing when no declared attribute is given, overlays the response on the seed and wraps errors with Update operation context.
  Tests: `identity/v3/identityproviders/api_test.go:TestIdentityProviderUpdatePatchesDirtyAttributes`, `identity/v3/identityproviders/api_test.go:TestIdentityProviderRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Python update with name= rewrites the id Body key and therefore the request path; Go always targets the ID argument and ignores id and name attributes.
- Python keeps an arbitrary value when two aliases of one Body key (for example is_enabled and enabled) are given; Go rejects conflicting alias values before HTTP.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent (Keystone v3 has no microversion).

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L3022-L3150)
- [`identity_provider.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/identity_provider.py)
