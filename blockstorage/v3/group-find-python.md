# Group Find Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("identity", help="group ID or exact name")
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--microversion", default="3.38")
    parser.add_argument("--strict", action="store_true")
    parser.add_argument("--summary", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect(
        cloud=args.cloud,
        block_storage_default_microversion=args.microversion,
    )
    value = conn.block_storage.find_group(
        args.identity,
        ignore_missing=not args.strict,
        details=not args.summary,
    )
    print(json.dumps(None if value is None else value.to_dict(),
                     indent=2, ensure_ascii=False))

if __name__ == "__main__":
    main()
```

See `Example_groupsFindRecord` in [example_group_find1_test.go](example_group_find1_test.go), compiled by `go test`.

See `Example_groupsFindRecord2` in [example_group_find_fragment1_test.go](example_group_find_fragment1_test.go), compiled by `go test`.

## References

- [find_group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2126-L2155)
- [Group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group.py#L26-L66)
- [Resource.find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2489-L2582)
- [Groups Python](groups-python.md)
- [Install](../../docs/install.md)
- [Groups](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/groups.py)
