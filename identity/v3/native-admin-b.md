# Native Admin B

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:identity/v3/regions.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithCreateField`

- Create POSTs {"region":{...}} to regions omitting empty fields (empty options send an empty envelope), spreads Extra and WithCreateField keys inside the envelope while rejecting core-tag or Extra key collisions and nil options before HTTP, decodes Extra from an explicit extra object or unknown keys, and accepts only 201.
  Tests: `identity/v3/regions/native_contracts_test.go:TestNativeRegionRoutesBodiesPagingAndDecode`, `identity/v3/regions/native_contracts_test.go:TestNativeRegionStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/regions call only; Python identity Region resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/regions.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.API.Get`

- Get GETs regions/{id}, decodes the region envelope with Extra from unknown keys where a {} or null envelope decodes to nil without an error, and accepts only 200.
  Tests: `identity/v3/regions/native_contracts_test.go:TestNativeRegionRoutesBodiesPagingAndDecode`, `identity/v3/regions/native_contracts_test.go:TestNativeRegionStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/regions call only; Python identity Region resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/regions.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithListQuery`

- List GETs regions with parent_region_id and WithListQuery parameters, follows the links.next string (null stops), accepts native pager 200/204/300 on the first page, returns io.EOF for a bodyless 204 and ends cleanly on an empty page, and rejects a nil option before HTTP.
  Tests: `identity/v3/regions/native_contracts_test.go:TestNativeRegionRoutesBodiesPagingAndDecode`, `identity/v3/regions/native_contracts_test.go:TestNativeRegionStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/regions call only; Python identity Region resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/regions.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithUpdateField`

- Update PATCHes {"region":{...}} to regions/{id} with the Description pointer able to send an empty string, no native Extra, WithUpdateField keys inside the envelope and core-key collisions or nil options rejected before HTTP, and accepts only 200.
  Tests: `identity/v3/regions/native_contracts_test.go:TestNativeRegionRoutesBodiesPagingAndDecode`, `identity/v3/regions/native_contracts_test.go:TestNativeRegionStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/regions call only; Python identity Region resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/regions.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.API.Delete`

- Delete sends DELETE regions/{id} without a body and accepts 202 or 204.
  Tests: `identity/v3/regions/native_contracts_test.go:TestNativeRegionRoutesBodiesPagingAndDecode`, `identity/v3/regions/native_contracts_test.go:TestNativeRegionStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/regions call only; Python identity Region resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/services.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithCreateField`

- Create POSTs {"service":{...}} to services always sending type (an empty string included) and the Enabled pointer, spreads Extra and WithCreateField keys inside the envelope while rejecting core-tag or Extra collisions and nil options before HTTP, decodes Extra with unknown keys plus name and description copies unless an extra object is present, and accepts only 201.
  Tests: `identity/v3/services/native_contracts_test.go:TestNativeServiceRoutesBodiesPagingAndDecode`, `identity/v3/services/native_contracts_test.go:TestNativeServiceStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/services call only; Python identity Service resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/services.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.Get`

- Get GETs services/{id}, decodes the service envelope where a {} or null envelope decodes to nil without an error, and accepts only 200.
  Tests: `identity/v3/services/native_contracts_test.go:TestNativeServiceRoutesBodiesPagingAndDecode`, `identity/v3/services/native_contracts_test.go:TestNativeServiceStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/services call only; Python identity Service resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/services.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithListQuery`

- List GETs services with type and name filters plus WithListQuery parameters, follows the links.next string (null stops), accepts native pager 200/204/300 on the first page, returns io.EOF for a bodyless 204 and ends cleanly on an empty page, and rejects a nil option before HTTP.
  Tests: `identity/v3/services/native_contracts_test.go:TestNativeServiceRoutesBodiesPagingAndDecode`, `identity/v3/services/native_contracts_test.go:TestNativeServiceStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/services call only; Python identity Service resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/services.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithUpdateField`

- Update PATCHes {"service":{...}} to services/{id} with Name, Description and Enabled pointers able to send empty or false values, Extra and WithUpdateField keys inside the envelope, core-key collisions rejected before HTTP, and accepts only 200.
  Tests: `identity/v3/services/native_contracts_test.go:TestNativeServiceRoutesBodiesPagingAndDecode`, `identity/v3/services/native_contracts_test.go:TestNativeServiceStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/services call only; Python identity Service resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/services.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.Delete`

- Delete sends DELETE services/{id} without a body and accepts 202 or 204.
  Tests: `identity/v3/services/native_contracts_test.go:TestNativeServiceRoutesBodiesPagingAndDecode`, `identity/v3/services/native_contracts_test.go:TestNativeServiceStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/services call only; Python identity Service resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/endpoints.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.WithCreateField`

- Create POSTs {"endpoint":{...}} to endpoints requiring interface, url and service_id, sending the Enabled pointer and WithCreateField keys inside the envelope, rejecting missing fields, core-key collisions and nil options before HTTP, decoding region_id, and accepting the POST default 201 or 202.
  Tests: `identity/v3/endpoints/native_contracts_test.go:TestNativeEndpointRoutesBodiesPagingAndDecode`, `identity/v3/endpoints/native_contracts_test.go:TestNativeEndpointStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/endpoints call only; Python identity Endpoint resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/endpoints.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.Get`

- Get GETs endpoints/{id}, decodes the endpoint envelope where a {} or null envelope decodes to nil without an error, and accepts only 200.
  Tests: `identity/v3/endpoints/native_contracts_test.go:TestNativeEndpointRoutesBodiesPagingAndDecode`, `identity/v3/endpoints/native_contracts_test.go:TestNativeEndpointStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/endpoints call only; Python identity Endpoint resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/endpoints.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.WithListOptions`

- List GETs endpoints with the interface, service_id and region_id filters copied from ListOpts onto the generated builder, because native List applies BuildQueryString to its argument instead of calling ToEndpointListParams; it follows the links.next string with native pager 200/204/300 handling, rejects raw query extensions and a nil option before HTTP, and the Resources collection maps query keys onto typed ListOpts and rejects unknown keys.
  Tests: `identity/v3/endpoints/native_contracts_test.go:TestNativeEndpointRoutesBodiesPagingAndDecode`, `identity/v3/endpoints/native_contracts_test.go:TestNativeEndpointStatusesDecodeAndPreflight`, `identity/v3/endpoints/native_contracts_test.go:TestNativeEndpointListFiltersMatchNativeQuery`.
- The sdkgen adapter copies q-tagged fields only for reviewed native calls that reflect their builder with BuildQueryString, disables query extensions for them, and fails generation for any unreviewed call of that shape.
  Tests: `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedQueryFieldsAreCopiedOnlyForReviewedLists`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/endpoints call only; Python identity Endpoint resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and copies the native q fields onto its builder so native BuildQueryString sees them; raw query extensions are unsupported because the native call cannot send them, and native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/endpoints.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.WithUpdateField`

- Update PATCHes {"endpoint":{...}} to endpoints/{id} omitting empty fields, sending the Enabled pointer and WithUpdateField keys inside the envelope, rejecting core-key collisions and nil options before HTTP, and accepts the PATCH default 200, 202 or 204 where a bodyless 204 returns a nil endpoint without an error.
  Tests: `identity/v3/endpoints/native_contracts_test.go:TestNativeEndpointRoutesBodiesPagingAndDecode`, `identity/v3/endpoints/native_contracts_test.go:TestNativeEndpointStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/endpoints call only; Python identity Endpoint resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/endpoints.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.Delete`

- Delete sends DELETE endpoints/{id} without a body and accepts 202 or 204.
  Tests: `identity/v3/endpoints/native_contracts_test.go:TestNativeEndpointRoutesBodiesPagingAndDecode`, `identity/v3/endpoints/native_contracts_test.go:TestNativeEndpointStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/endpoints call only; Python identity Endpoint resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/policies.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.WithCreateField`

- Create POSTs {"policy":{...}} to policies requiring type and a non-nil blob (an empty slice sends ""), always sending blob as a string, rejecting a type over 255 characters with StringFieldLengthExceedsLimit, missing fields, a blob or core-tag extension collision and nil options before HTTP, spreading Extra inside the envelope, and accepts only 201.
  Tests: `identity/v3/policies/native_contracts_test.go:TestNativePolicyRoutesBodiesPagingAndDecode`, `identity/v3/policies/native_contracts_test.go:TestNativePolicyStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/policies call only; Python identity Policy resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/policies.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.API.Get`

- Get GETs policies/{id}, decodes the policy envelope with Extra from unknown keys where a {} or null envelope decodes to nil without an error, and accepts only 200.
  Tests: `identity/v3/policies/native_contracts_test.go:TestNativePolicyRoutesBodiesPagingAndDecode`, `identity/v3/policies/native_contracts_test.go:TestNativePolicyStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/policies call only; Python identity Policy resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/policies.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.WithListQuery`

- List GETs policies with type, TYPE__COMPARATOR Filters (also leaking a "-" Python dict parameter from the q:"-" map) and WithListQuery parameters, returns an unwrapped InvalidListFilter before HTTP for a malformed filter key, follows the links.next string (null stops), accepts native pager 200/204/300 on the first page, returns io.EOF for a bodyless 204 and ends cleanly on an empty page, and rejects a nil option before HTTP.
  Tests: `identity/v3/policies/native_contracts_test.go:TestNativePolicyRoutesBodiesPagingAndDecode`, `identity/v3/policies/native_contracts_test.go:TestNativePolicyStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/policies call only; Python identity Policy resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/policies.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.WithUpdateField`

- Update PATCHes {"policy":{...}} to policies/{id} omitting an empty type and blob so a WithUpdateField blob is accepted, rejecting a type over 255 characters, core-key collisions and nil options before HTTP, spreading Extra inside the envelope, and accepts only 200.
  Tests: `identity/v3/policies/native_contracts_test.go:TestNativePolicyRoutesBodiesPagingAndDecode`, `identity/v3/policies/native_contracts_test.go:TestNativePolicyStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/policies call only; Python identity Policy resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/policies.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.API.Delete`

- Delete sends DELETE policies/{id} without a body and accepts 202 or 204.
  Tests: `identity/v3/policies/native_contracts_test.go:TestNativePolicyRoutesBodiesPagingAndDecode`, `identity/v3/policies/native_contracts_test.go:TestNativePolicyStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/policies call only; Python identity Policy resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/limits.BatchCreate`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.API.BatchCreate`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.WithBatchCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.WithBatchCreateField`

- BatchCreate POSTs an envelope-free row array {"limits":[...]} to limits requiring service_id and resource_name per row (ErrMissingInput before HTTP), always sending resource_limit (0 included), sending an empty array for an empty or nil batch, placing WithBatchCreateField keys at the root beside limits and rejecting a limits key or nil option before HTTP, decodes the limits array (nil when absent), and accepts only 201.
  Tests: `identity/v3/limits/native_contracts_test.go:TestNativeLimitRoutesBodiesPagingAndDecode`, `identity/v3/limits/native_contracts_test.go:TestNativeLimitStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/limits call only; Python identity Limit resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/limits.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.API.Get`

- Get GETs limits/{id}, decodes the limit envelope where a {} or null envelope decodes to nil without an error, and accepts only 200.
  Tests: `identity/v3/limits/native_contracts_test.go:TestNativeLimitRoutesBodiesPagingAndDecode`, `identity/v3/limits/native_contracts_test.go:TestNativeLimitStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/limits call only; Python identity Limit resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/limits.GetEnforcementModel`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.API.GetEnforcementModel`

- GetEnforcementModel GETs limits/model, decodes the model envelope's name and description where a null model is nil without an error, and accepts only 200.
  Tests: `identity/v3/limits/native_contracts_test.go:TestNativeLimitRoutesBodiesPagingAndDecode`, `identity/v3/limits/native_contracts_test.go:TestNativeLimitStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/limits call only; Python identity Limit resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/limits.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.WithListQuery`

- List GETs limits with region_id, project_id, domain_id, service_id and resource_name filters plus WithListQuery parameters, follows the links.next string (null stops), accepts native pager 200/204/300 on the first page, returns io.EOF for a bodyless 204 and ends cleanly on an empty page, and rejects a nil option before HTTP.
  Tests: `identity/v3/limits/native_contracts_test.go:TestNativeLimitRoutesBodiesPagingAndDecode`, `identity/v3/limits/native_contracts_test.go:TestNativeLimitStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/limits call only; Python identity Limit resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/limits.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.WithUpdateField`

- Update PATCHes {"limit":{...}} to limits/{id} with Description and ResourceLimit pointers able to send "" and 0, WithUpdateField keys inside the envelope, core-key collisions rejected before HTTP, and accepts only 200.
  Tests: `identity/v3/limits/native_contracts_test.go:TestNativeLimitRoutesBodiesPagingAndDecode`, `identity/v3/limits/native_contracts_test.go:TestNativeLimitStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/limits call only; Python identity Limit resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/limits.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/limits.API.Delete`

- Delete sends DELETE limits/{id} without a body and accepts 202 or 204.
  Tests: `identity/v3/limits/native_contracts_test.go:TestNativeLimitRoutesBodiesPagingAndDecode`, `identity/v3/limits/native_contracts_test.go:TestNativeLimitStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/limits call only; Python identity Limit resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/registeredlimits.BatchCreate`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.API.BatchCreate`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithBatchCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithBatchCreateField`

- BatchCreate POSTs an envelope-free row array {"registered_limits":[...]} to registered_limits requiring service_id and resource_name per row while the numeric required default_limit accepts and always sends 0, sending an empty array for an empty batch, placing WithBatchCreateField keys at the root and rejecting a registered_limits key or nil option before HTTP, decodes the array (nil when absent), and accepts only 201.
  Tests: `identity/v3/registeredlimits/native_contracts_test.go:TestNativeRegisteredLimitRoutesBodiesPagingAndDecode`, `identity/v3/registeredlimits/native_contracts_test.go:TestNativeRegisteredLimitStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/registeredlimits call only; Python identity RegisteredLimit resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/registeredlimits.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.API.Get`

- Get GETs registered_limits/{id}, decodes the registered_limit envelope where a {} or null envelope decodes to nil without an error, and accepts only 200.
  Tests: `identity/v3/registeredlimits/native_contracts_test.go:TestNativeRegisteredLimitRoutesBodiesPagingAndDecode`, `identity/v3/registeredlimits/native_contracts_test.go:TestNativeRegisteredLimitStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/registeredlimits call only; Python identity RegisteredLimit resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/registeredlimits.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithListQuery`

- List GETs registered_limits with region_id, service_id and resource_name filters plus WithListQuery parameters, follows the links.next string (null stops), accepts native pager 200/204/300 on the first page, returns io.EOF for a bodyless 204 and ends cleanly on an empty page, and rejects a nil option before HTTP.
  Tests: `identity/v3/registeredlimits/native_contracts_test.go:TestNativeRegisteredLimitRoutesBodiesPagingAndDecode`, `identity/v3/registeredlimits/native_contracts_test.go:TestNativeRegisteredLimitStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/registeredlimits call only; Python identity RegisteredLimit resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/registeredlimits.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithUpdateField`

- Update PATCHes {"registered_limit":{...}} to registered_limits/{id} with Description and DefaultLimit pointers able to send "" and 0, optional region_id, service_id and resource_name, WithUpdateField keys inside the envelope, core-key collisions and nil options rejected before HTTP, and accepts only 200.
  Tests: `identity/v3/registeredlimits/native_contracts_test.go:TestNativeRegisteredLimitRoutesBodiesPagingAndDecode`, `identity/v3/registeredlimits/native_contracts_test.go:TestNativeRegisteredLimitStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/registeredlimits call only; Python identity RegisteredLimit resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/registeredlimits.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.API.Delete`

- Delete sends DELETE registered_limits/{id} without a body and accepts 202 or 204.
  Tests: `identity/v3/registeredlimits/native_contracts_test.go:TestNativeRegisteredLimitRoutesBodiesPagingAndDecode`, `identity/v3/registeredlimits/native_contracts_test.go:TestNativeRegisteredLimitStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/registeredlimits call only; Python identity RegisteredLimit resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/projectendpoints.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projectendpoints.API.Create`

- Create sends a bodyless PUT OS-EP-FILTER/projects/{project}/endpoints/{endpoint} and accepts only 204.
  Tests: `identity/v3/projectendpoints/native_contracts_test.go:TestNativeProjectEndpointRoutesPagingAndDecode`, `identity/v3/projectendpoints/native_contracts_test.go:TestNativeProjectEndpointStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/projectendpoints call only; Python identity project endpoint association parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/projectendpoints.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projectendpoints.API.Delete`

- Delete sends DELETE OS-EP-FILTER/projects/{project}/endpoints/{endpoint} and accepts only 204, so 202 is an error.
  Tests: `identity/v3/projectendpoints/native_contracts_test.go:TestNativeProjectEndpointRoutesPagingAndDecode`, `identity/v3/projectendpoints/native_contracts_test.go:TestNativeProjectEndpointStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/projectendpoints call only; Python identity project endpoint association parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/projectendpoints.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projectendpoints.API.List`

- List GETs OS-EP-FILTER/projects/{project}/endpoints without options, decodes id, interface, region, service_id and url, follows the links.next string (null stops), accepts native pager 200/204/300 on the first page, returns io.EOF for a bodyless 204 and ends cleanly on an empty page, and yields the native error without request.Wrap context.
  Tests: `identity/v3/projectendpoints/native_contracts_test.go:TestNativeProjectEndpointRoutesPagingAndDecode`, `identity/v3/projectendpoints/native_contracts_test.go:TestNativeProjectEndpointStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/projectendpoints call only; Python identity project endpoint association parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/federation.CreateMapping`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federation.API.CreateMapping`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federation.WithCreateMappingOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federation.WithCreateMappingField`

- CreateMapping PUTs {"mapping":{"rules":[...]}} to OS-FEDERATION/mappings/{id} sending null for nil rules and nil local/remote arrays and always sending remote type, omitting other empty rule fields, placing WithCreateMappingField keys inside the envelope while rejecting a rules key or nil option before HTTP, decodes the rule tree, and accepts only 201.
  Tests: `identity/v3/federation/native_contracts_test.go:TestNativeMappingRoutesBodiesPagingAndDecode`, `identity/v3/federation/native_contracts_test.go:TestNativeMappingStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/federation call only; Python identity federation Mapping resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/federation.GetMapping`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federation.API.GetMapping`

- GetMapping GETs OS-FEDERATION/mappings/{id}, decodes the mapping envelope and rule tree where a {} or null envelope decodes to nil without an error and a malformed rule fails, and accepts only 200.
  Tests: `identity/v3/federation/native_contracts_test.go:TestNativeMappingRoutesBodiesPagingAndDecode`, `identity/v3/federation/native_contracts_test.go:TestNativeMappingStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/federation call only; Python identity federation Mapping resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/federation.ListMappings`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federation.API.ListMappings`

- ListMappings GETs OS-FEDERATION/mappings without options and follows the links.next string (null stops), accepts native pager 200/204/300 on the first page, returns io.EOF for a bodyless 204 and ends cleanly on an empty page.
  Tests: `identity/v3/federation/native_contracts_test.go:TestNativeMappingRoutesBodiesPagingAndDecode`, `identity/v3/federation/native_contracts_test.go:TestNativeMappingStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/federation call only; Python identity federation Mapping resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/federation.UpdateMapping`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federation.API.UpdateMapping`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federation.WithUpdateMappingOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federation.WithUpdateMappingField`

- UpdateMapping PATCHes {"mapping":{"rules":[...]}} to OS-FEDERATION/mappings/{id} with the same rule serialization as CreateMapping, WithUpdateMappingField keys inside the envelope, a rules key or nil option rejected before HTTP, and accepts only 200.
  Tests: `identity/v3/federation/native_contracts_test.go:TestNativeMappingRoutesBodiesPagingAndDecode`, `identity/v3/federation/native_contracts_test.go:TestNativeMappingStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/federation call only; Python identity federation Mapping resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/federation.DeleteMapping`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federation.API.DeleteMapping`

- DeleteMapping sends DELETE OS-FEDERATION/mappings/{id} without a body and accepts 202 or 204.
  Tests: `identity/v3/federation/native_contracts_test.go:TestNativeMappingRoutesBodiesPagingAndDecode`, `identity/v3/federation/native_contracts_test.go:TestNativeMappingStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/federation call only; Python identity federation Mapping resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

## References

- [regions](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/regions/requests.go)
- [services](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/services/requests.go)
- [endpoints](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/endpoints/requests.go)
- [policies](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/policies/requests.go)
- [limits](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/limits/requests.go)
- [registeredlimits](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/registeredlimits/requests.go)
- [projectendpoints](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/projectendpoints/requests.go)
- [federation](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/federation/requests.go)
