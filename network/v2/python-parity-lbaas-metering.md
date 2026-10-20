# Python Parity Lbaas Metering

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/create_load_balancer`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/delete_load_balancer`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/find_load_balancer`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/get_load_balancer`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/load_balancers`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/update_load_balancer`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/create_listener`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/delete_listener`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/find_listener`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/get_listener`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/listeners`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/update_listener`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/create_pool`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/delete_pool`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/find_pool`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/get_pool`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/pools`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/update_pool`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/create_pool_member`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/delete_pool_member`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/find_pool_member`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/get_pool_member`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/pool_members`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/update_pool_member`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/create_health_monitor`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/delete_health_monitor`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/find_health_monitor`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/get_health_monitor`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/health_monitors`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/update_health_monitor`

Status: `unsupported`.

Unresolved boundaries:

- Neutron LBaaS v2 (neutron-lbaas) is retired and replaced by Octavia (loadbalancer/v2); no supported Neutron serves /v2.0/lbaas, so the operation is excluded from the SDK target instead of being implemented.

### `python:network/v2/ips`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.API.All`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.API.List`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.WithRecordListOpts`

- All with no options sends GET /v2.0/floatingips with Accept application/json and follows floatingips_links, WithQuery forwards query-mapped keys such as floating_network_id and status, and WithName or WithBodyFilter return ErrUnsupported without HTTP.
  Tests: `network/v2/extensions/layer3/floatingips/python_parity_ips_test.go:TestPythonIPsListServerQueryAndPagesButNoLocalBodyFilters`.
- Source-shaped record lists send declared queries including tenant_id/tag wire aliases, keep status server-only, filter name/DNS/QoS/recursive port_details locally, discard unknown kwargs before encoding, and synchronize the shared name/address descriptor without changing native List/All.
  Tests: `network/v2/extensions/layer3/floatingips/python_list_records_test.go:TestPythonIPRecordsQueryClassificationAndAliasOrder`.
- Record view applies nullable/list/dictionary/integer descriptors and preserves unknown extensions while Wire and the actual page status/header/envelope remain independently owned.
  Tests: `network/v2/extensions/layer3/floatingips/python_list_records_test.go:TestPythonIPRecordsProjectionProofAndClone`, `network/v2/extensions/layer3/floatingips/python_list_records_test.go:TestPythonIPRecordsDescriptorFaultKeepsPhysicalPage`.
- The shared Source pager consumes max_items before local filters, supports deferred negative-cap stopping and consumer break without constructing later malformed rows; later HTTP failures retain native evidence and AllRecords discards partial results.
  Tests: `network/v2/extensions/layer3/floatingips/python_list_records_test.go:TestPythonIPRecordsCapsBeforeFiltersAndLazyBreak`, `network/v2/extensions/layer3/floatingips/python_list_records_test.go:TestPythonIPRecordsPaginationLateErrorsAndResponseProof`.
- Concrete options snapshot raw maps/bytes/headers/version, cancellation stops HTTP, and Proxy JMESPath observes an unconsumed generator rather than materializing the list.
  Tests: `network/v2/extensions/layer3/floatingips/python_list_records_test.go:TestPythonIPRecordsOwnedOptionsHeadersAndCancellation`, `network/v2/extensions/layer3/floatingips/python_list_records_test.go:TestPythonIPRecordsProxyGeneratorAndJMESPath`.

Python/Go differences:

- Native resource.WithStatus sends the status query and also filters case-insensitively in memory; the new record API instead keeps status server-only like Source. Native WithName/WithBodyFilter remain unsupported.
- Owned finite UTF-8 JSON records replace mutable Python Resource/runtime objects. Arbitrary Python objects, nonfinite JSON and full Requests/session/cache behavior remain explicit common profile boundaries.
- Go base_path overrides must be service-relative without traversal/query/fragment and continuation must remain in the selected collection; an empty override retains the default route.
- ListRecords/AllRecords reject truthy string JMESPath because it changes the return type. ProxyListRecords preserves Source unconsumed-generator evaluation and returns its separate opaque or JSON result.

## References

- [`_proxy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)
- [`load_balancer.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/load_balancer.py)
- [`listener.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/listener.py)
- [`pool.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/pool.py)
- [`pool_member.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/pool_member.py)
- [`health_monitor.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/health_monitor.py)
- [`flavor.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/flavor.py)
- [`service_profile.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/service_profile.py)
- [`metering_label.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/metering_label.py)
- [`metering_label_rule.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/metering_label_rule.py)
- [`floating_ip.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/floating_ip.py)
- [Python List Records](extensions/layer3/floatingips/python-list-records.md)
- [flavor](extensions/flavors/README.md)
- [service profile](extensions/serviceprofiles/README.md)
- [metering label](extensions/metering/labels/README.md)
- [metering rule](extensions/metering/rules/README.md)
