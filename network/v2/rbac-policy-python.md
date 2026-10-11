# Neutron v2 RBAC policy 조회·목록·찾기

기준은 Gophercloud **v2.15.0**, openstacksdk **ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe**의 [RBACPolicy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/rbac_policy.py)와 [proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L5432-L5500)입니다. [Neutron RBAC API](https://docs.openstack.org/api-ref/network/v2/#rbac-policies)는 `/v2.0/rbac-policies`에서 정책을 조회합니다. 새 owned record API는 생성·수정·삭제 API의 전체 Python parity를 판정하지 않습니다.

| Python | Go | 동작 |
|---|---|---|
| `get_rbac_policy(id)` | `GetRecord(ctx, id)` | literal ID 조회, 404 terminal |
| `get_rbac_policy(id, fields=[...])` | `WithRecordGetFields(...)` | public Body 이름을 wire 이름으로 변환 |
| `rbac_policies(**query)` | `ListRecords(ctx, options...)` | lazy 전체 목록, 선언된 query/local filter |
| `list(rbac_policies(...))` | `AllRecords(ctx, options...)` | 전체 성공 시 수집, 오류 시 partial slice 폐기 |
| `find_rbac_policy(id)` | `FindRecord(ctx, id)` | GET 우선, clean400/403/404만 전체 목록 fallback |
| `find_rbac_policy(id, ignore_missing=False)` | `WithRecordFindIgnoreMissing(false)` | 찾지 못하면 `resource.ErrNotFound` |

`rbacpolicies.New(network.RawClient())`로 기존 인증 client를 공유합니다. 사용자 interface/builder 구현 없이 `WithRecord…` 함수나 owned 옵션 struct를 사용합니다.

## 독립 Python main

`main.py`로 저장합니다. 기본 작업은 network 객체에 대한 목록입니다. `--action find --id ID`는 기본적으로 없으면 `None`, `--strict`를 추가하면 예외를 반환합니다.

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--action", choices=["list", "get", "find"], default="list")
    parser.add_argument("--id", default="")
    parser.add_argument("--fields", default="")
    parser.add_argument("--object-type", default="network")
    parser.add_argument("--target-project", default="")
    parser.add_argument("--strict", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    fields = args.fields.split(",") if args.fields else None
    if args.action == "get":
        value = conn.network.get_rbac_policy(args.id, fields=fields)
    else:
        query = {"object_type": args.object_type}
        if fields:
            query["fields"] = fields
        if args.target_project:
            query["target_project_id"] = args.target_project
        if args.action == "list":
            for value in conn.network.rbac_policies(**query):
                print(json.dumps(value.to_dict(), indent=2, ensure_ascii=False))
            return
        value = conn.network.find_rbac_policy(
            args.id, ignore_missing=not args.strict, **query)
    if value is None:
        print("not found")
    else:
        print(json.dumps(value.to_dict(), indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
```

## 독립 Go main

[설치 안내](../../docs/install.md)를 적용해 `main.go`로 저장합니다. 조회 결과의 `Resource`를 출력하며 actual HTTP body가 필요하면 `Wire`·`Envelope`를 확인합니다.

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "strings"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies"
)

func printPolicy(value *rbacpolicies.RBACPolicyRecord) {
    if value == nil {
        fmt.Println("not found")
        return
    }
    raw, err := json.MarshalIndent(value.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(raw))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    action := flag.String("action", "list", "list/get/find")
    id := flag.String("id", "", "RBAC policy literal ID")
    fields := flag.String("fields", "", "comma-separated requested fields")
    objectType := flag.String("object-type", "network", "RBAC object type")
    targetProject := flag.String("target-project", "", "target project filter")
    strict := flag.Bool("strict", false, "find reports missing resource")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    network, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    api := rbacpolicies.New(network.RawClient())
    var selected []string
    if *fields != "" { selected = strings.Split(*fields, ",") }
    switch *action {
    case "get":
        value, err := api.GetRecord(ctx, *id,
            rbacpolicies.WithRecordGetFields(selected...))
        if err != nil { log.Fatal(err) }
        printPolicy(value)
    case "list":
        options := []rbacpolicies.RecordListOption{
            rbacpolicies.WithRecordListQuery("object_type", *objectType),
        }
        if len(selected) != 0 {
            options = append(options, rbacpolicies.WithRecordListQuery("fields", selected))
        }
        if *targetProject != "" {
            options = append(options, rbacpolicies.WithRecordListQuery("target_project_id", *targetProject))
        }
        for value, err := range api.ListRecords(ctx, options...) {
            if err != nil { log.Fatal(err) }
            printPolicy(value)
        }
    case "find":
        options := []rbacpolicies.RecordFindOption{
            rbacpolicies.WithRecordFindIgnoreMissing(!*strict),
            rbacpolicies.WithRecordFindQuery("object_type", *objectType),
        }
        if len(selected) != 0 {
            options = append(options, rbacpolicies.WithRecordFindQuery("fields", selected))
        }
        if *targetProject != "" {
            options = append(options, rbacpolicies.WithRecordFindQuery("target_project_id", *targetProject))
        }
        value, err := api.FindRecord(ctx, *id, options...)
        if err != nil { log.Fatal(err) }
        printPolicy(value)
    default:
        log.Fatal("unknown action")
    }
}
```

## 속성·fields·확장

RBACPolicy는 `Resource`를 직접 상속합니다. declared Body는 inherited id/name과 object_id, target_project_id, project_id, tenant_id, object_type, action의 8개이며 computed location을 포함해 기본 view는 9개입니다. 누락은 null이고 명시 null·false·0·빈 문자열·빈 object/list를 보존합니다. typed Go native 모델과 달리 arbitrary extension Body도 보존합니다. tags·revision_number·created_at·if_match는 이 Source의 descriptor가 아니므로 기본값이나 타입 변환을 추가하지 않습니다. 서버가 이 이름들을 Body extension으로 반환하면 raw JSON 값으로 보존합니다. HTTP If-Match는 proof Header에 있으며 자동 Body attr가 되지 않습니다.

`target_project_id`의 wire 이름은 `target_tenant`입니다. `project_id`는 field 자체가 없을 때 tenant_id를 alias로 읽지만 explicit project_id=null이면 tenant_id로 대체하지 않습니다. canonical/wire 이름이 함께 있는 응답은 Source의 입력 순서대로 마지막 값이 선택됩니다. 개별 `WithRecordFindQuery`는 지정 순서를 보존하고, bulk Query map은 키를 정렬하므로 unordered map의 alias 우선순위는 Python kwargs 삽입 순서와 다를 수 있습니다.

Get의 nonempty fields는 public Body 이름을 wire 이름으로 바꿉니다. `target_project_id`→`target_tenant`, unknown field는 그대로 유지하며 순서·중복을 유지합니다. nil/empty fields는 query를 생략하고 Resource에 fields seed를 넣지 않습니다. Find/List의 kwargs fields는 `_get_with_fields`를 거치지 않아 원래 값 그대로 전송됩니다. name은 inherited readable Body지만 stock RBAC 응답에 항상 있는 필드로 가정하지 않으며 wire mutation 필드나 server query로 확대하지 않습니다.

Resource는 constructor/default/alias view, Wire는 원래 row, Envelope는 실제 전체 response body입니다. 각각의 raw bytes·Header가 독립적이며 status는 실제 응답값입니다. Wire에 self는 남기고 Source처럼 Resource에서는 제거합니다. direct GET의 connection/microversion/_synchronized/cls unknown response attr는 허용하지만 LIST constructor에서는 각 key의 존재 자체가 actual page 오류입니다. cap/consumer break 전에 소비하지 않은 row의 충돌은 검사하지 않습니다.

## 찾기·목록·권한

Find는 literal ID를 escaped 단일 segment로 GET합니다. kwargs는 constructor seed와 member query에 전달되며 unknown kwargs도 GET에는 포함됩니다. clean400/403/404만 list fallback을 시작합니다. all_projects는 member query에서 분리하고 RBAC list mapping에도 없으므로 버립니다. RBAC의 name은 query mapping에 없어서 자동 name server hint를 추가하지 않습니다. 전체 페이지에서 exact id/name 일치를 찾고 중복이나 뒤쪽 오류가 있으면 실패합니다. 기본 ignore_missing=true는 clean search 뒤 nil이고 strict 옵션은 ErrNotFound입니다. Get의404는 list fallback이나 ignore_missing을 적용하지 않습니다.

LIST server mapping은 inherited limit/marker와 action/fields/object_id/object_type/project_id/target_project_id→target_tenant입니다. target_project_id와 wire alias가 함께 있으면 canonical 값이 우선하며 null이면 query를 생략합니다. bool은 Python Requests처럼 True/False, array는 반복 query, null은 생략합니다. id/name/tenant_id는 declared nonquery Body local filter입니다. unknown kwargs는 값 직렬화 전에 버리며 server가 지원하는 sort/page_reverse 등도 pinned Source mapping에 없으면 임의로 보내지 않습니다. arbitrary readable extension이 있다고 unknown query가 local filter가 되는 것은 아닙니다.

재귀 dict subset과 Python JSON bool/numeric 비교를 지원합니다. truthy scalar와 빈 dict는 일치하고 nonempty dict가 scalar의 mapping.get을 필요로 하면 actual page 오류입니다. float truthiness는 Source binary64를 적용하여 1e-4000은 false입니다. 일반 숫자 equality는 공통의 exact-decimal Go 정책입니다. Source가 9007199254740992와9007199254740993.0을 같다고 보는 반올림이나 1e-4000==0을 성립시키는 underflow를 equality에 적용하지 않습니다. 큰 정수와 원래 float token은 Wire/Resource에 그대로 남습니다.

목록은 lazy 전체 페이지가 기본입니다. max_items는 local filtering 이전 raw row 수이며 omitted/zero limit에 hint를 적용합니다. paginated=false는 첫 페이지의 모든 row를 처리하고 consumer break는 즉시 멈춥니다. rbac_policies_links, top-level next, HTTP Link, singleton object, short-page marker fallback 및 빈 페이지 종료를 기존 common REST로 처리합니다. marker는 받은 Wire id에서 구하고 고정 origin/path/filter·loop guard를 유지합니다. dictionary links는 Go 호환 확장입니다. Go는 numeric controls를 엄격히 검증하고 marker를 하나의 nonempty query value로 제한합니다. Source의 임의 객체·iterable cursor 및 permissive adapter 동작을 모두 구현하지 않습니다.

Neutron **27.0.0**의 [get_rbac_policy 정책](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/rbac.py)은 project scope의 ADMIN_OR_PROJECT_READER이며 owner project reader가 조회할 수 있습니다. 공유 대상 project가 underlying network를 이용할 수 있다는 사실만으로 RBAC policy owner 권한을 얻지는 않습니다. 목록·조회 API를 읽기 대상으로 추가했고 create/update/delete와 wildcard target 관련 권한은 별도 정책입니다. 서버의 배포 policy override와 extension 지원은 실제 응답이 결정합니다.

## 버전·오류·검증 범위

Source RBACPolicy에는 `_max_microversion`이 없습니다. version discovery를 추가하지 않으며 configured version 또는 absence를 유지합니다. `WithRecordGet/ListMicroversion`은 해당 조회/모든 목록 페이지에 적용되고 Find의 per-call override는 GET에만 적용됩니다. Find fallback LIST는 원래 configured default/absence를 사용합니다. shared client는 변경하지 않습니다. transport setting 옵션은 Go가 제공하는 소유 확장입니다.

공통 Neutron reader의 기존200..399 admission을 사용합니다. Source raise_from_response의 <400 기준과 달리 합성100..199 response는 이 Go reader가 성공으로 받아들이지 않는 명시 정책입니다. 일반 net/http는 interim1xx를 처리합니다. 모든400..599 rejection과 accepted response의 physical Read/Close/cancel/source/callback 원인을 보존합니다. member accepted empty/malformed JSON은 Source처럼 ID/kwargs seed를 유지하고 valid nonobject/null envelope는 actual receipt 오류입니다. accepted 오류는 ResponseError에, rejected 오류는 native ErrUnexpectedResponseCode에 실제 status/body/header/method 증거가 남습니다.

LIST Accept는 기본 application/json으로 configured Accept보다 우선하고 operation Headers로만 바꿉니다. selected network version/absence와 owned headers, 다른 서비스 version·Authorization/X-Service-Token/Cookie/Host/framing의 absence를 native retry에서도 고정합니다. 일반 retry header와 provider의 live X-Auth-Token은 native 정책대로 동작합니다. source가 Read에서 바뀌고 Close에서 원복되더라도 실패는 유지되며 physical 원인이 있는404를 missing이나 fallback으로 바꾸지 않습니다.

nil API/context/option·invalid ID/known JSON/control·protected header는 HTTP 전 오류입니다. location·route·session·constructor controls는 SDK가 소유합니다. 동적 Python Resource/Munch/subclass·mutable lifecycle·Connection-derived location·HTTP cache hit/invalidation은 제공하지 않습니다. 반복 Get은 HTTP를 수행하며 증거가 없는 성공 status/body를 만들지 않습니다.

[회귀 테스트](extensions/rbacpolicies/python_records_contracts_test.go)는 기존 native RBAC fixture와 public Gophercloud testhelper 기반 internal/testcloud를 사용합니다. 별도 transport/pager/server 구현을 추가하지 않았습니다. root 통합 검증이 package/race와 이 standalone main을 실행하며 이 문서는 실행 전 통과나 실제 Neutron integration 성공을 주장하지 않습니다.

### Find 예약 인자와 선택 버전

`FindRecord`의 raw Query에 `rbac_policy`, `name_or_id`, `ignore_missing` 키가 있으면 값이 null·false·0이어도 HTTP 전에 오류가 납니다. Python의 공개 proxy와 내부 `_find`가 이미 이 인자들을 바인딩하므로 일반 query나 constructor seed로 다시 전달할 수 없습니다. missing 정책은 `WithRecordFindIgnoreMissing`으로 지정합니다. LIST는 별도 호출이며 일반 unknown kwargs 규칙을 유지합니다.

멤버 GET/Find 응답에 unknown `microversion` 키가 있으면 `Wire`는 서버의 값을 그대로 보존합니다. `Resource`의 같은 키는 실제 선택된 SDK 버전 문자열 또는 선택 버전이 없는 경우 null을 담습니다. Source의 런타임 Resource 속성이 unknown Body 값보다 우선하는 동작이며, 응답 확장 값이 요청 버전을 선택하지는 않습니다.
