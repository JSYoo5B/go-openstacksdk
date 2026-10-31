# Cloud Router Interfaces Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network/NetworkCloudMixin/list_router_interfaces`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.ListRouterInterfaces`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.ListRouterInterfaces`
- `github.com/JSYoo5B/go-openstacksdk/network.RouterInterfaceRouterID`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterInterfaceReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterInterfaceType`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterInterfaceTypeJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterInterfaceReadHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterInterfaceReadMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterInterfaceReadCurrentLocation`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.ListRecords`

- Library reads literal router dictionary id and sends only the declared device_id query without router member fetch/name lookup/response seeds. Null/bool/number/list/dict ID query behavior, missing/nonmapping input errors and real untyped owner values follow Source JSON-domain semantics.
  Tests: `network/cloud_router_interfaces_test.go:TestCloudRouterInterfacesRouterDictionaryReadDoesNotResolveOrSeed`.
- Every port page is collected before the Source discriminator. None/false/zero/underflow/empty values select both buckets; exact internal/external select their owner families; other truthy values yield empty success after I/O. Internal owners precede gateway owners with stable within-bucket order and no duplicate removal.
  Tests: `network/cloud_router_interfaces_test.go:TestCloudRouterInterfacesCollectsEveryPageBeforeStableOwnerBuckets`, `network/cloud_router_interfaces_test.go:TestCloudRouterInterfacesFalseyExactUnknownAndNonstringTypes`.
- Actual normalized Port Resource/raw Wire/envelope/header/status, selected Ports and partial Inventory/LastPort/JSON Value are independently owned. CurrentLocation is injected by Connection with loaded project semantics and no forged wire zone. Headers/microversion/JSON pointers are snapshots and source/API/context guards span callbacks, collection and selection.
  Tests: `network/cloud_router_interfaces_test.go:TestCloudRouterInterfacesOwnedInputsLocationHeadersAndRawReceipt`, `network/cloud_router_interfaces_test.go:TestCloudRouterInterfacesSourceGuardCancellationAndPreflight`, `connectiontest/connection_router_interface_records_test.go:TestConnectionRouterInterfacesClassifyCompletePortsWithConfiguredLocation`.
- Late physical pages at 200/203/401/404/429/503 preserve real response body/header/status and native pointer401 ErrOriginal VALUE receipt/ErrReauth, stop fault retry/backoff/reauth callbacks and prevent successful partial Ports/Value even for unknown interface_type.
  Tests: `network/cloud_router_interfaces_test.go:TestCloudRouterInterfacesLatePagesPreventPartialSuccessAndKeepPhysicalFaults`.

Python/Go differences:

- Go returns a raw JSON-domain router dictionary and concrete discriminator/header/version/location options plus a literal ID dictionary convenience; no user builder or extra router lookup is introduced. Malformed JSON and nil/unsafe sources fail as explicit Go errors.
- Go separates selected Ports/Value from actual partial Inventory/LastPort receipts with independent raw bytes/metadata and automatically configured Connection location. Unknown/nonstring owners are skipped instead of guessed; no fake interface Resource is synthesized.
- The existing frozen Port list transport retains its reviewed 200..399 status policy and service-relative request/paging/Link/cycle ownership. Python mutable Resource/dirty/session/cache and descriptor-laziness contracts remain SDK-R1/C1/S1 generic scope; stock controller read/field policies and authenticated live cloud acceptance remain separate.

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
