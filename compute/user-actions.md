# User Actions

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect()
console = conn.compute.create_server_remote_console("server-id", type="novnc")
print(console.protocol, console.type, console.url)
conn.compute.delete_keypair("old-login-key")
```

See `Example_userActions` in [example_user_actions_test.go](example_user_actions_test.go), compiled by `go test`.

## References

- [Console Selection](console-selection.md)
- [create_server_remote_console](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L2903-L2920)
- [Server Remote Console](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server_remote_console.py#L18-L77)
- [delete_keypair](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L842-L866)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338-L1404)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
