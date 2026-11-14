# Python Dhcp Networks

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
for row, err := range api.ListDHCPNetworkRecords(ctx, agentID,
	request.Query("is_shared", true),
	request.Query("fields", []string{"id", "name"}),
	request.Query("mtu", 1450),
) {
	...
}
```

## References

- [`dhcp_agent_hosting_networks(agent, **query)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)
- [`Resource.list`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
- [`DHCPAgentHostingNetwork`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/network.py)
