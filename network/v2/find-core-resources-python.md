# Find Core Resources Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--network", default="app-net")
    parser.add_argument("--port", default="")
    parser.add_argument("--subnet", default="")
    parser.add_argument("--project", default="")
    parser.add_argument("--strict", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    query = {"project_id": args.project} if args.project else {}
    finders = [
        ("network", args.network, conn.network.find_network),
        ("port", args.port, conn.network.find_port),
        ("subnet", args.subnet, conn.network.find_subnet),
    ]
    for kind, identity, find in finders:
        if not identity:
            continue
        result = find(identity, ignore_missing=not args.strict, **query)
        print(kind)
        print(json.dumps(result.to_dict() if result is not None else None,
                         indent=2, ensure_ascii=False))

if __name__ == "__main__":
    main()
```

See `Example_findCoreResources1` in [example_find_core_resources1_test.go](example_find_core_resources1_test.go), compiled by `go test`.

```python
port = conn.network.find_port(
    "app-port", binding_profile={"nested": {"enabled": True}})
```

See `Example_findPortRecord` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

## References

- [Install](../../docs/install.md)
- [Network find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L3214)
- [Port find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L3950)
- [Subnet find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L7487)
- [Resource.find/list/fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
- [NetworkResource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_base.py)
- [Network](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/network.py)
- [Port](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/port.py)
- [Subnet](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/subnet.py)
