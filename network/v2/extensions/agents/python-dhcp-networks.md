# Python Dhcp Networks

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/dhcp_agent_hosting_networks`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.ListDHCPNetworkRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.WithDHCPNetworkRecordsOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.WithDHCPNetworkRecordsQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.WithDHCPNetworkRecordsMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.WithDHCPNetworkRecordsPaginated`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.WithDHCPNetworkRecordsHeader`

- ListDHCPNetworkRecords sends GET agents/{id}/dhcp-networks with Network query keys translated to wire names (is_shared as shared=True) and list values repeated, drops unknown keys, follows a /v2.0 relative networks_links next href, then continues with marker=<last id> after a page without links while a limit is present until an empty page, and yields the typed Network, raw Fields without self, AgentID and status.
  Tests: `network/v2/extensions/agents/python_dhcp_networks_test.go:TestPythonDHCPNetworkRecordsQueryAndPaging`.
- Network Body attributes outside the query mapping such as mtu, subnet_ids and dns_domain=None filter rows locally with Python equality, max_items is sent as limit when no limit is given and counts filtered rows before stopping, and paginated=False reads only the first page.
  Tests: `network/v2/extensions/agents/python_dhcp_networks_test.go:TestPythonDHCPNetworkRecordsClientFiltersAndControls`.
- A blank agent ID, a nil option, a call-control query key such as max_items, a blank query key, a negative max_items, an unencodable value and a nil API fail before HTTP with resource.ErrInvalidOption in a ListDHCPNetworkRecords resource.OperationError, and a 404 is returned as the Gophercloud status error wrapped for agents.
  Tests: `network/v2/extensions/agents/python_dhcp_networks_test.go:TestPythonDHCPNetworkRecordsRejectsInputsAndErrorStatuses`.
- ListDHCPNetworks sends a bare GET agents/{id}/dhcp-networks and decodes the networks array, matching the Python call without query arguments.
  Tests: `network/v2/extensions/agents/python_parity_core_test.go:TestPythonParityCoreAgentDHCPNetworkRoutes`.

Python/Go differences:

- Python accepts the agent as an ID or an Agent Resource; Go takes an ID string, and Python base_path and jmespath_filters are not supported.
- Python compares client filters against typed Resource attributes (for example int mtu); Go compares the raw response JSON with Python equality.
- Python yields DHCPAgentHostingNetwork Resources; Go yields DHCPNetworkRecord values with the Gophercloud networks.Network decode and the raw row. Go encodes query keys in sorted order.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
for row, err := range api.ListDHCPNetworkRecords(ctx, agentID,
	agents.WithDHCPNetworkRecordsQuery("is_shared", true),
	agents.WithDHCPNetworkRecordsQuery("fields", []string{"id", "name"}),
	agents.WithDHCPNetworkRecordsQuery("mtu", 1450),
) {
	...
}
```

## References

- [`dhcp_agent_hosting_networks(agent, **query)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)
- [`Resource.list`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
- [`DHCPAgentHostingNetwork`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/network.py)
