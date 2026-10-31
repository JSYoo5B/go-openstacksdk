# Directory Marker

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_object_store/ObjectStoreCloudMixin/create_directory_marker_object`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.CreateDirectoryMarkerObject`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithDirectoryMarkerOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithDirectoryMarkerHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithDirectoryMarkerHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithDirectoryMarkerMetadata`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithDirectoryMarkerMetadataValue`

- One logical zero-byte PUT uses escaped literal container/object under captured ResourceBase, preserves leading/repeated/trailing slashes in the object name and forces one application/directory value over valid caller/source media. Actual201/202 returns Source=bytes, Mode=ordinary, Size0, empty MD5/SHA256 and only Ordinary phase;404 remains rejected with no acknowledgement or container creation.
  Tests: `objectstorage/v1/objects/directory_marker_core_test.go:TestDirectoryMarkerCoreWireAndMetadata`, `objectstorage/v1/objects/directory_marker_contracts_test.go:TestDirectoryMarkerContractsLiteralWireAndRawAcknowledgements`.
- Headers and suffix-only metadata factories and every callback own map snapshots. Later caller or retained callback-map mutation cannot alter prepared request headers; literal metadata and present empty values survive. Acknowledgement and attempt header/body snapshots remain independent.
  Tests: `objectstorage/v1/objects/directory_marker_core_test.go:TestDirectoryMarkerCoreWireAndMetadata`, `objectstorage/v1/objects/directory_marker_options_test.go:TestDirectoryMarkerOptionsFactoryAndCallbackSnapshots`, `objectstorage/v1/objects/directory_marker_contracts_test.go:TestDirectoryMarkerContractsPreflightAndSnapshots`.
- Full DirectoryMarkerOpts replaces earlier header/metadata settings and an empty full value resets them. Header and metadata helpers apply canonical case-insensitive overlays while the final directory media remains owned.
  Tests: `objectstorage/v1/objects/directory_marker_options_test.go:TestDirectoryMarkerOptionsReplacementAndOverlay`.
- Nil options, invalid targets, duplicate header/metadata aliases, malformed media, raw metadata/manifest/framing/auth headers and X-Detect-Content-Type fail before HTTP. Invalid original source or canceled context stops before option callbacks; callback/source/context causes remain jointly inspectable.
  Tests: `objectstorage/v1/objects/directory_marker_options_test.go:TestDirectoryMarkerOptionsPreflightAndJoinedCauses`, `objectstorage/v1/objects/directory_marker_contracts_test.go:TestDirectoryMarkerContractsPreflightAndSnapshots`.
- Shared API and reusable option factories keep per-call map/callback state and independent acknowledgement/physical response proof across parallel marker calls.
  Tests: `objectstorage/v1/objects/directory_marker_options_test.go:TestDirectoryMarkerOptionsParallelReuse`.
- Native503 retry,429 backoff,401 reauth and same-target307 redirect replay the owned empty marker within one Ordinary phase, retaining all actual physical responses and recovered causes. Options prepare once, captured ordinary headers remain stable, live auth refreshes and native callback mutation cannot rewrite retained rejected proof.
  Tests: `objectstorage/v1/objects/directory_marker_core_test.go:TestDirectoryMarkerCoreNativeReplay`, `objectstorage/v1/objects/directory_marker_contracts_test.go:TestDirectoryMarkerContractsNativeReplayAndPhysicalEvidence`.
- Native callbacks cannot change/remove/alias owned Content-Type or metadata, inject raw/JSON data, mutate the raw replay carrier, broaden accepted codes or add X-Detect-Content-Type. These faults stop before another physical send with the actual rejected response and original/native guard causes.
  Tests: `objectstorage/v1/objects/directory_marker_core_test.go:TestDirectoryMarkerCoreNativeOwnership`, `objectstorage/v1/objects/directory_marker_contracts_test.go:TestDirectoryMarkerContractsNativeMediaAndDetectGuards`.
- Raw redirect requests must retain one owned media value and exact empty framing. Media removal/change/alias/multiple values, detect flags, injected body, length and transfer changes stop before the redirected send and preserve the real307 proof plus callback/guard causes.
  Tests: `objectstorage/v1/objects/directory_marker_core_test.go:TestDirectoryMarkerCoreRedirectOwnership`, `objectstorage/v1/objects/directory_marker_contracts_test.go:TestDirectoryMarkerContractsNativeMediaAndDetectGuards`.
- Accepted201/202 read/Close faults retain raw acknowledgement and response-error proof without retrying; bodies close once. Context, transient Endpoint and detect-source faults stay observed even after restoration at Close. Acknowledgement, physical attempt and error header/body evidence do not alias.
  Tests: `objectstorage/v1/objects/directory_marker_core_test.go:TestDirectoryMarkerCoreAcceptedFaultProof`, `objectstorage/v1/objects/directory_marker_contracts_test.go:TestDirectoryMarkerContractsAcceptedFaultsAndSourceGuards`.
- An actual response returned together with transport error retains its physical status/header observation without fabricating acknowledgement or consumed body. No cleanup phase is added.
  Tests: `objectstorage/v1/objects/directory_marker_contracts_test.go:TestDirectoryMarkerContractsAcceptedFaultsAndSourceGuards`.
- Marker source X-Detect-Content-Type rejects before callbacks, while the existing ordinary CreateObject still accepts its separate valid media/detect/metadata policy.
  Tests: `objectstorage/v1/objects/directory_marker_core_test.go:TestDirectoryMarkerCoreOrdinaryPolicyIsolation`.
- Connection ObjectStorage/ObjectStorageV1 shares the original service client and marker facade; each workflow freshly captures ordinary source headers and auth. Reused factory maps stay owned,201/202 are returned as raw acknowledgements and result snapshots do not mutate the source or another observation.
  Tests: `connectiontest/connection_objectstorage_directory_marker_test.go:TestConnectionSwiftDirectoryMarkerOwnsMediaAndFreshCapture`.
- Swift generated native/service files remain byte-exact, accounts/containers/objects gain no generated Collections and generated guidance includes the dedicated marker method/media/guide together with existing CreateObject, WaitForDelete and DeleteObject.
  Tests: `internal/cmd/sdkgen/swift_directory_marker_test.go:TestSwiftDirectoryMarkerPreservesNativeAPIs`.
- CreateDirectoryMarkerObject sends one PUT with an empty body and Content-Type application/directory, the request the Python cloud helper sends through create_object(data='').
  Tests: `objectstorage/v1/objects/python_rejudge_test.go:TestPythonRejudgeObjectDirectoryMarkerRequest`.

Python/Go differences:

- Pinned cloud marker sets lowercase content-type=application/directory and delegates empty string data with generate_checksums=False through cloud/canonical create_object to one raw Object PUT. Go owns explicit empty bytes at a fixed literal route and returns existing CreateObjectResult Source=bytes/Mode=ordinary/Size0 and Ordinary raw physical-attempt evidence rather than mutable Object.
- Neither actual marker/data call chain creates a missing container despite the shared create_object docstring claim, nor normalizes name suffix, hashes, reads file, probes HEAD/info/stale, segments or cleans up. Go exposes only concrete Headers/Metadata options instead of Python kwargs that may bind real create controls or conflict with explicit data/checksum arguments.
- Go validates canonical headers and suffix-only metadata snapshots before forcing application/directory. Python Resource case aliases depend on insertion order, unknown kwargs are not universal HTTP headers, and BaseResource system metadata can overwrite prepared Content-Type. Go strict maps deliberately differ from descriptor coercion/alias semantics.
- Marker Content-Type remains owned across native callbacks and physical wire requests; X-Detect-Content-Type is rejected at options/source/native/raw boundaries because pinned Swift can otherwise ignore explicit media and guess from the route. This marker-specific policy leaves ordinary CreateObject media behavior separate.
- Go preserves native physical retry/backoff/reauth/redirect evidence within one logical PUT and retains independent acknowledgement/attempt/error response snapshots.201/202 are raw acknowledgements, and accepted processing/source/context faults can return both evidence and error; no cluster/staticweb completion result is invented.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "errors"
    "fmt"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func showMarker(result *objects.CreateObjectResult, err error) {
    if result != nil {
        fmt.Printf("source=%s mode=%s size=%d md5=%q sha256=%q\n",
            result.Source, result.Mode, result.Size, result.MD5, result.SHA256)
        if result.Ordinary != nil {
            for _, attempt := range result.Ordinary.Attempts {
                fmt.Printf("logical=%d physical=%d error=%v\n",
                    attempt.LogicalAttempt, attempt.PhysicalAttempt, attempt.Error)
                if attempt.Response != nil {
                    fmt.Printf("HTTP%d headers=%v body=%q\n",
                        attempt.Response.StatusCode, attempt.Response.Header, attempt.Response.Body)
                }
            }
            if result.Ordinary.Acknowledgement != nil {
                fmt.Println("acknowledged HTTP", result.Ordinary.Acknowledgement.StatusCode)
            }
        }
    }
    if err != nil {
        fmt.Println(err)
        var proof *resource.ResponseError
        if errors.As(err, &proof) {
            fmt.Printf("error proof HTTP%d headers=%v body=%q\n",
                proof.StatusCode, proof.Header, proof.Body)
        }
    }
}

func markerExample(ctx context.Context, conn *sdk.Connection) {
    service, err := conn.ObjectStorageV1(ctx)
    if err != nil {
        showMarker(nil, err)
        return
    }
    options := []objects.DirectoryMarkerOption{
        objects.WithDirectoryMarkerOpts(objects.DirectoryMarkerOpts{
            Headers: map[string]string{"X-Client": "directory-marker"},
            Metadata: map[string]string{"Owner": "sdk"},
        }),
        objects.WithDirectoryMarkerHeader("X-Trace", "initial"),
        objects.WithDirectoryMarkerHeaders(map[string]string{"x-trace": "marker"}),
        objects.WithDirectoryMarkerMetadata(map[string]string{"Purpose": "web-listing"}),
        objects.WithDirectoryMarkerMetadataValue("Build", "2026-10-04"),
    }
    result, err := service.Objects.CreateDirectoryMarkerObject(ctx, "backups", "reports/", options...)
    showMarker(result, err)
}

func main() {}
```

## References

- [pinned cloud helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_object_store.py#L231-L258)
- [cloud create_object](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_object_store.py#L260-L321)
- [canonical proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L398-L526)
- [Object.create](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/obj.py#L359-L372)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L917-L950)
- [ Base](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_base.py#L48-L66)
- [Obj](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/proxy/controllers/obj.py#L708-L717)
