# Manila scheduler statistics

`schedulerstats.API` wraps Gophercloud v2.15.0 `sharedfilesystems/v2/schedulerstats` with two read-only administrative operations. Policy normally restricts them to administrators; a denial surfaces as an HTTP response error. Python openstacksdk Resource and workflow equivalence is tracked separately. The service client's `Microversion` is sent as `X-OpenStack-Manila-API-Version` for the native Manila client types.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `List` | `GET /scheduler-stats/pools` | 200 | Pool names, host, backend and pool |
| `ListDetail` | `GET /scheduler-stats/pools/detail` | 200 | Adds `Capabilities` |

Both return a single page and a 204 response yields no items. Capacity fields (`TotalCapacityGB`, `FreeCapacityGB`, `AllocatedCapacityGB`) are float64: the string `"infinite"` decodes to positive infinity and missing, null or other values decode to zero, so zero does not prove the backend reported zero.

Known upstream limitation: the pinned Gophercloud `ListOpts` and `ListDetailOpts` fields are tagged for JSON rather than query encoding, so typed filters (`PoolName`, `HostName`, `BackendName`, `Capabilities`, `ShareType`, `ProjectID`) are not sent. Use `WithListQuery` or `WithListDetailQuery` with the Manila query names `pool`, `host`, `backend`, `capabilities`, `share_type` instead. Nil options fail with `resource.ErrInvalidOption` before any request.

Cancelling a context stops the local call.
