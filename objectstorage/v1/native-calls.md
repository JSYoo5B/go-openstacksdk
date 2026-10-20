# Native Calls

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:objectstorage/v1/accounts.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.WithGetOptions`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.WithGetHeader`

- Get HEADs the normalized client Endpoint (not ResourceBase) with X-Newest only when true and extension headers, rejects a core-header collision before HTTP, decodes numeric-string counters, nil absent quota and RFC1123 Date (malformed values fail) and accepts only 204.
  Tests: `objectstorage/v1/accounts/native_contracts_test.go:TestNativeAccountGetHeadsTheEndpointAndDecodesHeaders`, `objectstorage/v1/accounts/native_contracts_test.go:TestNativeAccountStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native objectstorage/v1/accounts call only; Python object store Resource parity and SDK-owned metadata, temp URL and lifecycle workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension header merge; native URL construction, escaping, OkCodes, paging and header decoding are unchanged.

### `gophercloud:objectstorage/v1/accounts.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.WithUpdateHeader`

- Update POSTs to the normalized Endpoint with X-Account-Meta-*, X-Remove-Account-Meta-*: remove, explicit pointer content-type/detect headers and temp URL key headers, rejects an invalid extension header before HTTP, decodes the response headers and accepts only 201/202/204.
  Tests: `objectstorage/v1/accounts/native_contracts_test.go:TestNativeAccountUpdatePostsMetadataHeaders`, `objectstorage/v1/accounts/native_contracts_test.go:TestNativeAccountStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native objectstorage/v1/accounts call only; Python object store Resource parity and SDK-owned metadata, temp URL and lifecycle workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension header merge; native URL construction, escaping, OkCodes, paging and header decoding are unchanged.

### `gophercloud:objectstorage/v1/containers.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithCreateHeader`

- Create PUTs the path-escaped container under ResourceBase with metadata and system headers (bool headers only when true), rejects empty or slash names and core-header collisions before HTTP, decodes response headers and accepts only 201/202/204.
  Tests: `objectstorage/v1/containers/native_contracts_test.go:TestNativeContainerHeaderCallsAndEscaping`, `objectstorage/v1/containers/native_contracts_test.go:TestNativeContainerStatusesNamesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native objectstorage/v1/containers call only; Python object store Resource parity and SDK-owned metadata, temp URL and lifecycle workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension header merge; native URL construction, escaping, OkCodes, paging and header decoding are unchanged.

### `gophercloud:objectstorage/v1/containers.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithGetOptions`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithGetHeader`

- Get HEADs the path-escaped container with X-Newest only when true, decodes counters, comma-split ACLs ([""] when absent), ParseBool versions flag (malformed fails), timestamp and storage policy, rejects slash names before HTTP and accepts only 200/204.
  Tests: `objectstorage/v1/containers/native_contracts_test.go:TestNativeContainerHeaderCallsAndEscaping`, `objectstorage/v1/containers/native_contracts_test.go:TestNativeContainerStatusesNamesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native objectstorage/v1/containers call only; Python object store Resource parity and SDK-owned metadata, temp URL and lifecycle workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension header merge; native URL construction, escaping, OkCodes, paging and header decoding are unchanged.

### `gophercloud:objectstorage/v1/containers.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithUpdateHeader`

- Update POSTs to the path-escaped container with metadata, remove-metadata and explicit pointer empty/false headers, rejects a nil option before HTTP and accepts only 201/202/204.
  Tests: `objectstorage/v1/containers/native_contracts_test.go:TestNativeContainerHeaderCallsAndEscaping`, `objectstorage/v1/containers/native_contracts_test.go:TestNativeContainerStatusesNamesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native objectstorage/v1/containers call only; Python object store Resource parity and SDK-owned metadata, temp URL and lifecycle workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension header merge; native URL construction, escaping, OkCodes, paging and header decoding are unchanged.

### `gophercloud:objectstorage/v1/containers.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.API.Delete`

- Delete DELETEs the path-escaped container, rejects slash names before HTTP, decodes response headers and accepts only 202/204.
  Tests: `objectstorage/v1/containers/native_contracts_test.go:TestNativeContainerHeaderCallsAndEscaping`, `objectstorage/v1/containers/native_contracts_test.go:TestNativeContainerStatusesNamesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native objectstorage/v1/containers call only; Python object store Resource parity and SDK-owned metadata, temp URL and lifecycle workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension header merge; native URL construction, escaping, OkCodes, paging and header decoding are unchanged.

### `gophercloud:objectstorage/v1/containers.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.API.List`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithListQuery`

- List GETs the account Endpoint with JSON Accept/Content-Type headers, ListOpts and extension query values, repeats the request with marker set to the last name until an empty page, treats a bodyless 204 as an empty listing and reports native page status errors without operation context.
  Tests: `objectstorage/v1/containers/native_contracts_test.go:TestNativeContainerListMarkerPagingAndBulkDelete`, `objectstorage/v1/containers/native_contracts_test.go:TestNativeContainerStatusesNamesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native objectstorage/v1/containers call only; Python object store Resource parity and SDK-owned metadata, temp URL and lifecycle workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension header merge; native URL construction, escaping, OkCodes, paging and header decoding are unchanged.

### `gophercloud:objectstorage/v1/containers.BulkDelete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.API.BulkDelete`

- BulkDelete POSTs path-escaped names one per line as text/plain to the account Endpoint with ?bulk-delete=true, rejects the whole request before HTTP when any name is empty or contains a slash, decodes deleted/not-found counts and errors and accepts only 200.
  Tests: `objectstorage/v1/containers/native_contracts_test.go:TestNativeContainerListMarkerPagingAndBulkDelete`, `objectstorage/v1/containers/native_contracts_test.go:TestNativeContainerStatusesNamesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native objectstorage/v1/containers call only; Python object store Resource parity and SDK-owned metadata, temp URL and lifecycle workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension header merge; native URL construction, escaping, OkCodes, paging and header decoding are unchanged.

### `gophercloud:objectstorage/v1/objects.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCreateBytes`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCreateQuery`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCreateHeader`

- Create retains native escaped object PUT, core/extension headers and query, default content MD5, explicit ETag with NoETag, native201/202 status and actual PUT header decoding. Final plain-nil Content creates an empty object rather than invoking the upstream nil-reader MD5 panic; typed-nil supplied Content is ErrInvalidOption before HTTP under every ETag mode.
  Tests: `objectstorage/v1/objects/create_defaults_test.go:TestNativeObjectCreateEmptyDefaults`, `objectstorage/v1/objects/create_defaults_test.go:TestNativeObjectCreateTypedNilReadersAndPreflight`, `objectstorage/v1/objects/native_contracts_test.go:TestNativeObjectCreateUpdateDeleteHeadersAndEscaping`, `objectstorage/v1/objects/native_contracts_test.go:TestNativeObjectStatusesNamesAndPreflight`, `objectstorage/v1/objects/native_contracts_test.go:TestNativeObjectCreateNilContentUsesEmptyBody`.
- WithCreateBytes snapshots nil/empty/nonempty bytes at option construction and creates a fresh reader on each application. Request options retain last-applied Content precedence; mutation of caller bytes and reusing an option do not change later uploads.
  Tests: `objectstorage/v1/objects/create_defaults_test.go:TestNativeObjectCreateBytesSnapshotAndOptionOrder`, `objectstorage/v1/objects/create_defaults_test.go:TestNativeObjectCreateEmptyDefaults`.
- Native checksum preparation retains physical read/seek causes and Create/objects wrapping without HTTP or closing a failed preflight reader. Empty bodies retain default ETag through an actual native503 retry to201;204 is still a native rejection.
  Tests: `objectstorage/v1/objects/create_defaults_test.go:TestNativeObjectCreateDefaultReaderErrorsRetainCause`, `objectstorage/v1/objects/create_defaults_test.go:TestNativeObjectCreateEmptyNativeRetryAndStatus`.

Python/Go differences:

- This declaration maps the native Gophercloud object Create call only. Python Resource and owned CreateObject bytes/file/SLO/DLO workflows remain separately tracked; fixing nil Content does not declare Python create_object full parity.
- The SDK applies an empty-reader default after final typed options, rejecting typed-nil readers before HTTP. Native URL, headers/query, status decoding and reader/retry behavior remain unchanged; WithCreateBytes is a reusable immutable input option, not a replacement for the owned CreateObject replay/evidence workflow.

### `gophercloud:objectstorage/v1/objects.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithGetOptions`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithGetQuery`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithGetHeader`

- Get HEADs the path-escaped object with X-Newest only when true and version-id/temp URL query values, decodes static-large-object bool and Unix delete-at headers, rejects empty names before HTTP and accepts only 200/204.
  Tests: `objectstorage/v1/objects/native_contracts_test.go:TestNativeObjectGetDownloadAndCopy`, `objectstorage/v1/objects/native_contracts_test.go:TestNativeObjectStatusesNamesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native objectstorage/v1/objects call only; Python object Resource parity and SDK-owned create, read, delete, metadata and wait workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension header/query merge; native URL construction, escaping, OkCodes, paging and header decoding are unchanged.

### `gophercloud:objectstorage/v1/objects.Download`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.Download`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithDownloadOptions`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithDownloadQuery`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithDownloadHeader`

- Download GETs the path-escaped object with range, If-Match and RFC1123 conditional time headers plus query values, returns a caller-owned body with decoded headers (closing the body on header errors) and accepts only 200/206/304.
  Tests: `objectstorage/v1/objects/native_contracts_test.go:TestNativeObjectGetDownloadAndCopy`, `objectstorage/v1/objects/native_contracts_test.go:TestNativeObjectStatusesNamesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native objectstorage/v1/objects call only; Python object Resource parity and SDK-owned create, read, delete, metadata and wait workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension header/query merge; native URL construction, escaping, OkCodes, paging and header decoding are unchanged.

### `gophercloud:objectstorage/v1/objects.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithUpdateHeader`

- Update POSTs to the path-escaped object with metadata, remove-metadata and explicit pointer empty/false/integer headers, rejects a nil option before HTTP and accepts only 201/202.
  Tests: `objectstorage/v1/objects/native_contracts_test.go:TestNativeObjectCreateUpdateDeleteHeadersAndEscaping`, `objectstorage/v1/objects/native_contracts_test.go:TestNativeObjectStatusesNamesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native objectstorage/v1/objects call only; Python object Resource parity and SDK-owned create, read, delete, metadata and wait workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension header/query merge; native URL construction, escaping, OkCodes, paging and header decoding are unchanged.

### `gophercloud:objectstorage/v1/objects.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithDeleteQuery`

- Delete DELETEs the path-escaped object with multipart-manifest/version-id query values and accepts only 202/204.
  Tests: `objectstorage/v1/objects/native_contracts_test.go:TestNativeObjectCreateUpdateDeleteHeadersAndEscaping`, `objectstorage/v1/objects/native_contracts_test.go:TestNativeObjectStatusesNamesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native objectstorage/v1/objects call only; Python object Resource parity and SDK-owned create, read, delete, metadata and wait workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension header/query merge; native URL construction, escaping, OkCodes, paging and header decoding are unchanged.

### `gophercloud:objectstorage/v1/objects.Copy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.Copy`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCopyOptions`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCopyQuery`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCopyHeader`

- Copy sends COPY to the path-escaped object with a required Destination that must start with a slash and contain a container/object separator, path-escaping each of its first three split segments, sends version-id query values and accepts only 201.
  Tests: `objectstorage/v1/objects/native_contracts_test.go:TestNativeObjectGetDownloadAndCopy`, `objectstorage/v1/objects/native_contracts_test.go:TestNativeObjectStatusesNamesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native objectstorage/v1/objects call only; Python object Resource parity and SDK-owned create, read, delete, metadata and wait workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension header/query merge; native URL construction, escaping, OkCodes, paging and header decoding are unchanged.

### `gophercloud:objectstorage/v1/objects.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.List`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithListQuery`

- List GETs the path-escaped container with JSON Accept/Content-Type headers, ListOpts (versions, delimiter) and extension query values, repeats with marker set to the last name or subdir until an empty page, and rejects a slash container name.
  Tests: `objectstorage/v1/objects/native_contracts_test.go:TestNativeObjectListAndBulkDelete`, `objectstorage/v1/objects/native_contracts_test.go:TestNativeObjectStatusesNamesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native objectstorage/v1/objects call only; Python object Resource parity and SDK-owned create, read, delete, metadata and wait workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension header/query merge; native URL construction, escaping, OkCodes, paging and header decoding are unchanged.

### `gophercloud:objectstorage/v1/objects.BulkDelete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.BulkDelete`

- BulkDelete POSTs escaped {container}/{object} lines as text/plain to the account Endpoint with ?bulk-delete=true, rejects a bad container or any empty object name before HTTP, decodes the counts and accepts only 200.
  Tests: `objectstorage/v1/objects/native_contracts_test.go:TestNativeObjectListAndBulkDelete`, `objectstorage/v1/objects/native_contracts_test.go:TestNativeObjectStatusesNamesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native objectstorage/v1/objects call only; Python object Resource parity and SDK-owned create, read, delete, metadata and wait workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension header/query merge; native URL construction, escaping, OkCodes, paging and header decoding are unchanged.

### `gophercloud:objectstorage/v1/objects.CreateTempURL`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.CreateTempURL`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCreateTempURLOptions`

- CreateTempURL signs {method}\n{expiry}\n{unescaped path after Split} with HMAC SHA1 (default), SHA256 or SHA512 using the explicit key without HTTP or else the container then account temp URL key via HEAD, returns the escaped object URL with temp_url_sig and temp_url_expires, and returns native ErrTempURLKeyNotFound, ErrTempURLDigestNotValid and split errors unwrapped (only option errors carry operation context).
  Tests: `objectstorage/v1/objects/native_contracts_test.go:TestNativeObjectCreateTempURL`.

Python/Go differences:

- This declaration maps the Gophercloud native objectstorage/v1/objects CreateTempURL call only; the SDK-owned GenerateTempURL and signing workflows are tracked separately.
- The generated facade wraps only option errors; key lookup, digest and split errors are returned as native errors without OperationError.

## References

- [accounts](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/objectstorage/v1/accounts/requests.go)
- [containers](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/objectstorage/v1/containers/requests.go)
- [objects](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/objectstorage/v1/objects/requests.go)
- [Reference](README.md)
- [Create Defaults Python](objects/create-defaults-python.md)
