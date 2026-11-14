# Cloud Security Group Reads Python

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
parser.add_argument("--mode", choices=("search", "get"), default="search")
parser.add_argument("--name", default="")
parser.add_argument("--filters", default="null")
parser.add_argument("--existing", default="")
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    filters = json.loads(args.filters)
    if args.mode == "get":
        reference = json.loads(args.existing) if args.existing else args.name
        result = conn.get_security_group(reference, filters)
    else:
        result = conn.search_security_groups(args.name or None, filters)
    if hasattr(result, "to_dict"):
        result = result.to_dict()
    elif isinstance(result, list):
        result = [value.to_dict() if hasattr(value, "to_dict") else value
                  for value in result]
    print(json.dumps(result, ensure_ascii=False, indent=2))
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
 mode:=flag.String("mode","search","search or get")
 name:=flag.String("name","","name, ID or fnmatch pattern")
 filters:=flag.String("filters","null","raw JSON filters; quote a JMESPath string")
 existing:=flag.String("existing","","already fetched JSON object containing id")
 flag.Parse()
 ctx,cancel:=context.WithTimeout(context.Background(),30*time.Second);defer cancel()
 var connect []sdk.ConnectionOption;if *cloud!=""{connect=append(connect,sdk.WithCloud(*cloud))}
 conn,err:=sdk.Connect(ctx,connect...);if err!=nil{log.Fatal(err)}
 options:=[]network.SecurityGroupReadOption{network.WithSecurityGroupReadFiltersJSON(json.RawMessage(*filters))}
 var result *network.SecurityGroupReadResult
 switch *mode{
 case "search":result,err=conn.SearchSecurityGroupRecords(ctx,*name,options...)
 case "get":var reference any=*name;if *existing!=""{reference=json.RawMessage(*existing)};result,err=conn.GetSecurityGroupRecord(ctx,reference,options...)
 default:log.Fatal("mode must be search or get")
 }
 if err!=nil{if result!=nil{fmt.Fprintf(os.Stderr,"backend=%s observed=%d\n",result.Backend,len(result.Inventory))};log.Fatal(err)}
 if result.Value==nil{fmt.Println("null");return};fmt.Println(string(result.Value))
}
```

## References

- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1834)
- [API Microversion History](https://docs.openstack.org/nova/2026.1/reference/api-microversion-history.html)
- [Secgroups](../compute/v2/secgroups/README.md)
- [Security Group](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/security_group.py)
- [Cloud Security Group Reads Test](cloud_security_group_reads_test.go)
- [Python List Proxy Test](v2/extensions/security/groups/python_list_proxy_test.go)
- [Connection Security Group Records Test](../connectiontest/connection_security_group_records_test.go)
