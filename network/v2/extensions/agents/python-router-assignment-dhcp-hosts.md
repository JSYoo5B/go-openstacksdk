# Python Router Assignment Dhcp Hosts

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--mode", choices=("list", "update"), default="list")
parser.add_argument("--network", default="")
parser.add_argument("--agent", default="")
parser.add_argument("--router", default="")
args = parser.parse_args()
with openstack.connect() as conn:
    if args.mode == "list":
        for row in conn.network.network_hosting_dhcp_agents(args.network, is_alive=True, max_items=20):
            print(json.dumps(row.to_dict(), indent=2))
    else:
        value = conn.network.update_router_in_agent(args.agent, args.router, ha_chassis_priority=0)
        print(json.dumps(value, indent=2))
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents"
)

func printJSON(value any) {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    mode := flag.String("mode", "list", "list or update")
    networkID := flag.String("network", "", "network ID for list")
    agentID := flag.String("agent", "", "L3 agent ID for update")
    routerID := flag.String("router", "", "router ID for update")
    flag.Parse()
    ctx := context.Background()
    conn, err := sdk.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.Network(ctx)
    if err != nil { log.Fatal(err) }
    api := service.API.Agents
    switch *mode {
    case "list":
        for row, err := range api.ListDHCPHostRecords(ctx, *networkID,
            request.Query("is_alive", true), request.MaxItems(20)) {
            if err != nil { log.Fatal(err) }
            printJSON(row.Resource)
        }
    case "update":
        result, err := api.UpdateRouterInAgent(ctx, *agentID, *routerID,
            agents.WithRouterAssignmentPriority(0))
        if err != nil { log.Fatal(err) }
        fmt.Printf("HTTP %d\n", result.StatusCode)
        printJSON(result.Value)
    default:
        log.Fatal("mode must be list or update")
    }
}
```
