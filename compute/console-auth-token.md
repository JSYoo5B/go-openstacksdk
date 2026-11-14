# Console Auth Token

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import sys
import openstack

conn = openstack.connect()
console = conn.compute.validate_console_auth_token(sys.argv[1])
print(console.to_dict())
```

See `Example_consoleAuthToken` in [example_console_auth_token_test.go](example_console_auth_token_test.go), compiled by `go test`.

## References

- [Policy](https://docs.openstack.org/nova/latest/configuration/policy.html)
- [Console Selection](console-selection.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [Implementation Plan](../docs/implementation-plan.md)
