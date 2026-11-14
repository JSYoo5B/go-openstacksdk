# Find Core Resources Python

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
    parser.add_argument("--network", default="app-net")
    parser.add_argument("--port", default="")
    parser.add_argument("--subnet", default="")
    parser.add_argument("--project", default="")
    parser.add_argument("--strict", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    query = {"project_id": args.project} if args.project else {}
    finders = [
        ("network", args.network, conn.network.find_network),
        ("port", args.port, conn.network.find_port),
        ("subnet", args.subnet, conn.network.find_subnet),
    ]
    for kind, identity, find in finders:
        if not identity:
            continue
        result = find(identity, ignore_missing=not args.strict, **query)
        print(kind)
        print(json.dumps(result.to_dict() if result is not None else None,
                         indent=2, ensure_ascii=False))

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
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/subnets"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func printResource(kind string, value *resource.RawResource) {
    fmt.Println(kind)
    if value == nil { fmt.Println("null"); return }
    body, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(body))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    networkName := flag.String("network", "app-net", "network ID or exact name")
    portName := flag.String("port", "", "optional port ID or exact name")
    subnetName := flag.String("subnet", "", "optional subnet ID or exact name")
    project := flag.String("project", "", "optional project filter")
    strict := flag.Bool("strict", false, "fail on logical absence")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    if *networkName != "" {
        options := []networks.RecordFindOption{request.IgnoreMissing(!*strict)}
        if *project != "" { options = append(options, networks.WithRecordFindQuery("project_id", *project)) }
        result, err := service.Networks.FindRecord(ctx, *networkName, options...)
        if err != nil { log.Fatal(err) }
        if result == nil { printResource("network", nil) } else { printResource("network", result.Resource) }
    }
    if *portName != "" {
        options := []ports.RecordFindOption{request.IgnoreMissing(!*strict)}
        if *project != "" { options = append(options, ports.WithRecordFindQuery("project_id", *project)) }
        result, err := service.Ports.FindRecord(ctx, *portName, options...)
        if err != nil { log.Fatal(err) }
        if result == nil { printResource("port", nil) } else { printResource("port", result.Resource) }
    }
    if *subnetName != "" {
        options := []subnets.RecordFindOption{request.IgnoreMissing(!*strict)}
        if *project != "" { options = append(options, subnets.WithRecordFindQuery("project_id", *project)) }
        result, err := service.Subnets.FindRecord(ctx, *subnetName, options...)
        if err != nil { log.Fatal(err) }
        if result == nil { printResource("subnet", nil) } else { printResource("subnet", result.Resource) }
    }
}
```

```python
port = conn.network.find_port(
    "app-port", binding_profile={"nested": {"enabled": True}})
```

```go
port, err := service.Ports.FindRecord(ctx, "app-port",
    ports.WithRecordFindQuery("binding_profile",
        map[string]any{"nested": map[string]any{"enabled": true}}))
```

## References

- [Install](../../docs/install.md)
- [Network find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L3214)
- [Port find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L3950)
- [Subnet find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L7487)
- [Resource.find/list/fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
- [NetworkResource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_base.py)
- [Network](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/network.py)
- [Port](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/port.py)
- [Subnet](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/subnet.py)
