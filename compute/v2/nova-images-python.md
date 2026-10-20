# Nova Images Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/get_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/images.API.GetRecord`

- GET /images/{escaped literal ID}; singular image unpack or root JSON object. Missing id retains requested ID; blank or malformed JSON syntax preserves Source seed; valid nonobject representation is an accepted-response error with HTTP proof.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageGetProjectionAliasesAndStatuses`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`, `compute/v2/images/image_records_test.go:TestPythonNovaImageRejectedResponseFailuresStopDeleteAndFind`, `compute/v2/images/image_records_test.go:TestPythonNovaImageCleanRejectionsRemainEligibleAfterNativeRetryHook`.
- Image declared Body view preserves untyped JSON and exact integers; response alias dictionary order, four Python integer descriptors and dictionary metadata descriptor project into 12 fields, including null location. Wire/Envelope/Header/Status preserve separately owned actual evidence.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageGetProjectionAliasesAndStatuses`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`, `compute/v2/images/image_records_test.go:TestPythonNovaImageRejectedResponseFailuresStopDeleteAndFind`, `compute/v2/images/image_records_test.go:TestPythonNovaImageCleanRejectionsRemainEligibleAfterNativeRetryHook`.
- GET400/403/404 and Delete404 observe rejected Read/Close/context/source faults through shared REST; only clean rejection may establish ignoreMissing/fallback. Identical native retry stop-hook return remains a clean status; mixed retry errors preserve causes and stop after one physical request.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageGetProjectionAliasesAndStatuses`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`, `compute/v2/images/image_records_test.go:TestPythonNovaImageRejectedResponseFailuresStopDeleteAndFind`, `compute/v2/images/image_records_test.go:TestPythonNovaImageCleanRejectionsRemainEligibleAfterNativeRetryHook`.

Python/Go differences:

- Explicit legacy Nova selected-version profile, not unrestricted modern Nova coverage: ordinary image methods accept 2.1..2.35; metadata methods accept 2.1..2.38. Empty selection means Nova default2.1; selected version is never downgraded or discovered. Pinned Image has inherited _max_microversion=None and Source default selection is not capped; Go preflight rejects unsupported selections before HTTP.
- Literal valid UTF-8 string IDs/keys, immutable owned records and null location replace Python mutable same-instance Resource/Munch/cache/Connection/session/subclass/warnings. Full Resource/session parity is a separate surface outside these callable Go profiles.
- Library guards protect selected compute source facts and headers; typed nonnegative list controls and fixed continuation origin/path/query are Go profile choices. Arbitrary Python session/path/version hooks and JMESPath surface are not provided.

### `python:compute/v2/find_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/images.API.FindRecord`

- Literal member GET first; only clean 400/403/404 allow fallback GET /images?name=... (summary, not detail). Complete exact ID/name search includes later pages; duplicate/missing/late errors retain shared FindIdentity policy.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageFindSummaryAndDeleteMissing`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`, `compute/v2/images/image_records_test.go:TestPythonNovaImageRejectedResponseFailuresStopDeleteAndFind`, `compute/v2/images/image_records_test.go:TestPythonNovaImageCleanRejectionsRemainEligibleAfterNativeRetryHook`.
- IgnoreMissing defaults true; false returns NotFound; successful member GET avoids list. Source find_image formal options have no details/query, which is reflected in concrete FindOpts.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageFindSummaryAndDeleteMissing`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`, `compute/v2/images/image_records_test.go:TestPythonNovaImageRejectedResponseFailuresStopDeleteAndFind`, `compute/v2/images/image_records_test.go:TestPythonNovaImageCleanRejectionsRemainEligibleAfterNativeRetryHook`.
- GET400/403/404 and Delete404 observe rejected Read/Close/context/source faults through shared REST; only clean rejection may establish ignoreMissing/fallback. Identical native retry stop-hook return remains a clean status; mixed retry errors preserve causes and stop after one physical request.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageFindSummaryAndDeleteMissing`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`, `compute/v2/images/image_records_test.go:TestPythonNovaImageRejectedResponseFailuresStopDeleteAndFind`, `compute/v2/images/image_records_test.go:TestPythonNovaImageCleanRejectionsRemainEligibleAfterNativeRetryHook`.

Python/Go differences:

- Explicit legacy Nova selected-version profile, not unrestricted modern Nova coverage: ordinary image methods accept 2.1..2.35; metadata methods accept 2.1..2.38. Empty selection means Nova default2.1; selected version is never downgraded or discovered. Pinned Image has inherited _max_microversion=None and Source default selection is not capped; Go preflight rejects unsupported selections before HTTP.
- Literal valid UTF-8 string IDs/keys, immutable owned records and null location replace Python mutable same-instance Resource/Munch/cache/Connection/session/subclass/warnings. Full Resource/session parity is a separate surface outside these callable Go profiles.
- Library guards protect selected compute source facts and headers; typed nonnegative list controls and fixed continuation origin/path/query are Go profile choices. Arbitrary Python session/path/version hooks and JMESPath surface are not provided.
- Corrects stale remaining text: pinned find_image fallback is /images, not /images/detail.

### `python:compute/v2/images`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/images.API.ListRecords`

- Lazy reusable GET /images/detail by default and /images when Details=false; server/name/status/type/min_disk/min_ram/changes_since map to exact Source wire query, with client alias precedence. Shared Requests JSON encoder handles null/repeated arrays/object keys/True spelling; other declared Body attrs filter locally; unknown kwargs ignored.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageListQueryDetailLocalFiltersAndCaps`, `compute/v2/images/image_records_test.go:TestPythonNovaImageListPagingAndWireMarker`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`.
- Raw-row MaxItems precedes local filtering and supplies absent/zero limit hint; typed limit/marker/paginated controls, plural singleton, array/top-level/HTTP next, short-page marker fallback with initial positive limit, early break and fixed route proof use shared REST. Fallback marker comes from original raw row independent of consumer mutations.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageListQueryDetailLocalFiltersAndCaps`, `compute/v2/images/image_records_test.go:TestPythonNovaImageListPagingAndWireMarker`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`.

Python/Go differences:

- Explicit legacy Nova selected-version profile, not unrestricted modern Nova coverage: ordinary image methods accept 2.1..2.35; metadata methods accept 2.1..2.38. Empty selection means Nova default2.1; selected version is never downgraded or discovered. Pinned Image has inherited _max_microversion=None and Source default selection is not capped; Go preflight rejects unsupported selections before HTTP.
- Literal valid UTF-8 string IDs/keys, immutable owned records and null location replace Python mutable same-instance Resource/Munch/cache/Connection/session/subclass/warnings. Full Resource/session parity is a separate surface outside these callable Go profiles.
- Library guards protect selected compute source facts and headers; typed nonnegative list controls and fixed continuation origin/path/query are Go profile choices. Arbitrary Python session/path/version hooks and JMESPath surface are not provided.
- Dictionary links:{next:...} is a Go REST extension; pinned generic Resource does not consume dictionary links.
- Local JSON filter retains exact JSON types and nested subset instead of Python bool/number equality and nonobject nested-filter exception behavior; server-spelling aliases that are not actual Body attrs do not become local filters.

### `python:compute/v2/delete_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/images.API.DeleteRecord`

- DELETE /images/{escaped literal ID}; no JSON body decode. Actual 200..399 accepted and clean404 ignored by default; WithDeleteIgnoreMissing(false) retains original error. Unsupported microversion guard runs before HTTP and ignoreMissing.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageFindSummaryAndDeleteMissing`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`, `compute/v2/images/image_records_test.go:TestPythonNovaImageRejectedResponseFailuresStopDeleteAndFind`, `compute/v2/images/image_records_test.go:TestPythonNovaImageCleanRejectionsRemainEligibleAfterNativeRetryHook`.
- GET400/403/404 and Delete404 observe rejected Read/Close/context/source faults through shared REST; only clean rejection may establish ignoreMissing/fallback. Identical native retry stop-hook return remains a clean status; mixed retry errors preserve causes and stop after one physical request.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageFindSummaryAndDeleteMissing`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`, `compute/v2/images/image_records_test.go:TestPythonNovaImageRejectedResponseFailuresStopDeleteAndFind`, `compute/v2/images/image_records_test.go:TestPythonNovaImageCleanRejectionsRemainEligibleAfterNativeRetryHook`.

Python/Go differences:

- Explicit legacy Nova selected-version profile, not unrestricted modern Nova coverage: ordinary image methods accept 2.1..2.35; metadata methods accept 2.1..2.38. Empty selection means Nova default2.1; selected version is never downgraded or discovered. Pinned Image has inherited _max_microversion=None and Source default selection is not capped; Go preflight rejects unsupported selections before HTTP.
- Literal valid UTF-8 string IDs/keys, immutable owned records and null location replace Python mutable same-instance Resource/Munch/cache/Connection/session/subclass/warnings. Full Resource/session parity is a separate surface outside these callable Go profiles.
- Library guards protect selected compute source facts and headers; typed nonnegative list controls and fixed continuation origin/path/query are Go profile choices. Arbitrary Python session/path/version hooks and JMESPath surface are not provided.

### `python:compute/v2/fetch_image_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/images.API.FetchMetadata`

- GET /images/{id}/metadata, using selected version through2.38; strict JSON object parse (no malformed/empty-body tolerance). Only metadata overlays seed; body id/unknown fields retained in Wire but do not override resource ID. Missing metadata stays null, dictionary descriptor projects nonobject metadata to{}.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageMetadataFetchAliasSetDeleteContracts`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`.
- Metadata view contains the literal image ID plus submitted/received metadata; other declared fields null, without a parent image GET. All successful statuses200..399 accepted and actual response evidence retained independently.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageMetadataFetchAliasSetDeleteContracts`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`.

Python/Go differences:

- Explicit legacy Nova selected-version profile, not unrestricted modern Nova coverage: ordinary image methods accept 2.1..2.35; metadata methods accept 2.1..2.38. Empty selection means Nova default2.1; selected version is never downgraded or discovered. Pinned Image has inherited _max_microversion=None and Source default selection is not capped; Go preflight rejects unsupported selections before HTTP.
- Literal valid UTF-8 string IDs/keys, immutable owned records and null location replace Python mutable same-instance Resource/Munch/cache/Connection/session/subclass/warnings. Full Resource/session parity is a separate surface outside these callable Go profiles.
- Library guards protect selected compute source facts and headers; typed nonnegative list controls and fixed continuation origin/path/query are Go profile choices. Arbitrary Python session/path/version hooks and JMESPath surface are not provided.
- Fetch requires nonnull JSON object rather than reproducing Python array/scalar membership behavior.

### `python:compute/v2/get_image_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/images.API.GetMetadata`

- Pinned deprecated alias of fetch_image_metadata; identical route/body/status/metadata projection and strict JSON parse.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageMetadataFetchAliasSetDeleteContracts`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`.
- Metadata view contains the literal image ID plus submitted/received metadata; other declared fields null, without a parent image GET. All successful statuses200..399 accepted and actual response evidence retained independently.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageMetadataFetchAliasSetDeleteContracts`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`.

Python/Go differences:

- Explicit legacy Nova selected-version profile, not unrestricted modern Nova coverage: ordinary image methods accept 2.1..2.35; metadata methods accept 2.1..2.38. Empty selection means Nova default2.1; selected version is never downgraded or discovered. Pinned Image has inherited _max_microversion=None and Source default selection is not capped; Go preflight rejects unsupported selections before HTTP.
- Literal valid UTF-8 string IDs/keys, immutable owned records and null location replace Python mutable same-instance Resource/Munch/cache/Connection/session/subclass/warnings. Full Resource/session parity is a separate surface outside these callable Go profiles.
- Library guards protect selected compute source facts and headers; typed nonnegative list controls and fixed continuation origin/path/query are Go profile choices. Arbitrary Python session/path/version hooks and JMESPath surface are not provided.
- Source deprecation warnings have documentation-only Go representation; fetch object-domain difference applies equally.

### `python:compute/v2/set_image_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/images.API.SetMetadata`

- POST /images/{id}/metadata with metadata envelope; nil input means{}. JSON snapshot before option callbacks owns nested data and exact numbers; no parent GET or native builder coercion.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageMetadataFetchAliasSetDeleteContracts`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`.
- Source ignores response JSON and returns submitted metadata; Go Resource uses submitted owned map, Wire=nil, and Envelope/Header/Status capture opaque actual reply. Accepted read/close/guard errors preserve real proof.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageMetadataFetchAliasSetDeleteContracts`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`.
- Metadata view contains the literal image ID plus submitted/received metadata; other declared fields null, without a parent image GET. All successful statuses200..399 accepted and actual response evidence retained independently.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageMetadataFetchAliasSetDeleteContracts`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`.

Python/Go differences:

- Explicit legacy Nova selected-version profile, not unrestricted modern Nova coverage: ordinary image methods accept 2.1..2.35; metadata methods accept 2.1..2.38. Empty selection means Nova default2.1; selected version is never downgraded or discovered. Pinned Image has inherited _max_microversion=None and Source default selection is not capped; Go preflight rejects unsupported selections before HTTP.
- Literal valid UTF-8 string IDs/keys, immutable owned records and null location replace Python mutable same-instance Resource/Munch/cache/Connection/session/subclass/warnings. Full Resource/session parity is a separate surface outside these callable Go profiles.
- Library guards protect selected compute source facts and headers; typed nonnegative list controls and fixed continuation origin/path/query are Go profile choices. Arbitrary Python session/path/version hooks and JMESPath surface are not provided.

### `python:compute/v2/delete_image_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/images.API.DeleteMetadata`

- Omitted Keys means PUT {metadata:{}}; explicit empty key slice means zero physical requests. Owned keys are validated before mutations, deduplicated, escaped once and deleted individually; strict404 and first error/cancel/source change stop later keys.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageMetadataFetchAliasSetDeleteContracts`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`.
- Metadata route ceiling2.38 applied before HTTP, source guarded across each physical request, successful opaque bodies never decoded.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageMetadataFetchAliasSetDeleteContracts`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`.

Python/Go differences:

- Explicit legacy Nova selected-version profile, not unrestricted modern Nova coverage: ordinary image methods accept 2.1..2.35; metadata methods accept 2.1..2.38. Empty selection means Nova default2.1; selected version is never downgraded or discovered. Pinned Image has inherited _max_microversion=None and Source default selection is not capped; Go preflight rejects unsupported selections before HTTP.
- Literal valid UTF-8 string IDs/keys, immutable owned records and null location replace Python mutable same-instance Resource/Munch/cache/Connection/session/subclass/warnings. Full Resource/session parity is a separate surface outside these callable Go profiles.
- Library guards protect selected compute source facts and headers; typed nonnegative list controls and fixed continuation origin/path/query are Go profile choices. Arbitrary Python session/path/version hooks and JMESPath surface are not provided.
- Python set(keys) iteration order is unspecified; Go uses first-seen deterministic order.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
images = conn.compute.images(details=False, min_disk=4, max_items=100)
image = conn.compute.get_image(image_id)
found = conn.compute.find_image(image_id)
metadata = conn.compute.fetch_image_metadata(image_id)
conn.compute.set_image_metadata(image_id, owner="ci")
conn.compute.delete_image_metadata(image_id, keys=["owner"])
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "strings"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/images"
)

func printImage(value *images.ImageRecord) {
    if value == nil { fmt.Println("null"); return }
    data, err := json.MarshalIndent(value.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    id := flag.String("id", "", "Nova legacy image ID")
    owner := flag.String("set-owner", "", "metadata owner to POST")
    keys := flag.String("delete-keys", "", "comma-separated metadata keys")
    clear := flag.Bool("clear-metadata", false, "replace metadata with empty object")
    remove := flag.Bool("delete-image", false, "delete image strictly")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud), sdk.WithMicroversion(sdk.Compute, "2.35"))
    if err != nil { log.Fatal(err) }
    service, err := conn.ComputeV2(ctx)
    if err != nil { log.Fatal(err) }
    api := service.Images
    for value, err := range api.ListRecords(ctx,
        images.WithListDetails(false), images.WithListMaxItems(100)) {
        if err != nil { log.Fatal(err) }
        printImage(value)
    }
    if *id == "" { return }
    value, err := api.GetRecord(ctx, *id)
    if err != nil { log.Fatal(err) }
    printImage(value)
    value, err = api.FindRecord(ctx, *id, images.WithFindIgnoreMissing(false))
    if err != nil { log.Fatal(err) }
    printImage(value)
    value, err = api.FetchMetadata(ctx, *id)
    if err != nil { log.Fatal(err) }
    printImage(value)
    if *owner != "" {
        value, err = api.SetMetadata(ctx, *id, map[string]any{"owner": *owner})
        if err != nil { log.Fatal(err) }
        printImage(value)
    }
    if *keys != "" {
        if err := api.DeleteMetadata(ctx, *id,
            images.WithDeleteMetadataKeys(strings.Split(*keys, ",")...)); err != nil { log.Fatal(err) }
    }
    if *clear {
        if err := api.DeleteMetadata(ctx, *id); err != nil { log.Fatal(err) }
    }
    if *remove {
        if err := api.DeleteRecord(ctx, *id,
            images.WithDeleteIgnoreMissing(false)); err != nil { log.Fatal(err) }
    }
}
```

## References

- [Compute](https://docs.openstack.org/api-ref/compute/#images-deprecated)
- [Routes](https://github.com/openstack/nova/blob/master/nova/api/openstack/compute/routes.py)
- [API Version Request](https://github.com/openstack/nova/blob/master/nova/api/openstack/api_version_request.py)
- [Install](../../docs/install.md)
- [Image Records Test](images/image_records_test.go)
