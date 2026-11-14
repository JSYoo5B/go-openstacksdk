# Python Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="devstack")
    parser.add_argument("--find", default="")
    parser.add_argument("--network-name", default="")
    args = parser.parse_args()
    with openstack.connect(cloud=args.cloud) as conn:
        if args.find:
            result = conn.network.find_network_ip_availability(args.find, ignore_missing=False)
            print(json.dumps(result.to_dict(), ensure_ascii=False))
        else:
            query = {"network_name": args.network_name} if args.network_name else {}
            for record in conn.network.network_ip_availabilities(**query):
                print(json.dumps(record.to_dict(), ensure_ascii=False))

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
    "os"
    "time"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities"
)

func run(ctx context.Context, cloud, identity, name string) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.NetworkV2(ctx)
    if err != nil { return err }
    api := service.NetworkIPAvailabilities
    if identity != "" {
        record, err := api.FindRecord(ctx, identity, request.IgnoreMissing(false))
        if err != nil { return err }
        return json.NewEncoder(os.Stdout).Encode(record.Resource.Body)
    }
    var options []networkipavailabilities.RecordListOption
    if name != "" { options = append(options, request.Query("network_name", name)) }
    for record, err := range api.ListRecords(ctx, options...) {
        if err != nil { return err }
        if err := json.NewEncoder(os.Stdout).Encode(record.Resource.Body); err != nil { return err }
    }
    return nil
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    identity := flag.String("find", "", "literal member ID or Source fallback id/name")
    name := flag.String("network-name", "", "server-side network name filter")
    timeout := flag.Duration("timeout", time.Minute, "whole operation deadline")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), *timeout)
    defer cancel()
    if err := run(ctx, *cloud, *identity, *name); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}
```

```sh
python main.py --cloud devstack --network-name public
go run main.go --cloud devstack --network-name public
python main.py --cloud devstack --find n1
go run main.go --cloud devstack --find n1
```
