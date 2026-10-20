# Native Admin

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:blockstorage/v3/volumetypes.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.WithCreateField`

- Create POSTs types with a volume_type envelope (required name, omitempty description and extra_specs, os-volume-type-access:is_public pointer) plus extension fields inside the envelope, rejects a missing name, core-key extensions and nil options before HTTP, decodes the volume_type key (zero value for {}) and accepts only 200.
  Tests: `blockstorage/v3/volumetypes/native_admin_contracts_test.go:TestNativeVolumeTypeAdminRoutesBodiesAndDecode`, `blockstorage/v3/volumetypes/native_admin_contracts_test.go:TestNativeVolumeTypeAdminStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumetypes call only; Python volume type Resource parity, encryption and project access resources are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumetypes.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.WithUpdateField`

- Update PUTs types/{id} with a volume_type envelope of pointer name/description/is_public (nil omitted, empty strings kept, {} when unset; is_public rather than the access-prefixed create key) plus extension fields, rejects core-key extensions before HTTP, decodes the volume_type key and accepts only 200.
  Tests: `blockstorage/v3/volumetypes/native_admin_contracts_test.go:TestNativeVolumeTypeAdminRoutesBodiesAndDecode`, `blockstorage/v3/volumetypes/native_admin_contracts_test.go:TestNativeVolumeTypeAdminStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumetypes call only; Python volume type Resource parity, encryption and project access resources are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumetypes.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.Delete`

- Delete sends a bodyless DELETE to types/{id} and accepts the default 202 and 204.
  Tests: `blockstorage/v3/volumetypes/native_admin_contracts_test.go:TestNativeVolumeTypeAdminRoutesBodiesAndDecode`, `blockstorage/v3/volumetypes/native_admin_contracts_test.go:TestNativeVolumeTypeAdminStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumetypes call only; Python volume type Resource parity, encryption and project access resources are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumetypes.CreateExtraSpecs`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.CreateExtraSpecs`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.WithCreateExtraSpecsOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.WithCreateExtraSpecsField`

- CreateExtraSpecs POSTs {"extra_specs": map} (null for a nil map) to types/{id}/extra_specs with extension fields placed beside the string-map envelope, rejects an extra_specs extension before HTTP, returns the extra_specs string map (nil when absent) and accepts only 200.
  Tests: `blockstorage/v3/volumetypes/native_admin_contracts_test.go:TestNativeVolumeTypeAdminRoutesBodiesAndDecode`, `blockstorage/v3/volumetypes/native_admin_contracts_test.go:TestNativeVolumeTypeAdminStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumetypes call only; Python volume type Resource parity, encryption and project access resources are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumetypes.UpdateExtraSpec`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.UpdateExtraSpec`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.WithUpdateExtraSpecOptions`

- UpdateExtraSpec PUTs the single {key: value} pair without an envelope to types/{id}/extra_specs/{unescaped key}, rejects zero or several pairs and any body extension (including the shared WithCreateExtraSpecsField option) before HTTP, returns the envelope-free string map with a wrapped decode error for non-string values and accepts only 200.
  Tests: `blockstorage/v3/volumetypes/native_admin_contracts_test.go:TestNativeVolumeTypeAdminRoutesBodiesAndDecode`, `blockstorage/v3/volumetypes/native_admin_contracts_test.go:TestNativeVolumeTypeAdminStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumetypes call only; Python volume type Resource parity, encryption and project access resources are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumetypes.DeleteExtraSpec`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.DeleteExtraSpec`

- DeleteExtraSpec sends a bodyless DELETE to types/{id}/extra_specs/{unescaped key} and accepts only 202.
  Tests: `blockstorage/v3/volumetypes/native_admin_contracts_test.go:TestNativeVolumeTypeAdminRoutesBodiesAndDecode`, `blockstorage/v3/volumetypes/native_admin_contracts_test.go:TestNativeVolumeTypeAdminStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumetypes call only; Python volume type Resource parity, encryption and project access resources are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumetypes.AddAccess`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.AddAccess`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.WithAddAccessOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.WithAddAccessField`

- AddAccess POSTs {"addProjectAccess": {"project": ...}} to types/{id}/action without requiring the project (an empty string is sent), merges extension fields inside the action, rejects a project extension before HTTP, ignores the response body and accepts only 202.
  Tests: `blockstorage/v3/volumetypes/native_admin_contracts_test.go:TestNativeVolumeTypeAdminRoutesBodiesAndDecode`, `blockstorage/v3/volumetypes/native_admin_contracts_test.go:TestNativeVolumeTypeAdminStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumetypes call only; Python volume type Resource parity, encryption and project access resources are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumetypes.RemoveAccess`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.RemoveAccess`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.WithRemoveAccessOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.WithRemoveAccessField`

- RemoveAccess POSTs {"removeProjectAccess": {"project": ...}} to types/{id}/action with extension fields inside the action, rejects nil options before HTTP, ignores the response body and accepts only 202.
  Tests: `blockstorage/v3/volumetypes/native_admin_contracts_test.go:TestNativeVolumeTypeAdminRoutesBodiesAndDecode`, `blockstorage/v3/volumetypes/native_admin_contracts_test.go:TestNativeVolumeTypeAdminStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumetypes call only; Python volume type Resource parity, encryption and project access resources are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumetypes.ListAccesses`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.ListAccesses`

- ListAccesses GETs types/{id}/os-volume-type-access as a single page of volume_type_access rows without following links, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors (200/204/300 expected) without operation context.
  Tests: `blockstorage/v3/volumetypes/native_admin_contracts_test.go:TestNativeVolumeTypeAdminRoutesBodiesAndDecode`, `blockstorage/v3/volumetypes/native_admin_contracts_test.go:TestNativeVolumeTypeAdminStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumetypes call only; Python volume type Resource parity, encryption and project access resources are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumetypes.CreateEncryption`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.CreateEncryption`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.WithCreateEncryptionOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.WithCreateEncryptionField`

- CreateEncryption POSTs an encryption envelope with required provider and non-omitempty key_size/control_location/cipher (zero values sent) plus extension fields to types/{id}/encryption, rejects a missing provider and core-key extensions before HTTP, decodes the encryption key (zero value for {}) and accepts only 200.
  Tests: `blockstorage/v3/volumetypes/native_admin_contracts_test.go:TestNativeVolumeTypeAdminRoutesBodiesAndDecode`, `blockstorage/v3/volumetypes/native_admin_contracts_test.go:TestNativeVolumeTypeAdminStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumetypes call only; Python volume type Resource parity, encryption and project access resources are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumetypes.GetEncryption`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.GetEncryption`

- GetEncryption GETs types/{id}/encryption, decodes the envelope-free body into GetEncryptionType with raw string timestamps (null as empty) and a zero value for {}, and accepts only 200.
  Tests: `blockstorage/v3/volumetypes/native_admin_contracts_test.go:TestNativeVolumeTypeAdminRoutesBodiesAndDecode`, `blockstorage/v3/volumetypes/native_admin_contracts_test.go:TestNativeVolumeTypeAdminStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumetypes call only; Python volume type Resource parity, encryption and project access resources are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumetypes.GetEncryptionSpec`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.GetEncryptionSpec`

- GetEncryptionSpec GETs types/{id}/encryption/{unescaped key}, returns the whole envelope-free body as map[string]any with float64 numbers, and accepts only 200.
  Tests: `blockstorage/v3/volumetypes/native_admin_contracts_test.go:TestNativeVolumeTypeAdminRoutesBodiesAndDecode`, `blockstorage/v3/volumetypes/native_admin_contracts_test.go:TestNativeVolumeTypeAdminStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumetypes call only; Python volume type Resource parity, encryption and project access resources are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumetypes.UpdateEncryption`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.UpdateEncryption`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.WithUpdateEncryptionOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.WithUpdateEncryptionField`

- UpdateEncryption PUTs an encryption envelope that always carries key_size, provider, control_location and cipher (zero values included) plus extension fields to types/{id}/encryption/{encryptionID}, rejects core-key extensions before HTTP, decodes the encryption key and accepts only 200.
  Tests: `blockstorage/v3/volumetypes/native_admin_contracts_test.go:TestNativeVolumeTypeAdminRoutesBodiesAndDecode`, `blockstorage/v3/volumetypes/native_admin_contracts_test.go:TestNativeVolumeTypeAdminStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumetypes call only; Python volume type Resource parity, encryption and project access resources are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumetypes.DeleteEncryption`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.DeleteEncryption`

- DeleteEncryption sends a bodyless DELETE to types/{id}/encryption/{encryptionID} and accepts the default 202 and 204.
  Tests: `blockstorage/v3/volumetypes/native_admin_contracts_test.go:TestNativeVolumeTypeAdminRoutesBodiesAndDecode`, `blockstorage/v3/volumetypes/native_admin_contracts_test.go:TestNativeVolumeTypeAdminStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumetypes call only; Python volume type Resource parity, encryption and project access resources are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/manageablevolumes.ManageExisting`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/manageablevolumes.API.ManageExisting`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/manageablevolumes.WithManageExistingOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/manageablevolumes.WithManageExistingField`

- ManageExisting POSTs an all-omitempty volume envelope (host, cluster, ref, name, availability_zone, description, volume_type, bootable, metadata; nothing required, {} when empty) plus extension fields to manageable_volumes, rejects core-key extensions and nil options before HTTP, decodes the volume key into a v3 Volume with zone-less millisecond timestamps (a Z suffix fails) and accepts only 202.
  Tests: `blockstorage/v3/manageablevolumes/native_contracts_test.go:TestNativeManageableVolumeRouteBodyAndDecode`, `blockstorage/v3/manageablevolumes/native_contracts_test.go:TestNativeManageableVolumeStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/manageablevolumes call only; Python volume manage parity and microversion 3.8 selection are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v2/schedulerstats.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/schedulerstats.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/schedulerstats.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/schedulerstats.WithListQuery`

- List GETs scheduler-stats/get_pools with tenant_id, detail only when true and extension query values as a single page of pools without following pools_links, decodes capacity numbers with "infinite" as +Inf and other strings as zero and max_over_subscription_ratio as a string, fails on mistyped capability fields, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `blockstorage/v2/schedulerstats/native_contracts_test.go:TestNativeSchedulerStatsListRouteAndDecode`, `blockstorage/v2/schedulerstats/native_contracts_test.go:TestNativeSchedulerStatsListStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/schedulerstats call only; Python storage pool Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v2/services.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/services.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/services.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/services.WithListQuery`

- List GETs os-services with binary, host and extension query values as a single page without following services_links, decodes updated_at in the zone-less millisecond layout (null as zero, a Z suffix fails), yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `blockstorage/v2/services/native_contracts_test.go:TestNativeCinderServiceListRouteAndDecode`, `blockstorage/v2/services/native_contracts_test.go:TestNativeCinderServiceListStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/services call only; Python block storage service Resource parity and service enable/disable actions are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/schedulerstats.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/schedulerstats.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/schedulerstats.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/schedulerstats.WithListQuery`

- List GETs scheduler-stats/get_pools with tenant_id, detail only when true and extension query values as a single page of pools without following pools_links, decodes capacity numbers with "infinite" as +Inf and other strings as zero and max_over_subscription_ratio as a string, fails on mistyped capability fields, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `blockstorage/v3/schedulerstats/native_contracts_test.go:TestNativeSchedulerStatsListRouteAndDecode`, `blockstorage/v3/schedulerstats/native_contracts_test.go:TestNativeSchedulerStatsListStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/schedulerstats call only; Python storage pool Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/services.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services.WithListQuery`

- List GETs os-services with binary, host and extension query values as a single page without following services_links, decodes updated_at in the zone-less millisecond layout (null as zero, a Z suffix fails), yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `blockstorage/v3/services/native_contracts_test.go:TestNativeCinderServiceListRouteAndDecode`, `blockstorage/v3/services/native_contracts_test.go:TestNativeCinderServiceListStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/services call only; Python block storage service Resource parity and service enable/disable actions are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v2/quotasets.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/quotasets.API.Get`

- Get GETs os-quota-sets/{projectID}, decodes typed limits from the quota_set key with unknown per-type keys kept in Extra as JSON numbers, returns nil without an error when the envelope is missing, wraps mistyped fields as decode errors and accepts only 200.
  Tests: `blockstorage/v2/quotasets/native_contracts_test.go:TestNativeQuotaSetV2RoutesBodiesAndDecode`, `blockstorage/v2/quotasets/native_contracts_test.go:TestNativeQuotaSetV2StatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/quotasets call only; Python quota set parity and the SDK-owned v3 project quota scope are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v2/quotasets.GetDefaults`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/quotasets.API.GetDefaults`

- GetDefaults GETs os-quota-sets/{projectID}/defaults with the same quota_set decode as Get (nil without an error for a missing envelope, wrapped decode errors for mistyped fields) and accepts only 200.
  Tests: `blockstorage/v2/quotasets/native_contracts_test.go:TestNativeQuotaSetV2RoutesBodiesAndDecode`, `blockstorage/v2/quotasets/native_contracts_test.go:TestNativeQuotaSetV2StatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/quotasets call only; Python quota set parity and the SDK-owned v3 project quota scope are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v2/quotasets.GetUsage`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/quotasets.API.GetUsage`

- GetUsage GETs os-quota-sets/{projectID}?usage=true, returns a QuotaUsageSet value with in_use/limit/reserved/allocated per typed resource, drops per-type usage keys, returns a zero value for a missing envelope and accepts only 200.
  Tests: `blockstorage/v2/quotasets/native_contracts_test.go:TestNativeQuotaSetV2RoutesBodiesAndDecode`, `blockstorage/v2/quotasets/native_contracts_test.go:TestNativeQuotaSetV2StatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/quotasets call only; Python quota set parity and the SDK-owned v3 project quota scope are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v2/quotasets.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/quotasets.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/quotasets.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/quotasets.WithUpdateField`

- Update PUTs a quota_set envelope of pointer limits (nil omitted, 0 and -1 kept), force only when true and Extra keys merged afterwards so they override typed limits, plus extension fields; it rejects extensions colliding with typed keys (even omitted) or Extra keys and nil options before HTTP, decodes the quota_set key (nil when missing) and accepts only 200.
  Tests: `blockstorage/v2/quotasets/native_contracts_test.go:TestNativeQuotaSetV2RoutesBodiesAndDecode`, `blockstorage/v2/quotasets/native_contracts_test.go:TestNativeQuotaSetV2StatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/quotasets call only; Python quota set parity and the SDK-owned v3 project quota scope are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v2/quotasets.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/quotasets.API.Delete`

- Delete sends a bodyless DELETE to os-quota-sets/{projectID} to reset overrides and accepts only 200.
  Tests: `blockstorage/v2/quotasets/native_contracts_test.go:TestNativeQuotaSetV2RoutesBodiesAndDecode`, `blockstorage/v2/quotasets/native_contracts_test.go:TestNativeQuotaSetV2StatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/quotasets call only; Python quota set parity and the SDK-owned v3 project quota scope are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v2/limits.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/limits.API.Get`

- Get GETs limits, decodes the ten absolute integer fields and rate groups with raw next-available strings from the limits key, returns nil without an error when the envelope is missing, wraps mistyped fields as decode errors and accepts only 200.
  Tests: `blockstorage/v2/limits/native_contracts_test.go:TestNativeLimitsV2GetRouteAndDecode`, `blockstorage/v2/limits/native_contracts_test.go:TestNativeLimitsV2GetStatusesAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/limits call only; Python limits parity and the SDK-owned v3 limits fetch are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

## References

- [Native Volume Types](native-volume-types.md)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/volumetypes/requests.go)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/manageablevolumes/requests.go)
- [schedulerstats](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/schedulerstats/requests.go)
- [services](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/services/requests.go)
- [v2 quotasets](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v2/quotasets/requests.go)
- [v2 limits](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v2/limits/requests.go)
- [Quotasets](quotasets/README.md)
- [limit](limits/README.md)
