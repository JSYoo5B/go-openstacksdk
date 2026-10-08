# Compute console 자동 선택

`conn.Compute(ctx)`의 `CreateConsole`은 고정 openstacksdk의 `conn.compute.create_console`에 대응합니다. 서버가 광고한 microversion 범위와 선택된 버전으로 modern remote-console 또는 legacy action을 선택합니다. 호출자는 서버 ID·console type·선택 protocol을 전달합니다.

| openstacksdk | gophercloudsdk |
|---|---|
| `conn.compute.create_console(server, "novnc")` | `service.CreateConsole(ctx, serverID, "novnc")` |
| `console_protocol="vnc"` | `compute.WithConsoleProtocol("vnc")` |
| `console_protocol=None` | 기본값 또는 `WithConsoleProtocolValue(request.Null[string]())` |
| modern Resource의 `to_dict()` | protocol/type/url/id/name/location을 가진 `json.RawMessage` |
| legacy console 값 | 실제 console JSON object/null/scalar/array를 가진 `json.RawMessage` |

```python
import openstack

conn = openstack.connect()
console = conn.compute.create_console("server-id", "novnc")
print(console)
```

아래 main은 인증 환경을 읽고 `-server-id SERVER_ID`의 console을 생성합니다. `-console-type` 기본값은 예제에서 정한 novnc이며 API의 type 인자는 필수입니다. `-protocol`을 지정하면 빈 문자열도 명시 값으로 전달합니다. 인증·실제 OpenStack 작업 실행은 사용자가 준비한 cloud에서 수행합니다.

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/gophercloudsdk"
    "github.com/JSYoo5B/gophercloudsdk/compute"
)

func main() {
    serverID := flag.String("server-id", "", "server ID")
    kind := flag.String("console-type", "novnc", "console type")
    protocol := flag.String("protocol", "", "optional console protocol")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *serverID, *kind, *protocol); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, serverID, kind, protocol string) error {
    if serverID == "" {
        return fmt.Errorf("-server-id is required")
    }
    conn, err := sdk.Connect(ctx)
    if err != nil {
        return err
    }
    service, err := conn.Compute(ctx)
    if err != nil {
        return err
    }
    var options []compute.ConsoleOption
    flag.Visit(func(f *flag.Flag) {
        if f.Name == "protocol" {
            options = append(options, compute.WithConsoleProtocol(protocol))
        }
    })
    console, err := service.CreateConsole(ctx, serverID, kind, options...)
    if err != nil {
        return err
    }
    fmt.Println(string(console))
    return nil
}
```

## 분기와 microversion

SDK는 선택된 catalog endpoint에서 project suffix를 제거한 version 경로를 먼저 GET합니다. discovery 요청에는 microversion 헤더와 body가 없습니다. 깨끗한 HTTP404/405 또는 유효한 문서에 일치하는 API 버전 행이 없는 경우에만 같은 prefix의 root 경로를 확인합니다. 응답 링크로 origin·인증 scope를 변경하지 않습니다. endpoint가 v2.1이면 discovery의 v2.0 행은 선택하지 않습니다. Nova 행은 허용된 status와 usable self-link도 있어야 합니다. missing/unknown status와 self-link가 없는 행은 제외합니다. self-link는 행의 유효성을 확인하는 데만 쓰며 요청 경로를 바꾸지 않습니다.

modern 분기는 광고된 min/max가 모두 있고 그 범위에2.6이 포함될 때 가능합니다. 선택된 버전이 있으면 같은 major2이고2.6 이상이어야 합니다. 선택2.100이 광고 max2.90보다 높더라도 이를 자동 낮추지는 않습니다. 광고 max2.5 또는 min2.7처럼2.6을 포함하지 않으면 선택 버전이 높아도 legacy 분기입니다. 범위가 없으면 legacy를 사용합니다.

선택 버전이 비어 있으면 광고 max를 modern2.99 또는 legacy2.100 한도로 선택해 작업의 private client에만 적용합니다. 원래 service client·cached Connection client는 바꾸지 않습니다. global `latest`는 유한 major2 조건을 증명하지 못하고, raw client에 명시한 `2.latest`는 같은 major의 비교를 통과하며 문자 그대로 전송됩니다. Connection의 공개 exact-version 설정 검증은 별도 정책입니다.

modern type `webmks`는 광고·선택 범위에서2.8, `spice-direct`는2.99 지원도 확인합니다. 부족하면 console POST 전 오류입니다. modern/legacy 선택은 console 요청 전에 끝나며 선택한 POST가403/404 등으로 실패해도 다른 API로 재시도하지 않습니다. 지원하지 않는 legacy type은 legacy 실행기의 오류를 반환합니다.

## Protocol·반환·오류

기본 protocol은 Python None에 해당하는 explicit JSON null seed입니다. modern 실행기는 알려진 type의 protocol을 요청 body에 추론하되 반환 seed를 바꾸지 않습니다. 따라서 응답이 protocol을 생략하면 기본 반환 protocol은 null입니다. 명시 빈 문자열도 같은 추론에 참여하고 반환 seed는 빈 문자열입니다. unknown modern type은 server가 검사합니다. legacy는 protocol을 사용하지 않습니다.

modern 결과는 protocol/type/url과 inherited Body id/name, computed location의6개 필드입니다. 응답 known 값은 null·큰 수·untyped 값도 그대로 보존하고 missing id/name은 null입니다. URI의 server_id·unknown 필드·응답이 제공한 location은 dictionary에 넣지 않습니다. `conn.Compute(ctx)`는 설정된 cloud/project/region location을 제공하고 zone은 null입니다. `compute.New(client, compute.Dependencies{})`로 직접 만들면 location은 null입니다. 기존 [modern 실행기의 Resource·Wire·receipt](user-actions.md)와 [legacy 실행기의 raw 값](keypairs-console.md)은 각각 직접 사용할 수 있습니다.

반환은 독립 JSON snapshot이며 Python mutable Resource/session 객체는 제공하지 않습니다. Go는 검증된 단일 path segment 서버 ID와 nullable string protocol을 사용합니다. accepted modern 비JSON/빈 응답은 기존 seed 정책을 유지하고, legacy는 실제 console key가 있는 유효 JSON을 요구합니다. Go의 고정 경로 reader는 첫 일치 행을 사용합니다. Python의 전체 version-data 정렬·일치하지 않는 모든 행의 bounds 정규화·전역 discovery cache는 이 작업에서 재현하지 않습니다. Go discovery는 captured token으로 인증하며 Python의 anonymous GET·401 후 인증 전환을 별도로 구현하지 않습니다. Python이 다른 discovery/HTTP 연결 오류에도 다음 경로를 시도할 수 있는 것과 달리 Go는 위의 clean404/405·유효 unmatched 조건만 허용합니다. discovery JSON 오류·read/Close 오류·source 변경·취소는 terminal이며 이후 POST를 하지 않습니다. 실제 HTTP·accepted processing 오류는 해당 단계의 body/header/status와 원래 cause를 보존합니다. UTF-8 정책과 Python response charset 처리 차이는 기존 실행기 문서와 같습니다.

## 검증과 재사용

새 composition 테스트는 public Gophercloud testhelper와 기존 testcloud·flavorIdentityClient·payloadContractTrack·secretFetchRoundTripFunc를 사용합니다. 공통 microversion 비교·유한 discovery는 기존 Cinder 구현에서 추출해 공유하고, Cinder import와 Connection discovery의 기존 테스트를 함께 실행합니다. 별도 HTTP harness나 공통 transport fault 표를 복제하지 않습니다. 실제 OpenStack 인증·Python 실행을 대신하는 검증은 아니며 이 main의 외부 소비자 빌드도 별도로 기록합니다.
