# Compute remote console 생성과 키페어 삭제

`conn.ComputeV2(ctx)`의 `RemoteConsoles.CreateConsole`과 `KeyPairs.DeleteKeypair`는 고정 Python Proxy의 기본값과 옵션 조합을 SDK가 처리합니다. caller builder나 별도 resolver를 구현하지 않고 명시 서버 ID·키페어 이름과 concrete 옵션을 전달합니다. 두 작업 모두 선행 GET/LIST나 완료 대기를 추가하지 않습니다.

## Python과 Go의 대응

| 고정 openstacksdk | go-openstacksdk |
|---|---|
| `conn.compute.create_server_remote_console(server_id, type="novnc")` | `service.RemoteConsoles.CreateConsole(ctx, serverID, remoteconsoles.WithConsoleCreateType("novnc"))` |
| 명시 `protocol="vnc"` | `remoteconsoles.WithConsoleCreateProtocol("vnc")` |
| 명시 `url="..."` | `remoteconsoles.WithConsoleCreateURL("...")` |
| `protocol=None` | `WithConsoleCreateProtocolValue(request.Null[string]())` |
| `conn.compute.delete_keypair(name)` | `service.KeyPairs.DeleteKeypair(ctx, name)` |
| `delete_keypair(name, ignore_missing=False)` | `keypairs.WithKeypairDeleteIgnoreMissing(false)` 추가 |
| `delete_keypair(name, user_id=owner_id)` | `keypairs.WithKeypairDeleteUserID(ownerID)` 추가 |

Python:

```python
import openstack

conn = openstack.connect()
console = conn.compute.create_server_remote_console("server-id", type="novnc")
print(console.protocol, console.type, console.url)
conn.compute.delete_keypair("old-login-key")
```

## 독립 실행 Go 예제

인증 설정을 준비하고 `-server-id SERVER_ID -keypair KEYPAIR_NAME`을 전달합니다. 예제는 remote console을 생성한 뒤 지정한 키페어를 삭제합니다. 자신의 키페어 삭제 scope가 기본이고 타 owner는 `-owner-user-id`로 명시합니다. 미존재를 오류로 받으려면 `-strict-missing`을 지정합니다. Compute microversion 2.10은 예제가 명시한 값이며 해당 cloud가 지원하는 버전으로 선택합니다. 문서 검증은 이 main의 컴파일이며 인증·실제 리소스 작업을 실행하는 검증은 별도입니다.

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
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/remoteconsoles"
)

func main() {
    serverID := flag.String("server-id", "", "server ID for the remote console")
    keyName := flag.String("keypair", "", "keypair name to delete")
    ownerID := flag.String("owner-user-id", "", "optional keypair owner")
    strict := flag.Bool("strict-missing", false, "report a missing keypair")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *serverID, *keyName, *ownerID, *strict); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, serverID, keyName, ownerID string, strict bool) error {
    if serverID == "" || keyName == "" {
        return fmt.Errorf("-server-id and -keypair are required")
    }
    conn, err := sdk.Connect(ctx, sdk.WithMicroversion(sdk.Compute, "2.10"))
    if err != nil {
        return err
    }
    service, err := conn.ComputeV2(ctx)
    if err != nil {
        return err
    }
    console, err := service.RemoteConsoles.CreateConsole(ctx, serverID,
        remoteconsoles.WithConsoleCreateType("novnc"))
    if console != nil {
        data, printErr := json.MarshalIndent(map[string]any{
            "resource": console.Resource,
            "wire": console.Wire,
            "status_code": console.StatusCode,
            "header": console.Header,
        }, "", "  ")
        if printErr != nil {
            return printErr
        }
        fmt.Println(string(data))
    }
    if err != nil {
        return fmt.Errorf("create remote console: %w", err)
    }
    options := []keypairs.KeypairDeleteOption{
        keypairs.WithKeypairDeleteIgnoreMissing(!strict),
    }
    if ownerID != "" {
        options = append(options, keypairs.WithKeypairDeleteUserID(ownerID))
    }
    if err := service.KeyPairs.DeleteKeypair(ctx, keyName, options...); err != nil {
        return fmt.Errorf("delete keypair: %w", err)
    }
    fmt.Println("keypair deletion accepted or missing")
    return nil
}
```

하나의 context를 인증과 두 작업에 함께 사용합니다. 뒤 작업이 실패해도 앞서 생성된 console을 되돌리거나 다시 만들지 않습니다. 반환 URL은 수동 데이터이며 SDK가 URL을 열거나 console 접속을 완료하지 않습니다.

## Console type·protocol과 선택 버전

`ConsoleCreateOpts`의 Protocol·Type·URL은 `request.Optional[string]`입니다. 기본값은 모두 생략이며 타입의 필수 enum이나 기본 console type을 합성하지 않습니다. 옵션 없는 호출은 `{"remote_console":{}}`를 전송하고 Nova가 유효성을 판단합니다. 알려지지 않은 비어 있지 않은 protocol도 그대로 전송합니다.

protocol이 생략/null/빈 문자열이고 type이 비어 있지 않으면 아래 protocol을 요청 body에 넣습니다. 알 수 없는 type은 protocol을 JSON null로 추론합니다. 명시한 비어 있지 않은 protocol은 type과 일치하지 않더라도 보존합니다.

| type | 추론 protocol | 선택된 Nova microversion 조건 |
|---|---|---|
| `novnc`, `xvpvnc` | `vnc` | 추가 local type gate 없음 |
| `spice-html5` | `spice` | 추가 local type gate 없음 |
| `rdp-html5` | `rdp` | 추가 local type gate 없음 |
| `serial` | `serial` | 추가 local type gate 없음 |
| `webmks` | `mks` | 2.8 이상 |
| `spice-direct` | `spice` | 2.99 이상 |

두 버전 조건은 명시 protocol이 있어도 적용합니다. Go는 공통 source tuple 비교로 선택된 같은 major2의 microversion을 검사하고 부족하면 HTTP 전에 오류를 반환합니다. `2.latest`는 통과하고 global `latest`는 유한 major2를 증명하지 못합니다. 이 direct 함수는 서버 범위 discovery·최대2.99 선택을 수행하거나 버전을 자동 올리지 않습니다. advertised/selected 비교와 modern/legacy 자동 선택은 별도 [상위 CreateConsole](console-selection.md)이 제공합니다. 실제 API 경로의 지원과 권한은 Nova가 검사합니다.

`WithConsoleCreateOptions`는 typed 속성을 교체하고 개별 Protocol·Type·URL 옵션은 마지막 값을 사용합니다. 각 `...Value` helper로 생략/null/빈 문자열을 구분합니다. `WithConsoleCreateField`는 추가 JSON 값을, `WithConsoleCreateHeader`는 이 작업의 추가 헤더를 받습니다. core protocol/type/url과 고정 부모 server_id는 extension으로 덮어쓰지 않습니다. 입력은 SDK가 복사하고 선택된 client·provider·경로·microversion을 유지합니다.

## ConsoleRecord의 Resource·Wire와 접수 응답

`ConsoleRecord.Resource`는 protocol/type/url과 caller의 고정 server_id를 가진 view입니다. 처음 선택한 protocol/type/url을 seed로 유지하고 응답이 제공한 같은 세 속성으로 덮습니다. 생략한 속성의 view 기본값은 null입니다. 추론한 protocol은 전송 body만 바꿉니다. 예를 들어 type=novnc·protocol 생략으로 vnc를 전송했더라도 응답에 protocol이 없으면 Resource의 protocol은 null입니다. 원래 null·빈 문자열도 응답이 해당 필드를 제공하기 전까지 그대로이고, 응답의 명시 null은 seed를 덮습니다.

`Wire`는 실제 응답 행이고 Resource와 독립된 원문입니다. 추가 응답 속성과 요청 extension의 응답 값은 Wire에 남으며 Resource에 추가하지 않습니다. 실제 응답의 server_id도 Wire에서 확인하고 Resource의 고정 부모를 바꾸지 않습니다. `Envelope`, `Header`, `StatusCode`는 실제 전체 body·헤더·접수 상태를 보존합니다. remote_console key가 없는 object 응답은 source처럼 flat object로 처리합니다. body가 비거나 유효한 UTF-8 JSON으로 해석할 수 없는 accepted 응답이면 seed를 유지한 Resource와 receipt를 반환하고 Wire는 nil입니다. Go는 올바르지 않은 UTF-8을 파싱 불가로 처리하며 Python response의 charset 감지·대체 문자 decoding을 재현하지 않습니다. 해석 가능한 root null/배열 또는 명시 remote_console의 null/배열은 오류입니다. 읽기·Close·context/source 오류는 JSON 변환의 관대한 처리로 숨기지 않습니다. 받은 receipt는 오류와 함께 남지만 이때 Resource와 Wire는 nil일 수 있습니다.

owned 생성은 source처럼 final200–399를 처리합니다. Python Resource와 달리 결과에는 mutable connection·save/delete lifecycle 또는 computed location을 만들지 않습니다. 입력은 nullable string concrete 옵션으로 제한하고 원문 결과·독립 복사·source/context 보호를 Go 정책으로 제공합니다. 이 생성 작업은 waiter나 console 연결 확인을 하지 않습니다.

기존 native `Create(ctx, serverID, CreateOpts, options...)`는 protocol/type 필수 입력·HTTP200·native `RemoteConsole` 반환을 그대로 유지합니다. type 추론·게이트와 seeded raw 결과가 필요하면 새 `CreateConsole`을 사용합니다. 별도 Python `get_server_console_url`의 legacy/new endpoint 선택은 이 함수의 계약에 포함하지 않습니다.

## 키페어 이름·owner·미존재

`DeleteKeypair(ctx, name, options...)`는 `/os-keypairs/{name}`로 바로 DELETE합니다. source의 keypair ID도 이름이며 numeric wire id나 `resource.Name`의 목록 검색을 사용하지 않습니다. Go는 validated 단일 path segment·텍스트 입력을 받고 Python Resource를 전달하는 대신 caller가 이름을 꺼내 전달합니다.

| 옵션 | 기본값과 의미 |
|---|---|
| `WithKeypairDeleteUserID(value)` | 빈 문자열은 query 생략; 그 외는 `user_id`로 전송 |
| `WithKeypairDeleteIgnoreMissing(false)` | 기본 true인 미존재 무시를 strict 오류로 변경 |
| `WithKeypairDeleteOptions(KeypairDeleteOpts{...})` | UserID와 선택적인 IgnoreMissing의 concrete 옵션 교체 |
| `WithKeypairDeleteQuery(key, value)` | 추가 wire query; core user_id/ignore_missing 보호 |
| `WithKeypairDeleteHeader(key, value)` | 작업별 추가 헤더 |

IgnoreMissing의 unset은 기본 true이며 명시 false도 보존합니다. JSON null bool은 오류로 거부합니다. owner를 지정해도 client microversion을 바꾸거나 새 local gate를 추가하지 않습니다. native 옵션은 owner query가 2.10부터 지원된다고 설명하므로 해당 cloud의 API 버전을 선택합니다. 자신의 기본 scope와 다른 owner의 작업 권한은 cloud 정책이 검사하며 403은 그대로 반환합니다.

final200–399는 접수 성공이고 body는 JSON으로 해석하지 않습니다. 기본값은 직접 owned DELETE의 정상404만 무시합니다. `WithKeypairDeleteIgnoreMissing(false)`이면 같은404가 오류입니다. accepted body read·Close에 포함된404나 callback/source/context 실패는 미존재 성공으로 숨기지 않으며 실제 body·헤더·상태·원인을 확인할 수 있습니다. SDK composition은 accepted 처리 오류만으로 DELETE를 다시 전송하지 않고 native provider retry·재인증은 별도로 유지합니다.

반환은 error뿐이며 Python의 None 결과와 대응합니다. nil은 접수 또는 무시한 미존재이고 삭제 완료 대기가 아닙니다. 기존 native `Delete(ctx, name, options...)`의 strict404·202/204, 기존 `Remove(ctx, resource.Ref, options...)`의 별도 lookup/missing 정책은 그대로입니다. 새 helper의 owner와 missing 정책을 다른 native 삭제에 소급하지 않습니다.

## 소스와 검증 범위

기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [create_server_remote_console](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L2903-L2920), [console 변환·버전 조건](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server_remote_console.py#L18-L77), [delete_keypair](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L842-L866)와 [Resource seed·응답 처리](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338-L1404)입니다.

두 named 함수는 위의 concrete 옵션·소유 결과·선택 버전·오류 정책으로 비교합니다. 전체 Resource/session/discovery와 다른 console·keypair·서버 lifecycle은 전체 SDK 목표에서 별도로 추적합니다. 실제 HTTP fixture, exact main 컴파일, 전체 gate와 지원 판정은 [지원 판정대장](../docs/sdk-support-ledger.md)에 기록합니다. Python 예제 실행이나 인증된 OpenStack 권한 검증을 문서 작성만으로 주장하지 않습니다.
