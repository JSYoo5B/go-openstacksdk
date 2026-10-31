# Volume Backup Actions

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/restore_backup`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.RestoreBackup`
- `github.com/JSYoo5B/go-openstacksdk.Connection.RestoreVolumeBackup`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.RestoreVolumeBackup`

- RestoreVolumeBackup, Connection.RestoreVolumeBackup and API.RestoreBackup use a safe literal nonempty UTF-8 BackupID and one queryless POST backups/{ID}/restore. Only nonempty literal VolumeID/Name enter the restore object; both nil/empty are rejected. There is no destination resolver, backup lookup, generated ID, wait or implicit forced microversion; the selected microversion, including omission, remains unchanged.
  Tests: `blockstorage/backup_proxy_action_contract_test.go:TestRestoreVolumeBackupSendsOnlyNonemptyLiteralTargetsAndKeepsSelectedMicroversion`, `blockstorage/backup_restore_options_test.go:TestRestoreVolumeBackupRejectsInvalidTargetSeedAndLocalDescriptorsBeforeHTTP`, `blockstorage/backup_proxy_action_service_test.go:TestBackupProxyServiceActionsPreserveNativeTypedRestoreAndSelectedVersionReset`.
- Direct source capture precedes original options. Local context/source/route/body UTF-8/Seed checks reject before HTTP with no ResponseError; Seed must be an object and any present Seed.id must be the same safe string ID. Descriptor and owned Location failures are local rather than borrowed HTTP proof.
  Tests: `blockstorage/backup_restore_options_test.go:TestRestoreVolumeBackupRejectsInvalidTargetSeedAndLocalDescriptorsBeforeHTTP`, `blockstorage/backup_restore_options_test.go:TestRestoreVolumeBackupCapturesSourceBeforeOriginalsAndRejectsDriftBeforeRestoration`, `connectiontest/connection_volume_backup_actions_test.go:TestConnectionBackupProxyActionPreflightAndOriginalFailuresNeverSelectService`.
- Restore factories and Prepare own pointer strings, raw Seed, complete Location and original option slice; originals execute once in order. Copies after each callback prevent retained option mutation, nil/canceled/callback errors stop later originals, and sticky five-fact source drift cannot be restored by a later original. Ordinary source headers are captured without freezing live authentication.
  Tests: `blockstorage/backup_restore_options_test.go:TestRestoreVolumeBackupPrepareOwnsFactoryValuesAndRemainsReusableWithoutHTTP`, `blockstorage/backup_restore_options_test.go:TestRestoreVolumeBackupPrepareCapturesOriginalSliceAndIsolatesRetainedCallbacks`, `blockstorage/backup_restore_options_test.go:TestRestoreVolumeBackupPrepareStopsOnNilContextCancellationNilOptionAndJoinedCallbackCauses`, `blockstorage/backup_restore_options_test.go:TestRestoreVolumeBackupCapturesSourceBeforeOriginalsAndRejectsDriftBeforeRestoration`, `blockstorage/backup_proxy_action_transport_test.go:TestBackupProxyActionTransportLiveReauthAndOwnedBodyRetryReachOneAcceptedAction`.
- Connection prepares originals and local input first, snapshots CurrentLocation next, then selects only cached Cinder v3. Originals execute once per call, getter/HTTP changes do not alter that call scope, and a later call refreshes recorded scope while reusing the cached client. Getter, callback and context failures keep RestoreVolumeBackup operation and reachable custom causes.
  Tests: `connectiontest/connection_volume_backup_actions_test.go:TestConnectionBackupProxyActionOriginalsRunOnceBeforeOnlyCachedCinderSelection`, `connectiontest/connection_volume_backup_actions_test.go:TestConnectionBackupProxyActionRestoreScopeAndOptionsSnapshotPrecedeGetterAndRefreshNextCall`, `connectiontest/connection_volume_backup_actions_test.go:TestConnectionBackupProxyActionPreflightAndOriginalFailuresNeverSelectService`, `connectiontest/connection_volume_backup_actions_test.go:TestConnectionBackupProxyActionGetterFailuresKeepOuterOperationAndCallerCauses`.
- A complete explicit owned Location override replaces recorded defaults without mutating them. The merged Backup project and literal availability_zone compute the result location; foreign project IDs clear configured project names/domain fields, and missing zone is null rather than the configured zone. Explicit valid Location can bypass malformed unused recorded scope.
  Tests: `connectiontest/connection_volume_backup_actions_test.go:TestConnectionBackupProxyActionRestoreCompleteLocationOverrideRemainsOwnedAndDoesNotChangeDefaults`, `connectiontest/connection_volume_backup_actions_test.go:TestConnectionBackupProxyActionResetDoesNotConsumeMalformedRecordedScope`, `blockstorage/backup_proxy_action_contract_test.go:TestRestoreVolumeBackupMergesKnownSeedAndResponseWhileKeepingPhysicalObjectAndValueIndependent`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelLocationUsesBothRowProjectAndLiteralAvailabilityZone`.
- Response translation selects present restore, otherwise present backup, otherwise flat object. A selected valid wrong shape blocks fallback; empty/malformed ordinary UTF-8 JSON preserves the seeded logical state with no invented physical object. Invalid UTF-8 and descriptor failures are terminal with current admitted proof, nil Value, and no decoder-triggered HTTP retry.
  Tests: `blockstorage/backup_proxy_action_contract_test.go:TestRestoreVolumeBackupUsesEnvelopePresenceAndToleratesOnlyMalformedJSONWithoutInventingObject`.
- Known Seed and response attributes form a partial logical Backup overlay, with response aliases consumed in parsed object order. Unknown and runtime/control response fields remain only in the actual selected RawResource. Request volume/name and native backup_id do not fabricate logical id/name. Response id remains arbitrary JSON and may replace the returned logical BackupID without changing the fixed request target.
  Tests: `blockstorage/backup_proxy_action_contract_test.go:TestRestoreVolumeBackupMergesKnownSeedAndResponseWhileKeepingPhysicalObjectAndValueIndependent`, `blockstorage/backup_proxy_action_contract_test.go:TestRestoreVolumeBackupRetainsArbitraryResponseIDWithoutChangingFixedRequestTarget`, `blockstorage/backup_proxy_action_service_test.go:TestBackupProxyServiceActionsPreserveNativeTypedRestoreAndSelectedVersionReset`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelProjectAliasesFollowParsedDuplicateFirstInsertionOrder`.
- Successful Value is the full nullable 23-Body-field plus computed-location Backup view, independently of the actual response object. Reused descriptors retain untyped nullable IDs/status/strings, ordinary Boolean truthiness rather than BoolStr, list wrapping, nullable ordinary dictionary metadata, and eager size/object_count integer conversion. Container integer values become zero; invalid superscript-digit strings fail. Exact-decimal Go truthiness and Python float boundaries are explicit differences.
  Tests: `blockstorage/backup_proxy_action_contract_test.go:TestRestoreVolumeBackupMergesKnownSeedAndResponseWhileKeepingPhysicalObjectAndValueIndependent`, `blockstorage/backup_proxy_action_contract_test.go:TestRestoreVolumeBackupUsesEnvelopePresenceAndToleratesOnlyMalformedJSONWithoutInventingObject`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelHasTwentyFourNullableFieldsAndOwnDescriptorOrder`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelOrdinaryBooleanDescriptorsAreNullableAndDoNotUseBoolStr`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelLinksWrapSingleValuesAndMetadataUsesOrdinaryNullableDict`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelBothIntegerDescriptorsConvertBeforeAnySelection`.
- Applied owns actual full admitted bytes/header/status. Backup owns only the selected physical object with actual metadata; Value and BackupID own the merged logical output independently. Sparse response raw objects never invent cached fields or seeded ID, and mutations of each result part do not alter the others.
  Tests: `blockstorage/backup_proxy_action_contract_test.go:TestRestoreVolumeBackupMergesKnownSeedAndResponseWhileKeepingPhysicalObjectAndValueIndependent`, `blockstorage/backup_proxy_action_contract_test.go:TestRestoreVolumeBackupRetainsArbitraryResponseIDWithoutChangingFixedRequestTarget`, `blockstorage/backup_proxy_action_contract_test.go:TestRestoreVolumeBackupUsesEnvelopePresenceAndToleratesOnlyMalformedJSONWithoutInventingObject`.
- The original exchange admits HTTP100..399 and rejects >=400 through the native error path before logical translation. Native rejected body/header/status remain reachable without Applied or invented ResponseError; expanded mutable native OkCodes cannot turn an original-policy rejection into success.
  Tests: `blockstorage/backup_proxy_action_contract_test.go:TestRestoreVolumeBackupSendsOnlyNonemptyLiteralTargetsAndKeepsSelectedMicroversion`, `blockstorage/backup_proxy_action_contract_test.go:TestRestoreVolumeBackupUsesEnvelopePresenceAndToleratesOnlyMalformedJSONWithoutInventingObject`, `blockstorage/backup_proxy_action_contract_test.go:TestBackupProxyPublicActionsRejectNativeHTTPWithoutBorrowedProofOrLogicalCompletion`, `blockstorage/backup_proxy_action_transport_test.go:TestBackupProxyActionTransportNativeAndExpandedRejectionsNeverBecomeApplied`.
- Admitted Read/Close, five-fact source drift and custom context failures retain current Applied and independently owned ResponseError proof with every joined cause, but publish no Value/Backup. Close occurs once and accepted body IO faults do not call native RetryFunc or trigger SDK replay/rollback.
  Tests: `blockstorage/backup_proxy_action_transport_test.go:TestBackupProxyActionTransportAcceptedFaultsKeepCurrentProofAndEveryCause`, `connectiontest/connection_volume_backup_actions_test.go:TestConnectionBackupProxyActionAcceptedIOKeepsActualAppliedAndJoinedSourceCancellation`.
- Live native reauthentication and bounded retry retain fixed selected source and captured ordinary headers with owned serialized JSON body. Mutable native body/decoder/retention options or Provider/Endpoint/ResourceBase/Type/Microversion changes stop resend; original rejection and callback causes remain reachable.
  Tests: `blockstorage/backup_proxy_action_transport_test.go:TestBackupProxyActionTransportLiveReauthAndOwnedBodyRetryReachOneAcceptedAction`, `blockstorage/backup_proxy_action_transport_test.go:TestBackupProxyActionTransportRetryMutationStopsBeforeSecondPhysicalRequest`.
- The closed shared POST helper checks exact physical serialized body, ContentLength, TransferEncoding and framing headers on redirects as well as ordinary attempts, closes the inspected Body once, and reinstalls an owned valid body. Changed method/path/origin or body/framing and injected Read/Close faults stop before a second physical request. Sticky faults survive native retry restoration/swallowing and preserve source/context/callback causes.
  Tests: `blockstorage/backup_proxy_action_transport_test.go:TestBackupProxyActionTransportSameTarget307CannotChangePhysicalBodyOrFraming`, `blockstorage/backup_proxy_action_transport_test.go:TestBackupProxyActionTransportPhysicalReadCloseFaultsStayTerminalAfterRetryRestoration`, `blockstorage/backup_proxy_action_transport_test.go:TestBackupProxyActionTransportRedirectScopeCannotChangeMethodPathOrOrigin`.
- Service RestoreBackup owns the same factories/options and wraps failures with its actual outer method/resource while package and Connection keep their own operation names. Existing native RestoreFromBackup remains an independent accepted202 three-string Restore DTO, permitting empty native options; it does not replace the new merged Backup view or acquire this declaration review.
  Tests: `blockstorage/backup_proxy_action_service_test.go:TestBackupProxyServiceActionsExposeOwnedLocalFactoriesAndCorrectOuterErrorLabels`, `blockstorage/backup_proxy_action_service_test.go:TestBackupProxyServiceActionsPreserveNativeTypedRestoreAndSelectedVersionReset`, `connectiontest/connection_volume_backup_actions_test.go:TestConnectionBackupProxyActionGetterFailuresKeepOuterOperationAndCallerCauses`.

Python/Go differences:

- Go VolumeID/Name are optional copied UTF-8 string pointers; nil/empty are omitted and both falsey fail locally. Python volume may be Resource/dict/literal and truthy volume triggers Resource._get_id before Backup construction; nonstring dynamic values and warning machinery are not typed Go inputs. Deprecated volume_id overrides volume in the source decorator, and warning-as-error effects are not recreated.
- Go Seed is an owned canonical object overlay with a matching safe ID, rather than mutation of a supplied connected Backup. Seed unknown/computed/runtime controls are ignored by the known state; source constructors/update descriptors and mutable dirty/cache state are not reproduced. Complete owned location plus configured/token scope replace constructor connection identity.
- Restore returns merged logical RawMessage BackupID and full nullable Backup Value separately from Applied and actual selected RawResource; source returns the same mutated Backup. Response id is an untyped attribute and is not the fixed request route; native backup_id stays unknown raw evidence rather than being mapped to logical id. No destination request values are backfilled as response fields.
- Go preserves literal raw unknown response fields/number spelling as physical evidence, while the logical view uses ordinary JSON descriptors and known-only overlay. Exact decimal Boolean/project equality versus Python parsed-float underflow/rounding, parser/UTF8/string-surrogate/depth/backend boundaries and Python mutable dictionary identities are explicit differences. Parsed duplicate collapse retains first insertion position for alias consumption; raw wire member text is not promoted to semantic Python identity.
- Ordinary UTF-8 malformed/empty JSON tolerance and present-envelope precedence are bounded response mappings. Go rejects invalid UTF8 and valid wrong shapes with actual response proof; Python selected JSON backend/encoding/exception behavior is not universally replicated. Existing native RestoreFromBackup still has its accepted202/three-string result and allows empty native options.
- Go accepts safe literal string BackupID, rejecting permissive Python urljoin/coercion and mutable Resource/dict/Munch/None inputs. Source constructor/session/cache/object identities are not reproduced; this is a coherent owned Go mapping of this direct Proxy declaration, not automatic parity for those separate dependencies.
- Source raise_from_response rejects >=400; the owned Go exchange admits finite HTTP100..399, preserving native rejected errors. Admitted result/error bytes/header are independent snapshots with joined IO/source/context proof. Native rejected response body Read/Close discarded errors remain a bounded transport limitation rather than universally preserved IO.
- Safe fixed route/method and closed exact POST body/framing plus sticky five-fact source guards constrain mutable native retry/redirect hooks. Ordinary native header hooks, live token, reauthentication and bounded retry remain. Parent context and operation wrapping replace Python exception/session identities; no SDK rollback/replay or wait is added.
- Source bindings are static pinned declarations: the historical restore33 and reset38 graphs overlap and are not new Python probes or authenticated OpenStack integration. Only this actual v3 Proxy target is classified; Backup method, generic Resource, native/v2, deprecated alias, other cloud/Proxy helpers and full SDK remain independent.

### `python:block_storage/v3/reset_backup_status`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.ResetBackupStatus`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ResetVolumeBackupStatus`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.ResetVolumeBackupStatus`

- ResetVolumeBackupStatus, Connection.ResetVolumeBackupStatus and API.ResetBackupStatus require a safe literal BackupID and explicit UTF-8 Status string, including empty, unknown, whitespace/control and Unicode values. One queryless POST backups/{ID}/action contains exactly os-reset_status.status. There is no status enum/default, resolver, wait or semantic status confirmation.
  Tests: `blockstorage/backup_proxy_action_contract_test.go:TestResetVolumeBackupStatusForwardsEveryUTF8StringWithForcedMicroversionAndOpaqueAck`, `blockstorage/backup_proxy_action_contract_test.go:TestResetVolumeBackupStatusPreflightRejectsUnsafeIDInvalidUTF8AndInvalidSourceBeforeHTTP`, `blockstorage/backup_proxy_action_service_test.go:TestBackupProxyServiceActionsPreserveNativeTypedRestoreAndSelectedVersionReset`.
- Reset forces action microversion3.64 independently of the selected client version, without discovery/default/cap and without mutating the original client or headers. Actual modern and legacy action headers must remain consistent through native retry and physical redirects; valid case-folded aliases are permitted while deletion, conflict and OmitHeaders are terminal.
  Tests: `blockstorage/backup_proxy_action_contract_test.go:TestResetVolumeBackupStatusForwardsEveryUTF8StringWithForcedMicroversionAndOpaqueAck`, `blockstorage/backup_proxy_action_transport_test.go:TestBackupProxyActionTransportResetRetryVersionAliasesAreValidatedBeforeResend`, `blockstorage/backup_proxy_action_transport_test.go:TestBackupProxyActionTransportPhysicalResetHeaderRejectionSurvivesRetryRepair`.
- Direct Reset captures source before ID/status validation. Invalid context/source/route/UTF-8 status fails before HTTP; Connection validates context/provider/input before selecting only cached Cinder v3. There are no reset options or CurrentLocation/Backup descriptor construction; malformed unused recorded scope cannot block an otherwise valid opaque reset.
  Tests: `blockstorage/backup_proxy_action_contract_test.go:TestResetVolumeBackupStatusPreflightRejectsUnsafeIDInvalidUTF8AndInvalidSourceBeforeHTTP`, `connectiontest/connection_volume_backup_actions_test.go:TestConnectionBackupProxyActionPreflightAndOriginalFailuresNeverSelectService`, `connectiontest/connection_volume_backup_actions_test.go:TestConnectionBackupProxyActionOriginalsRunOnceBeforeOnlyCachedCinderSelection`, `connectiontest/connection_volume_backup_actions_test.go:TestConnectionBackupProxyActionResetDoesNotConsumeMalformedRecordedScope`.
- Original HTTP100..399 acknowledgement is opaque: invalid UTF-8, malformed JSON, empty body or arbitrary JSON need no decoder. Applied holds actual bytes/header/status, BackupID is the fixed string input, and Completed becomes true only after exchange/read/Close/final captured-source validation succeeds. Completed is helper acknowledgement rather than a polled status postcondition.
  Tests: `blockstorage/backup_proxy_action_contract_test.go:TestResetVolumeBackupStatusForwardsEveryUTF8StringWithForcedMicroversionAndOpaqueAck`, `blockstorage/backup_proxy_action_service_test.go:TestBackupProxyServiceActionsPreserveNativeTypedRestoreAndSelectedVersionReset`, `connectiontest/connection_volume_backup_actions_test.go:TestConnectionBackupProxyActionResetDoesNotConsumeMalformedRecordedScope`.
- Original-policy >=400 keeps native rejected code/body/header and has no Applied, Completed or borrowed ResponseError. Expanded native OkCodes do not admit a rejected response; no reset parser or SDK replay is introduced.
  Tests: `blockstorage/backup_proxy_action_contract_test.go:TestBackupProxyPublicActionsRejectNativeHTTPWithoutBorrowedProofOrLogicalCompletion`, `blockstorage/backup_proxy_action_transport_test.go:TestBackupProxyActionTransportNativeAndExpandedRejectionsNeverBecomeApplied`.
- Admitted Read/Close/source/custom context faults retain actual Applied and independently owned ResponseError with all joined causes and Completed=false. Close occurs once and accepted faults do not invoke native RetryFunc; reset success never requires logical Backup conversion.
  Tests: `blockstorage/backup_proxy_action_transport_test.go:TestBackupProxyActionTransportAcceptedFaultsKeepCurrentProofAndEveryCause`, `connectiontest/connection_volume_backup_actions_test.go:TestConnectionBackupProxyActionAcceptedIOKeepsActualAppliedAndJoinedSourceCancellation`.
- Native reauthentication and bounded retry use live tokens with captured ordinary headers and owned serialized action body. Mutated native body/decoder/retention policy or five-fact source stops before resend with original and callback causes retained.
  Tests: `blockstorage/backup_proxy_action_transport_test.go:TestBackupProxyActionTransportLiveReauthAndOwnedBodyRetryReachOneAcceptedAction`, `blockstorage/backup_proxy_action_transport_test.go:TestBackupProxyActionTransportRetryMutationStopsBeforeSecondPhysicalRequest`.
- The action body/framing is verified at every physical POST, including same-target307 redirects. Changed bytes, size/framing, nil body and injected Read/Close failures are terminal; valid inspected Body is closed once and replaced by an owned body. Sticky rejection survives a repairing native RetryFunc and preserves joined source/context/callback causes.
  Tests: `blockstorage/backup_proxy_action_transport_test.go:TestBackupProxyActionTransportSameTarget307CannotChangePhysicalBodyOrFraming`, `blockstorage/backup_proxy_action_transport_test.go:TestBackupProxyActionTransportPhysicalReadCloseFaultsStayTerminalAfterRetryRestoration`.
- Method/path/origin redirect changes are rejected before a second transport call and before foreign credentials escape. Connection getter and service preflight failures keep exact ResetVolumeBackupStatus or ResetBackupStatus operation context and custom cancellation causes; cached selection is Cinder v3 only.
  Tests: `blockstorage/backup_proxy_action_transport_test.go:TestBackupProxyActionTransportRedirectScopeCannotChangeMethodPathOrOrigin`, `connectiontest/connection_volume_backup_actions_test.go:TestConnectionBackupProxyActionGetterFailuresKeepOuterOperationAndCallerCauses`, `blockstorage/backup_proxy_action_service_test.go:TestBackupProxyServiceActionsExposeOwnedLocalFactoriesAndCorrectOuterErrorLabels`.
- New service ResetBackupStatus returns owned opaque Applied plus Completed and forces3.64, while existing native ResetStatus remains selected-version accepted202 error-only. The actual source graph is v3 Proxy.reset_backup_status→Backup.reset_status→Backup._action; source Resource/session/warning identities and None return are explicit Go mappings, independently of native/v2/deprecated alias/Resource review status.
  Tests: `blockstorage/backup_proxy_action_service_test.go:TestBackupProxyServiceActionsPreserveNativeTypedRestoreAndSelectedVersionReset`, `blockstorage/backup_proxy_action_service_test.go:TestBackupProxyServiceActionsExposeOwnedLocalFactoriesAndCorrectOuterErrorLabels`, `connectiontest/connection_volume_backup_actions_test.go:TestConnectionBackupProxyActionResetDoesNotConsumeMalformedRecordedScope`.

Python/Go differences:

- Go Status is an explicitly supplied UTF8 string with no default/enum, including empty/control/unknown values. Python annotations do not runtime-reject null, numeric, bool or containers; those dynamic values are outside this typed request domain.
- Reset uses only safe ID and opaque acknowledgement and consumes no CurrentLocation or Backup descriptors. Python _get_resource may construct/update a mutable connected Resource/dict/Munch/None with location/descriptor/constructor side effects; those side effects are deliberately omitted from this ID-only Go operation.
- Source Proxy.reset_backup_status and Backup.reset_status return implicit None and do not update the supplied Backup status. Go adds independently owned Applied plus Completed, which means complete helper acknowledgement rather than semantic status change confirmation. Failed admitted exchanges retain Applied but leave Completed false.
- Source Backup._action explicitly uses its3.64 class maximum; Go forces3.64 on an operation copy and validates modern/legacy physical action headers. Selected original client policy remains unchanged. Existing native ResetStatus stays selected-version/accepted202/error-only, and deprecated reset_backup/reset warning behavior is not recreated.
- Go accepts safe literal string BackupID, rejecting permissive Python urljoin/coercion and mutable Resource/dict/Munch/None inputs. Source constructor/session/cache/object identities are not reproduced; this is a coherent owned Go mapping of this direct Proxy declaration, not automatic parity for those separate dependencies.
- Source raise_from_response rejects >=400; the owned Go exchange admits finite HTTP100..399, preserving native rejected errors. Admitted result/error bytes/header are independent snapshots with joined IO/source/context proof. Native rejected response body Read/Close discarded errors remain a bounded transport limitation rather than universally preserved IO.
- Safe fixed route/method and closed exact POST body/framing plus sticky five-fact source guards constrain mutable native retry/redirect hooks. Ordinary native header hooks, live token, reauthentication and bounded retry remain. Parent context and operation wrapping replace Python exception/session identities; no SDK rollback/replay or wait is added.
- Source bindings are static pinned declarations: the historical restore33 and reset38 graphs overlap and are not new Python probes or authenticated OpenStack integration. Only this actual v3 Proxy target is classified; Backup method, generic Resource, native/v2, deprecated alias, other cloud/Proxy helpers and full SDK remain independent.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cloudName, backupID := os.Getenv("CLOUD_NAME"), os.Getenv("BACKUP_ID")
	if cloudName == "" || backupID == "" {
		return fmt.Errorf("CLOUD_NAME and BACKUP_ID are required")
	}
	path, action := os.Getenv("BACKUP_PATH"), os.Getenv("BACKUP_ACTION")
	if path == "" {
		path = "connection"
	}
	if action == "" {
		action = "restore"
	}
	switch path {
	case "connection", "direct", "service", "native":
	default:
		return fmt.Errorf("unknown BACKUP_PATH %q", path)
	}
	volumeID, name := os.Getenv("RESTORE_VOLUME_ID"), os.Getenv("RESTORE_NAME")
	status, statusPresent := os.LookupEnv("BACKUP_STATUS")
	switch action {
	case "restore":
		if volumeID == "" && name == "" {
			return fmt.Errorf("RESTORE_VOLUME_ID or RESTORE_NAME is required")
		}
	case "reset":
		if !statusPresent {
			return fmt.Errorf("BACKUP_STATUS must be defined; an empty value is allowed")
		}
	default:
		return fmt.Errorf("unknown BACKUP_ACTION %q", action)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
	if err != nil {
		return err
	}
	if action == "reset" {
		input := blockstorage.ResetVolumeBackupStatusRequest{BackupID: backupID, Status: status}
		if path == "connection" {
			result, err := conn.ResetVolumeBackupStatus(ctx, input)
			inspectReset(result)
			return err
		}
		service, err := conn.BlockStorageV3(ctx)
		if err != nil {
			return err
		}
		switch path {
		case "direct":
			result, err := blockstorage.ResetVolumeBackupStatus(ctx, service.RawClient(), input)
			inspectReset(result)
			return err
		case "service":
			result, err := service.Backups.ResetBackupStatus(ctx, backupID, status)
			inspectReset(result)
			return err
		default:
			return service.Backups.ResetStatus(ctx, backupID, backups.ResetStatusOpts{Status: status})
		}
	}
	options := []blockstorage.RestoreVolumeBackupOption{
		blockstorage.WithRestoreVolumeBackupVolumeID(volumeID),
		blockstorage.WithRestoreVolumeBackupName(name),
	}
	if seed, present := os.LookupEnv("BACKUP_SEED"); present {
		options = append(options, blockstorage.WithRestoreVolumeBackupSeed(json.RawMessage(seed)))
	}
	policy, err := blockstorage.PrepareRestoreVolumeBackupOptions(ctx, options...)
	if err != nil {
		return err
	}
	input := blockstorage.RestoreVolumeBackupRequest{BackupID: backupID}
	if path == "connection" {
		result, err := conn.RestoreVolumeBackup(ctx, input, blockstorage.WithRestoreVolumeBackupOptions(policy))
		inspectRestore(result)
		return err
	}
	service, err := conn.BlockStorageV3(ctx)
	if err != nil {
		return err
	}
	if path == "native" {
		result, err := service.Backups.RestoreFromBackup(ctx, backupID, backups.RestoreOpts{VolumeID: volumeID, Name: name})
		if result != nil {
			fmt.Println("native restore:", result.BackupID, result.VolumeID, result.VolumeName)
		}
		return err
	}
	location, err := conn.CurrentLocation()
	if err != nil {
		return err
	}
	policy, err = blockstorage.PrepareRestoreVolumeBackupOptions(ctx,
		blockstorage.WithRestoreVolumeBackupOptions(policy),
		blockstorage.WithRestoreVolumeBackupLocation(location))
	if err != nil {
		return err
	}
	if path == "direct" {
		result, err := blockstorage.RestoreVolumeBackup(ctx, service.RawClient(), input, blockstorage.WithRestoreVolumeBackupOptions(policy))
		inspectRestore(result)
		return err
	}
	result, err := service.Backups.RestoreBackup(ctx, backupID, backups.WithRestoreBackupOptions(policy))
	inspectRestore(result)
	return err
}

func inspectRestore(result *blockstorage.RestoreVolumeBackupResult) {
	if result == nil {
		return
	}
	fmt.Println("merged logical backup ID JSON:", string(result.BackupID))
	inspectApplied(result.Applied)
	if result.Backup != nil {
		actual := result.Backup.Clone()
		fmt.Println("actual selected fields:", len(actual.Body), "physical backup_id:", string(actual.Body["backup_id"]))
	}
	if result.Value != nil {
		formatted, err := json.MarshalIndent(result.Value, "", "  ")
		if err == nil {
			fmt.Println("merged nullable Backup:", string(formatted))
		}
	}
}

func inspectReset(result *blockstorage.ResetVolumeBackupStatusResult) {
	if result == nil {
		return
	}
	fmt.Println("requested backup ID:", result.BackupID, "helper completed:", result.Completed)
	inspectApplied(result.Applied)
}

func inspectApplied(page *blockstorage.VolumeBackupMutationPage) {
	if page != nil {
		fmt.Println("admitted HTTP:", page.StatusCode, "actual body bytes:", len(page.Body))
	}
}
```

## References

- [Proxy.restore_backup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1871)
- [Backup.restore](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L200)
