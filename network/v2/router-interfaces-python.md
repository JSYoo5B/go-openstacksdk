# Router interface와 단일 gateway

`conn.NetworkV2(ctx).Routers`에 builder 구현 없이 사용할 수 있는 옵션 API를 제공합니다.
기준 Python Source는 `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`입니다.

| Python proxy | Go | 요청과 기본값 |
|---|---|---|
| `add_interface_to_router(router, subnet=None, port=None, advertise_host=False)` | `AddInterfaceRecord(ctx, id, options...)` | `PUT /routers/{id}/add_router_interface`, truthy port 우선, 그 외 subnet, truthy advertise_host만 전송 |
| `remove_interface_from_router(router, subnet=None, port=None)` | `RemoveInterfaceRecord(ctx, id, options...)` | `PUT /routers/{id}/remove_router_interface`, 같은 port/subnet 선택, advertise_host 인수 없음 |
| `add_gateway_to_router(router, **body)` | `AddGatewayRecord(ctx, id, options...)` | 기본은 stock `PUT /routers/{id}`의 `router.external_gateway_info` 설정 |
| `remove_gateway_from_router(router, **body)` | `RemoveGatewayRecord(ctx, id, options...)` | 기본은 stock `PUT /routers/{id}`에 `router.external_gateway_info: null` 전송 |

인터페이스에는 기본값을 담은 필드를 모두 보내지 않습니다. 인수가 없으면 Source처럼
`{"subnet_id":null}`을 전송하며 Neutron이 잘못된 ID를 판정합니다. port의 `""`, null,
false, 0과 빈 목록/객체는 subnet으로 넘어가고, truthy port가 있으면 subnet은 사용하지 않습니다.
`WithRouterInterfacePort`와 `WithRouterInterfaceSubnet`은 문자열 또는 JSON 값을 받습니다.
`*resource.RawResource`를 넘기면 Source의 SDK Resource처럼 해당 `id`를 선택하며, Resource의
ID가 null/빈 문자열이어도 port 우선순위는 유지합니다. 일반 Go map은 Resource로 해석하지 않고
Source의 일반 dictionary 입력처럼 JSON 값 그대로 처리합니다. Python의 Resource 인스턴스와
임의 객체 모델을 모두 Go 타입으로 재현하는 것은 별도 공통 Resource 작업입니다.

Python의 오래된 `port_id`/`subnet_id` 인수는 현재 `port`/`subnet`을 덮어씁니다.
`WithRouterInterfacePortID`/`WithRouterInterfaceSubnetID`도 옵션 순서와 관계없이 같은 규칙을
따릅니다. Go는 Python 런타임의 deprecation warning을 발생시키지 않습니다.
`WithRouterInterfaceAdvertiseHost(false)`는 `advertise_host`를 보내지 않으며, 이 옵션을
remove에 전달하면 호출 전에 오류를 반환합니다. 추가 인수의 실제 기능은 배포가 지원해야 합니다.
[Neutron 27.0.0 기본 L3 처리](https://github.com/openstack/neutron/blob/27.0.0/neutron/db/l3_db.py)는
인터페이스 동작에서 이 flag를 읽지 않습니다. SDK가 flag를 전송했다는 것과 EVPN host route
광고가 실제로 적용됐다는 것은 구분해야 합니다.

```python
import openstack

conn = openstack.connect(cloud="dev")
attached = conn.network.add_interface_to_router("ROUTER_ID", subnet="SUBNET_ID")
detached = conn.network.remove_interface_from_router("ROUTER_ID", port="PORT_ID")

# 현재 Neutron에 사용하는 단일 gateway 설정/제거
added = conn.network.update_router("ROUTER_ID", external_gateway_info={
    "network_id": "EXTERNAL_NETWORK_ID"
})
removed = conn.network.update_router("ROUTER_ID", external_gateway_info=None)

# 고정 Source의 legacy helper는 다음 별도 경로를 사용합니다.
# conn.network.add_gateway_to_router("ROUTER_ID", network_id="EXTERNAL_NETWORK_ID")
# conn.network.remove_gateway_from_router("ROUTER_ID")
```

## Gateway의 명시적 차이

Pinned Source gateway helper는 `/add_gateway_router`와 `/remove_gateway_router`를 호출합니다.
[Neutron 27.0.0 L3 extension](https://github.com/openstack/neutron/blob/27.0.0/neutron/extensions/l3.py)이
사용하는 [neutron-lib 3.21.1 ACTION_MAP](https://github.com/openstack/neutron-lib/blob/3.21.1/neutron_lib/api/definitions/l3.py)에는
이 두 경로가 등록되어 있지 않습니다. Go의 기본 동작은 현재 지원되는 `external_gateway_info`
갱신 경로로 단일 gateway의 의도를 수행합니다. 멀티 게이트웨이 동작은 별도
[라우터 action API](router-actions-python.md)를 사용합니다.

`AddGatewayRecord`는 nonblank 문자열 `network_id`를 요구합니다. 빈 gateway 객체가
기존 gateway를 제거하는 것으로 해석되는 사고를 막기 위한 기본 규칙입니다. 추가 속성은
alias를 바꾸거나 기본값을 만들지 않고 `external_gateway_info` 안에 보존합니다.
`RemoveGatewayRecord`는 null을 명시하며 요청 속성을 받지 않습니다. 전달한 속성을 버리지 않고
호출 전에 오류를 반환합니다. 해당 body를 실제로 처리하는 legacy backend에 연결할 때만
`WithSourceLegacyGatewayRoutes(true)`로 고정 Source 경로와 envelope 없는 kwargs를 선택합니다.
false를 지정하면 stock 경로로 돌아갑니다. `router`, `self`, `session`은 Source 메서드 인수와
충돌하는 키이므로 body 속성으로 사용할 수 없습니다.

Source gateway 메서드는 `Proxy.request(raise_exc=False)`로 받은 응답에 `raise_from_response`를
호출하지 않고 `response.json()`을 반환합니다. 따라서 4xx/5xx JSON도 Python의 반환 값이 될 수
있습니다. Go는 기본/legacy 경로 모두 4xx/5xx를 오류로 보존합니다. JSON 오류 본문을 성공으로
판정하는 동작은 제공하지 않습니다. 이 차이와 route bridge는 중앙 API 대응 판정에 따로 기록하며,
Source 전체 Resource/session 동작의 완성으로 집계하지 않습니다.

[Neutron 27.0.0 stock 정책](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/router.py)에서
interface 추가·제거와 `update_router:external_gateway_info`는 admin 또는 해당 프로젝트의
member에게 허용됩니다. gateway의 `network_id` 지정도 같은 사용자 범위이며, `enable_snat`과
`external_fixed_ips`를 명시하는 세부 분기는 admin 전용입니다. 라이브러리는 사용자 대신 권한을
판정하지 않고 Neutron의 실제 오류를 보존합니다. Legacy 경로가 지원되는지는 별도 backend의
구현·정책에 달려 있으며 stock 정책에서 legacy 동작 지원을 추론하지 않습니다.

## 독립 Go main

[설치 안내](../../docs/install.md)에 따라 모듈을 준비하고 아래를 `main.go`로 저장합니다.
선택한 동작 하나만 실행합니다.

```sh
go run . -cloud dev -router ROUTER_ID -action add-interface -subnet SUBNET_ID
go run . -cloud dev -router ROUTER_ID -action remove-interface -port PORT_ID
go run . -cloud dev -router ROUTER_ID -action add-gateway -network EXTERNAL_NETWORK_ID
go run . -cloud dev -router ROUTER_ID -action remove-gateway
```

```go
package main

import (
    "context"
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
    action := flag.String("action", "", "add-interface, remove-interface, add-gateway, remove-gateway")
    port := flag.String("port", "", "interface port ID; wins subnet")
    subnet := flag.String("subnet", "", "interface subnet ID")
    network := flag.String("network", "", "single external gateway network ID")
    advertise := flag.Bool("advertise-host", false, "add-interface only; requires supporting backend")
    legacy := flag.Bool("legacy-source-route", false, "gateway only; requires backend legacy action support")
    flag.Parse()
    if *id == "" { log.Fatal("-router is required") }
    switch *action {
    case "add-interface", "remove-interface":
        if *port == "" && *subnet == "" { log.Fatal("provide -port or -subnet") }
        if *legacy { log.Fatal("-legacy-source-route is a gateway option") }
        if *action == "remove-interface" && *advertise { log.Fatal("-advertise-host is add-interface only") }
    case "add-gateway":
        if *network == "" { log.Fatal("-network is required") }
    case "remove-gateway":
        if *network != "" && !*legacy { log.Fatal("stock remove-gateway accepts no network attribute") }
    default: log.Fatal("select a supported -action")
    }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    var result *routers.RouterJSONResult
    switch *action {
    case "add-interface", "remove-interface":
        options := []routers.RouterInterfaceOption{}
        if *port != "" { options = append(options, routers.WithRouterInterfacePort(*port)) }
        if *subnet != "" { options = append(options, routers.WithRouterInterfaceSubnet(*subnet)) }
        if *action == "add-interface" {
            options = append(options, routers.WithRouterInterfaceAdvertiseHost(*advertise))
            result, err = service.Routers.AddInterfaceRecord(ctx, *id, options...)
        } else {
            result, err = service.Routers.RemoveInterfaceRecord(ctx, *id, options...)
        }
    case "add-gateway", "remove-gateway":
        options := []routers.RouterGatewayOption{routers.WithSourceLegacyGatewayRoutes(*legacy)}
        if *network != "" { options = append(options, routers.WithRouterGatewayField("network_id", *network)) }
        if *action == "add-gateway" {
            result, err = service.Routers.AddGatewayRecord(ctx, *id, options...)
        } else {
            result, err = service.Routers.RemoveGatewayRecord(ctx, *id, options...)
        }
    }
    if result != nil {
        fmt.Printf("HTTP status=%d request_id=%s\n", result.StatusCode, result.Header.Get("X-Request-Id"))
        fmt.Printf("actual envelope=%s\n", result.Envelope)
        if result.Value != nil { fmt.Printf("complete JSON value=%s\n", result.Value) }
    }
    if err != nil { log.Fatal(err) }
}
```

`RouterJSONResult.Value`는 Source `response.json()`에 대응하는 전체 UTF-8 JSON입니다.
객체뿐 아니라 목록, 문자열, 숫자, bool, 명시적 null도 유지합니다. `Wire`는 전체 응답이 객체일
때만 존재하며 `router` envelope를 풀거나 Source Router descriptor로 변환하지 않습니다.
`Envelope`, `Header`, `StatusCode`는 독립적으로 복사한 실제 HTTP 관측입니다. Router의
canonical default나 cached seed를 합치지 않습니다. 성공 204도 응답이 비어 있으면 JSON decode
오류이며, Read/Close 실패·취소·source 변경은 실제 응답을 담은 `resource.ResponseError` 또는
native HTTP 오류로 남습니다. 오류 result에 관측이 남아도 `Value`는 성공 값으로 만들지 않습니다.

옵션의 reference/속성/header JSON은 복사하며, callback 뒤에도 Source/client/route/context를
검사합니다. 재시도와 재인증에는 원래 본문과 URI를 유지하며 caller callback이 소유권을 바꾸면
원인을 보존하고 중단합니다. 선택된 `OpenStack-API-Version` 값 또는 version 부재,
다른 서비스의 version header 부재와 JSON `Content-Type`을 재시도·물리 요청 전체에 고정합니다.
callback의 `MoreHeaders` 변경이나 `OmitHeaders`로 이를 바꾸면 두 번째 요청 전에 중단하고
최초 native HTTP 오류의 본문·header·status와 callback 원인을 함께 보존합니다.
인증·전송 framing header 주입도 차단하며 일반 `X-Retry` 같은 header 변경과 라이브러리의
라이브 인증 token 갱신은 허용합니다. 서비스 선택과 인증 Header는 라이브러리가 소유합니다.
테스트는 기존 Router fixture와 Gophercloud public `testhelper` 기반 `internal/testcloud`를
재사용합니다. 실제 OpenStack 인증·배포별 extension 동작이나 클라우드 상태는 실행 검증하지 않았습니다.
