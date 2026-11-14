# Python Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--server", required=True)
parser.add_argument("--changes-since", default=None)
args = parser.parse_args()
query = {} if args.changes_since is None else {"changes_since": args.changes_since}
with openstack.connect(cloud=args.cloud) as conn:
    for action in conn.compute.server_actions(args.server, **query):
        print(json.dumps(action.to_dict(), indent=2))
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/instanceactions"
)

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    server := flag.String("server", "", "literal server UUID")
    since := flag.String("changes-since", "", "optional ISO timestamp")
    flag.Parse()
    if *server == "" { log.Fatal("-server is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    compute, err := conn.ComputeV2(ctx)
    if err != nil { log.Fatal(err) }
    var options []instanceactions.RecordListOption
    if *since != "" {
        options = append(options, request.Query("changes_since", *since))
    }
    for value, err := range compute.InstanceActions.ListRecords(ctx, *server, options...) {
        if err != nil { log.Fatal(err) }
        data, err := json.MarshalIndent(value.Resource, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Println(string(data))
    }
}
```

## References

- [server_actions](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L3221)
- [ServerAction·ServerActionEvent](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server_action.py)
- [Install](../../../docs/install.md)
- [instance action policy](https://github.com/openstack/nova/blob/31.0.0/nova/policies/instance_actions.py)
- [controller](https://github.com/openstack/nova/blob/31.0.0/nova/api/openstack/compute/instance_actions.py)
- [schema](https://github.com/openstack/nova/blob/31.0.0/nova/api/openstack/compute/schemas/instance_actions.py)
