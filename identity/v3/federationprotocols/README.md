# Federationprotocols

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
conn.identity.create_federation_protocol("corp", id="saml2", mapping_id="corp-mapping")
protocol = conn.identity.find_federation_protocol("corp", "saml2")
conn.identity.delete_federation_protocol("corp", "saml2")
```

```go
api := federationprotocols.New(identityClient)
_, err := api.Create(ctx, "corp", "saml2",
    request.Field("mapping_id", "corp-mapping"))
if err != nil { return err }
protocol, err := api.Find(ctx, "corp", "saml2")
if err != nil { return err }
err = api.Delete(ctx, "corp", protocol.ID)
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2725-L2912)
- [`federation_protocol.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/federation_protocol.py)
