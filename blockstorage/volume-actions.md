# Volume Actions

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/reserve_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.ReserveVolume`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ReserveVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.ReserveVolume`

- ReserveVolume sends exactly {"os-reserve":null}. Use one explicit safe nonempty UTF-8 volume ID and the exact pinned null action through queryless POST volumes/{ID}/action. Unicode literal IDs are preserved, and VolumeDetail never inserts detail into the route. There is no UUID-only restriction, lookup, Resource conversion, status enum, options/defaults, builder, wait, refresh or rollback.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsPostSourceNullBodiesAndAcceptOpaqueHTTPReplies`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsRejectInvalidIDAndSourceBeforeAnyHTTP`.
- Direct calls capture the Cinder service and validate context/role/provider/endpoint/resource/header before ID admission. Connection checks context/provider/ID before selecting only cached Cinder v3, without CurrentLocation or recorded scope consumption. Getter errors retain the actual outer operation and custom cancellation causes before any request or fabricated proof.
  Tests: `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsUseOnlyCachedCinderAndIgnoreLocation`, `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsPreflightNeverSelectsCinder`, `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsGetterErrorsKeepOperationAndCancellation`.
- Selected nonempty microversion wins literally without discovery/cap. Otherwise operation-owned finite negotiation uses the Volume3.71 ceiling and server maximum before optional minimum, with tuple length/order retained; minimum above ceiling omits version and missing maximum ignores unused minimum. Original client/cache stays unchanged, each call owns its selected policy, and independent Backup import retains3.64 on the same server advertisement.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsSelectedVersionKeepsOwnedHeadersAndLiveNativeAuthentication`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsNegotiateFiniteVolumeCapWithoutChangingBackupCap`, `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsOwnDiscoveryPerCallAndKeepStageProof`, `internal/cloudsnapshot/backup_import_version_test.go:TestBackupImportVersionIndependentStringAndHeaderCases`, `internal/cloudsnapshot/backup_import_version_test.go:TestBackupImportVersionIndependentMaximumMinimumCases`.
- Finite discovery preserves captured encoded reverse-proxy prefix, admits200/300 supported body forms and first usable v3, and keeps admitted pages separate from action Applied. Only clean concrete native404/405 permits unavailable fallback. Accepted malformed/null/invalid-UTF8 or other discovery failures stop before action, with current ResponseError proof; later native rejection cannot borrow an earlier admitted page as current error proof.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsDiscoveryFailuresRetainCurrentStageAndNeverBorrowProof`.
- Action replies are opaque finite HTTP100..399 acknowledgements including empty, malformed and invalid-UTF8 bytes. Applied owns actual body/header/status, independently of separate calls or error proof. Completed means accepted Read/Close/source validation succeeded, never a server status postcondition. Original-policy>=400 and expanded native OkCodes retain native rejected evidence without Applied or Completed. Existing native Reserve/Unreserve/BeginDetaching keep their {} body and narrower status policy; all four new service methods and nil API preserve actual labels.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionAdmissionUsesOriginalSub400PolicyAndKeepsNativeRejections`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionServiceMethodsPreserveNativeBodiesCodesAndNilAPILabels`.
- Accepted Read/Close, captured-source drift and custom cancellation retain actual Applied and every joined cause, with Completed=false and an independently owned ResponseError. Body closes once and accepted IO errors invoke no caller retry or SDK replay; proof storage never aliases the caller response or another result.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsAcceptedFaultsKeepActualProofAndAllCauses`, `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionsAcceptedFaultsKeepAppliedAndEveryCauseWithoutCompletion`.
- Live native401 reauthentication and bounded503 caller retry retain exact owned null body, captured ordinary headers, selected source and operation version. Body/decoder/retention changes, version conflicts or source drift stop before resend while original/callback/custom cancellation causes remain reachable.
  Tests: `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionLiveReauthAndBoundedRetryKeepLiteralNullBody`, `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionRetryMutationStopsBodyVersionAndSourceBeforeResend`.
- Every physical POST including same-target307 redirects verifies exact method/path/origin, body bytes/framing and modern/legacy version policy. Changed body/framing/header/target stops before second transport attempt. Physical body Read/Close and header faults remain sticky after caller repair, close inspected body once and preserve actual causes.
  Tests: `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionPhysicalRedirectMutationRemainsStickyAfterRepair`.

Python/Go differences:

- Go takes safe literal UTF-8 ID only. Source accepts mutable Volume/VolumeDetail/Resource/dict/Munch inputs through _get_resource with constructor/to_dict/descriptor/computed location side effects and Python urljoin coercion. Go skips those identities/side effects and name lookup deliberately; this does not classify separate Resource declarations.
- Source Proxy and Resource methods return None without translating response or updating local fields. Go adds independently owned Microversion, Discovery, Applied and Completed acknowledgement with literal input VolumeID. No semantic server-state confirmation, wait, follow-up GET or rollback is added.
- Volume uses class ceiling3.71 while Backup import keeps3.64 through the shared Cinder policy. Selected nonempty version is forwarded literally; Keystoneauth Session canonicalization, dynamic nonstring defaults, full Adapter cache/link/envelope/header discovery and deployment HTTP behavior are outside this owned Go mapping.
- Finite same-origin body-only first usable v3 discovery, typed advertisement shapes/status and clean concrete404/405 fallback are explicit mapping policy. Accepted malformed discovery is terminal with current proof rather than reproducing every caught Keystoneauth DiscoveryFailure/None path or endpoint-data exception order.
- Utility tuple selection retains maximum-before-optional-minimum, length/order and missing bounds. Shared arbitrary-precision Unicode16 decimal/underscore string parsing and latest handling are bounded Go behavior; exact installed CPython digit limits, nonstring input normalization and pinned dependency runtime are not claimed.
- Owned finite HTTP100..399 policy preserves opaque action bytes and admitted IO/source/context proof separately from native>=400 rejection. Native discarded rejected POST body Read/Close faults remain a documented limitation; mutable OkCodes cannot widen SDK acknowledgement. No response model conversion is introduced.
- Fixed source/method/route/body/framing and operation microversion ownership constrain mutable native retry/redirect hooks while live authentication and bounded caller retry remain. Native Reserve/Unreserve/BeginDetaching keep {} and narrower success codes; new null action APIs remain separate.
- Only the actual pinned v3 Proxy declaration is mapped here. Native/v2/Resource/inherited targets and direct bootable/readonly setters remain independent. Source hashes, static review, Go compile and offline HTTP/race tests do not establish authenticated OpenStack or Python SDK runtime parity or full SDK completion.

### `python:block_storage/v3/unreserve_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.UnreserveVolume`
- `github.com/JSYoo5B/go-openstacksdk.Connection.UnreserveVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.UnreserveVolume`

- UnreserveVolume sends exactly {"os-unreserve":null}. Use one explicit safe nonempty UTF-8 volume ID and the exact pinned null action through queryless POST volumes/{ID}/action. Unicode literal IDs are preserved, and VolumeDetail never inserts detail into the route. There is no UUID-only restriction, lookup, Resource conversion, status enum, options/defaults, builder, wait, refresh or rollback.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsPostSourceNullBodiesAndAcceptOpaqueHTTPReplies`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsRejectInvalidIDAndSourceBeforeAnyHTTP`.
- Direct calls capture the Cinder service and validate context/role/provider/endpoint/resource/header before ID admission. Connection checks context/provider/ID before selecting only cached Cinder v3, without CurrentLocation or recorded scope consumption. Getter errors retain the actual outer operation and custom cancellation causes before any request or fabricated proof.
  Tests: `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsUseOnlyCachedCinderAndIgnoreLocation`, `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsPreflightNeverSelectsCinder`, `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsGetterErrorsKeepOperationAndCancellation`.
- Selected nonempty microversion wins literally without discovery/cap. Otherwise operation-owned finite negotiation uses the Volume3.71 ceiling and server maximum before optional minimum, with tuple length/order retained; minimum above ceiling omits version and missing maximum ignores unused minimum. Original client/cache stays unchanged, each call owns its selected policy, and independent Backup import retains3.64 on the same server advertisement.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsSelectedVersionKeepsOwnedHeadersAndLiveNativeAuthentication`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsNegotiateFiniteVolumeCapWithoutChangingBackupCap`, `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsOwnDiscoveryPerCallAndKeepStageProof`, `internal/cloudsnapshot/backup_import_version_test.go:TestBackupImportVersionIndependentStringAndHeaderCases`, `internal/cloudsnapshot/backup_import_version_test.go:TestBackupImportVersionIndependentMaximumMinimumCases`.
- Finite discovery preserves captured encoded reverse-proxy prefix, admits200/300 supported body forms and first usable v3, and keeps admitted pages separate from action Applied. Only clean concrete native404/405 permits unavailable fallback. Accepted malformed/null/invalid-UTF8 or other discovery failures stop before action, with current ResponseError proof; later native rejection cannot borrow an earlier admitted page as current error proof.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsDiscoveryFailuresRetainCurrentStageAndNeverBorrowProof`.
- Action replies are opaque finite HTTP100..399 acknowledgements including empty, malformed and invalid-UTF8 bytes. Applied owns actual body/header/status, independently of separate calls or error proof. Completed means accepted Read/Close/source validation succeeded, never a server status postcondition. Original-policy>=400 and expanded native OkCodes retain native rejected evidence without Applied or Completed. Existing native Reserve/Unreserve/BeginDetaching keep their {} body and narrower status policy; all four new service methods and nil API preserve actual labels.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionAdmissionUsesOriginalSub400PolicyAndKeepsNativeRejections`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionServiceMethodsPreserveNativeBodiesCodesAndNilAPILabels`.
- Accepted Read/Close, captured-source drift and custom cancellation retain actual Applied and every joined cause, with Completed=false and an independently owned ResponseError. Body closes once and accepted IO errors invoke no caller retry or SDK replay; proof storage never aliases the caller response or another result.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsAcceptedFaultsKeepActualProofAndAllCauses`, `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionsAcceptedFaultsKeepAppliedAndEveryCauseWithoutCompletion`.
- Live native401 reauthentication and bounded503 caller retry retain exact owned null body, captured ordinary headers, selected source and operation version. Body/decoder/retention changes, version conflicts or source drift stop before resend while original/callback/custom cancellation causes remain reachable.
  Tests: `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionLiveReauthAndBoundedRetryKeepLiteralNullBody`, `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionRetryMutationStopsBodyVersionAndSourceBeforeResend`.
- Every physical POST including same-target307 redirects verifies exact method/path/origin, body bytes/framing and modern/legacy version policy. Changed body/framing/header/target stops before second transport attempt. Physical body Read/Close and header faults remain sticky after caller repair, close inspected body once and preserve actual causes.
  Tests: `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionPhysicalRedirectMutationRemainsStickyAfterRepair`.

Python/Go differences:

- Go takes safe literal UTF-8 ID only. Source accepts mutable Volume/VolumeDetail/Resource/dict/Munch inputs through _get_resource with constructor/to_dict/descriptor/computed location side effects and Python urljoin coercion. Go skips those identities/side effects and name lookup deliberately; this does not classify separate Resource declarations.
- Source Proxy and Resource methods return None without translating response or updating local fields. Go adds independently owned Microversion, Discovery, Applied and Completed acknowledgement with literal input VolumeID. No semantic server-state confirmation, wait, follow-up GET or rollback is added.
- Volume uses class ceiling3.71 while Backup import keeps3.64 through the shared Cinder policy. Selected nonempty version is forwarded literally; Keystoneauth Session canonicalization, dynamic nonstring defaults, full Adapter cache/link/envelope/header discovery and deployment HTTP behavior are outside this owned Go mapping.
- Finite same-origin body-only first usable v3 discovery, typed advertisement shapes/status and clean concrete404/405 fallback are explicit mapping policy. Accepted malformed discovery is terminal with current proof rather than reproducing every caught Keystoneauth DiscoveryFailure/None path or endpoint-data exception order.
- Utility tuple selection retains maximum-before-optional-minimum, length/order and missing bounds. Shared arbitrary-precision Unicode16 decimal/underscore string parsing and latest handling are bounded Go behavior; exact installed CPython digit limits, nonstring input normalization and pinned dependency runtime are not claimed.
- Owned finite HTTP100..399 policy preserves opaque action bytes and admitted IO/source/context proof separately from native>=400 rejection. Native discarded rejected POST body Read/Close faults remain a documented limitation; mutable OkCodes cannot widen SDK acknowledgement. No response model conversion is introduced.
- Fixed source/method/route/body/framing and operation microversion ownership constrain mutable native retry/redirect hooks while live authentication and bounded caller retry remain. Native Reserve/Unreserve/BeginDetaching keep {} and narrower success codes; new null action APIs remain separate.
- Only the actual pinned v3 Proxy declaration is mapped here. Native/v2/Resource/inherited targets and direct bootable/readonly setters remain independent. Source hashes, static review, Go compile and offline HTTP/race tests do not establish authenticated OpenStack or Python SDK runtime parity or full SDK completion.

### `python:block_storage/v3/begin_volume_detaching`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.BeginVolumeDetaching`
- `github.com/JSYoo5B/go-openstacksdk.Connection.BeginVolumeDetaching`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.BeginVolumeDetaching`

- BeginVolumeDetaching sends exactly {"os-begin_detaching":null}. Use one explicit safe nonempty UTF-8 volume ID and the exact pinned null action through queryless POST volumes/{ID}/action. Unicode literal IDs are preserved, and VolumeDetail never inserts detail into the route. There is no UUID-only restriction, lookup, Resource conversion, status enum, options/defaults, builder, wait, refresh or rollback.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsPostSourceNullBodiesAndAcceptOpaqueHTTPReplies`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsRejectInvalidIDAndSourceBeforeAnyHTTP`.
- Direct calls capture the Cinder service and validate context/role/provider/endpoint/resource/header before ID admission. Connection checks context/provider/ID before selecting only cached Cinder v3, without CurrentLocation or recorded scope consumption. Getter errors retain the actual outer operation and custom cancellation causes before any request or fabricated proof.
  Tests: `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsUseOnlyCachedCinderAndIgnoreLocation`, `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsPreflightNeverSelectsCinder`, `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsGetterErrorsKeepOperationAndCancellation`.
- Selected nonempty microversion wins literally without discovery/cap. Otherwise operation-owned finite negotiation uses the Volume3.71 ceiling and server maximum before optional minimum, with tuple length/order retained; minimum above ceiling omits version and missing maximum ignores unused minimum. Original client/cache stays unchanged, each call owns its selected policy, and independent Backup import retains3.64 on the same server advertisement.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsSelectedVersionKeepsOwnedHeadersAndLiveNativeAuthentication`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsNegotiateFiniteVolumeCapWithoutChangingBackupCap`, `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsOwnDiscoveryPerCallAndKeepStageProof`, `internal/cloudsnapshot/backup_import_version_test.go:TestBackupImportVersionIndependentStringAndHeaderCases`, `internal/cloudsnapshot/backup_import_version_test.go:TestBackupImportVersionIndependentMaximumMinimumCases`.
- Finite discovery preserves captured encoded reverse-proxy prefix, admits200/300 supported body forms and first usable v3, and keeps admitted pages separate from action Applied. Only clean concrete native404/405 permits unavailable fallback. Accepted malformed/null/invalid-UTF8 or other discovery failures stop before action, with current ResponseError proof; later native rejection cannot borrow an earlier admitted page as current error proof.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsDiscoveryFailuresRetainCurrentStageAndNeverBorrowProof`.
- Action replies are opaque finite HTTP100..399 acknowledgements including empty, malformed and invalid-UTF8 bytes. Applied owns actual body/header/status, independently of separate calls or error proof. Completed means accepted Read/Close/source validation succeeded, never a server status postcondition. Original-policy>=400 and expanded native OkCodes retain native rejected evidence without Applied or Completed. Existing native Reserve/Unreserve/BeginDetaching keep their {} body and narrower status policy; all four new service methods and nil API preserve actual labels.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionAdmissionUsesOriginalSub400PolicyAndKeepsNativeRejections`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionServiceMethodsPreserveNativeBodiesCodesAndNilAPILabels`.
- Accepted Read/Close, captured-source drift and custom cancellation retain actual Applied and every joined cause, with Completed=false and an independently owned ResponseError. Body closes once and accepted IO errors invoke no caller retry or SDK replay; proof storage never aliases the caller response or another result.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsAcceptedFaultsKeepActualProofAndAllCauses`, `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionsAcceptedFaultsKeepAppliedAndEveryCauseWithoutCompletion`.
- Live native401 reauthentication and bounded503 caller retry retain exact owned null body, captured ordinary headers, selected source and operation version. Body/decoder/retention changes, version conflicts or source drift stop before resend while original/callback/custom cancellation causes remain reachable.
  Tests: `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionLiveReauthAndBoundedRetryKeepLiteralNullBody`, `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionRetryMutationStopsBodyVersionAndSourceBeforeResend`.
- Every physical POST including same-target307 redirects verifies exact method/path/origin, body bytes/framing and modern/legacy version policy. Changed body/framing/header/target stops before second transport attempt. Physical body Read/Close and header faults remain sticky after caller repair, close inspected body once and preserve actual causes.
  Tests: `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionPhysicalRedirectMutationRemainsStickyAfterRepair`.

Python/Go differences:

- Go takes safe literal UTF-8 ID only. Source accepts mutable Volume/VolumeDetail/Resource/dict/Munch inputs through _get_resource with constructor/to_dict/descriptor/computed location side effects and Python urljoin coercion. Go skips those identities/side effects and name lookup deliberately; this does not classify separate Resource declarations.
- Source Proxy and Resource methods return None without translating response or updating local fields. Go adds independently owned Microversion, Discovery, Applied and Completed acknowledgement with literal input VolumeID. No semantic server-state confirmation, wait, follow-up GET or rollback is added.
- Volume uses class ceiling3.71 while Backup import keeps3.64 through the shared Cinder policy. Selected nonempty version is forwarded literally; Keystoneauth Session canonicalization, dynamic nonstring defaults, full Adapter cache/link/envelope/header discovery and deployment HTTP behavior are outside this owned Go mapping.
- Finite same-origin body-only first usable v3 discovery, typed advertisement shapes/status and clean concrete404/405 fallback are explicit mapping policy. Accepted malformed discovery is terminal with current proof rather than reproducing every caught Keystoneauth DiscoveryFailure/None path or endpoint-data exception order.
- Utility tuple selection retains maximum-before-optional-minimum, length/order and missing bounds. Shared arbitrary-precision Unicode16 decimal/underscore string parsing and latest handling are bounded Go behavior; exact installed CPython digit limits, nonstring input normalization and pinned dependency runtime are not claimed.
- Owned finite HTTP100..399 policy preserves opaque action bytes and admitted IO/source/context proof separately from native>=400 rejection. Native discarded rejected POST body Read/Close faults remain a documented limitation; mutable OkCodes cannot widen SDK acknowledgement. No response model conversion is introduced.
- Fixed source/method/route/body/framing and operation microversion ownership constrain mutable native retry/redirect hooks while live authentication and bounded caller retry remain. Native Reserve/Unreserve/BeginDetaching keep {} and narrower success codes; new null action APIs remain separate.
- Only the actual pinned v3 Proxy declaration is mapped here. Native/v2/Resource/inherited targets and direct bootable/readonly setters remain independent. Source hashes, static review, Go compile and offline HTTP/race tests do not establish authenticated OpenStack or Python SDK runtime parity or full SDK completion.

### `python:block_storage/v3/abort_volume_detaching`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.AbortVolumeDetaching`
- `github.com/JSYoo5B/go-openstacksdk.Connection.AbortVolumeDetaching`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.AbortVolumeDetaching`

- AbortVolumeDetaching sends exactly {"os-roll_detaching":null}. Use one explicit safe nonempty UTF-8 volume ID and the exact pinned null action through queryless POST volumes/{ID}/action. Unicode literal IDs are preserved, and VolumeDetail never inserts detail into the route. There is no UUID-only restriction, lookup, Resource conversion, status enum, options/defaults, builder, wait, refresh or rollback.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsPostSourceNullBodiesAndAcceptOpaqueHTTPReplies`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsRejectInvalidIDAndSourceBeforeAnyHTTP`.
- Direct calls capture the Cinder service and validate context/role/provider/endpoint/resource/header before ID admission. Connection checks context/provider/ID before selecting only cached Cinder v3, without CurrentLocation or recorded scope consumption. Getter errors retain the actual outer operation and custom cancellation causes before any request or fabricated proof.
  Tests: `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsUseOnlyCachedCinderAndIgnoreLocation`, `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsPreflightNeverSelectsCinder`, `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsGetterErrorsKeepOperationAndCancellation`.
- Selected nonempty microversion wins literally without discovery/cap. Otherwise operation-owned finite negotiation uses the Volume3.71 ceiling and server maximum before optional minimum, with tuple length/order retained; minimum above ceiling omits version and missing maximum ignores unused minimum. Original client/cache stays unchanged, each call owns its selected policy, and independent Backup import retains3.64 on the same server advertisement.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsSelectedVersionKeepsOwnedHeadersAndLiveNativeAuthentication`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsNegotiateFiniteVolumeCapWithoutChangingBackupCap`, `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsOwnDiscoveryPerCallAndKeepStageProof`, `internal/cloudsnapshot/backup_import_version_test.go:TestBackupImportVersionIndependentStringAndHeaderCases`, `internal/cloudsnapshot/backup_import_version_test.go:TestBackupImportVersionIndependentMaximumMinimumCases`.
- Finite discovery preserves captured encoded reverse-proxy prefix, admits200/300 supported body forms and first usable v3, and keeps admitted pages separate from action Applied. Only clean concrete native404/405 permits unavailable fallback. Accepted malformed/null/invalid-UTF8 or other discovery failures stop before action, with current ResponseError proof; later native rejection cannot borrow an earlier admitted page as current error proof.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsDiscoveryFailuresRetainCurrentStageAndNeverBorrowProof`.
- Action replies are opaque finite HTTP100..399 acknowledgements including empty, malformed and invalid-UTF8 bytes. Applied owns actual body/header/status, independently of separate calls or error proof. Completed means accepted Read/Close/source validation succeeded, never a server status postcondition. Original-policy>=400 and expanded native OkCodes retain native rejected evidence without Applied or Completed. Existing native Reserve/Unreserve/BeginDetaching keep their {} body and narrower status policy; all four new service methods and nil API preserve actual labels.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionAdmissionUsesOriginalSub400PolicyAndKeepsNativeRejections`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionServiceMethodsPreserveNativeBodiesCodesAndNilAPILabels`.
- Accepted Read/Close, captured-source drift and custom cancellation retain actual Applied and every joined cause, with Completed=false and an independently owned ResponseError. Body closes once and accepted IO errors invoke no caller retry or SDK replay; proof storage never aliases the caller response or another result.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsAcceptedFaultsKeepActualProofAndAllCauses`, `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionsAcceptedFaultsKeepAppliedAndEveryCauseWithoutCompletion`.
- Live native401 reauthentication and bounded503 caller retry retain exact owned null body, captured ordinary headers, selected source and operation version. Body/decoder/retention changes, version conflicts or source drift stop before resend while original/callback/custom cancellation causes remain reachable.
  Tests: `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionLiveReauthAndBoundedRetryKeepLiteralNullBody`, `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionRetryMutationStopsBodyVersionAndSourceBeforeResend`.
- Every physical POST including same-target307 redirects verifies exact method/path/origin, body bytes/framing and modern/legacy version policy. Changed body/framing/header/target stops before second transport attempt. Physical body Read/Close and header faults remain sticky after caller repair, close inspected body once and preserve actual causes.
  Tests: `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionPhysicalRedirectMutationRemainsStickyAfterRepair`.

Python/Go differences:

- Go takes safe literal UTF-8 ID only. Source accepts mutable Volume/VolumeDetail/Resource/dict/Munch inputs through _get_resource with constructor/to_dict/descriptor/computed location side effects and Python urljoin coercion. Go skips those identities/side effects and name lookup deliberately; this does not classify separate Resource declarations.
- Source Proxy and Resource methods return None without translating response or updating local fields. Go adds independently owned Microversion, Discovery, Applied and Completed acknowledgement with literal input VolumeID. No semantic server-state confirmation, wait, follow-up GET or rollback is added.
- Volume uses class ceiling3.71 while Backup import keeps3.64 through the shared Cinder policy. Selected nonempty version is forwarded literally; Keystoneauth Session canonicalization, dynamic nonstring defaults, full Adapter cache/link/envelope/header discovery and deployment HTTP behavior are outside this owned Go mapping.
- Finite same-origin body-only first usable v3 discovery, typed advertisement shapes/status and clean concrete404/405 fallback are explicit mapping policy. Accepted malformed discovery is terminal with current proof rather than reproducing every caught Keystoneauth DiscoveryFailure/None path or endpoint-data exception order.
- Utility tuple selection retains maximum-before-optional-minimum, length/order and missing bounds. Shared arbitrary-precision Unicode16 decimal/underscore string parsing and latest handling are bounded Go behavior; exact installed CPython digit limits, nonstring input normalization and pinned dependency runtime are not claimed.
- Owned finite HTTP100..399 policy preserves opaque action bytes and admitted IO/source/context proof separately from native>=400 rejection. Native discarded rejected POST body Read/Close faults remain a documented limitation; mutable OkCodes cannot widen SDK acknowledgement. No response model conversion is introduced.
- Fixed source/method/route/body/framing and operation microversion ownership constrain mutable native retry/redirect hooks while live authentication and bounded caller retry remain. Native Reserve/Unreserve/BeginDetaching keep {} and narrower success codes; new null action APIs remain separate.
- Only the actual pinned v3 Proxy declaration is mapped here. Native/v2/Resource/inherited targets and direct bootable/readonly setters remain independent. Source hashes, static review, Go compile and offline HTTP/race tests do not establish authenticated OpenStack or Python SDK runtime parity or full SDK completion.

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

    "github.com/gophercloud/gophercloud/v2"
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
    action := os.Getenv("STATE_ACTION")
    if action == "" {
        action = "reserve"
    }
    switch action {
    case "reserve", "unreserve", "begin-detaching", "abort-detaching":
    default:
        return fmt.Errorf("unknown STATE_ACTION %q", action)
    }
    path := os.Getenv("STATE_ACTION_PATH")
    if path == "" {
        path = "connection"
    }
    switch path {
    case "connection", "direct", "service":
    default:
        return fmt.Errorf("unknown STATE_ACTION_PATH %q", path)
    }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil {
        return err
    }
    input := blockstorage.VolumeActionRequest{VolumeID: volumeID}
    result, err := apply(ctx, conn, action, path, input)
    inspect(result, err)
    return err
}

func apply(ctx context.Context, conn *sdk.Connection, action, path string, input blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
    if path == "connection" {
        calls := map[string]func(context.Context, blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error){
            "reserve": conn.ReserveVolume,
            "unreserve": conn.UnreserveVolume,
            "begin-detaching": conn.BeginVolumeDetaching,
            "abort-detaching": conn.AbortVolumeDetaching,
        }
        return calls[action](ctx, input)
    }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil {
        return nil, err
    }
    if path == "direct" {
        calls := map[string]func(context.Context, *gophercloud.ServiceClient, blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error){
            "reserve": blockstorage.ReserveVolume,
            "unreserve": blockstorage.UnreserveVolume,
            "begin-detaching": blockstorage.BeginVolumeDetaching,
            "abort-detaching": blockstorage.AbortVolumeDetaching,
        }
        return calls[action](ctx, cinder.RawClient(), input)
    }
    calls := map[string]func(context.Context, string) (*blockstorage.VolumeActionResult, error){
        "reserve": cinder.Volumes.ReserveVolume,
        "unreserve": cinder.Volumes.UnreserveVolume,
        "begin-detaching": cinder.Volumes.BeginVolumeDetaching,
        "abort-detaching": cinder.Volumes.AbortVolumeDetaching,
    }
    return calls[action](ctx, input.VolumeID)
}

func inspect(result *blockstorage.VolumeActionResult, err error) {
    if result != nil {
        fmt.Printf("volume ID: %q; microversion: %q\n", result.VolumeID, result.Microversion)
        for i, page := range result.Discovery {
            if page != nil {
                fmt.Println("admitted discovery response:", i, page.StatusCode, len(page.Body))
            }
        }
        if result.Applied != nil {
            fmt.Println("admitted action response:", result.Applied.StatusCode)
            fmt.Printf("opaque action body: %q\n", result.Applied.Body)
        }
        fmt.Println("helper acknowledgement completed:", result.Completed)
    }
    if err != nil {
        fmt.Println("action error:", err)
        var physical *resource.ResponseError
        if errors.As(err, &physical) {
            fmt.Println("current admitted error response:", physical.StatusCode, len(physical.Body))
        }
        if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
            fmt.Println("parent context interrupted the call")
        }
    }
}
```

```python
import openstack

conn = openstack.connect(cloud="devstack")
volume_id = "actual-volume-id"
conn.block_storage.reserve_volume(volume_id)
conn.block_storage.unreserve_volume(volume_id)
conn.block_storage.begin_volume_detaching(volume_id)
conn.block_storage.abort_volume_detaching(volume_id)
```

## References

- [v3 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1422)
- [Volume methods](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L405)
- [Source `_get_resource`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L563)
- [Volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L146)
- [Source utility](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L358)
- [native Reserve/Unreserve/BeginDetaching](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/volumes/requests.go#L363)
- [cloud bootable helper](volume-mutations.md)
- [Volume Flags](volume-flags.md)
