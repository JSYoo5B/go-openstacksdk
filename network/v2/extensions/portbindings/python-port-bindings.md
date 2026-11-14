# Python Port Bindings

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--port", required=True)
parser.add_argument("--host", required=True)
parser.add_argument("--mode", choices=("create", "list", "activate", "delete"), required=True)
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "create":
        row = conn.network.create_port_binding(
            args.port, host=args.host, profile={"capabilities": ["switchdev"]})
        value = row.to_dict()
    elif args.mode == "list":
        value = [row.to_dict() for row in conn.network.port_bindings(
            args.port, host=args.host)]
    elif args.mode == "activate":
        row = conn.network.activate_port_binding(args.port, args.host)
        value = None if row is None else row.to_dict()
    else:
        conn.network.delete_port_binding(args.port, args.host)
        value = None
    print(json.dumps(value, indent=2))
```

See `Example_portBindings` in [example_port_bindings_test.go](example_port_bindings_test.go), compiled by `go test`.

## References

- [Install](../../../../docs/install.md)
