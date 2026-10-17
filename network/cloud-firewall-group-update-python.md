# Cloud firewall group 수정: Python과 Go

기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, Gophercloud `v2.15.0`입니다. `service.UpdateFirewallGroupRecord`가 cloud 인자 검증, strict 대상 group 조회, policy/port 참조 변환과 raw 변경 감지를 처리합니다. SDK 소유 functional options를 사용하며 애플리케이션 builder 구현은 필요하지 않습니다.

| Python | Go |
| --- | --- |
| `conn.update_firewall_group(identity, filters=filters, **attrs)` | `service.UpdateFirewallGroupRecord(ctx, identity, options...)` |
| kwargs 없음 | group strict 조회 후 PUT 생략; `NoOp=true` |
| `egress_firewall_policy`, `ingress_firewall_policy` 이름/ID | group 조회 뒤 egress→ingress 고정 순서로 strict 조회·wire ID 치환 |
| policy alias에 `None` | 해당 policy 조회 없이 `*_firewall_policy_id:null` |
| `ports=[name_or_id,...]` | policy 단계 뒤 순서·중복을 유지하며 port 조회·반환 ID 치환 |
| `ports=[]` | port 조회 없이 빈 목록 변경 감지·필요하면 PUT |
| 같은 참조 ID로 해석됨 | 참조 조회까지 수행하고 PUT 생략 |

허용 kwargs는 `admin_state_up`, `description`, `egress_firewall_policy`, `ingress_firewall_policy`, `name`, `ports`, `project_id`, `shared`의 정확히 8개입니다. **고정 Source docstring은 직접 `egress_firewall_policy_id`/`ingress_firewall_policy_id`를 넣을 수 있다고 설명하지만 실제 valid_kwargs decorator에는 두 키가 없어 거부됩니다.** 이 cloud helper도 두 direct ID 키를 lookup 전에 거부합니다. 실제 허용 alias 값에 ID를 넣으면 lookup을 거쳐 사용합니다.

`WithFirewallGroupUpdateName`, `WithFirewallGroupUpdateAdminStateUp`, `WithFirewallGroupUpdateShared`, `WithFirewallGroupUpdateEgressPolicy`, `WithFirewallGroupUpdateIngressPolicy`, `WithFirewallGroupUpdatePorts`와 일반 `WithFirewallGroupUpdateAttribute`, `WithFirewallGroupUpdateJSON/Opts`를 제공합니다. `WithFirewallGroupUpdatePorts()`는 명시 빈 목록입니다. null policy는 JSON 또는 `WithFirewallGroupUpdateAttribute("egress_firewall_policy", nil)`로 지정합니다. JSON 옵션은 전체 attrs를 교체하고 뒤의 옵션이 같은 키를 덮습니다. SDK가 JSON/map을 복제합니다.

먼저 `firewall_groups` strict Find에 `WithFirewallGroupUpdateFilters`를 적용합니다. 입력 JSON 키 순서와 무관하게 egress policy, ingress policy 순서로 처리합니다. alias 키가 null이면 해당 wire ID를 null로 만들고 조회하지 않습니다. nonnull alias는 비어 있지 않은 문자열 참조 범위이며 strict policy lookup을 수행합니다. 그 뒤에 ports 목록을 처리합니다. 모든 참조 lookup에 group filters를 전달하지 않습니다. 동일 참조가 반복되면 lookup도 반복하며 반환 ID 순서와 중복을 유지합니다.

`ports`의 JSON 문자열 참조 array는 target/policy 단계 뒤 해석합니다. null·문자열·object·nonstring 항목은 해당 단계까지 수행한 뒤 오류이며 Source의 임의 Python iterable을 확장하지 않습니다. `ports:[]`는 정상 clear 입력입니다. 중간 참조 실패 시 이후 lookup/PUT을 하지 않고 이전 evidence가 남습니다. policy 별칭과 ports 외의 명시 null/false/빈 값은 raw 비교·전달합니다. 참조 ID 치환 뒤 ordinary wire raw 값의 Python equality로 dirty 필드만 PUT합니다. `NoOp`은 PUT 생략이며 모든 lookup의 생략을 뜻하지 않습니다.

Filters의 falsey JSON null/false/0/빈 문자열·array·object는 {}처럼 처리합니다. Truthy filters는 object여야 하고 `name_or_id`/`ignore_missing` 충돌은 null이어도 조회 전 오류입니다. 기존 `firewallDeleteProfile`/`neutronfind`, `ports.FindRecord`와 공개 testhelper 기반 fixture를 재사용하며 extension/service gate나 cache를 추가하지 않습니다. port helper는 Source get_port처럼 기본 missing 허용 lookup 뒤 nil이면 명시 NotFound로 바꿉니다.

`Lookup`은 group 원본 조회 record, `PolicyLookups`는 실제 egress→ingress policy 조회 record, `PortLookups`는 실제 입력 순서의 port 조회 record입니다. null policy에는 record가 추가되지 않습니다. 실패 record는 metadata-only일 수 있으므로 Resource가 항상 있다고 가정하지 않습니다. 원본 lookup Wire/Envelope/Header/StatusCode는 PUT receipt가 아닙니다. 응답 ID는 비어 있지 않은 문자열 범위이며 group PUT route ID를 escape합니다. `Group`은 native typed PUT 결과이고 실패하면 nil입니다. SDK 내부 private builder가 raw attrs를 upstream에 전달하며 사용자에게 interface 구현을 요구하지 않습니다. 기존 provider retry/reauthentication 설정은 유지되므로 모든 설정에서 단일 전송을 보장하지 않습니다.

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
    result = conn.update_firewall_group(args.identity, filters=json.loads(args.filters), **attrs)
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
    identity := flag.String("identity", "", "firewall group ID or exact name")
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
    result, err := service.UpdateFirewallGroupRecord(ctx, *identity,
        network.WithFirewallGroupUpdateJSON(json.RawMessage(*attrs)),
        network.WithFirewallGroupUpdateFilters(json.RawMessage(*filters)))
    if err != nil { log.Fatal(err) }
    var value any = result.Group
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
참조 지정: `go run . -identity example -attrs '{"egress_firewall_policy":"outbound","ingress_firewall_policy":"inbound","ports":["app-port"]}'`.
연결 비우기: `go run . -identity example -attrs '{"egress_firewall_policy":null,"ingress_firewall_policy":null,"ports":[]}'`.
효과 있는 kwargs는 실제 서버 상태를 변경합니다.

## 남은 차이와 검증 범위

참조는 JSON nonempty string/null policy와 nonempty string-array ports 범위입니다. 임의 Python 객체·iterable·nonstring/빈 응답 ID와 runtime 예외 객체는 동일하게 재현하지 않습니다. null ports의 target/policy 뒤 오류 순서는 유지합니다. Source docstring의 direct-ID 설명과 decorator의 실제 거부는 구분하며 이를 새로운 편의 동작으로 우회하지 않습니다.

참조 port 조회는 기존 `ports.FindRecord`의 status/encoding/Resource profile을 사용하며 firewall engine과 동일한 전체 Source 정책을 충족했다고 주장하지 않습니다.

Native upstream PUT의 HTTP 200 성공 정책과 typed decoder를 유지합니다. Source의 전체 accepted status, unknown 필드·부분 응답 병합, 빈/잘못된 성공 JSON에서 기존 Resource 유지, 같은 mutable instance 반환, 원본 PUT Envelope/Header/StatusCode는 구현하지 않습니다. Raw Find `headers`/`microversion`의 초기 query와 fallback formal 차이, `base_path`, `skip_cache`, `resource_response_key` fetch/constructor controls와 전체 Requests encoding/nonfinite JSON·mixed numeric/object equality·Resource/adapter/session/cache·동적 descriptor는 SDK-R1/C1/S1의 별도 계약입니다. 일부 reserved filter는 HTTP 전 오류입니다. 연산은 unresolved이고 실클라우드·릴리즈 검증 완료가 아닙니다.

집중 테스트는 `TestCloudFirewallGroupUpdateLookupOnlyAndDirtySelection`, `TestCloudFirewallGroupUpdateNameFallbackAndMissing`, `TestCloudFirewallGroupUpdateOwnedOptionsAndPreflight`, `TestCloudFirewallGroupUpdateConflictKeepsLookupAndSingleAttempt`, `TestCloudFirewallGroupUpdateFiltersStrictBindingAndFalseyDefaults`, `TestCloudFirewallGroupUpdateTargetPolicyPortsOrderAndEmptyClear`, `TestCloudFirewallGroupUpdateNullPoliciesAndEmptyPorts`, `TestCloudFirewallGroupUpdateReferenceFailuresPreservePhaseOrder`, `TestCloudFirewallGroupUpdateResolvedEqualReferencesStillLookUp`입니다. 기존 scripted fixture와 공개 Gophercloud testhelper 기반 `internal/testcloud`를 재사용합니다.

Root 최종 검증: 첫 집중17그룹60사례(group9/29+policy8/31), 독립 Go main/Python AST 각1개, 전체205개 test package와 license/vet/race/parity/progress/gofmt PASS. 실클라우드 검증은 별도입니다.
