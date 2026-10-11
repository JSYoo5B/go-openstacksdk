# Neutron Router actions와 openstacksdk 비교

`conn.NetworkV2(ctx).Routers`의 아래 `Record` API는 고정 openstacksdk
`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 사용자용 Router action 5개에 대응합니다.
요청 대상은 literal router ID이며, 이름 검색이나 사전 GET은 실행하지 않습니다.

| openstacksdk proxy | Go | PUT 경로 |
|---|---|---|
| `add_extra_routes_to_router(router, body)` | `AddExtraRoutesRecord(ctx, id, body, options...)` | `/routers/{id}/add_extraroutes` |
| `remove_extra_routes_from_router(router, body)` | `RemoveExtraRoutesRecord(ctx, id, body, options...)` | `/routers/{id}/remove_extraroutes` |
| `add_external_gateways(router, body)` | `AddExternalGatewaysRecord(ctx, id, body, options...)` | `/routers/{id}/add_external_gateways` |
| `update_external_gateways(router, body)` | `UpdateExternalGatewaysRecord(ctx, id, body, options...)` | `/routers/{id}/update_external_gateways` |
| `remove_external_gateways(router, body)` | `RemoveExternalGatewaysRecord(ctx, id, body, options...)` | `/routers/{id}/remove_external_gateways` |

Source는 `body`를 그대로 `session.put(json=body)`에 넘깁니다. Go도 envelope를 만들거나
Body alias를 바꾸지 않습니다. 따라서 API가 요구하는 `router` envelope를 호출자가 넣습니다.
`is_admin_state_up`, `if_match`, `if_revision` 같은 키를 본문에 넣으면 그 위치에 그대로
전송합니다. `UpdateWithAttributes`의 Body/조건부 Header 처리 규칙을 여기에는 적용하지 않습니다.
알 수 없는 확장 키, null, false, 0, 빈 목록/객체와 큰 정수도 보존합니다. `nil`은 본문 없이
전송하며, `{}`는 빈 JSON 객체입니다. `json.RawMessage("null")`은 명시적 JSON null을 전송하는
Go 입력입니다. Source의 Python `None`은 `json=None`에 해당하므로 본문이 없습니다.

[Neutron 27.0.0 stock 정책](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/router.py)에
따르면 이 5개 동작은 admin 또는 해당 router 프로젝트의 member가 호출할 수 있습니다.
추가·수정 게이트웨이에 `enable_snat`이나 `external_fixed_ips`를 지정하는 세부 분기는
admin 권한을 요구합니다. 아래 사용자 예제는 `network_id`만 지정합니다. 설치된 extension,
네트워크 접근 권한과 배포의 정책 변경에 대한 판단은 Neutron이 수행합니다.

```python
import openstack

conn = openstack.connect(cloud="dev")
router_id = "ROUTER_ID"
route_body = {"router": {"routes": [
    {"destination": "192.0.2.0/24", "nexthop": "10.0.0.9"}
]}}
gateway_body = {"router": {"external_gateways": [
    {"network_id": "EXTERNAL_NETWORK_ID"}
]}}

# 각 행은 해당 동작을 실행하는 독립적인 선택지입니다.
added = conn.network.add_extra_routes_to_router(router_id, route_body)
removed = conn.network.remove_extra_routes_from_router(router_id, route_body)
gateway_added = conn.network.add_external_gateways(router_id, gateway_body)
gateway_updated = conn.network.update_external_gateways(router_id, gateway_body)
gateway_removed = conn.network.remove_external_gateways(router_id, gateway_body)
print(added.to_dict())
```

## 독립 Go main

[설치 안내](../../docs/install.md)에 따라 모듈을 준비하고 아래를 `main.go`로 저장합니다.
예제는 `-action`으로 선택한 동작 하나만 실행합니다.

```sh
go run . -cloud dev -router ROUTER_ID -action add-routes \
  -body '{"router":{"routes":[{"destination":"192.0.2.0/24","nexthop":"10.0.0.9"}]}}'
go run . -cloud dev -router ROUTER_ID -action add-gateways \
  -body '{"router":{"external_gateways":[{"network_id":"EXTERNAL_NETWORK_ID"}]}}'
```

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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    id := flag.String("router", "", "literal router ID")
    action := flag.String("action", "", "add-routes, remove-routes, add-gateways, update-gateways, remove-gateways")
    body := flag.String("body", "", "complete JSON action body, including router envelope")
    flag.Parse()
    if *id == "" || !json.Valid([]byte(*body)) { log.Fatal("-router and a valid JSON -body are required") }
    switch *action {
    case "add-routes", "remove-routes", "add-gateways", "update-gateways", "remove-gateways":
    default: log.Fatal("select a supported -action")
    }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    api := service.Routers
    raw := json.RawMessage(*body)
    var result *routers.RouterActionRecord
    switch *action {
    case "add-routes": result, err = api.AddExtraRoutesRecord(ctx, *id, raw)
    case "remove-routes": result, err = api.RemoveExtraRoutesRecord(ctx, *id, raw)
    case "add-gateways": result, err = api.AddExternalGatewaysRecord(ctx, *id, raw)
    case "update-gateways": result, err = api.UpdateExternalGatewaysRecord(ctx, *id, raw)
    case "remove-gateways": result, err = api.RemoveExternalGatewaysRecord(ctx, *id, raw)
    }
    if result != nil {
        fmt.Printf("HTTP status=%d request_id=%s\n", result.StatusCode, result.Header.Get("X-Request-Id"))
        fmt.Printf("actual envelope=%s\n", result.Envelope)
        if result.Resource != nil {
            view, encodeErr := json.MarshalIndent(result.Resource, "", "  ")
            if encodeErr != nil { log.Fatal(encodeErr) }
            fmt.Printf("Source descriptor view=%s\n", view)
        }
    }
    if err != nil { log.Fatal(err) }
}
```

`RouterActionRecord.Resource`는 Source의 canonical descriptor view입니다. 응답의 wire alias를
정규화하고, 선언된 필드의 누락 값은 null, 누락된 `tags`는 `[]`로 채웁니다. 명시적 null은
유지하며, `project_id`가 없으면 `tenant_id`를 사용합니다. Router의 revision descriptor는
wire `revision`에 대응합니다. bool/int/list/dict 변환은 Source descriptor의 JSON 값 규칙을
적용하며, 응답에 함께 있는 canonical/wire alias는 JSON 객체의 순서대로 적용합니다.
Header `if_match`는 응답 Header를 한 문자열로 취급한 목록이며 쉼표를 분리하지 않습니다.
`Wire`에는 실제 응답 router 필드만, `Envelope`에는 전체 JSON, `Header`와 `StatusCode`에는
실제 HTTP 관측만 담깁니다. 각각 소유한 복사본이므로 view를 수정해도 Wire는 바뀌지 않습니다.

Python은 Router 객체를 받으면 기존 객체를 수정합니다. Go는 `WithRouterActionSeed(cached)` 또는
`RouterActionOpts{Seed: cached}`로 기존 `*resource.RawResource`를 제공하고 새 결과를 반환합니다.
seed의 선언된 Body alias와 `if_match` view를 보존하되, 경로와 초기 view의 ID는 명시적 인수로
고정합니다. 응답에서 반환된 ID는 view에 반영합니다. map seed에 서로 충돌하는 alias가 있으면
오류를 반환합니다. seed는 요청에 전송하지 않으며, 본문에 넣은 요청 값도 응답 seed로 쓰지 않습니다.
Source의 connection 기반 `location` 계산은 전체 공통 Resource 작업에서 추적하며 여기서는 null입니다.

```go
cached := &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{
    "name": json.RawMessage(`"cached-edge"`),
    "routes": json.RawMessage(`[]`),
}}}
row, err := api.AddExtraRoutesRecord(ctx, "ROUTER_ID", json.RawMessage(`{"router":{"routes":[]}}`),
    routers.WithRouterActionSeed(cached),
    routers.WithRouterActionHeaders(map[string]string{"X-Client-Trace": "router-action"}))
```

위 seed 조각은 독립 main의 후속 사용 예이며 `resource` import가 필요합니다.
header/seed 옵션과 입력 본문은 복사되며, 인증 Header와 서비스 경로를 옵션으로 바꿀 수 없습니다.
메서드는 설정된 client의 microversion을 사용하고 별도 extension 발견 요청이나 version gate를
추가하지 않습니다. Python 임의 객체·iterable의 런타임 동작 대신 Go의 UTF-8 JSON 입력을 받습니다.
정수 descriptor의 Unicode digit 판정은 공유한 Unicode 16.0 기준이며, Python 런타임과 무한대·비정상
숫자 입력 전체를 호환한다는 의미는 아닙니다.

성공 상태는 Source `raise_from_response` 정책에 맞춘 200..399이며, 이 action들에는
`ignore_missing` 인수가 없어 clean 404도 오류입니다. 비어 있거나 JSON으로 파싱되지 않는
성공 본문은 ID 또는 cached view를 남깁니다. 유효 JSON이지만 객체가 아니거나 `router`가 null이면
실제 응답 근거를 담은 `resource.ResponseError`입니다. Read/Close 실패·취소·source 변경도
관측된 Body/Header/status와 원인을 보존합니다. retry/재인증에는 원래 JSON과 route를 고정하며,
retry callback이 허용 status를 바꿔도 원래 정책을 다시 검사합니다. 202/204 성공과 seeded view는
서버가 모든 경로·게이트웨이 변경을 적용했다는 완료 판정이 아닙니다. 필요하면 별도 GET으로 확인합니다.

테스트는 기존 `internal/testcloud`를 통해 Gophercloud public `testhelper`를 재사용하고,
서버 대신 transport fixture가 물리적인 Read/Close 오류를 주입합니다. 실제 OpenStack 인증·호출은
이 문서의 검증에 포함되지 않습니다. 이 다섯 callable 대응과 전체 Resource/cache/session 및
서비스 전체 SDK parity의 완료 여부는 구분해서 [구현 계획](../../docs/implementation-plan.md)에 기록합니다.
