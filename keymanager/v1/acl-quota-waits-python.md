# Acl Quota Waits Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:key_manager/v1/wait_for_status`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.API.WaitFor`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders.API.WaitFor`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets.API.WaitFor`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Wait`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithUnlimitedWait`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithFailureStates`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPollInterval`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithTimeout`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithStatusAttribute`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithProgressCallback`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.API.WaitForStatus`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.New`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.Container`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.Order`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.Secret`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ForContainer`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ForOrder`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ForSecret`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithOptions`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithInterval`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithTimeout`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithUnlimitedWait`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithProgressCallback`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithHeader`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithFailures`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithAttribute`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.SecretStore`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ForSecretStore`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.SecretACL`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ForSecretACL`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.Quota`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ForQuota`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ProjectQuota`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ForProjectQuota`

- Order WaitFor with WithUnlimitedWait and WithFailureStates("ERROR") repeats GET orders/{id}, returns on a case-insensitive target match, fails with resource.FailedStateError only for ERROR, keeps polling through other error-like states and reports progress 0 after each nonterminal observation.
  Tests: `keymanager/v1/orders/python_parity_test.go:TestPythonOrderWaitForStatus`.
- WithStatusAttribute selects another string field and WithTimeout ends the wait with an error wrapping context.DeadlineExceeded.
  Tests: `keymanager/v1/orders/python_parity_test.go:TestPythonOrderWaitForStatus`.
- Owned concrete Container/Order/Secret constructors snapshot synchronized Resource and physical evidence; cached lower-equivalent nullable/empty/string status returns without HTTP and inputs/results do not alias.
  Tests: `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitCachedStatesAndIndependentTargetSnapshot`, `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitNormalizedComparisonIsNotUnicodeEqualFold`.
- Shared Collection.Wait/WaitDeleted and prepared WaitPolicy supply native context/timer/progress mechanics. Source defaults are status-unbounded/delete120/interval2, explicit timeout zero/negative exhausts before polling, zero interval shortens the pause. Compound callback/cancel/source errors retain the last accepted receipt.
  Tests: `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitDefaultTimeoutsAndZeroIntervalPolicy`, `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitCallbackFailureCancellationAndSourceAreTerminal`.
- Guarded common REST preserves rejected and accepted Read/Close, sticky source change and custom cancellation causes with actual status/body/header; native RetryFunc nil does not replay faulty physical responses. Clean503 retries retain owned selected version/headers and live token.
  Tests: `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitCompoundPhysicalFaultsRetainActualProofAndNeverReplay`, `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitCleanNativeRetryPreservesSelectedPolicyAndLiveToken`.
- State normalization preserves Python Unicode lower, null and empty values. Source failure ERROR defaults and explicit empty failures disable detection; callback0 occurs only on nonterminal observations. Declared custom nullable string attributes are available.
  Tests: `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitNormalizedComparisonIsNotUnicodeEqualFold`, `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitNullableAndInvalidSelectedAttributesKeepProof`, `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitFailureDefaultsExplicitEmptyAndCustomAttribute`.
- Synchronized descriptor values persist on omission, incoming null/false/zero replaces prior values, malformed generic metadata leaves existing view and records actual response, and fixed RequestID does not follow returned id/HREF.
  Tests: `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitSynchronizedSeedMergeAndGenericMalformedBody`.
- Secret polling follows prior Resource payload_content_type before current response content_types.default. Current payload_content_type alone does not choose the first payload. Complete metadata+payload success precedes status completion and binary/text proof is owned.
  Tests: `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerSecretWaitPayloadSelectionUsesPreviousSynchronizedResource`, `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerSecretWaitMetadataDefaultPayloadAndStrictFailures`.
- Option/state preflight, clone isolation and captured source/header/version ownership prevent HTTP when the owned contract is invalid.
  Tests: `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitOptionOwnershipAndPreflight`.
- Partial finite SecretStore GET polling added to existing Container/Order/Secret targets; actual raw/default/nullable projection, immutable literal identity, synchronized presence merge and Source bool/ref descriptors reuse existing guarded reader and prepared wait policy. Other known fetchable targets remain missing; no full named mapping claimed.
  Tests: `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitLiteralRouteRawProjectionAndDefaults`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitNativeCachedSnapshotKeepsExplicitIdentityAndNoFakeEnvelope`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitOrderedAliasesAndSynchronizedMerge`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitBooleanDescriptorUsesSourceFloatTruthiness`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitDescriptorAndShapeErrorsKeepActualPage`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitDeleteCleanAbsenceAndDisabledBackendLimitation`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitPreflightFailureAndCallbackPolicies`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitCompoundPhysicalFaultsNeverRetryOrReauthenticate`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitCleanRetryReauthOwnHeadersAndLiveToken`.
- Partial concrete SecretACL and effective Quota target constructors own stock parent/singleton routes and do not append response IDs/HREFs or the internal current-project selector. Literal defaults have no fabricated HTTP status, byte Envelope or Wire; nullable/untyped declared values and actual envelope/object/header/status are independently owned.
  Tests: `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLWaitOwnsParentRouteAndSourceDictDefaults`, `keymanager/v1/waits/python_quota_contracts_test.go:TestPythonKeymanagerQuotaWaitOwnsNoIDRouteAndEnvelopeRawDescriptors`.
- ForSecretACL snapshots actual Metadata.Body or native Read/ACLRef when Body is absent, retains immutable native SecretID, and does not fabricate Wire/Envelope. ForQuota requires original Body. Cached normalized string/null matches return before timers/HTTP and constructor/result snapshots do not alias. SecretACL URI secret_id is available for declared getattr selection without inserting it into to_dict Body.
  Tests: `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLWaitSnapshotsCachedNativeAndURIWithoutFakeEnvelope`, `keymanager/v1/waits/python_quota_contracts_test.go:TestPythonKeymanagerQuotaWaitCachedSnapshotAndNullableDefaults`.
- SecretACL read descriptor preserves objects/None and converts other JSON-domain values to empty dict exactly as pinned fields._convert_type(dict); literal Wire remains raw. Incoming declared null/false/zero/object/large-integer presence overlays synchronized values; omission and generic malformed/empty JSON retain previous descriptors. Quota unwraps quotas when present and uses root-object fallback otherwise.
  Tests: `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLWaitRawPresenceMergeAndReadDictCoercion`, `keymanager/v1/waits/python_quota_contracts_test.go:TestPythonKeymanagerQuotaWaitMergesFlatFallbackAndNullPresence`.
- SecretACL and ordinary Quota do not declare status; default status wait is unsupported before HTTP rather than invented null/active. Declared selected attributes must be string or null: cached non-string rejects before HTTP, fetched non-string/nonobject/envelope-invalid/UTF8-invalid retains actual proof. Numeric quota values are not converted to status strings.
  Tests: `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLWaitMissingStatusAndTypedAttributePreflight`, `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLQuotaWaitNonobjectAndSelectedValueErrorsKeepActualProof`, `keymanager/v1/waits/python_quota_contracts_test.go:TestPythonKeymanagerQuotaWaitEnvelopeShapeAndNumericAttributesAreExplicitFailures`.
- Status-free deletion fetches GET, ignores undeclared wire status:deleted, calls progress0 only after nonterminal accepted observations, and completes only on clean GET404 while returning the original Resource snapshot plus actual absence receipt. Status GET404 remains an error. No DELETE or configuration-deletion acknowledgement is synthesized.
  Tests: `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLQuotaDeleteSkipsUndeclaredStatusAndRetainsOriginalAt404`.
- Both finite targets reuse reader captured source/header/version and common prepared WaitPolicy. Twenty combined physical fault lanes retain actual method/URL/status/body/header and Read/Close/sticky-source/custom-cancel/context causes without retry/reauth. Native 401 pointer ErrUnableToReauthenticate exposes actual response via ErrOriginal and physical causes via ErrReauth; 404 physical faults do not prove deletion.
  Tests: `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLQuotaPhysicalFailuresNeverBecomeAbsenceOrRetry`.
- Clean 401 renewal and 503 retry remain native for both finite targets, preserve fixed stock routes and selected version/header policy, observe refreshed token and leave original client policy unmodified.
  Tests: `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLQuotaCleanRetryReauthPreservesStockRouteAndOwnedHeaders`.
- Concrete admin ProjectQuota target fetches only the original fixed escaped project route with project_quotas envelope/root fallback and seven nullable untyped Body descriptors. Raw extensions and actual receipt remain separate; no status is invented and cached/default matching nullable strings return without polling.
  Tests: `keymanager/v1/waits/python_project_quota_contracts_test.go:TestPythonProjectQuotaWaitFixedRouteRawDescriptorsAndDefaults`, `keymanager/v1/waits/python_project_quota_contracts_test.go:TestPythonProjectQuotaWaitCachedRecordOwnsSnapshotsAndNoStatus`.
- ProjectQuota metadata merges prior fields through generic malformed JSON and explicit-null observations, while invalid shapes/numeric selected attributes retain response errors. Deletion polls GET and completes only on clean404, ignoring undeclared status:deleted; accepted/rejected Close faults cannot become absence or replay.
  Tests: `keymanager/v1/waits/python_project_quota_contracts_test.go:TestPythonProjectQuotaWaitMergeIgnoresMalformedJSONAndPreservesNull`, `keymanager/v1/waits/python_project_quota_contracts_test.go:TestPythonProjectQuotaWaitDeleteOnlyCleanGET404`, `keymanager/v1/waits/python_project_quota_contracts_test.go:TestPythonProjectQuotaWaitShapeNumericAndPhysicalFailures`.

Python/Go differences:

- Legacy generated collection waits retain their historical five-minute policy. The concrete waits.API lane defaults status to unlimited/ERROR and deletion120s with interval2s, matching the pinned helper defaults.
- Legacy resource.WithPollInterval requires positive durations; concrete waits.WithInterval(0) uses Source100ms/min-positive-timeout coercion.
- Timeouts and failures are Python ResourceTimeout and ResourceFailure; Go returns context deadline errors and resource.FailedStateError.
- Finite SDK-owned concrete Target constructors with immutable routes and independent record snapshots replace Python arbitrary Resource subclasses, same-object mutation and dynamic getattr. All known pinned fetchable concrete targets now include admin ProjectQuota; arbitrary class/subtyping remains SDK-R1 as a deliberate finite Go API boundary.
- Source normalized nullable/empty/string states use PythonLower and private exact comparison tokens, not broad Unicode EqualFold. Go failures accept strings only, callbacks are typed func(int)error, declared attributes must be nullable strings.
- The SDK validates source/headers and applied option callbacks before cached-state success; Source returns its cached object before consulting session or many invalid timer/failure settings. Negative interval is a pre-HTTP Go guard rather than Python sleep-time ValueError after an observation.
- Status timeout defaults to unlimited, deletion timeout120s, interval2s. Nil Timeout is omission; explicit Unlimited removes the deletion SDK limit. Explicit zero/negative timeout fails before first GET unless a cached status already matches. Zero interval uses Source 100ms/min-positive-timeout pause.
- Fixed original route, selected microversion or absence and reserved headers survive native attempts. Python mutable response id can change subsequent fetch routing; Go preserves RequestID and leaves response id/HREF passive. Live auth token and ordinary native retry headers remain native.
- Source custom Secret.fetch does not raise HTTP errors itself; Go admits100..399 and rejects400..599 strictly with full physical causes, so payload404 cannot establish metadata absence. Accepted UTF-8 text is decoded strictly, opaque bytes remain Payload plus JSON base64 Resource values.
- Container/Order generic malformed/empty JSON preserves synchronized attributes as Source ValueError handling does; valid JSON null/nonobject fails. Secret malformed JSON fails. Go requires valid UTF-8 metadata and eagerly projects finite declared casts/aliases using the existing shared projector.
- Source projectors provide nullable descriptor defaults and locationnull; canonical wire aliases win deterministically. Unknown metadata is retained only in Wire. No Connection location/session HTTP cache, dirty-resource commit or arbitrary Python runtime value parity is claimed.
- Clean metadata404 deletion returns an owned original Resource view plus the new404 proof channels rather than Source object identity. Fetched deleted returns the updated view. Errors retain last actual evidence but return Resource=nil.
- Fixed literal/request selector routing never follows response HREF/native convenience ID; Source mutable id routing is not reproduced.
- Cached native Metadata.Body has no original object key order or byte envelope: sorted-key constructor aliases and no fabricated Envelope.
- Computed location ownednull and strict UTF8 finite JSON/declared attributes remain existing SDK-R1/C1 policies.
- Pinned generic Resource.fetch requires_id=True overrides SecretACL.requires_id=False and does not carry the public getter requires_id=False option. Generic waiting on a getter ACL/Quota with id=None fails before polling, or appends an artificial ID if set. Concrete Go targets deliberately correct this pinned request preparation defect by owning stock /secrets/{secretID}/acl and /quotas routes.
- SecretACL URI secret_id is selected from immutable request scope but is not added to Resource Body/to_dict; current-project is an internal wait selector and is never a wire project/member ID or synthesized Resource.id.
- No status descriptor is invented for ACL/Quota: default status waits fail preflight; deletion skips absent status and waits for clean GET404. Stock ACL deletion restores implicit default ACL GET200 and project override deletion leaves effective quota GET200; GET absence observation does not prove settings deletion.
- ACL wait read projection follows Source getter dict coercion while existing native secretacls API uses its independent stock object validation. Strict UTF8, declared string/null wait attributes and owned-null computed location remain bounded SDK policies.
- Admin ProjectQuota is a member route unlike effective Quota; its original project ID is seeded and immutable while response IDs remain passive. Like Quota/ACL it has no status descriptor and stock override reset can leave GET200, so absence waiting is not proof of settings deletion.
- This mapping covers finite SDK-owned concrete target constructors rather than arbitrary Python Resource/runtime subclass/cache/location behavior. SDK-R1/C1/S1 and actual deployment/permission/release validation remain explicit whole-SDK gaps.

### `python:key_manager/v1/wait_for_delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.API.WaitForDeletion`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders.API.WaitForDeletion`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets.API.WaitForDeletion`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.WaitDeleted`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithTimeout`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPollInterval`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.New`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.Container`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.Order`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.Secret`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ForContainer`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ForOrder`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ForSecret`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithOptions`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithInterval`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithTimeout`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithUnlimitedWait`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithProgressCallback`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithHeader`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.SecretStore`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ForSecretStore`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.SecretACL`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ForSecretACL`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.Quota`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ForQuota`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ProjectQuota`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ForProjectQuota`

- Order WaitForDeletion with WithTimeout(120s) repeats GET orders/{id} until 404 and then succeeds.
  Tests: `keymanager/v1/orders/python_parity_test.go:TestPythonOrderWaitForDelete`.
- Owned concrete Container/Order/Secret constructors snapshot synchronized Resource and physical evidence; cached lower-equivalent nullable/empty/string status returns without HTTP and inputs/results do not alias.
  Tests: `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitCachedStatesAndIndependentTargetSnapshot`, `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitNormalizedComparisonIsNotUnicodeEqualFold`.
- Shared Collection.Wait/WaitDeleted and prepared WaitPolicy supply native context/timer/progress mechanics. Source defaults are status-unbounded/delete120/interval2, explicit timeout zero/negative exhausts before polling, zero interval shortens the pause. Compound callback/cancel/source errors retain the last accepted receipt.
  Tests: `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitDefaultTimeoutsAndZeroIntervalPolicy`, `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitCallbackFailureCancellationAndSourceAreTerminal`.
- Guarded common REST preserves rejected and accepted Read/Close, sticky source change and custom cancellation causes with actual status/body/header; native RetryFunc nil does not replay faulty physical responses. Clean503 retries retain owned selected version/headers and live token.
  Tests: `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitCompoundPhysicalFaultsRetainActualProofAndNeverReplay`, `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitCleanNativeRetryPreservesSelectedPolicyAndLiveToken`.
- Deletion always performs the first metadata GET even when cached deleted. A clean metadata404 returns original Resource with actual404 evidence, fetched deleted returns fetched Resource; null/nonstring/forbidden/server errors are terminal.
  Tests: `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitDeletionOriginalAndFetchedReturnContracts`.
- Secret payload404 is a terminal marked payload failure rather than absence of the Secret. Metadata and payload native proof stay separate, and ACTIVE metadata cannot complete status wait when payload failed.
  Tests: `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerSecretWaitMetadataDefaultPayloadAndStrictFailures`, `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitCompoundPhysicalFaultsRetainActualProofAndNeverReplay`.
- Partial finite SecretStore GET polling added to existing Container/Order/Secret targets; actual raw/default/nullable projection, immutable literal identity, synchronized presence merge and Source bool/ref descriptors reuse existing guarded reader and prepared wait policy. Other known fetchable targets remain missing; no full named mapping claimed.
  Tests: `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitLiteralRouteRawProjectionAndDefaults`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitNativeCachedSnapshotKeepsExplicitIdentityAndNoFakeEnvelope`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitOrderedAliasesAndSynchronizedMerge`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitBooleanDescriptorUsesSourceFloatTruthiness`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitDescriptorAndShapeErrorsKeepActualPage`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitDeleteCleanAbsenceAndDisabledBackendLimitation`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitPreflightFailureAndCallbackPolicies`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitCompoundPhysicalFaultsNeverRetryOrReauthenticate`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitCleanRetryReauthOwnHeadersAndLiveToken`.
- Partial concrete SecretACL and effective Quota target constructors own stock parent/singleton routes and do not append response IDs/HREFs or the internal current-project selector. Literal defaults have no fabricated HTTP status, byte Envelope or Wire; nullable/untyped declared values and actual envelope/object/header/status are independently owned.
  Tests: `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLWaitOwnsParentRouteAndSourceDictDefaults`, `keymanager/v1/waits/python_quota_contracts_test.go:TestPythonKeymanagerQuotaWaitOwnsNoIDRouteAndEnvelopeRawDescriptors`.
- ForSecretACL snapshots actual Metadata.Body or native Read/ACLRef when Body is absent, retains immutable native SecretID, and does not fabricate Wire/Envelope. ForQuota requires original Body. Cached normalized string/null matches return before timers/HTTP and constructor/result snapshots do not alias. SecretACL URI secret_id is available for declared getattr selection without inserting it into to_dict Body.
  Tests: `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLWaitSnapshotsCachedNativeAndURIWithoutFakeEnvelope`, `keymanager/v1/waits/python_quota_contracts_test.go:TestPythonKeymanagerQuotaWaitCachedSnapshotAndNullableDefaults`.
- SecretACL read descriptor preserves objects/None and converts other JSON-domain values to empty dict exactly as pinned fields._convert_type(dict); literal Wire remains raw. Incoming declared null/false/zero/object/large-integer presence overlays synchronized values; omission and generic malformed/empty JSON retain previous descriptors. Quota unwraps quotas when present and uses root-object fallback otherwise.
  Tests: `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLWaitRawPresenceMergeAndReadDictCoercion`, `keymanager/v1/waits/python_quota_contracts_test.go:TestPythonKeymanagerQuotaWaitMergesFlatFallbackAndNullPresence`.
- SecretACL and ordinary Quota do not declare status; default status wait is unsupported before HTTP rather than invented null/active. Declared selected attributes must be string or null: cached non-string rejects before HTTP, fetched non-string/nonobject/envelope-invalid/UTF8-invalid retains actual proof. Numeric quota values are not converted to status strings.
  Tests: `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLWaitMissingStatusAndTypedAttributePreflight`, `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLQuotaWaitNonobjectAndSelectedValueErrorsKeepActualProof`, `keymanager/v1/waits/python_quota_contracts_test.go:TestPythonKeymanagerQuotaWaitEnvelopeShapeAndNumericAttributesAreExplicitFailures`.
- Status-free deletion fetches GET, ignores undeclared wire status:deleted, calls progress0 only after nonterminal accepted observations, and completes only on clean GET404 while returning the original Resource snapshot plus actual absence receipt. Status GET404 remains an error. No DELETE or configuration-deletion acknowledgement is synthesized.
  Tests: `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLQuotaDeleteSkipsUndeclaredStatusAndRetainsOriginalAt404`.
- Both finite targets reuse reader captured source/header/version and common prepared WaitPolicy. Twenty combined physical fault lanes retain actual method/URL/status/body/header and Read/Close/sticky-source/custom-cancel/context causes without retry/reauth. Native 401 pointer ErrUnableToReauthenticate exposes actual response via ErrOriginal and physical causes via ErrReauth; 404 physical faults do not prove deletion.
  Tests: `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLQuotaPhysicalFailuresNeverBecomeAbsenceOrRetry`.
- Clean 401 renewal and 503 retry remain native for both finite targets, preserve fixed stock routes and selected version/header policy, observe refreshed token and leave original client policy unmodified.
  Tests: `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLQuotaCleanRetryReauthPreservesStockRouteAndOwnedHeaders`.
- Concrete admin ProjectQuota target fetches only the original fixed escaped project route with project_quotas envelope/root fallback and seven nullable untyped Body descriptors. Raw extensions and actual receipt remain separate; no status is invented and cached/default matching nullable strings return without polling.
  Tests: `keymanager/v1/waits/python_project_quota_contracts_test.go:TestPythonProjectQuotaWaitFixedRouteRawDescriptorsAndDefaults`, `keymanager/v1/waits/python_project_quota_contracts_test.go:TestPythonProjectQuotaWaitCachedRecordOwnsSnapshotsAndNoStatus`.
- ProjectQuota metadata merges prior fields through generic malformed JSON and explicit-null observations, while invalid shapes/numeric selected attributes retain response errors. Deletion polls GET and completes only on clean404, ignoring undeclared status:deleted; accepted/rejected Close faults cannot become absence or replay.
  Tests: `keymanager/v1/waits/python_project_quota_contracts_test.go:TestPythonProjectQuotaWaitMergeIgnoresMalformedJSONAndPreservesNull`, `keymanager/v1/waits/python_project_quota_contracts_test.go:TestPythonProjectQuotaWaitDeleteOnlyCleanGET404`, `keymanager/v1/waits/python_project_quota_contracts_test.go:TestPythonProjectQuotaWaitShapeNumericAndPhysicalFailures`.

Python/Go differences:

- Legacy generated collection waits retain their historical five-minute policy. The concrete waits.API lane defaults status to unlimited/ERROR and deletion120s with interval2s, matching the pinned helper defaults.
- Finite SDK-owned concrete Target constructors with immutable routes and independent record snapshots replace Python arbitrary Resource subclasses, same-object mutation and dynamic getattr. All known pinned fetchable concrete targets now include admin ProjectQuota; arbitrary class/subtyping remains SDK-R1 as a deliberate finite Go API boundary.
- Source normalized nullable/empty/string states use PythonLower and private exact comparison tokens, not broad Unicode EqualFold. Go failures accept strings only, callbacks are typed func(int)error, declared attributes must be nullable strings.
- The SDK validates source/headers and applied option callbacks before cached-state success; Source returns its cached object before consulting session or many invalid timer/failure settings. Negative interval is a pre-HTTP Go guard rather than Python sleep-time ValueError after an observation.
- Status timeout defaults to unlimited, deletion timeout120s, interval2s. Nil Timeout is omission; explicit Unlimited removes the deletion SDK limit. Explicit zero/negative timeout fails before first GET unless a cached status already matches. Zero interval uses Source 100ms/min-positive-timeout pause.
- Fixed original route, selected microversion or absence and reserved headers survive native attempts. Python mutable response id can change subsequent fetch routing; Go preserves RequestID and leaves response id/HREF passive. Live auth token and ordinary native retry headers remain native.
- Source custom Secret.fetch does not raise HTTP errors itself; Go admits100..399 and rejects400..599 strictly with full physical causes, so payload404 cannot establish metadata absence. Accepted UTF-8 text is decoded strictly, opaque bytes remain Payload plus JSON base64 Resource values.
- Container/Order generic malformed/empty JSON preserves synchronized attributes as Source ValueError handling does; valid JSON null/nonobject fails. Secret malformed JSON fails. Go requires valid UTF-8 metadata and eagerly projects finite declared casts/aliases using the existing shared projector.
- Source projectors provide nullable descriptor defaults and locationnull; canonical wire aliases win deterministically. Unknown metadata is retained only in Wire. No Connection location/session HTTP cache, dirty-resource commit or arbitrary Python runtime value parity is claimed.
- Clean metadata404 deletion returns an owned original Resource view plus the new404 proof channels rather than Source object identity. Fetched deleted returns the updated view. Errors retain last actual evidence but return Resource=nil.
- Fixed literal/request selector routing never follows response HREF/native convenience ID; Source mutable id routing is not reproduced.
- Cached native Metadata.Body has no original object key order or byte envelope: sorted-key constructor aliases and no fabricated Envelope.
- Computed location ownednull and strict UTF8 finite JSON/declared attributes remain existing SDK-R1/C1 policies.
- Pinned generic Resource.fetch requires_id=True overrides SecretACL.requires_id=False and does not carry the public getter requires_id=False option. Generic waiting on a getter ACL/Quota with id=None fails before polling, or appends an artificial ID if set. Concrete Go targets deliberately correct this pinned request preparation defect by owning stock /secrets/{secretID}/acl and /quotas routes.
- SecretACL URI secret_id is selected from immutable request scope but is not added to Resource Body/to_dict; current-project is an internal wait selector and is never a wire project/member ID or synthesized Resource.id.
- No status descriptor is invented for ACL/Quota: default status waits fail preflight; deletion skips absent status and waits for clean GET404. Stock ACL deletion restores implicit default ACL GET200 and project override deletion leaves effective quota GET200; GET absence observation does not prove settings deletion.
- ACL wait read projection follows Source getter dict coercion while existing native secretacls API uses its independent stock object validation. Strict UTF8, declared string/null wait attributes and owned-null computed location remain bounded SDK policies.
- Admin ProjectQuota is a member route unlike effective Quota; its original project ID is seeded and immutable while response IDs remain passive. Like Quota/ACL it has no status descriptor and stock override reset can leave GET200, so absence waiting is not proof of settings deletion.
- This mapping covers finite SDK-owned concrete target constructors rather than arbitrary Python Resource/runtime subclass/cache/location behavior. SDK-R1/C1/S1 and actual deployment/permission/release validation remain explicit whole-SDK gaps.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import openstack
from openstack.key_manager.v1.secret_acl import SecretACL
from openstack.key_manager.v1.quota import Quota

class PollableSecretACL(SecretACL):
    def fetch(self, session, **kwargs):
        return super().fetch(session, requires_id=False, **kwargs)

class PollableQuota(Quota):
    def fetch(self, session, **kwargs):
        return super().fetch(session, requires_id=False, **kwargs)

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--kind", choices=["acl", "quota"], default="acl")
parser.add_argument("--secret-id")
parser.add_argument("--operation", choices=["status", "delete"], default="status")
parser.add_argument("--attribute", default="acl_ref")
parser.add_argument("--state")
parser.add_argument("--null-state", action="store_true")
parser.add_argument("--interval", type=float, default=2)
parser.add_argument("--timeout", type=float)
parser.add_argument("--correct-singleton-fetch", action="store_true")
args = parser.parse_args()
if args.kind == "acl" and not args.secret_id:
    parser.error("--secret-id is required for ACL")
if args.operation == "status" and args.state is None and not args.null_state:
    parser.error("--state or --null-state is required")
conn = openstack.connect(cloud=args.cloud)
if args.kind == "acl":
    cls = PollableSecretACL if args.correct_singleton_fetch else SecretACL
    res = cls.existing(secret_id=args.secret_id)
else:
    cls = PollableQuota if args.correct_singleton_fetch else Quota
    res = cls.existing()
options = {"interval": args.interval, "callback": lambda progress: print("progress", progress)}
if args.timeout is not None:
    options["wait"] = args.timeout
if args.operation == "delete":
    result = conn.key_manager.wait_for_delete(res, **options)
else:
    state = None if args.null_state else args.state
    result = conn.key_manager.wait_for_status(res, state, attribute=args.attribute, **options)
print(result.to_dict())
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    kind := flag.String("kind", "acl", "acl/quota")
    secretID := flag.String("secret-id", "", "literal secret ID for ACL")
    operation := flag.String("operation", "status", "status/delete")
    attribute := flag.String("attribute", "acl_ref", "declared string/null attribute")
    state := flag.String("state", "", "target string; empty string is explicit")
    nullState := flag.Bool("null-state", false, "target is Source None")
    interval := flag.Duration("interval", 2*time.Second, "polling interval")
    timeout := flag.Duration("timeout", 0, "nonzero override; zero keeps Source default")
    flag.Parse()
    ctx := context.Background()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.KeyManagerV1(ctx)
    if err != nil { log.Fatal(err) }
    var target waits.Target
    switch *kind {
    case "acl": target = waits.SecretACL(*secretID)
    case "quota": target = waits.Quota()
    default: log.Fatal("unknown kind")
    }
    options := []waits.Option{
        waits.WithInterval(*interval),
        waits.WithProgressCallback(func(progress int) error {
            fmt.Println("progress", progress)
            return nil
        }),
    }
    if *timeout != 0 { options = append(options, waits.WithTimeout(*timeout)) }
    var result *waits.Record
    switch *operation {
    case "delete": result, err = service.Waits.WaitForDelete(ctx, target, options...)
    case "status":
        options = append(options, waits.WithAttribute(*attribute))
        selected := state
        if *nullState { selected = nil }
        result, err = service.Waits.WaitForStatus(ctx, target, selected, options...)
    default: log.Fatal("unknown operation")
    }
    if err != nil { log.Fatal(err) }
    encoded, err := json.MarshalIndent(result.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(encoded))
}
```

## References

- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1275-L1336)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2585-L2715)
- [Install](../../docs/install.md)
- [Waits Python](waits-python.md)
- [Barbican ACL controller](https://github.com/openstack/barbican/blob/21.0.0/barbican/api/controllers/acls.py#L153-L207)
- [effective quota controller](https://github.com/openstack/barbican/blob/21.0.0/barbican/api/controllers/quotas.py#L47-L59)
- [Policy](https://docs.openstack.org/barbican/2025.2/configuration/policy.html)
- [Acls](https://github.com/openstack/barbican/blob/21.0.0/barbican/common/policies/acls.py#L20-L88)
- [Quotas](https://github.com/openstack/barbican/blob/21.0.0/barbican/common/policies/quotas.py#L18-L74)
- [Python Project Quota](waits/python-project-quota.md)
- [Python Project Records](quotas/python-project-records.md)
