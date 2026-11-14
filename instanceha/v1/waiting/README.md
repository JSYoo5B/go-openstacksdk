# Waiting

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

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
