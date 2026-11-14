# Python Port Association Find

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--vpn", required=True)
parser.add_argument("--association", required=True)
parser.add_argument("--port", required=True)
args = parser.parse_args()
with openstack.connect() as conn:
    row = conn.network.find_bgpvpn_port_association(
        args.association, args.vpn, ignore_missing=False, port_id=args.port)
    print(json.dumps(row.to_dict(), indent=2))
```

See `Example_portAssociationFind` in [example_port_association_find_test.go](example_port_association_find_test.go), compiled by `go test`.

## References

- [Proxy.find_bgpvpn_port_association](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)
- [Bgpvpn Port Association](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/bgpvpn_port_association.py)
