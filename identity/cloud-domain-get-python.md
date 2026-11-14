# Cloud Domain Get Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--domain-id", required=True)
    parser.add_argument("--domain-name", default="")
    args = parser.parse_args()
    conn = openstack.connect()
    row = conn.identity.get_domain(args.domain_id)
    print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))
    row = conn.get_domain(domain_id=args.domain_id,
                          filters={"ignored": "without evaluation"})
    print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))
    if args.domain_name:
        found = conn.get_domain(name_or_id=args.domain_name)
        if found is not None:
            print(json.dumps(found.to_dict(), indent=2, ensure_ascii=False))

if __name__ == "__main__":
    main()
```

See `Example_cloudDomainGet` in [example_cloud_domain_get_test.go](example_cloud_domain_get_test.go), compiled by `go test`.

## References

- [Python Records](v3/domains/python-records.md)
- [Domain](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/domain.py)
- [list_domains](v3/domains/python-records.md)
