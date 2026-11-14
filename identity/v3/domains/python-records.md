# Python Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import json
import openstack

def main():
    conn = openstack.connect()
    for row in conn.identity.domains(
        is_enabled=True, description="operations", limit=100, max_items=100
    ):
        print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))

    listed = conn.list_domains(is_enabled=True, description="operations")
    print(json.dumps([row.to_dict() for row in listed], indent=2))

    # Source distinguishes None from an explicitly empty identifier.
    server_filtered = conn.search_domains(filters={"name": "team"})
    locally_filtered = conn.search_domains(
        filters={"description": "operations"}, name_or_id="team-*"
    )
    print(json.dumps([row.to_dict() for row in server_filtered], indent=2))
    print(json.dumps([row.to_dict() for row in locally_filtered], indent=2))

if __name__ == "__main__":
    main()
```

See `Example_records` in [example_records_test.go](example_records_test.go), compiled by `go test`.

## References

- [Policy](https://docs.openstack.org/keystone/2025.2/configuration/policy.html)
- [Domain](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/domain.py)
- [Domains](https://github.com/openstack/keystone/blob/28.0.0/keystone/api/domains.py)
- [Python Record Mutations](python-record-mutations.md)
- [Python Domain Config](python-domain-config.md)
