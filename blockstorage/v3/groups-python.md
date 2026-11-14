# Groups Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--microversion", default="3.38")
    parser.add_argument("--action", choices=["list", "get", "create", "clone", "update", "delete"], default="list")
    parser.add_argument("--group", default="")
    parser.add_argument("--name", default="sample")
    parser.add_argument("--group-type", default="")
    parser.add_argument("--volume-types", default="")
    parser.add_argument("--source-group", default="")
    parser.add_argument("--source-snapshot", default="")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud,
        block_storage_default_microversion=args.microversion)
    api = conn.block_storage
    if args.action == "list":
        for value in api.groups():
            print(json.dumps(value.to_dict(), indent=2, ensure_ascii=False))
        return
    if args.action == "get":
        value = api.get_group(args.group, list_volume=True)
    elif args.action == "create":
        value = api.create_group(group_type=args.group_type,
            volume_types=args.volume_types.split(","), name=args.name)
    elif args.action == "clone":
        # Both IDs are required parameters of the pinned classmethod.
        value = api.create_group_from_source(
            group_snapshot_id=args.source_snapshot or None,
            source_group_id=args.source_group or None,
            name=args.name)
    elif args.action == "update":
        value = api.update_group(args.group, name=args.name)
    else:
        api.delete_group(args.group, delete_volumes=False, ignore_missing=False)
        return
    print(json.dumps(value.to_dict(), indent=2, ensure_ascii=False))

if __name__ == "__main__":
    main()
```

See `Example_groups` in [example_groups_test.go](example_groups_test.go), compiled by `go test`.

## References

- [Group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group.py)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2071-L2250)
- [Install](../../docs/install.md)
- [Group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group.py#L65)
- [Groups](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/groups.py)
- [group:delete](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/group_actions.py)
- [API](https://github.com/openstack/cinder/blob/27.0.0/cinder/group/api.py#L183-L200)
- [Python Records Test](groups/python_records_test.go)
- [Python Policy Test](groups/python_policy_test.go)
