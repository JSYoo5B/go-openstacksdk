# Volume Upload Image

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/upload_volume_to_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeImageUploadOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeImageUploadForce`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeImageUploadDiskFormat`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeImageUploadContainerFormat`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeImageUploadVisibility`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeImageUploadProtected`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareVolumeImageUploadOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.UploadVolumeToImage`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeImageUploadOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeImageUploadForce`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeImageUploadDiskFormat`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeImageUploadContainerFormat`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeImageUploadVisibility`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeImageUploadProtected`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.PrepareVolumeImageUploadOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.UploadVolumeToImage`
- `github.com/JSYoo5B/go-openstacksdk.Connection.UploadVolumeToImage`

- The required image name is literal UTF-8 text, including empty/control/path-like strings. Force defaults to an owned false pointer and is always sent; nil formats/visibility/protected are omitted, while explicit empty formats/visibility and false protected are included without format/visibility enums or invented defaults.
  Tests: `blockstorage/volume_image_upload_test.go:TestUploadVolumeToImageSerializesLiteralNamesDefaultsAndOptionalPresence`, `blockstorage/volume_image_upload_test.go:TestUploadVolumeToImageDirectPreflightAndOwnedOriginalsKeepFixedRequestPolicy`, `blockstorage/volume_image_upload_options_test.go:TestVolumeImageUploadOptionFactoriesOwnPointersDefaultsAndFullReplacement`, `blockstorage/volume_image_upload_options_test.go:TestVolumeImageUploadPrepareValidatesOnlyFinalUTF8AndPreservesLiteralStrings`.
- Only nonnil Visibility or Protected requests advertised 3.1 support, including empty/false. Both server bounds must contain fixed3.1 and any selected version must have the same major and satisfy3.1; a selected version above the advertised maximum can still pass. No-presence selected3.0 skips discovery; unselected required selection shares one probe and uses the ordinary3.71 cap/lower server maximum without rewriting the client.
  Tests: `blockstorage/volume_image_upload_test.go:TestUploadVolumeToImageConditional31GateUsesAdvertisedBoundsAndOrdinaryActionSelection`, `blockstorage/volume_image_upload_test.go:TestUploadVolumeToImageSerializesLiteralNamesDefaultsAndOptionalPresence`, `connection_volume_image_upload_test.go:TestConnectionVolumeImageUploadUsesCachedCinderAndGatesOnlyOptionalPresence`, `blockstorage/volume_image_upload_test.go:TestUploadVolumeToImageServiceKeepsNativeUploadImageOmission202AndTypedResult`.
- Owned factories/full replacement and HTTP-free Prepare preserve all pointer presence and reusable policy independence. The original option slice is copied before callbacks; every intermediate/final pointer is detached, final UTF-8 can be repaired by later replacement, and nil/callback/custom-cancellation failures return zero policy with every cause. Direct Source/ID/image admission precedes originals; Connection prepares originals once before image/ID/cached Cinder selection and consumes no CurrentLocation.
  Tests: `blockstorage/volume_image_upload_options_test.go:TestVolumeImageUploadOptionFactoriesOwnPointersDefaultsAndFullReplacement`, `blockstorage/volume_image_upload_options_test.go:TestVolumeImageUploadOriginalsOwnCallbackSliceAndIntermediatePointers`, `blockstorage/volume_image_upload_options_test.go:TestVolumeImageUploadPrepareValidatesOnlyFinalUTF8AndPreservesLiteralStrings`, `blockstorage/volume_image_upload_options_test.go:TestVolumeImageUploadPrepareNilCallbacksAndCancellationKeepZeroPolicyAndEveryCause`, `blockstorage/volume_image_upload_test.go:TestUploadVolumeToImageDirectPreflightAndOwnedOriginalsKeepFixedRequestPolicy`, `connection_volume_image_upload_test.go:TestConnectionVolumeImageUploadOriginalsRunOnceBeforeGetterAndRetainedPointersStayDetached`, `connection_volume_image_upload_test.go:TestConnectionVolumeImageUploadPreflightOptionsLiteralAndGetterFailuresKeepEveryCause`, `connection_volume_image_upload_test.go:TestConnectionVolumeImageUploadUsesCachedCinderAndGatesOnlyOptionalPresence`.
- Accepted action handling requires one UTF-8 JSON object and the exact os-volume_upload_image key. Its selected value can be any JSON shape including null, nonobject and object without image_id; interior whitespace/numeric/escape literals are preserved and the last outer duplicate wins. Missing/bad key/root/malformed/trailing/UTF8/empty204 failures retain current Applied and independent error proof, keep Upload nil/Completed false and issue no parser replay.
  Tests: `blockstorage/volume_image_upload_test.go:TestUploadVolumeToImageAcceptsArbitrarySelectedJSONAndKeepsFailedDecodeProof`, `blockstorage/volume_image_upload_test.go:TestUploadVolumeToImageServiceKeepsNativeUploadImageOmission202AndTypedResult`, `connection_volume_image_upload_test.go:TestConnectionVolumeImageUploadUsesCachedCinderAndGatesOnlyOptionalPresence`.
- Discovery, Applied, selected Upload and response-error snapshots own independent bytes/headers. Accepted Read/Close/source/custom-cancellation failures keep the current actual phase and every cause, retain earlier completed discovery independently, close once and leave Upload nil/Completed false. Completed reports successful helper response handling without a Glance lookup, image wait, refresh or rollback.
  Tests: `blockstorage/volume_image_upload_transport_test.go:TestUploadVolumeToImageAcceptedDiscoveryOrActionFailureKeepsCurrentProofAndNoValue`, `blockstorage/volume_image_upload_test.go:TestUploadVolumeToImageAcceptsArbitrarySelectedJSONAndKeepsFailedDecodeProof`, `connection_volume_image_upload_test.go:TestConnectionVolumeImageUploadUsesCachedCinderAndGatesOnlyOptionalPresence`.
- Live native401 reauthentication and bounded503 retry preserve the original serialized upload body, captured ordinary headers, chosen version and100..399 admission. Retry body/version/source mutations and physical redirected body/header/target/framing mutations block resend; original rejected HTTP retains native status/body/header with Applied nil and no borrowed discovery proof. The service facade keeps native UploadImage omitempty/202-only/typed VolumeImage behavior independently unchanged.
  Tests: `blockstorage/volume_image_upload_transport_test.go:TestUploadVolumeToImageLiveNativeRetriesAndPhysicalGuardsKeepBodyAndOriginalAdmission`, `blockstorage/volume_image_upload_test.go:TestUploadVolumeToImageServiceKeepsNativeUploadImageOmission202AndTypedResult`.

Python/Go differences:

- Go safe explicit current volume ID and UTF-8 literal image name replace Source supplied Resource/dict/Munch/None/dynamic inherited ID and constructor/descriptor/location/cache/session side effects. This helper adds no name finder, CurrentLocation, Volume model refresh, Glance lookup, image-byte upload, wait or rollback.
- Source annotations do not constrain runtime arguments: its own extended Proxy mock test forwards protected="5", and Python None/dynamic force can be forwarded literally. Go uses nullable typed strings/bools; Force nil means defaultfalse, optional nil means omission, and explicit empty/false remains present. Arbitrary Python runtime objects/custom serializers are outside this typed mapping.
- Source performs the3.1 requirement through endpoint-data utilities, then ordinary action selection through the Resource. Go captures one Source and uses finite operation-owned discovery/shared3.71 selection; this does not reproduce full Keystoneauth endpoint cache/getter count/canonicalization/exception order. Nonempty selected text is preserved; Go empty selection cannot represent every distinction between Python None and explicit empty default.
- The fixed3.1 gate compares3.1 with both advertised bounds and selected same-major minimum, not selected against the server maximum. A selected3.90 can pass advertised3.0..3.50 and is sent literally; global latest fails the same-major gate whereas3.latest passes. Dependency Session normalization/header behavior remains a separate declared boundary.
- The Source cast(dict, ...) is only an annotation cast and returns the exact selected JSON value; image_id and object shape are not required. Go returns owned literal RawMessage, preserving interior numeric/escape/whitespace spelling rather than Python mutable parsed objects/float rounding. Outer duplicate envelope keys follow ordinary last-key JSON decoding; successful null is nonnil literalnull, unlike error nil Upload.
- Go validates accepted UTF-8 JSON and an exact object envelope before exposing Upload, and exposes separate Discovery/Applied/Upload/Completed evidence. Accepted decode/IO/source/context faults keep actual completed HTTP proof without replay. These error/evidence choices do not promise identical Python exception taxonomy or final image/backend availability.
- Source raise_from_response admits status<400; Go uses finite100..399 and constrains native retry/redirect through fixed physical method/URL/body/framing/version and captured Source while permitting live provider auth. Original rejected HTTP remains native; discarded native rejected-body IO is not claimed fully reconstructed.
- Only the actual pinned v3 Proxy upload_volume_to_image target is classified as go_mapping. Native UploadImage omitempty/202-only/typed VolumeImage, Glance image.Service.UploadImage, v2/Resource/shared discovery dependencies are not promoted. Static Source/dependency comparison and Root offline Go HTTP/example checks do not imply Python/dependency/authenticated OpenStack runtime or full SDK completion.

### `python:block_storage/v3/create_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.CreateVolumeImageRecord`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.ExistingImageRecord`

- CreateVolumeImageRecord fills a falsey disk format from the captured cloud image_format (default qcow2, omitted when configured null), a falsey container format with bare and force from allow_duplicates, then sends one upload-to-image volume action through the existing Cinder upload engine.
  Tests: `connection_volume_image_create_test.go:TestConnectionCreateVolumeImageRecordDefaultsUploadAndExistingImage`.
- The upload result's image_id becomes Image.existing(id=...) through image.Service.ExistingImageRecord: a synchronized id-only record with the 65 declared fields and Connection location and no Glance request.
  Tests: `connection_volume_image_create_test.go:TestConnectionCreateVolumeImageRecordDefaultsUploadAndExistingImage`.
- A non-object result, missing, null, non-string or blank image_id returns the upload evidence with an invalid-option error under the Cloud operation; a non-string configured format and invalid volume IDs fail before HTTP.
  Tests: `connection_volume_image_create_test.go:TestConnectionCreateVolumeImageRecordFailuresKeepUploadEvidence`.

Python/Go differences:

- Python accepts a Volume Resource or ID and returns an Image Resource; Go takes a literal VolumeID and returns VolumeImageCreateResult separating the Cinder upload evidence from the ImageRecord.
- The pinned Proxy accepts wait and timeout but never reads them, so Go omits them. Python sends any configured image_format value; Go requires a string or null configuration and a nonblank string image_id.

### `python:cloud/_image/ImageCloudMixin/create_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.CreateCloudImageRecord`
- `github.com/JSYoo5B/go-openstacksdk/image.PrepareImageRecordCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCreateOpts`

- Without a volume CreateCloudImageRecord runs the owned CreateImageRecord graph with one prepared option snapshot and returns its Record when wait is falsey; with a volume it maps truthy allow_duplicates to force and Python-falsey container/disk formats to the Cinder Proxy defaults before CreateVolumeImageRecord.
  Tests: `connection_cloud_image_create_test.go:TestConnectionCloudImageCreateBranchesAndWait`, `connection_cloud_image_create_test.go:TestConnectionCloudImageCreateVolumeBodyAndPreflight`.
- A truthy wait repeats Cloud get_image on the created id: missing images keep polling, a found status other than exact queued or saving, including null, is returned, and WaitLookups/WaitLast keep the evidence.
  Tests: `connection_cloud_image_create_test.go:TestConnectionCloudImageCreateBranchesAndWait`.
- The iterate_timeout deadline uses the raw create timeout (default 3600 seconds, null unlimited); on timeout the image is deleted through Cloud delete_image with wait and the DeadlineExceeded error keeps the cleanup result and created record.
  Tests: `connection_cloud_image_create_test.go:TestConnectionCloudImageCreateBranchesAndWait`.
- Truthy non-string volume formats, non-numeric timeouts with wait, nil options and negative poll intervals fail before HTTP.
  Tests: `connection_cloud_image_create_test.go:TestConnectionCloudImageCreateVolumeBodyAndPreflight`.

Python/Go differences:

- Python returns an Image or None and raises ResourceTimeout or the cleanup exception; Go returns CloudImageCreateResult with Created, Volume, wait and Cleanup evidence and joins the timeout with any cleanup failure.
- Python keyword arguments become the concrete ImageRecordCreateOption values; the volume is a literal VolumeID and PollInterval is a Go extension for the Cloud wait and its cleanup.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "strconv"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    if err := run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run() error {
    cloudName, volumeID := os.Getenv("CLOUD_NAME"), os.Getenv("VOLUME_ID")
    if cloudName == "" || volumeID == "" { return fmt.Errorf("CLOUD_NAME and VOLUME_ID are required") }
    imageName, supplied := os.LookupEnv("IMAGE_NAME")
    if !supplied { return fmt.Errorf("IMAGE_NAME is required; explicit empty is allowed") }
    path := os.Getenv("IMAGE_UPLOAD_PATH")
    if path == "" { path = "connection" }
    if path != "connection" && path != "direct" && path != "service" { return fmt.Errorf("unknown IMAGE_UPLOAD_PATH %q", path) }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    var options []blockstorage.VolumeImageUploadOption
    if text, supplied := os.LookupEnv("IMAGE_UPLOAD_FORCE"); supplied {
        value, err := strconv.ParseBool(text)
        if err != nil { return fmt.Errorf("IMAGE_UPLOAD_FORCE: %w", err) }
        options = append(options, blockstorage.WithVolumeImageUploadForce(value))
    }
    if text, supplied := os.LookupEnv("IMAGE_UPLOAD_DISK_FORMAT"); supplied {
        options = append(options, blockstorage.WithVolumeImageUploadDiskFormat(text))
    }
    if text, supplied := os.LookupEnv("IMAGE_UPLOAD_CONTAINER_FORMAT"); supplied {
        options = append(options, blockstorage.WithVolumeImageUploadContainerFormat(text))
    }
    if text, supplied := os.LookupEnv("IMAGE_UPLOAD_VISIBILITY"); supplied {
        options = append(options, blockstorage.WithVolumeImageUploadVisibility(text))
    }
    if text, supplied := os.LookupEnv("IMAGE_UPLOAD_PROTECTED"); supplied {
        value, err := strconv.ParseBool(text)
        if err != nil { return fmt.Errorf("IMAGE_UPLOAD_PROTECTED: %w", err) }
        options = append(options, blockstorage.WithVolumeImageUploadProtected(value))
    }
    policy, err := blockstorage.PrepareVolumeImageUploadOptions(ctx, options...)
    if err != nil { return err }
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil { return err }
    input := blockstorage.VolumeActionRequest{VolumeID: volumeID}
    result, err := apply(ctx, conn, path, input, imageName, policy)
    inspect(result, err)
    return err
}

func apply(ctx context.Context, conn *sdk.Connection, path string, input blockstorage.VolumeActionRequest, imageName string, policy blockstorage.VolumeImageUploadOpts) (*blockstorage.VolumeImageUploadResult, error) {
    option := blockstorage.WithVolumeImageUploadOptions(policy)
    if path == "connection" { return conn.UploadVolumeToImage(ctx, input, imageName, option) }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { return nil, err }
    if path == "direct" { return blockstorage.UploadVolumeToImage(ctx, cinder.RawClient(), input, imageName, option) }
    return cinder.Volumes.UploadVolumeToImage(ctx, input.VolumeID, imageName, option)
}

func inspect(result *blockstorage.VolumeImageUploadResult, err error) {
    if result != nil {
        fmt.Printf("volume ID: %q; microversion: %q\n", result.VolumeID, result.Microversion)
        for index, page := range result.Discovery {
            if page != nil { fmt.Println("admitted discovery:", index, page.StatusCode, len(page.Body)) }
        }
        if page := result.Applied; page != nil { fmt.Printf("admitted action: HTTP %d; actual body %q\n", page.StatusCode, page.Body) }
        if result.Upload != nil {
            fmt.Printf("selected upload JSON: %s\n", result.Upload)
            // Optional application inspection; API success does not require image_id.
            var details map[string]json.RawMessage
            if json.Unmarshal(result.Upload, &details) == nil {
                if id, present := details["image_id"]; present { fmt.Printf("optional image_id raw value: %s\n", id) }
            }
        }
        fmt.Println("helper acknowledgement/parse completed:", result.Completed)
    }
    if err != nil {
        var proof *resource.ResponseError
        if errors.As(err, &proof) { fmt.Println("current admitted error response:", proof.StatusCode, len(proof.Body)) }
    }
}
```

```python
import openstack

conn = openstack.connect(cloud="devstack")
upload = conn.block_storage.upload_volume_to_image(volume_id, "new-image")
upload = conn.block_storage.upload_volume_to_image(
    volume_id, "", force=False, disk_format="", container_format="")
upload = conn.block_storage.upload_volume_to_image(
    volume_id, "new-image", visibility="", protected=False)
# The selected value need not be a dict or include image_id.
```

## References

- [Proxy.upload_volume_to_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1387-L1420)
- [Volume.upload_to_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L375-L403)
- [Discover](https://github.com/openstack/keystoneauth/blob/e759cf88c072d5bc7e21fc3fa884708ace9630af/keystoneauth1/discover.py#L433-L458)
- [Upload Image](../image/upload-image.md)
- [Volume Actions](volume-actions.md)
- [Volume image metadata](volume-image-metadata.md)
- [Block Storage API](README.md)
