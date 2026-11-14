# Tags Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

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

See `Example_tags` in [example_tags_test.go](example_tags_test.go), compiled by `go test`.

## References

- [Neutron 27.0.0 tagging router](https://github.com/openstack/neutron/blob/27.0.0/neutron/extensions/tagging.py)
- [extension middleware](https://github.com/openstack/neutron/blob/27.0.0/neutron/api/extensions.py)
- [Network](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/network.py)
- [QoS](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py)
- [Install](../../docs/install.md)
