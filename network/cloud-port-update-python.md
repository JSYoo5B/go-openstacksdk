# Cloud port 수정: Python과 Go

기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, Gophercloud `v2.15.0`입니다. `service.UpdatePortRecord`가 옵션 소유권, cloud 키 검증, ID/이름 조회, 변경 감지와 PUT을 처리합니다. 애플리케이션에 builder 구현을 요구하지 않습니다.

| Python | Go |
| --- | --- |
| `conn.update_port(identity, **attrs)` | `service.UpdatePortRecord(ctx, identity, network.WithPortUpdateJSON(raw))` |
| 생략한 kwargs | 조회만 수행; `NoOp=true`, `Lookup`에 기존 record, `Port=nil` |
| 조회 raw 값과 같은 kwargs | PUT 없이 같은 lookup record 제공 |
| 명시 `None`, `False`, 빈 list | JSON `null`, `false`, `[]`; 누락과 구분 |
| 없는 대상의 SDKException | `errors.Is(err, resource.ErrNotFound)`; PUT 없음 |
| 반환 Resource | `Lookup`은 조회 evidence, `Port`는 실제 PUT의 native typed 결과 |

허용 키는 `name`, `admin_state_up`, `fixed_ips`, `security_groups`, `allowed_address_pairs`, `extra_dhcp_opts`, `device_owner`, `device_id`, `binding:vnic_type`, `binding:profile`, `port_security_enabled`, `qos_policy_id`, `binding:host_id`입니다. 다른 cloud 키, canonical alias인 `is_admin_state_up`, `security_group_ids`, `binding_profile`, `if_revision`과 `filters`는 조회 전에 거부합니다. 확장 서버 필드나 권한 검증은 서버가 담당합니다.

`WithPortUpdateName`, `WithPortUpdateAdminStateUp`, `WithPortUpdateSecurityGroups`, `WithPortUpdatePortSecurityEnabled`, `WithPortUpdateDeviceID`, `WithPortUpdateDeviceOwner`, `WithPortUpdateQoSPolicyID`와 `WithPortUpdateAttribute`를 제공합니다. JSON 옵션은 전체 kwargs를 대체하고 이후 옵션이 같은 키를 덮습니다. bytes/map은 복제합니다. 예를 들어 `WithPortUpdateAttribute("qos_policy_id", nil)`은 명시 null입니다.

기존 `ports.FindRecord`를 사용하여 member GET부터 시도하고 필요한 경우 전체 이름 검색을 수행합니다. kwargs가 없어도 이 조회는 생략하지 않습니다. 변경값만 반환된 port ID에 PUT하고 별도의 재시도 정책을 추가하지 않습니다. 기존 provider의 RetryFunc/ReauthFunc 등 설정은 유지하므로 모든 설정에서 단일 전송을 보장하지 않습니다. boolean/number Python equality와 raw presence를 비교하므로 `true`와 `1`은 같고, 누락과 명시 null은 다릅니다. getter가 변환한 view 대신 조회 Wire 값을 비교합니다. `admin_state_up:"false"`와 update `true`는 서로 다른 raw 값입니다.

## 독립 Python 예제

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--identity", required=True)
parser.add_argument("--attrs", default="{}")
args = parser.parse_args()
attrs = json.loads(args.attrs)
if not isinstance(attrs, dict):
    parser.error("attrs must be a JSON object")
with openstack.connect(cloud=args.cloud) as conn:
    result = conn.update_port(args.identity, **attrs)
    print(json.dumps(result.to_dict(), indent=2))
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
    identity := flag.String("identity", "", "port ID or exact name")
    attrs := flag.String("attrs", "{}", "cloud update kwargs as JSON")
    flag.Parse()
    if *identity == "" { log.Fatal("identity is required") }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.Network(ctx)
    if err != nil { log.Fatal(err) }
    result, err := service.UpdatePortRecord(ctx, *identity,
        network.WithPortUpdateJSON(json.RawMessage(*attrs)))
    if err != nil { log.Fatal(err) }
    var value any = result.Port
    if result.NoOp { value = result.Lookup.Resource }
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

조회만: `go run . -identity example`.
수정: `go run . -identity example -attrs '{"name":"renamed","security_groups":[]}'`.
수정 예제는 실제 서버 상태를 변경합니다.

## 남은 차이

이 helper는 전체 Python mutable Resource 계약을 닫지 않습니다. 조회의 `Lookup.Wire/Envelope/Header/StatusCode`는 조회 응답 증거이며 PUT receipt가 아닙니다. `Port`는 기존 native typed decoder를 사용하므로 unknown extension 응답 필드, 부분 응답의 기존 값 병합, 빈/잘못된 성공 JSON에서 Python이 기존 Resource를 유지하는 동작과 같은 인스턴스 반환은 제공하지 않습니다. 이 때문에 해당 연산 판정은 unresolved입니다.

일반 wire-key 응답을 대상으로 변경 감지합니다. GET/LIST 응답의 canonical attribute 키만 있는 경우와 canonical/wire 키가 함께 들어오는 충돌·alias 우선순위, 정수와 부동소수점 혼합에서 원래 Python kwargs 타입을 보존하는 비교는 남은 경계입니다. JSON 유한값·문자열 ID 범위를 사용하며 임의 Python 객체, nonfinite JSON, 동적 descriptor와 전체 Resource/adapter/session/cache 정책은 SDK-R1/C1/S1의 별도 계약입니다. 이 예제는 실클라우드 검증 결과가 아닙니다.

집중 테스트는 `TestCloudPortUpdateLookupOnlyAndDirtySelection`, `TestCloudPortUpdateNameFallbackAndMissing`, `TestCloudPortUpdateOwnedOptionsAndPreflight`, `TestCloudPortUpdateConflictKeepsLookupAndSingleAttempt`입니다. 기존 scripted Neutron fixture는 공개 Gophercloud testhelper를 사용하는 `internal/testcloud`를 재사용합니다.

Native provider의 RetryFunc/ReauthFunc는 기존 동작을 유지합니다. 기본 409 응답의 단일 PUT 회귀가 모든 hook 설정에서 mutation 재전송 방지를 입증하지는 않습니다.

최종 root 검증: 수리 후 집중4그룹13사례·독립 Go main1개 빌드·Python AST1개·전체205개 test package와 license/vet/race/parity/progress/gofmt PASS. 실클라우드 검증은 별도입니다.
