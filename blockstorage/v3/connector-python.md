# Connector Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect()
connector = {
    "host": "compute-1",
    "wwnns": ["1000000000000001", "1000000000000002"],
    "nqn": "nqn.2014-08.org.example:compute-1",
    "do_local_attach": False,
    "extensions": {"optional": None, "number": 0, "devices": []},
}
connection_info = conn.block_storage.init_volume_attachment("volume-id", connector)
conn.block_storage.terminate_volume_attachment("volume-id", connector)
```

See `Example_connector` in [example_connector_test.go](example_connector_test.go), compiled by `go test`.

## References

- [Python Connector Test](volumes/python_connector_test.go)
