# Python Parity Sfc Tap

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/create_tap_mirror`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.WithCreateField`

- Create POSTs taas/tap_mirrors with a tap_mirror envelope holding name, description, port_id, mirror_type, remote_ip, string-quoted directions and project_id or other extension keys added through WithCreateField, decodes the created mirror, and rejects an extension that collides with a core key before HTTP.
  Tests: `network/v2/extensions/taas/tapmirrors/python_parity_test.go:TestPythonTapMirrorCreateBody`.

Python/Go differences:

- Python accepts a TapMirror Resource or an ID string; Go takes the ID string (or a resource.Ref).
- Python returns a TapMirror Resource whose project_id falls back to tenant_id; Go returns *tapmirrors.TapMirror with separate ProjectID and TenantID fields, and HTTP errors are gophercloud errors wrapped in resource.OperationError instead of openstack.exceptions.
- Neutron has no microversion, so neither SDK sends a version header.
- Python sends only the attributes passed in attrs; Go always sends name, port_id, mirror_type, remote_ip and directions even when empty and omits only empty description and tenant_id. Neutron requires port_id, mirror_type, remote_ip and directions and defaults name to an empty string, so valid calls produce the same result.
- Python passes the directions dict through unchanged; Go Directions sends IN/OUT tunnel IDs as JSON strings of integers, omits a zero ID and fails to decode a response whose tunnel IDs are unquoted numbers.
- Go accepts only 201 and 202 for create; Python accepts any non-error status.

### `python:network/v2/get_tap_mirror`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.API.GetWithFields`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.API.Get`

- GetWithFields sends GET taas/tap_mirrors/{id} with each Python TapMirror Body attribute name mapped to its wire name as repeated fields query keys, keeping caller order, repeats, empty and unknown names, and decodes the selected attributes into the typed model.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersSendAliasedRepeatedFields`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldExtractionUsesBodyMROWithoutExecutingSource`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationMatchesFreshPinnedSource`.
- GetWithFields with nil or empty fields sends the plain member GET without a query, which is Python's fields=None request, and a sparse response decodes with zero values for omitted attributes.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersWithoutFieldsSendPlainGet`.
- GetWithFields accepts HTTP 200 through 399 like the Python read, and returns 404 and other failures as the native gophercloud status error wrapped in resource.OperationError with the operation name.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersAcceptPythonStatusesAndWrapFailures`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationRejectsNativeResultAndInputDrift`.
- Get sends GET taas/tap_mirrors/{id} without a query, which is Python's request for fields=None, decodes the mirror and returns the native 404 error.
  Tests: `network/v2/extensions/taas/tapmirrors/python_parity_test.go:TestPythonTapMirrorGetWithoutFields`.

Python/Go differences:

- Python accepts a TapMirror Resource or an ID string; Go takes the ID string (or a resource.Ref).
- Python returns a TapMirror Resource whose project_id falls back to tenant_id; Go returns *tapmirrors.TapMirror with separate ProjectID and TenantID fields, and HTTP errors are gophercloud errors wrapped in resource.OperationError instead of openstack.exceptions.
- Neutron has no microversion, so neither SDK sends a version header.
- Python passes the directions dict through unchanged; Go Directions sends IN/OUT tunnel IDs as JSON strings of integers, omits a zero ID and fails to decode a response whose tunnel IDs are unquoted numbers.
- Go GetWithFields returns the Gophercloud v2.15.0 typed model, so attributes Neutron returns outside that model are dropped and attributes the server omits are zero values instead of None; Python returns a sparse Resource.
- The SDK field getter accepts HTTP 200..399 and keeps unknown field names in the query like Python; plain API.Get keeps the native status policy.

### `python:network/v2/tap_mirrors`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.API.All`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.WithListQueryValues`
- `github.com/JSYoo5B/go-openstacksdk/request.WithQueryValues`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithQuery`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithName`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPageSize`

- List with WithListQueryValues sends GET taas/tap_mirrors with fields, sort_key and sort_dir repeated once per value in order alongside name and limit, keeps the repeated keys when following the tap_mirrors_links next href with marker, lets a later option replace earlier values of the key and drops a key given no values.
  Tests: `network/v2/extensions/taas/tapmirrors/python_list_query_test.go:TestPythonTapMirrorListRepeatedQueryValues`.
- WithListQueryValues with a blank key fails before HTTP with resource.ErrInvalidOption wrapped in a List resource.OperationError, and a 400 list response is returned as the Gophercloud status error after sending the repeated sort_key values.
  Tests: `network/v2/extensions/taas/tapmirrors/python_list_query_test.go:TestPythonTapMirrorListRepeatedQueryRejectsInputAndStatus`.
- request.WithQueryValues snapshots the values, replaces an earlier single value, removes the key when given no values, and replaces the typed serialization of that key through ExtendQuery.
  Tests: `request/query_values_test.go:TestQueryValuesRepeatKeysAndExtendTypedQuery`.
- All sends GET taas/tap_mirrors with name, project_id and limit query values, follows the tap_mirrors_links next href with marker and decodes each mirror; a full page without links ends the listing after one request.
  Tests: `network/v2/extensions/taas/tapmirrors/python_parity_test.go:TestPythonTapMirrorListQueryAndPaging`.

Python/Go differences:

- Python sends only declared query keys (name, project_id, sort_key, sort_dir, fields, limit, marker), filters other Body attributes such as port_id locally and drops unknown keys; Go forwards every WithListQuery, WithListQueryValues or resource.WithQuery key to the server.
- Repeated query keys are available only through API.List; API.All and resource.WithQuery still send one value per key, so combining repeated fields with max_items means stopping the List iterator after that many rows.
- Python continues with marker=<last id> after a limit page without links; Go follows only tap_mirrors_links.
- Python returns TapMirror Resources whose project_id falls back to tenant_id; Go yields *tapmirrors.TapMirror with separate ProjectID and TenantID, and List HTTP errors are Gophercloud errors.
- Python passes the directions dict through unchanged; Go Directions decodes IN/OUT tunnel IDs from JSON strings of integers and fails on unquoted numbers.
- Neutron has no microversion, so neither SDK sends a version header.

### `python:network/v2/update_tap_mirror`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.WithUpdateField`

- Update PUTs taas/tap_mirrors/{id} with a tap_mirror envelope holding the pointer name and description values and extension keys from WithUpdateField, and decodes the updated mirror.
  Tests: `network/v2/extensions/taas/tapmirrors/python_parity_test.go:TestPythonTapMirrorUpdateBody`.

Python/Go differences:

- Python accepts a TapMirror Resource or an ID string; Go takes the ID string (or a resource.Ref).
- Python returns a TapMirror Resource whose project_id falls back to tenant_id; Go returns *tapmirrors.TapMirror with separate ProjectID and TenantID fields, and HTTP errors are gophercloud errors wrapped in resource.OperationError instead of openstack.exceptions.
- Neutron has no microversion, so neither SDK sends a version header.
- Python skips the request when no attribute is dirty; Go Update with empty UpdateOpts still sends PUT {"tap_mirror":{}}, so the caller omits the call instead.
- Go accepts only 200 for update; Python accepts any non-error status.
- Python passes the directions dict through unchanged; Go Directions sends IN/OUT tunnel IDs as JSON strings of integers, omits a zero ID and fails to decode a response whose tunnel IDs are unquoted numbers.

### `python:network/v2/delete_tap_mirror`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`
- `github.com/JSYoo5B/go-openstacksdk/resource.IgnoreMissing`

- Remove by ID and resource.IgnoreMissing around Delete each send one DELETE taas/tap_mirrors/{id}, return nil for 204 and 404 and return a 409 error, while Remove with WithMissingError reports a 404 as resource.ErrNotFound and plain Delete returns the native 404.
  Tests: `network/v2/extensions/taas/tapmirrors/python_parity_test.go:TestPythonTapMirrorDeleteIgnoreMissing`.

Python/Go differences:

- Python accepts a TapMirror Resource or an ID string; Go takes the ID string (or a resource.Ref).
- Python returns a TapMirror Resource whose project_id falls back to tenant_id; Go returns *tapmirrors.TapMirror with separate ProjectID and TenantID fields, and HTTP errors are gophercloud errors wrapped in resource.OperationError instead of openstack.exceptions.
- Neutron has no microversion, so neither SDK sends a version header.
- Python returns None; Go returns only an error. Go accepts 202 and 204 as success; Python accepts any non-error status.

## References

- [`tapmirrors`](extensions/taas/tapmirrors)
- [`python_parity_test.go`](extensions/taas/tapmirrors/python_parity_test.go)
- [`create_tap_mirror`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9066)
- [`get_tap_mirror`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9110)
- [`find_tap_mirror`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9096)
- [`tap_mirrors`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9138)
- [`update_tap_mirror`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9130)
- [`delete_tap_mirror`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9070)
- [`create_tap_flow`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L8987)
- [`get_tap_flow`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9029)
- [`find_tap_flow`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9015)
- [`tap_flows`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9057)
- [`update_tap_flow`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9049)
- [`delete_tap_flow`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L8991)
- [`create_tap_service`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9147)
- [`get_tap_service`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9191)
- [`find_tap_service`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9177)
- [`tap_services`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9219)
- [`update_tap_service`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9211)
- [`delete_tap_service`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9151)
- [`create_sfc_flow_classifier`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9228)
- [`get_sfc_flow_classifier`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9310)
- [`find_sfc_flow_classifier`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9283)
- [`sfc_flow_classifiers`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9347)
- [`update_sfc_flow_classifier`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9329)
- [`delete_sfc_flow_classifier`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9242)
- [`create_sfc_port_chain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9369)
- [`get_sfc_port_chain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9450)
- [`find_sfc_port_chain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9423)
- [`sfc_port_chains`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9483)
- [`update_sfc_port_chain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9467)
- [`delete_sfc_port_chain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9382)
- [`create_sfc_port_pair`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9502)
- [`get_sfc_port_pair`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9580)
- [`find_sfc_port_pair`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9554)
- [`sfc_port_pairs`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9613)
- [`update_sfc_port_pair`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9597)
- [`delete_sfc_port_pair`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9513)
- [`create_sfc_port_pair_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9632)
- [`get_sfc_port_pair_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9713)
- [`find_sfc_port_pair_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9686)
- [`sfc_port_pair_groups`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9750)
- [`update_sfc_port_pair_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9732)
- [`delete_sfc_port_pair_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9645)
- [`create_sfc_service_graph`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9770)
- [`get_sfc_service_graph`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9851)
- [`find_sfc_service_graph`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9824)
- [`sfc_service_graphs`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9886)
- [`update_sfc_service_graph`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9868)
- [`delete_sfc_service_graph`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9783)
- [`tap_mirror.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/tap_mirror.py)
- [`tap_flow.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/tap_flow.py)
- [`tap_service.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/tap_service.py)
- [`sfc_flow_classifier.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/sfc_flow_classifier.py)
- [`sfc_port_chain.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/sfc_port_chain.py)
- [`sfc_port_pair.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/sfc_port_pair.py)
- [`sfc_port_pair_group.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/sfc_port_pair_group.py)
- [`sfc_service_graph.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/sfc_service_graph.py)
- [flow classifier](extensions/sfc/flowclassifiers/README.md)
- [port pair](extensions/sfc/portpairs/README.md)
- [port pair group](extensions/sfc/portpairgroups/README.md)
- [port chain](extensions/sfc/portchains/README.md)
- [service graph](extensions/sfc/servicegraphs/README.md)
- [tap flow](extensions/taas/tapflows/README.md)
- [tap service](extensions/taas/tapservices/README.md)
