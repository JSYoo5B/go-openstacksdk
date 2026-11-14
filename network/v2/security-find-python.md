# Security Find Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
group = conn.network.find_security_group("web", project_id="PROJECT_ID")
rule = conn.network.find_security_group_rule("RULE_ID", security_group_id="GROUP_ID")
strict = conn.network.find_security_group("web", ignore_missing=False)
print(None if group is None else group.to_dict())
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
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func printJSON(value *resource.RawResource) {
    if value == nil { fmt.Println("null"); return }
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    group := flag.String("group", "", "security group exact name or ID")
    rule := flag.String("rule", "", "security group rule exact ID or name")
    project := flag.String("project", "", "optional project_id query")
    groupFilter := flag.String("security-group", "", "optional security_group_id rule query")
    strict := flag.Bool("strict", false, "return an error when missing")
    flag.Parse()
    if *group == "" && *rule == "" { log.Fatal("-group or -rule is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    if *group != "" {
        options := []groups.RecordFindOption{request.IgnoreMissing(!*strict)}
        if *project != "" {
            options = append(options, request.Query("project_id", *project))
        }
        value, err := service.SecurityGroups.FindRecord(ctx, *group, options...)
        if err != nil { log.Fatal(err) }
        if value == nil { printJSON(nil) } else { printJSON(value.Resource) }
    }
    if *rule != "" {
        options := []rules.RecordFindOption{request.IgnoreMissing(!*strict)}
        if *project != "" {
            options = append(options, request.Query("project_id", *project))
        }
        if *groupFilter != "" {
            options = append(options, request.Query("security_group_id", *groupFilter))
        }
        value, err := service.SecurityRules.FindRecord(ctx, *rule, options...)
        if err != nil { log.Fatal(err) }
        if value == nil { printJSON(nil) } else { printJSON(value.Resource) }
    }
}
```

## References

- [Security Group proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L6598-L6623)
- [Rule proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L6768-L6793)
- [Resource.find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2489-L2582)
- [Install](../../docs/install.md)
