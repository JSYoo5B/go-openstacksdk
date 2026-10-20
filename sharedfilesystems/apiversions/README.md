# Manila API versions

`apiversions.API` wraps Gophercloud v2.15.0 `sharedfilesystems/apiversions` for Manila version discovery. Python openstacksdk Resource and workflow equivalence is tracked separately.

Both operations ignore the version path of the service client's endpoint and request the endpoint's unversioned root:

| Operation | Route | Notes |
|---|---|---|
| `List` | `GET /` | Single page of `APIVersion`; discovery documents may answer `300 Multiple Choices`, which is accepted |
| `Get(v)` | `GET /{v}/` | Expects exactly one entry in `versions` |

`Get` returns `ErrVersionNotFound` for an empty `versions` list and `ErrMultipleVersionsFound` (with `Count`) for more than one; a 404 stays an HTTP response error and is not mapped to `resource.ErrNotFound`. `APIVersion` carries `ID`, `Status`, `Version` (maximum microversion), `MinVersion` and `Updated`.

`Find`, `All`, `WaitFor` and the other common `resource.Collection` methods look up by version ID with `Get` (a missing document becomes `resource.ErrNotFound`) and list with `List`. Filters other than `status` are unsupported and yield `resource.ErrUnsupported`. A `Microversion` set on the service client is still sent as `X-OpenStack-Manila-API-Version` for the native Manila client types, which discovery does not need. Cancelling a context fails before any request.
