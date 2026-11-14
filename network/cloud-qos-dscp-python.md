# Cloud QoS Dscp Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import os
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default=os.getenv("OS_CLOUD"))
    parser.add_argument("--policy", required=True)
    parser.add_argument("--rule", default="")
    parser.add_argument("--mode", choices=("list", "search", "get"), default="list")
    parser.add_argument("--filters", default="null")
    args = parser.parse_args()
    with openstack.connect(cloud=args.cloud) as conn:
        filters = json.loads(args.filters)
        if args.mode == "list":
            values = conn.list_qos_dscp_marking_rules(args.policy, filters)
        elif args.mode == "search":
            values = conn.search_qos_dscp_marking_rules(args.policy, args.rule or None, filters)
        else:
            if not args.rule:
                parser.error("--rule required for get")
            values = [conn.get_qos_dscp_marking_rule(args.policy, args.rule)]
        for value in values:
            print(json.dumps(value.to_dict() if hasattr(value, "to_dict") else value,
                             indent=2, ensure_ascii=False))

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
 "log"
 "os"
 "time"

 sdk "github.com/JSYoo5B/go-openstacksdk"
 "github.com/JSYoo5B/go-openstacksdk/network"
)
func main() {
 cloud:=flag.String("cloud",os.Getenv("OS_CLOUD"),"clouds.yaml cloud")
 policy:=flag.String("policy","","QoS policy ID or name")
 rule:=flag.String("rule","","rule ID or fnmatch pattern")
 mode:=flag.String("mode","list","list, search or get")
 filters:=flag.String("filters","null","JSON-domain Source filters")
 flag.Parse()
 if *policy=="" {log.Fatal("-policy required")}
 ctx,cancel:=context.WithTimeout(context.Background(),30*time.Second)
 defer cancel()
 var connect []sdk.ConnectionOption
 if *cloud!=""{connect=append(connect,sdk.WithCloud(*cloud))}
 conn,err:=sdk.Connect(ctx,connect...)
 if err!=nil{log.Fatal(err)}
 service,err:=conn.Network(ctx)
 if err!=nil{log.Fatal(err)}
 location,err:=conn.CurrentLocation()
 if err!=nil{log.Fatal(err)}
 options:=[]network.DSCPMarkingCloudOption{func(c *network.DSCPMarkingCloudOpts) error { loc := location.Clone(); c.CurrentLocation = &loc; return nil }}
 var result *network.DSCPMarkingCloudResult
 switch *mode {
 case "list":
  options=append(options,network.WithDSCPMarkingCloudFiltersJSON(json.RawMessage(*filters)))
  result,err=service.ListQoSDSCPMarkingRules(ctx,*policy,options...)
 case "search":
  options=append(options,network.WithDSCPMarkingCloudFiltersJSON(json.RawMessage(*filters)))
  result,err=service.SearchQoSDSCPMarkingRules(ctx,*policy,*rule,options...)
 case "get":
  if *rule==""{log.Fatal("-rule required for get")}
  result,err=service.GetQoSDSCPMarkingRule(ctx,*policy,*rule,options...)
 default:log.Fatal("mode must be list, search or get")
 }
 if err!=nil{
  if result!=nil{fmt.Fprintf(os.Stderr,"observed extensions=%d rules=%d\n",len(result.Extensions),len(result.Inventory))}
  log.Fatal(err)
 }
 if result.Value==nil{fmt.Printf("opaque result: %d values\n",len(result.Values));return}
 fmt.Println(string(result.Value))
}
```

## References

- [ Network](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py)
- [QoS](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py)
- [Cloud QoS Dscp Test](cloud_qos_dscp_test.go)
