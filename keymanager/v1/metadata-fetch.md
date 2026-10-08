# Container / Order metadata Fetch

Container와 Order의 metadata 조회를 Python public getter와 비교합니다. 고정 기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`와 Gophercloud v2.15.0입니다.

## Python과 Go API

| Python public getter | Go call | 반환 |
|---|---|---|
| `conn.key_manager.get_container(container_id)` | `service.Containers.Fetch(ctx, resource.ID(containerID), options...)` | `*containers.FetchedContainer, error` |
| `conn.key_manager.get_order(order_id)` | `service.Orders.Fetch(ctx, resource.ID(orderID), options...)` | `*orders.FetchedOrder, error` |

Python은 ID 문자열 또는 해당 mutable Resource를 받아 GET 하나를 실행합니다. Go Fetch는 실행할 explicit ID ref와 context를 받고, 반환된 Go 모델을 다시 읽을 때는 `value.Ref()`로 원래 request ID를 사용합니다. 이름 조회, 목록, 상태 대기, 생성·변경·삭제를 추가하지 않습니다. `resource.Name`은 지원하지 않는 입력이며 leaf Fetch에서 HTTP 전에 오류입니다. 입력 whole HREF나 임의 Python Resource/subclass overload는 제공하지 않습니다.

기존 `Containers.Get(ctx, string)`/`Orders.Get(ctx, string)`와 alias 결과는 유지합니다. 새 owned Fetch는 원문 timestamp·nullable 속성·임의 container 배열 요소·arbitrary order meta를 읽는 별도 결과입니다. 기존 alias와 상호 대체되는 return type이라고 설명하지 않습니다.

## 결과의 세 가지 identity와 소유권

- `RequestID` / `Ref()`: 현재 호출이 실행한 고정 ID. response `id`, foreign `container_ref`/`order_ref` 또는 `secret_ref` 때문에 다른 URL을 요청하지 않습니다.
- `Resource.Body["id"]`: source-shaped attribute. 응답에서 id를 생략하면 요청 ID seed를 유지하고, present id:null 또는 다른 id는 그대로 반영합니다.
- `ContainerID` / `OrderID` / `SecretID`: nullable HREF formatter 결과. URI 마지막 원문 component이고 %xx spelling·빈 trailing component를 보존합니다. 이는 실행할 request ID나 native UUID validation이 아닙니다.

`Resource`는 선언된 속성·default-null 및 source list/dict/formatter 결과를 제공합니다. `Wire`는 실제 응답 object의 canonical/null/unknown/self/extension 값이며 요청 ID를 주입하지 않습니다. `Envelope`는 실제 body 바이트, `Header`/`StatusCode`는 실제 HTTP 응답입니다. Resource/Wire/Envelope/receipt는 독립 snapshot입니다.

Container의 `secret_refs`/`consumers`는 arbitrary JSON 요소를 보존하며 non-null scalar/object는 한 요소 배열로 가공합니다. missing/null은 null, []는 []입니다. Order의 `meta`는 arbitrary 객체를 보존하며 non-null non-object는 {}로 가공하고 missing/null은 null입니다. created/updated는 `created_at`/`updated_at` view 속성이고 timestamp를 시간형으로 강제 변환하지 않습니다.

`Resource.Body["location"]`은 null입니다. Python의 일반 Connection getter는 Resource 생성 시 `conn.current_location`을 주입할 수 있습니다. 이 Go getter는 context와 인증된 ServiceClient를 사용하며 해당 Connection 위치 정보를 응답에서 합성하지 않는 차이가 있습니다.

Client attribute alias는 canonical wire key가 없을 때만 보충합니다. canonical present-null도 alias보다 우선하는 Go 정책입니다. Python mixed alias 입력은 object insertion order에 따르므로 이 충돌 우선순위 차이는 문서에 남깁니다. HREF parsing의 일부 authority에서 기존 Go parser가 Python보다 엄격한 차이도 유지합니다.

## 옵션과 오류

각 패키지의 `WithFetchHeader(key,value)`는 concrete 추가 헤더 옵션입니다. Query/body/argument option은 지원하지 않습니다. default SDK timeout은 새로 만들지 않고 context 제한을 사용합니다. configured Provider token, native authentication/reauthentication/transport를 이용하며 option callbacks를 한 번 준비합니다. source guard는 options 전 캡처한 서비스/path/provider/version을 유지합니다.

GET은 실제200..399 status의 object 응답을 읽고,404/403은 native HTTP 오류로 반환합니다. accepted empty/malformed/non-object body를 엄격한 ResponseError로 반환하는 정책은 pinned generic Resource가 JSON ValueError를 무시하거나 seed만 반환할 수 있는 경우와 다릅니다.

accepted read/Close/decode/formatter/source/context 오류에서도 실제 receipt 및 가능한 Wire를 오류 옆 결과로 유지합니다. 완성되지 않은 Resource view를 성공으로 만들지 않습니다. SDK 조합 로직의 오류 때문에 이름 재선택/fallback/DELETE/재전송하지 않으며 configured provider의 native retry·재인증 정책은 유지합니다. 

## 단독 Go 예제

SDK 모듈 안의 별도 디렉토리에 `main.go`로 저장합니다. `-kind container -id CONTAINER_ID` 또는 `-kind order -id ORDER_ID`로 조회 하나를 선택합니다. 두 경우 모두 기존 Connection→KeyManagerV1 facade→leaf Fetch를 이용합니다. 새 교차 서비스 API나 mock harness를 요구하지 않습니다.

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml entry")
    kind := flag.String("kind", "container", "container or order")
    id := flag.String("id", "", "explicit resource ID")
    timeout := flag.Duration("timeout", time.Minute, "authentication and metadata query timeout")
    flag.Parse()
    if err := run(*cloud, *kind, *id, *timeout); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(cloud, kind, id string, timeout time.Duration) error {
    if timeout <= 0 || id == "" || (kind != "container" && kind != "order") {
        return fmt.Errorf("positive timeout, explicit id and kind container/order are required")
    }
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil {
        return err
    }
    service, err := conn.KeyManagerV1(ctx)
    if err != nil {
        return err
    }
    var output map[string]any
    switch kind {
    case "container":
        value, fetchErr := service.Containers.Fetch(ctx, resource.ID(id))
        err = fetchErr
        if value != nil {
            output = map[string]any{
                "request_id": value.RequestID,
                "container_id": value.ContainerID,
                "resource": value.Resource,
                "wire": value.Wire,
                "envelope": string(value.Envelope),
                "header": value.Header,
                "status_code": value.StatusCode,
            }
        }
    case "order":
        value, fetchErr := service.Orders.Fetch(ctx, resource.ID(id))
        err = fetchErr
        if value != nil {
            output = map[string]any{
                "request_id": value.RequestID,
                "order_id": value.OrderID,
                "secret_id": value.SecretID,
                "resource": value.Resource,
                "wire": value.Wire,
                "envelope": string(value.Envelope),
                "header": value.Header,
                "status_code": value.StatusCode,
            }
        }
    }
    if output != nil {
        encoder := json.NewEncoder(os.Stdout)
        encoder.SetIndent("", "  ")
        err = errors.Join(err, encoder.Encode(output))
    }
    return err
}
```

Envelope를 문자열로 출력하므로 malformed accepted body도 부분 결과의 receipt로 표현할 수 있습니다. fetch 오류가 있으면 원래 오류를 반환하고 process는 실패하며, 받은 결과가 있다면 이를 먼저 출력합니다. 실제 cloud 인증·조회와 Python 예제 실행은 별도 환경에서 수행해야 합니다.


고정 source 근거는 [Container getter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L114), [Order getter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L220), [Container fields](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/container.py#L17), [Order fields](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/order.py#L17)입니다. [SDK 지원 판정대장](../../docs/sdk-support-ledger.md)에서 실제 검증과 named 판정을 추적합니다.
