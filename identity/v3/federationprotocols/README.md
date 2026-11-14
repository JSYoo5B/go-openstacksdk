# Federationprotocols

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
conn.identity.create_federation_protocol("corp", id="saml2", mapping_id="corp-mapping")
protocol = conn.identity.find_federation_protocol("corp", "saml2")
conn.identity.delete_federation_protocol("corp", "saml2")
```

See `ExampleAPI_Create` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2725-L2912)
- [`federation_protocol.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/federation_protocol.py)
