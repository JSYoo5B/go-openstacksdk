# Acl Quota Waits Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import openstack
from openstack.key_manager.v1.secret_acl import SecretACL
from openstack.key_manager.v1.quota import Quota

class PollableSecretACL(SecretACL):
    def fetch(self, session, **kwargs):
        return super().fetch(session, requires_id=False, **kwargs)

class PollableQuota(Quota):
    def fetch(self, session, **kwargs):
        return super().fetch(session, requires_id=False, **kwargs)

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--kind", choices=["acl", "quota"], default="acl")
parser.add_argument("--secret-id")
parser.add_argument("--operation", choices=["status", "delete"], default="status")
parser.add_argument("--attribute", default="acl_ref")
parser.add_argument("--state")
parser.add_argument("--null-state", action="store_true")
parser.add_argument("--interval", type=float, default=2)
parser.add_argument("--timeout", type=float)
parser.add_argument("--correct-singleton-fetch", action="store_true")
args = parser.parse_args()
if args.kind == "acl" and not args.secret_id:
    parser.error("--secret-id is required for ACL")
if args.operation == "status" and args.state is None and not args.null_state:
    parser.error("--state or --null-state is required")
conn = openstack.connect(cloud=args.cloud)
if args.kind == "acl":
    cls = PollableSecretACL if args.correct_singleton_fetch else SecretACL
    res = cls.existing(secret_id=args.secret_id)
else:
    cls = PollableQuota if args.correct_singleton_fetch else Quota
    res = cls.existing()
options = {"interval": args.interval, "callback": lambda progress: print("progress", progress)}
if args.timeout is not None:
    options["wait"] = args.timeout
if args.operation == "delete":
    result = conn.key_manager.wait_for_delete(res, **options)
else:
    state = None if args.null_state else args.state
    result = conn.key_manager.wait_for_status(res, state, attribute=args.attribute, **options)
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
    kind := flag.String("kind", "acl", "acl/quota")
    secretID := flag.String("secret-id", "", "literal secret ID for ACL")
    operation := flag.String("operation", "status", "status/delete")
    attribute := flag.String("attribute", "acl_ref", "declared string/null attribute")
    state := flag.String("state", "", "target string; empty string is explicit")
    nullState := flag.Bool("null-state", false, "target is Source None")
    interval := flag.Duration("interval", 2*time.Second, "polling interval")
    timeout := flag.Duration("timeout", 0, "nonzero override; zero keeps Source default")
    flag.Parse()
    ctx := context.Background()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.KeyManagerV1(ctx)
    if err != nil { log.Fatal(err) }
    var target waits.Target
    switch *kind {
    case "acl": target = waits.SecretACL(*secretID)
    case "quota": target = waits.Quota()
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
    case "delete": result, err = service.Waits.WaitForDelete(ctx, target, options...)
    case "status":
        options = append(options, waits.WithWait(resource.WithStatusAttribute(*attribute)))
        selected := state
        if *nullState { selected = nil }
        result, err = service.Waits.WaitForStatus(ctx, target, selected, options...)
    default: log.Fatal("unknown operation")
    }
    if err != nil { log.Fatal(err) }
    encoded, err := json.MarshalIndent(result.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(encoded))
}
```

## References

- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1275-L1336)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2585-L2715)
- [Install](../../docs/install.md)
- [Waits Python](waits-python.md)
- [Barbican ACL controller](https://github.com/openstack/barbican/blob/21.0.0/barbican/api/controllers/acls.py#L153-L207)
- [effective quota controller](https://github.com/openstack/barbican/blob/21.0.0/barbican/api/controllers/quotas.py#L47-L59)
- [Policy](https://docs.openstack.org/barbican/2025.2/configuration/policy.html)
- [Acls](https://github.com/openstack/barbican/blob/21.0.0/barbican/common/policies/acls.py#L20-L88)
- [Quotas](https://github.com/openstack/barbican/blob/21.0.0/barbican/common/policies/quotas.py#L18-L74)
- [Python Project Quota](waits/python-project-quota.md)
- [Python Project Records](quotas/python-project-records.md)
