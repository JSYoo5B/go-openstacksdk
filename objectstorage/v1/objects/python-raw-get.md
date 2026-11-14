# Python Raw Get

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import sys
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--container", default="assets")
parser.add_argument("--object", default="logs/part.bin")
parser.add_argument("--query")
parser.add_argument("--stream", action="store_true")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    response = conn.get_object_raw(
        args.container, args.object,
        query_string=args.query, stream=args.stream,
    )
    try:
        print(f"status={response.status_code}", file=sys.stderr)
        for chunk in response.iter_content(chunk_size=32768):
            sys.stdout.buffer.write(chunk)
    finally:
        response.close()
```

See `Example_rawGet` in [example_raw_get_test.go](example_raw_get_test.go), compiled by `go test`.

## References

- [`get_object_raw(container, obj, query_string=None, stream=False)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_object_store.py#L449-L484)
- [Install](../../../docs/install.md)
- [Python Raw Get Test](python_raw_get_test.go)
