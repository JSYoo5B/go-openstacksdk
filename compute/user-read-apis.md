# User Read APIs

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect()
flavor = conn.compute.get_flavor("flavor-id", get_extra_specs=True)
server = conn.compute.fetch_server_metadata("server-id")
keypair = conn.compute.get_keypair("login-key")
console = conn.compute.get_server_console_output("server-id", length=50)
print(flavor.extra_specs, server.metadata, keypair.fingerprint, console)
```

See `Example_userReadApis` in [example_user_read_apis_test.go](example_user_read_apis_test.go), compiled by `go test`.

## References

- [Finding Identities](../docs/finding-identities.md)
- [get_flavor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L246-L266)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L1876-L1906)
- [MetadataMixin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/metadata.py#L33-L46)
- [get_keypair](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L868-L890)
- [Server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server.py#L938-L952)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
