# Consistency Groups Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--microversion", default=None)
    parser.add_argument("--action", choices=["list", "get", "find", "create", "clone", "update", "delete"], default="list")
    parser.add_argument("--group", default="")
    parser.add_argument("--name", default="sample")
    parser.add_argument("--volume-types", default="")
    parser.add_argument("--source-group", default="")
    parser.add_argument("--source-snapshot", default="")
    parser.add_argument("--summary", action="store_true")
    parser.add_argument("--strict", action="store_true")
    parser.add_argument("--force", action="store_true")
    args = parser.parse_args()
    policy = {}
    if args.microversion is not None:
        policy["block_storage_default_microversion"] = args.microversion
    api = openstack.connect(cloud=args.cloud, **policy).block_storage
    if args.action == "list":
        for value in api.consistency_groups(details=not args.summary):
            print(json.dumps(value.to_dict(), indent=2, ensure_ascii=False))
        return
    if args.action == "get":
        value = api.get_consistency_group(args.group)
    elif args.action == "find":
        value = api.find_consistency_group(args.group,
            ignore_missing=not args.strict, details=not args.summary)
    elif args.action == "create":
        value = api.create_consistency_group(volume_types=args.volume_types,
            name=args.name)
    elif args.action == "clone":
        value = api.create_consistency_group_from_source(
            consistency_group=args.source_group or None,
            consistency_group_snapshot=args.source_snapshot or None,
            name=args.name)
    elif args.action == "update":
        value = api.update_consistency_group(args.group, name=args.name)
    else:
        # This exact pinned helper sends consistency_group rather than the
        # stock controller's consistencygroup and can receive HTTP400.
        api.delete_consistency_group(args.group, force=args.force)
        return
    print(json.dumps(None if value is None else value.to_dict(),
                     indent=2, ensure_ascii=False))

if __name__ == "__main__":
    main()
```

See `Example_consistencyGroups` in [example_consistency_groups_test.go](example_consistency_groups_test.go), compiled by `go test`.

## References

- [ConsistencyGroup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/consistency_group.py)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2322-L2504)
- [Groups Python](groups-python.md)
- [Config](https://github.com/openstack/cinder/blob/27.0.0/cinder/common/config.py#L59-L61)
- [v3 router](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/v3/router.py#L165-L171)
- [Install](../../docs/install.md)
- [stock v3 update](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/v3/consistencygroups.py#L34-L100)
- [stock controller](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/contrib/consistencygroups.py#L61-L76)
- [view builder](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/views/consistencygroups.py)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1418-L1445)
- [group policy](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/groups.py)
- [delete policy](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/group_actions.py#L58-L68)
- [Python Record Contracts Test](consistencygroups/python_record_contracts_test.go)
- [Python Find Contracts Test](consistencygroups/python_find_contracts_test.go)
