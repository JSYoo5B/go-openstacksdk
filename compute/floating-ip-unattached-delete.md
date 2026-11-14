# Floating Ip Unattached Delete

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `ExampleDeleteFloatingIPRequest` in [example_servers_test.go](example_servers_test.go), compiled by `go test`.

See `Example_floatingIpUnattachedDelete1` in [example_floating_ip_unattached_delete1_test.go](example_floating_ip_unattached_delete1_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect(cloud="dev")
deleted = conn.delete_unattached_floating_ips(retry=1)
if deleted is False:
    print("Some candidate deletions returned False")
else:
    print("Completed:", deleted)  # 0 is a successful empty result
```

## References

- [Floating Ip Queries](floating-ip-queries.md)
- [DeleteFloatingIP](floating-ip-delete.md)
- [Server Nova Floating Ip](server-nova-floating-ip.md)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1048-L1078)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L511-L566)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L977-L1046)
