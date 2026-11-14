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
    parser.add_argument("--find", default="")
    parser.add_argument("--network-name", default="")
    args = parser.parse_args()
    with openstack.connect(cloud=args.cloud) as conn:
        if args.find:
            result = conn.network.find_network_ip_availability(args.find, ignore_missing=False)
            print(json.dumps(result.to_dict(), ensure_ascii=False))
        else:
            query = {"network_name": args.network_name} if args.network_name else {}
            for record in conn.network.network_ip_availabilities(**query):
                print(json.dumps(record.to_dict(), ensure_ascii=False))

if __name__ == "__main__":
    main()
```

See `Example_records` in [example_records_test.go](example_records_test.go), compiled by `go test`.

```sh
python main.py --cloud devstack --network-name public
go run main.go --cloud devstack --network-name public
python main.py --cloud devstack --find n1
go run main.go --cloud devstack --find n1
```
