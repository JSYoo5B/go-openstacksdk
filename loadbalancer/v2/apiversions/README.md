# Octavia API versions

`apiversions.API` wraps Gophercloud v2.15.0 `loadbalancer/v2/apiversions`.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `List` | `GET /` on the service endpoint with the version suffix removed | 200 | Single page; yields `APIVersion{ID, Status}` from the `versions` array; a 204 or empty array yields no rows |

The request goes to the discovery document at the service root, not under the versioned `/v2` path. Non-success responses surface as the native unexpected-status error in the sequence. A pre-canceled context makes no request. There is no extension option on this call.
