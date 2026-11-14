# Python Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import json
import openstack

def main():
    conn = openstack.connect()
    for row in conn.identity.registered_limits(
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

```go
result, err := identity.RegisteredLimits.ProxyListRecords(ctx,
    registeredlimits.WithRecordListJMESPath("[*].id"),
)
if err != nil { log.Fatal(err) }
fmt.Println(string(result.JSON))

result, err = identity.RegisteredLimits.ProxyListRecords(ctx)
if err != nil { log.Fatal(err) }
generator := result.Value.(*registeredlimits.RecordListGenerator)
for row, err := range generator.Records {
    if err != nil { log.Fatal(err) }
    fmt.Println(string(row.Resource.Body["id"]))
}
```

## References

- [Policy](https://docs.openstack.org/keystone/latest/configuration/policy)
