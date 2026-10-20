# Magnum certificates

`certificates.API` wraps Gophercloud v2.15.0 `containerinfra/v1/certificates` for the `/certificates` routes. Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /certificates` | 201 only | `CSR` is required; exactly one of `ClusterUUID` and `BayUUID` must be set, otherwise the call fails before any request |
| `Get` | `GET /certificates/{cluster}` | 200 | Typed `Certificate` (PEM, CSR, links); unknown response fields are not retained |
| `Update` | `PATCH /certificates/{cluster}` | 202 only | Rotates the CA certificate; sends no request body |

Extension inputs use `WithCreateField`. A field extension that names a core input field, or a nil option, fails with `resource.ErrInvalidOption` before any request. There is no list, delete or common `Collection`; 404 responses stay native HTTP response errors.

`Update` decodes the response body even though the operation returns only an acknowledgement. A 202 with an empty body therefore returns an EOF error although the server accepted the rotation; check the certificate with `Get` before retrying instead of replaying blindly.

The package sends `OpenStack-API-Version: container-infra <version>` only when the `ServiceClient` has a `Microversion`. Cancelling a context stops the local call and does not roll back remote work; mutations are never replayed automatically.
