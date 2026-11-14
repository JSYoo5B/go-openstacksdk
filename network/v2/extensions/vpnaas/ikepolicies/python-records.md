# Python Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="devstack")
    parser.add_argument("--name", default="")
    parser.add_argument("--units", default="")
    args = parser.parse_args()
    query = {}
    if args.name:
        query["name"] = args.name
    if args.units:
        query["units"] = args.units
    with openstack.connect(cloud=args.cloud) as conn:
        for row in conn.network.vpn_ike_policies(**query):
            print(json.dumps(row.to_dict(), ensure_ascii=False))

if __name__ == "__main__":
    main()
```

See `Example_records` in [example_records_test.go](example_records_test.go), compiled by `go test`.

```sh
python main.py --cloud devstack --name policy
go run main.go --cloud devstack --name policy
```
