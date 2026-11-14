# Rbac Policy Mutations Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--mode", choices=("create", "update", "delete"), required=True)
parser.add_argument("--policy", default="")
parser.add_argument("--object", default="")
parser.add_argument("--target-project", default="")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "create":
        value = conn.network.create_rbac_policy(
            object_type="network", object_id=args.object,
            action="access_as_shared", target_project_id=args.target_project,
        )
        print(json.dumps(value.to_dict(), indent=2))
    elif args.mode == "update":
        value = conn.network.update_rbac_policy(
            args.policy, target_project_id=args.target_project
        )
        print(json.dumps(value.to_dict(), indent=2))
    else:
        conn.network.delete_rbac_policy(args.policy)
        print(json.dumps({"deleted": args.policy}, indent=2))
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies"
)

func run(ctx context.Context, cloud, mode, policyID, objectID, target string) (any, error) {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return nil, err }
    network, err := conn.NetworkV2(ctx)
    if err != nil { return nil, err }
    switch mode {
    case "create":
        value, err := network.RBACPolicies.CreateRecord(ctx,
            request.Field("object_type", "network"),
            request.Field("object_id", objectID),
            request.Field("action", "access_as_shared"),
            request.Field("target_project_id", target),
        )
        if err != nil { return nil, err }
        return value.Resource, nil
    case "update":
        value, err := network.RBACPolicies.UpdateRecord(ctx, policyID,
            request.Field("target_project_id", target),
        )
        if err != nil { return nil, err }
        return value.Resource, nil
    case "delete":
        return network.RBACPolicies.DeleteRecord(ctx, policyID)
    default:
        return nil, fmt.Errorf("mode must be create, update or delete")
    }
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    mode := flag.String("mode", "", "create, update or delete")
    policyID := flag.String("policy", "", "literal RBAC policy ID")
    objectID := flag.String("object", "", "network ID for create")
    target := flag.String("target-project", "", "concrete target project ID")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    value, err := run(ctx, *cloud, *mode, *policyID, *objectID, *target)
    if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(value); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}
```

## References

- [Install](../../docs/install.md)
- [Rbac](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/rbac.py#L89)
- [owner base](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py#L32)
- [Stock RBAC schema](https://github.com/openstack/neutron/blob/27.0.0/neutron/extensions/rbac.py#L60)
