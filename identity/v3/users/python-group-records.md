# Python Group Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--group-id", required=True)
    args = parser.parse_args()
    conn = openstack.connect()
    for row in conn.identity.group_users(
        args.group_id, email="member@example.com", limit=100, max_items=100
    ):
        print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))

if __name__ == "__main__":
    main()
```

See `Example_groupRecords` in [example_group_records_test.go](example_group_records_test.go), compiled by `go test`.

## References

- [Group](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/group.py)
- [controller](https://github.com/openstack/keystone/blob/28.0.0/keystone/api/groups.py)
- [Python Records](python_records.md)
