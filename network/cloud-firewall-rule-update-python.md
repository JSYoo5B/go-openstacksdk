# Cloud firewall rule 수정: Python과 Go

기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, Gophercloud `v2.15.0`입니다. `service.UpdateFirewallRuleRecord`가 cloud kwargs 검증, strict 이름/ID 조회, raw 변경 감지를 처리합니다. SDK 소유의 functional options를 사용하며 애플리케이션에 builder 구현을 요구하지 않습니다.

| Python | Go |
| --- | --- |
| `conn.update_firewall_rule(identity, filters=filters, **attrs)` | `service.UpdateFirewallRuleRecord(ctx, identity, options...)` |
| kwargs 없음 | strict 조회는 수행, PUT은 생략; `NoOp=true`, `Lookup` 존재 |
| raw 값이 같은 kwargs | 조회 뒤 PUT 생략; `Rule=nil` |
| 명시 None/false/빈 문자열·list | JSON null/false/빈 값 그대로 dirty 비교와 PUT |
| 없는 대상 | strict Find 오류; PUT 없음 |
| 반환 mutable Resource | 조회 evidence `Lookup`와 native PUT 결과 `Rule` 분리 |

허용 kwargs는 `action`, `description`, `destination_firewall_group_id`, `destination_ip_address`, `destination_port`, `enabled`, `ip_version`, `name`, `project_id`, `protocol`, `shared`, `source_firewall_group_id`, `source_ip_address`, `source_port`의 14개입니다. FW group ID 두 키는 고정 Resource의 선언 필드가 아니어도 cloud decorator가 허용하는 unknown body입니다. SDK가 이를 제거하지 않습니다. 이 allowlist 밖의 키와 잘못된 JSON은 lookup 전에 거부합니다. 서버 정책이나 필드 유효성을 SDK가 새로 강제하지 않습니다.

`WithFirewallRuleUpdateName`, `WithFirewallRuleUpdateAction`, `WithFirewallRuleUpdateEnabled`, `WithFirewallRuleUpdateShared`, `WithFirewallRuleUpdateProtocol`, 일반 `WithFirewallRuleUpdateAttribute`와 `WithFirewallRuleUpdateJSON/Opts`를 제공합니다. JSON 옵션은 전체 attrs를 바꾸고 뒤의 옵션이 같은 키를 덮습니다. SDK가 attrs와 filters의 JSON/map을 복제합니다. `WithFirewallRuleUpdateFilters`는 별도 strict Find kwargs입니다.

Filters의 falsey JSON null/false/0/빈 문자열·array·object는 {}처럼 처리합니다. Truthy filters는 object여야 하며 docstring의 JMESPath 문자열을 실제 구현이 지원한다고 해석하지 않습니다. `name_or_id`, `ignore_missing`은 이미 bound된 인자이므로 filters에 null로 넣어도 lookup 전에 충돌 오류입니다. Filters는 member GET와 필요한 fallback LIST에 기존 `firewallDeleteProfile("firewall_rules")`/`neutronfind` 정책으로 전달합니다. 새 extension/service gate나 cache는 추가하지 않습니다. 뒤의 중복·HTTP 오류도 관찰하고 strict missing을 정상 no-op으로 바꾸지 않습니다.

일반 wire-key raw 값과 Python equality로 비교하며 `true`와 `1`은 같습니다. null과 누락은 다르고, `enabled:"false"`를 getter bool로 바꾸어 비교하지 않습니다. 변경 필드만 반환된 ID에 PUT하고 ID는 member route용으로 escape합니다. `protocol:"ANY"`, `protocol:null`, 숫자·문자열 kwargs를 SDK가 lower-case나 typed enum으로 변환하지 않습니다. 기존 generated `rules.Update`의 core-field 제한을 피하기 위해 **SDK 내부의 private builder**가 upstream raw PUT에 연결됩니다. 이 interface는 애플리케이션 옵션이나 확장 책임으로 노출하지 않습니다.

조회 `Lookup`은 원본 Wire/Envelope/Header/StatusCode를 보관하며 Source summary 기본값·location=null view를 기존 helper로 구성합니다. 이는 PUT receipt가 아닙니다. PUT `Rule`은 native typed decoder 결과이며 실패하면 nil입니다. 기본 provider의 conflict 단일 시도 테스트는 모든 retry/reauthentication 설정에서 단일 전송을 보장하지 않습니다. 기존 provider 설정은 유지합니다.

## 독립 Python 예제

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--identity", required=True)
parser.add_argument("--attrs", default="{}")
parser.add_argument("--filters", default="{}")
args = parser.parse_args()
attrs = json.loads(args.attrs)
if not isinstance(attrs, dict):
    parser.error("attrs must be a JSON object")
with openstack.connect(cloud=args.cloud) as conn:
    result = conn.update_firewall_rule(args.identity, filters=json.loads(args.filters), **attrs)
    print(json.dumps(None if result is None else result.to_dict(), indent=2))
```

## 독립 Go 예제

[설치 안내](../docs/install.md)의 module에서 다음을 main.go로 저장합니다.

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

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    identity := flag.String("identity", "", "firewall rule ID or exact name")
    attrs := flag.String("attrs", "{}", "cloud update kwargs as JSON")
    filters := flag.String("filters", "{}", "JSON Source Find kwargs")
    flag.Parse()
    if *identity == "" { log.Fatal("identity is required") }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.Network(ctx)
    if err != nil { log.Fatal(err) }
    result, err := service.UpdateFirewallRuleRecord(ctx, *identity,
        network.WithFirewallRuleUpdateJSON(json.RawMessage(*attrs)),
        network.WithFirewallRuleUpdateFilters(json.RawMessage(*filters)))
    if err != nil { log.Fatal(err) }
    var value any = result.Rule
    if result.NoOp {
        value = nil
        if result.Lookup != nil { value = result.Lookup.Resource }
    }
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

조회만: `go run . -identity example`.
수정: `go run . -identity example -attrs '{"enabled":false,"description":""}'`.
이름 조회 필터: `go run . -identity example -filters '{"project_id":"owner"}' -attrs '{"action":"deny"}'`.
효과 있는 kwargs는 실제 서버 상태를 변경합니다.

## 남은 차이와 검증 범위

Native upstream update는 HTTP 200을 성공으로 받는 계약을 유지하며 Source의 전체 accepted status 정책을 복제하지 않습니다. Native typed decoder는 unknown 응답 필드를 버리고 bool·int·문자열 타입에 제약이 있습니다. 부분 응답 병합, 빈/잘못된 성공 JSON에서 기존 Resource 유지, 같은 mutable instance 반환과 원본 PUT Envelope/Header/StatusCode도 제공하지 않습니다. 조회 evidence를 mutation receipt로 해석하지 않습니다.

Raw Find filters의 `headers`/`microversion`은 common engine에서 Source의 member-query/formal fallback 의미를 모두 재현하지 못하며 이 profile에서는 HTTP 전 오류입니다. `base_path`, `skip_cache`, `resource_response_key`와 나머지 fetch/constructor control namespace도 일반 query로 완전히 구현하지 않습니다. 호출자가 임의 transport header를 쓰도록 이 차이를 숨기지 않습니다. 전체 Requests encoding/nonfinite JSON, 원래 Python kwargs 객체·혼합 numeric 타입, Resource/adapter/session/cache와 동적 descriptor 수명 주기는 SDK-R1/C1/S1의 별도 계약입니다. 연산은 unresolved이며 실클라우드·릴리즈 검증 완료가 아닙니다.

집중 테스트는 `TestCloudFirewallRuleUpdateLookupOnlyAndDirtySelection`, `TestCloudFirewallRuleUpdateNameFallbackAndMissing`, `TestCloudFirewallRuleUpdateOwnedOptionsAndPreflight`, `TestCloudFirewallRuleUpdateConflictKeepsLookupAndSingleAttempt`, `TestCloudFirewallRuleUpdateFiltersStrictBindingAndFalseyDefaults`입니다. 기존 scripted Neutron fixture와 공개 Gophercloud testhelper 기반 `internal/testcloud`를 재사용합니다.

Root 최종 검증: 첫 집중13그룹44사례(FW5/21+router8/23), 독립 Go main/Python AST 각1개, 전체205개 test package와 license/vet/race/parity/progress/gofmt PASS. 실클라우드 검증은 별도입니다.
