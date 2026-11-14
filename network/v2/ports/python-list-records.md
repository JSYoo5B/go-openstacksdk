# Python List Records

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
    parser.add_argument("--network", default="")
    parser.add_argument("--host", default="")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    query = {}
    if args.network:
        query["network_id"] = args.network
    if args.host:
        query["binding_host_id"] = args.host
    for port in conn.network.ports(**query):
        print(json.dumps(port.to_dict(), ensure_ascii=False))

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
    "os"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
)

func main() {
    cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud")
    networkID := flag.String("network", "", "server network_id filter")
    host := flag.String("host", "", "local binding_host_id filter")
    mode := flag.String("mode", "list", "list, proxy, eager or search")
    expression := flag.String("expression", "[*].id", "JMESPath expression")
    name := flag.String("name", "", "search name, ID or glob")
    filters := flag.String("filter", "", "search raw JSON filters")
    flag.Parse()
    if err := run(*cloud, *networkID, *host, *mode, *expression, *name, *filters); err != nil { log.Fatal(err) }
}

func run(cloud, networkID, host, mode, expression, name, filters string) error {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    network, err := conn.NetworkV2(ctx)
    if err != nil { return err }
    api := network.Ports
    var options []ports.RecordListOption
    if networkID != "" { options = append(options, ports.WithRecordListQuery("network_id", networkID)) }
    if host != "" { options = append(options, ports.WithRecordListQuery("binding_host_id", host)) }
    switch mode {
    case "list":
        for row, err := range api.ListRecords(ctx, options...) {
            if err != nil { return err }
            if err := printPort(row); err != nil { return err }
        }
    case "proxy":
        options = append(options, ports.WithRecordListQuery("jmespath_filters", expression))
        result, err := api.ProxyListRecords(ctx, options...)
        if err != nil { return err }
        if generator, ok := result.Value.(*ports.RecordListGenerator); ok {
            for row, err := range generator.Records {
                if err != nil { return err }
                if err := printPort(row); err != nil { return err }
            }
        } else if result.JSON != nil {
            fmt.Println(string(result.JSON))
        } else {
            fmt.Printf("opaque result: %T\n", result.Value)
        }
    case "eager":
        result, err := api.ListRecordValues(ctx, expression, options...)
        if err != nil { return err }
        fmt.Println(string(result.Value))
    case "search":
        var search []ports.RecordSearchOption
        if filters != "" { search = append(search, ports.WithRecordSearchFilters(json.RawMessage(filters))) }
        result, err := api.SearchRecords(ctx, name, search...)
        if err != nil { return err }
        fmt.Println(string(result.Value))
    default:
        return fmt.Errorf("unknown mode %q", mode)
    }
    return nil
}

func printPort(row *ports.PortRecord) error {
    data, err := json.MarshalIndent(row.Resource.Body, "", "  ")
    if err != nil { return err }
    fmt.Println(string(data))
    return nil
}
```

## References

- [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/port.py)
- [Source `Proxy._list`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866)
- [Port](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/port.py#L347)
