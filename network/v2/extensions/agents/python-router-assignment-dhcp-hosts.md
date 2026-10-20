# Python Router Assignment Dhcp Hosts

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/update_router_in_agent`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.UpdateRouterInAgent`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.WithRouterAssignmentPriority`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.WithRouterAssignmentPriorityJSON`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.WithRouterAssignmentOpts`

- One escaped PUT updates the L3 agent/router assignment without preliminary resource GETs. Omitted/null priority sends {}, explicit zero/negative/finite dynamic JSON is retained without a guessed local server range gate; the pinned Proxy direct response.json path returns arbitrary valid JSON even on HTTP error statuses.
  Tests: `network/v2/extensions/agents/python_router_assignment_test.go:TestPythonRouterAssignmentPriorityPayloadAndRawJSONStatus`.
- Physical response bytes/status/headers remain independent of parsed JSON and Clone state; malformed response and accepted body faults retain physical error evidence. Input/headers/options are cloned, invalid configuration/source changes stop before HTTP, and explicit version overrides replace inherited version headers without mutating the original client.
  Tests: `network/v2/extensions/agents/python_router_assignment_test.go:TestPythonRouterAssignmentInvalidJSONAndPhysicalEvidence`, `network/v2/extensions/agents/python_router_assignment_test.go:TestPythonRouterAssignmentOptionsOwnershipAndPreflight`, `network/v2/extensions/agents/python_router_assignment_test.go:TestPythonRouterAssignmentVersionOverrideAndCancellation`.

Python/Go differences:

- Go takes validated literal agent/router string IDs rather than arbitrary Python Resource objects, and returns an owned RouterAssignmentResult with parsed JSON and actual HTTP evidence. A nil error with a 4xx/5xx JSON Value is not successful scheduling confirmation.
- The finite JSON profile preserves the pinned normal SDK Proxy raise_exc=False and Agent method without raise_from_response; nonfinite JSON/other unsupported response encodings fail with physical evidence. Header/microversion options are explicit Go conveniences.
- Go JSON priority options preserve dynamic values; typed integer convenience adds no locally guessed 0..32767 range check. URI traversal/control inputs and external route replacement are rejected; generic mutable Resource/cache/session behavior remains global SDK work.

### `python:network/v2/network_hosting_dhcp_agents`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.ListDHCPHostRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.AllDHCPHostRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.ProxyDHCPHostRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.WithDHCPHostListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.WithDHCPHostListOpts`

- The network-scoped DHCP agent list uses one escaped fixed parent route without parent GET. It inherits the actual Agent query declarations despite the Source subclass comment, applies Agent aliases/bool/int/default descriptors, preserves untyped configuration and unknown Body/Wire values, and does not synthesize undeclared network_id URI state.
  Tests: `network/v2/extensions/agents/python_dhcp_hosts_test.go:TestPythonDHCPHostsInheritedQueriesAndDescriptorViews`.
- Common Source paging applies raw caps before local Body filters, preserves opaque generator JMESPath results and rejects bound/restored parent overrides. Complete list observation exposes late page errors and selected API changes rather than hiding partial results as success.
  Tests: `network/v2/extensions/agents/python_dhcp_hosts_test.go:TestPythonDHCPHostsCapsLocalFiltersProxyAndParentRoute`, `network/v2/extensions/agents/python_dhcp_hosts_test.go:TestPythonDHCPHostsPagingLateFailureAndOwnedOptions`.

Python/Go differences:

- Go accepts finite UTF-8 JSON kwargs and concrete owned header/version/paging options, string parent IDs and independent records instead of arbitrary Resource/session/class objects. Amounts use nonnegative Go integers, and unsupported URI formatters or restored parent-route changes are rejected.
- The SDK retains the common list engine safe-path and paging policies, including empty base_path retaining the selected default rather than Source yielding an empty generator. Source link precedence, local descriptor filters and opaque Proxy results are reused; unsafe/cyclic continuation routes remain terminal.
- The Source NetworkHostingDHCPAgent class declares no network_id URI descriptor; the parent controls the request path but is not inserted into Resource Body. Existing native Agent and reverse DHCP-network listing APIs retain their own reviewed behavior.

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
            agents.WithDHCPHostListQuery("is_alive", true), agents.WithDHCPHostListMaxItems(20)) {
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
