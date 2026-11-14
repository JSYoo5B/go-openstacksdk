# Share Attachments Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--server", required=True)
parser.add_argument("--share", default="")
parser.add_argument("--mode", choices=("list", "get", "create", "delete"), default="list")
parser.add_argument("--tag", default=None)
args = parser.parse_args()

with openstack.connect(cloud=args.cloud, compute_default_microversion="2.97") as conn:
    if args.mode == "list":
        for value in conn.compute.share_attachments(args.server):
            print(json.dumps(value.to_dict(), indent=2))
    elif args.mode == "get":
        value = conn.compute.get_share_attachment(args.server, args.share)
        print(json.dumps(value.to_dict(), indent=2))
    elif args.mode == "create":
        attrs = {} if args.tag is None else {"tag": args.tag}
        value = conn.compute.create_share_attachment(args.server, args.share, **attrs)
        print(json.dumps(value.to_dict(), indent=2))
    else:
        conn.compute.delete_share_attachment(args.server, args.share)
        print(json.dumps({"delete_requested": args.share}, indent=2))
```

See `Example_shareAttachments` in [example_share_attachments_test.go](example_share_attachments_test.go), compiled by `go test`.

## References

- [Compute Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L3422)
- [ShareMapping](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server_share.py)
- [Install](../../docs/install.md)
- [Server Shares](https://github.com/openstack/nova/blob/31.0.0/nova/policies/server_shares.py)
- [controller](https://github.com/openstack/nova/blob/31.0.0/nova/api/openstack/compute/server_shares.py)
- [schema](https://github.com/openstack/nova/blob/31.0.0/nova/api/openstack/compute/schemas/server_shares.py)
- [Base](https://github.com/openstack/nova/blob/31.0.0/nova/policies/base.py)
