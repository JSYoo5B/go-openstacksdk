# Container / Order / Secret 삭제

기존 `Remove`와 기본 `Resources.Delete`로 사용자 리소스를 삭제합니다. 기본 missing 정책, strict 옵션과 native Delete의 차이를 Python public API와 비교합니다.

Python pin: `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, native Gophercloud v2.15.0.

## Python과 Go

| Python | 기존 Go entry | 기본 missing 정책 / 성공 결과 |
|---|---|---|
| `conn.key_manager.delete_container(container_id)` | `service.Containers.Remove(ctx, resource.ID(containerID))` | 실제404 무시 / nil error |
| `conn.key_manager.delete_order(order_id)` | `service.Orders.Remove(ctx, resource.ID(orderID))` | 실제404 무시 / nil error |
| `conn.key_manager.delete_secret(secret_id)` | `service.Secrets.Remove(ctx, resource.ID(secretID))` | 실제404 무시 / nil error |
| 위 세 호출의 `ignore_missing=False` | 마지막 인자로 `resource.WithMissingError()` | 실제404 오류 / nil error |

세 Python public 메서드는 ID 문자열 또는 해당 Resource를 받아 DELETE 하나를 실행하며 성공 반환은 `None`이다. explicit ID를 넘기는 Go `Remove`도 목록·GET·상태 대기 없이 삭제한다. 기본 `Resources.Delete`를 직접 호출해도 같은 collection missing 정책이다. 값이 이미 없으면 기본 정책은 성공하므로 nil error를 “이번 HTTP가 실제 객체를 지웠다”는 증거로 해석하지 않는다.

조회한 owned Container/Order 값에서 이어갈 경우 `fetched.Ref()`는 원래 executable RequestID를 사용한다. 기존 owned Secret 조회는 고정 요청 identity인 `fetched.SecretID`를 `resource.ID(fetched.SecretID)`로 전달할 수 있다. response의 `id`, derived ID 또는 foreign HREF로 HTTP 경로를 자동으로 바꾸지 않는다. 이번 예제는 사전 Fetch를 하지 않고 caller가 지정한 ID로 직접 삭제한다.

## 옵션과 기존 확장

| 설정 | 동작 |
|---|---|
| Lookup option 생략 | collection 기본값 `ignore_missing=true` |
| `resource.WithMissingError()` | 실제404를 ErrNotFound와 native HTTP cause로 반환 |
| `resource.WithIgnoreMissing()` | 기본 missing 무시를 명시하며 마지막 옵션이 우선 |
| 호출 context | caller의 취소·deadline을 사용; 새 SDK timeout/DELETE 후 waiter 없음 |

Python named delete는 이름 탐색을 하지 않는다. 기존 `Containers.Remove(ctx, resource.Name(name))`와 `Secrets.Remove(ctx, resource.Name(name))`는 Go lookup 확장으로 유지하며, 이 경우 이름 해석 목록 요청이 생길 수 있다. `Orders.Remove`의 Name 입력은 기존대로 unsupported다. 이번 완료 판정과 단독 예제는 source와 직접 대응하는 explicit ID 삭제를 대상으로 한다.

기본 collection을 custom Resources로 바꾸는 기존 확장도 그대로다. 이 가이드의 owned REST·source guard 설명은 SDK가 구성한 기본 collection binding에 적용된다. native `Containers.Delete(ctx,string)`, `Orders.Delete(ctx,string)`, `Secrets.Delete(ctx,string)`는 missing을 자동으로 억제하지 않고 Gophercloud의202/204 strict 정책을 유지하며 별도 native 연산이다.

## 성공과 오류

개선된 기본 collection DELETE binding은 실제 최종 HTTP200..399를 받아 source의 status<400 계약을 매핑한다. 성공 body는 JSON 객체일 필요가 없으며 빈 body·malformed JSON·임의 bytes도 해석하지 않는다. Container/Order/Secret model, HREF formatter, payload/content type, location 또는 연결된 리소스를 성공 처리에 요구하지 않는다. 응답 수신은 삭제 완료 관측이나 서버에서의 eventual disappearance를 검증하는 wait가 아니다.

기본 missing 정책은 실제 clean DELETE404만 무시한다. strict404와403/409 등 native HTTP 오류는 원래 status/body/header/cause를 유지한다. accepted 응답의 read/Close/source/context 오류는 actual body/header/status의 `resource.ResponseError`와 원인을 반환한다. 그 내부 cause가404여도 missing으로 간주하지 않는다. 오류가 생겼다고 SDK가 GET 확인·name 재선택·fallback·DELETE 재전송을 추가하지 않으며 configured Provider의 native retry/reauthentication 정책은 유지한다.

selected source/path/provider/version과 고정 DELETE method/URL은 공통 `cloudread`/`rest`/operation guard로 검사한다. shared authenticated Provider의 현재 토큰과 ServiceClient ResourceBase를 이용하며 별도 builder나 Adapter를 만들 필요가 없다.

Python mutable Resource의 dirty headers/alternate reference/instance mutation과 generic session lifecycle은 concrete Ref+error-only Go API와 다른 방식이며 별도 SDK 목표로 남는다. 현재 Go API는 호출마다 명시적인 Ref로 대상을 지정하고 error로 성공 여부를 전달한다.

## 단독 Go 예제

`-cloud dev -kind container -id CONTAINER_ID`, `-kind order -id ORDER_ID`, `-kind secret -id SECRET_ID`로 caller가 지정한 객체 하나를 삭제한다. `-strict`를 주면 없는 ID도 오류다. context timeout은 Connection 인증과 삭제를 포함한다.

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "time"

    sdk "gophercloudsdk"
    "gophercloudsdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml entry")
    kind := flag.String("kind", "container", "container, order or secret")
    id := flag.String("id", "", "explicit resource ID")
    strict := flag.Bool("strict", false, "return an error when the ID is missing")
    timeout := flag.Duration("timeout", time.Minute, "authentication and deletion timeout")
    flag.Parse()
    if err := run(*cloud, *kind, *id, *strict, *timeout); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(cloud, kind, id string, strict bool, timeout time.Duration) error {
    if timeout <= 0 || (kind != "container" && kind != "order" && kind != "secret") {
        return fmt.Errorf("positive timeout and kind container/order/secret are required")
    }
    ref := resource.ID(id)
    if err := ref.Validate(); err != nil {
        return err
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
    var options []resource.LookupOption
    if strict {
        options = append(options, resource.WithMissingError())
    }
    switch kind {
    case "container":
        err = service.Containers.Remove(ctx, ref, options...)
    case "order":
        err = service.Orders.Remove(ctx, ref, options...)
    case "secret":
        err = service.Secrets.Remove(ctx, ref, options...)
    }
    if err != nil {
        return err
    }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    return encoder.Encode(map[string]any{
        "kind": kind,
        "request_id": id,
        "ignore_missing": !strict,
        "completed": true,
    })
}
```

출력의 `completed`는 선택한 missing 정책으로 호출이 성공했다는 뜻이다. 기본 missing 무시에서는 이미 없던 ID도 true이며 실제 DELETE status나 삭제된 model을 합성하지 않는다. SDK 모듈 안의 별도 디렉토리에 `main.go`로 저장한다. 실제 OpenStack/Python 실행은 별도 환경에서 수행한다.

고정 source는 [Container 삭제](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L58), [Order 삭제](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L169), [Secret 삭제](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L270), [공통 proxy 정책](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L669), [body 없는 DELETE](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2094)다. 실제 검증과 판정은 [지원 판정대장](../../docs/sdk-support-ledger.md)에서 추적한다.
