# Python Create Record

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/create_bgpvpn`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.CreateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithCreateRecordOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithCreateRecordLocalPref`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithCreateRecordField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithCreateRecordHeader`
- `github.com/JSYoo5B/go-openstacksdk/request.Null`
- `github.com/JSYoo5B/go-openstacksdk/request.Present`

- CreateRecord sends POST bgpvpn/bgpvpns with only the supplied attributes in the bgpvpn envelope, including local_pref 0, vni 0, an empty name, an empty route_targets list, null import_targets and unknown body attributes through WithCreateRecordField, sends an empty envelope when no attribute is supplied, accepts any status below 400 and decodes enveloped, flat and empty responses.
  Tests: `network/v2/extensions/bgpvpns/python_create_record_test.go:TestPythonBGPVPNCreateRecordSendsZeroLocalPref`.
- Extension fields that collide with typed keys, blank extension keys and nil options fail before HTTP, and a 400 response is returned as the Gophercloud status error wrapped in resource.OperationError.
  Tests: `network/v2/extensions/bgpvpns/python_create_record_test.go:TestPythonBGPVPNCreateRecordRejectsInputsAndErrorStatuses`.

Python/Go differences:

- Python converts values of list-typed attributes before sending; Go serializes the Go value as given.
- Python returns the Resource merged from request attributes and the response; Go returns BGPVPNRecord with the native typed decode, the raw response object fields, headers and status.

## References

- [`create_bgpvpn`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)
- [`BgpVpn`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/bgpvpn.py)
