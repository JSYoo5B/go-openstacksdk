# Waiting

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
notification = conn.instance_ha.get_notification("NOTIFICATION_UUID")
notification = conn.instance_ha.wait_for_status(
    notification, "finished", failures=["ERROR", "FAILED"], wait=180,
)
```

See `Example_notificationsWaitForStatus` in [example_waiting_fragment1_test.go](../example_waiting_fragment1_test.go), compiled by `go test`.

See `Example_vMovesInNotification` in [example_waiting_fragment2_test.go](../example_waiting_fragment2_test.go), compiled by `go test`.

See `Example_segmentsDelete` in [example_waiting_fragment3_test.go](../example_waiting_fragment3_test.go), compiled by `go test`.

## References

- [Masakari proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/instance_ha/v1/_proxy.py#L335)
- [Resource wait helpers](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2591)
- [Instanceha Wait Contracts Test](../../../api/instanceha_wait_contracts_test.go)
