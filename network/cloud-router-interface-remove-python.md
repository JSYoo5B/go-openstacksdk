# Cloud 라우터 인터페이스 제거: Python과 Go

고정 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 `conn.remove_router_interface(router, subnet_id=None, port_id=None)`에 대응하는 호출은 `service.RemoveRouterInterface(ctx, routerID, options...)`입니다. `conn.Network(ctx)`로 서비스를 선택한 뒤 concrete 옵션만 전달합니다. 양쪽 인자 검사, Python truthiness와 SDK Resource ID, port 우선 선택, PUT, 응답 JSON 처리와 실제 오류 증거를 SDK가 소유합니다. 사용자 builder/interface나 별도 조회 분기는 필요 없습니다. Gophercloud 기준은 `v2.15.0`입니다.

| Python | Go | 의미 |
| --- | --- | --- |
| `conn.remove_router_interface(router, subnet_id="s")` | `service.RemoveRouterInterface(ctx, router, network.WithRouterInterfaceRemoveSubnetID("s"))` | subnet만 있으면 subnet_id 전송 |
| `port_id="p"` | `WithRouterInterfaceRemovePortID("p")` | truthy port가 subnet보다 우선 |
| `subnet_id=None`, `port_id=None` | 인자 옵션 생략 또는 raw JSON null | 서비스/라우터 검증 전에 로컬 오류 |
| cached `Port`/`Subnet` Resource | `WithRouterInterfaceRemovePortID(record)` / `WithRouterInterfaceRemoveSubnetID(record)` | owned PortRecord/SubnetRecord/RawResource의 ID를 SDK가 추출 |
| plain `{"id":"p"}` | 같은 JSON object | Resource가 아니므로 object 자체를 전송 |
| 성공 결과 `None` | `result.Value` = JSON `null` | 응답 Body를 router Resource로 만들지 않음 |
| Proxy가 읽은 실제 `response.json()` | `result.ResponseJSON` | Source cloud가 버리는 원래 parsed JSON, 진단용 |
| 실제 HTTP | `Wire`, `Envelope`, `Header`, `StatusCode` | 상태·본문·헤더 증거, independent copies |

## 기본값과 순서

subnet_id와 port_id가 모두 falsey이면 Source처럼 HTTP 전에 거부합니다. None/null·false·0·빈 문자열/list/object·float underflow는 falsey입니다. 기본값도 이 경우입니다. Service는 이 검사를 라우터 ID와 network service 구성 오류보다 먼저 처리합니다. 취소·nil 옵션·잘못된 Go JSON은 명시적인 Go preflight 오류입니다.

검사를 통과하면 truthy port가 있으면 port_id만, 아니면 subnet_id만 보냅니다. Source 문서의 두 인자 설명과 달리 실제 코드도 둘 다 truthy일 때 port만 전송합니다. subnet/port의 이름을 조회하거나 ID 존재를 미리 GET하지 않습니다. 라우터 문자열도 이름 조회 없이 literal ID입니다. 포트와 서브넷의 실제 연결 관계·존재·권한은 서버가 검사합니다.

cached SDK Resource는 ID가 null/false/0/빈 문자열이어도 Resource 자체가 truthy입니다. 따라서 cached Port(id=None)은 port_id:null을 보내며 subnet으로 fallback하지 않습니다. plain JSON dict는 Resource._get_id 대상이 아니어서 `{"port_id":{"id":null}}`로 보존합니다. RawResource와 owned PortRecord/SubnetRecord를 concrete 옵션이 처리하며 호출자가 per-record 변환기를 작성할 필요가 없습니다.

`WithRouterInterfaceRemoveOpts`는 refs·raw bytes·Headers를 복제합니다. 뒤 individual 옵션은 해당 인자만 바꾸고, bulk 옵션은 전체 제거 설정을 대체합니다. 옵션 callback에 노출된 임시 config도 다음 단계의 설정과 분리됩니다. 선택한 service/API/client/source는 callback 전에 캡처하며 변경을 감지하면 후속 callback과 HTTP를 즉시 중단합니다. Header map과 live 인증 token을 분리하여 정상 token renewal은 허용합니다. Headers는 Go transport convenience이고 caller client의 기본 microversion을 유지합니다.

Connection에 새 forwarding 메서드는 추가하지 않습니다. `Connect`/`Network`를 먼저 호출하는 예제에서는 인증·catalog/service discovery가 cloud 인자 검사보다 앞설 수 있습니다. Source cloud 메서드 자체의 missing-ID 검사가 network Proxy 접근보다 앞선다는 계약과 이 초기화 순서를 구분해야 합니다. 이미 선택한 Service 또는 nil Service의 직접 호출에서는 missing-ID 검증이 HTTP 0입니다.

## 응답과 실패

Source는 Proxy의 `response.json()`을 실행한 뒤 그 값을 버리고 None을 반환합니다. 따라서 204의 빈 본문이나 malformed accepted JSON은 성공 None이 아닌 decode 오류입니다. valid JSON object/list/string/number/bool/null은 모두 읽은 뒤 성공 Value:null로 반환합니다. ResponseJSON·Wire·Envelope에 원래 응답을 보존하며 synthetic router ID/default/dirty state를 만들지 않습니다. 성공 PUT은 acknowledgement이며 backend convergence 증거가 아닙니다.

새 closed `routers.API.RemoveCloudInterfaceRecord`는 기존 reference/body/header/JSON projector와 REST를 재사용하며 Source100..399와 physical-fault callback suppression만 opt in합니다. 기존 `RemoveInterfaceRecord`의 default `subnet_id:null` 전송과 200..399 정책, native `RemoveInterface`의 required/status 정책은 바꾸지 않습니다. 이 helper는 단일 cloud 선언의 dependency이며 새 Source API를 추가로 완료했다고 집계하지 않습니다.

clean403/404/409는 그대로 오류이고 missing이나 성공으로 바꾸지 않습니다. read·Close·취소·source 오류는 accepted ResponseError 또는 native rejected evidence의 실제 body/header/status와 원인을 유지하며 retry/backoff/reauth callback으로 replay하지 않습니다. native401 `ErrUnableToReauthenticate`는 Unwrap이 없어 ErrOriginal의 value `ErrUnexpectedResponseCode`와 ErrReauth를 명시적으로 확인해야 합니다. 정상 clean503 retry와401 token renewal은 고정 body/route/headers로 유지됩니다. 실패 결과에는 Value/ResponseJSON 성공 자료가 없습니다. Wire가 있다면 실제 JSON object 관찰일 뿐 성공 Resource가 아닙니다.

## 독립 Python main

아래 예제는 실제 인터페이스 제거 PUT을 실행합니다.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--router", required=True)
parser.add_argument("--subnet-json", default="null")
parser.add_argument("--port-json", default="null")
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    result = conn.remove_router_interface(
        args.router,
        subnet_id=json.loads(args.subnet_json),
        port_id=json.loads(args.port_json),
    )
    print(json.dumps(result))
```

## 독립 Go main

[설치 안내](../docs/install.md)의 module/clouds.yaml 준비 후 main.go로 저장합니다. Go는 JSON 숫자를 float64로 먼저 바꾸지 않고 raw bytes를 전달합니다. 두 예제 모두 문자열 ID와 null/false/0 등의 입력을 받으며 Python은 json.loads의 원래 numeric 규칙을 사용합니다.

```go
package main

import (
 "context"
 "encoding/json"
 "flag"
 "fmt"
 "os"
 "time"

 sdk "github.com/JSYoo5B/go-openstacksdk"
 "github.com/JSYoo5B/go-openstacksdk/network"
)

func run(ctx context.Context, cloud, router, subnetJSON, portJSON string) error {
 conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
 if err != nil { return err }
 service, err := conn.Network(ctx)
 if err != nil { return err }
 result, err := service.RemoveRouterInterface(ctx, router,
  network.WithRouterInterfaceRemoveSubnetID(json.RawMessage(subnetJSON)),
  network.WithRouterInterfaceRemovePortID(json.RawMessage(portJSON)))
 if result != nil {
  fmt.Fprintf(os.Stderr, "PUT status=%d request-id=%q\n",
   result.StatusCode, result.Header.Get("X-Request-Id"))
 }
 if err != nil { return err }
 _, err = fmt.Fprintln(os.Stdout, string(result.Value))
 return err
}

func main() {
 cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
 router := flag.String("router", "", "literal router ID, not a name lookup")
 subnet := flag.String("subnet-json", "null", "subnet_id as JSON")
 port := flag.String("port-json", "null", "port_id as JSON; a truthy port wins")
 flag.Parse()
 if *router == "" {
  fmt.Fprintln(os.Stderr, "router is required")
  os.Exit(2)
 }
 ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
 defer cancel()
 if err := run(ctx, *cloud, *router, *subnet, *port); err != nil {
  fmt.Fprintln(os.Stderr, err)
  os.Exit(1)
 }
}
```

`python main.py --cloud dev --router router-id --subnet-json '"subnet-id"'`

`go run . --cloud dev --router router-id --subnet-json '"subnet-id"'`

`python main.py --cloud dev --router router-id --port-json '"port-id"'`

`go run . --cloud dev --router router-id --port-json '"port-id"'`

두 ID 옵션을 생략하면 Source validation 오류입니다. 성공 출력은 두 예제 모두 `null`입니다.

## 권한·표현 범위·검증

[Neutron 27 router 정책](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/router.py#L384-L398)의 remove_router_interface는 project scope의 ADMIN_OR_PROJECT_MEMBER입니다. parent-owner reader나 shared-subnet read 권한만으로 이 action을 허용하는 규칙은 아닙니다. 배포 policy·project ownership·포트/서브넷/라우터의 실제 제약은 서버가 최종 판단하며 SDK는 권한을 확대하지 않습니다.

Go router 인자는 검증된 비어 있지 않은 문자열 ID입니다. slash/percent/query 문자는 한 URI segment로 escape하고 blank/control/traversal ID는 거부합니다. Source의 arbitrary Router Resource/dict/null/nonstring router dispatch와 unsafe URI 조합까지 지원한다고 주장하지 않습니다. subnet/port는 raw JSON과 위 concrete SDK Resource를 표현하며 Python의 임의 runtime object·mutable Resource/Container/class/descriptor/session/cache 동작은 SDK-R1·C1·S1 별도 foundation remaining입니다. JSON float serialization·Unicode/runtime domain과 wire key order도 명시된 Go 표현 범위입니다. 이번 대상은 cloud remove_router_interface 한 선언이며 전체 SDK supported 증가량은 0입니다. 현재 reused JSON decoder는 UTF-8만 허용하지만 Source Requests.response.json()은 명시 ISO-8859-1 charset이나 UTF-16 등의 정상 wire JSON도 읽습니다. 이 경우 Go가 decode 오류가 되는 구체적인 wire-response gap이 남아, charset-aware seam을 검증하기 전 이 행은 unresolved입니다.

CPython3.14.8에서 실제 pinned Cloud.remove_router_interface→Neutron Proxy→Resource._get_id/_get_resource→Router._put/response.json을 실행한 75-observation Source receipt를 작성했습니다. Adapter.put과 _get_connection만 fixture이며 live cloud/실제 인증/ACL/배포 수락 증거가 아닙니다. 별도 독립 감사의 bounded router13 observations와 cached Resource falsey-ID truthiness 검토도 구현과 대조합니다. 오류·status·required/nullable·port 우선·라우터 lookup 없음·JSON을 읽은 뒤 None 반환을 구분합니다.

테스트는 기존 public Gophercloud testhelper 기반 testcloud/nativeRouter와 real response body observer를 재사용한 cloud8+leaf2, 총10그룹입니다. root의 Go/race 및 exact main build 결과와 Python AST/guide fence byte 일치는 서로 다른 검증입니다. 미실행 Go/전체 Source/live cloud PASS를 주장하지 않습니다.
