# Cloud QoS 정책 조회·삭제: Python과 Go

고정 Source는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, native 기준은 Gophercloud `v2.15.0`입니다. Cloud facade는 `conn.Network(ctx)`이며 실제 Resource API는 `conn.NetworkV2(ctx).QoSPolicies`입니다.

| Python | Go |
| --- | --- |
| `conn.get_qos_policy(name_or_id, filters=...)` | `GetQoSPolicyRecord(ctx, identity, WithQoSPolicyFilters(rawJSON))` |
| 조회 없음: `None` | nil error와 `result.Policy == nil` |
| `conn.delete_qos_policy(name_or_id)` | `DeleteQoSPolicyRecord(ctx, identity)` |
| 조회 없음: `False` | `result.Deleted == false`, Delete receipt 없음 |
| 조회 뒤 DELETE 성공/ignored404: `True` | `result.Deleted == true`, 실제 Delete receipt 보존 |
| 현재 호출의 extension 발견 결과 | carrier `Extensions` |
| 조회 Resource·HTTP·삭제 HTTP | `Policy.Resource`·`Policy.Wire`·`Policy.Envelope`와 별도 `Delete.Envelope/Header/StatusCode` |

[Neutron 27 고정 정책](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L42-L165)의 get_policy는 project reader의 소유 policy 또는 shared policy(및 admin)를 허용합니다. Delete_policy는 project scope에서 admin 또는 해당 project manager를 요구하고, historic deprecated 정책은 admin-only였습니다. SDK가 역할을 확대하지 않으며 배포의 정책 변경과 실제 인증이 최종 권한을 결정합니다. Shared policy를 읽을 수 있다는 사실은 그 policy의 삭제 권한을 보장하지 않습니다.

## 독립 Python

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--mode", choices=("get", "delete"), default="get")
parser.add_argument("--identity", required=True)
parser.add_argument("--filters", default="null")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "delete":
        print(json.dumps(conn.delete_qos_policy(args.identity)))
    else:
        value = conn.get_qos_policy(args.identity, filters=json.loads(args.filters))
        print(json.dumps(None if value is None else value.to_dict(), indent=2))
```

기본 조회: `python main.py --cloud dev --identity gold`. 삭제: `python main.py --cloud dev --mode delete --identity gold`.

## 독립 Go main

[설치 안내](../docs/install.md)의 module에서 `go run . -cloud dev -identity gold` 또는 `go run . -cloud dev -mode delete -identity gold`로 실행합니다.

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
    "github.com/JSYoo5B/go-openstacksdk/network"
)

func printJSON(value any) {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    mode := flag.String("mode", "get", "get or delete")
    identity := flag.String("identity", "", "literal ID or exact name")
    filters := flag.String("filters", "null", "get-only JSON kwargs")
    flag.Parse()
    if *identity == "" { log.Fatal("identity is required") }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.Network(ctx)
    if err != nil { log.Fatal(err) }
    switch *mode {
    case "get":
        result, err := service.GetQoSPolicyRecord(
            ctx, *identity, network.WithQoSPolicyFilters(json.RawMessage(*filters)),
        )
        if err != nil { log.Fatal(err) }
        if result.Policy == nil { printJSON(nil); return }
        printJSON(result.Policy.Resource)
    case "delete":
        result, err := service.DeleteQoSPolicyRecord(ctx, *identity)
        if err != nil { log.Fatal(err) }
        printJSON(result.Deleted)
    default:
        log.Fatal("mode must be get or delete")
    }
}
```

## Gate·조회·누락

각 호출은 cache 없이 Extension 목록을 끝까지 읽어 `qos` alias를 확인합니다. 첫 page에서 qos를 발견해도 뒤 page 오류·unhashable alias를 숨기지 않습니다. 미지원 extension은 `errors.Is(err, network.ErrUnavailableExtension)`이며 policy GET/DELETE를 수행하지 않습니다. 이미 완료한 extension evidence는 carrier에 남습니다.

Get의 filters는 gate 뒤에 해석합니다. 누락/null/false/0/JSON float underflow/empty string·array·object는 `{}`입니다. Truthy filters는 mapping이어야 하며 docstring의 string JMES 예제와 달리 실제 method는 `**filters`를 사용하므로 truthy string은 오류입니다. Mapping 안의 jmespath_filters는 ordinary member GET query/constructor seed이고 fallback List가 버리는 unknown attr입니다. Cloud get은 `_filter_list`나 glob/local identity predicate를 별도로 적용하지 않습니다.

Lookup은 literal identity를 한 segment로 escape하여 GET을 먼저 수행합니다. Query kwargs/constructor seed를 보존하고 실제 응답이 seed를 덮습니다. Clean400/403/404에서만 완전한 list fallback을 수행합니다. Declared name query hint와 server aliases 및 tenant_id/rules local filters를 library가 제공하며 exact ID/name matches의 후반 중복은 ErrAmbiguous입니다. 정상 조회의 absence는 get에서 nil,nil이고 delete에서는 false입니다. LIST403·후반 오류·physical read/Close/cancel/source fault는 absence로 바꾸지 않습니다.

Microversion은 member GET만 덮고 fallback은 원래 selected version입니다. ListBasePath는 fallback collection만 바꾸며 성공한 member GET에서는 경로를 검증하거나 사용하지 않습니다. Source lane은 CPython-loaded equality·JSON link precedence·100..399 completion·physical-stop 정책에 opt-in하고 기존 typed Find/List/Delete는 baseline을 유지합니다.

## Get 지원 범위와 정확한 remaining

기본 조회·ordinary JSON kwargs·body defaults/aliases·missing/fallback·member microversion·fallback-only list_base_path·실제 증거는 구현했습니다. 그러나 `get_qos_policy` 전체 named kwargs는 현재 **unresolved**이며 완성된 API 개수에 넣지 않습니다.

Raw kwargs key headers는 Source Resource.find에서 member GET query/constructor seed로 전달되고, fallback Resource.list에서는 headers formal로 소비됩니다. 현재 shared findKey는 이를 거부합니다. Concrete `WithQoSPolicyHeaders`는 HTTP header 편의 옵션이며 Source의 이 두 단계 kwargs 동작을 대체하지 않습니다.

Fetch controls requires_id·base_path·error_message·skip_cache·resource_response_key도 named filters로 도달하지만 shared findKey가 거부합니다. Source는 각각 member URI ID 포함 여부·member base path·오류 메시지·adapter cache·response envelope를 바꾸고 fallback 단계에서도 자신의 formal/unknown 규칙을 따릅니다. Member base_path와 이미 구현한 fallback-only list_base_path는 별도입니다. 이 remaining은 실제 단계별 공통 구현·테스트가 필요합니다.

## Delete 결과·ID·실제 receipt

Delete에는 filters 인자가 없습니다. Fresh gate 뒤 default ignore_missing=True Find를 한 번 수행하며 lookup 없음에서 DELETE 없이 false를 반환합니다. Lookup이 있으면 **반환된 Resource.id**를 사용하여 bodyless member DELETE를 보냅니다. Name나 caller identity를 실제 응답 ID 대신 다시 쓰지 않습니다.

Source Proxy의 delete ignore_missing 기본값도 True이므로 clean DELETE404는 조회 이후 경합의 absence를 무시하고 cloud 결과는 true입니다. Carrier Delete에는 실제404 status/body/header와 IgnoredMissing=true가 남습니다. 403/409/500 등은 terminal error이며 조회 receipt와 실제 DELETE receipt를 분리합니다. Successful malformed/empty body를 Resource로 decode하거나 가짜 resource를 만들지 않습니다. Physical404·Close failure·context cancellation은 true로 숨기지 않습니다.

숫자/true 같은 nonstring JSON ID는 공통 Python string conversion을 사용하므로 큰 integer ID를 float64로 훼손하지 않습니다. Go의 URI 정책은 literal member segment입니다. Null/false/0/underflow/empty returned ID는 pre-DELETE 오류입니다. Source는 null에서 InvalidRequest이고 utils.urljoin은 falsey nonnull ID를 collection 경로 `qos/policies/`로 만들 수 있습니다. Go가 이 collection DELETE를 재현하지 않는 경로 정책 차이를 명시하며, false나 성공으로 표시하지 않습니다. Slash/percent를 포함한 ID도 Go에서는 한 segment로 escape합니다.

Options와 Resource·Wire·Envelope·Header는 library-owned snapshots입니다. Mutable Python Resource/cache/class/session/current_location과 custom runtime 객체는 공통 SDK 기반 remaining으로 따로 추적합니다. Selected source/protocol guard는 native clean retry와 live token renewal을 유지하며 physical/cancel/source failure에서는 retry/reauth callback을 실행하지 않습니다. Native401은 unwrap하지 않는 ErrUnableToReauthenticate의 ErrOriginal과 value ErrUnexpectedResponseCode로 실제 증거를 읽고 ErrReauth의 처리 원인을 함께 보존합니다.

## 검증

기존 Gophercloud testhelper의 internal/testcloud HTTP fixture와 실제 response body observer를 재사용합니다. Source AST reference와 default/missing/filter/fallback/ID/deletion receipt, 24 physical/cancel/source 경계, clean retry/reauth, Source199 opt-in 및 기존 typed baseline 회귀를 검증합니다. Root의 focused/race/독립 main/generator ledger가 실제 실행 결과이며 배포 권한·real backend 성공 또는 미실행 PASS를 주장하지 않습니다.

Source: [Cloud get](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L220-L250), [Cloud delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1512-L1536), [Resource.find/fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py), [Proxy delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L670-L713).
