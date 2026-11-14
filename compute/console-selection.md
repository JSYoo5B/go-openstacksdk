# Console Selection

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect()
console = conn.compute.create_console("server-id", "novnc")
print(console)
```

See `Example_consoleSelection` in [example_console_selection_test.go](example_console_selection_test.go), compiled by `go test`.

## References

- [User Actions](user-actions.md)
- [Keypairs Console](keypairs-console.md)
