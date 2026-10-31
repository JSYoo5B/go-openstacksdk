# Volume Migration Reset

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/reset_volume_status`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.ResetVolumeStatus`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ResetVolumeStatus`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.ResetVolumeStatus`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeStatusResetOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeStatusResetStatus`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeStatusResetAttachStatus`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeStatusResetMigrationStatus`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareVolumeStatusResetOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeStatusResetOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeStatusResetStatus`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeStatusResetAttachStatus`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeStatusResetMigrationStatus`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.PrepareVolumeStatusResetOptions`

- Reset defaults all three statuses to omitted. Nil and empty typed strings are omitted, all nonempty UTF-8 strings are sent independently without enum/default injection; empty options still POST {os-reset_status:{}}.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsSerializeTypedDefaultsAndPresenceWithoutLookups`.
- Reusable owned option factories/full replacement and service-free preparation preserve copied pointers, original callback slice and effective policy; no caller builder is needed.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationOriginalOptionsOwnFactoryPointersRetainedConfigsAndCallbackOrder`, `connectiontest/connection_volume_migration_test.go:TestConnectionVolumeMigrationOriginalsRunOnceBeforeGetterAndRetainedValuesStayDetached`.
- Direct/source admission and Connection prepare-before-cached-Cinder ordering retain operation/context/getter faults before HTTP, without lookup or CurrentLocation.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsCaptureSourcesBeforeOriginalsAndStopBeforeRestoration`, `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsPreflightAndCallbackFailuresPreservePriorityAndCauses`, `connectiontest/connection_volume_migration_test.go:TestConnectionVolumeMigrationUsesCachedCinderAndSelectedVersionStillRequiresDiscovery`, `connectiontest/connection_volume_migration_test.go:TestConnectionVolumeMigrationOriginalsRunOnceBeforeGetterAndRetainedValuesStayDetached`, `connectiontest/connection_volume_migration_test.go:TestConnectionVolumeMigrationPreflightAndGetterFailuresRetainCauses`.
- Selected literal/cap3.71 uses ordinary shared Cinder negotiation with no API minimum gate; opaque action proof and Completed acknowledgement remain distinct.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsSerializeTypedDefaultsAndPresenceWithoutLookups`, `connectiontest/connection_volume_migration_test.go:TestConnectionVolumeMigrationUsesCachedCinderAndSelectedVersionStillRequiresDiscovery`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsNegotiateFiniteVolumeCapWithoutChangingBackupCap`.
- ResetVolumeStatus reaches the same applyRequired/Post accepted-fault policy whose existing Cinder regression retains actual Applied/all joined causes without completion or replay. New reset cases directly verify original native400/expanded503retry rejection keeps native evidence and cannot become Applied or Completed.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsKeepOriginalNativePolicyWhenRetryExpandsHTTPAdmission`, `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionsAcceptedFaultsKeepAppliedAndEveryCauseWithoutCompletion`.
- New package/Connection/service ResetVolumeStatus methods share the exact typed action and preserve nil API operation labels without originals or HTTP. Existing native ResetStatus remains a separately inspected source/static API; its always-included status/202/error-only behavior is not asserted by this new service test.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationServiceMethodsKeepSDKContractsAndNilAPIPriority`.

Python/Go differences:

- Source truthy status accepts arbitrary dynamic values; Go deliberately narrows to nullable UTF-8 strings and nonempty omission. Existing native required string body is not reused as Source-equivalent.
- Native Volume ResetStatus always-included status/202/error-only separation is established by current native/generated Source declarations and unchanged signature, not a newly executed native reset preservation scenario.
- Go takes one safe UTF-8 current volume ID and typed strings/bools rather than supplied mutable Volume/VolumeDetail/Resource/dict/Munch inputs and Source constructor/descriptor/location/urljoin side effects. Separate Resource/native/v2 declarations remain independent.
- Source returns None without response translation or refresh. Go adds owned VolumeID, Microversion, Discovery, Applied and Completed acknowledgement. No lookup, CurrentLocation, asynchronous wait/poll, cleanup or undo is added.
- Finite call-owned discovery and typed advertisement/evidence policy are explicit Go mappings. Complete Keystoneauth cache/link/envelope/canonicalization/discovery exception ordering and authenticated backend behavior are not claimed. Empty Go selection cannot represent Python explicit default empty separately from None.
- Opaque finite100..399 action policy maps Source<400. Accepted body IO/source/context proof remains separate from native rejection; native discarded rejected POST body IO remains a limitation. Exact method/route/body/framing/version guards constrain native retry/redirect while live auth remains.
- New11 groups cover typed body/defaults, options/capture/preflight, conditional required discovery, original admission and facade/service behavior. Accepted action IO, ordinary cap details and complete reauth/307/framing matrices use declared shared Cinder engine reuse where applicable; no new operation-specific exhaustive transport matrix is claimed. Standalone Prepare wrappers are compiled and exercised through Connection preparation, not a separate new reusable-Prepare return mutation matrix.
- Only the actual pinned v3 Proxy target is reviewed here. Root offline local HTTP race tests, vet and exact documentation compilation passed. Static Source hashes and offline Go checks do not prove authenticated OpenStack/Python runtime parity or full SDK completion.

### `python:block_storage/v3/migrate_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.MigrateVolume`
- `github.com/JSYoo5B/go-openstacksdk.Connection.MigrateVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.MigrateVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeMigrationOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeMigrationHost`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeMigrationCluster`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeMigrationForceHostCopy`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeMigrationLockVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareVolumeMigrationOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeMigrationOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeMigrationHost`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeMigrationCluster`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeMigrationForceHostCopy`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeMigrationLockVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.PrepareVolumeMigrationOptions`

- Host/Cluster nil omission differs from explicit empty inclusion. False/default ForceHostCopy/LockVolume fields are omitted; true fields are included. No host/cluster resolver, requirement or mutual-exclusion check is added.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsSerializeTypedDefaultsAndPresenceWithoutLookups`, `blockstorage/volume_migration_test.go:TestVolumeMigrationServiceMethodsKeepSDKContractsAndNilAPIPriority`.
- Cluster nonnil, including empty, requires advertised server min/max to support3.16 before action, even when a selected version exists; compatible selected same-major>=3.16 is additionally required. Failure prevents POST. The pinned Keystoneauth5.16.0 version_match has exact same-major/tuple semantics: global latest fails required3.16, while3.latest can pass the gate and keeps its literal POST policy.
  Tests: `blockstorage/volume_migration_test.go:TestMigrateVolumeRequiresAdvertisedSupportBeforeSelectedOrComputedVersion`, `connectiontest/connection_volume_migration_test.go:TestConnectionVolumeMigrationUsesCachedCinderAndSelectedVersionStillRequiresDiscovery`.
- Required server range tests3.16 rather than selected literal; high compatible selected version stays literal above advertised maximum. Unselected calls select cap3.71 from the same captured advertisement; no original client version mutation or dependency cache emulation.
  Tests: `blockstorage/volume_migration_test.go:TestMigrateVolumeRequiresAdvertisedSupportBeforeSelectedOrComputedVersion`, `connectiontest/connection_volume_migration_test.go:TestConnectionVolumeMigrationUsesCachedCinderAndSelectedVersionStillRequiresDiscovery`.
- Owned option preparation and original callback once/direct capture/Connection cached Cinder preserve caller-owned values and fixed payload across retry.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationOriginalOptionsOwnFactoryPointersRetainedConfigsAndCallbackOrder`, `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsCaptureSourcesBeforeOriginalsAndStopBeforeRestoration`, `connectiontest/connection_volume_migration_test.go:TestConnectionVolumeMigrationOriginalsRunOnceBeforeGetterAndRetainedValuesStayDetached`, `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsKeepOriginalNativePolicyWhenRetryExpandsHTTPAdmission`.
- Discovery and action phases own separate current proof; malformed accepted discovery, missing/unsupported gate bounds and source/context faults prevent action and preserve actual admitted evidence.
  Tests: `blockstorage/volume_migration_test.go:TestMigrateVolumeRequiresAdvertisedSupportBeforeSelectedOrComputedVersion`, `blockstorage/volume_migration_test.go:TestMigrateVolumeFiniteDiscoveryKeepsCurrentProofAndNeverBorrowsRejectedPages`.
- New MigrateVolume cases verify opaque action acknowledgement and original native rejection through the shared action engine, with no asynchronous wait, host GET, volume refresh or rollback. Existing shared Cinder accepted Read/Close/source/custom-context regression verifies the common Post failure path; it is not a new cluster-action IO matrix.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsSerializeTypedDefaultsAndPresenceWithoutLookups`, `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsKeepOriginalNativePolicyWhenRetryExpandsHTTPAdmission`, `blockstorage/volume_migration_test.go:TestVolumeMigrationServiceMethodsKeepSDKContractsAndNilAPIPriority`, `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionsAcceptedFaultsKeepAppliedAndEveryCauseWithoutCompletion`.

Python/Go differences:

- Source host/cluster arbitrary non-None values and flags arbitrary truthy values are narrowed to typed nullable strings/bools; explicit cluster empty retains required gate.
- Source require calls endpoint getter and both server bounds before default normalization; Go shares one finite guarded probe with ordinary action version selection and explicitly omits full Keystoneauth cache/canonicalization/taxonomy. Python selected default empty versus None is not independently expressible in Go.
- Keystoneauth5.16.0 commit e759cf88c072d5bc7e21fc3fa884708ace9630af statically binds normalize_version_number/version_between/version_match. Source requires>=5.16.0; this does not execute installed dependency/runtime or promote Adapter cache/Session behavior. Bounds normalize up front, required3.16 range is tuple comparison and selected gate requires equal major plus candidate>=required. Literal selected POST header forwarding remains a declared Go-versus-Session canonicalization boundary.
- Go takes one safe UTF-8 current volume ID and typed strings/bools rather than supplied mutable Volume/VolumeDetail/Resource/dict/Munch inputs and Source constructor/descriptor/location/urljoin side effects. Separate Resource/native/v2 declarations remain independent.
- Source returns None without response translation or refresh. Go adds owned VolumeID, Microversion, Discovery, Applied and Completed acknowledgement. No lookup, CurrentLocation, asynchronous wait/poll, cleanup or undo is added.
- Finite call-owned discovery and typed advertisement/evidence policy are explicit Go mappings. Complete Keystoneauth cache/link/envelope/canonicalization/discovery exception ordering and authenticated backend behavior are not claimed. Empty Go selection cannot represent Python explicit default empty separately from None.
- Opaque finite100..399 action policy maps Source<400. Accepted body IO/source/context proof remains separate from native rejection; native discarded rejected POST body IO remains a limitation. Exact method/route/body/framing/version guards constrain native retry/redirect while live auth remains.
- New11 groups cover typed body/defaults, options/capture/preflight, conditional required discovery, original admission and facade/service behavior. Accepted action IO, ordinary cap details and complete reauth/307/framing matrices use declared shared Cinder engine reuse where applicable; no new operation-specific exhaustive transport matrix is claimed. Standalone Prepare wrappers are compiled and exercised through Connection preparation, not a separate new reusable-Prepare return mutation matrix.
- Only the actual pinned v3 Proxy target is reviewed here. Root offline local HTTP race tests, vet and exact documentation compilation passed. Static Source hashes and offline Go checks do not prove authenticated OpenStack/Python runtime parity or full SDK completion.

### `python:block_storage/v3/complete_volume_migration`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.CompleteVolumeMigration`
- `github.com/JSYoo5B/go-openstacksdk.Connection.CompleteVolumeMigration`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.CompleteVolumeMigration`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeMigrationCompletionOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeMigrationCompletionError`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareVolumeMigrationCompletionOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeMigrationCompletionOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeMigrationCompletionError`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.PrepareVolumeMigrationCompletionOptions`

- Required newVolume is a literal UTF-8 body string, including empty/route-like text; it is not resolved or validated as the current route ID. new_volume and error are always sent.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsSerializeTypedDefaultsAndPresenceWithoutLookups`, `blockstorage/volume_migration_test.go:TestVolumeMigrationServiceMethodsKeepSDKContractsAndNilAPIPriority`, `connectiontest/connection_volume_migration_test.go:TestConnectionVolumeMigrationUsesCachedCinderAndSelectedVersionStillRequiresDiscovery`.
- Omitted/nil Error defaultsfalse, explicitfalse/true is always transmitted; reusable full policy factories own pointers and preserve callback once semantics.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsSerializeTypedDefaultsAndPresenceWithoutLookups`, `blockstorage/volume_migration_test.go:TestVolumeMigrationOriginalOptionsOwnFactoryPointersRetainedConfigsAndCallbackOrder`, `connectiontest/connection_volume_migration_test.go:TestConnectionVolumeMigrationOriginalsRunOnceBeforeGetterAndRetainedValuesStayDetached`.
- Direct/current ID and Connection prepare-before-Cinder admission keep operation/context/getter errors with no new-volume GET or CurrentLocation.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsCaptureSourcesBeforeOriginalsAndStopBeforeRestoration`, `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsPreflightAndCallbackFailuresPreservePriorityAndCauses`, `connectiontest/connection_volume_migration_test.go:TestConnectionVolumeMigrationOriginalsRunOnceBeforeGetterAndRetainedValuesStayDetached`, `connectiontest/connection_volume_migration_test.go:TestConnectionVolumeMigrationPreflightAndGetterFailuresRetainCauses`.
- No API minimum gate is added; selected literal/ordinary cap3.71 selection, separate Discovery and opaque Applied are operation-owned.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsSerializeTypedDefaultsAndPresenceWithoutLookups`, `connectiontest/connection_volume_migration_test.go:TestConnectionVolumeMigrationUsesCachedCinderAndSelectedVersionStillRequiresDiscovery`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsNegotiateFiniteVolumeCapWithoutChangingBackupCap`.
- Completed proves accepted body read/close/source helper completion only; Error=true informs Cinder, not SDK cleanup, migration completion, wait or undo.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsSerializeTypedDefaultsAndPresenceWithoutLookups`, `blockstorage/volume_migration_test.go:TestVolumeMigrationServiceMethodsKeepSDKContractsAndNilAPIPriority`, `connectiontest/connection_volume_migration_test.go:TestConnectionVolumeMigrationUsesCachedCinderAndSelectedVersionStillRequiresDiscovery`.
- Package/Connection/service completion and nil service labels match the actual helper. New cases verify rejected action/native policy and preflight causes; the shared Post accepted IO/source/custom-context regression retains current proof/all causes without replay and is explicit reused engine evidence.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsPreflightAndCallbackFailuresPreservePriorityAndCauses`, `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsKeepOriginalNativePolicyWhenRetryExpandsHTTPAdmission`, `blockstorage/volume_migration_test.go:TestVolumeMigrationServiceMethodsKeepSDKContractsAndNilAPIPriority`, `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionsAcceptedFaultsKeepAppliedAndEveryCauseWithoutCompletion`.

Python/Go differences:

- Source positional new_volume/error annotations do not perform coercion and allow dynamic null/nonbool values; Go deliberately uses required UTF-8 string and owned optionalbool.
- No native completion API is invented and no new-volume Resource constructor/_get_id shortcut or resolver is introduced.
- Go takes one safe UTF-8 current volume ID and typed strings/bools rather than supplied mutable Volume/VolumeDetail/Resource/dict/Munch inputs and Source constructor/descriptor/location/urljoin side effects. Separate Resource/native/v2 declarations remain independent.
- Source returns None without response translation or refresh. Go adds owned VolumeID, Microversion, Discovery, Applied and Completed acknowledgement. No lookup, CurrentLocation, asynchronous wait/poll, cleanup or undo is added.
- Finite call-owned discovery and typed advertisement/evidence policy are explicit Go mappings. Complete Keystoneauth cache/link/envelope/canonicalization/discovery exception ordering and authenticated backend behavior are not claimed. Empty Go selection cannot represent Python explicit default empty separately from None.
- Opaque finite100..399 action policy maps Source<400. Accepted body IO/source/context proof remains separate from native rejection; native discarded rejected POST body IO remains a limitation. Exact method/route/body/framing/version guards constrain native retry/redirect while live auth remains.
- New11 groups cover typed body/defaults, options/capture/preflight, conditional required discovery, original admission and facade/service behavior. Accepted action IO, ordinary cap details and complete reauth/307/framing matrices use declared shared Cinder engine reuse where applicable; no new operation-specific exhaustive transport matrix is claimed. Standalone Prepare wrappers are compiled and exercised through Connection preparation, not a separate new reusable-Prepare return mutation matrix.
- Only the actual pinned v3 Proxy target is reviewed here. Root offline local HTTP race tests, vet and exact documentation compilation passed. Static Source hashes and offline Go checks do not prove authenticated OpenStack/Python runtime parity or full SDK completion.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
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
    if cloudName == "" || volumeID == "" {
        return fmt.Errorf("CLOUD_NAME and VOLUME_ID are required")
    }
    action := os.Getenv("VOLUME_MIGRATION_ACTION")
    if action != "reset" && action != "migrate" && action != "complete" {
        return fmt.Errorf("VOLUME_MIGRATION_ACTION must be reset, migrate or complete")
    }
    path := os.Getenv("VOLUME_MIGRATION_PATH")
    if path == "" { path = "connection" }
    if path != "connection" && path != "direct" && path != "service" {
        return fmt.Errorf("unknown VOLUME_MIGRATION_PATH %q", path)
    }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    var reset blockstorage.VolumeStatusResetOpts
    var migration blockstorage.VolumeMigrationOpts
    var completion blockstorage.VolumeMigrationCompletionOpts
    var newVolume string
    switch action {
    case "reset":
        var options []blockstorage.VolumeStatusResetOption
        if value, supplied := os.LookupEnv("RESET_STATUS"); supplied {
            options = append(options, blockstorage.WithVolumeStatusResetStatus(value))
        }
        if value, supplied := os.LookupEnv("RESET_ATTACH_STATUS"); supplied {
            options = append(options, blockstorage.WithVolumeStatusResetAttachStatus(value))
        }
        if value, supplied := os.LookupEnv("RESET_MIGRATION_STATUS"); supplied {
            options = append(options, blockstorage.WithVolumeStatusResetMigrationStatus(value))
        }
        prepared, err := blockstorage.PrepareVolumeStatusResetOptions(ctx, options...)
        if err != nil { return err }
        reset = prepared
    case "migrate":
        var options []blockstorage.VolumeMigrationOption
        if value, supplied := os.LookupEnv("MIGRATION_HOST"); supplied {
            options = append(options, blockstorage.WithVolumeMigrationHost(value))
        }
        if value, supplied := os.LookupEnv("MIGRATION_CLUSTER"); supplied {
            options = append(options, blockstorage.WithVolumeMigrationCluster(value))
        }
        for _, control := range []struct {
            name string
            option func(bool) blockstorage.VolumeMigrationOption
        }{
            {"MIGRATION_FORCE_HOST_COPY", blockstorage.WithVolumeMigrationForceHostCopy},
            {"MIGRATION_LOCK_VOLUME", blockstorage.WithVolumeMigrationLockVolume},
        } {
            if text, supplied := os.LookupEnv(control.name); supplied {
                value, err := strconv.ParseBool(text)
                if err != nil { return fmt.Errorf("%s: %w", control.name, err) }
                options = append(options, control.option(value))
            }
        }
        prepared, err := blockstorage.PrepareVolumeMigrationOptions(ctx, options...)
        if err != nil { return err }
        migration = prepared
    case "complete":
        value, supplied := os.LookupEnv("NEW_VOLUME_ID")
        if !supplied { return fmt.Errorf("NEW_VOLUME_ID is required; explicit empty is allowed") }
        newVolume = value
        var options []blockstorage.VolumeMigrationCompletionOption
        if text, supplied := os.LookupEnv("MIGRATION_ERROR"); supplied {
            value, err := strconv.ParseBool(text)
            if err != nil { return fmt.Errorf("MIGRATION_ERROR: %w", err) }
            options = append(options, blockstorage.WithVolumeMigrationCompletionError(value))
        }
        prepared, err := blockstorage.PrepareVolumeMigrationCompletionOptions(ctx, options...)
        if err != nil { return err }
        completion = prepared
    }
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil { return err }
    input := blockstorage.VolumeActionRequest{VolumeID: volumeID}
    result, err := apply(ctx, conn, path, action, input, newVolume, reset, migration, completion)
    inspect(result, err)
    return err
}

func apply(ctx context.Context, conn *sdk.Connection, path, action string, input blockstorage.VolumeActionRequest, newVolume string, reset blockstorage.VolumeStatusResetOpts, migration blockstorage.VolumeMigrationOpts, completion blockstorage.VolumeMigrationCompletionOpts) (*blockstorage.VolumeActionResult, error) {
    r := blockstorage.WithVolumeStatusResetOptions(reset)
    m := blockstorage.WithVolumeMigrationOptions(migration)
    c := blockstorage.WithVolumeMigrationCompletionOptions(completion)
    if path == "connection" {
        switch action {
        case "reset": return conn.ResetVolumeStatus(ctx, input, r)
        case "migrate": return conn.MigrateVolume(ctx, input, m)
        default: return conn.CompleteVolumeMigration(ctx, input, newVolume, c)
        }
    }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { return nil, err }
    if path == "direct" {
        switch action {
        case "reset": return blockstorage.ResetVolumeStatus(ctx, cinder.RawClient(), input, r)
        case "migrate": return blockstorage.MigrateVolume(ctx, cinder.RawClient(), input, m)
        default: return blockstorage.CompleteVolumeMigration(ctx, cinder.RawClient(), input, newVolume, c)
        }
    }
    switch action {
    case "reset": return cinder.Volumes.ResetVolumeStatus(ctx, input.VolumeID, r)
    case "migrate": return cinder.Volumes.MigrateVolume(ctx, input.VolumeID, m)
    default: return cinder.Volumes.CompleteVolumeMigration(ctx, input.VolumeID, newVolume, c)
    }
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
conn.block_storage.reset_volume_status(volume_id, status="available", attach_status="detached")
conn.block_storage.migrate_volume(volume_id, host="host@backend", force_host_copy=True)
conn.block_storage.complete_volume_migration(volume_id, new_volume_id, error=False)
```

## References

- [reset Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1199)
- [migration/completion Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1336)
- [Volume.reset_status](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L209)
- [Volume.migrate/complete_migration](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L330)
- [Keystoneauth5.16.0 version_match](https://github.com/openstack/keystoneauth/blob/e759cf88c072d5bc7e21fc3fa884708ace9630af/keystoneauth1/discover.py#L433-L458)
- [Volume Actions](volume-actions.md)
- [Volume Resize Retype](volume-resize-retype.md)
- [service API](v3/volumes/README.md)
