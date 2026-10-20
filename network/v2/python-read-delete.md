# Python Read Delete

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/get_network`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.API.GetWithFields`

- GET uses the pinned member route and all inherited Resource.Body aliases, independently of list-query aliases. Caller field order, repeats, empty and unknown names survive; seventeen Source files and 208 effective Body field mappings are freshly SHA-checked and extracted using the existing AST-only MRO resolver without executing Source modules.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsPreservesAliasesOrderAndNativeModels`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldExtractionUsesBodyMROWithoutExecutingSource`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationMatchesFreshPinnedSource`.
- Nil and empty fields omit the query. Sparse selected responses are extracted as the existing typed model without ID seeding, and HTTP200..399 follow the dedicated Python read profile. SubnetPool alone adapts the native required prefix fields when omitted, preserving explicit null and invalid-type failures.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsEmptyAndSparseResponse`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsAcceptsSourceResponseStatuses`.
- Native unexpected status, response evidence, decoder and transport causes are preserved under OperationError. Fields and aliases are copied before HTTP callbacks and reusable concurrently; pre/post-request cancellation keeps custom context causes.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsKeepsNativeDecodeStatusAndWireCauses`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsCancellationStopsBeforeHTTP`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsSnapshotAndConcurrentReuse`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationRejectsNativeResultAndInputDrift`.

Python/Go differences:

- Python accepts a Resource or an ID and returns a mutable Resource; Go takes a string ID and returns the pinned Gophercloud typed model with native decoder behavior and zero values for omitted fields. Arbitrary Resource/session/cache behavior is tracked separately.
- The SDK-owned field getter accepts HTTP200..399, preserves unknown fields in the query, and never synthesizes an ID. Invalid JSON or known-field types remain errors; existing native Get keeps its native status and extraction policy.

### `python:network/v2/get_subnet`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.GetWithFields`

- GET uses the pinned member route and all inherited Resource.Body aliases, independently of list-query aliases. Caller field order, repeats, empty and unknown names survive; seventeen Source files and 208 effective Body field mappings are freshly SHA-checked and extracted using the existing AST-only MRO resolver without executing Source modules.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsPreservesAliasesOrderAndNativeModels`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldExtractionUsesBodyMROWithoutExecutingSource`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationMatchesFreshPinnedSource`.
- Nil and empty fields omit the query. Sparse selected responses are extracted as the existing typed model without ID seeding, and HTTP200..399 follow the dedicated Python read profile. SubnetPool alone adapts the native required prefix fields when omitted, preserving explicit null and invalid-type failures.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsEmptyAndSparseResponse`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsAcceptsSourceResponseStatuses`.
- Native unexpected status, response evidence, decoder and transport causes are preserved under OperationError. Fields and aliases are copied before HTTP callbacks and reusable concurrently; pre/post-request cancellation keeps custom context causes.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsKeepsNativeDecodeStatusAndWireCauses`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsCancellationStopsBeforeHTTP`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsSnapshotAndConcurrentReuse`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationRejectsNativeResultAndInputDrift`.

Python/Go differences:

- Python accepts a Resource or an ID and returns a mutable Resource; Go takes a string ID and returns the pinned Gophercloud typed model with native decoder behavior and zero values for omitted fields. Arbitrary Resource/session/cache behavior is tracked separately.
- The SDK-owned field getter accepts HTTP200..399, preserves unknown fields in the query, and never synthesizes an ID. Invalid JSON or known-field types remain errors; existing native Get keeps its native status and extraction policy.

### `python:network/v2/get_port`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.GetWithFields`

- GET uses the pinned member route and all inherited Resource.Body aliases, independently of list-query aliases. Caller field order, repeats, empty and unknown names survive; seventeen Source files and 208 effective Body field mappings are freshly SHA-checked and extracted using the existing AST-only MRO resolver without executing Source modules.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsPreservesAliasesOrderAndNativeModels`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldExtractionUsesBodyMROWithoutExecutingSource`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationMatchesFreshPinnedSource`.
- Nil and empty fields omit the query. Sparse selected responses are extracted as the existing typed model without ID seeding, and HTTP200..399 follow the dedicated Python read profile. SubnetPool alone adapts the native required prefix fields when omitted, preserving explicit null and invalid-type failures.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsEmptyAndSparseResponse`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsAcceptsSourceResponseStatuses`.
- Native unexpected status, response evidence, decoder and transport causes are preserved under OperationError. Fields and aliases are copied before HTTP callbacks and reusable concurrently; pre/post-request cancellation keeps custom context causes.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsKeepsNativeDecodeStatusAndWireCauses`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsCancellationStopsBeforeHTTP`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsSnapshotAndConcurrentReuse`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationRejectsNativeResultAndInputDrift`.

Python/Go differences:

- Python accepts a Resource or an ID and returns a mutable Resource; Go takes a string ID and returns the pinned Gophercloud typed model with native decoder behavior and zero values for omitted fields. Arbitrary Resource/session/cache behavior is tracked separately.
- The SDK-owned field getter accepts HTTP200..399, preserves unknown fields in the query, and never synthesizes an ID. Invalid JSON or known-field types remain errors; existing native Get keeps its native status and extraction policy.

### `python:network/v2/get_ip`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.API.GetWithFields`

- GET uses the pinned member route and all inherited Resource.Body aliases, independently of list-query aliases. Caller field order, repeats, empty and unknown names survive; seventeen Source files and 208 effective Body field mappings are freshly SHA-checked and extracted using the existing AST-only MRO resolver without executing Source modules.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsPreservesAliasesOrderAndNativeModels`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldExtractionUsesBodyMROWithoutExecutingSource`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationMatchesFreshPinnedSource`.
- Nil and empty fields omit the query. Sparse selected responses are extracted as the existing typed model without ID seeding, and HTTP200..399 follow the dedicated Python read profile. SubnetPool alone adapts the native required prefix fields when omitted, preserving explicit null and invalid-type failures.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsEmptyAndSparseResponse`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsAcceptsSourceResponseStatuses`.
- Native unexpected status, response evidence, decoder and transport causes are preserved under OperationError. Fields and aliases are copied before HTTP callbacks and reusable concurrently; pre/post-request cancellation keeps custom context causes.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsKeepsNativeDecodeStatusAndWireCauses`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsCancellationStopsBeforeHTTP`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsSnapshotAndConcurrentReuse`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationRejectsNativeResultAndInputDrift`.

Python/Go differences:

- Python accepts a Resource or an ID and returns a mutable Resource; Go takes a string ID and returns the pinned Gophercloud typed model with native decoder behavior and zero values for omitted fields. Arbitrary Resource/session/cache behavior is tracked separately.
- The SDK-owned field getter accepts HTTP200..399, preserves unknown fields in the query, and never synthesizes an ID. Invalid JSON or known-field types remain errors; existing native Get keeps its native status and extraction policy.

### `python:network/v2/get_router`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.GetWithFields`

- GET uses the pinned member route and all inherited Resource.Body aliases, independently of list-query aliases. Caller field order, repeats, empty and unknown names survive; seventeen Source files and 208 effective Body field mappings are freshly SHA-checked and extracted using the existing AST-only MRO resolver without executing Source modules.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsPreservesAliasesOrderAndNativeModels`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldExtractionUsesBodyMROWithoutExecutingSource`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationMatchesFreshPinnedSource`.
- Nil and empty fields omit the query. Sparse selected responses are extracted as the existing typed model without ID seeding, and HTTP200..399 follow the dedicated Python read profile. SubnetPool alone adapts the native required prefix fields when omitted, preserving explicit null and invalid-type failures.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsEmptyAndSparseResponse`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsAcceptsSourceResponseStatuses`.
- Native unexpected status, response evidence, decoder and transport causes are preserved under OperationError. Fields and aliases are copied before HTTP callbacks and reusable concurrently; pre/post-request cancellation keeps custom context causes.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsKeepsNativeDecodeStatusAndWireCauses`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsCancellationStopsBeforeHTTP`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsSnapshotAndConcurrentReuse`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationRejectsNativeResultAndInputDrift`.

Python/Go differences:

- Python accepts a Resource or an ID and returns a mutable Resource; Go takes a string ID and returns the pinned Gophercloud typed model with native decoder behavior and zero values for omitted fields. Arbitrary Resource/session/cache behavior is tracked separately.
- The SDK-owned field getter accepts HTTP200..399, preserves unknown fields in the query, and never synthesizes an ID. Invalid JSON or known-field types remain errors; existing native Get keeps its native status and extraction policy.

### `python:network/v2/get_security_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.API.GetWithFields`

- GET uses the pinned member route and all inherited Resource.Body aliases, independently of list-query aliases. Caller field order, repeats, empty and unknown names survive; seventeen Source files and 208 effective Body field mappings are freshly SHA-checked and extracted using the existing AST-only MRO resolver without executing Source modules.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsPreservesAliasesOrderAndNativeModels`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldExtractionUsesBodyMROWithoutExecutingSource`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationMatchesFreshPinnedSource`.
- Nil and empty fields omit the query. Sparse selected responses are extracted as the existing typed model without ID seeding, and HTTP200..399 follow the dedicated Python read profile. SubnetPool alone adapts the native required prefix fields when omitted, preserving explicit null and invalid-type failures.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsEmptyAndSparseResponse`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsAcceptsSourceResponseStatuses`.
- Native unexpected status, response evidence, decoder and transport causes are preserved under OperationError. Fields and aliases are copied before HTTP callbacks and reusable concurrently; pre/post-request cancellation keeps custom context causes.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsKeepsNativeDecodeStatusAndWireCauses`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsCancellationStopsBeforeHTTP`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsSnapshotAndConcurrentReuse`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationRejectsNativeResultAndInputDrift`.

Python/Go differences:

- Python accepts a Resource or an ID and returns a mutable Resource; Go takes a string ID and returns the pinned Gophercloud typed model with native decoder behavior and zero values for omitted fields. Arbitrary Resource/session/cache behavior is tracked separately.
- The SDK-owned field getter accepts HTTP200..399, preserves unknown fields in the query, and never synthesizes an ID. Invalid JSON or known-field types remain errors; existing native Get keeps its native status and extraction policy.

### `python:network/v2/get_security_group_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.API.GetWithFields`

- GET uses the pinned member route and all inherited Resource.Body aliases, independently of list-query aliases. Caller field order, repeats, empty and unknown names survive; seventeen Source files and 208 effective Body field mappings are freshly SHA-checked and extracted using the existing AST-only MRO resolver without executing Source modules.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsPreservesAliasesOrderAndNativeModels`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldExtractionUsesBodyMROWithoutExecutingSource`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationMatchesFreshPinnedSource`.
- Nil and empty fields omit the query. Sparse selected responses are extracted as the existing typed model without ID seeding, and HTTP200..399 follow the dedicated Python read profile. SubnetPool alone adapts the native required prefix fields when omitted, preserving explicit null and invalid-type failures.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsEmptyAndSparseResponse`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsAcceptsSourceResponseStatuses`.
- Native unexpected status, response evidence, decoder and transport causes are preserved under OperationError. Fields and aliases are copied before HTTP callbacks and reusable concurrently; pre/post-request cancellation keeps custom context causes.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsKeepsNativeDecodeStatusAndWireCauses`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsCancellationStopsBeforeHTTP`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsSnapshotAndConcurrentReuse`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationRejectsNativeResultAndInputDrift`.

Python/Go differences:

- Python accepts a Resource or an ID and returns a mutable Resource; Go takes a string ID and returns the pinned Gophercloud typed model with native decoder behavior and zero values for omitted fields. Arbitrary Resource/session/cache behavior is tracked separately.
- The SDK-owned field getter accepts HTTP200..399, preserves unknown fields in the query, and never synthesizes an ID. Invalid JSON or known-field types remain errors; existing native Get keeps its native status and extraction policy.

### `python:network/v2/get_subnet_pool`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.API.GetWithFields`

- GET uses the pinned member route and all inherited Resource.Body aliases, independently of list-query aliases. Caller field order, repeats, empty and unknown names survive; seventeen Source files and 208 effective Body field mappings are freshly SHA-checked and extracted using the existing AST-only MRO resolver without executing Source modules.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsPreservesAliasesOrderAndNativeModels`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldExtractionUsesBodyMROWithoutExecutingSource`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationMatchesFreshPinnedSource`.
- Nil and empty fields omit the query. Sparse selected responses are extracted as the existing typed model without ID seeding, and HTTP200..399 follow the dedicated Python read profile. SubnetPool alone adapts the native required prefix fields when omitted, preserving explicit null and invalid-type failures.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsEmptyAndSparseResponse`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsAcceptsSourceResponseStatuses`.
- Native unexpected status, response evidence, decoder and transport causes are preserved under OperationError. Fields and aliases are copied before HTTP callbacks and reusable concurrently; pre/post-request cancellation keeps custom context causes.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsKeepsNativeDecodeStatusAndWireCauses`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsCancellationStopsBeforeHTTP`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsSnapshotAndConcurrentReuse`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationRejectsNativeResultAndInputDrift`.

Python/Go differences:

- Python accepts a Resource or an ID and returns a mutable Resource; Go takes a string ID and returns the pinned Gophercloud typed model with native decoder behavior and zero values for omitted fields. Arbitrary Resource/session/cache behavior is tracked separately.
- The SDK-owned field getter accepts HTTP200..399, preserves unknown fields in the query, and never synthesizes an ID. Invalid JSON or known-field types remain errors; existing native Get keeps its native status and extraction policy.
- Only GetWithFields adapts omitted default_prefixlen/min_prefixlen/max_prefixlen to typed zero in an owned extraction map; explicit null and invalid values retain native errors. Existing native Get is unchanged.

### `python:network/v2/get_trunk`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.API.GetWithFields`

- GET uses the pinned member route and all inherited Resource.Body aliases, independently of list-query aliases. Caller field order, repeats, empty and unknown names survive; seventeen Source files and 208 effective Body field mappings are freshly SHA-checked and extracted using the existing AST-only MRO resolver without executing Source modules.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsPreservesAliasesOrderAndNativeModels`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldExtractionUsesBodyMROWithoutExecutingSource`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationMatchesFreshPinnedSource`.
- Nil and empty fields omit the query. Sparse selected responses are extracted as the existing typed model without ID seeding, and HTTP200..399 follow the dedicated Python read profile. SubnetPool alone adapts the native required prefix fields when omitted, preserving explicit null and invalid-type failures.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsEmptyAndSparseResponse`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsAcceptsSourceResponseStatuses`.
- Native unexpected status, response evidence, decoder and transport causes are preserved under OperationError. Fields and aliases are copied before HTTP callbacks and reusable concurrently; pre/post-request cancellation keeps custom context causes.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsKeepsNativeDecodeStatusAndWireCauses`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsCancellationStopsBeforeHTTP`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsSnapshotAndConcurrentReuse`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationRejectsNativeResultAndInputDrift`.

Python/Go differences:

- Python accepts a Resource or an ID and returns a mutable Resource; Go takes a string ID and returns the pinned Gophercloud typed model with native decoder behavior and zero values for omitted fields. Arbitrary Resource/session/cache behavior is tracked separately.
- The SDK-owned field getter accepts HTTP200..399, preserves unknown fields in the query, and never synthesizes an ID. Invalid JSON or known-field types remain errors; existing native Get keeps its native status and extraction policy.

### `python:network/v2/get_qos_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.GetWithFields`

- GET uses the pinned member route and all inherited Resource.Body aliases, independently of list-query aliases. Caller field order, repeats, empty and unknown names survive; seventeen Source files and 208 effective Body field mappings are freshly SHA-checked and extracted using the existing AST-only MRO resolver without executing Source modules.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsPreservesAliasesOrderAndNativeModels`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldExtractionUsesBodyMROWithoutExecutingSource`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationMatchesFreshPinnedSource`.
- Nil and empty fields omit the query. Sparse selected responses are extracted as the existing typed model without ID seeding, and HTTP200..399 follow the dedicated Python read profile. SubnetPool alone adapts the native required prefix fields when omitted, preserving explicit null and invalid-type failures.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsEmptyAndSparseResponse`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsAcceptsSourceResponseStatuses`.
- Native unexpected status, response evidence, decoder and transport causes are preserved under OperationError. Fields and aliases are copied before HTTP callbacks and reusable concurrently; pre/post-request cancellation keeps custom context causes.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsKeepsNativeDecodeStatusAndWireCauses`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsCancellationStopsBeforeHTTP`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsSnapshotAndConcurrentReuse`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationRejectsNativeResultAndInputDrift`.

Python/Go differences:

- Python accepts a Resource or an ID and returns a mutable Resource; Go takes a string ID and returns the pinned Gophercloud typed model with native decoder behavior and zero values for omitted fields. Arbitrary Resource/session/cache behavior is tracked separately.
- The SDK-owned field getter accepts HTTP200..399, preserves unknown fields in the query, and never synthesizes an ID. Invalid JSON or known-field types remain errors; existing native Get keeps its native status and extraction policy.

### `python:network/v2/get_address_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.API.GetWithFields`

- GET uses the pinned member route and all inherited Resource.Body aliases, independently of list-query aliases. Caller field order, repeats, empty and unknown names survive; seventeen Source files and 208 effective Body field mappings are freshly SHA-checked and extracted using the existing AST-only MRO resolver without executing Source modules.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsPreservesAliasesOrderAndNativeModels`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldExtractionUsesBodyMROWithoutExecutingSource`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationMatchesFreshPinnedSource`.
- Nil and empty fields omit the query. Sparse selected responses are extracted as the existing typed model without ID seeding, and HTTP200..399 follow the dedicated Python read profile. SubnetPool alone adapts the native required prefix fields when omitted, preserving explicit null and invalid-type failures.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsEmptyAndSparseResponse`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsAcceptsSourceResponseStatuses`.
- Native unexpected status, response evidence, decoder and transport causes are preserved under OperationError. Fields and aliases are copied before HTTP callbacks and reusable concurrently; pre/post-request cancellation keeps custom context causes.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsKeepsNativeDecodeStatusAndWireCauses`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsCancellationStopsBeforeHTTP`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonGetFieldsSnapshotAndConcurrentReuse`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationRejectsNativeResultAndInputDrift`.

Python/Go differences:

- Python accepts a Resource or an ID and returns a mutable Resource; Go takes a string ID and returns the pinned Gophercloud typed model with native decoder behavior and zero values for omitted fields. Arbitrary Resource/session/cache behavior is tracked separately.
- The SDK-owned field getter accepts HTTP200..399, preserves unknown fields in the query, and never synthesizes an ID. Invalid JSON or known-field types remain errors; existing native Get keeps its native status and extraction policy.

### `python:network/v2/delete_network`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.API.DeleteWithOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithDeleteRevision`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithDeleteIgnoreMissing`

- DELETE uses the pinned member route and an empty body. Omitted revision adds no condition; explicit zero, negative and int64 bounds send If-Match: revision_number=N, overriding case-insensitive configured aliases without modifying the shared client.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteRevisionMissingPolicyAndNativeRoutes`.
- HTTP200..399 succeeds with opaque body handling. Pure actual404 is ignored by default and strict missing returns the native error. Existing guarded REST rejection observation and IgnoreMissing preserve read/close/transport/context and accepted-response failures, including errors containing another404.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteResponseStatusesAndCauses`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteCancellationWireFailuresAndInvalidOptions`.
- Concrete options apply in order. Bulk constructors capture pointer values, replace the complete policy and clone per invocation; the final configuration is independently owned even when custom callbacks retain their supplied pointer. Reuse is race-tested; nil/failed options and pre-cancelled requests never reach HTTP.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteOptionsFreezeCallbacksAndReusableBulk`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteCancellationWireFailuresAndInvalidOptions`.

Python/Go differences:

- Python Resource-or-ID and default-value arguments become a literal string ID plus concrete DeleteOption values; revision is bounded to Go int64 instead of arbitrary Python integer.
- The SDK-owned operation fixes its method, target and selected revision header through retry hooks and preserves physical response failures; native generated Delete remains available with its original strict status policy.

### `python:network/v2/delete_subnet`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.DeleteWithOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithDeleteRevision`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithDeleteIgnoreMissing`

- DELETE uses the pinned member route and an empty body. Omitted revision adds no condition; explicit zero, negative and int64 bounds send If-Match: revision_number=N, overriding case-insensitive configured aliases without modifying the shared client.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteRevisionMissingPolicyAndNativeRoutes`.
- HTTP200..399 succeeds with opaque body handling. Pure actual404 is ignored by default and strict missing returns the native error. Existing guarded REST rejection observation and IgnoreMissing preserve read/close/transport/context and accepted-response failures, including errors containing another404.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteResponseStatusesAndCauses`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteCancellationWireFailuresAndInvalidOptions`.
- Concrete options apply in order. Bulk constructors capture pointer values, replace the complete policy and clone per invocation; the final configuration is independently owned even when custom callbacks retain their supplied pointer. Reuse is race-tested; nil/failed options and pre-cancelled requests never reach HTTP.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteOptionsFreezeCallbacksAndReusableBulk`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteCancellationWireFailuresAndInvalidOptions`.

Python/Go differences:

- Python Resource-or-ID and default-value arguments become a literal string ID plus concrete DeleteOption values; revision is bounded to Go int64 instead of arbitrary Python integer.
- The SDK-owned operation fixes its method, target and selected revision header through retry hooks and preserves physical response failures; native generated Delete remains available with its original strict status policy.

### `python:network/v2/delete_port`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.DeleteWithOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithDeleteRevision`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithDeleteIgnoreMissing`

- DELETE uses the pinned member route and an empty body. Omitted revision adds no condition; explicit zero, negative and int64 bounds send If-Match: revision_number=N, overriding case-insensitive configured aliases without modifying the shared client.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteRevisionMissingPolicyAndNativeRoutes`.
- HTTP200..399 succeeds with opaque body handling. Pure actual404 is ignored by default and strict missing returns the native error. Existing guarded REST rejection observation and IgnoreMissing preserve read/close/transport/context and accepted-response failures, including errors containing another404.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteResponseStatusesAndCauses`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteCancellationWireFailuresAndInvalidOptions`.
- Concrete options apply in order. Bulk constructors capture pointer values, replace the complete policy and clone per invocation; the final configuration is independently owned even when custom callbacks retain their supplied pointer. Reuse is race-tested; nil/failed options and pre-cancelled requests never reach HTTP.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteOptionsFreezeCallbacksAndReusableBulk`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteCancellationWireFailuresAndInvalidOptions`.

Python/Go differences:

- Python Resource-or-ID and default-value arguments become a literal string ID plus concrete DeleteOption values; revision is bounded to Go int64 instead of arbitrary Python integer.
- The SDK-owned operation fixes its method, target and selected revision header through retry hooks and preserves physical response failures; native generated Delete remains available with its original strict status policy.

### `python:network/v2/delete_ip`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.API.DeleteWithOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.WithDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.WithDeleteRevision`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.WithDeleteIgnoreMissing`

- DELETE uses the pinned member route and an empty body. Omitted revision adds no condition; explicit zero, negative and int64 bounds send If-Match: revision_number=N, overriding case-insensitive configured aliases without modifying the shared client.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteRevisionMissingPolicyAndNativeRoutes`.
- HTTP200..399 succeeds with opaque body handling. Pure actual404 is ignored by default and strict missing returns the native error. Existing guarded REST rejection observation and IgnoreMissing preserve read/close/transport/context and accepted-response failures, including errors containing another404.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteResponseStatusesAndCauses`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteCancellationWireFailuresAndInvalidOptions`.
- Concrete options apply in order. Bulk constructors capture pointer values, replace the complete policy and clone per invocation; the final configuration is independently owned even when custom callbacks retain their supplied pointer. Reuse is race-tested; nil/failed options and pre-cancelled requests never reach HTTP.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteOptionsFreezeCallbacksAndReusableBulk`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteCancellationWireFailuresAndInvalidOptions`.

Python/Go differences:

- Python Resource-or-ID and default-value arguments become a literal string ID plus concrete DeleteOption values; revision is bounded to Go int64 instead of arbitrary Python integer.
- The SDK-owned operation fixes its method, target and selected revision header through retry hooks and preserves physical response failures; native generated Delete remains available with its original strict status policy.

### `python:network/v2/delete_router`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.DeleteWithOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithDeleteRevision`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithDeleteIgnoreMissing`

- DELETE uses the pinned member route and an empty body. Omitted revision adds no condition; explicit zero, negative and int64 bounds send If-Match: revision_number=N, overriding case-insensitive configured aliases without modifying the shared client.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteRevisionMissingPolicyAndNativeRoutes`.
- HTTP200..399 succeeds with opaque body handling. Pure actual404 is ignored by default and strict missing returns the native error. Existing guarded REST rejection observation and IgnoreMissing preserve read/close/transport/context and accepted-response failures, including errors containing another404.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteResponseStatusesAndCauses`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteCancellationWireFailuresAndInvalidOptions`.
- Concrete options apply in order. Bulk constructors capture pointer values, replace the complete policy and clone per invocation; the final configuration is independently owned even when custom callbacks retain their supplied pointer. Reuse is race-tested; nil/failed options and pre-cancelled requests never reach HTTP.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteOptionsFreezeCallbacksAndReusableBulk`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteCancellationWireFailuresAndInvalidOptions`.

Python/Go differences:

- Python Resource-or-ID and default-value arguments become a literal string ID plus concrete DeleteOption values; revision is bounded to Go int64 instead of arbitrary Python integer.
- The SDK-owned operation fixes its method, target and selected revision header through retry hooks and preserves physical response failures; native generated Delete remains available with its original strict status policy.

### `python:network/v2/delete_security_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.API.DeleteWithOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithDeleteRevision`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithDeleteIgnoreMissing`

- DELETE uses the pinned member route and an empty body. Omitted revision adds no condition; explicit zero, negative and int64 bounds send If-Match: revision_number=N, overriding case-insensitive configured aliases without modifying the shared client.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteRevisionMissingPolicyAndNativeRoutes`.
- HTTP200..399 succeeds with opaque body handling. Pure actual404 is ignored by default and strict missing returns the native error. Existing guarded REST rejection observation and IgnoreMissing preserve read/close/transport/context and accepted-response failures, including errors containing another404.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteResponseStatusesAndCauses`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteCancellationWireFailuresAndInvalidOptions`.
- Concrete options apply in order. Bulk constructors capture pointer values, replace the complete policy and clone per invocation; the final configuration is independently owned even when custom callbacks retain their supplied pointer. Reuse is race-tested; nil/failed options and pre-cancelled requests never reach HTTP.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteOptionsFreezeCallbacksAndReusableBulk`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteCancellationWireFailuresAndInvalidOptions`.

Python/Go differences:

- Python Resource-or-ID and default-value arguments become a literal string ID plus concrete DeleteOption values; revision is bounded to Go int64 instead of arbitrary Python integer.
- The SDK-owned operation fixes its method, target and selected revision header through retry hooks and preserves physical response failures; native generated Delete remains available with its original strict status policy.

### `python:network/v2/delete_security_group_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.API.DeleteWithOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.WithDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.WithDeleteRevision`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.WithDeleteIgnoreMissing`

- DELETE uses the pinned member route and an empty body. Omitted revision adds no condition; explicit zero, negative and int64 bounds send If-Match: revision_number=N, overriding case-insensitive configured aliases without modifying the shared client.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteRevisionMissingPolicyAndNativeRoutes`.
- HTTP200..399 succeeds with opaque body handling. Pure actual404 is ignored by default and strict missing returns the native error. Existing guarded REST rejection observation and IgnoreMissing preserve read/close/transport/context and accepted-response failures, including errors containing another404.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteResponseStatusesAndCauses`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteCancellationWireFailuresAndInvalidOptions`.
- Concrete options apply in order. Bulk constructors capture pointer values, replace the complete policy and clone per invocation; the final configuration is independently owned even when custom callbacks retain their supplied pointer. Reuse is race-tested; nil/failed options and pre-cancelled requests never reach HTTP.
  Tests: `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteOptionsFreezeCallbacksAndReusableBulk`, `api/neutron_python_reads_deletes_test.go:TestNeutronPythonDeleteCancellationWireFailuresAndInvalidOptions`.

Python/Go differences:

- Python Resource-or-ID and default-value arguments become a literal string ID plus concrete DeleteOption values; revision is bounded to Go int64 instead of arbitrary Python integer.
- The SDK-owned operation fixes its method, target and selected revision header through retry hooks and preserves physical response failures; native generated Delete remains available with its original strict status policy.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "fmt"
    "os"
    "strconv"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(os.Getenv("OS_CLOUD")))
    if err != nil { panic(err) }
    service, err := conn.Network(ctx)
    if err != nil { panic(err) }
    selected, err := service.API.Networks.GetWithFields(ctx, os.Getenv("NETWORK_ID"),
        "id", "name", "is_admin_state_up", "revision_number")
    if err != nil { panic(err) }
    fmt.Printf("network: %s %s revision=%d\n", selected.ID, selected.Name, selected.RevisionNumber)

    if id := os.Getenv("DELETE_NETWORK_ID"); id != "" {
        var options []networks.DeleteOption
        if value := os.Getenv("DELETE_NETWORK_REVISION"); value != "" {
            revision, err := strconv.ParseInt(value, 10, 64)
            if err != nil { panic(err) }
            options = append(options, networks.WithDeleteRevision(revision))
        }
        if err := service.API.Networks.DeleteWithOptions(ctx, id, options...); err != nil { panic(err) }
    }
}
```

## References

- [Native Crud](native-crud.md)
- [..](../README.md)
