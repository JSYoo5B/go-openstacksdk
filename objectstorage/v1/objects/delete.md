# Delete

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:object_store/v1/delete_object`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.DeleteObject`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithDeleteObjectOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithDeleteObjectHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithDeleteObjectHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithDeleteObjectStaticLargeObject`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithoutDeleteObjectStaticLargeObject`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithDeleteObjectIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithoutDeleteObjectIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithDeleteObjectVersionID`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithDeleteObjectNewest`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithoutDeleteObjectNewest`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.ObjectDeleteBulkInfo.UnmarshalJSON`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.ObjectDeleteBulkError.Error`

- Automatic HEAD and DELETE use the same literal UTF8 object and effective ResourceBase with one escape, optional opaque version-id on both phases, explicit Newest values and no request body. Known true/false SLO skips HEAD.
  Tests: `objectstorage/v1/objects/delete_core_test.go:TestObjectDeleteCoreWorkflowAndAcknowledgements`, `objectstorage/v1/objects/delete_options_test.go:TestObjectDeleteOptionsSnapshots`, `objectstorage/v1/objects/delete_contracts_test.go:TestObjectDeleteContractsRoutesAndSelection`, `connectiontest/connection_objectstorage_object_delete_test.go:TestConnectionObjectDeleteSharesClientAndOwnsPhases`.
- Absent HEAD flag means ordinary deletion; one OWS-trimmed true/false/1/0 value selects SLO. Empty, malformed or duplicate flags fail with HEAD proof and no DELETE. HEAD200/204 and DELETE202/204 retain actual raw acknowledgements; SLO202/204 has nil Bulk and unknown counts.
  Tests: `objectstorage/v1/objects/delete_core_test.go:TestObjectDeleteCoreWorkflowAndAcknowledgements`, `objectstorage/v1/objects/delete_core_test.go:TestObjectDeleteCoreSLOFlagProjection`, `objectstorage/v1/objects/delete_contracts_test.go:TestObjectDeleteContractsRoutesAndSelection`, `objectstorage/v1/objects/delete_contracts_test.go:TestObjectDeleteContractsBulkOutcomes`.
- Default missing policy ignores only clean physical HEAD404 or DELETE404 and retains phase-specific proof. Strict missing remains a native error, completed Discovery survives later failure, and NumberNotFound is informational.
  Tests: `objectstorage/v1/objects/delete_core_test.go:TestObjectDeleteCoreMissingAndPhaseProofs`, `objectstorage/v1/objects/delete_contracts_test.go:TestObjectDeleteContractsMissingEvidence`, `objectstorage/v1/objects/delete_contracts_test.go:TestObjectDeleteContractsBulkOutcomes`, `connectiontest/connection_objectstorage_object_delete_test.go:TestConnectionObjectDeleteSharesClientAndOwnsPhases`.
- Only SLO actual200 decodes the required bulk JSON report after whitespace or heartbeats. The atomic decoder retains raw extensions, exact int64 counts and passive metadata. Embedded2xx plus empty Errors succeeds; other outcomes return parsed Bulk plus actual200 ResponseError and independent ObjectDeleteBulkError, retaining literal failure names and messages.
  Tests: `objectstorage/v1/objects/delete_core_test.go:TestObjectDeleteCoreBulkDecoderAndFailure`, `objectstorage/v1/objects/delete_contracts_test.go:TestObjectDeleteContractsBulkOutcomes`.
- Invalid accepted bulk reports, fractional or exponent counts, malformed error pairs, UTF8 faults and decoded Unicode control characters in Response Status fail with raw Deletion evidence and nil Bulk. Failed direct decoding leaves the previous model unchanged.
  Tests: `objectstorage/v1/objects/delete_core_test.go:TestObjectDeleteCoreBulkDecoderAndFailure`, `objectstorage/v1/objects/delete_contracts_test.go:TestObjectDeleteContractsBulkOutcomes`.
- Full options replace config; map and bool pointers are captured; Without helpers restore nil/default behavior. Callbacks run once and retained configs cannot change later phases. Canonical overlays and parallel reuse preserve owned requests and results, while live original auth changes remain visible.
  Tests: `objectstorage/v1/objects/delete_options_test.go:TestObjectDeleteOptionsSnapshots`, `objectstorage/v1/objects/delete_options_test.go:TestObjectDeleteOptionsParallelReuse`, `objectstorage/v1/objects/delete_contracts_test.go:TestObjectDeleteContractsOptionsAndSnapshots`, `connectiontest/connection_objectstorage_object_delete_test.go:TestConnectionObjectDeleteSharesClientAndOwnsPhases`.
- Invalid context, literal names, nil options, UTF8/control version-id, header case aliases and reserved auth/framing/metadata/Newest/SLO headers fail before HTTP. Invalid source headers fail before option callbacks.
  Tests: `objectstorage/v1/objects/delete_options_test.go:TestObjectDeleteOptionsPreflight`, `objectstorage/v1/objects/delete_contracts_test.go:TestObjectDeleteContractsOptionsAndSnapshots`.
- Read, Close, EOF-plus-error, custom context and source faults retain joined causes and partial phase evidence. They cannot make dirty404 ignored, bodies close once and accepted bodies are not replayed. Source drift stays sticky even if Close restores it; result mutation cannot corrupt owned error proof.
  Tests: `objectstorage/v1/objects/delete_core_test.go:TestObjectDeleteCoreBodyAndSourceFailures`, `objectstorage/v1/objects/delete_core_test.go:TestObjectDeleteCoreMissingAndPhaseProofs`, `objectstorage/v1/objects/delete_contracts_test.go:TestObjectDeleteContractsMissingEvidence`, `objectstorage/v1/objects/delete_contracts_test.go:TestObjectDeleteContractsPhaseErrorsAndGuards`.
- Original API, client, provider and target guards stop callback or native retry/backoff/reauth/redirect source changes and retain native rejected-status or nested404 evidence. Later DELETE transport or rejected errors retain completed Discovery.
  Tests: `objectstorage/v1/objects/delete_core_test.go:TestObjectDeleteCoreNativePolicyGuards`, `objectstorage/v1/objects/delete_contracts_test.go:TestObjectDeleteContractsPhaseErrorsAndGuards`, `objectstorage/v1/objects/delete_contracts_test.go:TestObjectDeleteContractsNativePolicy`.
- Every physical SLO DELETE forces one application/json Accept after native retry or redirect aliases while retaining other advanced native header overrides and live auth. Retry does not rerun callbacks, hooks cannot broaden phase codes, and original native Delete202/204 remains available.
  Tests: `objectstorage/v1/objects/delete_core_test.go:TestObjectDeleteCoreNativePolicyGuards`, `objectstorage/v1/objects/delete_options_test.go:TestObjectDeleteOptionsNativeRetrySnapshots`, `objectstorage/v1/objects/delete_contracts_test.go:TestObjectDeleteContractsNativePolicy`.
- Swift generated service/native API files remain byte-exact and accounts/containers/objects gain no generated Collections. The service guide retains the phase and SLO acknowledgement policy.
  Tests: `internal/cmd/sdkgen/swift_object_delete_test.go:TestSwiftObjectDeletePreservesNativeAndRegistry`.
- DeleteObject HEADs the object, then DELETEs it (with multipart-manifest=delete for an SLO manifest), and treats a 404 as ignored by default like Python ignore_missing=True.
  Tests: `objectstorage/v1/objects/python_rejudge_test.go:TestPythonRejudgeObjectDeleteRequests`.

Python/Go differences:

- Go requires explicit literal container/object and returns owned Discovery/Deletion responses, nullable SLO selection, missing flag and optional Bulk; Python reuses mutable Object/container metadata and returns None.
- Go uses strict nullable bool options and single true/false/1/0 HEAD header parsing; Python Header(type=bool) coerces nonempty strings with bool(value), including false.
- Go requires strict SLO200 bulk JSON embedded2xx plus emptyErrors and preserves failures through actual200 ResponseError/ObjectDeleteBulkError; pinned Python Resource.delete uses has_body=False and does not inspect embedded bulk failure.
- Go keeps native HEAD200/204 and DELETE202/204 compatibility; SLO202/204 is raw acknowledgement with nilBulk/unknown counts, while VersionID/Newest are explicit Go extensions applied to both phases.
- For SLO manifests Go adds Accept: application/json to read the bulk report; Python sends no Accept header and ignores the body.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "errors"
    "fmt"

    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func showDelete(result *objects.DeleteObjectResult) {
    if result == nil { return }
    fmt.Println("ignored missing:", result.IgnoredMissing)
    if result.StaticLargeObject == nil {
        fmt.Println("SLO flag unknown")
    } else {
        fmt.Println("SLO:", *result.StaticLargeObject)
    }
    for _, phase := range []struct {
        name string
        response *objects.DeleteObjectResponse
    }{{"HEAD", result.Discovery}, {"DELETE", result.Deletion}} {
        if phase.response != nil {
            fmt.Printf("%s status=%d transaction=%q observed-bytes=%d\n",
                phase.name, phase.response.StatusCode,
                phase.response.Header.Get("X-Trans-Id"), len(phase.response.Body))
        }
    }
    if result.Bulk != nil {
        fmt.Printf("bulk status=%d deleted=%d not-found=%d errors=%d\n",
            result.Bulk.ResponseCode, result.Bulk.NumberDeleted,
            result.Bulk.NumberNotFound, len(result.Bulk.Errors))
    }
}

func showDeleteError(err error) {
    var response *resource.ResponseError
    if errors.As(err, &response) {
        fmt.Printf("failed physical status=%d observed-bytes=%d\n",
            response.StatusCode, len(response.Body))
    }
    var bulk *objects.ObjectDeleteBulkError
    if errors.As(err, &bulk) {
        fmt.Printf("embedded bulk failure=%q errors=%d\n",
            bulk.ResponseStatus, len(bulk.Errors))
    }
}

func deleteObjects(ctx context.Context, api *objects.API, versionID string) error {
    knownSLO, allowMissing, newest := false, false, true
    auto := []objects.DeleteObjectOption{
        objects.WithDeleteObjectOpts(objects.DeleteObjectOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
            StaticLargeObject: &knownSLO, IgnoreMissing: &allowMissing, Newest: &newest,
        }),
        objects.WithDeleteObjectHeaders(map[string]string{"X-Trace-Label": "object-delete"}),
        objects.WithDeleteObjectHeader("X-Request-Label", "auto"),
        objects.WithoutDeleteObjectStaticLargeObject(),
        objects.WithDeleteObjectIgnoreMissing(false),
        objects.WithoutDeleteObjectIgnoreMissing(),
        objects.WithDeleteObjectVersionID(versionID),
        objects.WithDeleteObjectNewest(true),
        objects.WithoutDeleteObjectNewest(),
        objects.WithDeleteObjectNewest(false),
    }
    result, err := api.DeleteObject(ctx, "backups", "report.txt", auto...)
    showDelete(result)
    if err != nil {
        showDeleteError(err)
        return err
    }

    result, err = api.DeleteObject(ctx, "backups", "manifest",
        objects.WithDeleteObjectStaticLargeObject(true),
        objects.WithDeleteObjectIgnoreMissing(false),
    )
    showDelete(result)
    if err != nil { showDeleteError(err) }
    return err
}

func main() {}
```

```python
import openstack

conn = openstack.connect()
conn.object_store.delete_object("report.txt", container="backups")
conn.object_store.delete_object("strict-report.txt", container="backups", ignore_missing=False)
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L535-L561)
- [Object._raw_delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/obj.py#L374-L407)
- [Swift object API](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage-object-services.inc#L449-L625)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/objectstorage/v1/objects/requests.go#L455-L480)
- [DELETE default codes](https://github.com/gophercloud/gophercloud/blob/v2.15.0/provider_client.go#L582-L598)
- [SLO controller](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/slo.py#L1801-L1829)
- [bulk outcome](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/bulk.py#L331-L467)
- [Delete Core Test](delete_core_test.go)
- [Delete Options Test](delete_options_test.go)
- [public HTTP·missing·native hook](delete_contracts_test.go)
- [Connection Objectstorage Object Delete Test](../../../connectiontest/connection_objectstorage_object_delete_test.go)
- [Swift Object Delete Test](../../../internal/cmd/sdkgen/swift_object_delete_test.go)
