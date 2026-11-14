# Subnet Pool Records Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
pool = conn.network.find_subnet_pool("POOL_NAME_OR_ID")
rows = list(conn.network.subnet_pools(is_shared=False, default_quota=128))
if pool is not None:
    updated = conn.network.update_subnet_pool(pool.id, name="renamed", default_quota=0)
    unchanged = conn.network.update_subnet_pool(pool.id)
```

```sh
go run . -cloud dev -action find -pool POOL_NAME_OR_ID
go run . -cloud dev -action list
go run . -cloud dev -action update -pool POOL_ID -name renamed -quota 0
```

See `Example_subnetPoolRecords` in [example_subnet_pool_records_test.go](example_subnet_pool_records_test.go), compiled by `go test`.

## References

- [Install](../../docs/install.md)
- [Subnetpool](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/subnetpool.py)
- [neutron-lib 3.21.1 schema](https://github.com/openstack/neutron-lib/blob/3.21.1/neutron_lib/api/definitions/subnetpool.py)
- [address-scope extension](https://github.com/openstack/neutron-lib/blob/3.21.1/neutron_lib/api/definitions/address_scope.py)
