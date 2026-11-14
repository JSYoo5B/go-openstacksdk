# Python Summary

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--microversion", default="3.36")
    parser.add_argument("--all-projects", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect(
        cloud=args.cloud,
        block_storage_default_microversion=args.microversion,
    )
    value = conn.block_storage.summary(all_projects=args.all_projects)
    print(json.dumps(value.to_dict(), indent=2, ensure_ascii=False))

if __name__ == "__main__":
    main()
```

See `Example_summary` in [example_summary_test.go](example_summary_test.go), compiled by `go test`.

## References

- [summary Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1059-L1080)
- [BlockStorageSummary](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/block_storage_summary.py)
- [Resource.fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1778-L1837)
- [ListSummary](volume-summary-python.md)
- [Install](../../docs/install.md)
- [V3](https://docs.openstack.org/api-ref/block-storage/v3/#get-volumes-summary)
- [Volumes](https://github.com/openstack/cinder/blob/master/cinder/policies/volumes.py)
- [get_volume_summary](https://github.com/openstack/cinder/blob/master/cinder/volume/api.py)
- [python_summary_contracts_test.go](volumes/python_summary_contracts_test.go)
