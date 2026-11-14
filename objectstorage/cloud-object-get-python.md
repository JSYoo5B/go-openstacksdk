# Cloud Object Get Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import sys
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--container", default="assets")
parser.add_argument("--object", default="logs/part.bin")
parser.add_argument("--query")
parser.add_argument("--stream", action="store_true")
parser.add_argument("--chunk-size", type=int, default=1024)
output = parser.add_mutually_exclusive_group()
output.add_argument("--outfile")
output.add_argument("--discard-content", action="store_true")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    result = conn.get_object(
        args.container, args.object,
        query_string=args.query, stream=args.stream,
        resp_chunk_size=args.chunk_size,
        outfile="" if args.discard_content else args.outfile,
    )
    if result is None:
        print("missing", file=sys.stderr)
    else:
        headers, body = result
        print(json.dumps(headers, indent=2), file=sys.stderr)
        if body is not None:
            sys.stdout.write(body)
```

See `Example_cloudObjectGet` in [example_cloud_object_get_test.go](example_cloud_object_get_test.go), compiled by `go test`.

## References

- [cloud get_object](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_object_store.py#L512)
- [Proxy.get_object](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L288)
- [Install](../docs/install.md)
- [Python Cloud Get Test](v1/objects/python_cloud_get_test.go)
- [Cloud Object Get Test](cloud_object_get_test.go)
