# Cinder limits singleton

The SDK adds an extension-preserving read layer and fixed project scopes around
Cinder's read-only limits endpoint. Generated `API.Get(ctx)` still has its
original Gophercloud v2.15.0 signature and native result model.

| openstacksdk | Go SDK |
| --- | --- |
| `block_storage.get_limits()` | `service.Limits.Fetch(ctx)` |
| `block_storage.get_limits(project)` | `service.Limits.InProject(ctx, ref)`, then `scope.Get(ctx)` |
| `limits.absolute.max_total_volumes` | `result.Absolute.MaxTotalVolumes` (`*int64`) |
| `limits.rate[i].limits[j].next_available` | `result.Rate[i].Limits[j].NextAvailable` (raw JSON) |

```python
current = conn.block_storage.get_limits()
project = conn.block_storage.get_limits("project-id")
```

```go
import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/gophercloudsdk/blockstorage/v3/limits"
    "github.com/JSYoo5B/gophercloudsdk/resource"
)

func readLimits(ctx context.Context, client *gophercloud.ServiceClient) error {
    api := limits.New(client)
    current, err := api.Fetch(ctx)
    if err != nil {
        return err
    }
    _ = current.ProjectID // empty: the server chooses its implicit current project
    scope, err := api.InProject(ctx, resource.ID("project-id")) // Cinder 3.39+
    if err != nil {
        return err
    }
    result, err := scope.Get(ctx)
    if err != nil {
        return err
    }
    _ = result.Absolute.MaxTotalVolumes // nil distinguishes absent/null from zero
    return nil
}
```

Cinder added the admin-only `project_id` filter in API 3.39. An omitted version
uses 3.0 behavior: unfiltered `Fetch` works, while `InProject`, `CurrentProject`
and filtered `Fetch` fail before a Keystone lookup or Cinder HTTP. A selected
numeric 3.39+ version permits the filter. Symbolic `latest` remains valid for
unfiltered `Fetch`, but fails project-filter preflight because a deployment's
maximum may be below 3.39. Connection microversion range discovery can select a
numeric version before binding the scope; the SDK does not upgrade it here.
The pinned Python resource uses 3.39 as the negotiation ceiling only when
`session.default_microversion` is absent. An explicit session default is returned
without that cap. This Go scope retains the selected service version and keeps
its own numeric 3.39+ project-filter preflight. The separate
[cloud GetVolumeLimits helper](../../volume-limits.md) resolves Identity projects
first and does not add that minimum-version preflight.

Cinder silently ignores `project_id` for non-admin callers, even at 3.39+,
and returns the authenticated project's limits. The response has no project
identity to verify. `LimitsResource.ProjectID` records the requested filter;
it does not confirm which project's data the server returned. Cross-project
reads therefore require an admin context. The SDK does not infer server admin
policy from local role names.

`WithGetOptions(limits.GetOpts{ProjectID: "project-id"})` also filters `Fetch`.
`WithGetQuery` adds query extension fields using the same concrete request
configuration as other SDK operations. An explicit empty, invalid, repeated or
conflicting `project_id` fails before HTTP. Fixed scopes reject a replacement
project option, including the same ID supplied again. `tenant_id` is Nova's wire
selector and is rejected here. Extensions cannot change the fixed project.

Explicit project IDs require no lookup. Names require the separate Keystone v3
client provided with `WithIdentityClient`; lookup uses the shared exact-name and
ambiguity policy and resolves only once. `CurrentProject` reads the recorded
Keystone v2/v3 result without refreshing auth or guessing a project from an
endpoint. Manual tokens, domain and system scopes need an explicit ID.

The service client's catalog endpoint is retained independently of the target
`project_id` query. Source `MoreHeaders` cannot override a selected version with
a different Cinder version. Matching overrides are accepted case-insensitively
by header name. A selected microversion requires a recognized Cinder client type,
or a blank type with an explicit matching Cinder version header, so a 3.39
preflight cannot succeed while the actual request sends no Cinder version.
The general version header uses `volume VERSION`, as Gophercloud's Cinder client
does. Configure clients before sharing them between concurrent callers.

`LimitsResource` uses SDK-owned `AbsoluteLimit`, `RateLimits` and `RateLimit`
models. Absolute quotas, rate values and remaining counts use optional exact
`int64` pointers, including zero and null values. This differs from the native
`int` fields returned by generated `Get`. `NextAvailable` retains raw JSON:
numeric timestamps, strings, null and omission are preserved without imposing
the native string-only representation. `Body`, `AbsoluteBody` and each nested
rate `Body` retain extensions and distinguish missing values from JSON null.

Successful HTTP 200 responses retain headers and status code. If JSON decoding,
body reading or cancellation after response headers fails,
`LimitsResponseError` retains the accepted response's complete or partial raw
body, header and status. Its `Unwrap` keeps the original cause available through
`errors.Is`/`errors.As`. Other status codes retain native HTTP causes; 404 also
matches `resource.ErrNotFound`. Fixed method/path/query/origin guards prevent
redirects or retries from dropping the project selector, and shared provider
reauthentication retains the same target.

Limits are a singleton read, not a collection. The scope has no update, delete,
list, find or wait methods.

Sources: pinned openstacksdk
[`Limits`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/limits.py),
[`get_limits`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py),
official [Cinder API version history](https://docs.openstack.org/api-ref/block-storage/api_microversion_history.html),
and [server project-filter behavior](https://github.com/openstack/cinder/blob/master/cinder/api/v3/limits.py).
