# Messages Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--microversion", default="3.37")
    parser.add_argument("--message", default="")
    parser.add_argument("--project", default="")
    parser.add_argument("--delete", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud,
        block_storage_default_microversion=args.microversion)
    if args.message:
        if args.delete:
            conn.block_storage.delete_message(args.message, ignore_missing=False)
        else:
            value = conn.block_storage.get_message(args.message)
            print(json.dumps(value.to_dict(), indent=2, ensure_ascii=False))
    else:
        query = {"project_id": args.project} if args.project else {}
        for value in conn.block_storage.messages(**query):
            print(json.dumps(value.to_dict(), indent=2, ensure_ascii=False))

if __name__ == "__main__":
    main()
```

See `Example_messages` in [example_messages_test.go](example_messages_test.go), compiled by `go test`.

## References

- [Message](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/message.py)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L3029-L3064)
- [Install](../../docs/install.md)
- [V3](https://docs.openstack.org/api-ref/block-storage/v3/#show-message-details)
- [Policy Personas](https://docs.openstack.org/cinder/2026.1/configuration/block-storage/policy-personas.html#user-messages-microversion-3-3)
- [Policy](https://docs.openstack.org/cinder/2026.1/configuration/block-storage/policy.html)
- [Python Records Contracts Test](messages/python_records_contracts_test.go)
