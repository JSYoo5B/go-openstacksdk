# Python Update Record

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/update_vpn_ipsec_site_connection`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.WithUpdateRecordOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.WithUpdateRecordField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.WithUpdateRecordHeader`
- `github.com/JSYoo5B/go-openstacksdk/request.Null`
- `github.com/JSYoo5B/go-openstacksdk/request.Present`

- UpdateRecord sends PUT vpn/ipsec-site-connections/{id} with only the supplied attributes in the ipsec_site_connection envelope, including empty local_id, peer_address, peer_id, psk, initiator and names, an empty peer_cidrs list, mtu 0, an empty dpd object, admin_state_up false, a null endpoint group ID and other declared attributes through WithUpdateRecordField, accepts any status below 400, decodes enveloped and flat responses, and sends nothing when no attribute is supplied.
  Tests: `network/v2/extensions/vpnaas/siteconnections/python_update_record_test.go:TestPythonSiteConnectionUpdateRecordSendsEmptyValues`.
- Extension fields that collide with typed keys or the id identity, nil options and blank or dot IDs fail before HTTP, and a 404 response is returned as the Gophercloud status error wrapped in resource.OperationError.
  Tests: `network/v2/extensions/vpnaas/siteconnections/python_update_record_test.go:TestPythonSiteConnectionUpdateRecordRejectsInputsAndErrorStatuses`.

Python/Go differences:

- Python accepts a VpnIPSecSiteConnection Resource or an ID string; Go takes the ID string and rejects blank, '.' and '..' IDs before HTTP.
- Python takes is_admin_state_up as the keyword name; Go uses the AdminStateUp field that serializes as admin_state_up.
- Python drops undeclared keyword attributes because VpnIPSecSiteConnection does not allow unknown body attributes; WithUpdateRecordField sends any extension key except the typed keys and id.
- Python converts values of typed attributes such as mtu before sending; Go serializes the Go value as given.
- Python returns the Resource merged from request attributes and the response; Go returns ConnectionRecord with the native typed decode, the raw response object fields, headers and status, and Sent false when no request was needed.

## References

- [`update_vpn_ipsec_site_connection`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)
- [`VpnIPSecSiteConnection`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/vpn_ipsec_site_connection.py)
- [`Resource.commit`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
