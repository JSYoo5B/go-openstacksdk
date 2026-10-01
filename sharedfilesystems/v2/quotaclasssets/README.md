# Manila quota classes

This SDK-owned package adds the pinned Python quota-class operations that are
absent from Gophercloud v2.15.0. A class is identified by its exact name; it is
separate from project, user and share-type quota scopes.

| openstacksdk | Go SDK |
| --- | --- |
| `shared_file_system.get_quota_class_set("default")` | `classes.InClass(ctx, "default")`, then `scope.Get(ctx)` |
| `shared_file_system.update_quota_class_set("default", shares=0)` | `scope.Update(ctx, quotaclasssets.UpdateOpts{Shares: &zero})` |

```python
quota = conn.shared_file_system.get_quota_class_set("default")
quota = conn.shared_file_system.update_quota_class_set("default", shares=0)
```

```go
import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "gophercloudsdk/sharedfilesystems/v2/quotaclasssets"
)

func updateQuotaClass(ctx context.Context, client *gophercloud.ServiceClient) error {
    scope, err := quotaclasssets.New(client).InClass(ctx, "default")
    if err != nil {
        return err
    }
    zero := int64(0)
    _, err = scope.Update(ctx, quotaclasssets.UpdateOpts{Shares: &zero})
    return err
}
```

`InClass` performs no HTTP or Keystone lookup. A class name must be one unescaped
URL path segment. A UUID-shaped class name remains a class name. The catalog
endpoint's project component is retained without being interpreted as a quota
target. Omitted microversion or API 2.6 and earlier uses `os-quota-class-sets`;
2.7+ uses `quota-class-sets`. `latest` delegates version negotiation to the server.
Configure the service client's microversion before sharing it across callers;
the SDK does not upgrade it automatically.

The scope exposes only `Get` and `Update`. Manila creates a missing named class
through PUT if its policy permits; a separate create, delete, reset, list,
defaults or wait operation is not part of this API. Missing class fields are
filled by server defaults; the SDK does not synthesize quota defaults.

`UpdateOpts` contains twelve pinned typed limits with exact `int64` pointers.
Nil omits a field, zero sends zero, and -1 means unlimited; values below -1 fail
before HTTP. The class input has no project/user selector or `force` field.
`WithQuotaOptions` snapshots typed pointers and nested `Extra` JSON at option
construction; `WithUpdateField` captures an extension field without a builder.
Extensions cannot replace declared limits, fixed class names, scope selectors
or response-only metadata. The server validates unknown extension fields and
the microversions of individual newer limits.

`QuotaClassResource` keeps the requested `ClassName` separate from the server's
`ID`, plus headers, status code and all `quota_class_set` fields as raw JSON.
`QuotaClassSet` shares the project quota response's typed limit data model,
without sharing project scope methods. Raw JSON distinguishes null from
omission and preserves unknown fields and large nested integers exactly.

GET and PUT require HTTP 200. HTTP, decoding and context causes remain available
through `errors.Is`/`errors.As`; HTTP 404 also matches `resource.ErrNotFound`.
Fixed method, path, query and origin are retained across redirects, retries and
reauthentication.

Sources: pinned openstacksdk
[`QuotaClassSet`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/shared_file_system/v2/quota_class_set.py),
[`Proxy`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/shared_file_system/v2/_proxy.py),
official [Manila quota class API](https://docs.openstack.org/api-ref/shared-file-system/#quota-class-set),
and [API version history](https://docs.openstack.org/manila/latest/contributor/api_microversion_history.html).
