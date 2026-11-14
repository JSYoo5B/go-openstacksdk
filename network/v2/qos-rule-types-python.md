# QoS Rule Types Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--operation", choices=["get", "find", "list"], default="list")
parser.add_argument("--name", default="bandwidth_limit")
parser.add_argument("--fields")
parser.add_argument("--type")
parser.add_argument("--all-rules", action="store_true")
parser.add_argument("--all-supported", action="store_true")
parser.add_argument("--max-items", type=int, default=0)
parser.add_argument("--strict", action="store_true")
args = parser.parse_args()
conn = openstack.connect(cloud=args.cloud)
fields = args.fields.split(",") if args.fields else None
if args.operation == "get":
    value = conn.network.get_qos_rule_type(args.name, fields=fields)
    print(value.to_dict())
elif args.operation == "find":
    # The pinned public find method has no **query or fields argument.
    value = conn.network.find_qos_rule_type(args.name, ignore_missing=not args.strict)
    print(None if value is None else value.to_dict())
else:
    query = {}
    if fields:
        query["fields"] = fields
    if args.type is not None:
        query["type"] = args.type
    if args.all_rules:
        query["all_rules"] = True
    if args.all_supported:
        query["all_supported"] = True
    if args.max_items:
        query["max_items"] = args.max_items
    for value in conn.network.qos_rule_types(**query):
        print(value.to_dict())
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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes"
)

func printType(value *ruletypes.QoSRuleTypeRecord) {
    if value == nil { fmt.Println("not found"); return }
    data, err := json.MarshalIndent(value.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    operation := flag.String("operation", "list", "get/find/list")
    name := flag.String("name", "bandwidth_limit", "literal rule type name")
    fields := flag.String("fields", "", "comma-separated fields")
    typeFilter := flag.String("type", "", "server type filter")
    allRules := flag.Bool("all-rules", false, "server all_rules control")
    allSupported := flag.Bool("all-supported", false, "server all_supported control")
    maxItems := flag.Int("max-items", 0, "raw row cap; zero is unlimited")
    strict := flag.Bool("strict", false, "find reports missing")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    api := service.QoSRuleTypes
    var selected []string
    if *fields != "" { selected = strings.Split(*fields, ",") }
    switch *operation {
    case "get":
        value, err := api.GetRecord(ctx, *name, ruletypes.WithRecordGetFields(selected...))
        if err != nil { log.Fatal(err) }
        printType(value)
    case "find":
        value, err := api.FindRecord(ctx, *name, request.IgnoreMissing(!*strict))
        if err != nil { log.Fatal(err) }
        printType(value)
    case "list":
        options := []ruletypes.RecordListOption{request.MaxItems(*maxItems)}
        if len(selected) != 0 { options = append(options, request.Query("fields", selected)) }
        if *typeFilter != "" { options = append(options, request.Query("type", *typeFilter)) }
        if *allRules { options = append(options, request.Query("all_rules", true)) }
        if *allSupported { options = append(options, request.Query("all_supported", true)) }
        for value, err := range api.ListRecords(ctx, options...) {
            if err != nil { log.Fatal(err) }
            printType(value)
        }
    default: log.Fatal("unknown operation")
    }
}
```

## References

- [Policy](https://docs.openstack.org/neutron/2025.2/configuration/policy.html)
- [V2](https://docs.openstack.org/api-ref/network/v2/#qos-rule-types)
- [Install](../../docs/install.md)
- [three proxy declarations](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L5212-L5285)
- [QoSRuleType schema](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_rule_type.py#L16-L42)
- [Resource alternate id](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L654-L672)
