# Network Roles

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Configuration sources

`Connect` reads role configuration from the selected clouds.yaml cloud, merged with secure.yaml and its profile. Without `OS_CLOUD` (and without explicit authentication options) it reads Python's environment-variable cloud instead.

- Profiles: a `profile` (or legacy `cloud`) found in clouds-public.yaml is used; otherwise the embedded, unmodified built-in vendor profiles of the pinned Python SDK are consulted, with the cloud's own settings taking precedence. Native authentication then receives that built-in profile too. A `shutdown` profile and a profile given by URL (including the nested URLs of `ovh` and `vexxhost`, which Python fetches) are configuration errors; deprecated profiles load without a warning.
- Environment cloud: every `OS_*` variable except the `OS_TEST`, `OS_STD` and `OS_LOG` families becomes a lower-case string setting, as in `_get_os_environ`, so `OS_EXTERNAL_NETWORK`, `OS_INTERNAL_NETWORK`, `OS_USE_EXTERNAL_NETWORK`, `OS_USE_INTERNAL_NETWORK`, `OS_SECGROUP_SOURCE` and `OS_PROFILE` apply. An empty variable is a present setting. String `OS_HAS_*` values are rejected because the flag must be a boolean. Authentication still comes from Gophercloud's environment parser.

- Regions: when the cloud (or, without `regions` and `region_name`, its profile) has a `regions` list, the selected entry's non-null `values` replace the merged settings before roles are read, as `OpenStackConfig.get_one` does. A `networks` list in the values replaces the whole list. `WithRegion` must name an entry; without it the entry named by `region_name` is used when present, otherwise the first entry. Region values only reach SDK-owned settings, not native authentication or endpoint parsing.
- `has_network`: `WithServiceEnabled("network", false)` or `has_network: false` makes every role getter return empty results without a request, and makes the floating IP helpers use Nova even with the Neutron source, mirroring `has_service('network')`. Unset, the official `network` type is enabled.
- `use_external_network`/`use_internal_network`: YAML booleans, null, the empty string and any casing of `"true"` behave like Python's truthiness of the raw value. Other strings are false in Go but truthy in Python; numbers and lists are rejected by `Connect` (Python turns numbers, even 0, into truthy strings).
- Lookup failures: a missing Network endpoint always means no Neutron. Other endpoint lookup failures are returned unless `WithServiceLookupFailureAsUnavailable(true)` is set, which maps Python's `has_service` returning False: roles are empty (retried on the next call) and the floating IP helpers use Nova. Cancellation is always returned.
- Typed `WithNetworkRoles` and `WithServiceEnabled` options replace the configured policy for their concern.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
for net in conn.get_external_ipv4_networks():
    print(net.id, net.name)
print(conn.get_nat_source())
print(conn.get_nat_destination())
print(conn.get_default_network())
```

See `Example_networkRoles1` in [example_network_roles1_test.go](example_network_roles1_test.go), compiled by `go test`.

See `Example_rolesDiscover` in [example_network_roles_fragment1_test.go](example_network_roles_fragment1_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect(cloud="dev")
print(conn.get_default_network())
server = conn.create_server(
    name="web", image="ubuntu", flavor="c2", wait=True, auto_ip=True,
)
print(server.id)
```

See `Example_networkRoles2` in [example_network_roles2_test.go](example_network_roles2_test.go), compiled by `go test`.

## References

- [Server Automatic Ip](../compute/server-automatic-ip.md)
- [Create With Automatic Floating Ip](../compute/create-with-automatic-floating-ip.md)
- [Server Addresses](../compute/server-addresses.md)
- [Connection Network Role Consumers Test](../connectiontest/connection_network_role_consumers_test.go)
- [Floating Ip Roles Test](floating_ip_roles_test.go)
- [Connection Server Network Roles Test](../connectiontest/connection_server_network_roles_test.go)
- [CreateWithAutomaticFloatingIP](../compute/create-with-automatic-floating-ip.md)
- [Network Mutations](network-mutations.md)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L95-L432)
- [Cloud Region](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/config/cloud_region.py#L1427-L1495)
- [Config Loader](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/config/loader.py#L733-L790)
- [Server Ip Dispatch](../compute/server-ip-dispatch.md)
- [Nova backend](../compute/server-nova-floating-ip.md)
- [ready/wait](../compute/server-ready.md)
- [Roles Test](roles_test.go)
- [Connection Network Roles Test](../connectiontest/connection_network_roles_test.go)
