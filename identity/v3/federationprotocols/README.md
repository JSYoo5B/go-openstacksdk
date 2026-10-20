# Federationprotocols

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:identity/v3/create_federation_protocol`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.WithCreateField`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.WithCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.WithCreateHeaders`

- Create sends PUT OS-FEDERATION/identity_providers/{idp_id}/protocols/{id} with both IDs escaped, only mapping_id under protocol and without id, name or idp_id, accepts every status 200-399, overlays the response on the request seed and wraps 409 with Create operation context.
  Tests: `identity/v3/federationprotocols/api_test.go:TestFederationProtocolCreatePutsMemberWithoutID`, `identity/v3/federationprotocols/api_test.go:TestFederationProtocolRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Python takes the member ID from the id or name attribute and fails before HTTP without one; Go takes it as an explicit argument and ignores id and name fields.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent (Keystone v3 has no microversion).

### `python:identity/v3/delete_federation_protocol`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.WithDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.WithDeleteHeaders`

- Delete sends DELETE OS-FEDERATION/identity_providers/{idp_id}/protocols/{id}, ignores a clean 404 by default, returns it with WithDeleteIgnoreMissing(false) and returns 500 with Delete operation context.
  Tests: `identity/v3/federationprotocols/api_test.go:TestFederationProtocolGetUpdateDelete`, `identity/v3/federationprotocols/api_test.go:TestFederationProtocolRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Python lets idp be None when protocol is a FederationProtocol instance and fails with UnboundLocalError for a string protocol without idp; Go always takes the identity provider ID as an argument and sets IdentityProviderID from it.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent (Keystone v3 has no microversion).

### `python:identity/v3/find_federation_protocol`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.WithFindIgnoreMissing`

- Find sends GET OS-FEDERATION/identity_providers/{idp_id}/protocols/{name_or_id}?idp_id={idp_id} as Resource.find forwards the URI attribute to fetch, then after a clean 400, 403 or 404 lists the protocols without query matching the exact id; missing is nil by default or resource.NotFoundError with WithFindIgnoreMissing(false).
  Tests: `identity/v3/federationprotocols/api_test.go:TestFederationProtocolFindSendsIdentityProviderQuery`, `identity/v3/federationprotocols/api_test.go:TestFederationProtocolRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Python find_* emits deprecation warnings for ignore_missing=True (and find_identity_provider itself is deprecated); Go emits none.
- A duplicate exact match returns resource.AmbiguousError instead of openstack DuplicateResource; a missing resource with ignore_missing false returns resource.NotFoundError instead of NotFoundException.
- Go follows Keystone's links {"next": url} dictionary, which the pinned Python paginator ignores, as the other SDK-owned identity lists do; top-level next, HTTP Link and limit-driven marker continuation match Python.

### `python:identity/v3/get_federation_protocol`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.WithGetHeaders`

- Get sends GET OS-FEDERATION/identity_providers/{idp_id}/protocols/{id} without query, decodes the protocol envelope into ID, MappingID and raw views and returns 4xx/5xx with Get operation context.
  Tests: `identity/v3/federationprotocols/api_test.go:TestFederationProtocolGetUpdateDelete`, `identity/v3/federationprotocols/api_test.go:TestFederationProtocolRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Python lets idp be None when protocol is a FederationProtocol instance and fails with UnboundLocalError for a string protocol without idp; Go always takes the identity provider ID as an argument and sets IdentityProviderID from it.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent (Keystone v3 has no microversion).

### `python:identity/v3/federation_protocols`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.API.All`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.WithListOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.WithListID`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.WithListLimit`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.WithListMarker`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.WithListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.WithListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.WithListHeaders`

- List sends GET OS-FEDERATION/identity_providers/{idp_id}/protocols with the identity provider escaped, Accept: application/json and only the id server query, filters name and mapping_id locally, ignores unknown names, sets IdentityProviderID on every row, follows limit-driven marker continuation on the escaped path and rejects a blank identity provider before HTTP.
  Tests: `identity/v3/federationprotocols/api_test.go:TestFederationProtocolListQueryFiltersAndPaging`, `identity/v3/federationprotocols/api_test.go:TestFederationProtocolRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Go follows Keystone's links {"next": url} dictionary, which the pinned Python paginator ignores, as the other SDK-owned identity lists do; top-level next, HTTP Link and limit-driven marker continuation match Python.
- Local Body filters compare JSON values and distinguish booleans from numbers; Python compares type-converted attribute values (True == 1, int/bool conversion).
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent (Keystone v3 has no microversion).
- Go returns a lazy iterator (All collects it); Python returns a generator of Resources.

### `python:identity/v3/update_federation_protocol`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.WithUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols.WithUpdateHeaders`

- Update sends PATCH OS-FEDERATION/identity_providers/{idp_id}/protocols/{id} with only mapping_id under protocol, ignores id, and sends nothing when no declared attribute is given.
  Tests: `identity/v3/federationprotocols/api_test.go:TestFederationProtocolGetUpdateDelete`, `identity/v3/federationprotocols/api_test.go:TestFederationProtocolRejectsInvalidInputBeforeHTTP`.

Python/Go differences:

- Python accepts an ID string or a Resource and returns a mutable Resource; Go takes ID strings, escapes each as exactly one path segment (Python joins them unescaped), rejects blank, dot and control-character IDs before HTTP, and returns an owned typed model whose Body is the declared Body-key view without the Python location attribute.
- Python lets idp be None when protocol is a FederationProtocol instance and fails with UnboundLocalError for a string protocol without idp; Go always takes the identity provider ID as an argument and sets IdentityProviderID from it.
- Python update with name= rewrites the id Body key and therefore the request path; Go always targets the ID argument and ignores id and name attributes.
- Go header options are an explicit extension; SDK-owned auth and content headers are rejected before HTTP. Python base_path, microversion, allow_unknown_params and jmespath_filters keyword controls have no Go equivalent (Keystone v3 has no microversion).

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
conn.identity.create_federation_protocol("corp", id="saml2", mapping_id="corp-mapping")
protocol = conn.identity.find_federation_protocol("corp", "saml2")
conn.identity.delete_federation_protocol("corp", "saml2")
```

```go
api := federationprotocols.New(identityClient)
_, err := api.Create(ctx, "corp", "saml2",
    federationprotocols.WithCreateField("mapping_id", "corp-mapping"))
if err != nil { return err }
protocol, err := api.Find(ctx, "corp", "saml2")
if err != nil { return err }
err = api.Delete(ctx, "corp", protocol.ID)
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2725-L2912)
- [`federation_protocol.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/federation_protocol.py)
