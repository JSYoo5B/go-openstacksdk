# Cloud Object Stream Python

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
parser.add_argument("--chunk-size", type=int, default=1024)
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    stream = conn.stream_object(
        args.container, args.object,
        query_string=args.query, resp_chunk_size=args.chunk_size,
    )
    try:
        for chunk in stream:
            sys.stdout.buffer.write(chunk)
    finally:
        stream.close()
```

See `Example_cloudObjectStream` in [example_cloud_object_stream_test.go](example_cloud_object_stream_test.go), compiled by `go test`.

## References

- [cloud stream_object](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_object_store.py#L484)
- [Object.stream/_download](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/obj.py#L319)
- [Install](../docs/install.md)
- [Python Cloud Stream Test](v1/objects/python_cloud_stream_test.go)
- [Cloud Object Stream Test](cloud_object_stream_test.go)
