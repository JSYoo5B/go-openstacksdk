# Nova cloud 콘솔 출력: openstacksdk와 Go 사용 비교

`compute.Service.GetServerConsole`과 `Connection.GetServerConsole`은 서버 이름/ID를 조회하거나 이미 가진 dictionary를 사용해 콘솔 출력을 가져옵니다. SDK가 조회, 기본값, 옵션 복사, 응답 처리와 서비스 연결을 소유하므로 호출자가 interface builder를 구현할 필요가 없습니다.

기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, Gophercloud `v2.15.0`, Go `1.25.0`입니다. 아래 closed API는 사용할 수 있지만 named `get_server_console` 행은 Requests JSON의 나머지 문자셋 선택·추정 경로와 Python의 비유한 숫자 결과에 대한 잔여 2개 때문에 **unresolved**입니다. 이 부분 구현을 Python 전체 SDK parity 또는 `supported` 완료 수로 집계하지 않습니다.

| 사용 목적 | openstacksdk | Go |
|---|---|---|
| 이름 또는 ID | `conn.get_server_console("vm")` | `conn.GetServerConsole(ctx, "vm")` |
| 마지막 N줄 | `length=50` | `compute.WithServerConsoleLength(50)` |
| 생략과 명시적 0 | `None`과 `0` | 옵션 생략/`WithoutServerConsoleLength()`와 `WithServerConsoleLength(0)` |
| 이미 가진 서버 | `{"id": "s1"}` 또는 `Server` | `WithServerConsoleServerJSON(raw)` 또는 `WithServerConsoleServerRecord(record)` |
| literal length | 런타임은 `false`, 목록, dict도 변환 없이 전송 | `WithServerConsoleLengthJSON(raw)` |
| 콘솔 미지원 | 콘솔 **action** 400이면 `""` | `Value`가 JSON `""`, `Unsupported=true`, 실제 `Action.StatusCode=400` |
| output 키가 없음 | `None` | `Value`가 JSON `null`, `HasOutput=false` |
| output 값이 명시적 null | `None` | `Value`가 JSON `null`, `HasOutput=true` |
| 응답 증거 | Response/Resource 경로에 보관 | 조회는 `Server`, 입력은 `Seed`, POST는 `Action`으로 별도 보관 |

서버를 문자열로 받으면 원본의 `get_server(server, bare=True)` 경로를 따릅니다. 기존 `servers.API.FindRecord`가 GET을 먼저 시도하고, clean 400/403/404일 때만 상세 목록으로 이름을 찾습니다. 기본 `details=True`, `all_projects=False`이며 누락/중복 조회는 오류입니다. `bare=True`는 원본에서도 network/image/flavor 확장을 바로 건너뛰므로 여기에는 다른 서비스 조회가 필요하지 않습니다. Cloud 콘솔의 400 catch는 조회가 끝난 뒤 action에만 적용됩니다.

Go 비교 예제는 두 언어 모두 `clouds.yaml`의 `devstack`을 사용합니다. `--server-json`은 이름 조회를 건너뛰고 `--length-json null`은 length를 생략합니다. Python은 원래 반환값을 JSON으로 출력하고, Go는 값과 presence/미지원/실제 상태 정보를 함께 출력합니다.

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="devstack")
    parser.add_argument("--server", default="vm")
    parser.add_argument("--server-json", default="")
    parser.add_argument("--length-json", default="")
    args = parser.parse_args()
    server = json.loads(args.server_json) if args.server_json else args.server
    length = json.loads(args.length_json) if args.length_json else None
    with openstack.connect(cloud=args.cloud) as conn:
        output = conn.get_server_console(server, length=length)
        print(json.dumps(output, ensure_ascii=False))


if __name__ == "__main__":
    main()
```

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
    "github.com/JSYoo5B/go-openstacksdk/compute"
)

func run(ctx context.Context, cloud, server, suppliedServer, length string) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return err }
    var options []compute.ServerConsoleOption
    if suppliedServer != "" {
        options = append(options, compute.WithServerConsoleServerJSON(json.RawMessage(suppliedServer)))
    }
    if length != "" {
        options = append(options, compute.WithServerConsoleLengthJSON(json.RawMessage(length)))
    }
    result, err := conn.GetServerConsole(ctx, server, options...)
    if err != nil { return err }
    if result == nil { return fmt.Errorf("console result is missing") }
    return json.NewEncoder(os.Stdout).Encode(struct {
        Output json.RawMessage `json:"output"`
        HasOutput bool `json:"has_output"`
        Unsupported bool `json:"unsupported"`
        StatusCode int `json:"status_code"`
    }{result.Value, result.HasOutput, result.Unsupported, result.Action.StatusCode})
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    server := flag.String("server", "vm", "literal name or ID; ignored when server-json is supplied")
    suppliedServer := flag.String("server-json", "", "owned server dictionary, for example {\"id\":\"s1\"}")
    length := flag.String("length-json", "", "optional literal JSON length; empty flag omits it")
    timeout := flag.Duration("timeout", time.Minute, "whole lookup and action deadline")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), *timeout)
    defer cancel()
    if err := run(ctx, *cloud, *server, *suppliedServer, *length); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

실행 예:

```sh
python main.py --cloud devstack --server vm --length-json 50
go run main.go --cloud devstack --server vm --length-json 50
python main.py --cloud devstack --server-json '{"id":"s1"}' --length-json 0
go run main.go --cloud devstack --server-json '{"id":"s1"}' --length-json 0
```

서비스를 이미 얻었다면 같은 옵션을 바로 사용합니다:

```go
service, err := conn.Compute(ctx)
if err != nil { return err }
result, err := service.GetServerConsole(ctx, "vm", compute.WithServerConsoleLength(50))
```

서버 ID가 이미 확정됐고 lookup/400 변환이 필요 없는 경우에는 `service.API.Servers.ConsoleOutputRecord(ctx, "s1", servers.WithCloudConsoleLength(50))`을 사용할 수 있습니다. 이 leaf는 실제 native HTTP 400을 오류로 유지합니다. 기존 `ConsoleOutput`/generated `ShowConsoleOutput`의 문자열 projection과 transport 계약은 그대로 유지됩니다.

`ServerConsoleOpts`의 `Server`/`Length`는 각각 `*json.RawMessage`이고 `WithServerConsoleOpts`는 생성 시점과 적용 시점에 값을 복사합니다. `WithServerConsoleServerRecord`는 현재 `Resource.Body`를 dictionary seed로 복사하며 과거 lookup receipt를 새 POST의 응답으로 취급하지 않습니다. `WithoutServerConsoleServer`는 다시 문자열 조회로, `WithoutServerConsoleLength`는 기본 length 생략으로 돌립니다. 다른 유한 JSON length 값은 Nova가 검증하도록 그대로 보냅니다.

응답의 `output` 키는 대소문자를 구분합니다. `Output`은 누락으로 처리하고 null/false/0/목록/dict 값을 문자열로 강제 변환하지 않습니다. object가 아닌 JSON에서도 원본 membership 규칙을 유지합니다. `[]`나 `"quiet"`는 키가 없어 null이며, `["output"]`나 `"contains output"`는 string indexing 불가 오류입니다. root null/bool/숫자, 빈 204 본문, 깨진 JSON도 오류이며 실제 `Action.Envelope`, `Header`, `StatusCode`가 남습니다. `Value`와 `Envelope`는 서로 다른 복사본입니다. 명시된 응답 charset의 UTF-16(BOM/LE/BE), UTF-32(BOM/LE/BE), ISO-8859-1/latin1은 SDK 공통 `internal/responsejson.Normalize`에서 UTF-8 JSON으로 변환한 뒤 output을 읽습니다. `Value`에는 변환된 필드 값이 들어가고 `Action.Envelope`·`Header`와 `ResponseError.Body`에는 실제 wire 바이트·헤더를 보존합니다. charset이 없거나 UTF-8이면 기존 바이트를 복사하여 사용하며 인코딩을 추정하지 않습니다. 알 수 없거나 잘못된 charset 지정은 `ErrUnsupported`와 실제 응답 증거를 반환합니다.

accepted action 범위는 Source `raise_from_response`의 100..399입니다. 실제 net/http가 정보성 응답을 최종 응답으로 노출하지 않는 경우가 있으므로 199 계약 검증은 합성 RoundTripper lane에 한정합니다. 선택된 microversion은 유지하고, 없으면 기존 Nova discovery로 Server ceiling `2.100` 이하를 선택합니다. action의 `Accept: ""`, 버전, 경로, JSON body, 인증/프레이밍 header 경계는 native retry에도 고정하며 caller의 shared client를 변경하지 않습니다. Cloud는 옵션 callback 전에 private reader의 헤더를 캡처하고 조회부터 action까지 같은 복사본을 유지합니다. Callback이 ordinary shared header를 바꾸더라도 이미 시작한 workflow의 헤더를 바꾸지 않는 Go ownership 정책입니다.

Dictionary의 falsey id(null/false/0/empty string)는 Source와 같은 `/servers//action` 경로를 유지합니다. id 자체가 dict이면 Proxy `_get_resource`가 읽는 선택된 inner id 경로를 한 차례 추출합니다. outer dictionary에 id 키가 없으면 action 전에 실패하고, inner dictionary에 id가 없으면 Source처럼 falsey `/servers//action` 경로를 사용합니다. 이 좁은 추출은 arbitrary constructor attrs/descriptor 전체 변환을 구현했다는 뜻이 아닙니다. 일반 ID는 같은 service origin 안의 literal segment로 escape하며 controls/dot segment는 action HTTP 전에 거부합니다. Length JSON 검증도 leaf action 전에 수행하므로, cloud의 문자열 입력에서는 서버 조회 HTTP가 먼저 일어날 수 있습니다. 이는 Python의 임의 객체 문자열화와 mutable Resource/session 전체를 대신하는 닫힌 JSON 입력 정책입니다.

clean action 400은 body Read/Close, source guard, context, native callback과 재인증 처리가 성공한 응답을 뜻합니다. 400 본문 읽기/Close/취소/route 복원/추가 callback 오류, native hook이 성공 코드 집합을 넓힌 400은 `Unsupported`로 숨기지 않습니다. 403/404 및 다른 오류도 유지합니다. 실제 401의 원본 native body/header와 재인증 실패는 별도 필드가 있는 Gophercloud error에서도 보존합니다. action과 콘솔 leaf의 microversion discovery는 공통 `StopFaultRetryCallbacks`를 명시적으로 opt-in하여 물리 실패 뒤 retry/auth/backoff callback을 막습니다. 기존 공통 profile 기본값은 바꾸지 않습니다. 재사용한 FindRecord의 기존 lookup callback 정책도 새 계약으로 바꾸지는 않습니다. 이 HTTP POST에는 exactly-once 실행 보장이 없으므로 native retry 정책을 구성할 때 콘솔 조회 action의 성격을 고려해야 합니다.

미완료 경계는 다음 두 개입니다.

1. Requests `Response.json()`의 전체 문자셋 선택·Python codec registry·charset 없는 BOM/`guess_json_utf`·text fallback. 명시 charset의 UTF-16(BOM/LE/BE), UTF-32(BOM/LE/BE), ISO-8859-1/latin1은 지원하지만 임의 codec과 인코딩 추정 전체를 구현하지는 않습니다. charset 없는 비UTF-8 응답 및 미지원·잘못된 charset 지정은 `ErrUnsupported`와 실제 raw body/status/header를 유지합니다. BOM 없는 bare `utf-16`/`utf-32`의 native-endian 선택도 미지원입니다. 이 좁은 추가 구현으로 named 행을 완료 처리하지 않습니다.
2. Python JSON decoder가 반환할 수 있는 비유한 float/NaN/Infinity 표현. 실제 원본 `{"output":1e400}`은 `float('inf')`를 반환합니다. Go의 유한 raw JSON carrier는 이를 `ErrUnsupported`로 명시합니다. 유한 raw 숫자는 원래 literal과 큰 정수를 보존하고, Python float로 사전 변환하지 않는 Go 정책입니다.

기반 SDK의 Python Resource class/session/cache/constructor 완전 재현(SDK-R1/C1/S1), 임의 객체를 전달하는 mutation/attrs 경로는 별도 잔여입니다. 이 API가 존재한다고 `Proxy.get_server_console_output`, mutable `Server` 전체 또는 cloud `get_server`의 일반 확장 경로까지 완료했다고 판정하지 않습니다.

검증 근거: pinned 실제 Cloud/Proxy/Server와 Requests `2.34.2`/urllib3 `2.8.0`/CPython `3.14.8`을 호출한 Source 관찰 **68개**를 `/private/tmp/go-openstacksdk-cycle8-console-runtime.py`와 `-runtime-reference.json`에 보존했습니다. HTTP post는 메모리 `http.client.HTTPResponse`, find는 고정 Source Server/없음/오류 adapter로 대체했습니다. 이는 원본 lookup HTTP/auth/retry, live Nova, 운영 cloud ACL 검증이 아닙니다. Go 회귀는 기존 `internal/testcloud`, Gophercloud public testhelper 기반 cloud fixture, 기존 native server/physical body fixture를 재사용합니다. 담당자는 Go 검사를 실행하지 않았으며 root의 최종 검증 결과가 들어오면 별도 기록합니다. 가이드의 Python main은 AST 검증을 통과했고 Go main 빌드는 root 통합 검사 대상입니다.

명시 charset 추가의 Source 근거는 기존 68개 관찰 중 ISO-8859-1 `café` 응답과 동일한 Requests `Response.json()` 경로입니다. 새 Source matrix를 실행하거나 관찰 수를 늘리지 않았습니다. UTF-16/32/Latin-1 변환은 root 소유 공통 `responsejson` 검증에 통합하며, 이 leaf는 대표 세 charset 가족의 값/원본 응답 분리와 실패 원인 보존 회귀 2그룹으로 연결을 검증합니다. 이번 변경의 Go 검사와 main 빌드 결과는 root 통합 검증 대상입니다.

후속 charset 통합 검증: root의 콘솔 leaf/cloud 집중25그룹184사례, 독립 Go main 빌드·Python AST, 전체205개 test package gate가 PASS했습니다. 실클라우드 검증은 별도입니다.
