# Python Update Record

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="devstack")
    parser.add_argument("--container", default="c1")
    parser.add_argument("--attrs-json", default="{}")
    args = parser.parse_args()
    attrs = json.loads(args.attrs_json)
    with openstack.connect(cloud=args.cloud) as conn:
        result = conn.key_manager.update_container(args.container, **attrs)
        print(json.dumps(result.to_dict(), ensure_ascii=False))

if __name__ == "__main__":
    main()
```

See `Example_updateRecord` in [example_update_record_test.go](example_update_record_test.go), compiled by `go test`.

```sh
python main.py --cloud devstack --container c1 --attrs-json '{"name":"new","updated_at":null}'
go run main.go --cloud devstack --container c1 --attrs-json '{"name":"new","updated_at":null}'
```
