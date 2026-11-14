# Autoallocatedtopologies

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect()
validation = conn.network.validate_auto_allocated_topology(project="project-id")
print(validation.id, validation.project_id, validation.dry_run)
topology = conn.network.get_auto_allocated_topology(project="project-id")
print(topology.id)
```

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/JSYoo5B/go-openstacksdk/request"
    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/autoallocatedtopologies"
)

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    api := autoallocatedtopologies.New(service.RawClient())
    project := request.Custom(func(c *autoallocatedtopologies.Options) error { v := "project-id"; c.ProjectID = &v; return nil })
    validation, err := api.ValidateRecord(ctx, project)
    if err != nil { log.Fatal(err) }
    fmt.Println(string(validation.Resource.Body["id"]), string(validation.Resource.Body["dry_run"]))
    topology, err := api.GetRecord(ctx, project)
    if err != nil { log.Fatal(err) }
    fmt.Println(topology.ProjectID, string(topology.Resource.Body["id"]))
}
```

## References

- [network proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)
- [Neutron27 backend](https://github.com/openstack/neutron/blob/27.0.0/neutron/services/auto_allocate/db.py)
- [Auto Allocated Topology](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/auto_allocated_topology.py)
- [Records Test](records_test.go)
