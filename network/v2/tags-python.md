# Tags Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack
from openstack.network.v2.network import Network

conn = openstack.connect(cloud="dev")
network = Network(id="NETWORK_ID")
print(conn.network.get_tags(network))

conn.network.set_tags(network, ["team-a", "production"])
conn.network.add_tags(network, ["reviewed"])
conn.network.add_tag(network, "managed")
conn.network.check_tag(network, "managed")
conn.network.remove_tag(network, "managed")
conn.network.remove_all_tags(network)
```

```sh
go run . -cloud dev -kind network -id NETWORK_ID
go run . -cloud dev -kind network -id NETWORK_ID -operation set -tags team-a,production
go run . -cloud dev -kind network -id NETWORK_ID -operation check -tag managed
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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    kind := flag.String("kind", "network", "network, port, subnet, subnet-pool, router, floating-ip, security-group, security-group-rule, qos-policy, trunk")
    id := flag.String("id", "", "literal resource ID")
    operation := flag.String("operation", "get", "get, set, add-many, add, remove, clear, check")
    list := flag.String("tags", "", "comma-separated tag list; empty clears with set")
    tag := flag.String("tag", "", "individual tag")
    sourceRuleTags := flag.Bool("source-rule-tags", false, "enable Source SGR tag compatibility on a backend that supports it")
    flag.Parse()
    if *id == "" { log.Fatal("-id is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    target := attributestags.NewTaggedResource(attributestags.ResourceKind(*kind), *id)
    tags := []string{}
    if *list != "" { tags = strings.Split(*list, ",") }
    options := []attributestags.TagOption{}
    if *sourceRuleTags { options = append(options, attributestags.WithSourceSecurityGroupRuleTags()) }
    var result *attributestags.TagResult
    switch *operation {
    case "get": result, err = service.AttributeTags.GetTags(ctx, target, options...)
    case "set": result, err = service.AttributeTags.SetTags(ctx, target, tags, options...)
    case "add-many": result, err = service.AttributeTags.AddTags(ctx, target, tags, options...)
    case "add": result, err = service.AttributeTags.AddTag(ctx, target, *tag, options...)
    case "remove": result, err = service.AttributeTags.RemoveTag(ctx, target, *tag, options...)
    case "clear": result, err = service.AttributeTags.RemoveAllTags(ctx, target, options...)
    case "check": result, err = service.AttributeTags.CheckTag(ctx, target, *tag, options...)
    default: log.Fatalf("unknown -operation %q", *operation)
    }
    if err != nil { log.Fatal(err) }
    view := map[string]any{
        "kind": result.Resource.Kind(), "id": result.Resource.ID(),
        "tags": result.Resource.Tags(), "observed_tags": result.ObservedTags,
        "status_code": result.StatusCode,
    }
    data, err := json.MarshalIndent(view, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

## References

- [Neutron 27.0.0 tagging router](https://github.com/openstack/neutron/blob/27.0.0/neutron/extensions/tagging.py)
- [extension middleware](https://github.com/openstack/neutron/blob/27.0.0/neutron/api/extensions.py)
- [Network](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/network.py)
- [QoS](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py)
- [Install](../../docs/install.md)
