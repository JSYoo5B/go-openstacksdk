# Neutron network·port·subnet의 ID/name 검색

기준은 Gophercloud **v2.15.0**, openstacksdk **ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe**입니다. `conn.NetworkV2(ctx)`의 세 API는 고정 Python `Resource.find`와 `NetworkResource`의 검색·필터·응답 속성 규칙을 SDK가 직접 제공합니다. 새 검색은 일반 사용자 조회 단계이며, 실제 접근 범위는 Neutron의 소유 프로젝트·공유/RBAC 및 배포 정책에 따릅니다. 관리자 범위의 조회를 자동으로 시도하지 않습니다.

| openstacksdk | Go | 반환값 |
|---|---|---|
| `conn.network.find_network(name_or_id, **query)` | `network.Networks.FindRecord(ctx, nameOrID, options...)` | `*networks.NetworkRecord` |
| `conn.network.find_port(name_or_id, **query)` | `network.Ports.FindRecord(ctx, nameOrID, options...)` | `*ports.PortRecord` |
| `conn.network.find_subnet(name_or_id, **query)` | `network.Subnets.FindRecord(ctx, nameOrID, options...)` | `*subnets.SubnetRecord` |

`FindRecord`는 생략한 옵션을 SDK 기본값으로 채웁니다. 각 패키지의 `WithRecordFindQuery`, `WithRecordFindIgnoreMissing`, `WithRecordFindHeaders`, `WithRecordFindMicroversion`, `WithRecordFindOpts`가 JSON·헤더·포인터의 복사본을 소유합니다. 확장 query 또는 local Body filter를 사용하려고 별도 interface/builder를 구현할 필요가 없습니다.

## 독립 Python 예제

`main.py`로 저장하고 동일한 `clouds.yaml`에 대해 실행합니다. `python main.py --cloud dev --network app-net --port app-port --subnet app-subnet --project PROJECT_ID`는 조회만 수행합니다. `--strict`를 지정하면 목록 검색 결과가 없을 때 오류로 처리합니다.

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--network", default="app-net")
    parser.add_argument("--port", default="")
    parser.add_argument("--subnet", default="")
    parser.add_argument("--project", default="")
    parser.add_argument("--strict", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    query = {"project_id": args.project} if args.project else {}
    finders = [
        ("network", args.network, conn.network.find_network),
        ("port", args.port, conn.network.find_port),
        ("subnet", args.subnet, conn.network.find_subnet),
    ]
    for kind, identity, find in finders:
        if not identity:
            continue
        result = find(identity, ignore_missing=not args.strict, **query)
        print(kind)
        print(json.dumps(result.to_dict() if result is not None else None,
                         indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
```

## 독립 Go 예제

[설치 안내](../../docs/install.md)를 적용하고 `main.go`로 저장합니다. `go run . -cloud dev -network app-net -port app-port -subnet app-subnet -project PROJECT_ID`는 위 Python 예제와 같은 세 검색을 수행합니다. `-strict`는 missing policy만 바꾸며, HTTP/페이지 오류는 기본 설정에서도 오류입니다.

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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/subnets"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func printResource(kind string, value *resource.RawResource) {
    fmt.Println(kind)
    if value == nil { fmt.Println("null"); return }
    body, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(body))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    networkName := flag.String("network", "app-net", "network ID or exact name")
    portName := flag.String("port", "", "optional port ID or exact name")
    subnetName := flag.String("subnet", "", "optional subnet ID or exact name")
    project := flag.String("project", "", "optional project filter")
    strict := flag.Bool("strict", false, "fail on logical absence")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    if *networkName != "" {
        options := []networks.RecordFindOption{networks.WithRecordFindIgnoreMissing(!*strict)}
        if *project != "" { options = append(options, networks.WithRecordFindQuery("project_id", *project)) }
        result, err := service.Networks.FindRecord(ctx, *networkName, options...)
        if err != nil { log.Fatal(err) }
        if result == nil { printResource("network", nil) } else { printResource("network", result.Resource) }
    }
    if *portName != "" {
        options := []ports.RecordFindOption{ports.WithRecordFindIgnoreMissing(!*strict)}
        if *project != "" { options = append(options, ports.WithRecordFindQuery("project_id", *project)) }
        result, err := service.Ports.FindRecord(ctx, *portName, options...)
        if err != nil { log.Fatal(err) }
        if result == nil { printResource("port", nil) } else { printResource("port", result.Resource) }
    }
    if *subnetName != "" {
        options := []subnets.RecordFindOption{subnets.WithRecordFindIgnoreMissing(!*strict)}
        if *project != "" { options = append(options, subnets.WithRecordFindQuery("project_id", *project)) }
        result, err := service.Subnets.FindRecord(ctx, *subnetName, options...)
        if err != nil { log.Fatal(err) }
        if result == nil { printResource("subnet", nil) } else { printResource("subnet", result.Resource) }
    }
}
```

기존 typed `FindIdentity`·`Find`·native `Resources`와 새 record API는 함께 사용할 수 있습니다. Python의 descriptor·부분 응답·알 수 없는 확장 속성을 보존해야 하는 경우 `FindRecord`를 사용합니다.

## GET, fallback, missing

검색은 identity 문자열 전체를 하나의 escaped path segment로 넣어 `/networks/{identity}`, `/ports/{identity}`, `/subnets/{identity}`를 먼저 GET합니다. kwargs는 이 GET에 원래 이름으로 보내며, JSON 배열은 반복 query, bool은 `True`/`False`, null은 query에서 생략합니다. 객체 query는 Requests와 같이 key를 나열합니다. 공백이나 `/`, `%`가 포함된 이름도 literal GET을 먼저 시도합니다.

GET가 성공하면 그 결과를 바로 반환하며 목록용 local filter를 다시 적용하지 않습니다. GET가 **깨끗한 400·403·404**일 때만 목록 검색으로 이어집니다. 읽기·닫기·취소·retry callback·선택 client 변경 오류가 붙은 응답은 fallback 근거가 되지 않습니다. LIST 403/404나 후속 페이지 오류는 missing으로 숨기지 않습니다.

`name` kwargs가 없을 때만 목록 query에 `name=identity`를 추가합니다. 명시한 `name=""`는 빈 query 값이고, `name=nil`은 query를 보내지 않으면서 자동 name hint도 막습니다. 반환된 문자열 ID **또는** name을 정확히 비교하고 목록을 끝까지 확인합니다. 두 번째 일치가 나오면 `resource.ErrAmbiguous`입니다. ID가 없거나 숫자인 row도 name이 일치하면 반환할 수 있으며, 받은 ID를 새 요청 경로로 검증하거나 재조회하지 않습니다.

전체 목록에 일치가 없으면 기본값은 `nil, nil`입니다. `WithRecordFindIgnoreMissing(false)`는 `resource.ErrNotFound`로 바꿉니다. 이것은 고정 Python의 `ignore_missing=True` 기준이며 이후 Python 버전의 default 변경을 추측해 적용하지 않습니다.

## 서버 query와 local Body filter

서버 query의 canonical 이름과 wire alias는 고정 Python `QueryParameters` 선언에서 가져옵니다. canonical 값이 자신의 wire alias보다 우선하고, 같은 wire 이름에 대한 선언이 여러 개면 마지막 선언이 우선합니다. 값이 명시한 null이면 그 query를 없앱니다.

| 서비스 | canonical → wire 예 | 목록에서만 적용하는 Body filter 예 |
|---|---|---|
| Network | `is_shared → shared`, `is_router_external → router:external`, `provider_network_type → provider:network_type`, `ipv4_address_scope_id → ipv4_address_scope` | `mtu`, `segments`, `availability_zones`, `subnet_ids` |
| Port | `is_admin_state_up → admin_state_up`, `security_group_ids → security_groups`, `tenant_id → project_id` | `binding_profile`, `binding_vif_details`, `resource_request`, `trunk_details` |
| Subnet | `is_dhcp_enabled → enable_dhcp`, `subnet_pool_id → subnetpool_id`, `use_default_subnet_pool → use_default_subnetpool` | `prefix_length`, `allocation_pools`, `host_routes`, `service_types` |
| 공통 | `any_tags → tags-any`, `not_tags → not-tags`, `not_any_tags → not-tags-any` | 서비스에서 선언한 Body 속성만 분류 |

포트의 `project_id`와 `tenant_id`를 동시에 지정하면 Python 선언 순서상 `tenant_id` 값이 `project_id` wire query를 마지막에 대체합니다. `binding:profile`은 선언된 서버 query이고 `binding_profile`은 canonical Body의 local filter입니다. 이 두 키를 같은 범주로 취급하지 않습니다. 미등록 kwargs는 member GET에는 보내지만 fallback 목록에서는 버립니다. 특별 인자 `all_projects`는 member GET에도 보내지 않으며 세 Neutron 목록에서는 미등록 query라 버립니다.

Local scalar/list filter는 Python JSON-domain equality를 사용합니다. `true == 1`, 재귀 배열/객체의 bool·number 동등성도 유지합니다. 객체 filter는 재귀 subset 비교입니다. 실제 값이 falsey이면 빈 객체 filter도 일치하지 않습니다. 빈 객체 filter는 truthy scalar/list에도 일치하지만, 비어 있지 않은 객체 filter에서 실제 scalar/list의 dictionary 조회가 필요해지면 실제 페이지 증거를 보존한 오류입니다. 첫 filter 불일치에서 평가를 중단합니다. `WithRecordFindQuery`로 추가한 local filter 순서는 보존하고, bulk Go map에서 추가한 키는 이름순으로 고정합니다. 예를 들어 아래 포트 검색은 서버의 `binding:profile` query를 추가하는 대신 받은 `binding_profile`을 로컬에서 비교합니다.

```python
port = conn.network.find_port(
    "app-port", binding_profile={"nested": {"enabled": True}})
```

```go
port, err := service.Ports.FindRecord(ctx, "app-port",
    ports.WithRecordFindQuery("binding_profile",
        map[string]any{"nested": map[string]any{"enabled": true}}))
```

`WithRecordFindQuery("max_items", n)`는 decoded raw row 개수를 제한하며 local filter로 탈락한 row도 셉니다. 0은 무제한입니다. 요청 limit이 없거나 0이면 양수 cap을 limit hint로 사용합니다. `WithRecordFindQuery("paginated", false)`는 목록 첫 페이지만 확인합니다. 이 제어는 Source truthiness를 따르므로 null·0·빈 배열도 첫 페이지로 제한하고, 문자열 `"False"`와 1은 후속 페이지를 읽습니다. API가 제공하는 linked pages, HTTP Link, top-level next 및 명시 limit에 대한 short-page marker continuation을 지원합니다. 기본 검색에는 raw cap을 만들지 않습니다.

## 속성 뷰와 HTTP 증거

각 record는 `Resource`와 `Wire`의 독립 `RawResource`, 원래 whole response `Envelope`, 실제 `Header`와 `StatusCode`를 보관합니다. `Resource.Body`는 Python canonical 속성 이름, `Wire.Body`는 서버 row의 원래 key·JSON 값입니다. 날짜·extension·큰 정수는 native Go 모델에 넣으려고 문자열·시간·float64로 강제 변환하지 않습니다.

- 누락한 `tags`는 `[]`이고, network의 `is_port_security_enabled`/`is_router_external`과 port의 `is_port_security_enabled`는 `false`입니다. 명시한 null은 null로 유지합니다. 다른 누락 Body 속성은 null입니다.
- bool descriptor는 Python truthiness로 변환합니다. 문자열 `"False"`도 비어 있지 않으므로 true입니다. int descriptor는 Unicode decimal digit 문자열을 변환하고 JSON float를 정수로 truncation하며 Python int 하위 타입인 bool은 보존합니다.
- list descriptor의 non-null scalar는 한 원소 배열로, dict descriptor의 non-null 비객체 값은 `{}`로 변환합니다. 원본 값은 `Wire`에 있습니다.
- port/subnet에서 `project_id`가 없으면 `tenant_id`가 alias로 쓰입니다. `project_id:null`은 tenant alias로 대체하지 않습니다. `project_id`만 있는 row의 `tenant_id`는 여전히 null입니다.
- canonical·wire 키가 응답에서 함께 나오면 JSON dictionary 순서대로 마지막 값이 속성 뷰를 결정합니다. 미등록 Neutron 응답 속성은 보존하고 예약 key `self`는 속성 뷰에서만 버립니다.
- `if_match`는 Header 속성입니다. GET에서는 constructor Header seed에 실제 HTTP `If-Match`를 갱신하고 list descriptor로 반환합니다. 목록에서는 row의 `if_match`/`If-Match` constructor 속성만 소비하며 page HTTP 헤더를 개별 row의 속성으로 넣지 않습니다. 원래 page HTTP 헤더는 record metadata에 별도로 보존합니다.
- 성공한 GET의 ID·kwargs seed는 응답이 덮어쓴 부분만 갱신합니다. 204 또는 파싱할 수 없는 JSON 응답은 Source fetch처럼 seed를 반환하고 실제 Envelope/status를 남깁니다. JSON이 정상인데 row가 객체가 아닌 경우는 오류입니다.

`Resource.Body`, `Wire.Body`, record/header 값은 서로 복사본입니다. 응답 처리에서 문제가 생기면 accepted receipt의 실제 body/header/status 또는 native `ErrUnexpectedResponseCode`의 method/URL/body/header/status와 원래 physical read/close cause가 error chain에 남습니다. caller cancel cause도 `errors.Is`로 확인할 수 있습니다. SDK는 live provider token·transport·재인증을 공유하면서 route·서비스 종류·선택 microversion이 callback에서 바뀌면 작업을 중단합니다. 재시도에서 선택 version header나 그 부재·소유한 헤더를 변경할 수 없으며 native success-code 변경도 SDK의 원래 status policy를 넓히지 않습니다. Source 목록의 기본 Accept는 `application/json`이며 명시한 caller Header 옵션만 이를 덮어씁니다.

## 명시한 차이와 검증 범위

Go는 JSON-domain 입력, 구체 옵션과 escaped string identity를 사용합니다. `id` kwargs 중복, Resource/session/route control override와 보호된 auth/protocol header는 작업 전 오류입니다. 특히 inherited fetch의 `requires_id`, `error_message`, `skip_cache`, `resource_response_key`는 이 literal 검색의 query로 받지 않으며 잘못된 원래 이름의 query로 전송하지 않습니다. 목록 row의 `connection`, `microversion`, `_synchronized`는 Source constructor keyword 충돌과 같이 실제 페이지 오류로 처리합니다. Caller가 넘긴 Go map은 순서가 없으므로 GET seed에서 canonical 속성을 wire alias보다 우선합니다. 응답 JSON의 dictionary 순서는 보존합니다. Python 객체·subclass hook·warnings·임의 session 교체를 재현하지 않습니다.

이 검색 record는 호출마다 소유한 응답입니다. Python Connection의 공유 GET cache, dirty Resource identity/state, mutable Munch lifecycle와 connection-derived location은 프로젝트의 별도 공통 범위입니다. `location`은 literal 응답/seed 값이 없으면 null입니다. `WithRecordFindMicroversion`은 선택한 operation 설정을 복사하며 Resource 최대 버전을 자동 협상하지 않습니다. Neutron의 평상시 검색은 이 옵션 없이 v2.0 설정을 사용합니다.

Go는 pagination origin·collection·filter 변경, 반복 페이지와 상충하는 next link를 거부합니다. Python의 임의 경로/session 재지정과 다른 보호 규칙입니다. 알려진 malformed JSON·UTF-8·nil callback·음수 cap은 사전 오류로 처리하고, Python 런타임별 무제한 정수/Unicode 환경이나 특수 non-JSON 객체를 흉내내지 않습니다.

회귀 테스트는 각 leaf 패키지의 기존 fixture와 `internal/testcloud`를 재사용하며, 이 fixture는 Gophercloud 공개 `testhelper.SetupHTTP`를 사용합니다. 각 서비스의 `python_find_records_contracts_test.go`에 GET-first, Source defaults/alias, local filter, null/missing, late duplicate/error, raw cap, continuation/cycle, input ownership, retry/reauth와 physical cause 보존 사례가 있습니다. 테스트 실행·race 결과와 프로젝트 전체 gate 판정은 실행 로그 및 중앙 리뷰 기록을 기준으로 확인합니다. 이 문서는 실제 OpenStack 배포의 integration 통과를 주장하지 않습니다.

고정 근거는 [Network find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L3214), [Port find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L3950), [Subnet find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L7487), [Resource.find/list/fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py), [NetworkResource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_base.py), 각 [Network](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/network.py)·[Port](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/port.py)·[Subnet](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/subnet.py) descriptor입니다.
