# Native QoS

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:blockstorage/v3/qos.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.WithCreateField`

- Create POSTs {"qos_specs":{...}} to qos-specs always sending name, merging Specs keys after typed fields (a name spec overrides Name) and extensions inside the object, rejects a core-key collision before HTTP and accepts only 200.
  Tests: `blockstorage/v3/qos/native_contracts_test.go:TestNativeQoSRoutesBodiesPagingAndDecode`, `blockstorage/v3/qos/native_contracts_test.go:TestNativeQoSStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/qos admin call only; the default Cinder policy limits QoS spec management to administrators, and Python QoS Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/qos.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.API.Get`

- Get GETs qos-specs/{id}, decodes the qos_specs key with consumer and specs and accepts only 200.
  Tests: `blockstorage/v3/qos/native_contracts_test.go:TestNativeQoSRoutesBodiesPagingAndDecode`, `blockstorage/v3/qos/native_contracts_test.go:TestNativeQoSStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/qos admin call only; the default Cinder policy limits QoS spec management to administrators, and Python QoS Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/qos.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.WithListQuery`

- List GETs qos-specs with sort/marker/limit and extension query values and follows qos_specs_links next hrefs as received.
  Tests: `blockstorage/v3/qos/native_contracts_test.go:TestNativeQoSRoutesBodiesPagingAndDecode`, `blockstorage/v3/qos/native_contracts_test.go:TestNativeQoSStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/qos admin call only; the default Cinder policy limits QoS spec management to administrators, and Python QoS Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/qos.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.WithUpdateField`

- Update PUTs {"qos_specs":{...}} to qos-specs/{id} with consumer and merged Specs keys, returns the response spec map and accepts only 200.
  Tests: `blockstorage/v3/qos/native_contracts_test.go:TestNativeQoSRoutesBodiesPagingAndDecode`, `blockstorage/v3/qos/native_contracts_test.go:TestNativeQoSStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/qos admin call only; the default Cinder policy limits QoS spec management to administrators, and Python QoS Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/qos.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.WithDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.WithDeleteQuery`

- Delete DELETEs qos-specs/{id} with force=true only when requested and accepts only 202/204.
  Tests: `blockstorage/v3/qos/native_contracts_test.go:TestNativeQoSRoutesBodiesPagingAndDecode`, `blockstorage/v3/qos/native_contracts_test.go:TestNativeQoSStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/qos admin call only; the default Cinder policy limits QoS spec management to administrators, and Python QoS Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/qos.DeleteKeys`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.API.DeleteKeys`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.WithDeleteKeysOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.WithDeleteKeysField`

- DeleteKeys PUTs {"keys":[...]} (null without keys) to qos-specs/{id}/delete_keys, rejects a keys extension before HTTP and accepts only 202.
  Tests: `blockstorage/v3/qos/native_contracts_test.go:TestNativeQoSRoutesBodiesPagingAndDecode`, `blockstorage/v3/qos/native_contracts_test.go:TestNativeQoSStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/qos admin call only; the default Cinder policy limits QoS spec management to administrators, and Python QoS Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/qos.Associate`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.API.Associate`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.WithAssociateOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.WithAssociateQuery`

- Associate GETs qos-specs/{id}/associate with the required vol_type_id query (rejected before HTTP when empty) and accepts only 202.
  Tests: `blockstorage/v3/qos/native_contracts_test.go:TestNativeQoSRoutesBodiesPagingAndDecode`, `blockstorage/v3/qos/native_contracts_test.go:TestNativeQoSStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/qos admin call only; the default Cinder policy limits QoS spec management to administrators, and Python QoS Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/qos.Disassociate`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.API.Disassociate`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.WithDisassociateOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.WithDisassociateQuery`

- Disassociate GETs qos-specs/{id}/disassociate with the required vol_type_id query (rejected before HTTP when empty) and accepts only 202.
  Tests: `blockstorage/v3/qos/native_contracts_test.go:TestNativeQoSRoutesBodiesPagingAndDecode`, `blockstorage/v3/qos/native_contracts_test.go:TestNativeQoSStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/qos admin call only; the default Cinder policy limits QoS spec management to administrators, and Python QoS Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/qos.DisassociateAll`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.API.DisassociateAll`

- DisassociateAll GETs qos-specs/{id}/disassociate_all and accepts only 202.
  Tests: `blockstorage/v3/qos/native_contracts_test.go:TestNativeQoSRoutesBodiesPagingAndDecode`, `blockstorage/v3/qos/native_contracts_test.go:TestNativeQoSStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/qos admin call only; the default Cinder policy limits QoS spec management to administrators, and Python QoS Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/qos.ListAssociations`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.API.ListAssociations`

- ListAssociations GETs qos-specs/{id}/associations as a single page of id, name and association_type rows.
  Tests: `blockstorage/v3/qos/native_contracts_test.go:TestNativeQoSRoutesBodiesPagingAndDecode`, `blockstorage/v3/qos/native_contracts_test.go:TestNativeQoSStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/qos admin call only; the default Cinder policy limits QoS spec management to administrators, and Python QoS Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

## References

- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/qos/requests.go)
