# Cloud Security Group Reads Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import os
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default=os.getenv("OS_CLOUD"))
parser.add_argument("--mode", choices=("search", "get"), default="search")
parser.add_argument("--name", default="")
parser.add_argument("--filters", default="null")
parser.add_argument("--existing", default="")
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    filters = json.loads(args.filters)
    if args.mode == "get":
        reference = json.loads(args.existing) if args.existing else args.name
        result = conn.get_security_group(reference, filters)
    else:
        result = conn.search_security_groups(args.name or None, filters)
    if hasattr(result, "to_dict"):
        result = result.to_dict()
    elif isinstance(result, list):
        result = [value.to_dict() if hasattr(value, "to_dict") else value
                  for value in result]
    print(json.dumps(result, ensure_ascii=False, indent=2))
```

See `Example_cloudSecurityGroupReads` in [example_cloud_security_group_reads_test.go](example_cloud_security_group_reads_test.go), compiled by `go test`.

## References

- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1834)
- [API Microversion History](https://docs.openstack.org/nova/2026.1/reference/api-microversion-history.html)
- [Secgroups](../compute/v2/secgroups/README.md)
- [Security Group](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/security_group.py)
- [Cloud Security Group Reads Test](cloud_security_group_reads_test.go)
- [Python List Proxy Test](v2/extensions/security/groups/python_list_proxy_test.go)
- [Connection Security Group Records Test](../connectiontest/connection_security_group_records_test.go)
