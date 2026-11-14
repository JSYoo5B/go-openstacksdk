# Waits Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import openstack
from openstack.key_manager.v1.container import Container
from openstack.key_manager.v1.order import Order
from openstack.key_manager.v1.secret import Secret

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--kind", choices=["container", "order", "secret"], default="order")
parser.add_argument("--id", required=True)
parser.add_argument("--operation", choices=["status", "delete"], default="status")
parser.add_argument("--status", default="ACTIVE")
parser.add_argument("--null-status", action="store_true")
parser.add_argument("--interval", type=float, default=2)
parser.add_argument("--timeout", type=float)
parser.add_argument("--disable-failures", action="store_true")
args = parser.parse_args()
conn = openstack.connect(cloud=args.cloud)
resource_type = {"container": Container, "order": Order, "secret": Secret}[args.kind]
res = resource_type.existing(id=args.id)
options = {"interval": args.interval, "callback": lambda progress: print("progress", progress)}
if args.timeout is not None:
    options["wait"] = args.timeout
if args.operation == "delete":
    result = conn.key_manager.wait_for_delete(res, **options)
else:
    if args.disable_failures:
        options["failures"] = []
    status = None if args.null_status else args.status
    result = conn.key_manager.wait_for_status(res, status, **options)
print(result.to_dict())
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
    "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    kind := flag.String("kind", "order", "container/order/secret")
    id := flag.String("id", "", "literal executable ID")
    operation := flag.String("operation", "status", "status/delete")
    status := flag.String("status", "ACTIVE", "target state")
    nullStatus := flag.Bool("null-status", false, "target is Source None")
    interval := flag.Duration("interval", 2*time.Second, "polling interval")
    timeout := flag.Duration("timeout", 0, "positive override; zero keeps Source default")
    disableFailures := flag.Bool("disable-failures", false, "disable status failure states")
    flag.Parse()
    ctx := context.Background()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.KeyManagerV1(ctx)
    if err != nil { log.Fatal(err) }
    api := waits.New(service.RawClient())
    var target waits.Target
    switch *kind {
    case "container": target = waits.Container(*id)
    case "order": target = waits.Order(*id)
    case "secret": target = waits.Secret(*id)
    default: log.Fatal("unknown kind")
    }
    options := []waits.Option{
        waits.WithInterval(*interval),
        waits.WithWait(resource.WithProgressCallback(func(progress int) error {
            fmt.Println("progress", progress)
            return nil
        })),
    }
    if *timeout != 0 { options = append(options, waits.WithTimeout(*timeout)) }
    var result *waits.Record
    switch *operation {
    case "delete": result, err = api.WaitForDelete(ctx, target, options...)
    case "status":
        if *disableFailures { options = append(options, waits.WithWait(resource.WithFailureStates())) }
        selected := status
        if *nullStatus { selected = nil }
        result, err = api.WaitForStatus(ctx, target, selected, options...)
    default: log.Fatal("unknown operation")
    }
    if err != nil { log.Fatal(err) }
    data, err := json.MarshalIndent(result.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

## References

- [Install](../../docs/install.md)
- [Policy](https://docs.openstack.org/barbican/2025.2/configuration/policy.html)
- [ProjectQuota](waits/python-project-quota.md)
- [proxy wait declarations](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L615-L674)
- [shared status/deletion wait](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2585-L2715)
- [Secret.fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/secret.py#L91-L138)
- [timeout iterator](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53-L100)
