# Cloud 라우터 인터페이스: Python과 Go

openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [list_router_interfaces Source](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2205)와 Gophercloud `v2.15.0`를 기준으로 합니다.

| Python 입력 | Go |
| --- | --- |
| `conn.list_router_interfaces(router, interface_type=None)` | `Connection.ListRouterInterfaces(ctx, routerJSON, options...)` |
| 동일 서비스 호출 | `Service.ListRouterInterfaces` |
| 이미 받은 router dictionary | `json.RawMessage`를 그대로 전달 |
| literal router ID 편의 생성 | `RouterInterfaceRouterID(id)` |
| internal/external string | `WithRouterInterfaceType` |
| falsey·임의 JSON discriminator | `WithRouterInterfaceTypeJSON` |
| concrete headers/version/location | `WithRouterInterfaceReadHeaders` / `WithRouterInterfaceReadMicroversion` / `WithRouterInterfaceReadCurrentLocation` |
| 전체 옵션 교체 | `WithRouterInterfaceReadOpts` |

SDK는 router dictionary의 id key를 읽어 `network.ports(device_id=id)`에 전달합니다. router member GET이나 이름 lookup을 추가하지 않습니다. id=null은 Requests처럼 query를 생략하고 bool/number/list/dict id의 JSON-domain query encoding을 보존합니다. id key가 없거나 router가 nonmapping이면 port HTTP 이전에 오류입니다. `RouterInterfaceRouterID`는 dictionary를 만드는 SDK helper이며 fetch 또는 사용자 builder가 아닙니다.

모든 port 페이지를 먼저 수집한 뒤 interface_type을 검사합니다. None과 모든 falsey 값(null/false/0/underflow/빈 string/list/dict)은 내부와 외부를 모두 선택합니다. 정확한 `"internal"`은 내부만, `"external"`은 외부만이고 다른 truthy 값은 빈 성공 목록입니다. unknown discriminator도 실제 포트 조회를 생략하지 않습니다. 문자열 비교는 case sensitive입니다.

| 선택 | Source device_owner |
| --- | --- |
| internal | `network:router_interface` |
| internal | `network:router_interface_distributed` |
| internal | `network:ha_router_replicated_interface` |
| external | `network:router_gateway` |

`network:router_ha_interface`, compute ports, missing/nonstring owner는 제외합니다. 반환 순서는 전체 입력 순서에 따른 internal bucket 다음 gateway bucket입니다. gateway가 원래 첫 page에 있어도 internal rows가 먼저 나갑니다. 중복은 제거하지 않습니다.

RouterInterfaceReadResult는 실제 Inventory/LastPort와 선택된 Ports/Value를 구분합니다. Resource는 기존 Source Port default/descriptor/alias view, Wire는 원문 row, Envelope/Header/StatusCode는 실제 page입니다. 선택한 Ports와 Inventory/LastPort/raw Value는 독립적으로 소유합니다. 늦은 HTTP/constructor/source/cancel 실패는 partial Inventory 및 실제 응답 오류만 보존하고 Ports/Value를 성공으로 반환하지 않습니다. read/Close fault와 native pointer401의 두 원인·원문/status/header, fault retry/backoff/reauth 0 조건을 기존 leaf 처리에서 재사용합니다.

Connection은 CurrentLocation을 library-owned 기본값으로 전달합니다. Source의 project_id/tenant_id와 loaded equality/truthiness를 적용하고 foreign project의 current 이름을 지우며, Port에는 availability_zone descriptor가 없으므로 unknown wire zone이 location을 바꾸지 않습니다. 직접 client는 기본 location null을 유지합니다. 사용자 Header/JSON/location pointer는 캡처해 나중에 변형해도 이미 수행한 결과나 guard를 바꾸지 않습니다.

[Neutron 27 port 정책](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/port.py)은 project scope의 get_port와 binding/resource_request 등의 field read 규칙을 구분합니다. SDK는 조회 권한이나 누락된 보호 fields를 가정하지 않습니다. 선택 과정에서 port 생성·router 갱신·관리자 동작을 수행하지 않습니다. API 응답에 없는 owner를 추측해 인터페이스를 합성하지 않습니다.

Go는 raw router JSON과 concrete 옵션을 사용합니다. malformed JSON·nil context/service·unsafe source/header와 route drift는 Go 오류입니다. 기존 Port list의 reviewed 200..399 transport policy와 안전한 URI/paging/Link/cycle 정책을 유지하며 Python mutable Resource/session/cache SDK-R1/C1/S1과 실제 인증 cloud acceptance는 별도 범위입니다.

## 독립 Python main

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

## 전체 SDK 독립 Go main

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

예: `go run . -router '{"id":"router-id"}' -interface-type '"internal"'`. 이미 selected client를 가진 서비스는 `network.New(client).ListRouterInterfaces(ctx, network.RouterInterfaceRouterID("router-id"))`로 동일 기능을 사용합니다. 기존 router record가 있다면 `json.Marshal(record.Resource.Body)`를 입력으로 전달하며 SDK가 router 조회를 다시 하지 않습니다.

[서비스 계약](cloud_router_interfaces_test.go)의 6개 그룹과 [Connection 계약](../connection_router_interface_records_test.go)은 전체 paging·owner bucket 순서, falsey/unknown discriminator, raw router id query, default/foreign location, input/result ownership, context/source preflight와 실제 후속 200·203·401·404·429·503 Content-Length fault를 검증합니다. 이 HTTP fixture는 Gophercloud public testhelper/testcloud를 재사용합니다.
