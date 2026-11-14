# Cloud Router Interfaces Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import os
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default=os.getenv("OS_CLOUD"))
parser.add_argument("--router", required=True, help='JSON dictionary, e.g. {"id":"router-id"}')
parser.add_argument("--interface-type", default="null", help='JSON None/falsey or "internal"/"external"')
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    ports = conn.list_router_interfaces(json.loads(args.router),
                                        json.loads(args.interface_type))
    print(json.dumps([value.to_dict() for value in ports], ensure_ascii=False, indent=2))
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
 "time"
 sdk "github.com/JSYoo5B/go-openstacksdk"
 "github.com/JSYoo5B/go-openstacksdk/network"
)
func main(){
 cloud:=flag.String("cloud",os.Getenv("OS_CLOUD"),"clouds.yaml cloud")
 router:=flag.String("router","","Source router dictionary JSON")
 kind:=flag.String("interface-type","null","JSON discriminator")
 flag.Parse();if *router==""{log.Fatal("-router required")}
 ctx,cancel:=context.WithTimeout(context.Background(),30*time.Second);defer cancel()
 var connect []sdk.ConnectionOption;if *cloud!=""{connect=append(connect,sdk.WithCloud(*cloud))}
 conn,err:=sdk.Connect(ctx,connect...);if err!=nil{log.Fatal(err)}
 result,err:=conn.ListRouterInterfaces(ctx,json.RawMessage(*router),network.WithRouterInterfaceTypeJSON(json.RawMessage(*kind)))
 if err!=nil{if result!=nil{fmt.Fprintf(os.Stderr,"observed ports=%d\n",len(result.Inventory))};log.Fatal(err)}
 fmt.Println(string(result.Value))
}
```

## References

- [list_router_interfaces Source](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2205)
- [Port](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/port.py)
- [Cloud Router Interfaces Test](cloud_router_interfaces_test.go)
- [Connection Router Interface Records Test](../connectiontest/connection_router_interface_records_test.go)
