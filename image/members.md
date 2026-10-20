# Members

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:image/v2/add_member`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.AddImageMember`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.AddImageMemberRecord`
- `github.com/JSYoo5B/go-openstacksdk.Connection.AddImageMemberRecord`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordWriteOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordWriteHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordWriteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordWriteAttribute`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordWriteAttributes`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordMemberID`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordStatus`

- Legacy typed AddImageMember profile only: All methods require an explicit parent image Ref; direct imageID has only fixed member endpoint, explicit parentName uses unchanged native exact-name resolver. No Member parent recovery, memberName/project lookup, response-ID synthesis or response-driven routes.
  Tests: `image/members_contracts_test.go:TestImageMembersFixedParentRoutesAndBodies`, `image/members_core_test.go:TestImageMembersCoreFixedRoutesAndActualEvidence`, `connection_image_members_test.go:TestConnectionImageMembersShareClientAndConcreteDefaults`, `internal/cmd/sdkgen/glance_members_test.go:TestGlanceMemberWorkflowsPreserveNativeScopeAndConcreteDefaults`.
- Legacy typed AddImageMember profile only: Canonical ImageMember requires validUTF8 nonnull JSON object. Optional image_id/member_id/status/schema/created_at/updated_at are strict nullable strings; missing/null nil, empty present, literal dates unparsed. Unknown/case-decoy/large extension numbers are independent raw fields, links passive/nil typed. Mismatched response IDs and arbitrary response status remain passive; atomic schema errors retain original actual200 proof.
  Tests: `image/members_contracts_test.go:TestImageMembersCanonicalRawModelsAndFiniteList`, `image/members_core_test.go:TestImageMembersCoreCanonicalPresenceAndAtomicDecoder`.
- Legacy typed AddImageMember profile only: Source/context/client/provider/prefix/ordinary headers captured before usercallbacks. MemberID is nonempty safeUTF8 single unescaped segment; status input exactly pending/accepted/rejected; invalid inputs/protectedheaders/nilerrorcallbacks fail before parentName/memberHTTP. ParentName failures remain errors despite missing defaults.
  Tests: `image/members_contracts_test.go:TestImageMembersOwnedOptionsAndPreflight`, `image/members_core_test.go:TestImageMembersCoreCompletePreflightAndParentMissing`, `image/members_options_test.go:TestImageMembersOptionsValidationAndDefaults`.
- Legacy typed AddImageMember profile only: Concrete FullOpts replaces configuration, Header/Headers merge; map/pointer/helper input/optionslice/percallback configs independently copied and callbacks applied once. Prepared route/headers stay fixed with live originalprovider token; callback/source/provider and postName changes rejected. Reuse and parallel helpers retain owned settings.
  Tests: `image/members_contracts_test.go:TestImageMembersOwnedOptionsAndPreflight`, `image/members_core_test.go:TestImageMembersCoreCapturedSourceAndRequestOwnership`, `image/members_options_test.go:TestImageMembersOptionsSnapshotsReplacementAndLastWins`, `image/members_options_test.go:TestImageMembersOptionsCallbacksPointersAndSliceOwnership`, `image/members_options_test.go:TestImageMembersOptionsParallelReusableHelpers`.
- Legacy typed AddImageMember profile only: Accepted200 typed failures read/Close/context/UTF8/JSON/schema return nil model plus ownedResponseError retaining actual bytes/header/status and original read/Close/ctx.Err/customcause. Remove actual204 yields opaque actualack including nonnilack+error on handlingfailure. BodyClose once, no acceptedbody replay; rejectedstatuses keepnative ownership.
  Tests: `image/members_contracts_test.go:TestImageMembersAcceptedEvidenceAndMutationNoReplay`, `image/members_core_test.go:TestImageMembersCoreOwnedBodyFailuresAndPhysical404`.
- Legacy typed AddImageMember profile only: Shared REST/fixedrequest keeps configured native prebody retry/reauth/backoff and identical-target redirect callbacks, blocks changedmethod/origin/path/query before transport. Owned serializedbody/RawBody/KeepResponseBody/JSONResponse cannot mutate; nilbody differsJSONnull, same serializedreplacement allowed. Actual status gate remains200/204 despite hook-expandedOkCodes. Native reauth ErrOriginal/ErrReauth fields and existing native memberABI/scopes unchanged.
  Tests: `image/members_contracts_test.go:TestImageMembersNativeHooksAndCompatibility`, `image/members_core_test.go:TestImageMembersCoreCapturedSourceAndRequestOwnership`, `internal/cmd/sdkgen/glance_members_test.go:TestGlanceMemberWorkflowsPreserveNativeScopeAndConcreteDefaults`.
- Legacy typed AddImageMember profile only: Add sends POST200 at fixedcollection with only member JSON; pending state/visibility/owner/policy/quota server-owned. Same bounded safeID preflight applies even JSON input; no metadataGET, genericattrs, statusoverride, discovery or automaticacceptance.
  Tests: `image/members_contracts_test.go:TestImageMembersFixedParentRoutesAndBodies`, `image/members_core_test.go:TestImageMembersCoreFixedRoutesAndActualEvidence`, `image/members_core_test.go:TestImageMembersCoreCompletePreflightAndParentMissing`.
- Owned Member Record family requires an explicit parent Ref on every call. Direct parent ID uses a fixed route; explicit parent Name alone invokes the unchanged native exact-name resolver. Child-selecting calls extract only identity from literal ID or a cloned Resource Body: present id wins over member_id even when null, empty or invalid, and member_id is used only when id is absent. Literal validUTF8 identities are escaped once as a single path segment; Record.ImageID, Wire, receipt and other attributes never recover the parent or choose the child route. Identity is captured before location callbacks.
  Tests: `image/member_record_mutations_test.go:TestImageMemberRecordMutationUsesPresentIdentityBeforeAlias`, `image/member_record_mutations_test.go:TestImageMemberRecordMutationsCaptureChildIDBeforeCallbacks`, `image/member_record_mutations_test.go:TestImageMemberRecordFamilyReusesExplicitParentNameLookup`.
- Owned family snapshots concrete options and checks context, captured source and service binding around callbacks. Observed option/guard failures stop later callbacks that could restore state; nil options, invalid identities, invalid parent and protected or malformed headers fail before HTTP. Explicit Name lookup applies options once and parent lookup failures remain errors. Shared guarded semantic selection preserves option and context causes without changing unguarded callers.
  Tests: `image/member_record_mutations_test.go:TestImageMemberRecordMutationsFullPreflight`, `image/member_record_mutations_test.go:TestImageMemberRecordFamilyStopsAfterOptionGuardFailure`, `image/member_record_mutations_test.go:TestImageMemberRecordFamilyReusesExplicitParentNameLookup`, `resource/filter_selection_guard_test.go:TestFilterSelectionGuardStopsBeforeRestoringOption`, `resource/filter_selection_guard_test.go:TestFilterSelectionGuardPreservesCausesAndSkipsCanceledOptions`, `resource/filter_selection_guard_test.go:TestFilterSelectionGuardRetainsSelectionBoundaries`.
- Owned family retains actual accepted Read/Close/context/source/outer-guard failure evidence and original causes with an independent submitted Record or opaque partial acknowledgement, closes the physical body once and does not automatically replay an accepted response. No accepted response yields nil Record/ack and native rejected-status proof without a fabricated receipt. Connection delegates through the shared Image binding, captured current location and live authentication; the native member ABI remains separate.
  Tests: `image/member_record_mutations_test.go:TestImageMemberRecordMutationsAcceptedPhysicalFailuresKeepPartialEvidence`, `image/member_record_mutations_test.go:TestImageMemberRecordFamilyRejectsNativeStatusesWithoutFakeReceipt`, `member_record_mutations_integration_test.go:TestConnectionOwnedImageMemberFamilySharesSourceLocationAndActualReceipts`.
- Owned Add/Get/Update use the actual200..399 acceptance policy and the shared Member Record translator. A valid object overlays the fresh submitted raw seed, handles member/member_id aliases in response order, retains omitted declared values and projects nine fields including fixed image URI/current location while preserving passive actual Wire. Empty or invalid JSON syntax tolerates the fresh seed; valid null/scalar/array or invalidUTF8 response shapes return a submitted Record plus actual ResponseError evidence. No accepted processing failure or tolerated syntax automatically replays HTTP.
  Tests: `image/member_record_mutations_test.go:TestGetImageMemberRecordFreshConstructorAliasesAndAcceptedBodies`, `image/member_record_mutations_test.go:TestGetImageMemberRecordMalformedAcceptedBodyRetainsFreshSeedEvidence`, `image/member_record_mutations_test.go:TestImageMemberRecordWritesTranslateAcceptedStatusesAndPreserveDefaults`, `image/member_record_mutations_test.go:TestImageMemberRecordWritesMalformedAcceptedBodiesKeepSubmittedReceipt`.
- Owned record inputs, projected Resource Body/Header, passive Wire, Envelope, record Header and physical response storage remain independent. Supplied records contribute identity only to a new convenience request, so old status/name/date fields and prior response/location evidence do not become new submitted attributes. Returned Record state and receipts can be inspected without mutating the original caller input or response channels.
  Tests: `image/member_record_mutations_test.go:TestImageMemberRecordMutationResultsDoNotAliasRequestOrPhysicalResponse`, `image/member_record_mutations_test.go:TestUpdateImageMemberRecordCreatesFreshBoundMemberAndRawBody`, `image/member_record_mutations_test.go:TestGetImageMemberRecordFreshConstructorAliasesAndAcceptedBodies`.
- Owned writes select supplied raw JSON for the seven declared Body descriptors only and send a flat body, with member_id transposed to member. Unknown attributes become neither properties nor query; even an unknown attribute's factory-captured encoding error is ignored at selection. Canonical member_id wins a competing member alias on Add. FullOpts replaces the entire configuration, plural Attributes replaces the semantic attribute set without clearing headers, singular attributes append and Header/Headers merge. Factory/map/slice/callback settings are independently captured; native prebody retry keeps the fixed serialized body and live authentication while allowing ordinary retry headers. Update's bound member/member_id/image_id collisions and non-semantic query carriers are rejected before HTTP.
  Tests: `image/member_record_mutations_test.go:TestAddImageMemberRecordOwnsFlatDeclaredBodyAndIgnoresUnknown`, `image/member_record_mutations_test.go:TestImageMemberRecordWriteOptionsAndLocationAreOwned`, `image/member_record_mutations_test.go:TestImageMemberRecordMutationsFullPreflight`, `image/member_record_mutations_test.go:TestImageMemberRecordWritesNativeRetryRetainsBodyAndLiveAuth`.
- Owned Add constructs a fresh unsynchronized Member and always POSTs the collection. No attributes sends {}, without inventing a required member. Supplied id/name/member/status/date/schema values retain arbitrary JSON-domain shapes and explicit null; a supplied id remains in the POST body. Required member, enum/schema rules, sharing authorization and resulting visibility remain server-owned rather than narrowed to the legacy typed memberID-only profile.
  Tests: `image/member_record_mutations_test.go:TestAddImageMemberRecordOwnsFlatDeclaredBodyAndIgnoresUnknown`.

Python/Go differences:

- Legacy typed AddImageMember profile only: Python arbitraryattrs and mutable created Member differ from bounded memberID/exactmember JSON and passive actual response proof.
- Legacy typed AddImageMember profile only: No full Resource/session/cache or serverpolicy/quota parity.
- Owned Member Record is a finite immutable JSON-domain convenience profile. Supplied Record identity is copied into a fresh Member request rather than reusing a Python Resource/Munch/subclass or mutating the same instance. Arbitrary object/descriptor/default/history/Header/URI lifecycle remains SDK-R1, Adapter cache/conditional GET remains SDK-C1, and complete session/base_path/microversion/class/exception/conflict controls remain SDK-S1 in docs/implementation-plan.md; these global requirements are not declared completed by this mapping.
- Go uses encoding/json snapshots, complete validUTF8 JSON for selected known values, fixed literal UTF8 identity validation/escaping and canonical member_id precedence on Add. Python arbitrary objects, kwargs insertion order, alternate parser/surrogate/number spelling and coercion are not runtime-equivalent. Unknown Source attributes are discarded by the fixed Member profile; the Go factory may encode an unknown value before its captured error is ignored at selection.
- Go fixed-profile Source controls are rejected before HTTP instead of implementing dynamic constructor/routing/session behavior. In particular the pinned Add helper's __conflicting_attrs can replace bound image_id, but Go reserves that control. Source microversion negotiation and arbitrary class/resource/connection/base_path overrides remain SDK-S1. Explicit parent Name resolution is an additive Go native resolver convenience; no supplied Member parent recovery is added.
- Owned Add/Get/Update accept actual200..399 and return independent submitted Record plus actual receipt on accepted processing failure; Python may mutate and partially overlay the same object. Tolerated invalid syntax is not proof of server schema, sharing policy or rollback. No actual Python SDK, authenticated deployment, server policy or cloud mutation was executed by this local contract audit.

### `python:image/v2/get_member`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.GetImageMember`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.GetImageMemberRecord`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetImageMemberRecord`

- Legacy typed GetImageMember profile only: All methods require an explicit parent image Ref; direct imageID has only fixed member endpoint, explicit parentName uses unchanged native exact-name resolver. No Member parent recovery, memberName/project lookup, response-ID synthesis or response-driven routes.
  Tests: `image/members_contracts_test.go:TestImageMembersFixedParentRoutesAndBodies`, `image/members_core_test.go:TestImageMembersCoreFixedRoutesAndActualEvidence`, `connection_image_members_test.go:TestConnectionImageMembersShareClientAndConcreteDefaults`, `internal/cmd/sdkgen/glance_members_test.go:TestGlanceMemberWorkflowsPreserveNativeScopeAndConcreteDefaults`.
- Legacy typed GetImageMember profile only: Canonical ImageMember requires validUTF8 nonnull JSON object. Optional image_id/member_id/status/schema/created_at/updated_at are strict nullable strings; missing/null nil, empty present, literal dates unparsed. Unknown/case-decoy/large extension numbers are independent raw fields, links passive/nil typed. Mismatched response IDs and arbitrary response status remain passive; atomic schema errors retain original actual200 proof.
  Tests: `image/members_contracts_test.go:TestImageMembersCanonicalRawModelsAndFiniteList`, `image/members_core_test.go:TestImageMembersCoreCanonicalPresenceAndAtomicDecoder`.
- Legacy typed GetImageMember profile only: Source/context/client/provider/prefix/ordinary headers captured before usercallbacks. MemberID is nonempty safeUTF8 single unescaped segment; status input exactly pending/accepted/rejected; invalid inputs/protectedheaders/nilerrorcallbacks fail before parentName/memberHTTP. ParentName failures remain errors despite missing defaults.
  Tests: `image/members_contracts_test.go:TestImageMembersOwnedOptionsAndPreflight`, `image/members_core_test.go:TestImageMembersCoreCompletePreflightAndParentMissing`, `image/members_options_test.go:TestImageMembersOptionsValidationAndDefaults`.
- Legacy typed GetImageMember profile only: Concrete FullOpts replaces configuration, Header/Headers merge; map/pointer/helper input/optionslice/percallback configs independently copied and callbacks applied once. Prepared route/headers stay fixed with live originalprovider token; callback/source/provider and postName changes rejected. Reuse and parallel helpers retain owned settings.
  Tests: `image/members_contracts_test.go:TestImageMembersOwnedOptionsAndPreflight`, `image/members_core_test.go:TestImageMembersCoreCapturedSourceAndRequestOwnership`, `image/members_options_test.go:TestImageMembersOptionsSnapshotsReplacementAndLastWins`, `image/members_options_test.go:TestImageMembersOptionsCallbacksPointersAndSliceOwnership`, `image/members_options_test.go:TestImageMembersOptionsParallelReusableHelpers`.
- Legacy typed GetImageMember profile only: Accepted200 typed failures read/Close/context/UTF8/JSON/schema return nil model plus ownedResponseError retaining actual bytes/header/status and original read/Close/ctx.Err/customcause. Remove actual204 yields opaque actualack including nonnilack+error on handlingfailure. BodyClose once, no acceptedbody replay; rejectedstatuses keepnative ownership.
  Tests: `image/members_contracts_test.go:TestImageMembersAcceptedEvidenceAndMutationNoReplay`, `image/members_core_test.go:TestImageMembersCoreOwnedBodyFailuresAndPhysical404`.
- Legacy typed GetImageMember profile only: Shared REST/fixedrequest keeps configured native prebody retry/reauth/backoff and identical-target redirect callbacks, blocks changedmethod/origin/path/query before transport. Owned serializedbody/RawBody/KeepResponseBody/JSONResponse cannot mutate; nilbody differsJSONnull, same serializedreplacement allowed. Actual status gate remains200/204 despite hook-expandedOkCodes. Native reauth ErrOriginal/ErrReauth fields and existing native memberABI/scopes unchanged.
  Tests: `image/members_contracts_test.go:TestImageMembersNativeHooksAndCompatibility`, `image/members_core_test.go:TestImageMembersCoreCapturedSourceAndRequestOwnership`, `internal/cmd/sdkgen/glance_members_test.go:TestGlanceMemberWorkflowsPreserveNativeScopeAndConcreteDefaults`.
- Legacy typed GetImageMember profile only: Get direct fixedmember GET200 only; no IgnoreMissing or cache shortcut. Actual canonical response may have missing/empty/mismatched IDs and is never overwritten with requestIDs. No responseName/listfallback/schema following.
  Tests: `image/members_contracts_test.go:TestImageMembersFixedParentRoutesAndBodies`, `image/members_core_test.go:TestImageMembersCoreFixedRoutesAndActualEvidence`, `image/members_contracts_test.go:TestImageMembersCanonicalRawModelsAndFiniteList`.
- Owned Member Record family requires an explicit parent Ref on every call. Direct parent ID uses a fixed route; explicit parent Name alone invokes the unchanged native exact-name resolver. Child-selecting calls extract only identity from literal ID or a cloned Resource Body: present id wins over member_id even when null, empty or invalid, and member_id is used only when id is absent. Literal validUTF8 identities are escaped once as a single path segment; Record.ImageID, Wire, receipt and other attributes never recover the parent or choose the child route. Identity is captured before location callbacks.
  Tests: `image/member_record_mutations_test.go:TestImageMemberRecordMutationUsesPresentIdentityBeforeAlias`, `image/member_record_mutations_test.go:TestImageMemberRecordMutationsCaptureChildIDBeforeCallbacks`, `image/member_record_mutations_test.go:TestImageMemberRecordFamilyReusesExplicitParentNameLookup`.
- Owned family snapshots concrete options and checks context, captured source and service binding around callbacks. Observed option/guard failures stop later callbacks that could restore state; nil options, invalid identities, invalid parent and protected or malformed headers fail before HTTP. Explicit Name lookup applies options once and parent lookup failures remain errors. Shared guarded semantic selection preserves option and context causes without changing unguarded callers.
  Tests: `image/member_record_mutations_test.go:TestImageMemberRecordMutationsFullPreflight`, `image/member_record_mutations_test.go:TestImageMemberRecordFamilyStopsAfterOptionGuardFailure`, `image/member_record_mutations_test.go:TestImageMemberRecordFamilyReusesExplicitParentNameLookup`, `resource/filter_selection_guard_test.go:TestFilterSelectionGuardStopsBeforeRestoringOption`, `resource/filter_selection_guard_test.go:TestFilterSelectionGuardPreservesCausesAndSkipsCanceledOptions`, `resource/filter_selection_guard_test.go:TestFilterSelectionGuardRetainsSelectionBoundaries`.
- Owned family retains actual accepted Read/Close/context/source/outer-guard failure evidence and original causes with an independent submitted Record or opaque partial acknowledgement, closes the physical body once and does not automatically replay an accepted response. No accepted response yields nil Record/ack and native rejected-status proof without a fabricated receipt. Connection delegates through the shared Image binding, captured current location and live authentication; the native member ABI remains separate.
  Tests: `image/member_record_mutations_test.go:TestImageMemberRecordMutationsAcceptedPhysicalFailuresKeepPartialEvidence`, `image/member_record_mutations_test.go:TestImageMemberRecordFamilyRejectsNativeStatusesWithoutFakeReceipt`, `member_record_mutations_integration_test.go:TestConnectionOwnedImageMemberFamilySharesSourceLocationAndActualReceipts`.
- Owned Add/Get/Update use the actual200..399 acceptance policy and the shared Member Record translator. A valid object overlays the fresh submitted raw seed, handles member/member_id aliases in response order, retains omitted declared values and projects nine fields including fixed image URI/current location while preserving passive actual Wire. Empty or invalid JSON syntax tolerates the fresh seed; valid null/scalar/array or invalidUTF8 response shapes return a submitted Record plus actual ResponseError evidence. No accepted processing failure or tolerated syntax automatically replays HTTP.
  Tests: `image/member_record_mutations_test.go:TestGetImageMemberRecordFreshConstructorAliasesAndAcceptedBodies`, `image/member_record_mutations_test.go:TestGetImageMemberRecordMalformedAcceptedBodyRetainsFreshSeedEvidence`, `image/member_record_mutations_test.go:TestImageMemberRecordWritesTranslateAcceptedStatusesAndPreserveDefaults`, `image/member_record_mutations_test.go:TestImageMemberRecordWritesMalformedAcceptedBodiesKeepSubmittedReceipt`.
- Owned record inputs, projected Resource Body/Header, passive Wire, Envelope, record Header and physical response storage remain independent. Supplied records contribute identity only to a new convenience request, so old status/name/date fields and prior response/location evidence do not become new submitted attributes. Returned Record state and receipts can be inspected without mutating the original caller input or response channels.
  Tests: `image/member_record_mutations_test.go:TestImageMemberRecordMutationResultsDoNotAliasRequestOrPhysicalResponse`, `image/member_record_mutations_test.go:TestUpdateImageMemberRecordCreatesFreshBoundMemberAndRawBody`, `image/member_record_mutations_test.go:TestGetImageMemberRecordFreshConstructorAliasesAndAcceptedBodies`.
- Owned Get follows the pinned convenience proxy's fresh Member constructor: it seeds member_id rather than id, performs one fixed member GET and does not fetch or reuse an existing Member's attributes. Actual response member can replace fallback projected identity, explicit response id including null wins, and omitted or tolerated invalid-syntax responses retain the requested alternate identity. No Find, member-name lookup, cache shortcut or IgnoreMissing is introduced.
  Tests: `image/member_record_mutations_test.go:TestGetImageMemberRecordFreshConstructorAliasesAndAcceptedBodies`.

Python/Go differences:

- Legacy typed GetImageMember profile only: Python convenience proxy extractsIDs and builds fresh Resource; upper passive raw nullable/date DTO differs from Resource descriptors/cache/session and native time.Time model.
- Legacy typed GetImageMember profile only: Strict200/fullUTF8/canonical failure evidence differs from Python broad non-error status/ValueError handling.
- Legacy typed GetImageMember profile only: The pinned get_member extracts member/image Resource IDs and creates a fresh Member seeded with the requested member alternate ID and image URI. Go takes explicit parent Ref plus literal memberID, does not reuse a supplied mutable Member or synthesize requested IDs into the response; canonical MemberID/ImageID are actual nullable response values. Explicit image Name resolution is an additive Go convenience.
- Legacy typed GetImageMember profile only: Python Member.member_id is a Body descriptor for member (alternate ID) and recognizes both descriptor and wire names while retaining request identity when the response omits it. Go projects the canonical member_id key only; a response member alias, id, location or other extension remains accessible as exact RawMessage in Metadata.Body instead of being promoted or defaulted. Dates remain literal nullable strings and unknown response data is retained rather than discarded.
- Legacy typed GetImageMember profile only: Get is strict about a nonempty safe UTF-8 member segment, context/source and ordinary-header options, and accepts actual HTTP200 only. Wrong canonical types or malformed successful bodies return nil typed DTO plus actual ResponseError evidence instead of Python dynamic descriptor values or a seeded Resource after ValueError. HTTP404 is an error; no Find/list fallback or IgnoreMissing policy is introduced.
- Legacy typed GetImageMember profile only: No authenticated deployment was executed by this audit. Existing local contract tests and root-run regression checks establish the named Go mapping; full mutable Resource/session behavior and independent Find/List/mutation APIs remain outside this getter.
- Owned Member Record is a finite immutable JSON-domain convenience profile. Supplied Record identity is copied into a fresh Member request rather than reusing a Python Resource/Munch/subclass or mutating the same instance. Arbitrary object/descriptor/default/history/Header/URI lifecycle remains SDK-R1, Adapter cache/conditional GET remains SDK-C1, and complete session/base_path/microversion/class/exception/conflict controls remain SDK-S1 in docs/implementation-plan.md; these global requirements are not declared completed by this mapping.
- Go uses encoding/json snapshots, complete validUTF8 JSON for selected known values, fixed literal UTF8 identity validation/escaping and canonical member_id precedence on Add. Python arbitrary objects, kwargs insertion order, alternate parser/surrogate/number spelling and coercion are not runtime-equivalent. Unknown Source attributes are discarded by the fixed Member profile; the Go factory may encode an unknown value before its captured error is ignored at selection.
- Go fixed-profile Source controls are rejected before HTTP instead of implementing dynamic constructor/routing/session behavior. In particular the pinned Add helper's __conflicting_attrs can replace bound image_id, but Go reserves that control. Source microversion negotiation and arbitrary class/resource/connection/base_path overrides remain SDK-S1. Explicit parent Name resolution is an additive Go native resolver convenience; no supplied Member parent recovery is added.
- Owned Add/Get/Update accept actual200..399 and return independent submitted Record plus actual receipt on accepted processing failure; Python may mutate and partially overlay the same object. Tolerated invalid syntax is not proof of server schema, sharing policy or rollback. No actual Python SDK, authenticated deployment, server policy or cloud mutation was executed by this local contract audit.

### `python:image/v2/update_member`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.UpdateImageMember`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.UpdateImageMemberRecord`
- `github.com/JSYoo5B/go-openstacksdk.Connection.UpdateImageMemberRecord`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordWriteOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordWriteHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordWriteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordWriteAttribute`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordWriteAttributes`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordMemberID`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordStatus`

- Legacy typed UpdateImageMember profile only: All methods require an explicit parent image Ref; direct imageID has only fixed member endpoint, explicit parentName uses unchanged native exact-name resolver. No Member parent recovery, memberName/project lookup, response-ID synthesis or response-driven routes.
  Tests: `image/members_contracts_test.go:TestImageMembersFixedParentRoutesAndBodies`, `image/members_core_test.go:TestImageMembersCoreFixedRoutesAndActualEvidence`, `connection_image_members_test.go:TestConnectionImageMembersShareClientAndConcreteDefaults`, `internal/cmd/sdkgen/glance_members_test.go:TestGlanceMemberWorkflowsPreserveNativeScopeAndConcreteDefaults`.
- Legacy typed UpdateImageMember profile only: Canonical ImageMember requires validUTF8 nonnull JSON object. Optional image_id/member_id/status/schema/created_at/updated_at are strict nullable strings; missing/null nil, empty present, literal dates unparsed. Unknown/case-decoy/large extension numbers are independent raw fields, links passive/nil typed. Mismatched response IDs and arbitrary response status remain passive; atomic schema errors retain original actual200 proof.
  Tests: `image/members_contracts_test.go:TestImageMembersCanonicalRawModelsAndFiniteList`, `image/members_core_test.go:TestImageMembersCoreCanonicalPresenceAndAtomicDecoder`.
- Legacy typed UpdateImageMember profile only: Source/context/client/provider/prefix/ordinary headers captured before usercallbacks. MemberID is nonempty safeUTF8 single unescaped segment; status input exactly pending/accepted/rejected; invalid inputs/protectedheaders/nilerrorcallbacks fail before parentName/memberHTTP. ParentName failures remain errors despite missing defaults.
  Tests: `image/members_contracts_test.go:TestImageMembersOwnedOptionsAndPreflight`, `image/members_core_test.go:TestImageMembersCoreCompletePreflightAndParentMissing`, `image/members_options_test.go:TestImageMembersOptionsValidationAndDefaults`.
- Legacy typed UpdateImageMember profile only: Concrete FullOpts replaces configuration, Header/Headers merge; map/pointer/helper input/optionslice/percallback configs independently copied and callbacks applied once. Prepared route/headers stay fixed with live originalprovider token; callback/source/provider and postName changes rejected. Reuse and parallel helpers retain owned settings.
  Tests: `image/members_contracts_test.go:TestImageMembersOwnedOptionsAndPreflight`, `image/members_core_test.go:TestImageMembersCoreCapturedSourceAndRequestOwnership`, `image/members_options_test.go:TestImageMembersOptionsSnapshotsReplacementAndLastWins`, `image/members_options_test.go:TestImageMembersOptionsCallbacksPointersAndSliceOwnership`, `image/members_options_test.go:TestImageMembersOptionsParallelReusableHelpers`.
- Legacy typed UpdateImageMember profile only: Accepted200 typed failures read/Close/context/UTF8/JSON/schema return nil model plus ownedResponseError retaining actual bytes/header/status and original read/Close/ctx.Err/customcause. Remove actual204 yields opaque actualack including nonnilack+error on handlingfailure. BodyClose once, no acceptedbody replay; rejectedstatuses keepnative ownership.
  Tests: `image/members_contracts_test.go:TestImageMembersAcceptedEvidenceAndMutationNoReplay`, `image/members_core_test.go:TestImageMembersCoreOwnedBodyFailuresAndPhysical404`.
- Legacy typed UpdateImageMember profile only: Shared REST/fixedrequest keeps configured native prebody retry/reauth/backoff and identical-target redirect callbacks, blocks changedmethod/origin/path/query before transport. Owned serializedbody/RawBody/KeepResponseBody/JSONResponse cannot mutate; nilbody differsJSONnull, same serializedreplacement allowed. Actual status gate remains200/204 despite hook-expandedOkCodes. Native reauth ErrOriginal/ErrReauth fields and existing native memberABI/scopes unchanged.
  Tests: `image/members_contracts_test.go:TestImageMembersNativeHooksAndCompatibility`, `image/members_core_test.go:TestImageMembersCoreCapturedSourceAndRequestOwnership`, `internal/cmd/sdkgen/glance_members_test.go:TestGlanceMemberWorkflowsPreserveNativeScopeAndConcreteDefaults`.
- Legacy typed UpdateImageMember profile only: Update fixedmember PUT200 with only exactstatus pending/accepted/rejected. Empty/uppercase/whitespace/unknown input rejected before callbacks/lookup/HTTP; server controls authorization and visibility effects, no read-modify-write/cache mutation/wait.
  Tests: `image/members_contracts_test.go:TestImageMembersFixedParentRoutesAndBodies`, `image/members_core_test.go:TestImageMembersCoreFixedRoutesAndActualEvidence`, `image/members_core_test.go:TestImageMembersCoreCompletePreflightAndParentMissing`.
- Owned Member Record family requires an explicit parent Ref on every call. Direct parent ID uses a fixed route; explicit parent Name alone invokes the unchanged native exact-name resolver. Child-selecting calls extract only identity from literal ID or a cloned Resource Body: present id wins over member_id even when null, empty or invalid, and member_id is used only when id is absent. Literal validUTF8 identities are escaped once as a single path segment; Record.ImageID, Wire, receipt and other attributes never recover the parent or choose the child route. Identity is captured before location callbacks.
  Tests: `image/member_record_mutations_test.go:TestImageMemberRecordMutationUsesPresentIdentityBeforeAlias`, `image/member_record_mutations_test.go:TestImageMemberRecordMutationsCaptureChildIDBeforeCallbacks`, `image/member_record_mutations_test.go:TestImageMemberRecordFamilyReusesExplicitParentNameLookup`.
- Owned family snapshots concrete options and checks context, captured source and service binding around callbacks. Observed option/guard failures stop later callbacks that could restore state; nil options, invalid identities, invalid parent and protected or malformed headers fail before HTTP. Explicit Name lookup applies options once and parent lookup failures remain errors. Shared guarded semantic selection preserves option and context causes without changing unguarded callers.
  Tests: `image/member_record_mutations_test.go:TestImageMemberRecordMutationsFullPreflight`, `image/member_record_mutations_test.go:TestImageMemberRecordFamilyStopsAfterOptionGuardFailure`, `image/member_record_mutations_test.go:TestImageMemberRecordFamilyReusesExplicitParentNameLookup`, `resource/filter_selection_guard_test.go:TestFilterSelectionGuardStopsBeforeRestoringOption`, `resource/filter_selection_guard_test.go:TestFilterSelectionGuardPreservesCausesAndSkipsCanceledOptions`, `resource/filter_selection_guard_test.go:TestFilterSelectionGuardRetainsSelectionBoundaries`.
- Owned family retains actual accepted Read/Close/context/source/outer-guard failure evidence and original causes with an independent submitted Record or opaque partial acknowledgement, closes the physical body once and does not automatically replay an accepted response. No accepted response yields nil Record/ack and native rejected-status proof without a fabricated receipt. Connection delegates through the shared Image binding, captured current location and live authentication; the native member ABI remains separate.
  Tests: `image/member_record_mutations_test.go:TestImageMemberRecordMutationsAcceptedPhysicalFailuresKeepPartialEvidence`, `image/member_record_mutations_test.go:TestImageMemberRecordFamilyRejectsNativeStatusesWithoutFakeReceipt`, `member_record_mutations_integration_test.go:TestConnectionOwnedImageMemberFamilySharesSourceLocationAndActualReceipts`.
- Owned Add/Get/Update use the actual200..399 acceptance policy and the shared Member Record translator. A valid object overlays the fresh submitted raw seed, handles member/member_id aliases in response order, retains omitted declared values and projects nine fields including fixed image URI/current location while preserving passive actual Wire. Empty or invalid JSON syntax tolerates the fresh seed; valid null/scalar/array or invalidUTF8 response shapes return a submitted Record plus actual ResponseError evidence. No accepted processing failure or tolerated syntax automatically replays HTTP.
  Tests: `image/member_record_mutations_test.go:TestGetImageMemberRecordFreshConstructorAliasesAndAcceptedBodies`, `image/member_record_mutations_test.go:TestGetImageMemberRecordMalformedAcceptedBodyRetainsFreshSeedEvidence`, `image/member_record_mutations_test.go:TestImageMemberRecordWritesTranslateAcceptedStatusesAndPreserveDefaults`, `image/member_record_mutations_test.go:TestImageMemberRecordWritesMalformedAcceptedBodiesKeepSubmittedReceipt`.
- Owned record inputs, projected Resource Body/Header, passive Wire, Envelope, record Header and physical response storage remain independent. Supplied records contribute identity only to a new convenience request, so old status/name/date fields and prior response/location evidence do not become new submitted attributes. Returned Record state and receipts can be inspected without mutating the original caller input or response channels.
  Tests: `image/member_record_mutations_test.go:TestImageMemberRecordMutationResultsDoNotAliasRequestOrPhysicalResponse`, `image/member_record_mutations_test.go:TestUpdateImageMemberRecordCreatesFreshBoundMemberAndRawBody`, `image/member_record_mutations_test.go:TestGetImageMemberRecordFreshConstructorAliasesAndAcceptedBodies`.
- Owned writes select supplied raw JSON for the seven declared Body descriptors only and send a flat body, with member_id transposed to member. Unknown attributes become neither properties nor query; even an unknown attribute's factory-captured encoding error is ignored at selection. Canonical member_id wins a competing member alias on Add. FullOpts replaces the entire configuration, plural Attributes replaces the semantic attribute set without clearing headers, singular attributes append and Header/Headers merge. Factory/map/slice/callback settings are independently captured; native prebody retry keeps the fixed serialized body and live authentication while allowing ordinary retry headers. Update's bound member/member_id/image_id collisions and non-semantic query carriers are rejected before HTTP.
  Tests: `image/member_record_mutations_test.go:TestAddImageMemberRecordOwnsFlatDeclaredBodyAndIgnoresUnknown`, `image/member_record_mutations_test.go:TestImageMemberRecordWriteOptionsAndLocationAreOwned`, `image/member_record_mutations_test.go:TestImageMemberRecordMutationsFullPreflight`, `image/member_record_mutations_test.go:TestImageMemberRecordWritesNativeRetryRetainsBodyAndLiveAuth`.
- Owned Update extracts identity then constructs a fresh Member with bound member_id and explicit parent image URI. It does not commit the supplied Record or reuse its original attributes/baseline. The bound alternate member value makes the fresh constructor dirty, so no attributes, unknown-only attributes or an unchanged status still sends PUT with member rather than becoming a local no-op.
  Tests: `image/member_record_mutations_test.go:TestUpdateImageMemberRecordCreatesFreshBoundMemberAndRawBody`.
- Owned Update keeps the original selected identity as the PUT member body value. An explicit valid string id attribute may retarget the child route, but id is excluded from the PUT body; parent scope remains fixed. Direct member/member_id/image_id attribute collisions are rejected before HTTP. Other declared values retain raw JSON, including null/list/bool status and future status strings, without the legacy exact-status enum restriction.
  Tests: `image/member_record_mutations_test.go:TestUpdateImageMemberRecordCreatesFreshBoundMemberAndRawBody`, `image/member_record_mutations_test.go:TestImageMemberRecordMutationsFullPreflight`.

Python/Go differences:

- Legacy typed UpdateImageMember profile only: Python **attrs/genericcommit accepts broaderbody and mutableResource semantics; Go exactstatus enum and requestbody preflight are deliberate constraints.
- Legacy typed UpdateImageMember profile only: No cached mutation, acceptancewait, policy discovery or fullResource/session parity.
- Owned Member Record is a finite immutable JSON-domain convenience profile. Supplied Record identity is copied into a fresh Member request rather than reusing a Python Resource/Munch/subclass or mutating the same instance. Arbitrary object/descriptor/default/history/Header/URI lifecycle remains SDK-R1, Adapter cache/conditional GET remains SDK-C1, and complete session/base_path/microversion/class/exception/conflict controls remain SDK-S1 in docs/implementation-plan.md; these global requirements are not declared completed by this mapping.
- Go uses encoding/json snapshots, complete validUTF8 JSON for selected known values, fixed literal UTF8 identity validation/escaping and canonical member_id precedence on Add. Python arbitrary objects, kwargs insertion order, alternate parser/surrogate/number spelling and coercion are not runtime-equivalent. Unknown Source attributes are discarded by the fixed Member profile; the Go factory may encode an unknown value before its captured error is ignored at selection.
- Go fixed-profile Source controls are rejected before HTTP instead of implementing dynamic constructor/routing/session behavior. In particular the pinned Add helper's __conflicting_attrs can replace bound image_id, but Go reserves that control. Source microversion negotiation and arbitrary class/resource/connection/base_path overrides remain SDK-S1. Explicit parent Name resolution is an additive Go native resolver convenience; no supplied Member parent recovery is added.
- Owned Add/Get/Update accept actual200..399 and return independent submitted Record plus actual receipt on accepted processing failure; Python may mutate and partially overlay the same object. Tolerated invalid syntax is not proof of server schema, sharing policy or rollback. No actual Python SDK, authenticated deployment, server policy or cloud mutation was executed by this local contract audit.

### `python:image/v2/remove_member`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.RemoveImageMember`
- `github.com/JSYoo5B/go-openstacksdk/image.WithRemoveImageMemberOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithRemoveImageMemberHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithRemoveImageMemberHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithRemoveImageMemberIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.RemoveImageMemberRecord`
- `github.com/JSYoo5B/go-openstacksdk.Connection.RemoveImageMemberRecord`

- Legacy typed RemoveImageMember profile only: All methods require an explicit parent image Ref; direct imageID has only fixed member endpoint, explicit parentName uses unchanged native exact-name resolver. No Member parent recovery, memberName/project lookup, response-ID synthesis or response-driven routes.
  Tests: `image/members_contracts_test.go:TestImageMembersFixedParentRoutesAndBodies`, `image/members_core_test.go:TestImageMembersCoreFixedRoutesAndActualEvidence`, `connection_image_members_test.go:TestConnectionImageMembersShareClientAndConcreteDefaults`, `internal/cmd/sdkgen/glance_members_test.go:TestGlanceMemberWorkflowsPreserveNativeScopeAndConcreteDefaults`.
- Legacy typed RemoveImageMember profile only: Actual204 acknowledgement contains fixed chosen ImageID/requestMemberID and independent opaque Body/Header/StatusCode; no canonicalMember JSON/date model is decoded for deletion. NonJSON/invalidUTF8/partial accepted bytes remain evidence. Ignored404 never becomes acknowledgement.
  Tests: `image/members_contracts_test.go:TestImageMembersFixedParentRoutesAndBodies`, `image/members_core_test.go:TestImageMembersCoreFixedRoutesAndActualEvidence`, `image/members_contracts_test.go:TestImageMembersAcceptedEvidenceAndMutationNoReplay`, `image/members_core_test.go:TestImageMembersCoreOwnedBodyFailuresAndPhysical404`.
- Legacy typed RemoveImageMember profile only: Source/context/client/provider/prefix/ordinary headers captured before usercallbacks. MemberID is nonempty safeUTF8 single unescaped segment; status input exactly pending/accepted/rejected; invalid inputs/protectedheaders/nilerrorcallbacks fail before parentName/memberHTTP. ParentName failures remain errors despite missing defaults.
  Tests: `image/members_contracts_test.go:TestImageMembersOwnedOptionsAndPreflight`, `image/members_core_test.go:TestImageMembersCoreCompletePreflightAndParentMissing`, `image/members_options_test.go:TestImageMembersOptionsValidationAndDefaults`.
- Legacy typed RemoveImageMember profile only: Concrete FullOpts replaces configuration, Header/Headers merge; map/pointer/helper input/optionslice/percallback configs independently copied and callbacks applied once. Prepared route/headers stay fixed with live originalprovider token; callback/source/provider and postName changes rejected. Reuse and parallel helpers retain owned settings.
  Tests: `image/members_contracts_test.go:TestImageMembersOwnedOptionsAndPreflight`, `image/members_core_test.go:TestImageMembersCoreCapturedSourceAndRequestOwnership`, `image/members_options_test.go:TestImageMembersOptionsSnapshotsReplacementAndLastWins`, `image/members_options_test.go:TestImageMembersOptionsCallbacksPointersAndSliceOwnership`, `image/members_options_test.go:TestImageMembersOptionsParallelReusableHelpers`.
- Legacy typed RemoveImageMember profile only: Actual204 read/Close/context failure retains nonnilack plus independent ownedResponseError with actual bytes/header/status and original read/Close/ctx.Err/customcause. Owned404 handlingfailure has nilack+ResponseError404. BodyClose once, no acceptedbody replay; rejectedstatuses keepnative ownership.
  Tests: `image/members_contracts_test.go:TestImageMembersAcceptedEvidenceAndMutationNoReplay`, `image/members_core_test.go:TestImageMembersCoreOwnedBodyFailuresAndPhysical404`.
- Legacy typed RemoveImageMember profile only: Shared REST/fixedrequest keeps configured native prebody retry/reauth/backoff and identical-target redirect callbacks, blocks changedmethod/origin/path/query before transport. Owned serializedbody/RawBody/KeepResponseBody/JSONResponse cannot mutate; nilbody differsJSONnull, same serializedreplacement allowed. Actual status gate remains200/204 despite hook-expandedOkCodes. Native reauth ErrOriginal/ErrReauth fields and existing native memberABI/scopes unchanged.
  Tests: `image/members_contracts_test.go:TestImageMembersNativeHooksAndCompatibility`, `image/members_core_test.go:TestImageMembersCoreCapturedSourceAndRequestOwnership`, `internal/cmd/sdkgen/glance_members_test.go:TestGlanceMemberWorkflowsPreserveNativeScopeAndConcreteDefaults`.
- Legacy typed RemoveImageMember profile only: Remove originalactual204 opaqueack contains fixed chosen ImageID/requestMemberID and independent actualBody/Header/status. IgnoreMissing pointer nil/defaulttrue accepts only clean physicallyowned memberDELETE404: fullread+Close+context success nil,nil; handlingfailure nil+ResponseError404, parent/logical/nested404 neverignored. Private404 bypasses nativeRetryFunc only for thatresponse; false retains native404 policy. Quiet404 is not proof ofabsence.
  Tests: `image/members_contracts_test.go:TestImageMembersAcceptedEvidenceAndMutationNoReplay`, `image/members_core_test.go:TestImageMembersCoreOwnedBodyFailuresAndPhysical404`, `image/members_contracts_test.go:TestImageMembersNativeHooksAndCompatibility`, `image/members_core_test.go:TestImageMembersCoreCompletePreflightAndParentMissing`.
- Owned Member Record family requires an explicit parent Ref on every call. Direct parent ID uses a fixed route; explicit parent Name alone invokes the unchanged native exact-name resolver. Child-selecting calls extract only identity from literal ID or a cloned Resource Body: present id wins over member_id even when null, empty or invalid, and member_id is used only when id is absent. Literal validUTF8 identities are escaped once as a single path segment; Record.ImageID, Wire, receipt and other attributes never recover the parent or choose the child route. Identity is captured before location callbacks.
  Tests: `image/member_record_mutations_test.go:TestImageMemberRecordMutationUsesPresentIdentityBeforeAlias`, `image/member_record_mutations_test.go:TestImageMemberRecordMutationsCaptureChildIDBeforeCallbacks`, `image/member_record_mutations_test.go:TestImageMemberRecordFamilyReusesExplicitParentNameLookup`.
- Owned family snapshots concrete options and checks context, captured source and service binding around callbacks. Observed option/guard failures stop later callbacks that could restore state; nil options, invalid identities, invalid parent and protected or malformed headers fail before HTTP. Explicit Name lookup applies options once and parent lookup failures remain errors. Shared guarded semantic selection preserves option and context causes without changing unguarded callers.
  Tests: `image/member_record_mutations_test.go:TestImageMemberRecordMutationsFullPreflight`, `image/member_record_mutations_test.go:TestImageMemberRecordFamilyStopsAfterOptionGuardFailure`, `image/member_record_mutations_test.go:TestImageMemberRecordFamilyReusesExplicitParentNameLookup`, `resource/filter_selection_guard_test.go:TestFilterSelectionGuardStopsBeforeRestoringOption`, `resource/filter_selection_guard_test.go:TestFilterSelectionGuardPreservesCausesAndSkipsCanceledOptions`, `resource/filter_selection_guard_test.go:TestFilterSelectionGuardRetainsSelectionBoundaries`.
- Owned family retains actual accepted Read/Close/context/source/outer-guard failure evidence and original causes with an independent submitted Record or opaque partial acknowledgement, closes the physical body once and does not automatically replay an accepted response. No accepted response yields nil Record/ack and native rejected-status proof without a fabricated receipt. Connection delegates through the shared Image binding, captured current location and live authentication; the native member ABI remains separate.
  Tests: `image/member_record_mutations_test.go:TestImageMemberRecordMutationsAcceptedPhysicalFailuresKeepPartialEvidence`, `image/member_record_mutations_test.go:TestImageMemberRecordFamilyRejectsNativeStatusesWithoutFakeReceipt`, `member_record_mutations_integration_test.go:TestConnectionOwnedImageMemberFamilySharesSourceLocationAndActualReceipts`.
- Owned Remove accepts actual200..399 and returns an opaque acknowledgement containing fixed chosen parent/member IDs and independent actual Body/Header/StatusCode. It does not parse Member JSON, dates or UTF8, so nonJSON and invalidUTF8 bytes remain evidence. Accepted processing failures retain partial acknowledgement evidence and causes without automatic replay; ignored404 never becomes a fake acknowledgement.
  Tests: `image/member_record_mutations_test.go:TestRemoveImageMemberRecordRetainsEveryAcceptedOpaqueAcknowledgement`, `image/member_record_mutations_test.go:TestImageMemberRecordMutationsAcceptedPhysicalFailuresKeepPartialEvidence`.
- Owned Remove's nil/defaulttrue IgnoreMissing suppresses only a clean final direct native member DELETE404 after read/Close/context/source handling succeeds. False retains native404. Read/Close/cancellation/source/retry-hook failures, nested transport404 and parentName lookup failures remain errors. Configured native retry hooks remain active, including clean final404 after a prebody retry; quiet404 returns nil,nil rather than proving server absence.
  Tests: `image/member_record_mutations_test.go:TestRemoveImageMemberRecordMissingPolicyHonorsOnlyCleanFinal404`, `image/member_record_mutations_test.go:TestImageMemberRecordFamilyReusesExplicitParentNameLookup`.

Python/Go differences:

- Legacy typed RemoveImageMember profile only: Pinned Python optionalimage signature does not recover supplied Member parent; Go requires explicitparent on everycall.
- Legacy typed RemoveImageMember profile only: Python proxy returnsNone on successfuldelete/ignoredNotFound; Go returns actual204ack, narrow cause-safe physical404 and explicit privateRetryFunc boundary.
- Owned Member Record is a finite immutable JSON-domain convenience profile. Supplied Record identity is copied into a fresh Member request rather than reusing a Python Resource/Munch/subclass or mutating the same instance. Arbitrary object/descriptor/default/history/Header/URI lifecycle remains SDK-R1, Adapter cache/conditional GET remains SDK-C1, and complete session/base_path/microversion/class/exception/conflict controls remain SDK-S1 in docs/implementation-plan.md; these global requirements are not declared completed by this mapping.
- Go uses encoding/json snapshots, complete validUTF8 JSON for selected known values, fixed literal UTF8 identity validation/escaping and canonical member_id precedence on Add. Python arbitrary objects, kwargs insertion order, alternate parser/surrogate/number spelling and coercion are not runtime-equivalent. Unknown Source attributes are discarded by the fixed Member profile; the Go factory may encode an unknown value before its captured error is ignored at selection.
- Go fixed-profile Source controls are rejected before HTTP instead of implementing dynamic constructor/routing/session behavior. In particular the pinned Add helper's __conflicting_attrs can replace bound image_id, but Go reserves that control. Source microversion negotiation and arbitrary class/resource/connection/base_path overrides remain SDK-S1. Explicit parent Name resolution is an additive Go native resolver convenience; no supplied Member parent recovery is added.
- Owned Remove returns an actual opaque200..399 acknowledgement instead of Python None. Go preserves accepted partial bytes/header/status and causes, and only suppresses a clean final direct native404 after configured native retry/handling. The legacy private404 retry bypass is confined to RemoveImageMember. Python's optional image signature does not recover a supplied Member parent; Go requires explicit parent on every call. No actual Python SDK, authenticated deployment or cloud mutation was executed by this local contract audit.

### `python:image/v2/find_member`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.FindImageMember`
- `github.com/JSYoo5B/go-openstacksdk/image.WithFindImageMemberOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithFindImageMemberHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithFindImageMemberHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithFindImageMemberIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.FindImageMemberRecord`
- `github.com/JSYoo5B/go-openstacksdk/image.WithFindImageMemberRecordOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithFindImageMemberRecordHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithFindImageMemberRecordHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithFindImageMemberRecordIgnoreMissing`

- All methods require an explicit parent image Ref; direct imageID has only fixed member endpoint, explicit parentName uses unchanged native exact-name resolver. No Member parent recovery, memberName/project lookup, response-ID synthesis or response-driven routes.
  Tests: `image/members_contracts_test.go:TestImageMembersFixedParentRoutesAndBodies`, `image/members_core_test.go:TestImageMembersCoreFixedRoutesAndActualEvidence`, `connection_image_members_test.go:TestConnectionImageMembersShareClientAndConcreteDefaults`, `internal/cmd/sdkgen/glance_members_test.go:TestGlanceMemberWorkflowsPreserveNativeScopeAndConcreteDefaults`.
- Canonical ImageMember requires validUTF8 nonnull JSON object. Optional image_id/member_id/status/schema/created_at/updated_at are strict nullable strings; missing/null nil, empty present, literal dates unparsed. Unknown/case-decoy/large extension numbers are independent raw fields, links passive/nil typed. Mismatched response IDs and arbitrary response status remain passive; atomic schema errors retain original actual200 proof.
  Tests: `image/members_contracts_test.go:TestImageMembersCanonicalRawModelsAndFiniteList`, `image/members_core_test.go:TestImageMembersCoreCanonicalPresenceAndAtomicDecoder`.
- Source/context/client/provider/prefix/ordinary headers captured before usercallbacks. MemberID is nonempty safeUTF8 single unescaped segment; status input exactly pending/accepted/rejected; invalid inputs/protectedheaders/nilerrorcallbacks fail before parentName/memberHTTP. ParentName failures remain errors despite missing defaults.
  Tests: `image/members_contracts_test.go:TestImageMembersOwnedOptionsAndPreflight`, `image/members_core_test.go:TestImageMembersCoreCompletePreflightAndParentMissing`, `image/members_options_test.go:TestImageMembersOptionsValidationAndDefaults`.
- Concrete FullOpts replaces configuration, Header/Headers merge; map/pointer/helper input/optionslice/percallback configs independently copied and callbacks applied once. Prepared route/headers stay fixed with live originalprovider token; callback/source/provider and postName changes rejected. Reuse and parallel helpers retain owned settings.
  Tests: `image/members_contracts_test.go:TestImageMembersOwnedOptionsAndPreflight`, `image/members_core_test.go:TestImageMembersCoreCapturedSourceAndRequestOwnership`, `image/members_options_test.go:TestImageMembersOptionsSnapshotsReplacementAndLastWins`, `image/members_options_test.go:TestImageMembersOptionsCallbacksPointersAndSliceOwnership`, `image/members_options_test.go:TestImageMembersOptionsParallelReusableHelpers`.
- Accepted200 typed failures read/Close/context/UTF8/JSON/schema return nil model plus ownedResponseError retaining actual bytes/header/status and original read/Close/ctx.Err/customcause. Remove actual204 yields opaque actualack including nonnilack+error on handlingfailure. BodyClose once, no acceptedbody replay; rejectedstatuses keepnative ownership.
  Tests: `image/members_contracts_test.go:TestImageMembersAcceptedEvidenceAndMutationNoReplay`, `image/members_core_test.go:TestImageMembersCoreOwnedBodyFailuresAndPhysical404`.
- Shared REST/fixedrequest keeps configured native prebody retry/reauth/backoff and identical-target redirect callbacks, blocks changedmethod/origin/path/query before transport. Owned serializedbody/RawBody/KeepResponseBody/JSONResponse cannot mutate; nilbody differsJSONnull, same serializedreplacement allowed. Actual status gate remains200/204 despite hook-expandedOkCodes. Native reauth ErrOriginal/ErrReauth fields and existing native memberABI/scopes unchanged.
  Tests: `image/members_contracts_test.go:TestImageMembersNativeHooksAndCompatibility`, `image/members_core_test.go:TestImageMembersCoreCapturedSourceAndRequestOwnership`, `internal/cmd/sdkgen/glance_members_test.go:TestGlanceMemberWorkflowsPreserveNativeScopeAndConcreteDefaults`.
- Find is literal fixedmember GET only, no400/403/LIST/memberName fallback or ID equality assertion. IgnoreMissing nil/defaulttrue accepts only owned fullyhandled GET404=>nil,nil, failure retains404 ResponseError. False preserves strict200/native404 callback/retry; parentName/logical/nested404 neverignored. Server may mask authorization404, so nil is not absence proof.
  Tests: `image/members_contracts_test.go:TestImageMembersFixedParentRoutesAndBodies`, `image/members_contracts_test.go:TestImageMembersAcceptedEvidenceAndMutationNoReplay`, `image/members_core_test.go:TestImageMembersCoreOwnedBodyFailuresAndPhysical404`, `image/members_contracts_test.go:TestImageMembersNativeHooksAndCompatibility`, `image/members_core_test.go:TestImageMembersCoreCompletePreflightAndParentMissing`.
- Owned Find seeds the exact member string as id and accepts actual 200..399 GET. Invalid JSON/empty accepted bodies retain seeded defaults with Wire nil; valid nonobject/whole UTF-8 faults fail. Known Body response attrs overlay the seed with ordered alias/null semantics while parent URI/location stay fixed.
  Tests: `image/member_record_find_test.go:TestFindImageMemberRecordSeededGetAndTolerance`.
- Only clean final native direct-GET 400/403/404 enables the shared query-free member list. Resource.Collection.FindIdentity matches exact declared ID or inherited name, detects a second match immediately, scans later pages after the first candidate, discards candidates on later failure and never refetches a selected row.
  Tests: `image/member_record_find_test.go:TestFindImageMemberRecordCompatibleFallbackAndUniqueness`.
- Rejected GET 400/403/404 Read/Close/source failures remain terminal with original native proof. Glance explicitly preserves an identical native RetryFunc return as a clean final rejection; extra hook causes, wrapped/nested errors, accepted IO faults and SDK status-policy failures cannot enable fallback. The default retry policy of unrelated workflows remains unchanged.
  Tests: `image/member_record_find_test.go:TestFindImageMemberRecordNativeHooksAndTerminalFailures`, `image/member_records_test.go:TestImageMemberRecordsReadCloseAndStickyGuardEvidence`, `blockstorage/snapshot_mutation_transport_test.go:TestVolumeSnapshotMutationDelete404RequiresCleanReadCloseAndGuard`, `api/compute_keypair_cloud_test.go:TestComputeCloudKeypairCreateAndDeleteOutcomes`.
- IgnoreMissing defaults to true only after successful exhaustive lookup finds no match; false exposes ErrNotFound and duplicate/physical/list/parent-resolution failures remain errors. Present id:null suppresses alternate identity while inherited name can still match. The original nonblank UTF-8 member string is compared unchanged and escaped as one GET segment.
  Tests: `image/member_record_find_test.go:TestFindImageMemberRecordCompatibleFallbackAndUniqueness`, `image/member_record_find_test.go:TestFindImageMemberRecordPreflightAndLegacyBoundaries`.
- One operation captures source, headers, options and current location before callbacks and shares the prepared parent/client between GET and fallback pages. Callback ownership and original provider/token are preserved; current source changes are checked at each physical and yield boundary.
  Tests: `image/member_record_find_test.go:TestFindImageMemberRecordOptionsLocationAndFallbackSnapshot`, `image/member_records_test.go:TestImageMemberRecordsReadCloseAndStickyGuardEvidence`.
- Cached Connection Image and ImageV2 calls share native clients, option factory snapshots and location across the complete GET-list search; selected rows retain actual page status/proof. Existing literal FindImageMember and generated scoped Members remain unchanged.
  Tests: `connection_image_member_records_test.go:TestConnectionImageMemberRecordsShareLocationAndFallback`, `image/member_record_find_test.go:TestFindImageMemberRecordPreflightAndLegacyBoundaries`.

Python/Go differences:

- Existing FindImageMember retains its seven legacy literal-GET contracts above; FindImageMemberRecord supplies owned GET-first exact ID/name search and duplicate detection without changing that ABI.
- Go returns independent ImageMemberRecord values rather than reusing mutable cached Python Member instances. Explicit parent-name resolution and request header options are Go extensions. ErrNotFound/ErrAmbiguous replace Python exception classes and the upcoming-default warning is not emitted.
- Only selected rejected GET 400/403/404 bodies receive physical IO observation; other native rejected statuses/list attempts retain existing native IO policy. Whole UTF-8, protected headers, safe literal input and fixed-route continuation bounds remain explicit Go differences.
- Glance server may mask authorization as 404; missing nil is not proof of absence. Source inspection/local HTTP tests do not establish authenticated cloud behavior, and full Resource/session/cache/discovery remains a separate SDK objective.

### `python:image/v2/members`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.ListImageMembers`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImageMembersOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImageMembersHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImageMembersHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImageMembersMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.AllImageMembers`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.ListImageMemberRecords`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.AllImageMemberRecords`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordListHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordListFilter`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordListFilters`

- All methods require an explicit parent image Ref; direct imageID has only fixed member endpoint, explicit parentName uses unchanged native exact-name resolver. No Member parent recovery, memberName/project lookup, response-ID synthesis or response-driven routes.
  Tests: `image/members_contracts_test.go:TestImageMembersFixedParentRoutesAndBodies`, `image/members_core_test.go:TestImageMembersCoreFixedRoutesAndActualEvidence`, `connection_image_members_test.go:TestConnectionImageMembersShareClientAndConcreteDefaults`, `internal/cmd/sdkgen/glance_members_test.go:TestGlanceMemberWorkflowsPreserveNativeScopeAndConcreteDefaults`.
- Finite collection requires whole validUTF8 nonnull JSONobject and canonical nonnull membersarray. Consumed ImageMember rows atomically decode nullable string fields/literaldates and own independent raw unknownfields/header/status; unusedrowcanonical errors skipped oncap/earlybreak. No envelope/schema/link pagination or response-ID routing.
  Tests: `image/members_contracts_test.go:TestImageMembersCanonicalRawModelsAndFiniteList`, `image/members_core_test.go:TestImageMembersCoreCanonicalPresenceAndAtomicDecoder`, `image/members_core_test.go:TestImageMembersCoreFiniteLazyListAndConsumedRows`.
- Source/context/client/provider/prefix/ordinary headers captured before usercallbacks. MemberID is nonempty safeUTF8 single unescaped segment; status input exactly pending/accepted/rejected; invalid inputs/protectedheaders/nilerrorcallbacks fail before parentName/memberHTTP. ParentName failures remain errors despite missing defaults.
  Tests: `image/members_contracts_test.go:TestImageMembersOwnedOptionsAndPreflight`, `image/members_core_test.go:TestImageMembersCoreCompletePreflightAndParentMissing`, `image/members_options_test.go:TestImageMembersOptionsValidationAndDefaults`.
- Concrete FullOpts replaces configuration, Header/Headers merge; map/pointer/helper input/optionslice/percallback configs independently copied and callbacks applied once. Prepared route/headers stay fixed with live originalprovider token; callback/source/provider and postName changes rejected. Reuse and parallel helpers retain owned settings.
  Tests: `image/members_contracts_test.go:TestImageMembersOwnedOptionsAndPreflight`, `image/members_core_test.go:TestImageMembersCoreCapturedSourceAndRequestOwnership`, `image/members_options_test.go:TestImageMembersOptionsSnapshotsReplacementAndLastWins`, `image/members_options_test.go:TestImageMembersOptionsCallbacksPointersAndSliceOwnership`, `image/members_options_test.go:TestImageMembersOptionsParallelReusableHelpers`.
- Collection acceptedread/Close/context/wholeUTF8/JSON/envelopeschema failure yields terminalnilrow+ownedResponseError retaining full original200 bytes/header/status and causes. Consumed malformedrow retains same fullproof; All discards partialrows. BodyClose once and no acceptedbody replay.
  Tests: `image/members_contracts_test.go:TestImageMembersAcceptedEvidenceAndMutationNoReplay`, `image/members_core_test.go:TestImageMembersCoreOwnedBodyFailuresAndPhysical404`, `image/members_contracts_test.go:TestImageMembersCanonicalRawModelsAndFiniteList`, `image/members_core_test.go:TestImageMembersCoreFiniteLazyListAndConsumedRows`.
- Shared REST/fixedrequest keeps configured native prebody retry/reauth/backoff and identical-target redirect callbacks, blocks changedmethod/origin/path/query before transport. Owned serializedbody/RawBody/KeepResponseBody/JSONResponse cannot mutate; nilbody differsJSONnull, same serializedreplacement allowed. Actual status gate remains200/204 despite hook-expandedOkCodes. Native reauth ErrOriginal/ErrReauth fields and existing native memberABI/scopes unchanged.
  Tests: `image/members_contracts_test.go:TestImageMembersNativeHooksAndCompatibility`, `image/members_core_test.go:TestImageMembersCoreCapturedSourceAndRequestOwnership`, `internal/cmd/sdkgen/glance_members_test.go:TestGlanceMemberWorkflowsPreserveNativeScopeAndConcreteDefaults`.
- List snapshot is lazy periteration; each iteration makes one finite body/query-freeGET200 after required parentresolution. Required canonical members nonnullarray, wholeUTF8/JSON before rows; rowcanonical decode onlywhenconsumed. MaxItems zero unlimited,negative invalid,positive localrawrowcap; earlybreak/cap skipsunusedrowerrors/no continuation/query/fanout. Eachrow owns metadata/header/raw, source/context checked perrow. All helper returns nonnil[] onempty and discards partial onerror.
  Tests: `image/members_contracts_test.go:TestImageMembersCanonicalRawModelsAndFiniteList`, `image/members_core_test.go:TestImageMembersCoreFiniteLazyListAndConsumedRows`, `image/members_contracts_test.go:TestImageMembersOwnedOptionsAndPreflight`, `image/members_options_test.go:TestImageMembersOptionsValidationAndDefaults`.
- Owned member records add the pinned nine-field declared view beside unchanged finite typed APIs. Untyped Body values and ordered member/member_id aliases preserve raw precision, canonical alias precedence and present null ID; fixed parent URI and current location override passive response fields. Resource, Wire, Envelope, headers and identity own independent bytes.
  Tests: `image/member_records_test.go:TestImageMemberRecordsProjectionAndOwnedReceipts`, `image/member_records_test.go:TestImageMemberRecordsIdentityAliasesAndLocation`.
- Public Proxy.members drops all query kwargs. The owned default is query-free with generic pagination enabled; explicit Go options add server limit/marker, raw-row cap, pagination, headers and declared local filters. Unknown filter keys are ignored before encoding and exact JSON semantic filtering reuses the shared selector.
  Tests: `image/member_records_test.go:TestImageMemberRecordsPreflightAndExtensionFilters`, `image/member_records_test.go:TestImageMemberRecordsLazyRepeatedOptionSnapshots`.
- Shared guarded pagination follows fixed collection links and HTTP Link, ignores Source-falsey top-level next values and permits initial-positive-limit wire-marker fallback. Unused rows/continuations are skipped at raw cap or early break; partial All results survive later errors, and nil/empty/member-object/array envelopes follow the declared collection policy.
  Tests: `image/member_records_test.go:TestImageMemberRecordsPagingAndRawConsumption`, `internal/rest/falsey_next_test.go:TestListFalseyNextIsOptInAndAllowsHeaderOrMarkerFallback`, `internal/rest/falsey_next_test.go:TestListTruthyNonstringNextAndDefaultFalseyTypeKeepExactPageEvidence`.
- Owned pages accept actual 200..399 and require whole UTF-8 JSON with the mandatory members envelope. Consumed constructor binding collisions and malformed rows are terminal with actual page proof; accepted Read/Close/context/source faults retain independent receipt and cause without replay.
  Tests: `image/member_records_test.go:TestImageMemberRecordsEnvelopeFailuresAndActualStatus`, `image/member_records_test.go:TestImageMemberRecordsReadCloseAndStickyGuardEvidence`.
- Options/helpers and iteration slices snapshot maps and pointers, full options replace and merge helpers use last-wins. Each lazy iteration captures source and headers before callbacks/location, resolves an explicit parent once, retains live provider token, and makes observed source faults sticky across Read and Close.
  Tests: `image/member_records_test.go:TestImageMemberRecordsLazyRepeatedOptionSnapshots`, `image/member_records_test.go:TestImageMemberRecordsPreflightAndExtensionFilters`, `image/member_records_test.go:TestImageMemberRecordsReadCloseAndStickyGuardEvidence`.
- Both Connection Image/ImageV2 entries retain the same native service client and current location through owned list and GET-list fallback. Legacy finite member list, literal Find and generated native member ABI remain separate and unchanged.
  Tests: `connection_image_member_records_test.go:TestConnectionImageMemberRecordsShareLocationAndFallback`, `image/member_records_test.go:TestImageMemberRecordsPreserveLegacyFiniteMemberList`, `image/member_record_find_test.go:TestFindImageMemberRecordPreflightAndLegacyBoundaries`.

Python/Go differences:

- Existing ListImageMembers/AllImageMembers retain their seven legacy contracts above; ListImageMemberRecords/AllImageMemberRecords supply the owned pinned view and generic pager without changing that ABI.
- Go limit/marker/max-items/paginated/header/local-filter options and explicit parent-name resolution are extensions; the pinned public members proxy drops every query kwarg. Go exposes fixed image_id URI provenance in the declared view although Python to_dict excludes URI fields by default.
- Whole UTF-8, protected headers, safe literal parent identity, fixed-origin/path/query pagination and conflicting-link rejection are Go bounds. Exact decimal truthiness can differ from Python binary underflow; physical Close faults remain errors.
- Mutable Python Resource/cache/session/discovery identity remains a separate whole-SDK objective. These named contracts use Source inspection and local HTTP tests; authenticated OpenStack/Python runtime behavior was not executed.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"
    "fmt"

    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func shareAndReadMembers(ctx context.Context, owner, member *image.Service, parent resource.Ref, memberID string) (*image.ImageMemberAcknowledgement, error) {
    common := []image.ImageMemberOption{
        image.WithImageMemberOpts(image.ImageMemberOpts{
            Headers: map[string]string{"X-Request-Source": "example"},
        }),
        image.WithImageMemberHeader("X-Member-Request", "single"),
        image.WithImageMemberHeaders(map[string]string{"X-Example": "members"}),
    }
    created, err := owner.AddImageMember(ctx, parent, memberID, common...)
    if err != nil {
        return nil, err
    }
    fmt.Println("created response", created.StatusCode)
    current, err := owner.GetImageMember(ctx, parent, memberID, common...)
    if err != nil {
        return nil, err
    }
    fmt.Println("current response", current.StatusCode)
    updated, err := member.UpdateImageMember(ctx, parent, memberID, "accepted", common...)
    if err != nil {
        return nil, err
    }
    fmt.Println("updated response", updated.StatusCode)

    found, err := owner.FindImageMember(ctx, parent, memberID,
        image.WithFindImageMemberOpts(image.FindImageMemberOpts{}),
        image.WithFindImageMemberHeader("X-Member-Request", "find"),
        image.WithFindImageMemberHeaders(map[string]string{"X-Example": "members"}),
        image.WithFindImageMemberIgnoreMissing(false))
    if err != nil {
        return nil, err
    }
    fmt.Println("found response", found.StatusCode)

    listOptions := []image.ListImageMembersOption{
        image.WithListImageMembersOpts(image.ListImageMembersOpts{}),
        image.WithListImageMembersHeader("X-Member-Request", "list"),
        image.WithListImageMembersHeaders(map[string]string{"X-Example": "members"}),
        image.WithListImageMembersMaxItems(20),
    }
    for row, err := range owner.ListImageMembers(ctx, parent, listOptions...) {
        if err != nil {
            return nil, err
        }
        fmt.Println("member row fields", len(row.Body))
    }
    all, err := owner.AllImageMembers(ctx, parent, listOptions...)
    if err != nil {
        return nil, err
    }
    fmt.Println("collected members", len(all))

    ack, err := owner.RemoveImageMember(ctx, parent, memberID,
        image.WithRemoveImageMemberOpts(image.RemoveImageMemberOpts{}),
        image.WithRemoveImageMemberHeader("X-Member-Request", "remove"),
        image.WithRemoveImageMemberHeaders(map[string]string{"X-Example": "members"}),
        image.WithRemoveImageMemberIgnoreMissing(true))
    return ack, err
}
```

```python
def share_and_read_members(owner_conn, member_conn, image_id, member_id):
    created = owner_conn.image.add_member(image_id, member=member_id)
    current = owner_conn.image.get_member(member_id, image_id)
    updated = member_conn.image.update_member(
        member_id, image_id, status="accepted"
    )
    rows = list(owner_conn.image.members(image_id))
    found = owner_conn.image.find_member(
        member_id, image_id, ignore_missing=False
    )
    removed = owner_conn.image.remove_member(
        member_id, image=image_id, ignore_missing=True
    )
    return created, current, updated, rows, found, removed
```

## References

- [Member Record Mutations](member-record-mutations.md)
- [Member Records](member-records.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1222-L1386)
- [Resource.find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2489-L2582)
- [Image Members](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/image_members.py#L234-L280)
- [serializer](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/image_members.py#L396-L435)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#list-image-members)
- [show](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/image_members.py#L282-L312)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#create-image-member)
- [Members Contracts Test](members_contracts_test.go)
- [core tests](members_core_test.go)
- [option tests](members_options_test.go)
- [Connection test](../connection_image_members_test.go)
- [generator test](../internal/cmd/sdkgen/glance_members_test.go)
