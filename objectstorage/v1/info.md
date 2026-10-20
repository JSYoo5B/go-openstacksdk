# Info

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:object_store/v1/get_info`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.Service.GetInfo`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithGetInfoOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithGetInfoHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithGetInfoHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.Info.UnmarshalJSON`

- Fresh fixed GET derives the escaped catalog prefix/version path, ignores ResourceBase for routing and preserves actual200 capabilities and header evidence.
  Tests: `objectstorage/v1/info_contracts_test.go:TestObjectInfoContractsRoutesAndModels`, `connection_objectstorage_info_test.go:TestConnectionSwiftInfoSharesClientAndCapturesEachWorkflow`.
- UTF8 nonnull root object and five nullable object sections project atomically; unknown plugin/large-number/raw inherited metadata remain exact and canonical section bytes own storage independently.
  Tests: `objectstorage/v1/info_contracts_test.go:TestObjectInfoContractsRoutesAndModels`, `objectstorage/v1/info_contracts_test.go:TestObjectInfoContractsAcceptedResponse`, `objectstorage/v1/info_core_test.go:TestInfoCoreModelAtomicRawSections`, `objectstorage/v1/info_core_test.go:TestInfoCoreModelFailuresAndAcceptedEvidence`.
- Full options replace config; header factories/callbacks own maps, last canonical overlay wins, valid ordinary source drift leaves the captured request unchanged and original live auth remains active.
  Tests: `objectstorage/v1/info_contracts_test.go:TestObjectInfoContractsOptionsAndSource`, `objectstorage/v1/info_contracts_test.go:TestObjectInfoContractsNativePolicy`, `connection_objectstorage_info_test.go:TestConnectionSwiftInfoSharesClientAndCapturesEachWorkflow`, `objectstorage/v1/info_options_test.go:TestInfoOptionsFactoryAndCallbackOwnership`, `objectstorage/v1/info_options_test.go:TestInfoOptionsParallelReuse`, `objectstorage/v1/info_options_test.go:TestInfoOptionsNativeRetrySnapshots`.
- Original source/API/provider/Endpoint/ResourceBase/Type/Microversion changes and reserved/malformed header aliases fail before another HTTP attempt; retries cannot inject body or broaden the public actual200 status gate.
  Tests: `objectstorage/v1/info_contracts_test.go:TestObjectInfoContractsOptionsAndSource`, `objectstorage/v1/info_contracts_test.go:TestObjectInfoContractsNativePolicy`, `objectstorage/v1/info_contracts_test.go:TestObjectInfoContractsAcceptedResponse`, `objectstorage/v1/info_options_test.go:TestInfoOptionsHeaderValidationAndPreflight`, `objectstorage/v1/info_core_test.go:TestInfoCoreCallbackAndWireGuards`.
- Accepted read/Close/custom context/source failures preserve partial raw response proof and joined causes, clear typed Info and never replay accepted processing; returned data cannot mutate the owned error proof.
  Tests: `objectstorage/v1/info_contracts_test.go:TestObjectInfoContractsAcceptedResponse`, `objectstorage/v1/info_contracts_test.go:TestObjectInfoContractsNativePolicy`, `objectstorage/v1/info_core_test.go:TestInfoCoreModelFailuresAndAcceptedEvidence`, `objectstorage/v1/info_core_test.go:TestInfoCoreCallbackAndWireGuards`.
- Existing Swift generated native service/API files remain exact and accounts/containers/objects do not acquire new generated Collections.
  Tests: `internal/cmd/sdkgen/swift_info_test.go:TestSwiftInfoPreservesNativeAndRegistry`.
- GetInfo sends GET /info beside the versioned endpoint and exposes the swift and slo sections, as Python Info.fetch does.
  Tests: `objectstorage/v1/python_rejudge_test.go:TestPythonRejudgeSwiftInfoAndSegmentSize`.

Python/Go differences:

- Go returns owned Info raw fields and actual200 evidence with atomic nullable-object sections; Python fetch mutates a fresh Resource, accepts response translation below400 and coerces non-dict known sections to empty dict.
- Go derives a fixed escaped-path URL with ASCII regex, rejects query/fragment/userinfo/opaque URLs and treats semicolon as path; Python uses Unicode digits and preserves URL params/query/fragment.
- Inherited Metadata timestamps and Links stay passive/nil while every root JSON field remains raw; Go does not follow capability links or synthesize signed admin query.
- Go returns typed result values with copied raw response evidence instead of a mutable Python Resource, so there is no Resource descriptor coercion, dirty-attribute tracking or seeded-object merge.
- Go has no client-side GET response cache or skip_cache control; every call sends its request.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.

### `python:object_store/v1/get_object_segment_size`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.Service.GetObjectSegmentSize`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithObjectSegmentSizeOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithObjectSegmentSizeHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithObjectSegmentSizeHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithObjectSegmentSize`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithoutObjectSegmentSize`

- Fresh actual200 capabilities select exact int64 requested/default1073741824/explicit0 sizes using maximum first then minimum, preserve inverted-bound ordering and use0 for absent/null sections or bounds.
  Tests: `objectstorage/v1/info_contracts_test.go:TestObjectInfoContractsSegmentBounds`, `connection_objectstorage_info_test.go:TestConnectionSwiftInfoSharesClientAndCapturesEachWorkflow`, `objectstorage/v1/info_core_test.go:TestInfoCoreSegmentSelection`.
- Reusable full/header/size/reset options own maps/pointers and preserve callback/source snapshots; each workflow captures the shared Connection client and live authentication separately.
  Tests: `objectstorage/v1/info_contracts_test.go:TestObjectInfoContractsOptionsAndSource`, `connection_objectstorage_info_test.go:TestConnectionSwiftInfoSharesClientAndCapturesEachWorkflow`, `objectstorage/v1/info_options_test.go:TestInfoOptionsFactoryAndCallbackOwnership`, `objectstorage/v1/info_options_test.go:TestInfoOptionsParallelReuse`.
- Negative requested size, nil option/context, malformed/case-alias/reserved source or caller headers fail before HTTP; invalid source headers are checked before callbacks.
  Tests: `objectstorage/v1/info_options_test.go:TestInfoOptionsHeaderValidationAndPreflight`.
- Only clean physically accepted404/412 bypass retry and set fallback maximum2684354561/minimum0; result retains requested/selected size, actual status/header/raw body and nil Info.
  Tests: `objectstorage/v1/info_contracts_test.go:TestObjectInfoContractsFallbackEvidence`, `objectstorage/v1/info_core_test.go:TestInfoCorePhysicalFallbackAndDirtyResponses`.
- Invalid accepted JSON/model/bounds return a nonnil raw result with requested size and actual proof, Size0/no fallback; decoded Info survives bounds failure and both bounds stay atomic.
  Tests: `objectstorage/v1/info_contracts_test.go:TestObjectInfoContractsAcceptedResponse`, `objectstorage/v1/info_contracts_test.go:TestObjectInfoContractsSegmentBounds`, `objectstorage/v1/info_core_test.go:TestInfoCoreModelFailuresAndAcceptedEvidence`, `objectstorage/v1/info_core_test.go:TestInfoCoreSegmentBoundsAtomic`.
- Accepted read/Close/EOF-plus-error/custom context/source faults remain joined and retain immutable partial evidence without retry; dirty404/412 never finalize fallback or selected size.
  Tests: `objectstorage/v1/info_contracts_test.go:TestObjectInfoContractsAcceptedResponse`, `objectstorage/v1/info_contracts_test.go:TestObjectInfoContractsFallbackEvidence`, `objectstorage/v1/info_contracts_test.go:TestObjectInfoContractsNativePolicy`, `objectstorage/v1/info_core_test.go:TestInfoCorePhysicalFallbackAndDirtyResponses`, `objectstorage/v1/info_core_test.go:TestInfoCoreCallbackAndWireGuards`.
- Transport/retry/reauth nested404 and other actual rejected statuses remain native errors; fixed URL/body ownership and original-source guards preserve configured advanced native header overwrite policy.
  Tests: `objectstorage/v1/info_contracts_test.go:TestObjectInfoContractsFallbackEvidence`, `objectstorage/v1/info_contracts_test.go:TestObjectInfoContractsNativePolicy`, `objectstorage/v1/info_contracts_test.go:TestObjectInfoContractsAcceptedResponse`, `objectstorage/v1/info_options_test.go:TestInfoOptionsNativeRetrySnapshots`, `objectstorage/v1/info_core_test.go:TestInfoCoreCallbackAndWireGuards`.
- Existing Swift generated native service/API files remain exact and accounts/containers/objects do not acquire new generated Collections.
  Tests: `internal/cmd/sdkgen/swift_info_test.go:TestSwiftInfoPreservesNativeAndRegistry`.
- GetObjectSegmentSize with no size selects 1 GiB within the /info bounds and falls back to 2684354561 after a 404, the values Python returns.
  Tests: `objectstorage/v1/python_rejudge_test.go:TestPythonRejudgeSwiftInfoAndSegmentSize`.

Python/Go differences:

- Go uses optional non-negative int64 Size with nil default1073741824, strict exact int64 non-negative bounds and absent/null section or bound values0; Python requires positional int|None, does not preflight negative/type inputs, raises AttributeError for swift/slo None and TypeError for null bounds.
- Go exposes RequestedSize, selected Size, observed bounds, optional Info and actual raw response evidence; Python returns a scalar and its404/412 fallback maximum is float2684354561.0.
- Only clean physically accepted404/412 responses with read/Close/context/source checks permit integer fallback2684354561; nested native/transport statuses or accepted processing errors do not fallback. Maximum comparison remains before minimum even for inverted bounds.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "errors"
    "fmt"

    objectstorage "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func reportResponseError(err error) {
    var response *resource.ResponseError
    if errors.As(err, &response) {
        fmt.Printf("failed status=%d transaction=%q observed-bytes=%d\n",
            response.StatusCode, response.Header.Get("X-Trans-Id"), len(response.Body))
    }
}

func showSegment(result *objectstorage.ObjectSegmentSizeResult) {
    if result == nil { return }
    fmt.Printf("requested=%d max=%d min=%d selected=%d fallback=%t status=%d raw-bytes=%d\n",
        result.RequestedSize, result.MaxFileSize, result.MinSegmentSize,
        result.Size, result.UsedFallback, result.StatusCode, len(result.Body))
}

func capabilities(ctx context.Context, service *objectstorage.Service) error {
    info, err := service.GetInfo(ctx,
        objectstorage.WithGetInfoOpts(objectstorage.GetInfoOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
        }),
        objectstorage.WithGetInfoHeaders(map[string]string{"X-Trace-Label": "capabilities"}),
        objectstorage.WithGetInfoHeader("X-Request-Label", "info"),
    )
    if err != nil {
        reportResponseError(err)
        return err
    }
    fmt.Printf("info status=%d transaction=%q\n", info.StatusCode, info.Header.Get("X-Trans-Id"))
    if info.Swift != nil {
        if maximum, ok := info.Swift["max_file_size"]; ok {
            fmt.Println("advertised max_file_size:", string(maximum))
        }
    }
    fmt.Println("SLO section present:", info.SLO != nil)
    if plugin, ok := info.Body["my_plugin"]; ok {
        fmt.Println("raw plugin:", string(plugin))
    }

    large := int64(8 * 1024 * 1024 * 1024)
    defaultOptions := []objectstorage.ObjectSegmentSizeOption{
        objectstorage.WithObjectSegmentSizeOpts(objectstorage.ObjectSegmentSizeOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
            Size: &large,
        }),
        objectstorage.WithObjectSegmentSizeHeaders(map[string]string{"X-Trace-Label": "segment"}),
        objectstorage.WithObjectSegmentSizeHeader("X-Request-Label", "default"),
        objectstorage.WithObjectSegmentSize(2 * 1024 * 1024 * 1024),
        objectstorage.WithoutObjectSegmentSize(),
    }
    selected, err := service.GetObjectSegmentSize(ctx, defaultOptions...)
    showSegment(selected)
    if err != nil {
        reportResponseError(err)
        return err
    }

    selected, err = service.GetObjectSegmentSize(ctx,
        objectstorage.WithObjectSegmentSize(large),
    )
    showSegment(selected)
    if err != nil { reportResponseError(err) }
    return err
}

func main() {}
```

```python
import openstack

conn = openstack.connect()
caps = conn.object_store.get_info()
print("Swift capability:", caps.swift)
print("SLO capability:", caps.slo)
print("default selection:", conn.object_store.get_object_segment_size(None))
print("large selection:", conn.object_store.get_object_segment_size(8 * 1024**3))
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L860-L919)
- [Info Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/info.py#L25-L96)
- [Swift `/info` API](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage_info.inc#L1-L46)
- [Info](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/proxy/controllers/info.py#L55-L108)
- [effective constraints](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/proxy/server.py#L359-L378)
- [Constraints](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/constraints.py#L28-L66)
- [Slo](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/slo.py#L1893-L1922)
- [Info Core Test](info_core_test.go)
- [Info Options Test](info_options_test.go)
- [public HTTP·native retry·fallback](info_contracts_test.go)
- [Connection Objectstorage Info Test](../../connection_objectstorage_info_test.go)
- [Swift Info Test](../../internal/cmd/sdkgen/swift_info_test.go)
