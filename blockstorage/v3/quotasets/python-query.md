# Python Query

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect()
quota = conn.block_storage.get_quota_set(
    "project-id", usage=True, fields=["volumes", "gigabytes"]
)
print(quota.volumes, quota.usage, quota.reservation)
```

```go
package main

import (
    "context"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    scope, err := conn.BlockStorageProjectQuotas(ctx, resource.ID("project-id"))
    if err != nil { log.Fatal(err) }
    quota, err := scope.GetRecord(ctx,
        quotasets.WithQuotaUsage(true),
        request.Query("fields", []string{"volumes", "gigabytes"}),
    )
    if err != nil { log.Fatal(err) }
    fmt.Println(string(quota.Resource.Body["volumes"]))
    fmt.Println(string(quota.Resource.Body["usage"]), string(quota.Resource.Body["reservation"]))
    fmt.Println(quota.ProjectID, quota.StatusCode)
}
```

## References

- [block storage proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py)
- [Quotas](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/quotas.py)
- [Base](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/base.py)
- [controller](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/contrib/quotas.py)
- [quota class](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/quota_set.py)
- [Quota Set](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/quota_set.py)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
- [Python Query Test](python_query_test.go)
- [Read Test](../../../internal/quotafetch/read_test.go)
