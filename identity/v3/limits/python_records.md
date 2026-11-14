# Python Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import json
import openstack

def main():
    conn = openstack.connect()
    for row in conn.identity.limits(
        service_id="compute-service-id",
        resource_name="cores",
        description="team quota",
        max_items=100,
    ):
        print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))

if __name__ == "__main__":
    main()
```

See `Example_records1` in [example_records1_test.go](example_records1_test.go), compiled by `go test`.

See `ExampleAPI_ProxyListRecords` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

## References

- [Policy](https://docs.openstack.org/keystone/latest/configuration/policy)
