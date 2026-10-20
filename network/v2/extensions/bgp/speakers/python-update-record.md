# Python Update Record

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/update_bgp_speaker`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.WithUpdateRecordOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.WithUpdateRecordName`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.WithUpdateRecordAdvertiseFloatingIPHostRoutes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.WithUpdateRecordAdvertiseTenantNetworks`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.WithUpdateRecordField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.WithUpdateRecordHeader`
- `github.com/JSYoo5B/go-openstacksdk/request.Null`
- `github.com/JSYoo5B/go-openstacksdk/request.Present`

- UpdateRecord sends PUT bgp-speakers/{id} with only the supplied attributes in the bgp_speaker envelope, so a name-only update omits both advertise flags, while explicit false, null, an empty name and unknown body attributes through WithUpdateRecordField are sent; it accepts any status below 400, decodes enveloped and flat responses, and sends nothing when no attribute is supplied.
  Tests: `network/v2/extensions/bgp/speakers/python_update_record_test.go:TestPythonBGPSpeakerUpdateRecordSendsOnlySuppliedFlags`.
- Extension fields that collide with typed keys or the id identity, nil options and a blank ID fail before HTTP, and a 400 response is returned as the Gophercloud status error wrapped in resource.OperationError.
  Tests: `network/v2/extensions/bgp/speakers/python_update_record_test.go:TestPythonBGPSpeakerUpdateRecordRejectsInputsAndErrorStatuses`.

Python/Go differences:

- Python accepts a BgpSpeaker Resource or an ID string; Go takes the ID string and rejects blank, '.' and '..' IDs before HTTP.
- Python silently discards an id attribute from the commit; WithUpdateRecordField rejects the id key before HTTP.
- Python returns the Resource merged from request attributes and the response; Go returns BGPSpeakerRecord with the native typed decode, the raw response object fields, headers and status, and Sent false when no request was needed.

## References

- [`update_bgp_speaker`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)
- [`BgpSpeaker`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/bgp_speaker.py)
- [`Resource.commit`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
