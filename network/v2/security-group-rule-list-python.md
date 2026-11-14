# Security Group Rule List Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import os
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default=os.getenv("OS_CLOUD"))
    parser.add_argument("--group", default="")
    parser.add_argument("--direction", default="")
    parser.add_argument("--name", default="")
    parser.add_argument("--page-size", type=int, default=-1)
    parser.add_argument("--max-items", type=int, default=0)
    parser.add_argument("--single-page", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    query = {"max_items": args.max_items, "paginated": not args.single_page}
    if args.group:
        query["security_group_id"] = args.group
    if args.direction:
        query["direction"] = args.direction
    if args.name:
        query["name"] = args.name
    if args.page_size >= 0:
        query["limit"] = args.page_size
    for rule in conn.network.security_group_rules(**query):
        print(json.dumps(rule.to_dict(), ensure_ascii=False))

if __name__ == "__main__":
    main()
```

```go
package main

import (
	"github.com/JSYoo5B/go-openstacksdk/request"
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "os"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules"
)

func main() {
    cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud")
    group := flag.String("group", "", "server security_group_id filter")
    direction := flag.String("direction", "", "server direction filter")
    name := flag.String("name", "", "local inherited name filter")
    pageSize := flag.Int("page-size", -1, "server limit; -1 omits it")
    maxItems := flag.Int("max-items", 0, "raw cap before local filtering")
    single := flag.Bool("single-page", false, "read first page only")
    flag.Parse()
    if err := run(*cloud, *group, *direction, *name, *pageSize, *maxItems, *single); err != nil { log.Fatal(err) }
}

func run(cloud, group, direction, name string, pageSize, maxItems int, single bool) error {
    if pageSize < -1 || maxItems < 0 { return fmt.Errorf("invalid page-size or max-items") }
    ctx := context.Background()
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    network, err := conn.NetworkV2(ctx)
    if err != nil { return err }
    options := []rules.RecordListOption{
        request.MaxItems(maxItems),
        request.Paginated(!single),
    }
    if pageSize >= 0 { options = append(options, request.Limit(pageSize)) }
    for _, filter := range []struct{ key, value string }{
        {"security_group_id", group}, {"direction", direction}, {"name", name},
    } {
        if filter.value != "" { options = append(options, rules.WithRecordListQuery(filter.key, filter.value)) }
    }
    out := json.NewEncoder(os.Stdout)
    for record, readErr := range network.SecurityRules.ListRecords(ctx, options...) {
        if readErr != nil { return readErr }
        if err := out.Encode(map[string]any{
            "resource": record.Resource.Body,
            "wire": record.Wire.Body,
            "envelope": record.Envelope,
            "headers": record.Header,
            "status_code": record.StatusCode,
        }); err != nil { return err }
    }
    return nil
}
```

## References

- [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L6820-L6842)
- [Source SecurityGroupRule](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/security_group_rule.py)
- [Source Proxy._list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866-L916)
- [Neutron 27.0.0 policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/security_group.py#L228-L247)
- [stock DB](https://github.com/openstack/neutron/blob/27.0.0/neutron/db/securitygroups_db.py#L1021-L1050)
- [controller](https://github.com/openstack/neutron/blob/27.0.0/neutron/api/v2/base.py#L292-L331)
- [Python List Contracts Test](extensions/security/rules/python_list_contracts_test.go)
