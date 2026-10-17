# Cloud subnet 수정: Python과 Go

기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, Gophercloud `v2.15.0`입니다. `service.UpdateSubnetRecord`가 기본 인자, kwargs 선택, ID/이름 조회, raw 변경 감지를 처리합니다. SDK 소유 옵션을 제공하므로 애플리케이션 builder 구현은 필요하지 않습니다.

| Python | Go |
| --- | --- |
| `conn.update_subnet(identity, **args)` | `service.UpdateSubnetRecord(ctx, identity, network.WithSubnetUpdateJSON(raw))` |
| 효과 있는 인자 없음 → None/HTTP 0 | `NoOp=true`, `Lookup=nil`, `Subnet=nil`; 서비스/대상 조회 생략 |
| 기존 raw 값과 같음 → 조회 Resource | `NoOp=true`, `Lookup` 존재, `Subnet=nil`; PUT 없음 |
| `enable_dhcp=False` | `WithSubnetUpdateEnableDHCP(false)`; 생략하지 않고 전달 |
| `disable_gateway_ip=True` | `WithSubnetUpdateDisableGatewayIP(true)`; `gateway_ip:null` 전달 |
| 없는 대상 | `errors.Is(err, resource.ErrNotFound)`; PUT 없음 |

허용 인자는 `subnet_name`, `enable_dhcp`, `gateway_ip`, `disable_gateway_ip`, `allocation_pools`, `dns_nameservers`, `host_routes`의 7개입니다. `name`이나 `is_dhcp_enabled`, 확장 키, `if_revision`, `filters`는 이 cloud 메서드의 인자가 아닙니다. `WithSubnetUpdateName`은 `subnet_name`을 받아 wire `name`으로 번역합니다. `WithSubnetUpdateAttribute`, `WithSubnetUpdateJSON`, `WithSubnetUpdateOpts`와 DHCP/gateway/DNS 편의 옵션을 제공합니다. JSON 옵션은 전체 인자를 교체하며 이후 옵션이 같은 키를 덮습니다. SDK가 JSON/map을 복제합니다.

`enable_dhcp`만 null이 아니면 값을 전달합니다. 따라서 false, 0, 빈 문자열·list도 effective입니다. 나머지 인자는 Python truthy일 때만 선택하며 빈 문자열·list·object, null, false, 0은 생략합니다. 빈 DNS/route/allocation list로 값을 지우는 계약을 추가하지 않습니다. `disable_gateway_ip`가 truthy이면 gateway를 null로 만들며, `gateway_ip`도 truthy이면 조회 전에 오류입니다.

인자 판정 후 효과 있는 body가 없으면 서비스가 nil이거나 identity가 비어 있어도 HTTP 없이 성공합니다. Go context 취소와 잘못된 옵션은 이 no-op으로 숨기지 않습니다. 효과 있는 body가 있을 때 서비스/API 유효성을 확인하고 client source를 고정합니다. 옵션 적용 중 API/client 포인터 교체는 감지하지만 고정 전에 옵션이 직접 바꾼 client 내부 설정까지 port와 동일한 capture-before-options 보장을 주장하지 않습니다.

조회는 기존 `subnets.FindRecord`의 member GET와 필요한 이름 검색을 재사용합니다. 일반 wire-key raw 응답에서 Python equality로 변경값만 비교하여 반환된 ID에 PUT합니다. `true`와 `1`은 같고, raw `enable_dhcp:"false"`와 `true`는 다릅니다. `Lookup`의 Wire/Envelope/Header/StatusCode는 조회 응답 증거입니다. PUT 결과는 별도 native `Subnet`이며 거부된 PUT에는 nil을 유지합니다. 기존 provider retry/reauthentication 설정은 유지하고, 모든 설정에서 단일 전송을 보장하지 않습니다.

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
    result = conn.update_subnet(args.identity, **attrs)
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
    identity := flag.String("identity", "", "subnet ID or exact name")
    attrs := flag.String("attrs", "{}", "cloud update kwargs as JSON")
    flag.Parse()
    if *identity == "" { log.Fatal("identity is required") }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.Network(ctx)
    if err != nil { log.Fatal(err) }
    result, err := service.UpdateSubnetRecord(ctx, *identity,
        network.WithSubnetUpdateJSON(json.RawMessage(*attrs)))
    if err != nil { log.Fatal(err) }
    var value any = result.Subnet
    if result.NoOp {
        value = nil
        if result.Lookup != nil { value = result.Lookup.Resource }
    }
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

인자 없는 호출: `go run . -identity example`.
DHCP 비활성화: `go run . -identity example -attrs '{"enable_dhcp":false}'`.
이름 수정: `go run . -identity example -attrs '{"subnet_name":"renamed"}'`.
효과 있는 인자는 실제 서버 상태를 변경합니다.

## 남은 차이와 검증 범위

이 helper는 전체 Python mutable Resource 계약을 닫지 않습니다. PUT은 native typed decoder를 사용하므로 unknown extension 응답 필드, 부분 응답의 기존 값 병합, 빈/잘못된 성공 JSON에서 기존 Resource 유지, 같은 mutable instance 반환을 제공하지 않습니다. 원본 PUT Envelope/Header/StatusCode도 이 bridge가 노출하지 않습니다. 조회의 원본 receipt를 mutation receipt로 해석하지 않습니다.

Member GET/LIST가 canonical attribute 키만 사용하거나 wire 키와 함께 보내는 경우의 constructor/response mapping·alias 우선순위, 원래 Python kwargs의 임의 타입·혼합 numeric 타입 비교가 남습니다. 기본 lookup decoder의 전체 Requests encoding/비유한 JSON 동작, 동적 descriptor와 전체 Resource/adapter/session/cache 정책은 SDK-R1/C1/S1의 별도 계약입니다. 실클라우드·릴리즈 검증도 완료한 것이 아닙니다. 연산 판정은 unresolved를 유지합니다.

집중 테스트는 `TestCloudSubnetUpdateEffectiveValuesAndDirtySelection`, `TestCloudSubnetUpdateIneffectiveAndConflictBeforeServiceLookup`, `TestCloudSubnetUpdateOptionCancellationCannotBecomeNoOp`, `TestCloudSubnetUpdateNameFallbackAndMissing`, `TestCloudSubnetUpdateOwnedOptionsAndPreflight`, `TestCloudSubnetUpdateConflictKeepsLookupAndSingleAttempt`입니다. 기존 scripted Neutron fixture와 공개 Gophercloud testhelper 기반 `internal/testcloud`를 재사용합니다.

최종 root 검증: 타입 수리 후 서브넷6그룹15사례와 포트 회귀4그룹13사례, 독립 Go main1개 빌드·Python AST1개, 전체205개 test package 및 license/vet/race/parity/progress/gofmt PASS. 최초 컴파일/예제 실패와 수리 내역은 판정대장에 별도 기록했습니다. 실클라우드 검증은 별도입니다.
