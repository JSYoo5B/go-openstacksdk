# Python List Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import os
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default=os.getenv("OS_CLOUD"))
    parser.add_argument("--network", default="")
    parser.add_argument("--host", default="")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    query = {}
    if args.network:
        query["network_id"] = args.network
    if args.host:
        query["binding_host_id"] = args.host
    for port in conn.network.ports(**query):
        print(json.dumps(port.to_dict(), ensure_ascii=False))

if __name__ == "__main__":
    main()
```

See `Example_listRecords` in [example_list_records_test.go](example_list_records_test.go), compiled by `go test`.

## References

- [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/port.py)
- [Source `Proxy._list`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866)
- [Port](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/port.py#L347)
