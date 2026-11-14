# Python Parity L3 QoS Fw

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example_deleteBandwidthLimitRule` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

## References

- [`network/v2/_proxy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)
- [`extensions/fwaas_v2/groups/python_parity_test.go`](extensions/fwaas_v2/groups/python_parity_test.go)
- [`extensions/fwaas_v2/policies/python_parity_test.go`](extensions/fwaas_v2/policies/python_parity_test.go)
- [`extensions/fwaas_v2/rules/python_parity_test.go`](extensions/fwaas_v2/rules/python_parity_test.go)
- [`extensions/qos/rules/python_parity_test.go`](extensions/qos/rules/python_parity_test.go)
- [`extensions/agents/python_parity_test.go`](extensions/agents/python_parity_test.go)
- [`extensions/layer3/routers/python_parity_test.go`](extensions/layer3/routers/python_parity_test.go)
- [Python Router Assignment Dhcp Hosts](extensions/agents/python-router-assignment-dhcp-hosts.md)
- [`agent.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/agent.py)
- [qos-bandwidth-limit-records-python.md](qos-bandwidth-limit-records-python.md)
- [conntrack helper](extensions/layer3/conntrackhelpers/README.md)
- [NDP proxy](extensions/layer3/ndpproxies/README.md)
- [Defaultrules](extensions/security/defaultrules/README.md)
- [Defaultstatefulness](extensions/security/defaultstatefulness/README.md)
- [Packet Rate Mutations Python](extensions/qos/rules/packet-rate-mutations-python.md)
- [`openstack/network/v2/_proxy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)
- [`firewall_group.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/firewall_group.py)
- [`firewall_policy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/firewall_policy.py)
- [`firewall_rule.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/firewall_rule.py)
- [`router.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/router.py)
- [`qos_bandwidth_limit_rule.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_bandwidth_limit_rule.py)
- [`qos_dscp_marking_rule.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_dscp_marking_rule.py)
- [`qos_minimum_bandwidth_rule.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_minimum_bandwidth_rule.py)
- [`qos_minimum_packet_rate_rule.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_minimum_packet_rate_rule.py)
- [`qos_packet_rate_limit_rule.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_packet_rate_limit_rule.py)
- [`l3_conntrack_helper.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/l3_conntrack_helper.py)
- [`ndp_proxy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/ndp_proxy.py)
- [`default_security_group_rule.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/default_security_group_rule.py)
- [`security_groups_default_statefulness.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/security_groups_default_statefulness.py)
- [`proxy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py)
- [`resource.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
