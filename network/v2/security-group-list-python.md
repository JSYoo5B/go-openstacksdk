# Security Group List Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument('--cloud', required=True)
parser.add_argument('--name')
parser.add_argument('--limit', type=int)
parser.add_argument('--max-items', type=int, default=0)
args = parser.parse_args()
conn = openstack.connect(cloud=args.cloud)
query = {'max_items': args.max_items}
if args.name is not None:
    query['name'] = args.name
if args.limit is not None:
    query['limit'] = args.limit
for group in conn.network.security_groups(**query):
    print(json.dumps(group.to_dict(), ensure_ascii=False))
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
    "os"
    "time"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups"
)

func main() {
    cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud")
    name := flag.String("name", "", "optional server name filter")
    limit := flag.Int("limit", -1, "page limit; -1 omits it")
    cap := flag.Int("max-items", 0, "raw row cap; zero is unlimited")
    paginated := flag.Bool("paginated", true, "follow all retained pages")
    flag.Parse()
    if err := run(*cloud, *name, *limit, *cap, *paginated); err != nil {
        log.Fatal(err)
    }
}

func run(cloud, name string, limit, maxItems int, paginated bool) error {
    if limit < -1 { return fmt.Errorf("-limit must be -1 or nonnegative") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    network, err := conn.NetworkV2(ctx)
    if err != nil { return err }
    options := []groups.RecordListOption{
        request.MaxItems(maxItems),
        request.Paginated(paginated),
        groups.WithRecordListHeaders(map[string]string{"X-Request-Source": "security-group-list-example"}),
    }
    if name != "" { options = append(options, groups.WithRecordListQuery("name", name)) }
    if limit >= 0 { options = append(options, request.Limit(limit)) }
    out := json.NewEncoder(os.Stdout)
    for record, readErr := range network.SecurityGroups.ListRecords(ctx, options...) {
        if readErr != nil { return readErr }
        if err := out.Encode(map[string]any{
            "resource": record.Resource.Body,
            "wire": record.Wire.Body,
            "envelope": record.Envelope,
            "headers": record.Header,
            "status_code": record.StatusCode,
        }); err != nil { return err }
    }
    return nil
}
```

## References

- [`ef55d7d`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L6647-L6665)
- [`SecurityGroup`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/security_group.py)
- [Security Group](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/security_group.py#L125-L153)
- [extension](https://github.com/openstack/neutron/blob/27.0.0/neutron/extensions/securitygroup.py)
- [stock Controller._items](https://github.com/openstack/neutron/blob/27.0.0/neutron/api/v2/base.py#L292-L378)
- [Python List Contracts Test](extensions/security/groups/python_list_contracts_test.go)
