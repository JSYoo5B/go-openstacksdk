# Dscp Marking Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--policy", required=True)
    parser.add_argument("--operation", choices=("list", "get", "find"), default="list")
    parser.add_argument("--rule", default="")
    parser.add_argument("--page-size", type=int, default=0)
    parser.add_argument("--max-items", type=int, default=0)
    parser.add_argument("--mark", type=int)
    parser.add_argument("--strict", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect()
    if args.operation == "get":
        result = conn.network.get_qos_dscp_marking_rule(args.rule, args.policy)
        print(result.to_dict())
    elif args.operation == "find":
        result = conn.network.find_qos_dscp_marking_rule(
            args.rule, args.policy, ignore_missing=not args.strict
        )
        print(None if result is None else result.to_dict())
    else:
        query = {"limit": args.page_size, "max_items": args.max_items}
        if args.mark is not None:
            query["dscp_mark"] = args.mark
        for rule in conn.network.qos_dscp_marking_rules(args.policy, **query):
            print(rule.qos_policy_id, rule.to_dict())

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

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func show(record *rules.DSCPMarkingRecord) {
    if record == nil { fmt.Println("missing"); return }
    body, err := json.Marshal(record.Resource.Body)
    if err != nil { log.Fatal(err) }
    fmt.Printf("policy=%s HTTP=%d body=%s\n", record.PolicyID, record.StatusCode, body)
}

func main() {
    policy := flag.String("policy", "", "parent policy ID")
    operation := flag.String("operation", "list", "list, get or find")
    rule := flag.String("rule", "", "literal rule ID or find identity")
    pageSize := flag.Int("page-size", 0, "server page size")
    maxItems := flag.Int("max-items", 0, "raw row cap; zero means no cap")
    mark := flag.Int("mark", -1, "optional local DSCP mark filter")
    strict := flag.Bool("strict", false, "find absence is an error")
    flag.Parse()
    if *policy == "" { log.Fatal("--policy is required") }
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    network, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    switch *operation {
    case "get":
        record, err := network.QoSRules.GetDSCPMarkingRecord(ctx, *policy, *rule)
        if err != nil { log.Fatal(err) }
        show(record)
    case "find":
        record, err := network.QoSRules.FindDSCPMarkingRecord(ctx, *policy, *rule,
            request.IgnoreMissing(!*strict))
        if err != nil { log.Fatal(err) }
        show(record)
    case "list":
        options := []rules.DSCPMarkingListOption{
            request.Limit(*pageSize),
            request.MaxItems(*maxItems),
        }
        if *mark >= 0 {
            options = append(options, request.Query("dscp_mark", *mark))
        }
        for record, err := range network.QoSRules.ListDSCPMarkingRecords(ctx, *policy, options...) {
            if err != nil { log.Fatal(err) }
            show(record)
        }
    default:
        log.Fatal("--operation must be list, get or find")
    }
}
```

## References

- [Source rule](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_dscp_marking_rule.py)
- [Neutron 27.0.0 policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L337-L357)
- [Python Dscp Marking Contracts Test](python_dscp_marking_contracts_test.go)
