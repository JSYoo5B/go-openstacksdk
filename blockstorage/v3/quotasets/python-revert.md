# Reverting Cinder project quota overrides with deletion params

`ProjectQuotaScope.RevertRecord` follows the scalar-project path of pinned
`revert_quota_set(project, **query)`: prepare a quota singleton bound to that
project, send one bodyless DELETE with strict missing behavior, and return
actual response evidence. It does not fetch quota state before or after reset.
Existing `Reset` and the native `Delete` keep their existing 200-only contract.

```python
import openstack

conn = openstack.connect()
conn.block_storage.revert_quota_set(
    "project-id", user_id="user-id", force=False, optional=None,
)
```

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets"
    "github.com/JSYoo5B/go-openstacksdk/resource"
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    provider := &gophercloud.ProviderClient{}
    provider.SetToken(os.Getenv("OS_TOKEN"))
    client := &gophercloud.ServiceClient{
        ProviderClient: provider, Type: "volumev3",
        Endpoint: os.Getenv("OS_VOLUME_ENDPOINT"),
    }
    scope, err := quotasets.New(client).InProject(context.Background(), resource.ID("project-id"))
    if err != nil { panic(err) }
    result, err := scope.RevertRecord(context.Background(),
        quotasets.WithQuotaRevertQuery("user_id", "user-id"),
        quotasets.WithQuotaRevertQuery("force", false),
        quotasets.WithQuotaRevertQuery("optional", nil))
    if err != nil { panic(err) }
    fmt.Println(result.ProjectID, result.StatusCode)
}
```

This Source path uses `urllib.parse.urlencode(params)` without `doseq`, rather
than Requests' ordinary query expansion. Each value becomes one Python string:
null becomes `None`, false becomes `False`, a list becomes `['a', 1, True, None]`,
and an object becomes `{'key': 'value'}`. Empty lists are sent as `[]`; they are
not omitted. Large JSON integers preserve precision. Single-query options retain
insertion order and replacing a key retains its first position; bulk Go maps
receive deterministic sorted key order when copied.

All query names remain literal deletion params. In particular `base_path`,
`microversion`, `headers`, `project_id` and `error_message` in Query do not change
the route, HTTP headers, bound project, negotiated version or error text. The
selected Cinder microversion is preserved without discovery because this quota
class has no maximum. Optional typed Go Headers/Microversion conveniences own
transport policy separately and leave the selected client unchanged.

A clean 404 is an error with `resource.ErrNotFound` and the native status, body,
headers and method preserved. There is no ignore-missing option in this Source
lane. Accepted statuses 100–399 retain opaque raw response bytes without JSON
or quota decoding. Physical read/close/source/cancellation errors preserve actual
proof and cannot become deletion success or trigger a replay. Clean configured
adapter retries preserve the selected URL, query and bodyless request; attempts
to replace that request are rejected.

Python returns None; Go returns `QuotaRevertRecord` containing ProjectID,
Microversion, Envelope, Header and StatusCode. This owned receipt does not model
mutable Python Resource dirty/cache lifecycle, arbitrary Project objects, dynamic
classes or a refreshed authentication scope. Project resolution remains the
existing InProject/CurrentProject contract. The finite JSON/Python-string policy
and pinned shared adapter guards remain the existing SDK boundaries.

Tests use local fixtures. An additional AST probe executed the actual pinned
Proxy.revert_quota_set, Resource._prepare_request and Resource.delete bodies with
explicit stubs to verify params binding, urlencode and bodyless translation; it
is not a full installed Python adapter, real cloud or release validation.
