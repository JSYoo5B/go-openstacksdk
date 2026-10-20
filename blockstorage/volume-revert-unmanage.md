# Volume Revert Unmanage

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/revert_volume_to_snapshot`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.RevertVolumeToSnapshot`
- `github.com/JSYoo5B/go-openstacksdk.Connection.RevertVolumeToSnapshot`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.RevertVolumeToSnapshot`

- Revert uses a required literal UTF-8 snapshotID body string, including empty/control/path-like strings, with exact {revert:{snapshot_id:value}} and safe current volume action route. No snapshot/current-volume GET or name resolver.
  Tests: `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanageKeepLiteralBodyDomainsAndOpaqueAcknowledgements`.
- Required3.40 server support is checked before action even when selected version exists: both advertisement bounds must contain3.40 and selected candidate must same-major>=3.40; compatible selected above server maximum remains raw literal. New revert range cases verify3.39/equal3.40/high selected and missing/range bounds; global latest/3.latest same-major comparisons are existing requireSupport engine regression plus actual fixed3.40 Source/code evidence.
  Tests: `blockstorage/volume_revert_unmanage_test.go:TestRevertVolumeToSnapshotRequiresAdvertised340AndReusesProbeForOrdinarySelection`, `blockstorage/volume_migration_test.go:TestMigrateVolumeRequiresAdvertisedSupportBeforeSelectedOrComputedVersion`.
- Unselected required calls reuse the same finite admitted advertisement to select ordinary cap3.71/maximum/optionalminimum. Missing/unsupported/malformed/IO/source/context gate evidence ends before POST with admitted discovery preserved.
  Tests: `blockstorage/volume_revert_unmanage_test.go:TestRevertVolumeToSnapshotRequiresAdvertised340AndReusesProbeForOrdinarySelection`, `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanageFailuresKeepOnlyTheActualDiscoveryOrActionProof`.
- Package, service and Connection have required parameters and no options/builder. Context/provider/typed input preflight occurs before HTTP; Connection uses cached Cinder without CurrentLocation or Resource conversion.
  Tests: `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanagePreflightAndServiceMethodsPreserveNativeSeparation`, `connection_volume_revert_unmanage_test.go:TestConnectionVolumeRevertUnmanageUsesCachedCinderAndLiteralSnapshotBody`, `connection_volume_revert_unmanage_test.go:TestConnectionVolumeRevertUnmanagePreflightAndGetterFaultsKeepCauses`.
- Action accepted response is opaque and independently owned; Completed acknowledges successful admission/read/close/source guard, while accepted faults retain partial Applied and joined causes without rollback/replay.
  Tests: `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanageKeepLiteralBodyDomainsAndOpaqueAcknowledgements`, `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanageFailuresKeepOnlyTheActualDiscoveryOrActionProof`.
- Original native HTTP rejection/retry policy and fixed physical request/source/live-auth boundaries cannot fabricate current Applied from discovery proof. No asynchronous wait, status/latest-snapshot validation or backend cleanup.
  Tests: `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanageFailuresKeepOnlyTheActualDiscoveryOrActionProof`, `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanageLiveAuthAndBoundedRetryPreserveBodyAndOriginalHTTPPolicy`.

Python/Go differences:

- Source local snapshot Resource construction can expose arbitrary snapshot.id values; Go deliberately restricts this body operand to literal UTF-8 string without restricting it as a route. Detached/available/latest-snapshot documentation preconditions remain server concerns; no implementation GET/check is introduced.
- The actual required Source default comparison uses pinned Keystoneauth5.16.0 version_match: global latest is incompatible with major3, 3.latest can pass; actual Session canonicalization is a separately documented dependency boundary.
- Go safe UTF-8 current route ID and typed literal snapshot string replace mutable Resource/dict/Munch/untyped ID and constructor/descriptor/location/cache side effects. No lookup, model refresh or CurrentLocation is added.
- Source returns None and discards opaque action response; Go owns VolumeID/Microversion/Discovery/Applied/Completed. Completed means helper acknowledgement, not backend completion/removal/deletion.
- Finite call-owned discovery shares advertisement rather than reproducing Keystoneauth Adapter cache/link/envelope/exception order/Session canonicalization; selected literal forwarding remains a Go mapping. Empty selected Go string cannot separately represent Python explicit default empty versus None.
- Source status<400 maps finite Go100..399. Original native rejection remains native with no Applied; discarded rejected POST body IO is not claimed fully reconstructed. Fixed method/URL/body/framing/version and accepted phase proof are Go-owned policies.
- Only this actual pinned v3 Proxy declaration is classified. Separate native/v2/Resource/manage/read/metadata dependencies are not automatically promoted; Source static evidence and passed local Go QA do not prove authenticated Python/cloud runtime or full SDK completion.
- Current installed seven new groups directly verify literal/null bodies, required3.40 range/probe reuse, preflight/cache/location omission, native Unmanage separation, opaque/accepted Read+Close+source+customcontext proof and liveauth/bounded retry/body mutation. Ordinary unselected cap3.71 and global latest/3.latest version comparison use separately declared existing common-engine regression; no new operation-specific exhaustive redirect/framing matrix is claimed.

### `python:block_storage/v3/unmanage_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.UnmanageVolume`
- `github.com/JSYoo5B/go-openstacksdk.Connection.UnmanageVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.UnmanageVolume`

- UnmanageVolume sends exact {os-unmanage:null} to the safe current volume action route with no lookup, DELETE, options, force flag or model construction.
  Tests: `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanageKeepLiteralBodyDomainsAndOpaqueAcknowledgements`, `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanagePreflightAndServiceMethodsPreserveNativeSeparation`, `connection_volume_revert_unmanage_test.go:TestConnectionVolumeRevertUnmanageUsesCachedCinderAndLiteralSnapshotBody`.
- No minimum gate is added. Selected nonempty version remains literal; otherwise finite ordinary discovery selects cap3.71/maximum/optionalminimum without source microversion mutation. Selected unmanage policy is tested directly; ordinary unselected cap details are existing shared Negotiate regression with actual same engine reuse.
  Tests: `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanageKeepLiteralBodyDomainsAndOpaqueAcknowledgements`, `connection_volume_revert_unmanage_test.go:TestConnectionVolumeRevertUnmanageUsesCachedCinderAndLiteralSnapshotBody`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsNegotiateFiniteVolumeCapWithoutChangingBackupCap`.
- Package, service and Connection share required current volume ID and no caller builder/options; context/provider/input preflight and cached Cinder selection retain operation labels/causes without CurrentLocation.
  Tests: `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanagePreflightAndServiceMethodsPreserveNativeSeparation`, `connection_volume_revert_unmanage_test.go:TestConnectionVolumeRevertUnmanageUsesCachedCinderAndLiteralSnapshotBody`, `connection_volume_revert_unmanage_test.go:TestConnectionVolumeRevertUnmanagePreflightAndGetterFaultsKeepCauses`.
- Opaque100..399 action response and independent Discovery/Applied proof expose helper Completed acknowledgement, not confirmed management removal or backend data deletion.
  Tests: `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanageKeepLiteralBodyDomainsAndOpaqueAcknowledgements`, `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanagePreflightAndServiceMethodsPreserveNativeSeparation`, `connection_volume_revert_unmanage_test.go:TestConnectionVolumeRevertUnmanageUsesCachedCinderAndLiteralSnapshotBody`.
- Accepted Read/Close/source/context failures preserve current Applied/all causes with Completedfalse and no SDK automatic replay/cleanup; original rejected HTTP remains native without borrowing discovery proof.
  Tests: `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanageFailuresKeepOnlyTheActualDiscoveryOrActionProof`, `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanageLiveAuthAndBoundedRetryPreserveBodyAndOriginalHTTPPolicy`.
- Existing native API.Unmanage is a separate unchanged {os-unmanage:{}}/202/error-only API; the new Source mapping exposes UnmanageVolume and does not broaden native/v2/Resource declarations.
  Tests: `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanagePreflightAndServiceMethodsPreserveNativeSeparation`.

Python/Go differences:

- Native Gophercloud v2.15.0 Unmanage differs intentionally: empty object body, only202, ErrResult. The Source null body/new owned result is implemented independently rather than changing native semantics.
- Go safe UTF-8 current route ID replaces mutable Resource/dict/Munch/untyped ID and constructor/descriptor/location/cache side effects. No lookup, model refresh or CurrentLocation is added.
- Source returns None and discards opaque action response; Go owns VolumeID/Microversion/Discovery/Applied/Completed. Completed means helper acknowledgement, not backend completion/removal/deletion.
- Finite call-owned discovery shares advertisement rather than reproducing Keystoneauth Adapter cache/link/envelope/exception order/Session canonicalization; selected literal forwarding remains a Go mapping. Empty selected Go string cannot separately represent Python explicit default empty versus None.
- Source status<400 maps finite Go100..399. Original native rejection remains native with no Applied; discarded rejected POST body IO is not claimed fully reconstructed. Fixed method/URL/body/framing/version and accepted phase proof are Go-owned policies.
- Only this actual pinned v3 Proxy declaration is classified. Separate native/v2/Resource/manage/read/metadata dependencies are not automatically promoted; Source static evidence and passed local Go QA do not prove authenticated Python/cloud runtime or full SDK completion.
- Current installed seven new groups directly verify literal/null bodies, required3.40 range/probe reuse, preflight/cache/location omission, native Unmanage separation, opaque/accepted Read+Close+source+customcontext proof and liveauth/bounded retry/body mutation. Ordinary unselected cap3.71 and global latest/3.latest version comparison use separately declared existing common-engine regression; no new operation-specific exhaustive redirect/framing matrix is claimed.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "os"
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
    if cloudName == "" || volumeID == "" {
        return fmt.Errorf("CLOUD_NAME and VOLUME_ID are required")
    }
    action := os.Getenv("VOLUME_LIFECYCLE_ACTION")
    if action != "revert" && action != "unmanage" {
        return fmt.Errorf("VOLUME_LIFECYCLE_ACTION must be revert or unmanage")
    }
    path := os.Getenv("VOLUME_LIFECYCLE_PATH")
    if path == "" { path = "connection" }
    if path != "connection" && path != "direct" && path != "service" {
        return fmt.Errorf("unknown VOLUME_LIFECYCLE_PATH %q", path)
    }
    var snapshotID string
    if action == "revert" {
        value, supplied := os.LookupEnv("SNAPSHOT_ID")
        if !supplied { return fmt.Errorf("SNAPSHOT_ID is required; explicit empty is allowed") }
        snapshotID = value
    }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil { return err }
    input := blockstorage.VolumeActionRequest{VolumeID: volumeID}
    result, err := apply(ctx, conn, path, action, input, snapshotID)
    inspect(result, err)
    return err
}

func apply(ctx context.Context, conn *sdk.Connection, path, action string, input blockstorage.VolumeActionRequest, snapshotID string) (*blockstorage.VolumeActionResult, error) {
    if path == "connection" {
        if action == "revert" { return conn.RevertVolumeToSnapshot(ctx, input, snapshotID) }
        return conn.UnmanageVolume(ctx, input)
    }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { return nil, err }
    if path == "direct" {
        if action == "revert" { return blockstorage.RevertVolumeToSnapshot(ctx, cinder.RawClient(), input, snapshotID) }
        return blockstorage.UnmanageVolume(ctx, cinder.RawClient(), input)
    }
    if action == "revert" { return cinder.Volumes.RevertVolumeToSnapshot(ctx, input.VolumeID, snapshotID) }
    return cinder.Volumes.UnmanageVolume(ctx, input.VolumeID)
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
        if errors.As(err, &proof) {
            fmt.Println("current admitted error response:", proof.StatusCode, len(proof.Body))
        }
    }
}
```

```python
import openstack

conn = openstack.connect(cloud="devstack")
conn.block_storage.revert_volume_to_snapshot(volume_id, snapshot_id)
conn.block_storage.unmanage_volume(volume_id)
```

## References

- [revert Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1220-L1239)
- [unmanage Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1293-L1303)
- [Volume.revert_to_snapshot](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L226-L232)
- [Volume.unmanage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L311-L315)
- [Keystoneauth5.16.0 version_match](https://github.com/openstack/keystoneauth/blob/e759cf88c072d5bc7e21fc3fa884708ace9630af/keystoneauth1/discover.py#L433-L458)
- [Volume Actions](volume-actions.md)
- [Volume Migration Reset](volume-migration-reset.md)
- [service API](v3/volumes/README.md)
