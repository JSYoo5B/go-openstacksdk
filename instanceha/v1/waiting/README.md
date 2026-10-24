# Waiting

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:instance_ha/v1/wait_for_status`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/instanceha/v1/notifications.API.WaitForStatus`
- `github.com/JSYoo5B/go-openstacksdk/instanceha/v1/segments.API.WaitForStatus`
- `github.com/JSYoo5B/go-openstacksdk/instanceha/v1/hosts.SegmentScope.WaitForStatus`
- `github.com/JSYoo5B/go-openstacksdk/instanceha/v1/vmoves.NotificationScope.WaitForStatus`

- Four typed facades preserve fixed UUID routes, parent scopes and actual HTTP evidence. Notification and VMove use status, while Segment and Host require an explicit exported string attribute before status polling. Deletion observes404 or declared deleted status without submitting DELETE or following Location.
  Tests: `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitFourFacadesPreserveFixedRoutesAndHTTPBody`, `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitStatuslessModelsRejectDefaultBeforeHTTP`, `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitDeletionUsesStatusOrAbsenceWithoutDELETE`, `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitDeadlineDefaultsAndOverrides`, `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitErrorOnlyDefaultsAndFailureReplacement`, `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitRevalidatesVersionAndKeepsNativeAndDecodeFailures`.
- Instance-ha proxy status defaults are unlimited SDK time, ERROR-only failures and2-second intervals, with target precedence and caller replacement. Delete defaults to120seconds. Actual outgoing request deadlines prove parent/override policy while Resources.Wait keeps5minutes. VMove polling rechecks1.3 before every GET; callbacks and errors stop further requests.
  Tests: `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitFourFacadesPreserveFixedRoutesAndHTTPBody`, `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitStatuslessModelsRejectDefaultBeforeHTTP`, `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitDeletionUsesStatusOrAbsenceWithoutDELETE`, `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitDeadlineDefaultsAndOverrides`, `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitErrorOnlyDefaultsAndFailureReplacement`, `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitRevalidatesVersionAndKeepsNativeAndDecodeFailures`.
- Four facades preserve fixed routes; waits use real status, deadlines and configured failures.
  Tests: `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitFourFacadesPreserveFixedRoutesAndHTTPBody`, `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitDeadlineDefaultsAndOverrides`, `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitErrorOnlyDefaultsAndFailureReplacement`, `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitRevalidatesVersionAndKeepsNativeAndDecodeFailures`, `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitStatuslessModelsRejectDefaultBeforeHTTP`.

Python/Go differences:

- Explicit Ref returns fresh typed models rather than mutating a cached Python Resource; WaitForDelete returns error instead of the original Resource pointer. Segment/Host names use exact lookup once, while Notification/VMove name lookup is unsupported.
- Go uses notification_uuid/uuid route identities rather than database id, preserves fixed parent scope and never retargets polling from response IDs. Pinned inherited Python Resource identity behavior is documented separately.
- Go requires nonempty status, positive intervals and exported typed string attributes. Nullable Python targets/attributes, interval None/zero, arbitrary Resource/custom attrs and falsy successful fetches have distinct semantics.
- HTTP contexts carry deadlines and cancellation priority. Strict success-envelope decoding preserves ResponseError evidence and does not trigger an automatic resend; nested workflow progress is not synthesized as a top-level callback percentage.
- Python wait helpers take a Resource and may return a cached target status without fetching; Go takes a fixed resource.Ref, always polls through the typed getter, and returns a fresh model (an error only for deletion). Defaults are five minutes and two seconds with context deadlines.

### `python:instance_ha/v1/wait_for_delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/instanceha/v1/notifications.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/instanceha/v1/segments.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/instanceha/v1/hosts.SegmentScope.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/instanceha/v1/vmoves.NotificationScope.WaitForDelete`

- Four typed facades preserve fixed UUID routes, parent scopes and actual HTTP evidence. Notification and VMove use status, while Segment and Host require an explicit exported string attribute before status polling. Deletion observes404 or declared deleted status without submitting DELETE or following Location.
  Tests: `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitFourFacadesPreserveFixedRoutesAndHTTPBody`, `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitStatuslessModelsRejectDefaultBeforeHTTP`, `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitDeletionUsesStatusOrAbsenceWithoutDELETE`, `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitDeadlineDefaultsAndOverrides`, `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitErrorOnlyDefaultsAndFailureReplacement`, `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitRevalidatesVersionAndKeepsNativeAndDecodeFailures`.
- Instance-ha proxy status defaults are unlimited SDK time, ERROR-only failures and2-second intervals, with target precedence and caller replacement. Delete defaults to120seconds. Actual outgoing request deadlines prove parent/override policy while Resources.Wait keeps5minutes. VMove polling rechecks1.3 before every GET; callbacks and errors stop further requests.
  Tests: `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitFourFacadesPreserveFixedRoutesAndHTTPBody`, `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitStatuslessModelsRejectDefaultBeforeHTTP`, `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitDeletionUsesStatusOrAbsenceWithoutDELETE`, `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitDeadlineDefaultsAndOverrides`, `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitErrorOnlyDefaultsAndFailureReplacement`, `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitRevalidatesVersionAndKeepsNativeAndDecodeFailures`.
- Deletion waits use status or 404 absence without issuing DELETE, with deadlines and native failure causes.
  Tests: `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitDeletionUsesStatusOrAbsenceWithoutDELETE`, `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitDeadlineDefaultsAndOverrides`, `api/instanceha_wait_contracts_test.go:TestInstanceHAWaitRevalidatesVersionAndKeepsNativeAndDecodeFailures`.

Python/Go differences:

- Explicit Ref returns fresh typed models rather than mutating a cached Python Resource; WaitForDelete returns error instead of the original Resource pointer. Segment/Host names use exact lookup once, while Notification/VMove name lookup is unsupported.
- Go uses notification_uuid/uuid route identities rather than database id, preserves fixed parent scope and never retargets polling from response IDs. Pinned inherited Python Resource identity behavior is documented separately.
- Go requires nonempty status, positive intervals and exported typed string attributes. Nullable Python targets/attributes, interval None/zero, arbitrary Resource/custom attrs and falsy successful fetches have distinct semantics.
- HTTP contexts carry deadlines and cancellation priority. Strict success-envelope decoding preserves ResponseError evidence and does not trigger an automatic resend; nested workflow progress is not synthesized as a top-level callback percentage.
- Python wait helpers take a Resource and may return a cached target status without fetching; Go takes a fixed resource.Ref, always polls through the typed getter, and returns a fresh model (an error only for deletion). Defaults are five minutes and two seconds with context deadlines.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
notification = conn.instance_ha.get_notification("NOTIFICATION_UUID")
notification = conn.instance_ha.wait_for_status(
    notification, "finished", failures=["ERROR", "FAILED"], wait=180,
)
```

```go
service, err := conn.InstanceHAV1(ctx)
if err != nil { return err }
notification, err := service.Notifications.WaitForStatus(ctx,
    resource.ID("22222222-2222-4222-8222-222222222222"), "finished",
    resource.WithFailureStates("ERROR", "FAILED"),
    resource.WithTimeout(3*time.Minute))
if err != nil { return err }
fmt.Println(notification.UUID, notification.Status, notification.Header, notification.Body)
```

```go
service, err := conn.InstanceHAV1(ctx)
if err != nil { return err }
scope, err := service.VMoves.InNotification(ctx,
    resource.ID("22222222-2222-4222-8222-222222222222"))
if err != nil { return err }
move, err := scope.WaitForStatus(ctx,
    resource.ID("55555555-5555-4555-8555-555555555555"), "succeeded",
    resource.WithTimeout(5*time.Minute))
if err != nil { return err }
fmt.Println(move.NotificationID, move.UUID, move.Status)
```

```go
service, err := conn.InstanceHAV1(ctx)
if err != nil { return err }
ref := resource.ID("11111111-1111-4111-8111-111111111111")
if err := service.Segments.Delete(ctx, ref); err != nil { return err }
if err := service.Segments.WaitForDelete(ctx, ref,
    resource.WithTimeout(10*time.Minute),
    resource.WithPollInterval(time.Second)); err != nil { return err }
```

## References

- [Masakari proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/instance_ha/v1/_proxy.py#L335)
- [Resource wait helpers](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2591)
- [Instanceha Wait Contracts Test](../../../api/instanceha_wait_contracts_test.go)
