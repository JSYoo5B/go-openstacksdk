# Cloud QoS 정책 목록·검색: Python과 Go

고정 Source는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, native 기준은 Gophercloud `v2.15.0`입니다. Cloud facade는 `conn.Network(ctx)`로 얻고, 실제 Resource API는 `conn.NetworkV2(ctx).QoSPolicies`에 있습니다.

| Python cloud helper | Go cloud facade |
| --- | --- |
| `conn.list_qos_policies(filters=...)` | `ListQoSPolicyRecords(ctx, WithQoSPolicyFilters(rawJSON))` |
| `conn.search_qos_policies(name_or_id=..., filters=...)` | `SearchQoSPolicyRecords(ctx, identity, WithQoSPolicyFilters(rawJSON))` |
| `filters=None` 및 falsey filters | 옵션 생략 또는 JSON null/false/0/empty 값 |
| filter kwargs의 `headers`, `microversion`, `base_path` | raw filters 그대로 Source Proxy lane으로 전달 |
| Go 전용 concrete header/version 편의 옵션 | `WithQoSPolicyHeaders`, `WithQoSPolicyMicroversion` |
| Resource 결과와 실제 HTTP 증거 | `Policies`, `Extensions`, 각 record의 `Resource`·`Wire`·`Envelope`·`Header`·`StatusCode` |
| `list(Proxy)`의 arbitrary iterable 결과 | carrier `Value`, fully JSON일 때만 `JSON` |

[고정 Neutron 27 QoS 정책](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L42-L71)의 `get_policy`는 project scope에서 admin 또는 해당 project reader 또는 shared QoS policy를 허용합니다. SDK가 역할을 확대하지 않으며 배포의 변경된 정책과 실제 인증이 최종 권한을 결정합니다. 목록/검색은 QoS extension을 조회한 뒤 정책 GET만 수행하며 정책 생성·변경·삭제 권한을 사용하지 않습니다.

## 독립 Python

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--mode", choices=("list", "search"), default="list")
parser.add_argument("--name", default="")
parser.add_argument("--filters", default="null")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    filters = json.loads(args.filters)
    if args.mode == "search":
        values = conn.search_qos_policies(name_or_id=args.name, filters=filters)
    else:
        values = conn.list_qos_policies(filters=filters)
    for value in values:
        if hasattr(value, "to_dict"):
            value = value.to_dict()
        print(json.dumps(value, indent=2))
```

예: `python main.py --cloud dev --mode search --name gold --filters '{"is_shared":true}'`. Dictionary filters는 실제 method가 kwargs로 넘기며 전체 collection을 모은 뒤 `_filter_list`를 실행하는 API가 아닙니다.

## 독립 Go main

[설치 안내](../docs/install.md)의 module에서 `go run . -cloud dev -mode search -name gold -filters '{"is_shared":true}'`로 실행합니다.

```go
package main

import (
    "bytes"
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    mode := flag.String("mode", "list", "list or search")
    name := flag.String("name", "", "literal server name query")
    filters := flag.String("filters", "null", "JSON filters")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.Network(ctx)
    if err != nil { log.Fatal(err) }
    options := []network.QoSPolicyReadOption{
        network.WithQoSPolicyFilters(json.RawMessage(*filters)),
    }
    var result *network.QoSPolicyRecords
    switch *mode {
    case "list":
        result, err = service.ListQoSPolicyRecords(ctx, options...)
    case "search":
        result, err = service.SearchQoSPolicyRecords(ctx, *name, options...)
    default:
        log.Fatal("mode must be list or search")
    }
    if err != nil {
        if result != nil {
            log.Printf("completed extension records: %d", len(result.Extensions))
        }
        log.Fatal(err)
    }
    if result.JSON == nil {
        fmt.Printf("opaque iterable result: %T\n", result.Value)
        return
    }
    var readable any
    decoder := json.NewDecoder(bytes.NewReader(result.JSON))
    decoder.UseNumber()
    if err := decoder.Decode(&readable); err != nil { log.Fatal(err) }
    data, err := json.MarshalIndent(readable, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

## 실제 실행 순서와 필터

각 호출은 cache 없이 Extension 목록을 끝까지 순회하여 alias 집합을 만든 뒤 `qos`를 검사합니다. 첫 page에서 qos를 발견해도 뒤 page의 실패를 숨기지 않습니다. Extension이 없으면 `errors.Is(err, network.ErrUnavailableExtension)`으로 구별하며 policy HTTP는 발생하지 않습니다. Extension의 실제 receipt와 실패한 요청의 오류 chain은 별도 증거입니다.

Filter 옵션의 JSON 검증과 user option callback은 이 gate 뒤에 실행합니다. 누락·null·false·0·-0.0·JSON float underflow·empty string/array/object는 `{}`입니다. Truthy list filters는 `**filters`용 mapping이어야 하며 truthy string을 JMESPath 표현식으로 해석하지 않습니다. Search는 truthy 문자열 identity를 name query에 넣은 뒤 Python `dict.update`처럼 mapping/JSON iterable-of-pairs를 병합합니다. 이후 filters.name이 null이면 identity name query도 제거되고, override 값이면 그 값을 서버로 보냅니다. 문자열 pair 항목은 Unicode character 두 개를 key/value로 쓸 수 있지만 최종 kwargs key는 문자열이어야 합니다.

Search는 ID member GET·glob match·client name predicate를 추가하지 않습니다. `name_or_id="literal/id*"`도 서버 `name=literal/id*`입니다. Server filter가 다른 이름의 행을 반환해도 cloud wrapper가 다시 제거하지 않습니다.

Source QoSPolicy 선언 query는 canonical/wire aliases를 library가 처리합니다. `is_shared`는 `shared`, `any_tags`는 `tags-any`이며 `fields`·ID 배열은 반복 query입니다. Local Body filters는 tenant_id와 rules입니다. Query/filter용으로 선언되지 않은 값은 Resource.list가 버립니다. 선언/default/alias/unknown response JSON은 owned PolicyRecord이고 큰 정수와 실제 body/Header/status를 Wire/Envelope에 유지합니다. Raw max_items는 local filtering 이전 행을 세며 single page·raw cap·terminal error와 continuation은 공통 owned pager 계약을 따릅니다.

## Proxy list()와 Go 차이

Cloud helper는 Python `list(self.network.qos_policies(**query))`입니다. Deprecated `jmespath_filters`는 수집한 Policy 배열 대신 실제 lazy generator에 평가합니다. `@`는 policy 목록을 소비하고, `` `[1,2]` ``는 policy HTTP 없이 `[1,2]`, string literal은 Unicode character 목록을 돌려줍니다. `[0]`의 null과 number/bool 결과는 iterable이 아니므로 policy HTTP 없이 오류입니다. Extension gate는 그 전에 이미 수행합니다.

`to_array(@)`는 generator 한 개를 담은 배열입니다. Value에 opaque `*policies.RecordListGenerator`가 보존되며 JSON은 nil입니다. Nested generator를 소비하지 않거나 HTTP가 발생한 것처럼 receipt를 만들지 않습니다. Source JMES dictionary의 list()는 삽입순서의 keys지만 공통 Go evaluator는 map을 반환하므로 Go는 keys를 lexical order로 고정합니다. 예를 들어 `` `{"z":1,"a":2}` `` 결과가 Python에서는 `["z","a"]`, Go에서는 `["a","z"]`입니다. 이 순서 차이를 Source와 같다고 표시하지 않습니다.

Raw headers·microversion·base_path와 ordered __conflicting_attrs는 새 Source Proxy lane에서 소비합니다. Source 제어 값의 안전한 string-header map, service-relative path·origin/filter/cycle 제한, malformed UTF8/JSON preflight와 eager descriptor view는 명시적인 Go 정책입니다. 이 lane만 Source JSON link precedence·CPython-loaded numeric equality·physical fault 정책에 opt-in하고 기존 typed List/Get의 예약 query·pager/equality 계약을 유지합니다. Python의 mutable Resource/cache/session/custom class/Connection-derived location은 공통 잔여 기반 계약으로 따로 추적합니다.

실패하면 carrier Policies/Value/JSON에 partial policy 목록을 남기지 않습니다. 이미 완료한 Extensions는 남으며 actual rejected/late/body/Close/cancel/source 오류에는 실제 receipt/원인을 유지합니다. SDK-owned protocol guard는 clean retry/live token renewal을 유지하고 physical failure나 selected route/protocol 변경을 missing/success로 바꾸지 않습니다.

## 검증

기존 Gophercloud testhelper 기반 `internal/testcloud`와 실제 response body observer를 재사용합니다. Gate 전체 소비·매 호출 재조회·필터 단계/alias/update·arbitrary Proxy iterable·safe base path·Source link/equality·atomic late failure·option snapshot·concurrent factory reuse·physical fault/clean retry를 계약 테스트로 확인합니다. Root의 focused/race/기존 Policies 회귀/독립 main 검증 ledger가 실행 결과이며 이 가이드는 실제 배포 검증이나 미실행 PASS를 주장하지 않습니다.

Source: [Cloud list/search](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L253-L348), [extension gate](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L43-L50), [Proxy._list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866-L916), [QoSPolicy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_policy.py).
