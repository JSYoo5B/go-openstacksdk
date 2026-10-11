# Barbican waits: pinned Python and concrete Go records

`keymanager/v1/waits`는 Secret·Container·Order를 SDK가 직접 선택하고 조회하는 대기 API입니다. 애플리케이션에서 status adapter나 builder를 구현할 필요가 없습니다. 비교 기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`와 Gophercloud v2.15.0입니다.

| Python | Go |
| --- | --- |
| `Container.existing(id=id)` | `waits.Container(id)` |
| `Order.existing(id=id)` | `waits.Order(id)` |
| `Secret.existing(id=id)` | `waits.Secret(id)` |
| 기존 `Container`, `Order`, `Secret` 객체 | `ForContainer(FetchedContainer)`, `ForOrder(FetchedOrder)`, `ForSecret(FoundSecretRecord)` |
| `conn.key_manager.wait_for_status(res, status)` | `api.WaitForStatus(ctx, target, &status)` |
| 상태 값 `None` | `api.WaitForStatus(ctx, target, nil)` |
| `conn.key_manager.wait_for_delete(res)` | `api.WaitForDelete(ctx, target)` |
| `interval=...`, `wait=...` | `WithInterval(duration)`, `WithTimeout(duration)` |
| 상태 대기의 `wait=None` | 기본값 또는 `WithUnlimitedWait()` |
| 삭제 대기의 `wait=None` | `WithUnlimitedWait()` |
| `failures=[]` | `WithFailures()` |
| `attribute="sub_status"` | `WithAttribute("sub_status")` |
| `callback(progress)` | `WithProgressCallback(func(int) error)` |
| `res.to_dict()` | `json.MarshalIndent(record.Resource, "", "  ")` |
| 실제 관찰한 HTTP 증거 | `Wire`, `Envelope`, `Header`, `StatusCode`, `Payload` |

기존 leaf `WaitFor`/`WaitDeleted`와 이 패키지는 반환 타입과 Secret 조회 계약이 다릅니다. 이 패키지는 누적된 Source 속성과 마지막 실제 응답을 각각 보존하고, Secret.fetch의 조건부 payload 조회까지 완료한 뒤 상태를 판정합니다.

## 독립 Python 프로그램

`clouds.yaml`의 `dev` cloud를 준비하고 `python wait_barbican.py --kind order --id UUID --status ACTIVE`로 실행합니다. `--operation delete`는 삭제가 완료됐는지 조회하며 삭제 요청 자체를 보내지 않습니다.

```python
import argparse
import openstack
from openstack.key_manager.v1.container import Container
from openstack.key_manager.v1.order import Order
from openstack.key_manager.v1.secret import Secret

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--kind", choices=["container", "order", "secret"], default="order")
parser.add_argument("--id", required=True)
parser.add_argument("--operation", choices=["status", "delete"], default="status")
parser.add_argument("--status", default="ACTIVE")
parser.add_argument("--null-status", action="store_true")
parser.add_argument("--interval", type=float, default=2)
parser.add_argument("--timeout", type=float)
parser.add_argument("--disable-failures", action="store_true")
args = parser.parse_args()
conn = openstack.connect(cloud=args.cloud)
resource_type = {"container": Container, "order": Order, "secret": Secret}[args.kind]
res = resource_type.existing(id=args.id)
options = {"interval": args.interval, "callback": lambda progress: print("progress", progress)}
if args.timeout is not None:
    options["wait"] = args.timeout
if args.operation == "delete":
    result = conn.key_manager.wait_for_delete(res, **options)
else:
    if args.disable_failures:
        options["failures"] = []
    status = None if args.null_status else args.status
    result = conn.key_manager.wait_for_status(res, status, **options)
print(result.to_dict())
```

## 독립 Go main

[설치 안내](../../docs/install.md)에 따라 module을 준비하고 `go run . -kind order -id UUID -status ACTIVE`로 실행합니다. 양수 `-timeout 10m`를 주면 SDK timeout을 지정하고, 별도로 부모 context가 취소될 수도 있습니다.

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
    "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    kind := flag.String("kind", "order", "container/order/secret")
    id := flag.String("id", "", "literal executable ID")
    operation := flag.String("operation", "status", "status/delete")
    status := flag.String("status", "ACTIVE", "target state")
    nullStatus := flag.Bool("null-status", false, "target is Source None")
    interval := flag.Duration("interval", 2*time.Second, "polling interval")
    timeout := flag.Duration("timeout", 0, "positive override; zero keeps Source default")
    disableFailures := flag.Bool("disable-failures", false, "disable status failure states")
    flag.Parse()
    ctx := context.Background()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.KeyManagerV1(ctx)
    if err != nil { log.Fatal(err) }
    api := waits.New(service.RawClient())
    var target waits.Target
    switch *kind {
    case "container": target = waits.Container(*id)
    case "order": target = waits.Order(*id)
    case "secret": target = waits.Secret(*id)
    default: log.Fatal("unknown kind")
    }
    options := []waits.Option{
        waits.WithInterval(*interval),
        waits.WithProgressCallback(func(progress int) error {
            fmt.Println("progress", progress)
            return nil
        }),
    }
    if *timeout != 0 { options = append(options, waits.WithTimeout(*timeout)) }
    var result *waits.Record
    switch *operation {
    case "delete": result, err = api.WaitForDelete(ctx, target, options...)
    case "status":
        if *disableFailures { options = append(options, waits.WithFailures()) }
        selected := status
        if *nullStatus { selected = nil }
        result, err = api.WaitForStatus(ctx, target, selected, options...)
    default: log.Fatal("unknown operation")
    }
    if err != nil { log.Fatal(err) }
    data, err := json.MarshalIndent(result.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

## 기본값과 관찰 계약

상태 대기의 기본 timeout은 무제한이고 삭제 대기의 기본값은 120초입니다. 둘 다 interval은 2초입니다. `Options.Timeout == nil`은 이 기본값을 선택합니다. `WithTimeout(0)` 또는 음수 timeout은 첫 HTTP 전에 deadline 오류가 납니다. 상태가 캐시에 이미 맞으면 Source처럼 timer를 만들기 전에 성공하므로 이 경우 HTTP는 0회입니다. interval 0은 Source처럼 100ms 또는 더 짧은 양수 timeout을 pause로 사용합니다. 음수 interval은 Go에서 HTTP 전에 거부합니다. Python은 첫 조회 뒤 sleep 단계에서 실패할 수 있습니다.

상태 비교는 Python Unicode lower 규칙을 따르며 null·빈 문자열·일반 문자열을 구별합니다. 예를 들어 `İ`와 `i`는 같지 않고 `İ`와 `i`+combining dot은 같습니다. 기본 실패 상태는 `ERROR`이고, 명시적 빈 failures는 이를 비활성화합니다. 성공·실패 상태에서는 callback을 호출하지 않습니다. 세 Source 모델에는 progress descriptor가 없어 callback 값은 0입니다. Wire의 unknown progress가 callback 값을 바꾸지는 않습니다. Go callback은 오류를 반환할 수 있고 이 오류나 callback 중 취소·source 변경은 다음 조회 전에 중단시킵니다.

`Target`은 입력 리소스를 복사합니다. 캐시 성공은 그 복사본을 반환하며 원래 Go 리소스를 변경하지 않습니다. 각 poll의 Source view는 이전 선언 속성을 유지하고 현재 응답에 실제로 존재하는 선언 값만 갱신합니다. null·false·0은 omission과 다릅니다. `Wire`는 현재 응답에 실제 존재한 값만 보관하고 unknown 확장은 Source view에 추가하지 않습니다. Container/Order는 Source처럼 JSON 파싱 실패나 빈 응답에서 기존 속성을 유지하지만, 유효한 JSON의 nonobject/null은 관찰 오류입니다. Secret은 custom `.json()`을 사용하므로 malformed JSON도 오류입니다. Go는 모든 메타데이터에 valid UTF-8을 요구합니다.

Secret의 payload 선택은 **이전 synchronized Resource**의 nonnull payload_content_type을 우선하고, 없으면 **현재 메타데이터 응답**의 content_types.default를 사용합니다. 현재 응답의 payload_content_type만으로 첫 payload를 선택하지는 않습니다. 두 번째 poll부터 이전 poll에서 받은 값이 선택에 영향을 줄 수 있습니다. payload 요청은 처음 선택한 고정 ID 아래 `/payload`로 보내며, text/plain은 엄격한 UTF-8 문자열, 다른 종류는 Resource의 JSON base64와 Payload의 실제 bytes로 구분합니다. payload가 실패하면 ACTIVE·deleted 메타데이터만으로 대기를 완료하지 않습니다.

삭제 대기는 캐시의 deleted 상태가 있어도 첫 GET을 보냅니다. 깨끗한 **메타데이터** 404는 원래 Resource view를 반환하고 마지막 404 proof를 별도로 보존합니다. 조회된 deleted 상태는 갱신된 Resource를 반환합니다. null/nonstring status와 payload 404는 삭제 성공이 아닙니다. Read·Close·native retry·callback·source 변경·취소의 복합 오류와 실제 응답 status/body/header를 보존합니다. 오류 반환 record는 Resource가 nil이며 성공한 리소스로 사용할 수 없습니다. 실패 후 마지막 metadata 및 payload receipt를 점검할 수 있습니다.

## 경계와 남은 타입

대기는 GET 권한을 따릅니다. Order는 `order:get`, Container는 `container:get`, Secret은 `secret:get`, payload가 선택되면 `secret:decrypt`가 적용됩니다. 프로젝트 member·owner·ACL 조건은 서버가 판정하며 별도 wait 권한은 없습니다. [Barbican 2025.2 정책](https://docs.openstack.org/barbican/2025.2/configuration/policy.html).

이 API는 Source의 임의 Resource subclass, mutable object identity, Connection location 및 response cache를 재현하지 않습니다. 고정 route/version/SDK 헤더 정책을 소유하며 응답 ID/HREF나 native callback이 다음 요청을 다른 리소스로 돌릴 수 없습니다. 이는 Python 객체의 id 변경 가능성에 대한 명시적 Go guard입니다. Secret.fetch의 unchecked HTTP 응답과 달리 Go는 모든 거부 응답을 오류로 처리합니다. 임의 Python callback 값·동적 getattr은 지원하지 않고 선언된 nullable string 속성과 `func(int) error`를 제공합니다.

**wait_for_status/wait_for_delete 전체 선언은 아직 미완료입니다.** SecretStore는 조회 가능한 status 모델이지만 기존 공통 projector의 세 타입 범위 밖입니다. SecretACL·일반 Quota는 fetch 가능한 모델이며 status가 없더라도 다른 선언 문자열 속성을 대상으로 하거나 clean404 삭제 대기를 할 수 있습니다. 이 타입들의 raw projection과 concrete route constructors는 남아 있습니다. ProjectQuota는 admin 단계에서 별도 다룹니다. SecretConsumer는 pinned Source의 allow_fetch가 false여서 일반 polling 자체가 지원되지 않습니다. 임의 Resource subclass는 SDK-R1의 일반 경계로 구분하며 위의 알려진 concrete 타입 누락을 그 경계로 숨기지 않습니다.

Source: [proxy wait declarations](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L615-L674), [shared status/deletion wait](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2585-L2715), [Secret.fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/secret.py#L91-L138), [timeout iterator](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53-L100).
