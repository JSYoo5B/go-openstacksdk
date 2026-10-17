# Cloud router 수정: Python과 Go

기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, Gophercloud `v2.15.0`입니다. `service.UpdateRouterRecord`가 기본 인자 선택, external gateway 조립, 선택적인 extraroute 조회와 ID/이름 변경 작업을 처리합니다. SDK 소유 functional options를 사용하며 애플리케이션 builder 구현은 필요하지 않습니다.

| Python | Go |
| --- | --- |
| `conn.update_router(identity, **args)` | `service.UpdateRouterRecord(ctx, identity, network.WithRouterUpdateJSON(raw))` |
| 효과 있는 인자 없음 → None/HTTP 0 | `NoOp=true`, `Lookup=nil`, `Router=nil`; 서비스/대상 검증 생략 |
| 기존 raw 값과 같은 수정 | `NoOp=true`, `Lookup` 존재, `Router=nil`; PUT 없음 |
| `admin_state_up=False`, `enable_snat=False` | 명시 false를 생략하지 않음 |
| `routes=[]` | fresh `extraroute` 조회 후 routes 비우기 |
| extraroute 없음 | routes만 버리고 다른 인자로 계속; `RoutesDropped=true` |
| 없는 router | `errors.Is(err, resource.ErrNotFound)`; PUT 없음 |

허용 인자는 `name`, `admin_state_up`, `ext_gateway_net_id`, `enable_snat`, `ext_fixed_ips`, `routes`의 6개입니다. `external_gateway_info`, `is_admin_state_up`, `if_revision`, 임의 확장 키와 `filters`는 cloud 인자가 아닙니다. `WithRouterUpdateName`, `WithRouterUpdateAdminStateUp`, `WithRouterUpdateExternalGatewayNetworkID`, `WithRouterUpdateEnableSNAT`, `WithRouterUpdateRoutes`와 일반 `WithRouterUpdateAttribute`를 제공합니다. `WithRouterUpdateJSON`은 전체 인자를 대체하고 이후 옵션이 같은 키를 덮으며 SDK가 bytes/map을 복제합니다.

`name`, `ext_gateway_net_id`, `ext_fixed_ips`는 Python truthy일 때 선택합니다. `admin_state_up`, `enable_snat`, `routes`는 null이 아니면 false·0·빈 문자열·빈 list도 선택합니다. 생성의 admin_state_up=true 기본값은 수정에 적용하지 않습니다. gateway는 기존 생성 helper와 같은 규칙으로 `network_id`, `enable_snat`, `external_fixed_ips`를 새 dictionary로 조립하며 기존 gateway dictionary와 합치지 않습니다.

routes가 null이면 extension HTTP를 하지 않습니다. routes가 null이 아니면 값이 빈 list나 false여도 lookup 전에 전체 extension 목록을 새로 읽습니다. `extraroute`가 없을 때 routes만 버리고, 다른 인자가 없으면 extension 조회 뒤 no-op을 반환합니다. `RoutesDropped`로 이를 관찰할 수 있으며 Source의 warning logger 동작은 별도 Go 로깅 API로 복제하지 않습니다. 요청 alias를 먼저 찾았더라도 후반 페이지 HTTP 오류나 잘못된 alias를 숨기지 않습니다. 그런 실패는 routes 미지원으로 처리하지 않습니다. `Extensions`에는 실제로 관찰한 record가 남습니다.

옵션 적용 전 API/client 포인터를 기록하고, 효과 있는 인자가 있으면 포인터 변경을 거부합니다. client source 고정과 라우트 guard는 효과 있는 인자 판정 후 시작합니다. 무효 인자 no-op은 서비스 검증 없이 반환하되 잘못된 옵션과 Go context 취소를 숨기지 않습니다. 고정 전 직접 변경된 client 내부 설정까지 before-options capture 보장을 주장하지 않습니다.

기존 `routers.FindRecord`와 native `UpdateWithAttributes`를 재사용합니다. 조회의 ordinary wire-key raw 값에서 Python equality로 변경한 필드만 PUT합니다. 조회 `Lookup.Wire/Envelope/Header/StatusCode`는 조회 evidence이며 PUT receipt가 아닙니다. `Router`는 별도 native typed PUT 결과이고, 실패한 PUT에서는 nil입니다. 기존 provider retry/reauthentication 설정이 유지되므로 모든 설정에서 단일 전송을 보장하지 않습니다.

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
    result = conn.update_router(args.identity, **attrs)
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
    identity := flag.String("identity", "", "router ID or exact name")
    attrs := flag.String("attrs", "{}", "cloud update kwargs as JSON")
    flag.Parse()
    if *identity == "" { log.Fatal("identity is required") }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.Network(ctx)
    if err != nil { log.Fatal(err) }
    result, err := service.UpdateRouterRecord(ctx, *identity,
        network.WithRouterUpdateJSON(json.RawMessage(*attrs)))
    if err != nil { log.Fatal(err) }
    var value any = result.Router
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
관리 상태 수정: `go run . -identity example -attrs '{"admin_state_up":false}'`.
추가 경로 비우기: `go run . -identity example -attrs '{"routes":[]}'`.
효과 있는 인자는 실제 서버 상태를 변경합니다.

## 남은 차이와 검증 범위

Native typed PUT 결과에는 unknown extension 필드가 없으며, 부분 응답 병합·빈/잘못된 성공 JSON에서 기존 Resource 유지·동일 mutable instance 반환을 제공하지 않습니다. 원본 PUT Envelope/Header/StatusCode도 이 bridge가 노출하지 않습니다. Member GET/LIST canonical-only 또는 canonical/wire 충돌의 mapping 우선순위, 원래 Python kwargs의 임의 객체·혼합 numeric 타입, 전체 Requests encoding/nonfinite JSON, SDK-R1/C1/S1의 Resource/adapter/session/cache 수명 주기는 별도 계약입니다. 연산 판정은 unresolved를 유지하며 실클라우드·릴리즈 검증을 완료한 것으로 해석하지 않습니다.

집중 테스트는 `TestCloudRouterUpdateEffectiveValuesAndDirtySelection`, `TestCloudRouterUpdateIneffectiveBeforeServiceLookup`, `TestCloudRouterUpdateRoutesGateAndDrop`, `TestCloudRouterUpdateExtensionFailureCannotBecomeDroppedRoutes`, `TestCloudRouterUpdateOptionCancellationCannotBecomeNoOp`, `TestCloudRouterUpdateNameFallbackAndMissing`, `TestCloudRouterUpdateOwnedOptionsAndPreflight`, `TestCloudRouterUpdateConflictKeepsLookupAndSingleAttempt`입니다. 공개 Gophercloud testhelper 기반 기존 `internal/testcloud` fixture와 SDK extension parser를 재사용합니다.

Root 검증: 최초 focused18그룹51사례(router8/23+기존10/28), 독립 Go main/Python AST 각1개, 전체205개 test package와 license/vet/race/parity/progress/gofmt PASS. 실클라우드 검증은 별도입니다.
