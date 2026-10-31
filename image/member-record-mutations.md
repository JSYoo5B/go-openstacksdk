# Member Record Mutations

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
  Tests: `image/members_contracts_test.go:TestImageMembersFixedParentRoutesAndBodies`, `image/members_core_test.go:TestImageMembersCoreFixedRoutesAndActualEvidence`, `connectiontest/connection_image_members_test.go:TestConnectionImageMembersShareClientAndConcreteDefaults`, `internal/cmd/sdkgen/glance_members_test.go:TestGlanceMemberWorkflowsPreserveNativeScopeAndConcreteDefaults`.
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
  Tests: `image/member_record_mutations_test.go:TestImageMemberRecordMutationsAcceptedPhysicalFailuresKeepPartialEvidence`, `image/member_record_mutations_test.go:TestImageMemberRecordFamilyRejectsNativeStatusesWithoutFakeReceipt`, `connectiontest/member_record_mutations_integration_test.go:TestConnectionOwnedImageMemberFamilySharesSourceLocationAndActualReceipts`.
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
  Tests: `image/members_contracts_test.go:TestImageMembersFixedParentRoutesAndBodies`, `image/members_core_test.go:TestImageMembersCoreFixedRoutesAndActualEvidence`, `connectiontest/connection_image_members_test.go:TestConnectionImageMembersShareClientAndConcreteDefaults`, `internal/cmd/sdkgen/glance_members_test.go:TestGlanceMemberWorkflowsPreserveNativeScopeAndConcreteDefaults`.
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
  Tests: `image/member_record_mutations_test.go:TestImageMemberRecordMutationsAcceptedPhysicalFailuresKeepPartialEvidence`, `image/member_record_mutations_test.go:TestImageMemberRecordFamilyRejectsNativeStatusesWithoutFakeReceipt`, `connectiontest/member_record_mutations_integration_test.go:TestConnectionOwnedImageMemberFamilySharesSourceLocationAndActualReceipts`.
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
  Tests: `image/members_contracts_test.go:TestImageMembersFixedParentRoutesAndBodies`, `image/members_core_test.go:TestImageMembersCoreFixedRoutesAndActualEvidence`, `connectiontest/connection_image_members_test.go:TestConnectionImageMembersShareClientAndConcreteDefaults`, `internal/cmd/sdkgen/glance_members_test.go:TestGlanceMemberWorkflowsPreserveNativeScopeAndConcreteDefaults`.
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
  Tests: `image/member_record_mutations_test.go:TestImageMemberRecordMutationsAcceptedPhysicalFailuresKeepPartialEvidence`, `image/member_record_mutations_test.go:TestImageMemberRecordFamilyRejectsNativeStatusesWithoutFakeReceipt`, `connectiontest/member_record_mutations_integration_test.go:TestConnectionOwnedImageMemberFamilySharesSourceLocationAndActualReceipts`.
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
  Tests: `image/members_contracts_test.go:TestImageMembersFixedParentRoutesAndBodies`, `image/members_core_test.go:TestImageMembersCoreFixedRoutesAndActualEvidence`, `connectiontest/connection_image_members_test.go:TestConnectionImageMembersShareClientAndConcreteDefaults`, `internal/cmd/sdkgen/glance_members_test.go:TestGlanceMemberWorkflowsPreserveNativeScopeAndConcreteDefaults`.
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
  Tests: `image/member_record_mutations_test.go:TestImageMemberRecordMutationsAcceptedPhysicalFailuresKeepPartialEvidence`, `image/member_record_mutations_test.go:TestImageMemberRecordFamilyRejectsNativeStatusesWithoutFakeReceipt`, `connectiontest/member_record_mutations_integration_test.go:TestConnectionOwnedImageMemberFamilySharesSourceLocationAndActualReceipts`.
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

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

owner = openstack.connect(cloud="owner")
recipient = openstack.connect(cloud="recipient")
image_id = "image-id"
member_id = "recipient-project-id"

created = owner.image.add_member(image_id, member_id=member_id)
fetched = owner.image.get_member(member_id, image_id)
updated = recipient.image.update_member(fetched, image_id, status="accepted")
owner.image.remove_member(member_id, image_id, ignore_missing=False)
```

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
    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    ownerCloud := flag.String("owner-cloud", os.Getenv("OPENSTACKSDK_OWNER_CLOUD"), "owner clouds.yaml cloud")
    memberCloud := flag.String("member-cloud", os.Getenv("OPENSTACKSDK_MEMBER_CLOUD"), "shared recipient clouds.yaml cloud")
    imageID := flag.String("image-id", os.Getenv("OPENSTACKSDK_IMAGE_ID"), "literal parent image ID")
    memberID := flag.String("member-id", os.Getenv("OPENSTACKSDK_MEMBER_ID"), "literal recipient project ID")
    status := flag.String("status", "accepted", "requested member status")
    flag.Parse()
    if *ownerCloud == "" || *memberCloud == "" || *imageID == "" || *memberID == "" {
        log.Fatal("owner-cloud, member-cloud, image-id and member-id are required")
    }
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    if err := run(ctx, *ownerCloud, *memberCloud, *imageID, *memberID, *status); err != nil {
        var response *resource.ResponseError
        if errors.As(err, &response) {
            fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
        }
        log.Fatal(err)
    }
}

func showRecord(operation string, record *image.ImageMemberRecord, callErr error) error {
    value := map[string]any{"operation": operation}
    if record != nil {
        value["resource"] = record.Resource
        value["wire"] = record.Wire
        value["image_id"] = record.ImageID
        value["status_code"] = record.StatusCode
        value["envelope_bytes"] = len(record.Envelope)
    }
    if callErr != nil { value["error"] = callErr.Error() }
    body, err := json.MarshalIndent(value, "", "  ")
    if err != nil { return err }
    fmt.Println(string(body))
    return callErr
}

func showAcknowledgement(ack *image.ImageMemberAcknowledgement, callErr error) error {
    value := map[string]any{"operation": "remove"}
    if ack != nil {
        value["image_id"] = ack.ImageID
        value["member_id"] = ack.MemberID
        value["status_code"] = ack.StatusCode
        value["response_bytes"] = len(ack.Body)
    }
    if callErr != nil { value["error"] = callErr.Error() }
    body, err := json.MarshalIndent(value, "", "  ")
    if err != nil { return err }
    fmt.Println(string(body))
    return callErr
}

func run(ctx context.Context, ownerCloud, memberCloud, imageID, memberID, status string) error {
    owner, err := openstack.Connect(ctx, openstack.WithCloud(ownerCloud))
    if err != nil { return err }
    recipient, err := openstack.Connect(ctx, openstack.WithCloud(memberCloud))
    if err != nil { return err }
    parent := resource.ID(imageID)
    input := image.ImageMemberRecordRequest{ID: memberID}
    created, err := owner.AddImageMemberRecord(ctx, parent,
        image.WithImageMemberRecordMemberID(memberID),
        image.WithImageMemberRecordWriteHeader("X-Request-Source", "member-record-example"))
    if err := showRecord("add", created, err); err != nil { return err }
    fetched, err := owner.GetImageMemberRecord(ctx, parent, input,
        image.WithImageMemberHeader("X-Request-Source", "member-record-example"))
    if err := showRecord("get", fetched, err); err != nil { return err }
    updated, err := recipient.UpdateImageMemberRecord(ctx, parent,
        image.ImageMemberRecordRequest{Record: fetched},
        image.WithImageMemberRecordStatus(status))
    if err := showRecord("update", updated, err); err != nil { return err }
    service, err := recipient.Image(ctx)
    if err != nil { return err }
    repeated, err := service.UpdateImageMemberRecord(ctx, parent,
        image.ImageMemberRecordRequest{Record: updated},
        image.WithImageMemberRecordStatus(status))
    if err := showRecord("update-again", repeated, err); err != nil { return err }
    ack, err := owner.RemoveImageMemberRecord(ctx, parent, input,
        image.WithRemoveImageMemberIgnoreMissing(false),
        image.WithRemoveImageMemberHeader("X-Request-Source", "member-record-example"))
    return showAcknowledgement(ack, err)
}
```

## References

- [Member Records](member-records.md)
- [Members](members.md)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L223-L292)
- [Install](../docs/install.md)
- [add_member 1222–1241](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1222-L1241)
- [get_member 1316–1336](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1316-L1336)
- [update_member 1356–1386](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1356-L1386)
- [remove_member 1243–1271](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1243-L1271)
- [Member descriptor 16–43](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/member.py#L16-L43)
- [generic proxy create/update/delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L669-L799)
- [Resource response translation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338-L1394)
- [microversion negotiation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1418-L1439)
- [create](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1519-L1638)
- [commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881-L1937)
- [delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2094-L2152)
