# Cloud firewall policy 수정: Python과 Go

기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, Gophercloud `v2.15.0`입니다. `service.UpdateFirewallPolicyRecord`가 cloud 인자 검증, strict 대상 policy 조회, rule 참조 변환과 raw 변경 감지를 처리합니다. SDK 소유 functional options를 사용하며 애플리케이션 builder 구현은 필요하지 않습니다.

| Python | Go |
| --- | --- |
| `conn.update_firewall_policy(identity, filters=filters, **attrs)` | `service.UpdateFirewallPolicyRecord(ctx, identity, options...)` |
| kwargs 없음 | policy strict 조회 후 PUT 생략; `NoOp=true` |
| `firewall_rules=[name_or_id, ...]` | policy 조회 뒤 입력 순서대로 rule strict 조회·반환 ID 치환 |
| `firewall_rules=[]` | rule 조회 없이 빈 list로 변경 감지·필요하면 PUT |
| 같은 ID list로 해석되는 참조 | rule 조회까지 수행하고 PUT 생략 |
| 없는 policy 또는 rule | strict 오류; PUT 없음 |
| 반환 mutable Resource | `Lookup`, `RuleLookups`, native `Policy` 분리 |

허용 kwargs는 `audited`, `description`, `firewall_rules`, `name`, `project_id`, `shared`의 6개입니다. 나머지 키와 잘못된 attrs JSON은 lookup 전에 거부합니다. `WithFirewallPolicyUpdateName`, `WithFirewallPolicyUpdateAudited`, `WithFirewallPolicyUpdateShared`, `WithFirewallPolicyUpdateDescription`, `WithFirewallPolicyUpdateRules`와 일반 `WithFirewallPolicyUpdateAttribute`, `WithFirewallPolicyUpdateJSON/Opts`를 제공합니다. `WithFirewallPolicyUpdateRules()`는 명시 빈 list입니다. JSON 옵션은 attrs 전체를 교체하며 뒤의 옵션이 같은 키를 덮습니다. SDK가 JSON/map을 복제합니다.

먼저 `firewall_policies` strict Find에 `WithFirewallPolicyUpdateFilters`를 적용합니다. `firewall_rules` 키가 있으면 **대상 policy를 찾은 다음에** rule 참조를 해석합니다. 각 rule은 필터 없이 strict Find하며 policy filters를 전달하지 않습니다. 참조 순서와 중복을 유지하므로 같은 이름이 두 번 나오면 조회도 두 번 합니다. 앞선 rule을 찾은 뒤 다음 rule에서 실패하면 `Lookup`과 앞선 `RuleLookups`가 남고 이후 rule과 PUT은 실행하지 않습니다. 응답 ID는 비어 있지 않은 문자열 범위이며 policy PUT route용 ID는 escape합니다.

Go는 JSON string-reference array 범위를 제공합니다. null·문자열·object·nonstring array item은 policy lookup 뒤 오류이며 임의 Python iterable을 확장하지 않습니다. 빈 array는 정상 입력이고 rule 조회가 없습니다. `firewall_rules` 외의 명시 null/false/빈 값은 raw 값으로 비교·전달합니다. rule ID 변환 뒤 ordinary wire raw 값에 Python equality를 적용해 dirty 필드만 PUT합니다. `true`와 `1`은 같고 누락과 null은 다릅니다. 결과의 `NoOp`은 PUT을 생략했다는 뜻이며 policy/rule lookup 생략을 뜻하지 않습니다.

Filters의 falsey JSON null/false/0/빈 문자열·array·object는 {}처럼 처리합니다. Truthy filters는 object여야 합니다. `name_or_id`와 `ignore_missing` 충돌은 null이어도 조회 전 오류입니다. member GET·이름 fallback과 실패 증거는 기존 `firewallDeleteProfile`/`neutronfind`를 재사용합니다. FWaaS extension/service gate나 cache를 추가하지 않습니다.

`Lookup`은 policy 조회, `RuleLookups`는 입력 순서의 rule 조회에서 얻은 Resource/Wire/Envelope/Header/StatusCode입니다. 실패한 참조의 metadata-only record가 포함될 수도 있으므로 Resource가 항상 있다고 가정하지 않습니다. 이들은 PUT receipt가 아닙니다. `Policy`는 native PUT typed 결과이고 실패하면 nil입니다. SDK 내부 private builder가 raw 변경값을 upstream update에 연결하며 사용자에게 interface 책임을 넘기지 않습니다. 기존 provider retry/reauthentication 설정을 유지하므로 모든 설정에서 단일 전송을 보장하지 않습니다.

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
    result = conn.update_firewall_policy(args.identity, filters=json.loads(args.filters), **attrs)
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
    identity := flag.String("identity", "", "firewall policy ID or exact name")
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
    result, err := service.UpdateFirewallPolicyRecord(ctx, *identity,
        network.WithFirewallPolicyUpdateJSON(json.RawMessage(*attrs)),
        network.WithFirewallPolicyUpdateFilters(json.RawMessage(*filters)))
    if err != nil { log.Fatal(err) }
    var value any = result.Policy
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
규칙 지정: `go run . -identity example -attrs '{"firewall_rules":["allow-ssh","allow-dns"]}'`.
규칙 비우기: `go run . -identity example -attrs '{"firewall_rules":[]}'`.
효과 있는 kwargs는 실제 서버 상태를 변경합니다.

## 남은 차이와 검증 범위

일반 JSON 문자열 참조 목록을 지원하며 Python 문자열·mapping·임의 iterable iteration, arbitrary object/kwargs 타입과 비문자열·빈 ID의 경로 변환은 별도 범위입니다. null의 target-first 오류 순서는 유지하지만 Python TypeError와 동일한 오류 객체를 반환하지 않습니다.

Native upstream PUT의 HTTP 200 성공 정책과 typed decoder를 유지합니다. Source 전체 accepted status 정책, unknown 필드·부분 응답 병합, 빈/잘못된 성공 JSON에서 기존 Resource 유지, 같은 mutable instance 반환, 원본 PUT Envelope/Header/StatusCode는 구현하지 않습니다. 조회 evidence를 mutation receipt로 해석하지 않습니다.

Raw Find `headers`/`microversion`의 초기 query와 fallback formal 차이 및 `base_path`, `skip_cache`, `resource_response_key` 등의 fetch/constructor controls는 완전히 구현하지 않습니다. 현재 unsupported reserved filter는 HTTP 전 오류입니다. 전체 Requests encoding/nonfinite JSON, mixed numeric/object equality, Resource/adapter/session/cache·동적 descriptor 수명 주기는 SDK-R1/C1/S1의 별도 계약입니다. 연산은 unresolved이며 실클라우드·릴리즈 검증 완료가 아닙니다.

집중 테스트는 `TestCloudFirewallPolicyUpdateLookupOnlyAndDirtySelection`, `TestCloudFirewallPolicyUpdateNameFallbackAndMissing`, `TestCloudFirewallPolicyUpdateOwnedOptionsAndPreflight`, `TestCloudFirewallPolicyUpdateConflictKeepsLookupAndSingleAttempt`, `TestCloudFirewallPolicyUpdateFiltersStrictBindingAndFalseyDefaults`, `TestCloudFirewallPolicyUpdateOrderedRulesTargetFirstAndEmptyClear`, `TestCloudFirewallPolicyUpdateRuleFailureKeepsPriorEvidence`, `TestCloudFirewallPolicyUpdateUnsupportedRuleShapesAfterTargetLookup`입니다. 기존 scripted fixture와 공개 Gophercloud testhelper 기반 `internal/testcloud`를 재사용합니다.

Root 최종 검증: 첫 집중13그룹52사례(policy8/31+rule5/21), 독립 Go main/Python AST 각1개, 전체205개 test package 및 license/vet/race/parity/progress/gofmt PASS. 실클라우드 검증은 별도입니다.
