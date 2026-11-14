# Group Replication Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("group", help="literal group ID")
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--microversion", default="3.38")
    parser.add_argument("--action", choices=["enable", "disable", "failover"],
                        default="enable")
    parser.add_argument("--allow-attached-volume", action="store_true")
    parser.add_argument("--secondary-backend-id", default=None)
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud,
        block_storage_default_microversion=args.microversion)
    if args.action == "enable":
        conn.block_storage.enable_group_replication(args.group)
    elif args.action == "disable":
        conn.block_storage.disable_group_replication(args.group)
    else:
        conn.block_storage.failover_group_replication(
            args.group,
            allowed_attached_volume=args.allow_attached_volume,
            secondary_backend_id=args.secondary_backend_id,
        )
    print("action call completed without exception")

if __name__ == "__main__":
    main()
```

See `Example_groupReplication` in [example_group_replication_test.go](example_group_replication_test.go), compiled by `go test`.

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2275-L2319)
- [Group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group.py#L101-L140)
- [Group leaf](groups-python.md)
- [V3](https://docs.openstack.org/api-ref/block-storage/v3/#group-replication-groups-action)
- [Group Actions](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/group_actions.py#L81-L116)
- [Policy](https://docs.openstack.org/cinder/2025.2/configuration/block-storage/policy.html)
- [Groups](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/schemas/groups.py#L132-L173)
- [controller](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/v3/groups.py#L339-L404)
- [Install](../../docs/install.md)
