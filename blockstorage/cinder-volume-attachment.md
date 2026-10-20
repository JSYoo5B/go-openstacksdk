# Cinder Volume Attachment

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/attach_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCinderVolumeAttachOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCinderVolumeAttachInstance`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCinderVolumeAttachHostName`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareCinderVolumeAttachOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.AttachCinderVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithCinderVolumeAttachOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithCinderVolumeAttachInstance`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithCinderVolumeAttachHostName`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.PrepareCinderVolumeAttachOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.AttachCinderVolume`
- `github.com/JSYoo5B/go-openstacksdk.Connection.AttachCinderVolume`

- Attach includes required literal mountpoint and selects Instance by nonnil presence, including empty; Instance wins HostName and only one selected field is sent. HostName fallback includes explicit empty. No mode, instance/host lookup, Nova request or state/poll is added.
  Tests: `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentLiteralBranchesAcceptOpaqueAcknowledgementsWithoutLookup`, `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentServiceMethodsPreserveNativeAttachDetachContracts`, `connection_cinder_volume_attachment_test.go:TestConnectionCinderVolumeAttachmentsUseCachedCinderAndIgnoreLocation`.
- Both selectors nil fail locally; only final selected selector UTF-8 is consumed, so inactive invalid HostName is ignored. Required mountpoint accepts empty/control/path-like UTF-8 text and fails invalid UTF-8 before direct originals/HTTP.
  Tests: `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentLiteralBranchesAcceptOpaqueAcknowledgementsWithoutLookup`, `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentPreflightAndOwnedOriginalOptionsPrecedeHTTP`, `blockstorage/cinder_volume_attachment_options_test.go:TestCinderVolumeAttachmentPrepareValidatesFinalActiveValuesAndJoinsCauses`, `blockstorage/cinder_volume_attachment_options_test.go:TestCinderVolumeAttachmentDirectPreflightAndStickySourceStopOriginals`, `connection_cinder_volume_attachment_test.go:TestConnectionCinderVolumeAttachmentPreflightOptionsAndGetterFailuresKeepCauses`.
- Owned factories/full replacement and reusable HTTP-free Prepare copy nullable pointers, isolate returned policy from original inputs/results and reject missing/final-invalid selection with zero policy. No caller builder/interface is required.
  Tests: `blockstorage/cinder_volume_attachment_options_test.go:TestCinderVolumeAttachmentOptionsFactoriesDefaultsAndReplacementOwnership`, `blockstorage/cinder_volume_attachment_options_test.go:TestCinderVolumeAttachmentPrepareValidatesFinalActiveValuesAndJoinsCauses`.
- Direct Source/current-ID/required-literal admission precedes original callbacks; originals snapshot their slice, execute once in order and copy retained policy before/after each. Source faults are sticky and stop later restoration callbacks with all context/callback causes.
  Tests: `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentPreflightAndOwnedOriginalOptionsPrecedeHTTP`, `blockstorage/cinder_volume_attachment_options_test.go:TestCinderVolumeAttachmentOriginalsOwnSlicePointersAndConnectorThroughHTTP`, `blockstorage/cinder_volume_attachment_options_test.go:TestCinderVolumeAttachmentDirectPreflightAndStickySourceStopOriginals`.
- Connection preflight prepares originals once before required literal/current ID/cached Cinder getter; only Cinder is selected, configured CurrentLocation is not consumed, and retained values/getter/context/source faults preserve local operation/cause evidence.
  Tests: `connection_cinder_volume_attachment_test.go:TestConnectionCinderVolumeAttachmentsUseCachedCinderAndIgnoreLocation`, `connection_cinder_volume_attachment_test.go:TestConnectionCinderVolumeAttachmentOriginalsRunOnceBeforeGetterAndRetainedValuesStayDetached`, `connection_cinder_volume_attachment_test.go:TestConnectionCinderVolumeAttachmentPreflightOptionsAndGetterFailuresKeepCauses`.
- No action-specific minimum is added. Nonempty selected3.1/3.90/latest remains literal without discovery; unselected ordinary cap3.71/lower maximum/high minimum no-version policy owns separate admitted Discovery and action proof without source version mutation.
  Tests: `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentUsesOrdinarySelectedOrCappedMicroversionWithoutMinimumGate`, `connection_cinder_volume_attachment_test.go:TestConnectionCinderVolumeAttachmentsUseCachedCinderAndIgnoreLocation`.
- Opaque accepted action bytes/Header/status are independently owned. Completed acknowledges admission/read/Close/source success; accepted Read/Close/source/custom-context failures retain Applied/independent ResponseError and all causes with Completedfalse and no SDK replay.
  Tests: `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentLiteralBranchesAcceptOpaqueAcknowledgementsWithoutLookup`, `blockstorage/cinder_volume_attachment_transport_test.go:TestCinderVolumeAttachmentAcceptedFailuresKeepIndependentActualProofAndJoinedCauses`.
- Live auth/reauth/bounded retry preserve original action body/selected version/admission and block changed body/header/source/physical307 body/framing/target before resend. Native Attach mode/both-selectors/202/error-only and nil service policy remain separate from SDK service AttachCinderVolume.
  Tests: `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentServiceMethodsPreserveNativeAttachDetachContracts`, `blockstorage/cinder_volume_attachment_transport_test.go:TestCinderVolumeAttachmentNativeAuthRetriesAndPhysicalGuardsPreserveOriginalActionPolicy`.

Python/Go differences:

- Source nullable selectors can hold dynamic nonstring values; Go narrows to nullable UTF-8 strings and validates only selected instance/host. Explicit empty remains present. Direct Source attach has no mode; native Mode/omitempty/two nonempty selectors is preserved independently.
- Go safe current volume route ID and required UTF-8 body strings and nullable selectors replace supplied Resource/dict/Munch/None/untyped IDs, arbitrary dynamic annotations and constructor/descriptor/location/cache side effects; no name finder, refresh/model, CurrentLocation, wait/poll or additional server-state checks are introduced.
- Source action helpers return None after opaque response.status_code<400. Go finite100..399 policy adds owned VolumeID/Microversion/Discovery/Applied/Completed. Completed is helper acknowledgement, not server attachment/device state or backend readiness.
- Finite operation-owned ordinary discovery/max3.71 is an explicit Go mapping rather than complete Keystoneauth endpoint-data cache/link/envelope/exception ordering/Session header canonicalization. Nonempty selected literal is preserved; empty Go selection cannot separately represent Python explicit empty default versus None.
- Fixed method/route/serialized body/framing/selected version and admitted response ownership constrain retry/redirect while live native auth and allowed extra retry headers remain. Original native rejected HTTP is distinct and cannot become Applied; discarded rejected POST body IO is not claimed fully reconstructed.
- Only the actual pinned v3 Proxy attach_volume target is classified. Native/v2/Resource and existing cloud AttachVolume/DetachVolume workflows remain independently reviewed; the unchanged cloud boundary here is statically bound, not a new cloud workflow runtime test. No Python Source/dependency/authenticated OpenStack execution or full SDK completion is claimed.

### `python:block_storage/v3/detach_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCinderVolumeDetachOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCinderVolumeDetachForce`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCinderVolumeDetachConnector`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareCinderVolumeDetachOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.DetachCinderVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithCinderVolumeDetachOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithCinderVolumeDetachForce`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithCinderVolumeDetachConnector`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.PrepareCinderVolumeDetachOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.DetachCinderVolume`
- `github.com/JSYoo5B/go-openstacksdk.Connection.DetachCinderVolume`

- Detach Force nil/default/false sends os-detach with required attachment_id literal, including empty/control/path-like UTF-8. Inactive connector values/keys are not validated or sent, even if malformed. Force true selects os-force_detach without a separate lookup/wait/rollback workflow.
  Tests: `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentLiteralBranchesAcceptOpaqueAcknowledgementsWithoutLookup`, `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentPreflightAndOwnedOriginalOptionsPrecedeHTTP`, `blockstorage/cinder_volume_attachment_options_test.go:TestCinderVolumeAttachmentPrepareValidatesFinalActiveValuesAndJoinsCauses`, `connection_cinder_volume_attachment_test.go:TestConnectionCinderVolumeAttachmentsUseCachedCinderAndIgnoreLocation`.
- Force true omits nil/empty connector and includes all members of a nonempty object, including false/null/empty string and exact raw large/nested JSON values. Only active UTF-8 keys/complete UTF-8 JSON members are admitted; invalid/nil/trailing raw values fail locally and final repaired/inactive policy can succeed.
  Tests: `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentLiteralBranchesAcceptOpaqueAcknowledgementsWithoutLookup`, `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentPreflightAndOwnedOriginalOptionsPrecedeHTTP`, `blockstorage/cinder_volume_attachment_options_test.go:TestCinderVolumeAttachmentPrepareValidatesFinalActiveValuesAndJoinsCauses`.
- Owned false default pointer is fresh. Full replacement clears prior connector/force and restores defaultfalse; reusable WithOptions/Connector/Prepare own pointers/maps/every raw byte member, and failures return zero policy with joined context/callback causes.
  Tests: `blockstorage/cinder_volume_attachment_options_test.go:TestCinderVolumeAttachmentOptionsFactoriesDefaultsAndReplacementOwnership`, `blockstorage/cinder_volume_attachment_options_test.go:TestCinderVolumeAttachmentPrepareValidatesFinalActiveValuesAndJoinsCauses`.
- Original option slice/callback order/retained pointer-map-rawvalue ownership remain detached through HTTP. Direct Source/current ID/required literal preflight precedes callbacks; sticky source/cancel/callback errors stop later originals before physical proof.
  Tests: `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentPreflightAndOwnedOriginalOptionsPrecedeHTTP`, `blockstorage/cinder_volume_attachment_options_test.go:TestCinderVolumeAttachmentOriginalsOwnSlicePointersAndConnectorThroughHTTP`, `blockstorage/cinder_volume_attachment_options_test.go:TestCinderVolumeAttachmentDirectPreflightAndStickySourceStopOriginals`.
- Connection prepares originals once before body/route/cached Cinder admission; default/force calls need no Nova or CurrentLocation. Retained maps/getter mutation cannot alter the owned prepared request, while preflight/option/getter/context/source failures retain exact operation/cause without HTTP.
  Tests: `connection_cinder_volume_attachment_test.go:TestConnectionCinderVolumeAttachmentsUseCachedCinderAndIgnoreLocation`, `connection_cinder_volume_attachment_test.go:TestConnectionCinderVolumeAttachmentOriginalsRunOnceBeforeGetterAndRetainedValuesStayDetached`, `connection_cinder_volume_attachment_test.go:TestConnectionCinderVolumeAttachmentPreflightOptionsAndGetterFailuresKeepCauses`.
- Normal and force detach use ordinary selected literal or finite cap3.71/max/optionalminimum negotiation with no minimum gate. Original client selection is not mutated and admitted Discovery/action proof are independent.
  Tests: `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentUsesOrdinarySelectedOrCappedMicroversionWithoutMinimumGate`, `connection_cinder_volume_attachment_test.go:TestConnectionCinderVolumeAttachmentsUseCachedCinderAndIgnoreLocation`.
- Opaque acknowledgement and Completed remain distinct from backend disconnect/state completion. Accepted Read/Close/source/custom cancellation retain actual independent Applied/error proof and all causes without replay; rejected HTTP cannot borrow or fabricate admitted proof.
  Tests: `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentLiteralBranchesAcceptOpaqueAcknowledgementsWithoutLookup`, `blockstorage/cinder_volume_attachment_transport_test.go:TestCinderVolumeAttachmentAcceptedFailuresKeepIndependentActualProofAndJoinedCauses`, `blockstorage/cinder_volume_attachment_transport_test.go:TestCinderVolumeAttachmentNativeAuthRetriesAndPhysicalGuardsPreserveOriginalActionPolicy`.
- SDK service DetachCinderVolume normal/force/empty-ID/connector/result behavior remains distinct from native Detach empty omission/normal-only202/error-only. Live reauth/bounded retry/physical307 guards preserve original body/framing/URL/version/status policy and nil API preflight.
  Tests: `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentServiceMethodsPreserveNativeAttachDetachContracts`, `blockstorage/cinder_volume_attachment_transport_test.go:TestCinderVolumeAttachmentNativeAuthRetriesAndPhysicalGuardsPreserveOriginalActionPolicy`.

Python/Go differences:

- Source arbitrary force truthiness and truthy connector objects are deliberately narrowed to nullable bool and owned JSON object map. Inactive connector is not consumed. Raw numeric precision/spelling/map serialization are Go JSON representation boundaries rather than arbitrary Python objects/insertion-order serialization.
- Source force action does not call backend disconnect or a separate rollback helper; Go force flag only chooses the action. Native Detach normal-only/omitempty202/error-only remains unchanged.
- Go safe current volume route ID and required UTF-8 body strings, nullable force bool and owned JSON connector replace supplied Resource/dict/Munch/None/untyped IDs, arbitrary dynamic annotations and constructor/descriptor/location/cache side effects; no name finder, refresh/model, CurrentLocation, wait/poll or additional server-state checks are introduced.
- Source action helpers return None after opaque response.status_code<400. Go finite100..399 policy adds owned VolumeID/Microversion/Discovery/Applied/Completed. Completed is helper acknowledgement, not server attachment/device state or backend readiness.
- Finite operation-owned ordinary discovery/max3.71 is an explicit Go mapping rather than complete Keystoneauth endpoint-data cache/link/envelope/exception ordering/Session header canonicalization. Nonempty selected literal is preserved; empty Go selection cannot separately represent Python explicit empty default versus None.
- Fixed method/route/serialized body/framing/selected version and admitted response ownership constrain retry/redirect while live native auth and allowed extra retry headers remain. Original native rejected HTTP is distinct and cannot become Applied; discarded rejected POST body IO is not claimed fully reconstructed.
- Only the actual pinned v3 Proxy detach_volume target is classified. Native/v2/Resource and existing cloud AttachVolume/DetachVolume workflows remain independently reviewed; the unchanged cloud boundary here is statically bound, not a new cloud workflow runtime test. No Python Source/dependency/authenticated OpenStack execution or full SDK completion is claimed.

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
    action, path := os.Getenv("CINDER_ATTACHMENT_ACTION"), os.Getenv("CINDER_ATTACHMENT_PATH")
    if action != "attach" && action != "detach" { return fmt.Errorf("CINDER_ATTACHMENT_ACTION must be attach or detach") }
    if path == "" { path = "connection" }
    if path != "connection" && path != "direct" && path != "service" { return fmt.Errorf("unknown CINDER_ATTACHMENT_PATH %q", path) }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    var bodyValue string
    var attach blockstorage.CinderVolumeAttachOpts
    var detach blockstorage.CinderVolumeDetachOpts
    if action == "attach" {
        value, supplied := os.LookupEnv("CINDER_MOUNTPOINT")
        if !supplied { return fmt.Errorf("CINDER_MOUNTPOINT is required; explicit empty is allowed") }
        bodyValue = value
        var options []blockstorage.CinderVolumeAttachOption
        if value, supplied := os.LookupEnv("CINDER_INSTANCE"); supplied {
            options = append(options, blockstorage.WithCinderVolumeAttachInstance(value))
        }
        if value, supplied := os.LookupEnv("CINDER_HOST_NAME"); supplied {
            options = append(options, blockstorage.WithCinderVolumeAttachHostName(value))
        }
        prepared, err := blockstorage.PrepareCinderVolumeAttachOptions(ctx, options...)
        if err != nil { return err }
        attach = prepared
    } else {
        value, supplied := os.LookupEnv("CINDER_ATTACHMENT_ID")
        if !supplied { return fmt.Errorf("CINDER_ATTACHMENT_ID is required; explicit empty is allowed") }
        bodyValue = value
        force := false
        if text, supplied := os.LookupEnv("CINDER_FORCE"); supplied {
            value, err := strconv.ParseBool(text)
            if err != nil { return fmt.Errorf("CINDER_FORCE: %w", err) }
            force = value
        }
        options := []blockstorage.CinderVolumeDetachOption{blockstorage.WithCinderVolumeDetachForce(force)}
        if text, supplied := os.LookupEnv("CINDER_CONNECTOR_JSON"); force && supplied {
            var connector map[string]json.RawMessage
            if err := json.Unmarshal([]byte(text), &connector); err != nil { return fmt.Errorf("CINDER_CONNECTOR_JSON: %w", err) }
            options = append(options, blockstorage.WithCinderVolumeDetachConnector(connector))
        }
        prepared, err := blockstorage.PrepareCinderVolumeDetachOptions(ctx, options...)
        if err != nil { return err }
        detach = prepared
    }
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil { return err }
    result, err := apply(ctx, conn, path, action, blockstorage.VolumeActionRequest{VolumeID: volumeID}, bodyValue, attach, detach)
    inspect(result, err)
    return err
}

func apply(ctx context.Context, conn *sdk.Connection, path, action string, input blockstorage.VolumeActionRequest, bodyValue string, attach blockstorage.CinderVolumeAttachOpts, detach blockstorage.CinderVolumeDetachOpts) (*blockstorage.VolumeActionResult, error) {
    a := blockstorage.WithCinderVolumeAttachOptions(attach)
    d := blockstorage.WithCinderVolumeDetachOptions(detach)
    if path == "connection" {
        if action == "attach" { return conn.AttachCinderVolume(ctx, input, bodyValue, a) }
        return conn.DetachCinderVolume(ctx, input, bodyValue, d)
    }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { return nil, err }
    if path == "direct" {
        if action == "attach" { return blockstorage.AttachCinderVolume(ctx, cinder.RawClient(), input, bodyValue, a) }
        return blockstorage.DetachCinderVolume(ctx, cinder.RawClient(), input, bodyValue, d)
    }
    if action == "attach" { return cinder.Volumes.AttachCinderVolume(ctx, input.VolumeID, bodyValue, a) }
    return cinder.Volumes.DetachCinderVolume(ctx, input.VolumeID, bodyValue, d)
}

func inspect(result *blockstorage.VolumeActionResult, err error) {
    if result != nil {
        fmt.Printf("volume ID: %q; microversion: %q\n", result.VolumeID, result.Microversion)
        for index, page := range result.Discovery {
            if page != nil { fmt.Println("admitted discovery response:", index, page.StatusCode, len(page.Body)) }
        }
        if result.Applied != nil {
            fmt.Println("admitted action response:", result.Applied.StatusCode)
            fmt.Printf("opaque body: %q\n", result.Applied.Body)
        }
        fmt.Println("helper acknowledgement completed:", result.Completed)
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
conn.block_storage.attach_volume(volume_id, "/dev/vdb", instance=server_id)
conn.block_storage.attach_volume(volume_id, "", instance="", host_name="ignored")
conn.block_storage.detach_volume(volume_id, attachment_id)
conn.block_storage.detach_volume(volume_id, attachment_id, force=True,
                                 connector={"host": "", "multipath": False, "extra": None})
```

## References

- [Proxy attach/detach](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1241-L1280)
- [Volume.attach/detach](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L234-L271)
- [AttachVolume](attach-volume.md)
- [DetachVolume](detach-volume.md)
- [Volume Actions](volume-actions.md)
- [service API](v3/volumes/README.md)
