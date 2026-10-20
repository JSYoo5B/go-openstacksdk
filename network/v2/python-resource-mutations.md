# Python Resource Mutations

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/create_address_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.API.CreateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithAttribute`

- POST sends the pinned singular resource envelope with only supplied Body attributes; omitted attrs produce an empty object, no native required-field check or synthesized fields. Canonical Body names map to wire names, explicit false/zero/null/empty values survive, and unknown/query-only attrs remain in Body. Response native model comes from the server without request ID seeding.
  Tests: `network/v2/extensions/security/addressgroups/python_mutations_contracts_test.go:TestPythonAddressGroupCreatePreservesOmissionPresenceAliasesAndExtensions`.
- Options snapshot JSON values, are reusable, and later wire-key options override earlier ones. Invalid serialization/options and Resource/Proxy controls fail before HTTP. SDK HTTP success codes are 200..399; native typed decode errors, HTTP status/body/headers and preflight/callback/post-response custom context causes remain observable.
  Tests: `network/v2/extensions/security/addressgroups/python_mutations_contracts_test.go:TestPythonAddressGroupMutationOptionSnapshotsAndPreflight`, `network/v2/extensions/security/addressgroups/python_mutations_contracts_test.go:TestPythonAddressGroupCreateStatusDecodeAndContextCauses`.

Python/Go differences:

- The callable Body mutation profile uses functional WithAttribute options and native Go models; existing generated native Create is unchanged.
- Success is the explicit SDK HTTP 200..399 profile. Native typed decoding remains stricter than Source Resource._translate_response, which can keep its request view on empty or invalid JSON success bodies. Returned unknown fields, Python descriptor access/conversion and request attributes absent from the response are not synthesized.
- Resource/Proxy control attributes base_path, microversion, connection, _synchronized, resource_type, __conflicting_attrs and location are rejected in this Body API; configured Gophercloud client session policy is retained.
- Full inherited Resource/session behavior remains tracked separately in SDK-R1/C1/S1: Full mutable Python Resource return identity/state and request-attribute fallback for partial/empty success responses remain separate from the callable native Go profile.
- Full inherited Resource/session behavior remains tracked separately in SDK-R1/C1/S1: Proxy/Resource control attributes, computed metadata/location, dynamic microversion and complete Python adapter/session/exception behavior remain separate.
- Native Gophercloud skips HTTP204 JSON decoding; this explicit native model profile returns nil,nil rather than constructing an observed response model from requested attributes. Other bodyless/native decoder behavior remains distinct from Python seeded mutable Resource state.

### `python:network/v2/update_address_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.API.UpdateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithAttribute`

- A literal-ID update with no attributes sends no HTTP and returns a fresh native model containing the requested ID; an id attribute is rejected before HTTP.
  Tests: `network/v2/extensions/security/addressgroups/python_mutations_contracts_test.go:TestPythonAddressGroupUpdateEmptyAttributesReturnsIDWithoutHTTP`.
- Specified empty strings, null and false, canonical Body aliases and unknown extension fields are sent in one PUT resource envelope without lookup. The returned native model comes from the response rather than merged request attributes. HTTP status/decode/cancellation causes are preserved under the 200..399 profile.
  Tests: `network/v2/extensions/security/addressgroups/python_mutations_contracts_test.go:TestPythonAddressGroupUpdatePreservesChangedBodyAndResponseModel`, `network/v2/extensions/security/addressgroups/python_mutations_contracts_test.go:TestPythonAddressGroupUpdateStatusCancellationAndDecodePolicy`.

Python/Go differences:

- Literal IDs and functional Body attribute options replace Python string-or-Resource inputs. No attributes reproduces the ID-only Source.new/commit no-op; specified attributes are committed without a stored mutable Resource dirty/cache comparison. Existing native Update remains unchanged.
- Success is the SDK HTTP 200..399 profile. The returned native Go model is server-decoded, does not retain unknown or response-omitted request attributes, and uses native decode errors.
- Resource/Proxy control attributes are explicitly rejected in the Body option surface; update id belongs to the method argument and is rejected as an attribute.
- Full inherited Resource/session behavior remains tracked separately in SDK-R1/C1/S1: Mutable Resource instance input, same-object return, prior cached value/dirty comparisons and request-attribute fallback for partial or bodyless success remain separate.
- Full inherited Resource/session behavior remains tracked separately in SDK-R1/C1/S1: Proxy/Resource control attributes, computed metadata, dynamic microversion and complete Python adapter/session/exception behavior remain separate.
- Native Gophercloud skips HTTP204 JSON decoding; this explicit native model profile returns nil,nil rather than constructing an observed response model from requested attributes. Other bodyless/native decoder behavior remains distinct from Python seeded mutable Resource state.

### `python:network/v2/delete_address_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.API.DeleteWithOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithDeleteIgnoreMissing`

- A literal ID is deleted directly at the pinned resource path with no lookup/query/body. Omitted or nil IgnoreMissing defaults to true; explicit false preserves a 404. Later options override earlier options, constructor pointer snapshots survive mutation and options are reusable. HTTP 200..399 successes do not decode an opaque body.
  Tests: `network/v2/extensions/security/addressgroups/python_mutations_contracts_test.go:TestPythonAddressGroupDeleteDefaultsStrictOptionsAndSourceStatuses`.
- Physical 404 response read failures are not treated as missing success. Native HTTP status/body/headers, invalid option rejection before HTTP and pre/post-response custom context causes remain observable.
  Tests: `network/v2/extensions/security/addressgroups/python_mutations_contracts_test.go:TestPythonAddressGroupDeleteDefaultsStrictOptionsAndSourceStatuses`, `network/v2/extensions/security/addressgroups/python_mutations_contracts_test.go:TestPythonAddressGroupDeletePreservesTerminalFailuresAndPreflight`.

Python/Go differences:

- DeleteOpts exposes only IgnoreMissing because these five pinned Source declarations have no if_revision parameter. It adapts to the already-tested shared request.DeleteNeutron helper; ordinary native Delete/Remove keep native status behavior.
- A string literal ID and functional options replace Python string-or-Resource inputs. The SDK success status profile is HTTP 200..399; terminal wire/transport/cancellation errors alongside a physical 404 remain errors.
- Full inherited Resource/session behavior remains tracked separately in SDK-R1/C1/S1: Mutable Resource instance input/state and complete Python adapter/session/dynamic microversion/exception behavior remain separate from the literal-ID callable Go profile.

### `python:network/v2/create_address_scope`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/addressscopes.API.CreateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/addressscopes.WithAttribute`

- POST sends the pinned singular resource envelope with only supplied Body attributes; omitted attrs produce an empty object, no native required-field check or synthesized fields. Canonical Body names map to wire names, explicit false/zero/null/empty values survive, and unknown/query-only attrs remain in Body. Response native model comes from the server without request ID seeding.
  Tests: `network/v2/extensions/layer3/addressscopes/python_mutations_contracts_test.go:TestPythonAddressScopeCreatePreservesOmissionPresenceAliasesAndExtensions`.
- Options snapshot JSON values, are reusable, and later wire-key options override earlier ones. Invalid serialization/options and Resource/Proxy controls fail before HTTP. SDK HTTP success codes are 200..399; native typed decode errors, HTTP status/body/headers and preflight/callback/post-response custom context causes remain observable.
  Tests: `network/v2/extensions/layer3/addressscopes/python_mutations_contracts_test.go:TestPythonAddressScopeMutationOptionSnapshotsAndPreflight`, `network/v2/extensions/layer3/addressscopes/python_mutations_contracts_test.go:TestPythonAddressScopeCreateStatusDecodeAndContextCauses`.

Python/Go differences:

- The callable Body mutation profile uses functional WithAttribute options and native Go models; existing generated native Create is unchanged.
- Success is the explicit SDK HTTP 200..399 profile. Native typed decoding remains stricter than Source Resource._translate_response, which can keep its request view on empty or invalid JSON success bodies. Returned unknown fields, Python descriptor access/conversion and request attributes absent from the response are not synthesized.
- Resource/Proxy control attributes base_path, microversion, connection, _synchronized, resource_type, __conflicting_attrs and location are rejected in this Body API; configured Gophercloud client session policy is retained.
- Full inherited Resource/session behavior remains tracked separately in SDK-R1/C1/S1: Full mutable Python Resource return identity/state and request-attribute fallback for partial/empty success responses remain separate from the callable native Go profile.
- Full inherited Resource/session behavior remains tracked separately in SDK-R1/C1/S1: Proxy/Resource control attributes, computed metadata/location, dynamic microversion and complete Python adapter/session/exception behavior remain separate.
- Native Gophercloud skips HTTP204 JSON decoding; this explicit native model profile returns nil,nil rather than constructing an observed response model from requested attributes. Other bodyless/native decoder behavior remains distinct from Python seeded mutable Resource state.

### `python:network/v2/update_address_scope`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/addressscopes.API.UpdateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/addressscopes.WithAttribute`

- A literal-ID update with no attributes sends no HTTP and returns a fresh native model containing the requested ID; an id attribute is rejected before HTTP.
  Tests: `network/v2/extensions/layer3/addressscopes/python_mutations_contracts_test.go:TestPythonAddressScopeUpdateEmptyAttributesReturnsIDWithoutHTTP`.
- Specified empty strings, null and false, canonical Body aliases and unknown extension fields are sent in one PUT resource envelope without lookup. The returned native model comes from the response rather than merged request attributes. HTTP status/decode/cancellation causes are preserved under the 200..399 profile.
  Tests: `network/v2/extensions/layer3/addressscopes/python_mutations_contracts_test.go:TestPythonAddressScopeUpdatePreservesChangedBodyAndResponseModel`, `network/v2/extensions/layer3/addressscopes/python_mutations_contracts_test.go:TestPythonAddressScopeUpdateStatusCancellationAndDecodePolicy`.

Python/Go differences:

- Literal IDs and functional Body attribute options replace Python string-or-Resource inputs. No attributes reproduces the ID-only Source.new/commit no-op; specified attributes are committed without a stored mutable Resource dirty/cache comparison. Existing native Update remains unchanged.
- Success is the SDK HTTP 200..399 profile. The returned native Go model is server-decoded, does not retain unknown or response-omitted request attributes, and uses native decode errors.
- Resource/Proxy control attributes are explicitly rejected in the Body option surface; update id belongs to the method argument and is rejected as an attribute.
- Full inherited Resource/session behavior remains tracked separately in SDK-R1/C1/S1: Mutable Resource instance input, same-object return, prior cached value/dirty comparisons and request-attribute fallback for partial or bodyless success remain separate.
- Full inherited Resource/session behavior remains tracked separately in SDK-R1/C1/S1: Proxy/Resource control attributes, computed metadata, dynamic microversion and complete Python adapter/session/exception behavior remain separate.
- Native Gophercloud skips HTTP204 JSON decoding; this explicit native model profile returns nil,nil rather than constructing an observed response model from requested attributes. Other bodyless/native decoder behavior remains distinct from Python seeded mutable Resource state.

### `python:network/v2/delete_address_scope`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/addressscopes.API.DeleteWithOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/addressscopes.WithDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/addressscopes.WithDeleteIgnoreMissing`

- A literal ID is deleted directly at the pinned resource path with no lookup/query/body. Omitted or nil IgnoreMissing defaults to true; explicit false preserves a 404. Later options override earlier options, constructor pointer snapshots survive mutation and options are reusable. HTTP 200..399 successes do not decode an opaque body.
  Tests: `network/v2/extensions/layer3/addressscopes/python_mutations_contracts_test.go:TestPythonAddressScopeDeleteDefaultsStrictOptionsAndSourceStatuses`.
- Physical 404 response read failures are not treated as missing success. Native HTTP status/body/headers, invalid option rejection before HTTP and pre/post-response custom context causes remain observable.
  Tests: `network/v2/extensions/layer3/addressscopes/python_mutations_contracts_test.go:TestPythonAddressScopeDeleteDefaultsStrictOptionsAndSourceStatuses`, `network/v2/extensions/layer3/addressscopes/python_mutations_contracts_test.go:TestPythonAddressScopeDeletePreservesTerminalFailuresAndPreflight`.

Python/Go differences:

- DeleteOpts exposes only IgnoreMissing because these five pinned Source declarations have no if_revision parameter. It adapts to the already-tested shared request.DeleteNeutron helper; ordinary native Delete/Remove keep native status behavior.
- A string literal ID and functional options replace Python string-or-Resource inputs. The SDK success status profile is HTTP 200..399; terminal wire/transport/cancellation errors alongside a physical 404 remain errors.
- Full inherited Resource/session behavior remains tracked separately in SDK-R1/C1/S1: Mutable Resource instance input/state and complete Python adapter/session/dynamic microversion/exception behavior remain separate from the literal-ID callable Go profile.

### `python:network/v2/create_subnet_pool`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.API.CreateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithAttribute`

- POST sends the pinned singular resource envelope with only supplied Body attributes; omitted attrs produce an empty object, no native required-field check or synthesized fields. Canonical Body names map to wire names, explicit false/zero/null/empty values survive, and unknown/query-only attrs remain in Body. Response native model comes from the server without request ID seeding.
  Tests: `network/v2/extensions/subnetpools/python_mutations_contracts_test.go:TestPythonSubnetPoolCreatePreservesOmissionPresenceAliasesAndExtensions`.
- Options snapshot JSON values, are reusable, and later wire-key options override earlier ones. Invalid serialization/options and Resource/Proxy controls fail before HTTP. SDK HTTP success codes are 200..399; native typed decode errors, HTTP status/body/headers and preflight/callback/post-response custom context causes remain observable.
  Tests: `network/v2/extensions/subnetpools/python_mutations_contracts_test.go:TestPythonSubnetPoolMutationOptionSnapshotsAndPreflight`, `network/v2/extensions/subnetpools/python_mutations_contracts_test.go:TestPythonSubnetPoolCreateStatusDecodeAndContextCauses`.

Python/Go differences:

- The callable Body mutation profile uses functional WithAttribute options and native Go models; existing generated native Create is unchanged.
- Success is the explicit SDK HTTP 200..399 profile. Native typed decoding remains stricter than Source Resource._translate_response, which can keep its request view on empty or invalid JSON success bodies. Returned unknown fields, Python descriptor access/conversion and request attributes absent from the response are not synthesized.
- Resource/Proxy control attributes base_path, microversion, connection, _synchronized, resource_type, __conflicting_attrs and location are rejected in this Body API; configured Gophercloud client session policy is retained.
- Native SubnetPool creation decode still requires default_prefixlen/min_prefixlen/max_prefixlen and supports their numeric strings; sparse selected-read normalization does not apply to creation.
- Full inherited Resource/session behavior remains tracked separately in SDK-R1/C1/S1: Full mutable Python Resource return identity/state and request-attribute fallback for partial/empty success responses remain separate from the callable native Go profile.
- Full inherited Resource/session behavior remains tracked separately in SDK-R1/C1/S1: Proxy/Resource control attributes, computed metadata/location, dynamic microversion and complete Python adapter/session/exception behavior remain separate.
- Native Gophercloud skips HTTP204 JSON decoding; this explicit native model profile returns nil,nil rather than constructing an observed response model from requested attributes. Other bodyless/native decoder behavior remains distinct from Python seeded mutable Resource state.

### `python:network/v2/delete_subnet_pool`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.API.DeleteWithOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithDeleteIgnoreMissing`

- A literal ID is deleted directly at the pinned resource path with no lookup/query/body. Omitted or nil IgnoreMissing defaults to true; explicit false preserves a 404. Later options override earlier options, constructor pointer snapshots survive mutation and options are reusable. HTTP 200..399 successes do not decode an opaque body.
  Tests: `network/v2/extensions/subnetpools/python_mutations_contracts_test.go:TestPythonSubnetPoolDeleteDefaultsStrictOptionsAndSourceStatuses`.
- Physical 404 response read failures are not treated as missing success. Native HTTP status/body/headers, invalid option rejection before HTTP and pre/post-response custom context causes remain observable.
  Tests: `network/v2/extensions/subnetpools/python_mutations_contracts_test.go:TestPythonSubnetPoolDeleteDefaultsStrictOptionsAndSourceStatuses`, `network/v2/extensions/subnetpools/python_mutations_contracts_test.go:TestPythonSubnetPoolDeletePreservesTerminalFailuresAndPreflight`.

Python/Go differences:

- DeleteOpts exposes only IgnoreMissing because these five pinned Source declarations have no if_revision parameter. It adapts to the already-tested shared request.DeleteNeutron helper; ordinary native Delete/Remove keep native status behavior.
- A string literal ID and functional options replace Python string-or-Resource inputs. The SDK success status profile is HTTP 200..399; terminal wire/transport/cancellation errors alongside a physical 404 remain errors.
- Full inherited Resource/session behavior remains tracked separately in SDK-R1/C1/S1: Mutable Resource instance input/state and complete Python adapter/session/dynamic microversion/exception behavior remain separate from the literal-ID callable Go profile.

### `python:network/v2/create_trunk`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.API.CreateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithAttribute`

- POST sends the pinned singular resource envelope with only supplied Body attributes; omitted attrs produce an empty object, no native required-field check or synthesized fields. Canonical Body names map to wire names, explicit false/zero/null/empty values survive, and unknown/query-only attrs remain in Body. Response native model comes from the server without request ID seeding.
  Tests: `network/v2/extensions/trunks/python_mutations_contracts_test.go:TestPythonTrunkCreatePreservesOmissionPresenceAliasesAndExtensions`.
- Options snapshot JSON values, are reusable, and later wire-key options override earlier ones. Invalid serialization/options and Resource/Proxy controls fail before HTTP. SDK HTTP success codes are 200..399; native typed decode errors, HTTP status/body/headers and preflight/callback/post-response custom context causes remain observable.
  Tests: `network/v2/extensions/trunks/python_mutations_contracts_test.go:TestPythonTrunkMutationOptionSnapshotsAndPreflight`, `network/v2/extensions/trunks/python_mutations_contracts_test.go:TestPythonTrunkCreateStatusDecodeAndContextCauses`.

Python/Go differences:

- The callable Body mutation profile uses functional WithAttribute options and native Go models; existing generated native Create is unchanged.
- Success is the explicit SDK HTTP 200..399 profile. Native typed decoding remains stricter than Source Resource._translate_response, which can keep its request view on empty or invalid JSON success bodies. Returned unknown fields, Python descriptor access/conversion and request attributes absent from the response are not synthesized.
- Resource/Proxy control attributes base_path, microversion, connection, _synchronized, resource_type, __conflicting_attrs and location are rejected in this Body API; configured Gophercloud client session policy is retained.
- Native Trunk response timestamps retain RFC3339 requirements.
- Full inherited Resource/session behavior remains tracked separately in SDK-R1/C1/S1: Full mutable Python Resource return identity/state and request-attribute fallback for partial/empty success responses remain separate from the callable native Go profile.
- Full inherited Resource/session behavior remains tracked separately in SDK-R1/C1/S1: Proxy/Resource control attributes, computed metadata/location, dynamic microversion and complete Python adapter/session/exception behavior remain separate.
- Native Gophercloud skips HTTP204 JSON decoding; this explicit native model profile returns nil,nil rather than constructing an observed response model from requested attributes. Other bodyless/native decoder behavior remains distinct from Python seeded mutable Resource state.

### `python:network/v2/delete_trunk`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.API.DeleteWithOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithDeleteIgnoreMissing`

- A literal ID is deleted directly at the pinned resource path with no lookup/query/body. Omitted or nil IgnoreMissing defaults to true; explicit false preserves a 404. Later options override earlier options, constructor pointer snapshots survive mutation and options are reusable. HTTP 200..399 successes do not decode an opaque body.
  Tests: `network/v2/extensions/trunks/python_mutations_contracts_test.go:TestPythonTrunkDeleteDefaultsStrictOptionsAndSourceStatuses`.
- Physical 404 response read failures are not treated as missing success. Native HTTP status/body/headers, invalid option rejection before HTTP and pre/post-response custom context causes remain observable.
  Tests: `network/v2/extensions/trunks/python_mutations_contracts_test.go:TestPythonTrunkDeleteDefaultsStrictOptionsAndSourceStatuses`, `network/v2/extensions/trunks/python_mutations_contracts_test.go:TestPythonTrunkDeletePreservesTerminalFailuresAndPreflight`.

Python/Go differences:

- DeleteOpts exposes only IgnoreMissing because these five pinned Source declarations have no if_revision parameter. It adapts to the already-tested shared request.DeleteNeutron helper; ordinary native Delete/Remove keep native status behavior.
- A string literal ID and functional options replace Python string-or-Resource inputs. The SDK success status profile is HTTP 200..399; terminal wire/transport/cancellation errors alongside a physical 404 remain errors.
- Full inherited Resource/session behavior remains tracked separately in SDK-R1/C1/S1: Mutable Resource instance input/state and complete Python adapter/session/dynamic microversion/exception behavior remain separate from the literal-ID callable Go profile.

### `python:network/v2/create_qos_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.CreateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithAttribute`

- POST sends the pinned singular resource envelope with only supplied Body attributes; omitted attrs produce an empty object, no native required-field check or synthesized fields. Canonical Body names map to wire names, explicit false/zero/null/empty values survive, and unknown/query-only attrs remain in Body. Response native model comes from the server without request ID seeding.
  Tests: `network/v2/extensions/qos/policies/python_mutations_contracts_test.go:TestPythonQoSPolicyCreatePreservesOmissionPresenceAliasesAndExtensions`.
- Options snapshot JSON values, are reusable, and later wire-key options override earlier ones. Invalid serialization/options and Resource/Proxy controls fail before HTTP. SDK HTTP success codes are 200..399; native typed decode errors, HTTP status/body/headers and preflight/callback/post-response custom context causes remain observable.
  Tests: `network/v2/extensions/qos/policies/python_mutations_contracts_test.go:TestPythonQoSPolicyMutationOptionSnapshotsAndPreflight`, `network/v2/extensions/qos/policies/python_mutations_contracts_test.go:TestPythonQoSPolicyCreateStatusDecodeAndContextCauses`.

Python/Go differences:

- The callable Body mutation profile uses functional WithAttribute options and native Go models; existing generated native Create is unchanged.
- Success is the explicit SDK HTTP 200..399 profile. Native typed decoding remains stricter than Source Resource._translate_response, which can keep its request view on empty or invalid JSON success bodies. Returned unknown fields, Python descriptor access/conversion and request attributes absent from the response are not synthesized.
- Resource/Proxy control attributes base_path, microversion, connection, _synchronized, resource_type, __conflicting_attrs and location are rejected in this Body API; configured Gophercloud client session policy is retained.
- Full inherited Resource/session behavior remains tracked separately in SDK-R1/C1/S1: Full mutable Python Resource return identity/state and request-attribute fallback for partial/empty success responses remain separate from the callable native Go profile.
- Full inherited Resource/session behavior remains tracked separately in SDK-R1/C1/S1: Proxy/Resource control attributes, computed metadata/location, dynamic microversion and complete Python adapter/session/exception behavior remain separate.
- Native Gophercloud skips HTTP204 JSON decoding; this explicit native model profile returns nil,nil rather than constructing an observed response model from requested attributes. Other bodyless/native decoder behavior remains distinct from Python seeded mutable Resource state.

### `python:network/v2/delete_qos_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.DeleteWithOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithDeleteIgnoreMissing`

- A literal ID is deleted directly at the pinned resource path with no lookup/query/body. Omitted or nil IgnoreMissing defaults to true; explicit false preserves a 404. Later options override earlier options, constructor pointer snapshots survive mutation and options are reusable. HTTP 200..399 successes do not decode an opaque body.
  Tests: `network/v2/extensions/qos/policies/python_mutations_contracts_test.go:TestPythonQoSPolicyDeleteDefaultsStrictOptionsAndSourceStatuses`.
- Physical 404 response read failures are not treated as missing success. Native HTTP status/body/headers, invalid option rejection before HTTP and pre/post-response custom context causes remain observable.
  Tests: `network/v2/extensions/qos/policies/python_mutations_contracts_test.go:TestPythonQoSPolicyDeleteDefaultsStrictOptionsAndSourceStatuses`, `network/v2/extensions/qos/policies/python_mutations_contracts_test.go:TestPythonQoSPolicyDeletePreservesTerminalFailuresAndPreflight`.

Python/Go differences:

- DeleteOpts exposes only IgnoreMissing because these five pinned Source declarations have no if_revision parameter. It adapts to the already-tested shared request.DeleteNeutron helper; ordinary native Delete/Remove keep native status behavior.
- A string literal ID and functional options replace Python string-or-Resource inputs. The SDK success status profile is HTTP 200..399; terminal wire/transport/cancellation errors alongside a physical 404 remain errors.
- Full inherited Resource/session behavior remains tracked separately in SDK-R1/C1/S1: Mutable Resource instance input/state and complete Python adapter/session/dynamic microversion/exception behavior remain separate from the literal-ID callable Go profile.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
ag = conn.network.create_address_group(name="web", addresses=[])
scope = conn.network.create_address_scope(name="scope", ip_version=4, is_shared=False)
pool = conn.network.create_subnet_pool(
    name="pool", prefixes=["10.60.0.0/16"], default_prefix_length=24,
    minimum_prefix_length=16, maximum_prefix_length=28, is_shared=False,
)
trunk = conn.network.create_trunk(port_id=parent_port_id, is_admin_state_up=False, sub_ports=[])
policy = conn.network.create_qos_policy(name="gold", is_shared=False, is_default=False)

unchanged = conn.network.update_address_group(ag.id)
conn.network.update_address_group(ag.id, description="")
conn.network.update_address_scope(scope.id, is_shared=False)
conn.network.delete_address_group(ag.id)
conn.network.delete_address_scope(scope.id, ignore_missing=False)
```

```go
err := api.DeleteWithOptions(ctx, id)
err = api.DeleteWithOptions(ctx, id, addressgroups.WithDeleteIgnoreMissing(false))
strict := false
option := addressgroups.WithDeleteOptions(addressgroups.DeleteOpts{IgnoreMissing: &strict})
```

```go
package main

import (
    "context"
    "fmt"
    "os"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/addressscopes"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(os.Getenv("OS_CLOUD")))
    if err != nil { panic(err) }
    service, err := conn.Network(ctx)
    if err != nil { panic(err) }
    kind := os.Getenv("NEUTRON_MUTATION_KIND")
    deleteID, updateID := os.Getenv("NEUTRON_DELETE_ID"), os.Getenv("NEUTRON_UPDATE_ID")
    switch kind {
    case "address-group":
        api := service.API.SecurityAddressGroups
        value, err := api.CreateWithAttributes(ctx, addressgroups.WithAttribute("name", "web"), addressgroups.WithAttribute("addresses", []string{}))
        if err != nil { panic(err) }
        fmt.Println(value.ID)
        if updateID != "" {
            if _, err := api.UpdateWithAttributes(ctx, updateID, addressgroups.WithAttribute("description", "")); err != nil { panic(err) }
        }
        if deleteID != "" {
            if err := api.DeleteWithOptions(ctx, deleteID); err != nil { panic(err) }
        }
    case "address-scope":
        api := service.API.AddressScopes
        value, err := api.CreateWithAttributes(ctx, addressscopes.WithAttribute("name", "scope"), addressscopes.WithAttribute("ip_version", 4), addressscopes.WithAttribute("is_shared", false))
        if err != nil { panic(err) }
        fmt.Println(value.ID)
        if updateID != "" {
            if _, err := api.UpdateWithAttributes(ctx, updateID, addressscopes.WithAttribute("is_shared", false)); err != nil { panic(err) }
        }
        if deleteID != "" {
            if err := api.DeleteWithOptions(ctx, deleteID, addressscopes.WithDeleteIgnoreMissing(false)); err != nil { panic(err) }
        }
    case "subnet-pool":
        api := service.API.SubnetPools
        value, err := api.CreateWithAttributes(ctx, subnetpools.WithAttribute("name", "pool"), subnetpools.WithAttribute("prefixes", []string{"10.60.0.0/16"}), subnetpools.WithAttribute("default_prefix_length", 24), subnetpools.WithAttribute("minimum_prefix_length", 16), subnetpools.WithAttribute("maximum_prefix_length", 28), subnetpools.WithAttribute("is_shared", false))
        if err != nil { panic(err) }
        fmt.Println(value.ID)
        if deleteID != "" {
            if err := api.DeleteWithOptions(ctx, deleteID); err != nil { panic(err) }
        }
    case "trunk":
        api := service.API.Trunks
        value, err := api.CreateWithAttributes(ctx, trunks.WithAttribute("port_id", os.Getenv("PARENT_PORT_ID")), trunks.WithAttribute("is_admin_state_up", false), trunks.WithAttribute("sub_ports", []map[string]any{}))
        if err != nil { panic(err) }
        fmt.Println(value.ID)
        if deleteID != "" {
            if err := api.DeleteWithOptions(ctx, deleteID); err != nil { panic(err) }
        }
    case "qos-policy":
        api := service.API.QoSPolicies
        value, err := api.CreateWithAttributes(ctx, policies.WithAttribute("name", "gold"), policies.WithAttribute("is_shared", false), policies.WithAttribute("is_default", false))
        if err != nil { panic(err) }
        fmt.Println(value.ID)
        if deleteID != "" {
            if err := api.DeleteWithOptions(ctx, deleteID); err != nil { panic(err) }
        }
    default:
        fmt.Println("set NEUTRON_MUTATION_KIND to address-group, address-scope, subnet-pool, trunk or qos-policy")
    }
}
```

## References

- [typed native CRUD](native-crud.md)
- [Python Read Delete](python-read-delete.md)
