# Python Record Find

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--extension", required=True)
parser.add_argument("--strict", action="store_true")
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    value = conn.compute.find_extension(args.extension, ignore_missing=not args.strict)
    print(json.dumps(None if value is None else value.to_dict(), indent=2))
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
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/extensions"
)

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    identity := flag.String("extension", "", "literal alias or exact name")
    strict := flag.Bool("strict", false, "return missing errors")
    flag.Parse()
    if *identity == "" { log.Fatal("-extension is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    compute, err := conn.ComputeV2(ctx)
    if err != nil { log.Fatal(err) }
    value, err := compute.Extensions.FindRecord(ctx, *identity,
        request.IgnoreMissing(!*strict))
    if err != nil { log.Fatal(err) }
    var output any
    if value != nil { output = value.Resource }
    data, err := json.MarshalIndent(output, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

## References

- [find_extension](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L108)
- [Extension](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/extension.py)
- [Install](../../../docs/install.md)
- [extension policy](https://github.com/openstack/nova/blob/31.0.0/nova/policies/extensions.py)
