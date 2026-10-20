# Tags Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/add_tag`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags.API.AddTag`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags.WithSourceSecurityGroupRuleTags`

- One bodyless individual PUT appends even duplicate tag to an independently copied cache. Cached null causes a post-HTTP local-transition error, retaining successful acknowledgement; input ref is unchanged and no rollback/fetch/dedup occurs.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsCacheTransitionsDoNotInventResponseState`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsNullCacheFailureKeepsAcknowledgement`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsLiteralSegmentsExtensionsAndSourceHeaders`.
- SDK-owned ResourceKind routes cover nine stock-compatible classes plus explicit Source-only SecurityGroupRule capability without caller builders, arbitrary resource URLs, parent GETs or name lookup. QoS tag parent is stock unprefixed policies; API reference and cached tag slices remain immutable.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsKindsRoutesAndBulkCapabilities`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsLiteralSegmentsExtensionsAndSourceHeaders`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsCacheTransitionsDoNotInventResponseState`.
- Captured selected Neutron Source and concrete options guard source/client/route/context before callbacks and physical calls. Unknown kind, invalid literal ID/tag, nil option, reserved tags Body, forbidden auth/framing Header, query/secondary arguments and body on bodyless calls fail before HTTP. Caller input slices/options/extension JSON are independent snapshots.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsPreflightAndSourceCapture`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsCallerSlicesAndRetainedOptionsAreSnapshots`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsLiteralSegmentsExtensionsAndSourceHeaders`.
- Actual 200..399 success policy preserves response Body/Header/StatusCode separately from input-derived cache. Strict rejected404 and other errors keep native status/body/header and callback causes. Accepted/rejected physical Read/Close, context custom cause and sticky source faults are retained; clean retries preserve request serialization and tampering stops before resend.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsStrictMissingAndAcceptedCodes`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsPhysicalFaultsRetainAcceptedAndRejectedProof`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsRetryOwnershipCleanRetryAndDeniedCause`.
- Pinned Python declares SecurityGroupRule tag capabilities while stock Neutron27.0.0 tag registry lacks that kind. All seven default SGR calls fail ErrUnsupported before HTTP; SDK-owned WithSourceSecurityGroupRuleTags explicitly permits the fixed Source route on a backend providing it. These fixtures establish Source compatibility only, not stock27 endpoint/authorization acceptance. The option is rejected on other resource kinds.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsSecurityGroupRuleSourceCapabilityIsExplicit`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsKindsRoutesAndBulkCapabilities`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsPreflightAndSourceCapture`.
- Library-owned HeaderPolicy fixes selected generic network microversion value or absence and rejects incompatible Nova/Cinder version headers plus reserved auth/framing header injection through native retry. Bulk JSON Content-Type retains selected service/option value or application/json; owned value deletion and OmitHeaders suppression fail before resend while retaining the original503 proof and callback causes. Ordinary retry headers and bodyless Content-Type remain native; shared live X-Auth-Token is refreshed rather than frozen.
  Tests: `network/v2/extensions/attributestags/python_tags_policy_test.go:TestPythonTagsRetryCannotChangeSelectedOrAbsentVersionHeaders`, `network/v2/extensions/attributestags/python_tags_policy_test.go:TestPythonTagsRetryCannotAddReservedHeaders`, `network/v2/extensions/attributestags/python_tags_policy_test.go:TestPythonTagsRetryOwnsBulkJSONContentTypeAndOmissions`, `network/v2/extensions/attributestags/python_tags_policy_test.go:TestPythonTagsOrdinaryRetryHeadersAndLiveTokenKeepNativePolicy`.

Python/Go differences:

- Go returns explicit context/error and immutable concrete TaggedResource/TagResult instead of mutating arbitrary Python Resource subclasses. Fresh cache is [], explicit cached/fetched null remains nil; actual response evidence is separate from local cache.
- Go owns ten fixed resource kinds and literal escaped ID/tag segments. Empty/dot/control/non-UTF-8 individual segments are rejected before HTTP. Dynamic user subclasses/extension registry are separate shared SDK work, not proof of completed full Resource parity.
- Source QoSPolicy.set_tags overrides /policies and omits explicit raise_from_response; its inherited other TagMixin methods use /qos/policies. Go consistently uses stock 27.0.0 tag-controller policies path and checks actual errors for all methods.
- Go Header options and bulk Body extension options are explicit additions with SDK-owned auth/framing/core-field restrictions. They do not invent Python inventory operations or change server authorization.
- Source SecurityGroupRule tagging is explicit compatibility-only via WithSourceSecurityGroupRuleTags. Default stock27 capability is preflight ErrUnsupported; backend support and authenticated acceptance are not inferred from a fixture.
- The owned tag lane accepts final HTTP200..399; custom-transport final1xx status is not covered by this bounded named mapping. Pinned exceptions.py generic status policy accepts all statuses below400.
- Pinned Source Proxy.request supports optional Connection HTTP response caching: GET tag reads/checks can return cached responses when caching/expiration is configured, while writes invalidate the cache prefix. Go deliberately performs transport GETs and has only immutable local tag state; shared response-cache hits/invalidation are not implemented.
- Go owns operation protocol/auth/framing constraints across native retry. Selected network version and bulk JSON Content-Type values or required absence cannot be changed or hidden with OmitHeaders. Ordinary retry headers, bodyless Content-Type and shared live X-Auth-Token keep native behavior; this is explicit Go policy, not evidence of arbitrary Python Session parity.

### `python:network/v2/add_tags`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags.API.AddTags`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags.WithSourceSecurityGroupRuleTags`

- One POST on eight Network TagMixin kinds implements tags-or-[] for nil/empty lists. Returned cache is supplied list rather than old/new union, while actual response is retained independently. QoSPolicy and Trunk common TagMixin lacks Source add_tags; these kinds return explicit ErrUnsupported before HTTP and do not create invented inventory rows.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsKindsRoutesAndBulkCapabilities`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsCacheTransitionsDoNotInventResponseState`.
- SDK-owned ResourceKind routes cover nine stock-compatible classes plus explicit Source-only SecurityGroupRule capability without caller builders, arbitrary resource URLs, parent GETs or name lookup. QoS tag parent is stock unprefixed policies; API reference and cached tag slices remain immutable.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsKindsRoutesAndBulkCapabilities`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsLiteralSegmentsExtensionsAndSourceHeaders`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsCacheTransitionsDoNotInventResponseState`.
- Captured selected Neutron Source and concrete options guard source/client/route/context before callbacks and physical calls. Unknown kind, invalid literal ID/tag, nil option, reserved tags Body, forbidden auth/framing Header, query/secondary arguments and body on bodyless calls fail before HTTP. Caller input slices/options/extension JSON are independent snapshots.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsPreflightAndSourceCapture`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsCallerSlicesAndRetainedOptionsAreSnapshots`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsLiteralSegmentsExtensionsAndSourceHeaders`.
- Actual 200..399 success policy preserves response Body/Header/StatusCode separately from input-derived cache. Strict rejected404 and other errors keep native status/body/header and callback causes. Accepted/rejected physical Read/Close, context custom cause and sticky source faults are retained; clean retries preserve request serialization and tampering stops before resend.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsStrictMissingAndAcceptedCodes`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsPhysicalFaultsRetainAcceptedAndRejectedProof`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsRetryOwnershipCleanRetryAndDeniedCause`.
- Pinned Python declares SecurityGroupRule tag capabilities while stock Neutron27.0.0 tag registry lacks that kind. All seven default SGR calls fail ErrUnsupported before HTTP; SDK-owned WithSourceSecurityGroupRuleTags explicitly permits the fixed Source route on a backend providing it. These fixtures establish Source compatibility only, not stock27 endpoint/authorization acceptance. The option is rejected on other resource kinds.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsSecurityGroupRuleSourceCapabilityIsExplicit`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsKindsRoutesAndBulkCapabilities`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsPreflightAndSourceCapture`.
- Library-owned HeaderPolicy fixes selected generic network microversion value or absence and rejects incompatible Nova/Cinder version headers plus reserved auth/framing header injection through native retry. Bulk JSON Content-Type retains selected service/option value or application/json; owned value deletion and OmitHeaders suppression fail before resend while retaining the original503 proof and callback causes. Ordinary retry headers and bodyless Content-Type remain native; shared live X-Auth-Token is refreshed rather than frozen.
  Tests: `network/v2/extensions/attributestags/python_tags_policy_test.go:TestPythonTagsRetryCannotChangeSelectedOrAbsentVersionHeaders`, `network/v2/extensions/attributestags/python_tags_policy_test.go:TestPythonTagsRetryCannotAddReservedHeaders`, `network/v2/extensions/attributestags/python_tags_policy_test.go:TestPythonTagsRetryOwnsBulkJSONContentTypeAndOmissions`, `network/v2/extensions/attributestags/python_tags_policy_test.go:TestPythonTagsOrdinaryRetryHeadersAndLiveTokenKeepNativePolicy`.

Python/Go differences:

- Go returns explicit context/error and immutable concrete TaggedResource/TagResult instead of mutating arbitrary Python Resource subclasses. Fresh cache is [], explicit cached/fetched null remains nil; actual response evidence is separate from local cache.
- Go owns ten fixed resource kinds and literal escaped ID/tag segments. Empty/dot/control/non-UTF-8 individual segments are rejected before HTTP. Dynamic user subclasses/extension registry are separate shared SDK work, not proof of completed full Resource parity.
- Source QoSPolicy.set_tags overrides /policies and omits explicit raise_from_response; its inherited other TagMixin methods use /qos/policies. Go consistently uses stock 27.0.0 tag-controller policies path and checks actual errors for all methods.
- Go Header options and bulk Body extension options are explicit additions with SDK-owned auth/framing/core-field restrictions. They do not invent Python inventory operations or change server authorization.
- Source method absence for QoSPolicy/Trunk is represented as preflight ErrUnsupported; stock bulk POST capability remains distinct from this pinned proxy mapping.
- Source SecurityGroupRule tagging is explicit compatibility-only via WithSourceSecurityGroupRuleTags. Default stock27 capability is preflight ErrUnsupported; backend support and authenticated acceptance are not inferred from a fixture.
- The owned tag lane accepts final HTTP200..399; custom-transport final1xx status is not covered by this bounded named mapping. Pinned exceptions.py generic status policy accepts all statuses below400.
- Pinned Source Proxy.request supports optional Connection HTTP response caching: GET tag reads/checks can return cached responses when caching/expiration is configured, while writes invalidate the cache prefix. Go deliberately performs transport GETs and has only immutable local tag state; shared response-cache hits/invalidation are not implemented.
- Go owns operation protocol/auth/framing constraints across native retry. Selected network version and bulk JSON Content-Type values or required absence cannot be changed or hidden with OmitHeaders. Ordinary retry headers, bodyless Content-Type and shared live X-Auth-Token keep native behavior; this is explicit Go policy, not evidence of arbitrary Python Session parity.

### `python:network/v2/check_tag`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags.API.CheckTag`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags.WithSourceSecurityGroupRuleTags`

- One bodyless individual GET returns resource plus receipt and retains cached tags on accepted status. Source actual return is Resource despite proxy bool annotation; HTTP404 raises and is not translated to false/nil. Native Confirm keeps its separate API/default.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsCacheTransitionsDoNotInventResponseState`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsStrictMissingAndAcceptedCodes`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsPhysicalFaultsRetainAcceptedAndRejectedProof`.
- SDK-owned ResourceKind routes cover nine stock-compatible classes plus explicit Source-only SecurityGroupRule capability without caller builders, arbitrary resource URLs, parent GETs or name lookup. QoS tag parent is stock unprefixed policies; API reference and cached tag slices remain immutable.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsKindsRoutesAndBulkCapabilities`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsLiteralSegmentsExtensionsAndSourceHeaders`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsCacheTransitionsDoNotInventResponseState`.
- Captured selected Neutron Source and concrete options guard source/client/route/context before callbacks and physical calls. Unknown kind, invalid literal ID/tag, nil option, reserved tags Body, forbidden auth/framing Header, query/secondary arguments and body on bodyless calls fail before HTTP. Caller input slices/options/extension JSON are independent snapshots.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsPreflightAndSourceCapture`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsCallerSlicesAndRetainedOptionsAreSnapshots`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsLiteralSegmentsExtensionsAndSourceHeaders`.
- Actual 200..399 success policy preserves response Body/Header/StatusCode separately from input-derived cache. Strict rejected404 and other errors keep native status/body/header and callback causes. Accepted/rejected physical Read/Close, context custom cause and sticky source faults are retained; clean retries preserve request serialization and tampering stops before resend.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsStrictMissingAndAcceptedCodes`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsPhysicalFaultsRetainAcceptedAndRejectedProof`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsRetryOwnershipCleanRetryAndDeniedCause`.
- Pinned Python declares SecurityGroupRule tag capabilities while stock Neutron27.0.0 tag registry lacks that kind. All seven default SGR calls fail ErrUnsupported before HTTP; SDK-owned WithSourceSecurityGroupRuleTags explicitly permits the fixed Source route on a backend providing it. These fixtures establish Source compatibility only, not stock27 endpoint/authorization acceptance. The option is rejected on other resource kinds.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsSecurityGroupRuleSourceCapabilityIsExplicit`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsKindsRoutesAndBulkCapabilities`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsPreflightAndSourceCapture`.
- Library-owned HeaderPolicy fixes selected generic network microversion value or absence and rejects incompatible Nova/Cinder version headers plus reserved auth/framing header injection through native retry. Bulk JSON Content-Type retains selected service/option value or application/json; owned value deletion and OmitHeaders suppression fail before resend while retaining the original503 proof and callback causes. Ordinary retry headers and bodyless Content-Type remain native; shared live X-Auth-Token is refreshed rather than frozen.
  Tests: `network/v2/extensions/attributestags/python_tags_policy_test.go:TestPythonTagsRetryCannotChangeSelectedOrAbsentVersionHeaders`, `network/v2/extensions/attributestags/python_tags_policy_test.go:TestPythonTagsRetryCannotAddReservedHeaders`, `network/v2/extensions/attributestags/python_tags_policy_test.go:TestPythonTagsRetryOwnsBulkJSONContentTypeAndOmissions`, `network/v2/extensions/attributestags/python_tags_policy_test.go:TestPythonTagsOrdinaryRetryHeadersAndLiveTokenKeepNativePolicy`.

Python/Go differences:

- Go returns explicit context/error and immutable concrete TaggedResource/TagResult instead of mutating arbitrary Python Resource subclasses. Fresh cache is [], explicit cached/fetched null remains nil; actual response evidence is separate from local cache.
- Go owns ten fixed resource kinds and literal escaped ID/tag segments. Empty/dot/control/non-UTF-8 individual segments are rejected before HTTP. Dynamic user subclasses/extension registry are separate shared SDK work, not proof of completed full Resource parity.
- Source QoSPolicy.set_tags overrides /policies and omits explicit raise_from_response; its inherited other TagMixin methods use /qos/policies. Go consistently uses stock 27.0.0 tag-controller policies path and checks actual errors for all methods.
- Go Header options and bulk Body extension options are explicit additions with SDK-owned auth/framing/core-field restrictions. They do not invent Python inventory operations or change server authorization.
- The Source proxy bool annotation/doc is inaccurate: actual TagMixin returns Resource and raises404. Go returns TagResult plus error; it does not apply native Confirm false-on404 behavior.
- Source SecurityGroupRule tagging is explicit compatibility-only via WithSourceSecurityGroupRuleTags. Default stock27 capability is preflight ErrUnsupported; backend support and authenticated acceptance are not inferred from a fixture.
- The owned tag lane accepts final HTTP200..399; custom-transport final1xx status is not covered by this bounded named mapping. Pinned exceptions.py generic status policy accepts all statuses below400.
- Pinned Source Proxy.request supports optional Connection HTTP response caching: GET tag reads/checks can return cached responses when caching/expiration is configured, while writes invalidate the cache prefix. Go deliberately performs transport GETs and has only immutable local tag state; shared response-cache hits/invalidation are not implemented.
- Go owns operation protocol/auth/framing constraints across native retry. Selected network version and bulk JSON Content-Type values or required absence cannot be changed or hidden with OmitHeaders. Ordinary retry headers, bodyless Content-Type and shared live X-Auth-Token keep native behavior; this is explicit Go policy, not evidence of arbitrary Python Session parity.

### `python:network/v2/get_tags`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags.API.GetTags`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags.WithSourceSecurityGroupRuleTags`

- One GET to kind/id/tags decodes a flat object. Missing tags retains prior cached/default empty list, present empty and null remain distinct, unknown fields stay in actual Body; malformed/nonobject/non-string-list replies return actual accepted-response receipt without replacing cache.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsFetchPresenceNullAndDecodeEvidence`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsNoContentFetchAndMutationBodies`.
- SDK-owned ResourceKind routes cover nine stock-compatible classes plus explicit Source-only SecurityGroupRule capability without caller builders, arbitrary resource URLs, parent GETs or name lookup. QoS tag parent is stock unprefixed policies; API reference and cached tag slices remain immutable.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsKindsRoutesAndBulkCapabilities`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsLiteralSegmentsExtensionsAndSourceHeaders`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsCacheTransitionsDoNotInventResponseState`.
- Captured selected Neutron Source and concrete options guard source/client/route/context before callbacks and physical calls. Unknown kind, invalid literal ID/tag, nil option, reserved tags Body, forbidden auth/framing Header, query/secondary arguments and body on bodyless calls fail before HTTP. Caller input slices/options/extension JSON are independent snapshots.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsPreflightAndSourceCapture`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsCallerSlicesAndRetainedOptionsAreSnapshots`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsLiteralSegmentsExtensionsAndSourceHeaders`.
- Actual 200..399 success policy preserves response Body/Header/StatusCode separately from input-derived cache. Strict rejected404 and other errors keep native status/body/header and callback causes. Accepted/rejected physical Read/Close, context custom cause and sticky source faults are retained; clean retries preserve request serialization and tampering stops before resend.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsStrictMissingAndAcceptedCodes`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsPhysicalFaultsRetainAcceptedAndRejectedProof`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsRetryOwnershipCleanRetryAndDeniedCause`.
- Pinned Python declares SecurityGroupRule tag capabilities while stock Neutron27.0.0 tag registry lacks that kind. All seven default SGR calls fail ErrUnsupported before HTTP; SDK-owned WithSourceSecurityGroupRuleTags explicitly permits the fixed Source route on a backend providing it. These fixtures establish Source compatibility only, not stock27 endpoint/authorization acceptance. The option is rejected on other resource kinds.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsSecurityGroupRuleSourceCapabilityIsExplicit`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsKindsRoutesAndBulkCapabilities`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsPreflightAndSourceCapture`.
- Library-owned HeaderPolicy fixes selected generic network microversion value or absence and rejects incompatible Nova/Cinder version headers plus reserved auth/framing header injection through native retry. Bulk JSON Content-Type retains selected service/option value or application/json; owned value deletion and OmitHeaders suppression fail before resend while retaining the original503 proof and callback causes. Ordinary retry headers and bodyless Content-Type remain native; shared live X-Auth-Token is refreshed rather than frozen.
  Tests: `network/v2/extensions/attributestags/python_tags_policy_test.go:TestPythonTagsRetryCannotChangeSelectedOrAbsentVersionHeaders`, `network/v2/extensions/attributestags/python_tags_policy_test.go:TestPythonTagsRetryCannotAddReservedHeaders`, `network/v2/extensions/attributestags/python_tags_policy_test.go:TestPythonTagsRetryOwnsBulkJSONContentTypeAndOmissions`, `network/v2/extensions/attributestags/python_tags_policy_test.go:TestPythonTagsOrdinaryRetryHeadersAndLiveTokenKeepNativePolicy`.

Python/Go differences:

- Go returns explicit context/error and immutable concrete TaggedResource/TagResult instead of mutating arbitrary Python Resource subclasses. Fresh cache is [], explicit cached/fetched null remains nil; actual response evidence is separate from local cache.
- Go owns ten fixed resource kinds and literal escaped ID/tag segments. Empty/dot/control/non-UTF-8 individual segments are rejected before HTTP. Dynamic user subclasses/extension registry are separate shared SDK work, not proof of completed full Resource parity.
- Source QoSPolicy.set_tags overrides /policies and omits explicit raise_from_response; its inherited other TagMixin methods use /qos/policies. Go consistently uses stock 27.0.0 tag-controller policies path and checks actual errors for all methods.
- Go Header options and bulk Body extension options are explicit additions with SDK-owned auth/framing/core-field restrictions. They do not invent Python inventory operations or change server authorization.
- Go requires a flat JSON object and tags string array or null; Python descriptor may convert arbitrary scalar/list elements. Malformed/incompatible replies preserve actual accepted receipt rather than claiming valid Resource state.
- Source SecurityGroupRule tagging is explicit compatibility-only via WithSourceSecurityGroupRuleTags. Default stock27 capability is preflight ErrUnsupported; backend support and authenticated acceptance are not inferred from a fixture.
- The owned tag lane accepts final HTTP200..399; custom-transport final1xx status is not covered by this bounded named mapping. Pinned exceptions.py generic status policy accepts all statuses below400.
- Pinned Source Proxy.request supports optional Connection HTTP response caching: GET tag reads/checks can return cached responses when caching/expiration is configured, while writes invalidate the cache prefix. Go deliberately performs transport GETs and has only immutable local tag state; shared response-cache hits/invalidation are not implemented.
- Go owns operation protocol/auth/framing constraints across native retry. Selected network version and bulk JSON Content-Type values or required absence cannot be changed or hidden with OmitHeaders. Ordinary retry headers, bodyless Content-Type and shared live X-Auth-Token keep native behavior; this is explicit Go policy, not evidence of arbitrary Python Session parity.

### `python:network/v2/remove_all_tags`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags.API.RemoveAllTags`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags.WithSourceSecurityGroupRuleTags`

- One strict bodyless collection DELETE returns copied resource with empty cache only after accepted response; HTTP404 and retry callback failure do not become successful deletion. No synthetic server tags or parent query are created.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsCacheTransitionsDoNotInventResponseState`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsStrictMissingAndAcceptedCodes`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsRetryOwnershipCleanRetryAndDeniedCause`.
- SDK-owned ResourceKind routes cover nine stock-compatible classes plus explicit Source-only SecurityGroupRule capability without caller builders, arbitrary resource URLs, parent GETs or name lookup. QoS tag parent is stock unprefixed policies; API reference and cached tag slices remain immutable.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsKindsRoutesAndBulkCapabilities`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsLiteralSegmentsExtensionsAndSourceHeaders`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsCacheTransitionsDoNotInventResponseState`.
- Captured selected Neutron Source and concrete options guard source/client/route/context before callbacks and physical calls. Unknown kind, invalid literal ID/tag, nil option, reserved tags Body, forbidden auth/framing Header, query/secondary arguments and body on bodyless calls fail before HTTP. Caller input slices/options/extension JSON are independent snapshots.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsPreflightAndSourceCapture`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsCallerSlicesAndRetainedOptionsAreSnapshots`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsLiteralSegmentsExtensionsAndSourceHeaders`.
- Actual 200..399 success policy preserves response Body/Header/StatusCode separately from input-derived cache. Strict rejected404 and other errors keep native status/body/header and callback causes. Accepted/rejected physical Read/Close, context custom cause and sticky source faults are retained; clean retries preserve request serialization and tampering stops before resend.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsStrictMissingAndAcceptedCodes`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsPhysicalFaultsRetainAcceptedAndRejectedProof`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsRetryOwnershipCleanRetryAndDeniedCause`.
- Pinned Python declares SecurityGroupRule tag capabilities while stock Neutron27.0.0 tag registry lacks that kind. All seven default SGR calls fail ErrUnsupported before HTTP; SDK-owned WithSourceSecurityGroupRuleTags explicitly permits the fixed Source route on a backend providing it. These fixtures establish Source compatibility only, not stock27 endpoint/authorization acceptance. The option is rejected on other resource kinds.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsSecurityGroupRuleSourceCapabilityIsExplicit`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsKindsRoutesAndBulkCapabilities`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsPreflightAndSourceCapture`.
- Library-owned HeaderPolicy fixes selected generic network microversion value or absence and rejects incompatible Nova/Cinder version headers plus reserved auth/framing header injection through native retry. Bulk JSON Content-Type retains selected service/option value or application/json; owned value deletion and OmitHeaders suppression fail before resend while retaining the original503 proof and callback causes. Ordinary retry headers and bodyless Content-Type remain native; shared live X-Auth-Token is refreshed rather than frozen.
  Tests: `network/v2/extensions/attributestags/python_tags_policy_test.go:TestPythonTagsRetryCannotChangeSelectedOrAbsentVersionHeaders`, `network/v2/extensions/attributestags/python_tags_policy_test.go:TestPythonTagsRetryCannotAddReservedHeaders`, `network/v2/extensions/attributestags/python_tags_policy_test.go:TestPythonTagsRetryOwnsBulkJSONContentTypeAndOmissions`, `network/v2/extensions/attributestags/python_tags_policy_test.go:TestPythonTagsOrdinaryRetryHeadersAndLiveTokenKeepNativePolicy`.

Python/Go differences:

- Go returns explicit context/error and immutable concrete TaggedResource/TagResult instead of mutating arbitrary Python Resource subclasses. Fresh cache is [], explicit cached/fetched null remains nil; actual response evidence is separate from local cache.
- Go owns ten fixed resource kinds and literal escaped ID/tag segments. Empty/dot/control/non-UTF-8 individual segments are rejected before HTTP. Dynamic user subclasses/extension registry are separate shared SDK work, not proof of completed full Resource parity.
- Source QoSPolicy.set_tags overrides /policies and omits explicit raise_from_response; its inherited other TagMixin methods use /qos/policies. Go consistently uses stock 27.0.0 tag-controller policies path and checks actual errors for all methods.
- Go Header options and bulk Body extension options are explicit additions with SDK-owned auth/framing/core-field restrictions. They do not invent Python inventory operations or change server authorization.
- Source SecurityGroupRule tagging is explicit compatibility-only via WithSourceSecurityGroupRuleTags. Default stock27 capability is preflight ErrUnsupported; backend support and authenticated acceptance are not inferred from a fixture.
- The owned tag lane accepts final HTTP200..399; custom-transport final1xx status is not covered by this bounded named mapping. Pinned exceptions.py generic status policy accepts all statuses below400.
- Pinned Source Proxy.request supports optional Connection HTTP response caching: GET tag reads/checks can return cached responses when caching/expiration is configured, while writes invalidate the cache prefix. Go deliberately performs transport GETs and has only immutable local tag state; shared response-cache hits/invalidation are not implemented.
- Go owns operation protocol/auth/framing constraints across native retry. Selected network version and bulk JSON Content-Type values or required absence cannot be changed or hidden with OmitHeaders. Ordinary retry headers, bodyless Content-Type and shared live X-Auth-Token keep native behavior; this is explicit Go policy, not evidence of arbitrary Python Session parity.

### `python:network/v2/remove_tag`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags.API.RemoveTag`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags.WithSourceSecurityGroupRuleTags`

- One bodyless strict individual DELETE removes only first cached occurrence; absent cached tag still sends the DELETE and is locally harmless. HTTP404 remains error. Cached null after accepted action retains actual acknowledgement in local-transition failure.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsCacheTransitionsDoNotInventResponseState`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsNullCacheFailureKeepsAcknowledgement`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsStrictMissingAndAcceptedCodes`.
- SDK-owned ResourceKind routes cover nine stock-compatible classes plus explicit Source-only SecurityGroupRule capability without caller builders, arbitrary resource URLs, parent GETs or name lookup. QoS tag parent is stock unprefixed policies; API reference and cached tag slices remain immutable.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsKindsRoutesAndBulkCapabilities`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsLiteralSegmentsExtensionsAndSourceHeaders`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsCacheTransitionsDoNotInventResponseState`.
- Captured selected Neutron Source and concrete options guard source/client/route/context before callbacks and physical calls. Unknown kind, invalid literal ID/tag, nil option, reserved tags Body, forbidden auth/framing Header, query/secondary arguments and body on bodyless calls fail before HTTP. Caller input slices/options/extension JSON are independent snapshots.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsPreflightAndSourceCapture`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsCallerSlicesAndRetainedOptionsAreSnapshots`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsLiteralSegmentsExtensionsAndSourceHeaders`.
- Actual 200..399 success policy preserves response Body/Header/StatusCode separately from input-derived cache. Strict rejected404 and other errors keep native status/body/header and callback causes. Accepted/rejected physical Read/Close, context custom cause and sticky source faults are retained; clean retries preserve request serialization and tampering stops before resend.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsStrictMissingAndAcceptedCodes`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsPhysicalFaultsRetainAcceptedAndRejectedProof`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsRetryOwnershipCleanRetryAndDeniedCause`.
- Pinned Python declares SecurityGroupRule tag capabilities while stock Neutron27.0.0 tag registry lacks that kind. All seven default SGR calls fail ErrUnsupported before HTTP; SDK-owned WithSourceSecurityGroupRuleTags explicitly permits the fixed Source route on a backend providing it. These fixtures establish Source compatibility only, not stock27 endpoint/authorization acceptance. The option is rejected on other resource kinds.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsSecurityGroupRuleSourceCapabilityIsExplicit`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsKindsRoutesAndBulkCapabilities`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsPreflightAndSourceCapture`.
- Library-owned HeaderPolicy fixes selected generic network microversion value or absence and rejects incompatible Nova/Cinder version headers plus reserved auth/framing header injection through native retry. Bulk JSON Content-Type retains selected service/option value or application/json; owned value deletion and OmitHeaders suppression fail before resend while retaining the original503 proof and callback causes. Ordinary retry headers and bodyless Content-Type remain native; shared live X-Auth-Token is refreshed rather than frozen.
  Tests: `network/v2/extensions/attributestags/python_tags_policy_test.go:TestPythonTagsRetryCannotChangeSelectedOrAbsentVersionHeaders`, `network/v2/extensions/attributestags/python_tags_policy_test.go:TestPythonTagsRetryCannotAddReservedHeaders`, `network/v2/extensions/attributestags/python_tags_policy_test.go:TestPythonTagsRetryOwnsBulkJSONContentTypeAndOmissions`, `network/v2/extensions/attributestags/python_tags_policy_test.go:TestPythonTagsOrdinaryRetryHeadersAndLiveTokenKeepNativePolicy`.

Python/Go differences:

- Go returns explicit context/error and immutable concrete TaggedResource/TagResult instead of mutating arbitrary Python Resource subclasses. Fresh cache is [], explicit cached/fetched null remains nil; actual response evidence is separate from local cache.
- Go owns ten fixed resource kinds and literal escaped ID/tag segments. Empty/dot/control/non-UTF-8 individual segments are rejected before HTTP. Dynamic user subclasses/extension registry are separate shared SDK work, not proof of completed full Resource parity.
- Source QoSPolicy.set_tags overrides /policies and omits explicit raise_from_response; its inherited other TagMixin methods use /qos/policies. Go consistently uses stock 27.0.0 tag-controller policies path and checks actual errors for all methods.
- Go Header options and bulk Body extension options are explicit additions with SDK-owned auth/framing/core-field restrictions. They do not invent Python inventory operations or change server authorization.
- Source SecurityGroupRule tagging is explicit compatibility-only via WithSourceSecurityGroupRuleTags. Default stock27 capability is preflight ErrUnsupported; backend support and authenticated acceptance are not inferred from a fixture.
- The owned tag lane accepts final HTTP200..399; custom-transport final1xx status is not covered by this bounded named mapping. Pinned exceptions.py generic status policy accepts all statuses below400.
- Pinned Source Proxy.request supports optional Connection HTTP response caching: GET tag reads/checks can return cached responses when caching/expiration is configured, while writes invalidate the cache prefix. Go deliberately performs transport GETs and has only immutable local tag state; shared response-cache hits/invalidation are not implemented.
- Go owns operation protocol/auth/framing constraints across native retry. Selected network version and bulk JSON Content-Type values or required absence cannot be changed or hidden with OmitHeaders. Ordinary retry headers, bodyless Content-Type and shared live X-Auth-Token keep native behavior; this is explicit Go policy, not evidence of arbitrary Python Session parity.

### `python:network/v2/set_tags`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags.API.SetTags`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags.WithSourceSecurityGroupRuleTags`

- One unwrapped PUT tags body replaces remote tags. Typed nil sends JSON null, empty nonnil slice sends [], duplicate/empty values are not prevalidated as server domain rules. Source-style returned cache records caller list independently from server Body and ObservedTags remains false, including accepted204 without JSON.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsCacheTransitionsDoNotInventResponseState`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsLiteralSegmentsExtensionsAndSourceHeaders`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsNoContentFetchAndMutationBodies`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsCallerSlicesAndRetainedOptionsAreSnapshots`.
- SDK-owned ResourceKind routes cover nine stock-compatible classes plus explicit Source-only SecurityGroupRule capability without caller builders, arbitrary resource URLs, parent GETs or name lookup. QoS tag parent is stock unprefixed policies; API reference and cached tag slices remain immutable.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsKindsRoutesAndBulkCapabilities`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsLiteralSegmentsExtensionsAndSourceHeaders`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsCacheTransitionsDoNotInventResponseState`.
- Captured selected Neutron Source and concrete options guard source/client/route/context before callbacks and physical calls. Unknown kind, invalid literal ID/tag, nil option, reserved tags Body, forbidden auth/framing Header, query/secondary arguments and body on bodyless calls fail before HTTP. Caller input slices/options/extension JSON are independent snapshots.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsPreflightAndSourceCapture`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsCallerSlicesAndRetainedOptionsAreSnapshots`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsLiteralSegmentsExtensionsAndSourceHeaders`.
- Actual 200..399 success policy preserves response Body/Header/StatusCode separately from input-derived cache. Strict rejected404 and other errors keep native status/body/header and callback causes. Accepted/rejected physical Read/Close, context custom cause and sticky source faults are retained; clean retries preserve request serialization and tampering stops before resend.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsStrictMissingAndAcceptedCodes`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsPhysicalFaultsRetainAcceptedAndRejectedProof`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsRetryOwnershipCleanRetryAndDeniedCause`.
- Pinned Python declares SecurityGroupRule tag capabilities while stock Neutron27.0.0 tag registry lacks that kind. All seven default SGR calls fail ErrUnsupported before HTTP; SDK-owned WithSourceSecurityGroupRuleTags explicitly permits the fixed Source route on a backend providing it. These fixtures establish Source compatibility only, not stock27 endpoint/authorization acceptance. The option is rejected on other resource kinds.
  Tests: `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsSecurityGroupRuleSourceCapabilityIsExplicit`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsKindsRoutesAndBulkCapabilities`, `network/v2/extensions/attributestags/python_tags_contracts_test.go:TestPythonTagsPreflightAndSourceCapture`.
- Library-owned HeaderPolicy fixes selected generic network microversion value or absence and rejects incompatible Nova/Cinder version headers plus reserved auth/framing header injection through native retry. Bulk JSON Content-Type retains selected service/option value or application/json; owned value deletion and OmitHeaders suppression fail before resend while retaining the original503 proof and callback causes. Ordinary retry headers and bodyless Content-Type remain native; shared live X-Auth-Token is refreshed rather than frozen.
  Tests: `network/v2/extensions/attributestags/python_tags_policy_test.go:TestPythonTagsRetryCannotChangeSelectedOrAbsentVersionHeaders`, `network/v2/extensions/attributestags/python_tags_policy_test.go:TestPythonTagsRetryCannotAddReservedHeaders`, `network/v2/extensions/attributestags/python_tags_policy_test.go:TestPythonTagsRetryOwnsBulkJSONContentTypeAndOmissions`, `network/v2/extensions/attributestags/python_tags_policy_test.go:TestPythonTagsOrdinaryRetryHeadersAndLiveTokenKeepNativePolicy`.

Python/Go differences:

- Go returns explicit context/error and immutable concrete TaggedResource/TagResult instead of mutating arbitrary Python Resource subclasses. Fresh cache is [], explicit cached/fetched null remains nil; actual response evidence is separate from local cache.
- Go owns ten fixed resource kinds and literal escaped ID/tag segments. Empty/dot/control/non-UTF-8 individual segments are rejected before HTTP. Dynamic user subclasses/extension registry are separate shared SDK work, not proof of completed full Resource parity.
- Source QoSPolicy.set_tags overrides /policies and omits explicit raise_from_response; its inherited other TagMixin methods use /qos/policies. Go consistently uses stock 27.0.0 tag-controller policies path and checks actual errors for all methods.
- Go Header options and bulk Body extension options are explicit additions with SDK-owned auth/framing/core-field restrictions. They do not invent Python inventory operations or change server authorization.
- Source SecurityGroupRule tagging is explicit compatibility-only via WithSourceSecurityGroupRuleTags. Default stock27 capability is preflight ErrUnsupported; backend support and authenticated acceptance are not inferred from a fixture.
- The owned tag lane accepts final HTTP200..399; custom-transport final1xx status is not covered by this bounded named mapping. Pinned exceptions.py generic status policy accepts all statuses below400.
- Pinned Source Proxy.request supports optional Connection HTTP response caching: GET tag reads/checks can return cached responses when caching/expiration is configured, while writes invalidate the cache prefix. Go deliberately performs transport GETs and has only immutable local tag state; shared response-cache hits/invalidation are not implemented.
- Go owns operation protocol/auth/framing constraints across native retry. Selected network version and bulk JSON Content-Type values or required absence cannot be changed or hidden with OmitHeaders. Ordinary retry headers, bodyless Content-Type and shared live X-Auth-Token keep native behavior; this is explicit Go policy, not evidence of arbitrary Python Session parity.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack
from openstack.network.v2.network import Network

conn = openstack.connect(cloud="dev")
network = Network(id="NETWORK_ID")
print(conn.network.get_tags(network))

conn.network.set_tags(network, ["team-a", "production"])
conn.network.add_tags(network, ["reviewed"])
conn.network.add_tag(network, "managed")
conn.network.check_tag(network, "managed")
conn.network.remove_tag(network, "managed")
conn.network.remove_all_tags(network)
```

```sh
go run . -cloud dev -kind network -id NETWORK_ID
go run . -cloud dev -kind network -id NETWORK_ID -operation set -tags team-a,production
go run . -cloud dev -kind network -id NETWORK_ID -operation check -tag managed
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "strings"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    kind := flag.String("kind", "network", "network, port, subnet, subnet-pool, router, floating-ip, security-group, security-group-rule, qos-policy, trunk")
    id := flag.String("id", "", "literal resource ID")
    operation := flag.String("operation", "get", "get, set, add-many, add, remove, clear, check")
    list := flag.String("tags", "", "comma-separated tag list; empty clears with set")
    tag := flag.String("tag", "", "individual tag")
    sourceRuleTags := flag.Bool("source-rule-tags", false, "enable Source SGR tag compatibility on a backend that supports it")
    flag.Parse()
    if *id == "" { log.Fatal("-id is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    target := attributestags.NewTaggedResource(attributestags.ResourceKind(*kind), *id)
    tags := []string{}
    if *list != "" { tags = strings.Split(*list, ",") }
    options := []attributestags.TagOption{}
    if *sourceRuleTags { options = append(options, attributestags.WithSourceSecurityGroupRuleTags()) }
    var result *attributestags.TagResult
    switch *operation {
    case "get": result, err = service.AttributeTags.GetTags(ctx, target, options...)
    case "set": result, err = service.AttributeTags.SetTags(ctx, target, tags, options...)
    case "add-many": result, err = service.AttributeTags.AddTags(ctx, target, tags, options...)
    case "add": result, err = service.AttributeTags.AddTag(ctx, target, *tag, options...)
    case "remove": result, err = service.AttributeTags.RemoveTag(ctx, target, *tag, options...)
    case "clear": result, err = service.AttributeTags.RemoveAllTags(ctx, target, options...)
    case "check": result, err = service.AttributeTags.CheckTag(ctx, target, *tag, options...)
    default: log.Fatalf("unknown -operation %q", *operation)
    }
    if err != nil { log.Fatal(err) }
    view := map[string]any{
        "kind": result.Resource.Kind(), "id": result.Resource.ID(),
        "tags": result.Resource.Tags(), "observed_tags": result.ObservedTags,
        "status_code": result.StatusCode,
    }
    data, err := json.MarshalIndent(view, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

## References

- [Neutron 27.0.0 tagging router](https://github.com/openstack/neutron/blob/27.0.0/neutron/extensions/tagging.py)
- [extension middleware](https://github.com/openstack/neutron/blob/27.0.0/neutron/api/extensions.py)
- [Network](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/network.py)
- [QoS](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py)
- [Install](../../docs/install.md)
