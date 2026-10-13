# Port cloud 목록·subnet 선택·bulk 생성

기준은 openstacksdk [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`](https://github.com/openstack/openstacksdk/tree/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe)와 Gophercloud v2.15.0입니다. [기존 목록·검색](python-list-records.md)의 descriptor, kwargs, 페이지, 실제 응답 계약 위에 다음 세 기능을 제공합니다.

| Python | 전체 Connection / Go 서비스 | 결과 |
|---|---|---|
| `conn.list_ports(filters=...)` | `conn.ListPortValues(ctx, ports.WithSourceCloudListFilters(raw))` / `service.Ports.ListSourceValues` | Source `list()` 변환, JSON 값과 실제 Port 구별 |
| `conn.network.get_subnet_ports(subnet)` | `service.Ports.GetSubnetPorts(ctx, ports.SubnetPortID(id))` | 선택한 포트와 실제 전체 inventory |
| `conn.network.create_ports(data)` | `service.Ports.CreatePorts(ctx, []ports.BulkPortInput{...})` | 즉시 POST의 receipt와 lazy response records |

기존 `conn.ListPorts(ctx, ...ports.RecordListOption)`의 typed 반환은 유지합니다. 문자열·JSON literal·opaque generator까지 다루려면 `ListPortValues`를 사용합니다. 별도 interface builder나 response mapper를 구현하지 않습니다.

## Cloud `list()` 변환

[Source cloud `list_ports`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L202)는 network service가 없으면 필터를 평가하기 전에 `[]`를 반환합니다. Go Connection도 깨끗한 catalog endpoint 부재에서 동일하게 callbacks와 필터 확장을 생략합니다. Endpoint 조회 자체의 오류·취소·잘못된 설정은 오류이며, service를 configuration에서 끄는 Source 설정 모델은 별도 공통 과제입니다.

필터 생략·null·false·0·빈 문자열/목록/객체와 CPython float underflow는 필터 없음입니다. truthy 필터는 kwargs용 JSON object여야 합니다. 여기의 문자열 필터는 cloud `search_ports`의 JMESPath 문자열과 다릅니다. object 안의 `jmespath_filters`는 원본 Proxy의 generator에 평가하며, 그 결과를 Python `list()`처럼 변환합니다.

| `jmespath_filters` | `ListPortValues` 결과 | HTTP |
|---|---|---|
| 생략·falsey·문자열 아닌 값, `@` | 실제 records를 수집한 `Values`, `Ports`, `Inventory` | 실제 목록 GET |
| `` `[1,null,true,{"id":"literal"}]` `` | 그대로인 JSON 값 목록, `Ports` 없음 | 0 |
| `` `{"z":1,"a":2}` `` | object key 목록 | 0 |
| `'한😀'`, `''` | Unicode 문자 목록, 빈 문자열은 `[]` | 0 |
| `to_array(@)` | generator pointer를 담은 `Values`, `JSON` 없음 | 반환 시 0, 명시 순회 시 GET |
| `[*].id`, `type(@)`, `` `null` ``, `` `true` ``, `` `42` `` | noniterable 오류 | 0 |
| `length(@)`, `to_string(@)` | generator 타입/직렬화 오류 | 0 |

JSON 값은 `PortRecord`로 합성하지 않습니다. 실제 records에만 `Wire`, `Envelope`, `Header`, `StatusCode`가 있습니다. Connection은 실제 Resource 속성에 current cloud/project/region과 remote project/zone으로 계산한 location을 보충하며 Wire는 보존합니다. 중첩 generator는 자동 수집하지 않고 JSON marshaling도 오류입니다. 뒤 페이지가 실패하면 실제 읽은 `Inventory`만 보존하며 성공한 `Values/Ports/JSON`으로 표시하지 않습니다.

## Subnet의 포트

`SubnetPortID`는 literal ID입니다. `SubnetPortRecord`와 `SubnetPortNative`는 전달된 subnet의 ID를 복사하며 추가 GET을 보내지 않습니다. `SubnetPortValue`는 Source `_get_id`가 Resource 아닌 값을 그대로 반환하는 JSON 입력을 보존합니다. `SubnetPortFields`는 ID만 가져오는 입력 view이며 응답 증거가 아닙니다.

`GetSubnetPorts`는 [Source loop](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L4047)처럼 **필터 없는 전체 포트 목록**을 순회해 fixed IP의 subnet_id를 비교합니다. 동일 포트의 fixed IP 두 개가 일치하면 같은 pointer를 두 번 반환합니다. 입력값은 CPython JSON의 bool/int/float와 재귀 equality를 따릅니다. fixed_ips null, 잘못된 fixed IP object, 누락된 subnet_id는 실제 page receipt가 있는 오류입니다. Go는 오류 전에 확인한 `Inventory`와 선택한 `Ports`도 보존합니다.

## Bulk 생성

[Source create_ports](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L3894)와 같이 SDK가 `POST /ports` 한 번의 `{"ports":[attrs1,attrs2]}` 요청을 소유합니다. 각 행을 개별 생성하지 않습니다. `BulkPort(WithBulkPortAttribute(...))`는 삽입 순서를 소유하며 `BulkPortJSON`은 raw JSON object 순서와 unknown extension을 보존합니다. 공개 `Attributes` map / `BulkPortFields`의 별개 key 순서는 Go 정책상 lexical이며 alias 충돌 순서가 중요하면 factory나 raw JSON을 사용합니다.

요청 Body는 raw 값입니다. 응답 descriptor의 bool/list/dict 변환을 제출 속성에 적용하지 않습니다. canonical/wire alias는 같은 wire key로 바뀌고 마지막 별개 spelling이 이깁니다. request의 microversion 속성은 constructor formal로 소비되고 `WithBulkPortMicroversion` 또는 캡처한 client 값이 HTTP version을 결정합니다. request `cls/self/connection/_synchronized`는 POST 전에 오류이며, base_path/resource_type/__conflicting_attrs/location은 bulk에서는 unknown Body 속성입니다.

모든 입력 행의 헤더를 준비하고, **마지막 행의 if_match만** 요청 헤더에 적용합니다. 마지막 행에 헤더가 없으면 앞선 행의 조건은 적용하지 않습니다. 캡처한 client / `WithBulkPortHeaders` 설정은 유지합니다. if_match string list는 `a, b`로 합치고 null/false/0은 Source처럼 `None`/`False`/`0`입니다. 앞 행의 잘못된 header도 POST 전에 오류입니다. Go는 CR/LF와 인증·framing·다른 서비스 version 헤더 변경을 preflight에서 거부합니다.

`CreatePorts`가 반환될 때 POST와 HTTP 상태 확인·JSON decode·ports lookup은 이미 끝났습니다. `result.Records`는 그 응답 행만 lazy 변환하며 `result.All()`도 POST를 다시 보내지 않습니다. 누락값은 descriptor 기본값이고 입력 행과 병합하거나 행 수를 맞추지 않습니다. response의 cls/self/connection/_synchronized/microversion 충돌이나 malformed 행은 순회할 때 실제 완료된 POST receipt를 가진 오류입니다. 그 전에 나온 정상 행은 iterator consumer가 유지할 수 있습니다. `All`은 projection 오류 시 partial slice를 폐기합니다. Go iterator는 반복해서 같은 저장 응답을 변환할 수 있습니다. Source generator는 일회용입니다.

204·비JSON·missing ports·ports null/bool/number는 이미 완료된 POST receipt를 가진 즉시 오류입니다. Source의 empty iterable인 ports `{}`와 `""`도 빈 결과로 처리하고, nonempty object/string은 iteration-time row 오류를 보존합니다. HTTP response의 If-Match는 record 속성을 seed하지 않으며 JSON row에 있는 if_match만 속성에 반영합니다. 물리 read/close 실패는 실제 bytes와 혼합 원인을 보존하고 retry/reauth callback을 실행하지 않습니다. 깨끗한 reauthentication/retry는 동일 bulk body·route·선택 version·조건부 헤더를 보존합니다.

bulk user/admin 권한과 원자성은 서버가 결정합니다. [공식 Neutron bulk API](https://docs.openstack.org/api-ref/network/v2/#bulk-create-ports)는 bulk-capable plugin 계약을 설명합니다. [Neutron 27 Pecan BodyValidation](https://github.com/openstack/neutron/blob/27.0.0/neutron/pecan_wsgi/hooks/body_validation.py)과 [Policy hooks](https://github.com/openstack/neutron/blob/27.0.0/neutron/pecan_wsgi/hooks/policy_enforcement.py)는 각 행을 검증하고 create_port 및 attribute 권한을 적용합니다. [기본 ML2](https://github.com/openstack/neutron/blob/27.0.0/neutron/plugins/ml2/plugin.py)는 native bulk transaction을 제공하지만 모든 backend의 live rollback을 이 SDK 테스트가 보장하지 않습니다. [기본 port attribute 정책](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/port.py)에서 binding_profile은 service 역할, binding_host_id는 admin/service 등 별도 권한이 적용됩니다. SDK가 N개의 POST로 원자성을 흉내 내지 않습니다.

## 독립 Python 예제

`--mode=values`가 기본입니다. `--mode=bulk --data='[{"network_id":"...","name":"one"},{"network_id":"...","name":"two"}]'`는 실제 포트를 생성합니다.

```python
import argparse
import json
import os
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default=os.getenv("OS_CLOUD"))
    parser.add_argument("--mode", choices=("values", "subnet", "bulk"), default="values")
    parser.add_argument("--filters", default="{}")
    parser.add_argument("--subnet", default="")
    parser.add_argument("--data", default="[]")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    if args.mode == "values":
        values = conn.list_ports(filters=json.loads(args.filters))
        for value in values:
            print(json.dumps(value.to_dict() if hasattr(value, "to_dict") else value, ensure_ascii=False, indent=2))
    elif args.mode == "subnet":
        if not args.subnet:
            parser.error("--subnet required")
        for value in conn.network.get_subnet_ports(args.subnet):
            print(json.dumps(value.to_dict(), ensure_ascii=False, indent=2))
    else:
        for value in conn.network.create_ports(json.loads(args.data)):
            print(json.dumps(value.to_dict(), ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
```

## 독립 Go main

`` --filters='{"jmespath_filters":"`[1,null]`"}' ``는 HTTP 없이 JSON 값을 반환합니다. `to_array(@)` 같은 opaque 결과는 `Values`에서 generator를 명시적으로 순회합니다. 아래 예제는 JSON으로 표현되는 값만 출력합니다.

```go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
)

func printJSON(value any) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(encoded))
}

func main() {
	cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud")
	mode := flag.String("mode", "values", "values, subnet or bulk")
	filters := flag.String("filters", "{}", "Source cloud filters JSON")
	subnetID := flag.String("subnet", "", "literal subnet ID")
	data := flag.String("data", "[]", "nonempty JSON array of port attribute objects")
	flag.Parse()
	ctx := context.Background()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	if *mode == "values" {
		result, err := conn.ListPortValues(ctx, ports.WithSourceCloudListFilters(json.RawMessage(*filters)))
		if err != nil {
			log.Fatal(err)
		}
		if result.JSON == nil {
			log.Fatal("opaque result; consume RecordListGenerator.Records explicitly")
		}
		printJSON(result.JSON)
		return
	}
	network, err := conn.NetworkV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	switch *mode {
	case "subnet":
		if *subnetID == "" {
			log.Fatal("--subnet required")
		}
		result, err := network.Ports.GetSubnetPorts(ctx, ports.SubnetPortID(*subnetID))
		if err != nil {
			log.Fatal(err)
		}
		for _, record := range result.Ports {
			printJSON(record.Resource.Body)
		}
	case "bulk":
		var rows []json.RawMessage
		if err := json.Unmarshal([]byte(*data), &rows); err != nil {
			log.Fatal(err)
		}
		inputs := make([]ports.BulkPortInput, len(rows))
		for i, row := range rows {
			inputs[i] = ports.BulkPortJSON(row)
		}
		result, err := network.Ports.CreatePorts(ctx, inputs)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("POST completed: %d\n", result.StatusCode)
		for record, err := range result.Records {
			if err != nil {
				log.Fatal(err)
			}
			printJSON(record.Resource.Body)
		}
	default:
		log.Fatal("--mode must be values, subnet or bulk")
	}
}
```

## 비교·검증 경계

공개 Gophercloud testhelper를 쓰는 기존 testcloud/native fixture와 공통 SDK response/fault 계층을 재사용합니다. 고정 Source의 실제 methods로 local list conversion/subnet 선택 41건, bulk request/response 50건과 constructor·추가 response 15건을 별도 Python runtime에서 비교했습니다. Go contract tests는 single POST, alias/header/version, ownership/guards, polymorphic 값, eager/lazy 오류, 실제 receipt와 fault/retry를 다룹니다. 실제 OpenStack deployment의 policy/plugin/rollback 검증은 수행하지 않았습니다.

JMESPath object key 순서는 Go lexical이며 Python literal object insertion order와 다를 수 있습니다. JMESPath strict syntax·exact JSON numeric evaluator·custom Resource type 함수 차이와 route/header protection은 기존 공통 SDK 정책입니다. Go는 concrete JSON 입력·captured service client/context를 사용하므로 Python session/Resource class 교체나 임의 Python 객체의 연산자·callable을 지원하지 않습니다. 공통 Resource mutable state·cloud configuration 전체 계약은 SDK-C1/R1 추적으로 남아 있으며 세 helper의 함수별 계약과 구별합니다.
