# Keypairs Console

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
created = conn.compute.create_keypair(name="workstation", public_key=public_key)
console = conn.compute.get_server_console_url(server_id, "novnc")
```

See `Example_keypairsConsole` in [example_keypairs_console_test.go](example_keypairs_console_test.go), compiled by `go test`.

## References

- [User Actions](user-actions.md)
