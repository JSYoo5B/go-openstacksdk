# Python Token Actions

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
validated = conn.identity.validate_token(token, nocatalog=True)
print(validated.expires_at, validated.subject_token)
valid = conn.identity.check_token(token, allow_expired=True)
```

See `Example_tokenActions` in [example_token_actions_test.go](example_token_actions_test.go), compiled by `go test`.

## References

- [Install](../../docs/install.md)
- [Token Actions Test](tokens/token_actions_test.go)
