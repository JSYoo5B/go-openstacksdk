# Floating Ip Delete

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
result, err := conn.DeleteFloatingIP(ctx,
	compute.DeleteFloatingIPRequest{ID: ipID},
	func(c *compute.FloatingIPDeleteOpts) error { c.Retries = 1; return nil },
	compute.WithFloatingIPDeleteTimeout(time.Minute),
)
```

See `Example_floatingIpDelete1` in [example_floating_ip_delete1_test.go](example_floating_ip_delete1_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect(cloud="dev")
deleted = conn.delete_floating_ip("IP_ID")
accepted = conn.delete_floating_ip("IP_ID", retry=0)
```

## References

- [Floating Ip Queries](floating-ip-queries.md)
- [Server Nova Floating Ip](server-nova-floating-ip.md)
- [public retry/verification](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L977-L1017)
- [Neutron/Nova delete helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1019-L1046)
- [ Utils](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py#L148-L201)
- [Available](floating-ip-available.md)
- [IP helper](server-ip-helpers.md)
- [CreateFloatingIP](floating-ip-create.md)
- [Floating Ip Unattached Delete](floating-ip-unattached-delete.md)
