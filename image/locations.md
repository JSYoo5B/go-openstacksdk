# Locations

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:image/v2/add_image_location`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.AddImageLocation`
- `github.com/JSYoo5B/go-openstacksdk/image.WithAddImageLocationOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageLocationValidation`
- `github.com/JSYoo5B/go-openstacksdk/image.WithAddImageLocationHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithAddImageLocationHeaders`

- Add sends one prepared flat root JSON url plus default validation_data:{} to fixed POST images/{chosenID}/locations with no query. URL is nonempty validUTF8 literal JSON data, never parsed/normalized/fetched or used as client target; scheme/case/whitespace/percent/query/fragment remain unchanged. Supplied validation pointer requires both nonempty validUTF8 strings; no local algorithm/hex/digest whitelist or normalization/verification.
  Tests: `image/locations_contracts_test.go:TestImageLocationsFixedRoutesAndPassiveData`, `image/locations_core_test.go:TestImageLocationsCoreFixedRoutesAndPassiveData`, `image/locations_contracts_test.go:TestImageLocationsCanonicalModelsAndPreflight`, `image/locations_core_test.go:TestImageLocationsCoreStrictArrayAndCanonicalSchema`, `image/locations_core_test.go:TestImageLocationsCoreCompletePreflight`.
- Like pinned Resource.create raise_from_response, every actual 200..399 response establishes the Add acknowledgement with fixed request ImageID/URL and independently cloned opaque Body/Header/status; 400 and above keep native errors whose Expected list is 200..399, and expanded OkCodes cannot widen it.
  Tests: `image/locations_contracts_test.go:TestImageLocationsFixedRoutesAndPassiveData`, `image/locations_contracts_test.go:TestImageLocationsAcceptedFailuresAndStatusPolicy`, `image/locations_core_test.go:TestImageLocationsCoreStrictStatusesAndRetryOwnership`, `image/locations_python_view_test.go:TestImageLocationPythonViewSeedAcrossAcceptedStatuses`.
- Accepted202 read/Close/context failures retain partial/full result together with independent ResponseError evidence and all original read/Close/ctx.Err/custom causes. Body closes once, no accepted replay or follow-up GET/task/wait. Native rejected/transport/prebody failures remain nil result with original evidence; missing is strict.
  Tests: `image/locations_contracts_test.go:TestImageLocationsAcceptedFailuresAndStatusPolicy`, `image/locations_core_test.go:TestImageLocationsCoreAcceptedFailuresKeepOwnedProof`.
- Source/client/provider/base/ordinary headers/context and Ref are validated and captured before caller callbacks. Direct ID causes only the requested location endpoint; explicit exact Name uses existing native all-pages resolver once and rejects absent/ambiguous/late surfaced native lookup failures or unsafe resolved IDs before location HTTP. Native Name model/body/continuation policy remains unchanged.
  Tests: `image/locations_contracts_test.go:TestImageLocationsPreparedOptionsAndExactNames`, `image/locations_core_test.go:TestImageLocationsCoreExactNameAndPreparedPayload`, `image/locations_options_test.go:TestImageLocationsOptionsSnapshotsAndReplacement`, `image/locations_options_test.go:TestImageLocationsOptionsOwnCallbackHandlesAndSlice`, `image/locations_options_test.go:TestImageLocationsOptionsRejectOwnedHeadersAndAliases`, `image/locations_options_test.go:TestImageLocationsOptionsParallelReusableHelpers`, `image/locations_contracts_test.go:TestImageLocationsCanonicalModelsAndPreflight`, `image/locations_core_test.go:TestImageLocationsCoreStrictArrayAndCanonicalSchema`, `image/locations_core_test.go:TestImageLocationsCoreCompletePreflight`.
- Concrete option helpers snapshot caller maps and validation pointers at creation/application, own option slices and each callback config, invoke once and copy again. Full replacement and ordinary canonical header merge preserve prepared policy across retained handles and parallel helper reuse. Protected headers and nil/error callbacks fail before HTTP with original causes.
  Tests: `image/locations_contracts_test.go:TestImageLocationsPreparedOptionsAndExactNames`, `image/locations_core_test.go:TestImageLocationsCoreExactNameAndPreparedPayload`, `image/locations_options_test.go:TestImageLocationsOptionsSnapshotsAndReplacement`, `image/locations_options_test.go:TestImageLocationsOptionsOwnCallbackHandlesAndSlice`, `image/locations_options_test.go:TestImageLocationsOptionsRejectOwnedHeadersAndAliases`, `image/locations_options_test.go:TestImageLocationsOptionsParallelReusableHelpers`, `image/locations_contracts_test.go:TestImageLocationsCanonicalModelsAndPreflight`, `image/locations_core_test.go:TestImageLocationsCoreStrictArrayAndCanonicalSchema`, `image/locations_core_test.go:TestImageLocationsCoreCompletePreflight`.
- Captured original prefix/ordinary headers/body and original provider identity are fixed while latest original token remains live. Safe source changes retain captured requests; provider replacement/foreign retarget rejects. Shared prebody reauth/retry/backoff and identical-target redirects remain configured native policy; changed method/origin/path/query is guarded.
  Tests: `image/locations_contracts_test.go:TestImageLocationsPreparedOptionsAndExactNames`, `image/locations_core_test.go:TestImageLocationsCoreExactNameAndPreparedPayload`, `image/locations_options_test.go:TestImageLocationsOptionsSnapshotsAndReplacement`, `image/locations_options_test.go:TestImageLocationsOptionsOwnCallbackHandlesAndSlice`, `image/locations_options_test.go:TestImageLocationsOptionsRejectOwnedHeadersAndAliases`, `image/locations_options_test.go:TestImageLocationsOptionsParallelReusableHelpers`, `image/locations_contracts_test.go:TestImageLocationsProviderHooksAndExistingCompatibility`, `image/locations_core_test.go:TestImageLocationsCoreStrictStatusesAndRetryOwnership`, `connectiontest/connection_image_locations_test.go:TestConnectionImageLocationsShareNativeClientAndPreservePassiveData`, `internal/cmd/sdkgen/glance_locations_test.go:TestGlanceLocationsPreserveNativeImagePatchAndDocumentDedicatedEndpoints`.
- Shared DoJSON preserves original serialized Add body or nil Get body and response ownership. Equivalent serialized body replacement is normalized to the original; changed URL/hash, in-place mutation, RawBody/KeepResponseBody/JSONResponse or nil-to-JSONnull/unserializable body rejects before another request with original/hook/encoding causes. Expanded OkCodes cannot override strict original202/200. Native reauth ErrOriginal/ErrReauth remain inspectable via fields; prior image/native workflows remain unchanged.
  Tests: `image/locations_contracts_test.go:TestImageLocationsProviderHooksAndExistingCompatibility`, `image/locations_core_test.go:TestImageLocationsCoreStrictStatusesAndRetryOwnership`, `connectiontest/connection_image_locations_test.go:TestConnectionImageLocationsShareNativeClientAndPreservePassiveData`, `internal/cmd/sdkgen/glance_locations_test.go:TestGlanceLocationsPreserveNativeImagePatchAndDocumentDedicatedEndpoints`, `image/locations_contracts_test.go:TestImageLocationsAcceptedFailuresAndStatusPolicy`, `image/locations_core_test.go:TestImageLocationsCoreAcceptedFailuresKeepOwnedProof`.
- Resource is the Python ImageLocation view: seeded image_id/url/validation_data (default {}), null id/name/metadata and the Connection location, kept for empty or invalid JSON and overlaid only by declared id/name/url/validation_data/metadata of a JSON object response with self and other keys ignored. Valid non-object JSON, non-object dict-typed fields and location failures return the acknowledgement with a ResponseError and no view.
  Tests: `image/locations_python_view_test.go:TestImageLocationPythonViewSeedAcrossAcceptedStatuses`, `image/locations_python_view_test.go:TestImageLocationPythonViewOverlayAndFailures`.

Python/Go differences:

- Pinned Python accepts an Image Resource or ID and returns a mutable ImageLocation Resource; Go uses an explicit ID/Name Ref and concrete owned options and returns AddImageLocationResult with the acknowledgement plus an owned ImageLocation view. Explicit Name resolution is an additional convenience and SDK-wide mutable Resource/session behavior is tracked separately as SDK-R1/S1.
- Go retains native Name pager model/body/continuation and rejected native response policy while owning accepted response bytes/header/status and body failures.
- Python ImageLocation._max_microversion 2.17 is a framework capability hint. Go preserves the existing source version policy without inventing a location microversion header, capability discovery or minimum deployment support gate.
- Nil Go validation defaults to {} like Python None, but Go accepts only the paired os_hash_algo/os_hash_value strings while Python accepts any dict, and Go rejects non-object dict-typed response fields that Python dict() might coerce. Neither the view nor the acknowledgement proves a stored location or completed asynchronous validation, which the server owns.

### `python:image/v2/image_locations`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.GetImageLocations`
- `github.com/JSYoo5B/go-openstacksdk/image.WithGetImageLocationsOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithGetImageLocationsHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithGetImageLocationsHeaders`

- Get performs one bodyless/query-free fixed-parent GET accepting actual200 finite barearray only. Empty[] yields nonnil empty rows. No Link/marker/limit fallback or row ID/URL based continuation; canonical request ImageID stays fixed and returned metadata remains passive.
  Tests: `image/locations_contracts_test.go:TestImageLocationsFixedRoutesAndPassiveData`, `image/locations_core_test.go:TestImageLocationsCoreFixedRoutesAndPassiveData`.
- Every row must be a nonnull object. Canonical optional url is nil for missing/null and otherwise strictstring pointer; metadata is nil for missing/null, nonnil rawmap for{} and otherwise strictobject. Unknown/case-decoy/id/status/image_id/next and raw null/zero/false/nested/large/fraction JSON are preserved without float64 conversion or target/capability inference.
  Tests: `image/locations_contracts_test.go:TestImageLocationsCanonicalModelsAndPreflight`, `image/locations_core_test.go:TestImageLocationsCoreStrictArrayAndCanonicalSchema`, `image/locations_core_test.go:TestImageLocationsCoreCompletePreflight`, `image/locations_contracts_test.go:TestImageLocationsFixedRoutesAndPassiveData`, `image/locations_core_test.go:TestImageLocationsCoreFixedRoutesAndPassiveData`.
- GET read/Close/context/wholeUTF8/JSON/root/schema/row errors return nil typed result, with whole actual200 bytes/header/status and every original cause retained in ResponseError. No partial rows or single-object coercion; accepted body closes once without replay. Unexpected/missing statuses remain native errors without fallback.
  Tests: `image/locations_contracts_test.go:TestImageLocationsAcceptedFailuresAndStatusPolicy`, `image/locations_core_test.go:TestImageLocationsCoreAcceptedFailuresKeepOwnedProof`, `image/locations_contracts_test.go:TestImageLocationsCanonicalModelsAndPreflight`, `image/locations_core_test.go:TestImageLocationsCoreStrictArrayAndCanonicalSchema`, `image/locations_core_test.go:TestImageLocationsCoreCompletePreflight`.
- Source/client/provider/base/ordinary headers/context and Ref are validated and captured before caller callbacks. Direct ID causes only the requested location endpoint; explicit exact Name uses existing native all-pages resolver once and rejects absent/ambiguous/late surfaced native lookup failures or unsafe resolved IDs before location HTTP. Native Name model/body/continuation policy remains unchanged.
  Tests: `image/locations_contracts_test.go:TestImageLocationsPreparedOptionsAndExactNames`, `image/locations_core_test.go:TestImageLocationsCoreExactNameAndPreparedPayload`, `image/locations_options_test.go:TestImageLocationsOptionsSnapshotsAndReplacement`, `image/locations_options_test.go:TestImageLocationsOptionsOwnCallbackHandlesAndSlice`, `image/locations_options_test.go:TestImageLocationsOptionsRejectOwnedHeadersAndAliases`, `image/locations_options_test.go:TestImageLocationsOptionsParallelReusableHelpers`, `image/locations_contracts_test.go:TestImageLocationsCanonicalModelsAndPreflight`, `image/locations_core_test.go:TestImageLocationsCoreStrictArrayAndCanonicalSchema`, `image/locations_core_test.go:TestImageLocationsCoreCompletePreflight`.
- Concrete option helpers snapshot caller maps and validation pointers at creation/application, own option slices and each callback config, invoke once and copy again. Full replacement and ordinary canonical header merge preserve prepared policy across retained handles and parallel helper reuse. Protected headers and nil/error callbacks fail before HTTP with original causes.
  Tests: `image/locations_contracts_test.go:TestImageLocationsPreparedOptionsAndExactNames`, `image/locations_core_test.go:TestImageLocationsCoreExactNameAndPreparedPayload`, `image/locations_options_test.go:TestImageLocationsOptionsSnapshotsAndReplacement`, `image/locations_options_test.go:TestImageLocationsOptionsOwnCallbackHandlesAndSlice`, `image/locations_options_test.go:TestImageLocationsOptionsRejectOwnedHeadersAndAliases`, `image/locations_options_test.go:TestImageLocationsOptionsParallelReusableHelpers`, `image/locations_contracts_test.go:TestImageLocationsCanonicalModelsAndPreflight`, `image/locations_core_test.go:TestImageLocationsCoreStrictArrayAndCanonicalSchema`, `image/locations_core_test.go:TestImageLocationsCoreCompletePreflight`.
- Captured original prefix/ordinary headers/body and original provider identity are fixed while latest original token remains live. Safe source changes retain captured requests; provider replacement/foreign retarget rejects. Shared prebody reauth/retry/backoff and identical-target redirects remain configured native policy; changed method/origin/path/query is guarded.
  Tests: `image/locations_contracts_test.go:TestImageLocationsPreparedOptionsAndExactNames`, `image/locations_core_test.go:TestImageLocationsCoreExactNameAndPreparedPayload`, `image/locations_options_test.go:TestImageLocationsOptionsSnapshotsAndReplacement`, `image/locations_options_test.go:TestImageLocationsOptionsOwnCallbackHandlesAndSlice`, `image/locations_options_test.go:TestImageLocationsOptionsRejectOwnedHeadersAndAliases`, `image/locations_options_test.go:TestImageLocationsOptionsParallelReusableHelpers`, `image/locations_contracts_test.go:TestImageLocationsProviderHooksAndExistingCompatibility`, `image/locations_core_test.go:TestImageLocationsCoreStrictStatusesAndRetryOwnership`, `connectiontest/connection_image_locations_test.go:TestConnectionImageLocationsShareNativeClientAndPreservePassiveData`, `internal/cmd/sdkgen/glance_locations_test.go:TestGlanceLocationsPreserveNativeImagePatchAndDocumentDedicatedEndpoints`.
- Shared DoJSON preserves original serialized Add body or nil Get body and response ownership. Equivalent serialized body replacement is normalized to the original; changed URL/hash, in-place mutation, RawBody/KeepResponseBody/JSONResponse or nil-to-JSONnull/unserializable body rejects before another request with original/hook/encoding causes. Expanded OkCodes cannot override strict original202/200. Native reauth ErrOriginal/ErrReauth remain inspectable via fields; prior image/native workflows remain unchanged.
  Tests: `image/locations_contracts_test.go:TestImageLocationsProviderHooksAndExistingCompatibility`, `image/locations_core_test.go:TestImageLocationsCoreStrictStatusesAndRetryOwnership`, `connectiontest/connection_image_locations_test.go:TestConnectionImageLocationsShareNativeClientAndPreservePassiveData`, `internal/cmd/sdkgen/glance_locations_test.go:TestGlanceLocationsPreserveNativeImagePatchAndDocumentDedicatedEndpoints`, `image/locations_contracts_test.go:TestImageLocationsAcceptedFailuresAndStatusPolicy`, `image/locations_core_test.go:TestImageLocationsCoreAcceptedFailuresKeepOwnedProof`.
- GetImageLocations sends one GET images/{image}/locations with no query and returns each url and metadata row, as Python _list(ImageLocation) does.
  Tests: `image/python_rejudge_test.go:TestPythonRejudgeImageLocationsAndTaskDefaultRequests`.

Python/Go differences:

- Pinned Python accepts Image Resource/ID and uses inherited mutable ImageLocation Resource, descriptors, dirty/cache/session/microversion behavior. Go uses explicit ID/Name Ref, concrete owned options and actual raw response evidence; explicit Name resolution is an additional convenience.
- Go retains native Name pager model/body/continuation and rejected native response policy, while owning accepted response bytes/header/status and body failures. Full Python Resource/adapter/exception/cache/transport policy remains broader.
- Python ImageLocation._max_microversion2.17 is a framework capability hint. Go preserves existing source version policy without inventing a location microversion header, capability discovery or minimum deployment support gate.
- Pinned Python image_locations returns generic Resource.list with permissive nonarray-to-one-row coercion, URI attribute overlay, descriptors/filtering/cache and generic paging policies. Go reads the finite strict barearray endpoint once and retains unknown data without inventing stable location identity or collection CRUD.
- Actual dedicated server GET returns all locations and strips id/status; it has no request paging or show_multiple_locations condition in this handler. Legacy image metadata locations exposure and dedicated GET permission policy are separate boundaries.
- Python wraps rows in generic Resource.list and coerces a non-array body into one row; Go requires the JSON array the server returns.
- Go returns typed result values with copied raw response evidence instead of a mutable Python Resource, so there is no Resource descriptor coercion, dirty-attribute tracking or seeded-object merge.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"

    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func addDefaultLocation(ctx context.Context, service *image.Service, ref resource.Ref, locationURL string) (*image.AddImageLocationResult, error) {
    return service.AddImageLocation(ctx, ref, locationURL)
}

func addValidatedThenGetLocations(ctx context.Context, service *image.Service, ref resource.Ref, locationURL, algorithm, value string) (ack *image.AddImageLocationResult, locations *image.ImageLocationsResult, err error) {
    ack, err = service.AddImageLocation(ctx, ref, locationURL,
        image.WithAddImageLocationOpts(image.AddImageLocationOpts{
            Headers: map[string]string{"X-Request-Source": "example"},
        }),
        image.WithImageLocationValidation(algorithm, value),
        image.WithAddImageLocationHeader("X-Location-Request", "add"))
    if err != nil {
        return ack, nil, err
    }
    locations, err = service.GetImageLocations(ctx, ref,
        image.WithGetImageLocationsOpts(image.GetImageLocationsOpts{
            Headers: map[string]string{"X-Request-Source": "example"},
        }),
        image.WithGetImageLocationsHeader("X-Location-Request", "get"))
    return ack, locations, err
}
```

```python
def add_and_read_locations(conn, image_id, location_url, validation_data=None):
    created = conn.image.add_image_location(
        image_id, location_url, validation_data=validation_data
    )
    locations = list(conn.image.image_locations(image_id))
    return created, locations
```

## References

- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#add-location)
- [serializer](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L1882-L1883)
- [controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L1161-L1257)
- [Config](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/common/config.py#L414)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#get-location)
- [controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L1259-L1280)
- [Locations Contracts Test](locations_contracts_test.go)
- [core tests](locations_core_test.go)
- [option tests](locations_options_test.go)
- [Connection test](../connectiontest/connection_image_locations_test.go)
- [generator test](../internal/cmd/sdkgen/glance_locations_test.go)
