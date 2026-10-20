# Update

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:image/v2/images.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.WithUpdateOptions`

- Generated API accepts concrete UpdateOpts and the SDK owns the upstream builder and Extract invocation. All nine native concrete Patch types retain order and raw zero/false/empty values. Property add/replace includes string value; remove omits it, and property Names retain native unescaped JSON Pointer paths.
  Tests: `image/v2/images/update_contracts_test.go:TestNativeImageUpdateConcretePatchesAndOrder`.
- Nil/allocated-empty UpdateOpts both issue PATCH JSON[], nil tags is null and allocated-empty tags is[]. WithUpdateOptions replaces the whole base set, including nil replacement. Custom inherited Patch map extension remains available; the SDK does not claim it removed the element interface.
  Tests: `image/v2/images/update_contracts_test.go:TestNativeImageUpdateConcretePatchesAndOrder`, `image/v2/images/update_contracts_test.go:TestNativeImageUpdateEmptyReplacementAndInheritedPatchInterface`.
- Nil/error option and nonempty generic body/query/header/argument extensions fail before HTTP with OperationError context and original cause. Custom inherited Patch JSON marshaling error also fails before transport with its original cause; concrete UpdateOpts does not expose arbitrary external UpdateOptsBuilder.
  Tests: `image/v2/images/update_contracts_test.go:TestNativeImageUpdateRejectsRequestExtensionsAndMarshalFailuresBeforeHTTP`.
- Native Update uses service ResourceBase, PATCH and v2.1 JSON-patch content type with default OkCodes200 only. Native microversion/source headers/live auth/context, rejected-response RetryFunc and accepted JSON-decoder retry remain upstream policies. Default rejected201/203/204/404/503 preserve actual status/expected/method/URL/body/header in native error.
  Tests: `image/v2/images/update_contracts_test.go:TestNativeImageUpdateConcretePatchesAndOrder`, `image/v2/images/update_contracts_test.go:TestNativeImageUpdateStrictHTTPContextAndRetryPolicy`.
- Extract preserves native Image field/timestamp/unknown Properties handling, float64-based size and unknown number decoding, last import-method/store-ID header values and original response identity. It does not inject declared raw view, Envelope/status or an owned acknowledgement into the native result.
  Tests: `image/v2/images/update_contracts_test.go:TestNativeImageUpdateExtractsRichImageUnknownPropertiesAndLastHeaders`.
- Valid empty object returns native Image, valid JSONnull may return nil,nil. Body decode failures and invalid known types preserve upstream error plus possible partial Image; OperationError keeps causes inspectable without fabricating owned ResponseError or replacing partial fields.
  Tests: `image/v2/images/update_contracts_test.go:TestNativeImageUpdateExtractorErrorsKeepNativePartialResults`.

Python/Go differences:

- This mapping closes the pinned Gophercloud Update declaration with library-owned outer builder/options/result handling. It does not close Python update_image: native explicit patches always send HTTP, preserve caller order and default strict200 instead of mutable Resource dirty/no-op/sorted-diff/descriptor behavior.
- The SDK accepts concrete UpdateOpts instead of arbitrary external UpdateOptsBuilder. Its inherited Patch element interface remains optional for arbitrary maps; concrete nine Patch types cover the pinned built-in requests. Nil Patch elements retain native unsafe behavior; input slice/map/custom object snapshots and arbitrary callback safety are not added by this facade.
- Native IDs go through ServiceURL, property Names are interpolated without JSON Pointer escaping, schema/enum/range/readonly/policy/permission checks remain server-owned, and configured native client/hooks can change retry/header/response policies. Owned literal/route/body/source guards and ACK contracts are separate APIs.
- Native Image extraction retains its typed/null/date/float64/unknown-key behavior and possible partial pointer plus error or nil,nil. Accepted body Close/decoder/retry policy and response evidence are not promoted to the owned raw ImageRecord guarantees.
- The current/original raw body snapshot foundation preserves sparse presence before descriptor projection for future owned commit. It is not a mutable Resource dirty/header/cache/session state machine, and Python update_image remains unresolved with SDK-R1/C1/S1 work outstanding.
- Ordinary project-owned modify_image is a core-user candidate under the pinned server default; public visibility, owner and location operations can require additional admin policy. Local HTTP tests/builds establish client contracts, not deployed permissions/storage or actual cloud outcomes.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/images"
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    cloud := flag.String("cloud", "", "required clouds.yaml cloud name")
    imageID := flag.String("image-id", "", "required native image ID")
    name := flag.String("name", "release-image", "new image name")
    tag := flag.String("tag", "reviewed", "one tag replacing the whole tag set")
    flag.Parse()

    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, *name, *tag); err != nil {
        var native gophercloud.ErrUnexpectedResponseCode
        if errors.As(err, &native) {
            fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", native.Actual, len(native.Body))
        }
        log.Fatal(err)
    }
}

func run(ctx context.Context, cloud, imageID, name, tag string) error {
    if cloud == "" || imageID == "" {
        return fmt.Errorf("-cloud and -image-id are required")
    }
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }

    patches := images.UpdateOpts{
        images.ReplaceImageName{NewName: name},
        images.UpdateVisibility{Visibility: images.ImageVisibilityPrivate},
        images.ReplaceImageHidden{NewHidden: false},
        images.ReplaceImageProtected{NewProtected: false},
        images.ReplaceImageMinDisk{NewMinDisk: 0},
        images.ReplaceImageMinRam{NewMinRam: 0},
        images.ReplaceImageTags{NewTags: []string{tag}},
        images.UpdateImageProperty{Op: images.AddOp, Name: "sdk_demo_build", Value: "draft"},
        images.UpdateImageProperty{Op: images.ReplaceOp, Name: "sdk_demo_build", Value: "ready"},
        images.UpdateImageProperty{Op: images.AddOp, Name: "sdk_demo_temporary", Value: "temporary"},
        images.UpdateImageProperty{Op: images.RemoveOp, Name: "sdk_demo_temporary"},
    }
    // WithUpdateOptions replaces the base nil array with these concrete patches.
    value, operationErr := service.API.Images.Update(ctx, imageID, nil, images.WithUpdateOptions(patches))
    output := map[string]any{
        "image": value,
        "partial": value != nil && operationErr != nil,
    }
    if operationErr != nil { output["error"] = operationErr.Error() }
    body, marshalErr := json.MarshalIndent(output, "", "  ")
    if marshalErr != nil { return errors.Join(operationErr, marshalErr) }
    fmt.Println(string(body))
    return operationErr
}
```

```python
import openstack

conn = openstack.connect(cloud="dev")
updated = conn.image.update_image("image-id", name="release-image", is_protected=False)
same = conn.image.update_image(updated, name=updated.name)
assert same is updated
```

## References

- [Update](../../update.md)
- [Image Records](../../image-records.md)
- [Install](../../../docs/install.md)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/images/requests.go#L260-L418)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/images/requests.go#L236-L248)
- [Results](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/images/results.go#L161-L183)
- [Python proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1056-L1068)
- [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1875-L1937)
- [owned UpdateImageRecord](../../image-record-update.md)
- [Source pyproject](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/pyproject.toml#L15-L20)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L81-L116)
- [API policy](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/policy.py#L203-L229)
- [controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L657-L718)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L160-L171)
- [Contracts Test](../../../api/contracts_test.go)
- [Update Contracts Test](../../update_contracts_test.go)
- [Mutation Contracts Test](../../mutation_contracts_test.go)
- [SDK Support Ledger](../../../docs/sdk-support-ledger.md)
