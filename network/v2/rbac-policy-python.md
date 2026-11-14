# Rbac Policy Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--action", choices=["list", "get", "find"], default="list")
    parser.add_argument("--id", default="")
    parser.add_argument("--fields", default="")
    parser.add_argument("--object-type", default="network")
    parser.add_argument("--target-project", default="")
    parser.add_argument("--strict", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    fields = args.fields.split(",") if args.fields else None
    if args.action == "get":
        value = conn.network.get_rbac_policy(args.id, fields=fields)
    else:
        query = {"object_type": args.object_type}
        if fields:
            query["fields"] = fields
        if args.target_project:
            query["target_project_id"] = args.target_project
        if args.action == "list":
            for value in conn.network.rbac_policies(**query):
                print(json.dumps(value.to_dict(), indent=2, ensure_ascii=False))
            return
        value = conn.network.find_rbac_policy(
            args.id, ignore_missing=not args.strict, **query)
    if value is None:
        print("not found")
    else:
        print(json.dumps(value.to_dict(), indent=2, ensure_ascii=False))

if __name__ == "__main__":
    main()
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "strings"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies"
)

func printPolicy(value *rbacpolicies.RBACPolicyRecord) {
    if value == nil {
        fmt.Println("not found")
        return
    }
    raw, err := json.MarshalIndent(value.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(raw))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    action := flag.String("action", "list", "list/get/find")
    id := flag.String("id", "", "RBAC policy literal ID")
    fields := flag.String("fields", "", "comma-separated requested fields")
    objectType := flag.String("object-type", "network", "RBAC object type")
    targetProject := flag.String("target-project", "", "target project filter")
    strict := flag.Bool("strict", false, "find reports missing resource")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    network, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    api := rbacpolicies.New(network.RawClient())
    var selected []string
    if *fields != "" { selected = strings.Split(*fields, ",") }
    switch *action {
    case "get":
        value, err := api.GetRecord(ctx, *id,
            rbacpolicies.WithRecordGetFields(selected...))
        if err != nil { log.Fatal(err) }
        printPolicy(value)
    case "list":
        options := []rbacpolicies.RecordListOption{
            request.Query("object_type", *objectType),
        }
        if len(selected) != 0 {
            options = append(options, request.Query("fields", selected))
        }
        if *targetProject != "" {
            options = append(options, request.Query("target_project_id", *targetProject))
        }
        for value, err := range api.ListRecords(ctx, options...) {
            if err != nil { log.Fatal(err) }
            printPolicy(value)
        }
    case "find":
        options := []rbacpolicies.RecordFindOption{
            request.IgnoreMissing(!*strict),
            request.Query("object_type", *objectType),
        }
        if len(selected) != 0 {
            options = append(options, request.Query("fields", selected))
        }
        if *targetProject != "" {
            options = append(options, request.Query("target_project_id", *targetProject))
        }
        value, err := api.FindRecord(ctx, *id, options...)
        if err != nil { log.Fatal(err) }
        printPolicy(value)
    default:
        log.Fatal("unknown action")
    }
}
```

## References

- [RBACPolicy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/rbac_policy.py)
- [proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L5432-L5500)
- [Neutron RBAC API](https://docs.openstack.org/api-ref/network/v2/#rbac-policies)
- [Install](../../docs/install.md)
- [Rbac](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/rbac.py)
- [Python Records Contracts Test](extensions/rbacpolicies/python_records_contracts_test.go)
