# Autoallocatedtopologies

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect()
validation = conn.network.validate_auto_allocated_topology(project="project-id")
print(validation.id, validation.project_id, validation.dry_run)
topology = conn.network.get_auto_allocated_topology(project="project-id")
print(topology.id)
```

See `Example_autoallocatedtopologies` in [example_autoallocatedtopologies_test.go](example_autoallocatedtopologies_test.go), compiled by `go test`.

## References

- [network proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)
- [Neutron27 backend](https://github.com/openstack/neutron/blob/27.0.0/neutron/services/auto_allocate/db.py)
- [Auto Allocated Topology](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/auto_allocated_topology.py)
- [Records Test](records_test.go)
