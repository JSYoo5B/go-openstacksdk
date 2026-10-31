# Subscriptions

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:message/v2/create_subscription`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.New`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.API.InQueue`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.QueueScope.Create`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.WithCreateSubscriber`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.WithCreateTTL`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.WithCreateDeliveryOptions`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.WithCreateField`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.WithCreateHeader`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.WithCreateClientID`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.WithCreateProjectID`

- Pinned _proxy.py:191-205 and Subscription.create_opts(request_key=None) map to one POST queues/{fixed queue}/subscriptions with flat subscriber/optional ttl/options JSON, accepting 201 and an actual root object. subscription_id is an optional response field: omitted/null/empty values return the decoded raw model with no ID inferred from Location or request inputs. Omitted ttl remains omitted rather than submitting the documented server default 3600. No Queue GET, confirmation or implicit fetch is performed; collection-valued or foreign Location remains passive response evidence rather than an ID/action/request target.
  Tests: `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsFourRoutesFixedQueueAndRawModels`, `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsCreateSnapshotsExactMetadataAndIdentity`, `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsMutationAndReadEvidenceNeverResend`.
- Owned CreateOptions/metadata/extension/header snapshots preserve false/null/precise nested numbers, reusable concurrent options and last-wins helpers. Unknown extension fields cannot overwrite concrete fields, response identities or fixed queue metadata. Null TTL, non-object delivery options and unsupported query/argument/auth headers fail before HTTP.
  Tests: `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsCreateSnapshotsExactMetadataAndIdentity`, `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsInvalidOptionsAndMissingClientIdentity`.
- Unexpected status retains native errors. Malformed/null/array accepted root, wrong-type canonical typed response fields and accepted body-read failures retain actual ResponseError body/header/status/cause; accepted decode/read failures are not resent. Optional response identity absence, null or empty is not itself a decode failure.
  Tests: `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsMutationAndReadEvidenceNeverResend`, `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsListTerminalErrorsAndSourceCancellation`.
- Concrete request client/project identities and extra headers are copied into request-local ServiceClient headers. Source configured Client-ID and other headers remain unchanged; typed overrides apply to this operation only and SDK-owned auth/body/identity header extensions are rejected. Different values under case-variant custom request-header keys are rejected before HTTP; normal canonical WithHeader helpers remain last-wins. The original ProviderClient remains live for authentication and middleware.
  Tests: `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsTypedRequestIdentityAllOperations`, `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsCreateSnapshotsExactMetadataAndIdentity`, `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsInvalidOptionsAndMissingClientIdentity`, `connectiontest/connection_scoped_sdk_owned_test.go:TestConnectionSDKOwnedSubscriptionAndConsumerScopes`.
- The SDK-owned Messaging.Subscriptions facade shares the original authenticated message ServiceClient/Provider with native Queues/Messages/Claims. InQueue fixes a validated escaped queue segment with zero HTTP, preserves selected ResourceBase/live provider token, and never performs Queue lookup or follows response locations. Invalid/nil source, unsafe scope/child identities and canceled context fail before HTTP; the registry declares queue_subscription rather than fabricated native CRUD/Find/Wait capabilities.
  Tests: `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsFourRoutesFixedQueueAndRawModels`, `connectiontest/connection_scoped_sdk_owned_test.go:TestConnectionSDKOwnedSubscriptionAndConsumerScopes`, `internal/cmd/sdkgen/messaging_services_test.go:TestMessagingSDKOwnedRegistryAndQueueScopeCapabilities`.

Python/Go differences:

- Python Subscription age/ttl/subscriber/source/subscription_id descriptors are untyped and options uses dict conversion. Go creation deliberately uses optional int64 TTL with null rejection, a required string subscriber and object-only raw options; the server still evaluates subscriber scheme and TTL range. Response subscription_id is optional and does not have a fabricated nonempty/safe-segment requirement: exact canonical strings/null/omission remain response evidence; wrong-type canonical typed fields still receive deliberate Go strict decode errors.
- Go returns only the decoded accepted response with owned exact-key Body/Header/StatusCode, not the Python request-seeded Subscription with descriptor/header/URI state. Canonical literal id presence (including null) overrides alternate subscription_id; case variants remain raw extensions. Location is passive and is not used to synthesize an ID.
- Full inherited Subscription/MessageResource/Resource descriptor, coercion/default, seeded mutable instance, alternate-ID mutation and returned Resource lifecycle/cache behavior are not implemented by caller-owned Go values and explicit string IDs. Python str-or-Subscription input overloads and generic session/adapter/base_path/skip_cache controls remain separate.
- Pinned MessageResource generates a UUID for an unconfigured logical request/iterator and derives X-PROJECT-ID from the supplied resource/params or session project. Go reuses configured Connection/source Client-ID (or an explicit typed override), requires an effective Client-ID and does not infer an auth-result project; configured or typed project headers supply that policy. Shared client mutation is not part of the Go API.
- Direct **attrs still permits broader Python descriptor/coercion/unknown-body and inherited _create base_path/__conflicting_attrs behavior. Concrete Go body/field protection intentionally narrows that surface; passing source fields age/source/id through creation is not a new supported overload.
- Resource objects, Python str-or-Subscription overloads, **attrs coercion, per-request random Client-ID and cache controls are deliberately not reproduced; Go uses explicit string IDs, concrete options and a stable configured Client-ID (or a typed override).

### `python:message/v2/get_subscription`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.New`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.API.InQueue`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.QueueScope.Get`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.WithGetOptions`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.WithGetHeader`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.WithGetClientID`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.WithGetProjectID`

- One GET queues/{fixed queue}/subscriptions/{escaped literal identity} accepts 200 and a flat object. Get remains strict for native missing/error status and does not use name lookup, alternate Location, response source/ID or an incidental queue name to retarget the request.
  Tests: `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsFourRoutesFixedQueueAndRawModels`, `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsMutationAndReadEvidenceNeverResend`.
- Exact canonical response fields populate owned typed/raw convenience values: present literal id including null wins over subscription_id; incompatible case variants remain Body extensions. Raw ttl/age/options preserve omission/null/precise nested numbers independently of Body. Unexpected success codes, accepted decode/read failures and canceled/invalid source remain terminal with actual evidence and no implicit follow-up.
  Tests: `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsFourRoutesFixedQueueAndRawModels`, `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsMutationAndReadEvidenceNeverResend`, `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsListTerminalErrorsAndSourceCancellation`.
- Concrete request client/project identities and extra headers are copied into request-local ServiceClient headers. Source configured Client-ID and other headers remain unchanged; typed overrides apply to this operation only and SDK-owned auth/body/identity header extensions are rejected. Different values under case-variant custom request-header keys are rejected before HTTP; normal canonical WithHeader helpers remain last-wins. The original ProviderClient remains live for authentication and middleware.
  Tests: `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsTypedRequestIdentityAllOperations`, `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsCreateSnapshotsExactMetadataAndIdentity`, `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsInvalidOptionsAndMissingClientIdentity`, `connectiontest/connection_scoped_sdk_owned_test.go:TestConnectionSDKOwnedSubscriptionAndConsumerScopes`.
- The SDK-owned Messaging.Subscriptions facade shares the original authenticated message ServiceClient/Provider with native Queues/Messages/Claims. InQueue fixes a validated escaped queue segment with zero HTTP, preserves selected ResourceBase/live provider token, and never performs Queue lookup or follows response locations. Invalid/nil source, unsafe scope/child identities and canceled context fail before HTTP; the registry declares queue_subscription rather than fabricated native CRUD/Find/Wait capabilities.
  Tests: `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsFourRoutesFixedQueueAndRawModels`, `connectiontest/connection_scoped_sdk_owned_test.go:TestConnectionSDKOwnedSubscriptionAndConsumerScopes`, `internal/cmd/sdkgen/messaging_services_test.go:TestMessagingSDKOwnedRegistryAndQueueScopeCapabilities`.

Python/Go differences:

- Go Get accepts a validated literal ID string and returns an owned exact-key projection; it does not mutate a seeded Python Subscription. Canonical subscriber/source/subscription_id/id values are stricter typed strings, while ttl/age/options and full raw Body remain precise/untyped response data.
- Python MessageResource.fetch accepts skip_cache/base_path and supplied Resource header descriptors, deriving missing request IDs/project from UUID/session. Go exposes concrete per-call identity/header options on its fixed queue scope with the stable configured Client-ID policy.
- Full inherited Subscription/MessageResource/Resource descriptor, coercion/default, seeded mutable instance, alternate-ID mutation and returned Resource lifecycle/cache behavior are not implemented by caller-owned Go values and explicit string IDs. Python str-or-Subscription input overloads and generic session/adapter/base_path/skip_cache controls remain separate.
- Pinned MessageResource generates a UUID for an unconfigured logical request/iterator and derives X-PROJECT-ID from the supplied resource/params or session project. Go reuses configured Connection/source Client-ID (or an explicit typed override), requires an effective Client-ID and does not infer an auth-result project; configured or typed project headers supply that policy. Shared client mutation is not part of the Go API.
- Resource objects, Python str-or-Subscription overloads, **attrs coercion, per-request random Client-ID and cache controls are deliberately not reproduced; Go uses explicit string IDs, concrete options and a stable configured Client-ID (or a typed override).

### `python:message/v2/subscriptions`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.New`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.API.InQueue`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.QueueScope.List`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.QueueScope.All`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.WithListHeader`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.WithListClientID`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.WithListProjectID`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.WithListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.WithListPaginated`

- Pinned _proxy.py:207-229 fixes queue_name before MessageResource.list; Go scopes it in the URI and rejects route/control query injection. List is lazy, expects 200 with a subscriptions array and copies original caller queries/options/headers per iteration. Explicit positive limit/marker are submitted; the selected fixed endpoint and live ProviderClient are retained across pages.
  Tests: `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsPythonMarkerPagingIgnoresLinksAndKeepsSource`, `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsInvalidOptionsAndMissingClientIdentity`, `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsFourRoutesFixedQueueAndRawModels`.
- The custom pinned MessageResource.list algorithm is implemented rather than generic Resource.list: advertised links/next are ignored; after each nonempty full page, even with no initial limit, next query receives limit=decoded page count and marker=last canonical Resource-style ID. A page shorter than the active limit or an empty page stops. Paginated=false stops after the first page. Missing/null terminal marker and repeated marker receive deliberate Go accepted-evidence/cycle guards.
  Tests: `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsPythonMarkerPagingIgnoresLinksAndKeepsSource`, `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsListControlsLazyDecodeAndReuse`, `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsListTerminalErrorsAndSourceCancellation`.
- Go MaxItems is a local raw decoded-row cap with zero unbounded, negative preflight failure and no wire key/limit hint. It stops before unused row typed decode/marker continuation; consumer break stops without another request. Whole-page JSON syntax still must decode before any rows. Reused iterators own independent counters; late HTTP/decode/cycle/source/context errors invalidate All instead of returning partial success, with actual accepted page evidence where available.
  Tests: `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsListControlsLazyDecodeAndReuse`, `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsListTerminalErrorsAndSourceCancellation`.
- Concrete request client/project identities and extra headers are copied into request-local ServiceClient headers. Source configured Client-ID and other headers remain unchanged; typed overrides apply to this operation only and SDK-owned auth/body/identity header extensions are rejected. Different values under case-variant custom request-header keys are rejected before HTTP; normal canonical WithHeader helpers remain last-wins. The original ProviderClient remains live for authentication and middleware.
  Tests: `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsTypedRequestIdentityAllOperations`, `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsCreateSnapshotsExactMetadataAndIdentity`, `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsInvalidOptionsAndMissingClientIdentity`, `connectiontest/connection_scoped_sdk_owned_test.go:TestConnectionSDKOwnedSubscriptionAndConsumerScopes`.
- The SDK-owned Messaging.Subscriptions facade shares the original authenticated message ServiceClient/Provider with native Queues/Messages/Claims. InQueue fixes a validated escaped queue segment with zero HTTP, preserves selected ResourceBase/live provider token, and never performs Queue lookup or follows response locations. Invalid/nil source, unsafe scope/child identities and canceled context fail before HTTP; the registry declares queue_subscription rather than fabricated native CRUD/Find/Wait capabilities.
  Tests: `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsFourRoutesFixedQueueAndRawModels`, `connectiontest/connection_scoped_sdk_owned_test.go:TestConnectionSDKOwnedSubscriptionAndConsumerScopes`, `internal/cmd/sdkgen/messaging_services_test.go:TestMessagingSDKOwnedRegistryAndQueueScopeCapabilities`.

Python/Go differences:

- Pinned MessageResource.list ignores its headers/microversion/max_items parameters and does not perform generic Resource.list Body-local filter transposition. Go header options and effective local MaxItems are deliberate supported extensions; no generic Body filtering/semantic classification or class-specific automatic maximum-microversion negotiation is claimed.
- Go extension query forwards nonreserved unknown wire keys, including frozen repeated values; the pinned inherited QueryParameters transpose only retains limit/marker. Go rejects invalid/repeated/empty limit and invalid marker earlier and rejects malformed subscriptions envelopes, invalid consumed typed values, null last-ID and marker cycles with explicit errors.
- Python per-iterator UUID/session-project derivation and bare cls.existing(**row) objects differ from Go stable configured headers, caller-owned exact raw fields and actual page Header/StatusCode. Go queue scope remains authoritative and does not infer a URI overlay from returned source.
- Full inherited Subscription/MessageResource/Resource descriptor, coercion/default, seeded mutable instance, alternate-ID mutation and returned Resource lifecycle/cache behavior are not implemented by caller-owned Go values and explicit string IDs. Python str-or-Subscription input overloads and generic session/adapter/base_path/skip_cache controls remain separate.
- Pinned MessageResource generates a UUID for an unconfigured logical request/iterator and derives X-PROJECT-ID from the supplied resource/params or session project. Go reuses configured Connection/source Client-ID (or an explicit typed override), requires an effective Client-ID and does not infer an auth-result project; configured or typed project headers supply that policy. Shared client mutation is not part of the Go API.
- Inherited Proxy._list base_path/__conflicting_attrs/deprecated jmespath filtering and arbitrary Python adapter/session controls are not exposed. Ordinary typed options/query extensions do not implement these wrappers; custom MessageResource list quirks remain explicitly separated from generic Resource filtering/cache contracts.
- Resource objects, Python str-or-Subscription overloads, **attrs coercion, per-request random Client-ID and cache controls are deliberately not reproduced; Go uses explicit string IDs, concrete options and a stable configured Client-ID (or a typed override).

### `python:message/v2/delete_subscription`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.New`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.API.InQueue`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.QueueScope.Delete`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.WithDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.WithDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.WithDeleteHeader`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.WithDeleteClientID`
- `github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions.WithDeleteProjectID`

- Pinned _proxy.py:250-278 default ignore_missing=True maps to one DELETE queues/{fixed queue}/subscriptions/{escaped literal identity}, accepting 204 with no request JSON or implicit GET/Queue lookup. Go returns error-only; explicit false preserves actual native404, while default true suppresses only actual HTTP404. Other status errors remain terminal.
  Tests: `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsFourRoutesFixedQueueAndRawModels`, `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsMutationAndReadEvidenceNeverResend`.
- Wrapped transport404 and an accepted204 read failure whose cause contains native404 are never treated as logical absence. Original transport/context/native cause and accepted body/header/status are preserved with no SDK resend. Invalid source/unsafe identities/protected header or unsupported field/query/argument options fail before HTTP.
  Tests: `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsMutationAndReadEvidenceNeverResend`, `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsInvalidOptionsAndMissingClientIdentity`, `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsListTerminalErrorsAndSourceCancellation`.
- Concrete request client/project identities and extra headers are copied into request-local ServiceClient headers. Source configured Client-ID and other headers remain unchanged; typed overrides apply to this operation only and SDK-owned auth/body/identity header extensions are rejected. Different values under case-variant custom request-header keys are rejected before HTTP; normal canonical WithHeader helpers remain last-wins. The original ProviderClient remains live for authentication and middleware.
  Tests: `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsTypedRequestIdentityAllOperations`, `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsCreateSnapshotsExactMetadataAndIdentity`, `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsInvalidOptionsAndMissingClientIdentity`, `connectiontest/connection_scoped_sdk_owned_test.go:TestConnectionSDKOwnedSubscriptionAndConsumerScopes`.
- The SDK-owned Messaging.Subscriptions facade shares the original authenticated message ServiceClient/Provider with native Queues/Messages/Claims. InQueue fixes a validated escaped queue segment with zero HTTP, preserves selected ResourceBase/live provider token, and never performs Queue lookup or follows response locations. Invalid/nil source, unsafe scope/child identities and canceled context fail before HTTP; the registry declares queue_subscription rather than fabricated native CRUD/Find/Wait capabilities.
  Tests: `api/messaging_subscriptions_test.go:TestMessagingSubscriptionsFourRoutesFixedQueueAndRawModels`, `connectiontest/connection_scoped_sdk_owned_test.go:TestConnectionSDKOwnedSubscriptionAndConsumerScopes`, `internal/cmd/sdkgen/messaging_services_test.go:TestMessagingSDKOwnedRegistryAndQueueScopeCapabilities`.

Python/Go differences:

- The direct default ignore_missing=True is retained, but Go returns error-only rather than the mutable seeded Subscription/None result of Python _delete. A successful acknowledgement does not fabricate a returned Subscription or update cached model state.
- Literal child IDs are validated/escaped and header identity is selected by SDK-owned options; Python also accepts Subscription Resource inputs and broader supplied descriptor/session state.
- Full inherited Subscription/MessageResource/Resource descriptor, coercion/default, seeded mutable instance, alternate-ID mutation and returned Resource lifecycle/cache behavior are not implemented by caller-owned Go values and explicit string IDs. Python str-or-Subscription input overloads and generic session/adapter/base_path/skip_cache controls remain separate.
- Pinned MessageResource generates a UUID for an unconfigured logical request/iterator and derives X-PROJECT-ID from the supplied resource/params or session project. Go reuses configured Connection/source Client-ID (or an explicit typed override), requires an effective Client-ID and does not infer an auth-result project; configured or typed project headers supply that policy. Shared client mutation is not part of the Go API.
- Resource objects, Python str-or-Subscription overloads, **attrs coercion, per-request random Client-ID and cache controls are deliberately not reproduced; Go uses explicit string IDs, concrete options and a stable configured Client-ID (or a typed override).

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package subscriptionexample

import (
    "context"
    "fmt"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions"
)

func Subscribe(ctx context.Context, client *gophercloud.ServiceClient) error {
    queue, err := subscriptions.New(client).InQueue(ctx, "events")
    if err != nil { return err }
    created, err := queue.Create(ctx, subscriptions.CreateOpts{
        Subscriber: "mailto:example@example.test",
    }, subscriptions.WithCreateDeliveryOptions(map[string]any{
        "from": "Zaqar", "subject": "new event",
    }))
    if err != nil { return err }
    current, err := queue.Get(ctx, created.SubscriptionID)
    if err != nil { return err }
    fmt.Println(current.Subscriber, string(current.TTL))
    for item, err := range queue.List(ctx,
        subscriptions.WithListOptions(subscriptions.ListOpts{Limit: 10}),
        subscriptions.WithListMaxItems(5)) {
        if err != nil { return err }
        fmt.Println(item.ID)
    }
    return queue.Delete(ctx, created.SubscriptionID,
        subscriptions.WithDeleteIgnoreMissing(false))
}
```

## References

- [Message](https://docs.openstack.org/api-ref/message/#subscriptions-subscriptions)
- [Messaging Subscriptions Test](../../../api/messaging_subscriptions_test.go)
