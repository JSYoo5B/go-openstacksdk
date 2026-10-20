# Bare Metal API versions

`apiversions.API` wraps Gophercloud v2.15.0 `baremetal/apiversions` for Ironic version discovery. Discovery is read-only and independent of the Python openstacksdk Resource layer.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `List` | `GET <endpoint>/` | 200 | Returns `DefaultVersion` and `Versions` with `ID`, `MinVersion`, `Version` and `Status` |
| `Get` | `GET <endpoint>/{version}` | 200 | `Get(ctx, "v1")` unwraps the `version` object |

The routes are relative to the service client endpoint as configured; these calls are meant for the unversioned Ironic root. Any other status, including 202 and 204, is returned as `gophercloud.ErrUnexpectedResponseCode`, and a 404 for an unknown version is not converted to a missing-resource error.

`MinVersion` and `Version` are the lowest and highest supported microversions as reported by the server. Reading them does not select a microversion: when the client `Microversion` is set and the service client type is `baremetal`, it is sent as `X-OpenStack-Ironic-API-Version` on these requests as well, and is absent otherwise. Use the reported range to choose `WithMicroversion` or `WithMicroversionRange` explicitly.

A pre-canceled context makes no request and a deadline stays `context.DeadlineExceeded`.
