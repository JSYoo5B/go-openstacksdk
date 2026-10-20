# Python Update Record

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/update_bgp_peer`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.WithUpdateRecordOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.WithUpdateRecordName`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.WithUpdateRecordPassword`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.WithUpdateRecordField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.WithUpdateRecordHeader`
- `github.com/JSYoo5B/go-openstacksdk/request.Null`
- `github.com/JSYoo5B/go-openstacksdk/request.Present`

- UpdateRecord sends PUT bgp-peers/{id} with only the supplied attributes in the bgp_peer envelope, including an empty name, a null name, an empty password and unknown body attributes through WithUpdateRecordField, accepts any status below 400, decodes enveloped, flat and empty responses, and sends nothing when no attribute is supplied.
  Tests: `network/v2/extensions/bgp/peers/python_update_record_test.go:TestPythonBGPPeerUpdateRecordSendsEmptyName`.
- Extension fields that collide with typed keys or the id identity, empty extension keys, nil options and blank or dot IDs fail before HTTP, and a 409 response is returned as the Gophercloud status error wrapped in resource.OperationError.
  Tests: `network/v2/extensions/bgp/peers/python_update_record_test.go:TestPythonBGPPeerUpdateRecordRejectsInputsAndErrorStatuses`.

Python/Go differences:

- Python accepts a BgpPeer Resource or an ID string; Go takes the ID string and rejects blank, '.' and '..' IDs before HTTP.
- Python silently discards an id attribute from the commit; WithUpdateRecordField rejects the id key before HTTP.
- Python returns the Resource merged from request attributes and the response; Go returns BGPPeerRecord with the native typed decode, the raw response object fields, headers and status, and Sent false when no request was needed.

## References

- [`update_bgp_peer`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)
- [`BgpPeer`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/bgp_peer.py)
- [`Resource.commit`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
