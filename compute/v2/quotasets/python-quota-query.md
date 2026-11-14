# Python Quota Query

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect()
quota = conn.compute.get_quota_set(
    "project-id", usage=True, user_id="user-id", fields=["cores", "ram"]
)
print(quota.cores, quota.usage, quota.reservation)
```

```go
package main

import (
    "context"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    scope, err := conn.ProjectQuotas(ctx, resource.ID("project-id"))
    if err != nil { log.Fatal(err) }
    user, err := scope.InUser(ctx, resource.ID("user-id"))
    if err != nil { log.Fatal(err) }
    quota, err := user.GetRecord(ctx,
        quotasets.WithQuotaUsage(true),
        request.Query("fields", []string{"cores", "ram"}),
    )
    if err != nil { log.Fatal(err) }
    fmt.Println(string(quota.Resource.Body["cores"]))
    fmt.Println(string(quota.Resource.Body["usage"]), string(quota.Resource.Body["reservation"]))
    fmt.Println(quota.ProjectID, quota.UserID, quota.StatusCode)
}
```

## References

- [compute proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py)
- [Quota Sets](https://github.com/openstack/nova/blob/31.0.0/nova/policies/quota_sets.py)
- [base rule](https://github.com/openstack/nova/blob/31.0.0/nova/policies/base.py)
- [controller](https://github.com/openstack/nova/blob/31.0.0/nova/api/openstack/compute/quota_sets.py)
- [SDK Support Ledger](../../../docs/sdk-support-ledger.md)
- [quota class](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/quota_set.py)
- [Quota Set](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/quota_set.py)
- [Python Quota Query Test](python_quota_query_test.go)
- [Read Test](../../../internal/quotafetch/read_test.go)
