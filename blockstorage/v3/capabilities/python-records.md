# Backend capability lookup

```python
import openstack

conn = openstack.connect()
capabilities = conn.block_storage.get_capabilities("host@backend")
print(capabilities.properties)
```

See `Example_records` in [example_records_test.go](example_records_test.go), compiled by `go test`.

The host is a literal escaped URI segment. Only declared fields enter Resource;
Wire retains unknown extensions. Properties uses the shared Source dictionary
projection and replication_targets uses its list projection, preserving literal
JSON in Wire. Missing values and explicit null remain visible. Headers can be
supplied with `request.Headers`; mutable Python Resource inputs and
Requests decoding outside the shared finite JSON profile remain outside this
scalar-host API. A 404 is never converted into an empty capability set. These
examples do not establish live-cloud validation.
