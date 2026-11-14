# Secret Store Waits Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import os
import openstack
from openstack.key_manager.v1.secret_store import SecretStore

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("id")
    parser.add_argument("--cloud", default=os.getenv("OS_CLOUD"))
    parser.add_argument("--status", default="ACTIVE")
    parser.add_argument("--delete", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    target = SecretStore(id=args.id)
    if args.delete:
        value = conn.key_manager.wait_for_delete(target)
    else:
        value = conn.key_manager.wait_for_status(target, args.status)
    print(json.dumps(value.to_dict(), ensure_ascii=False))

if __name__ == "__main__":
    main()
```

See `Example_secretStoreWaits` in [example_secret_store_waits_test.go](example_secret_store_waits_test.go), compiled by `go test`.

## References

- [Source schema](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/secret_store.py)
- [stock controller](https://github.com/openstack/barbican/blob/21.0.0/barbican/api/controllers/secretstores.py#L102-L156)
- [Barbican 21 stock policy](https://github.com/openstack/barbican/blob/21.0.0/barbican/common/policies/secretstores.py#L126-L138)
- [2025.2 policy configuration](https://docs.openstack.org/barbican/2025.2/configuration/policy.html)
