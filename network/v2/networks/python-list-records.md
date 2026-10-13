# Network의 Python/Go 목록 레코드

비교 기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, Gophercloud `v2.15.0`입니다. `conn.NetworkV2(ctx).Networks`의 `ListRecords`·`AllRecords`는 기본값과 확장 속성을 SDK가 처리하는 resource 목록입니다. `ProxyListRecords`는 deprecated JMESPath까지 포함한 고정 Python `conn.network.networks(**query)`의 반환 유형을 보존합니다.

| Python | Go |
| --- | --- |
| `conn.network.networks(**query)` | `ProxyListRecords(ctx, options...)` |
| 기본 resource generator 순회 | `ListRecords(ctx, options...)` |
| `list(conn.network.networks(**query))` | `AllRecords(ctx, options...)` |
| `name=...`, `is_router_external=True` | `WithRecordListQuery("name", value)`, `WithRecordListQuery("is_router_external", true)` |
| `max_items=n`, `paginated=False` | `WithRecordListMaxItems(n)`, `WithRecordListPaginated(false)` |
| `headers=...`, `microversion=...`, `base_path=...` | `WithRecordListHeaders`, `WithRecordListMicroversion`, `WithRecordListBasePath` |
| `__conflicting_attrs=...` | `WithRecordListConflictingAttrs(json.RawMessage(...))` |
| `jmespath_filters=expression` | `ProxyListRecords(..., WithRecordListJMESPath(expression))` |
| `value.to_dict()` | `value.Resource`의 JSON |
| 원래 행·실제 페이지와 HTTP metadata | `Wire`, `Envelope`, `Header`, `StatusCode` |

이 문서는 하나의 named operation `python:network/v2/networks`에 대한 것입니다. Python cloud의 `list_networks`·`search_networks`는 별도 연산이며 서비스 존재 검사와 cloud orchestration까지 이 leaf로 완료했다고 세지 않습니다.

## 독립 Python 프로그램

`clouds.yaml`의 `dev` cloud로 실행합니다. 기본 목록과 실제 Proxy의 JMESPath 분기를 같은 프로그램에서 비교할 수 있습니다.

```python
import argparse
import json
import types
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--name")
parser.add_argument("--max-items", type=int, default=0)
parser.add_argument("--jmes")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    query = {}
    if args.name is not None:
        query["name"] = args.name
    if args.max_items:
        query["max_items"] = args.max_items
    if args.jmes is not None:
        query["jmespath_filters"] = args.jmes
    result = conn.network.networks(**query)
    if isinstance(result, types.GeneratorType):
        for value in result:
            print(json.dumps(value.to_dict(), indent=2))
    else:
        print(result)
```

## 독립 Go main

[설치 안내](../../../docs/install.md)에 따라 module을 준비한 뒤 `go run . -cloud dev -name public`으로 실행합니다. `-jmes '[0]'`는 고정 Source처럼 네트워크 HTTP를 보내지 않고 `null`을 출력합니다. `-jmes '@'`는 generator를 반환하므로 예제의 순회 시 HTTP가 시작됩니다.

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
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    name := flag.String("name", "", "server-side network name filter")
    maximum := flag.Int("max-items", 0, "raw row cap; zero is unbounded")
    expression := flag.String("jmes", "", "deprecated Python Proxy expression")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    network, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    options := []networks.RecordListOption{
        networks.WithRecordListMaxItems(*maximum),
        networks.WithRecordListJMESPath(*expression),
    }
    if *name != "" {
        options = append(options, networks.WithRecordListQuery("name", *name))
    }
    result, err := network.Networks.ProxyListRecords(ctx, options...)
    if err != nil { log.Fatal(err) }
    if result.JSON != nil {
        fmt.Println(string(result.JSON))
        return
    }
    generator, ok := result.Value.(*networks.RecordListGenerator)
    if !ok {
        // Expressions such as to_array(@) retain nested generator values.
        // They have no JSON representation and do not issue a request here.
        fmt.Printf("opaque expression result: %T\n", result.Value)
        return
    }
    for value, err := range generator.Records {
        if err != nil { log.Fatal(err) }
        data, err := json.MarshalIndent(value.Resource, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Println(string(data))
    }
}
```

## 속성·기본값·확장

Network의 전체 effective Body는 29개입니다. 상속된 id·revision_number·tags를 포함하며 빠진 `is_port_security_enabled`·`is_router_external`은 false, tags는 빈 배열, 나머지 Body와 if_match는 null입니다. 실제 null은 기본값으로 교체하지 않습니다. List는 caller query를 row seed로 사용하지 않습니다.

Resource는 canonical 속성 이름으로, Wire는 원래 행의 이름과 값으로 보존합니다. 예를 들어 `router:external`, `provider:network_type`, `provider:segmentation_id`, `subnets`, `vlan_qinq`는 각각 `is_router_external`, `provider_network_type`, `provider_segmentation_id`, `subnet_ids`, `is_vlan_qinq`로 읽습니다. 응답의 canonical/wire가 함께 있으면 JSON dictionary 순서의 마지막 값이 적용됩니다. Network의 project_id에는 tenant_id fallback이 없으므로 tenant_id 확장은 별도 그대로 남습니다.

Bool은 Python truthiness를 따릅니다. `"false"`는 true이고, 0·빈 container는 false, `1e-400`은 binary64 underflow로 false입니다. List 속성은 nonnull scalar를 한 항목 배열로 만들며 null을 유지합니다. Integer descriptor는 bool을 유지하고 float를 binary64 후 절단하며 Unicode decimal-digit 문자열을 처리합니다. Unknown extension·timestamp의 원래 text·큰 정수·nested JSON을 보존하고 self는 Resource에서 제거합니다.

List Header descriptor는 row의 exact `if_match` 또는 대소문자를 무시하는 wire `if-match`에서 읽습니다. Whole-page HTTP If-Match는 각 row의 descriptor가 되지 않으며 실제 Header에 남습니다. `IF_MATCH`는 unknown Body입니다. Resource·Wire·Envelope·Header의 bytes/map은 독립 snapshot입니다.

Resource.location은 현재 SDK가 소유하는 null이며 실제 row location은 Wire에 남습니다. Python current_location/Computed constructor 값을 합성하지 않습니다. 반환 JSON view는 전체 선언을 투영하므로 Python descriptor getter를 나중에 읽을 때 생길 오류가 Go에서는 row 반환 전에 발생할 수 있습니다. Mutable Resource/cache/dirty lifecycle는 프로젝트의 공통 SDK 계약에서 별도로 추적합니다.

## Query·로컬 필터·페이지

Server query의 canonical 이름은 limit·marker·description·fields·name·status·project_id·sort_key·sort_dir·id, ipv4_address_scope_id·ipv6_address_scope_id, is_admin_state_up·is_port_security_enabled·is_router_external·is_shared, provider_network_type·provider_physical_network·provider_segmentation_id, tags·any_tags·not_tags·not_any_tags의 23개입니다. Wire 이름도 받아 총 35개 spelling을 처리합니다. Canonical이 wire alias보다 presence로 우선하므로 null·false·zero를 구분합니다. Query bool은 True/False, 배열은 Requests의 반복 query 형태입니다. 목록 fields는 원문 이름을 보내며 GetWithFields처럼 canonical 속성을 변환하지 않습니다.

Server query에 없는 canonical Body 14개는 local filter입니다: availability_zone_hints·availability_zones·created_at·dns_domain·is_default·mtu·pvlan·qos_policy_id·segments·subnet_ids·updated_at·is_vlan_transparent·is_vlan_qinq·revision_number. Nested dictionaries는 Source의 subset/falsey 규칙, arrays는 전체 값, bool-number는 Python 동등 비교를 적용합니다. Local predicates는 옵션 순서로 평가하며 먼저 false이면 뒤의 predicate 오류를 읽지 않습니다. 새 record profile의 숫자 비교는 JSON number를 CPython처럼 읽습니다. Float token은 binary64로 반올림하고 underflow/overflow를 적용하며 integer token은 임의 정밀도를 유지합니다. 따라서 9007199254740992와 9007199254740993.0, 0과 1e-400은 같고 두 개의 순수 큰 integer는 합치지 않습니다. 기존 native/semantic Collection의 exact-decimal 비교 정책은 변경하지 않습니다. Body wire 이름과 unknown kwargs는 값의 JSON marshaler 호출 전에 버립니다.

기본 ListRecords는 lazy이고 List limit은 생략합니다. Positive max_items는 raw rows를 제한하고 limit hint를 줄 수 있으며 local filter에서 제외한 행도 cap에 포함됩니다. Paginated false와 consumer break는 다음 페이지를 요청하지 않습니다. 네이티브 typed List/All의 전체 페이지 decode나 기본 모델 제한을 새 레코드 API에 강제하지 않습니다.

공통 pager는 links·networks_links·top-level next·HTTP Link, limit/marker fallback 및 versioned collection alias를 읽습니다. 원래 origin/collection/filter를 유지하며 foreign target·filter rewrite·cycle·truthy 잘못된 next는 실제 whole-page 증거를 남기고 실패합니다. Dictionary links는 Go 편의이며 Source의 모든 permissive adapter routing을 재현하지 않습니다. Falsey next의 float truthiness는 Python binary64를 따릅니다. AllRecords는 late failure에서 partial slice를 버리고, iterator consumer는 이미 반환된 행을 보유합니다.

Named headers/microversion/base_path는 `WithRecordListQuery`의 Source 철자 또는 concrete 전용 함수로 지정하며 순차 옵션의 마지막 값이 적용됩니다. None은 기본값을 선택하고 explicit empty microversion은 version Header를 생략합니다. Header는 안전한 이름과 string 값의 concrete map만 지원하며 Source의 None-valued header 제거는 재현하지 않습니다.

`__conflicting_attrs`는 한 번만 object 순서대로 원래 kwargs에 복원합니다. 기존 키의 위치를 유지하고 새로운 키를 끝에 넣습니다. Falsey 값은 무시하고 truthy nonobject는 오류입니다. Proxy가 이미 Resource.list에 전달한 paginated/base_path가 복원되면 중복 인수 오류를 보존합니다. 복원한 headers/microversion/max_items는 해당 Source formal 값이 되고, 복원한 jmespath_filters는 Resource.list unknown 인수로 버려 원래 Proxy 표현식을 교체하지 않습니다. 자기 안의 __conflicting_attrs를 다시 펼치지 않습니다. cls/session/self/resource_type는 bound 인수 충돌로 presence만으로 거부합니다.

allow_unknown_params는 고정 Resource.list가 값과 무관하게 `_validate(..., allow_unknown_params=True)`를 사용하므로 소비하고 무시합니다. list_base_path·connection·_synchronized·requires_id·error_message·skip_cache·resource_response_key·location은 이 목록의 Body/Query가 아니어서 값 평가 없이 버립니다. Row의 constructor 충돌은 목록 kwargs와 별개의 실제 응답 검사입니다.

Base path는 같은 service의 상대 경로로 지정하고 원래 selected client를 수정하지 않습니다. Continuation은 새 경로의 collection에 고정됩니다. Foreign URL·query·fragment·traversal은 HTTP 전에 거부합니다. Source의 임의 adapter retargeting과 percent-format URI는 지원하지 않습니다. Empty base_path는 Go에서 기본 collection을 선택하는 편의입니다; 고정 Source는 empty URI에서 요청 없이 끝납니다. Typed option을 넘길 때의 client/argument 검증은 Source의 일부 generator 내부 검증보다 일찍 실행될 수 있습니다.

## 실제 Proxy의 deprecated JMESPath

고정 `Proxy._list`는 Resource.list generator를 JSON 배열로 수집하지 않고 그대로 jmespath.search에 넘깁니다. 따라서 `[*].id`, `[0]`, `type(@)`는 null로 끝나며 HTTP가 없습니다. `length(@)`·`sort_by(@,&id)`·`contains(@,\`x\`)`는 type error, `@`·`not_null(@,\`[]\`)`는 generator, `to_array(@)`는 generator를 담은 배열입니다. `@ == @`와 중첩 동일 generator의 equality도 보존합니다.

ProxyListRecords의 Value에 해당 결과가 남습니다. Fully JSON인 결과만 JSON에 encoding되며 generator 또는 nested generator는 JSON이 nil입니다. `to_string(@)`와 generator 자체의 JSON marshal은 generator를 소비하지 않고 실패합니다. Falsey/nonstring jmespath_filters는 Source처럼 무시됩니다. Resource-only ListRecords/AllRecords는 truthy string expression을 ProxyListRecords로 안내하는 오류로 처리합니다. Python cloud `_filter_list`가 실제 배열을 수집한 뒤 수행하는 표현식과 이 Proxy 분기를 혼동하지 않습니다.

## HTTP 오류·권한·검증 범위

Completed status 200..399를 읽고 400..599는 native 거부 오류를 보존합니다. Synthetic informational status 100..199와 Requests replacement UTF-8 decoding은 이 Go completion profile과 다릅니다. Accepted decode/descriptor/local-filter 오류에는 실제 status·whole-body·header가 붙고 물리 Read·Close·source 변경·취소 오류는 retry/reauth를 실행하지 않습니다. Native 401의 `*ErrUnableToReauthenticate`는 ErrOriginal에 실제 응답, ErrReauth에 처리 실패 원인을 유지합니다. 깨끗한 503 retry와 401 token 갱신은 허용하지만 callback이 선택된 body/version/조건/auth/route를 바꾸면 실제 receipt와 함께 실패합니다.

Neutron 27.0.0의 기본 `get_network`는 project reader, service, shared/external network 및 advsvc 경로를 허용합니다. Provider 속성 세 개와 segments는 별도 ADMIN 정책입니다. 일반 user 응답에서 숨겨진 속성은 선언의 null 기본값으로 남으며 SDK가 권한을 올리지 않습니다. [고정 Neutron 정책](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/network.py#L196-L265).

테스트는 기존 Gophercloud testhelper를 사용하는 internal/testcloud 및 기존 네트워크 recorder/물리 body fixture를 재사용합니다. Fixture·race·독립 main build는 실제 인증된 cloud 정책/extension/backend 검증과 구별합니다. 이 가이드 자체는 테스트를 실행했다고 주장하지 않습니다.

Source: [Proxy.networks](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L3258), [Network](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/network.py#L17), [Proxy._list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866), [Resource.list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2155).
