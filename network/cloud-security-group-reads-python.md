# Cloud 보안 그룹 검색·조회: Python과 Go

기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, Gophercloud `v2.15.0`입니다. [고정 Source](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1834)는 backend 선택, 전체 목록, `_filter_list`, `_get_entity` 순서를 정합니다.

| Python | 전체 Connection | 서비스/독립 cloud |
| --- | --- | --- |
| `search_security_groups(name_or_id, filters)` | `SearchSecurityGroupRecords(ctx, nameOrID, options...)` | `Service.SearchSecurityGroups` / `SecurityGroupCloud.SearchSecurityGroups` |
| `get_security_group(name_or_id, filters)` | `GetSecurityGroupRecord(ctx, reference, options...)` | `Service.GetSecurityGroup` / `SecurityGroupCloud.GetSecurityGroup` |
| raw dictionary/JSON string filters | `WithSecurityGroupReadFiltersJSON` | 동일 |
| JMESPath 편의 옵션 | `WithSecurityGroupReadExpression` | 동일 |
| headers·microversion·current location | `WithSecurityGroupReadHeaders` / `WithSecurityGroupReadMicroversion` / `WithSecurityGroupReadCurrentLocation` | 동일 |
| 완전한 옵션 교체 | `WithSecurityGroupReadOpts` | 동일 |

Connection의 `SecurityGroupCloud(ctx)`가 concrete SDK dependency를 구성합니다. Neutron catalog endpoint가 정상적으로 없다는 경우에만 Nova를 선택합니다. HTTP 오류·취소·여러 원인이 합쳐진 discovery 실패는 Nova로 바꾸지 않습니다. Neutron resource GET의 404도 Nova fallback 조건이 아닙니다. Connection은 current location을 자동 snapshot하며 전체 옵션 reset 후 location=nil이면 factory의 기본 location으로 돌아갑니다.

이미 선택한 Neutron client는 `network.New(client).SearchSecurityGroups(...)`를 사용합니다. 이 Service convenience는 Neutron 기본 경로입니다. 별도의 Source 정책이 필요한 연결은 `NewSecurityGroupCloud(SecurityGroupReadDependencies{Neutron: service, Nova: computeClient, Source: &source, NetworkAvailable: &available, UseDirectGet: direct, Strict: strict, CurrentLocation: &location})`로 구성합니다. 모두 concrete SDK 옵션이며 사용자 interface/builder/normalizer를 요구하지 않습니다. Source=nil은 neutron 기본값입니다. empty/unrecognized source는 unavailable feature, source의 소문자는 feature 존재 검사에만 쓰고 Neutron 선택은 정확한 `"neutron"` 비교입니다. 따라서 대문자 `"NEUTRON"`의 Nova 분기도 실제 Source처럼 보존합니다.

## 목록·검색·getter 계약

nameOrID는 server name query로 바꾸지 않습니다. dictionary filters만 Source `list_security_groups(**filters)`에 pushdown하고 선언된 서버 query·local Body filters·unknown discard를 leaf가 처리합니다. 이후 전체 목록에 ID/name 정확 일치·fnmatch glob·원래 dictionary 필터를 다시 적용합니다. 문자열 JMESPath와 다른 nonmapping filters는 list에 None을 전달해 전체 목록을 읽은 뒤 처리합니다. 따라서 truthy scalar/list의 mapping 오류도 실제 목록 뒤이며 빈 목록에서는 해당 key가 방문되지 않습니다. null·false·0·underflow float·빈 string/list/dict는 falsey입니다. ordered/nested mapping과 CPython loaded numeric equality를 공통 계층에서 재사용합니다.

JMESPath 결과는 임의 JSON이므로 `Expression=true`, `Value`에 scalar/null/object/list 결과가 들어가며 `Groups=nil`입니다. getter는 Source 순서인 truthiness→len→integer index 0을 사용합니다. empty/falsey 결과는 None에 대응하는 Value=nil, 한 ordinary group은 Group, 두 개 이상은 `resource.ErrAmbiguous`를 포함한 오류입니다. expression `[false]`·`[0]`는 false/0을 그대로 선택하지만 `[null]`은 None입니다. truthy scalar의 len 오류와 integer key 0이 없는 object 오류를 숨기지 않습니다.

기본 getter는 전체 목록 검색이며 member GET-first Find로 바꾸지 않습니다. explicit `UseDirectGet` 정책에서 UUID-like reference만 Source `get_security_group_by_id` 경로를 사용합니다. 해당 경로는 search filters를 무시하고 literal member 404를 오류로 보존합니다. 기존 SecurityGroupRecord, RawResource 또는 id key가 있는 JSON object는 `ExistingSecurityGroupRead`가 backend/discovery/HTTP/option callbacks 없이 반환합니다. id=null/false/0도 존재한 key이며 이미 받은 입력을 가상 응답으로 꾸미지 않습니다. Connection도 서비스 선택 전에 이 동일 helper를 호출합니다.

## Proxy·Nova·응답 증거

Source Proxy lane은 `groups.ProxyListRecords`와 `WithRecordProxyQuery`를 제공합니다. headers/microversion/base_path/paginated/max_items, 한 번의 conflicting attrs 복원과 bound formal 중복 검사를 처리합니다. `%(<key>)s` 경로는 JSON kwargs에서 바인딩하며 빈 base_path는 HTTP 없이 빈 generator입니다. 기존 typed ListRecords의 reserved/status policy는 유지합니다. JMESPath를 opaque generator에 먼저 평가하므로 literal 결과는 Neutron group HTTP 없이 반환되고 `to_array(@)` 같은 nested generator는 실제 `.Records` 소비 전까지 읽지 않습니다. cloud dictionary filter가 falsey literal을 건너뛰는 순서, ID phase의 더 이른 get() 오류 및 opaque membership이 실제 generator를 소비한 뒤 unknown-key 오류가 되는 Source 동작을 보존합니다.

Nova fallback은 Source처럼 `/os-security-groups` 한 번을 GET하고 JSON fields를 library가 정규화합니다. 숫자/문자열 id를 임의로 string으로 바꾸지 않으며 artifact를 제거하고 group/rule properties, tenant/project alias, remote prefix/group, protocol과 legacy rules를 처리합니다. Strict=false는 project aliases와 properties flatten을 유지하고 Strict=true는 이 compatibility fields를 생략합니다. Source의 eager pop/default 평가, port_range_min의 Python int 변환 및 port_range_max 문자열이 변환되지 않는 실제 버그도 보존합니다. Nova row는 Backend=nova이고 Resource는 normalized Munch view, Wire/Envelope/Header/status는 실제 Nova 응답입니다. Neutron class 객체로 받았다고 주장하지 않습니다.

[공식 Nova microversion 이력](https://docs.openstack.org/nova/2026.1/reference/api-microversion-history.html)의 2.36부터 `/os-security-groups` proxy는 404를 반환합니다. 이 fallback이 모든 현대 배포에서 동작한다고 보장하지 않습니다. 기존 [Nova native 설명](../compute/v2/secgroups/README.md)과 배포의 microversion/API 조건을 함께 확인합니다. SDK가 실패를 Neutron으로 재시도하거나 일부 성공으로 바꾸지 않습니다. [Neutron 27 get_security_group 정책](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/security_group.py)은 project reader/admin 또는 shared group을 구분합니다. API 권한 및 반환 가능한 fields는 배포 정책과 인증이 정합니다.

SecurityGroupReadResult의 Inventory/LastGroup/NovaResponse는 실제 관측 증거이고 Groups/Group/Values/Value는 성공한 선택입니다. Resource/Wire/Envelope/Header/status, selected records와 Inventory, raw filter/location pointers를 따로 소유합니다. 늦은 페이지·정규화·descriptor·context/source 실패는 partial Inventory와 원문 응답 오류만 남기고 성공 결과를 제거합니다. 실제 물리401 ErrUnableToReauthenticate의 ErrOriginal VALUE response 및 ErrReauth 원인을 보존하고 fault retry/backoff/reauth를 실행하지 않습니다. Source/client/API/route는 callbacks 및 모든 HTTP 경계에서 고정합니다.

Go의 concrete transport 옵션은 string-valued headers·nullable string microversion/base_path·safe service-relative routes와 bounded integer max_items를 사용하며 일부 Python 동적 오류보다 먼저 검증합니다. Numeric percent-format routes, Python 임의 객체/주소 repr, mutable Resource·dirty/session/cache SDK-R1/C1/S1은 별도 범위입니다. lexical dict-key/JMESPath key 함수 순서, URI escape·paging bounds/cycle·Link repair와 deprecated 경고 생략은 명시적인 Go 차이입니다. Fixture/Source reference는 authenticated live cloud acceptance가 아닙니다.

## 독립 Python main

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

예: `go run . -mode search -name 'web-*' -filters '{"project_id":"project"}'`, `go run . -filters '"[].id"'`, `go run . -mode get -existing '{"id":null,"name":"already fetched"}'`. [서비스 테스트](cloud_security_group_reads_test.go), [Source Proxy 테스트](v2/extensions/security/groups/python_list_proxy_test.go), [Connection 테스트](../connection_security_group_records_test.go)는 Neutron/Nova 조건 분기, selection/getter, 실제 응답 및 소유권을 검증합니다.
