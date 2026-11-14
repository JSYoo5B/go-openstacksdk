# Trunk Address Find Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import json
import openstack

with openstack.connect(cloud="devstack") as conn:
    trunk = conn.network.find_trunk(
        "application-trunk", ignore_missing=False,
        project_id="project-id", is_admin_state_up=False,
        fields=["id", "name", "admin_state_up"],
    )
    address_group = conn.network.find_address_group(
        "application-addresses", ignore_missing=False,
        project_id="project-id", addresses=["10.0.0.0/24"],
    )
    print(json.dumps(trunk.to_dict(), indent=2))
    print(json.dumps(address_group.to_dict(), indent=2))
```

```go
package main

import (
	"github.com/JSYoo5B/go-openstacksdk/request"
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func run(ctx context.Context, cloud, kind, identity, project string) (*resource.RawResource, error) {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return nil, err }
    network, err := conn.NetworkV2(ctx)
    if err != nil { return nil, err }
    switch kind {
    case "trunk":
        options := []trunks.RecordFindOption{
            request.IgnoreMissing(false),
            trunks.WithRecordFindQuery("is_admin_state_up", false),
        }
        if project != "" { options = append(options, trunks.WithRecordFindQuery("project_id", project)) }
        record, err := network.Trunks.FindRecord(ctx, identity, options...)
        if err != nil || record == nil { return nil, err }
        return record.Resource, nil
    case "address-group":
        options := []addressgroups.RecordFindOption{
            request.IgnoreMissing(false),
        }
        if project != "" { options = append(options, addressgroups.WithRecordFindQuery("project_id", project)) }
        record, err := network.SecurityAddressGroups.FindRecord(ctx, identity, options...)
        if err != nil || record == nil { return nil, err }
        return record.Resource, nil
    default:
        return nil, fmt.Errorf("unknown kind %q", kind)
    }
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    kind := flag.String("kind", "trunk", "trunk or address-group")
    identity := flag.String("identity", "application-trunk", "exact ID or name")
    project := flag.String("project", "", "optional project filter")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    value, err := run(ctx, *cloud, *kind, *identity, *project)
    if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(value); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}
```
