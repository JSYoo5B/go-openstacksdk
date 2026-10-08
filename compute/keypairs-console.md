# Keypair 생성과 legacy console URL: Python과 Go

Connection의 `ComputeV2(ctx)`에서 두 user 작업을 사용합니다. 입력 기본값과 응답 해석은 SDK가 처리하고, 애플리케이션이 builder interface를 구현하지 않습니다.

| 고정 openstacksdk 호출 | Go 호출 | 결과 |
|---|---|---|
| `conn.compute.create_keypair(**attrs)` | `service.KeyPairs.CreateKeypair(ctx, options...)` | 입력과 응답을 병합한 Resource, 실제 Wire와 접수 receipt |
| `conn.compute.get_server_console_url(server, console_type)` | `service.Servers.ConsoleURL(ctx, serverID, consoleType)` | 실제 `console` 값의 `json.RawMessage` |

```python
# conn과 public_key, server_id를 준비한 함수 안에서
created = conn.compute.create_keypair(name="workstation", public_key=public_key)
console = conn.compute.get_server_console_url(server_id, "novnc")
```

아래 Go main은 공개 키를 가져오고, `-server-id`를 지정하면 legacy console URL도 요청합니다. `-console-type`의 novnc 기본값은 이 예제의 CLI 선택이며 SDK 메서드는 type을 필수로 받습니다.

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "os"
    "time"

    sdk "github.com/JSYoo5B/gophercloudsdk"
    "github.com/JSYoo5B/gophercloudsdk/compute/v2/keypairs"
)

func main() {
    name := flag.String("keypair", "", "keypair name to create")
    publicKeyPath := flag.String("public-key-file", "", "SSH public key to import")
    owner := flag.String("owner-user-id", "", "optional owner")
    serverID := flag.String("server-id", "", "optional server ID for a legacy console")
    consoleType := flag.String("console-type", "novnc", "legacy console type")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *name, *publicKeyPath, *owner, *serverID, *consoleType); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, name, publicKeyPath, owner, serverID, consoleType string) error {
    if name == "" || publicKeyPath == "" {
        return fmt.Errorf("-keypair and -public-key-file are required")
    }
    publicKey, err := os.ReadFile(publicKeyPath)
    if err != nil { return err }
    conn, err := sdk.Connect(ctx, sdk.WithMicroversion(sdk.Compute, "2.10"))
    if err != nil { return err }
    service, err := conn.ComputeV2(ctx)
    if err != nil { return err }
    options := []keypairs.KeypairCreateOption{
        keypairs.WithKeypairCreateName(name),
        keypairs.WithKeypairCreatePublicKey(string(publicKey)),
    }
    if owner != "" {
        options = append(options, keypairs.WithKeypairCreateUserID(owner))
    }
    created, err := service.KeyPairs.CreateKeypair(ctx, options...)
    if err != nil { return fmt.Errorf("create keypair: %w", err) }
    data, err := json.MarshalIndent(map[string]any{
        "resource": created.Resource, "wire": created.Wire,
        "status_code": created.StatusCode, "header": created.Header,
    }, "", "  ")
    if err != nil { return err }
    fmt.Println(string(data))
    if serverID != "" {
        console, err := service.Servers.ConsoleURL(ctx, serverID, consoleType)
        if err != nil { return fmt.Errorf("get legacy console: %w", err) }
        fmt.Println(string(console))
    }
    return nil
}
```

인증과 두 작업은 같은 context를 사용합니다. 이 예제는 공개 키를 그대로 전송하며 입력 문자열을 trim하거나 type을 추가하지 않습니다. 콘솔 조회가 실패해도 앞선 keypair를 재생성하거나 삭제하지 않습니다. SDK가 반환 URL에 접속하지는 않습니다.

## CreateKeypair의 입력과 반환

`KeypairCreateOpts{Attributes: map[string]any{...}}`와 `WithKeypairCreateOptions`는 body 속성을 묶어서 설정합니다. `WithKeypairCreateAttribute`는 개별 속성을 선택하고 Name/PublicKey/Type/UserID helper는 문자열 입력의 편의 함수입니다. null이나 문자열 이외의 source 값은 Attribute 또는 bulk map으로 지정합니다. 옵션 없는 호출은 `{"keypair":{}}`를 POST하며 필수 이름·enum·owner 권한은 Nova가 검사합니다. 이름은 body 데이터이므로 member 경로의 ID 검증을 적용하지 않습니다.

| 입력 이름 | 전송 필드 | 반환 Resource |
|---|---|---|
| `created_at`, `fingerprint`, `private_key`, `public_key`, `user_id` | 같은 이름 | 입력 seed 위에 실제 응답 값·null을 덮음 |
| `name`, `id` | `name` | `name`과 논리 `id`가 같은 원문 값 |
| `deleted`, `is_deleted` | `deleted` | `is_deleted`에 bool view 투영; null은 null |
| `type` | supplied 값만 전송 | 최종 값이 생략된 경우만 `ssh`; null·빈 문자열은 유지 |

bulk map과 실제 응답의 `name`/`id`는 Python처럼 name이 우선하며 null도 이 규칙에 포함합니다. `deleted`/`is_deleted`는 Python 입력·응답의 insertion order와 달리 Go에서는 canonical deleted가 우선합니다. 개별 옵션은 canonical 필드의 마지막 값을 사용합니다. Go map에 입력 순서가 없으므로 중복 alias의 선택을 명시한 정책입니다. 요청의 `deleted`는 raw 값을 그대로 보내고 bool 변환은 반환 view에만 적용합니다. 예를 들어 문자열 `"false"`는 true, 빈 배열은 false입니다. 알 수 없는 semantic 속성은 직렬화 전에 버립니다. 명시 vendor JSON은 `WithKeypairCreateField`로 보내며 모든 선언 필드와 alias를 덮어쓸 수 없습니다. 추가 헤더는 `WithKeypairCreateHeader`로 지정합니다.

`KeypairRecord.Resource`는 위 source 속성9개의 독립 view이고 `Wire`는 실제 응답 행입니다. response의 numeric `id`도 name이 없으면 Resource의 name/id로 옮기며, 실제 name이 null이어도 id보다 우선합니다. Wire에는 원래 id/name/deleted·vendor 속성·큰 JSON 수가 그대로 남습니다. 입력이나 기본값을 Wire에 합성하지 않습니다. `Envelope`·`Header`·`StatusCode`는 실제 응답을 보존합니다.

accepted 200..399의 빈 body·비JSON은 seed Resource와 nil Wire로 반환합니다. 유효한 JSON의 잘못된 root 또는 keypair envelope는 오류입니다. read/Close·source/context 오류도 실제 receipt와 함께 반환하며 JSON 변환 실패처럼 무시하지 않습니다. Go는 올바르지 않은 UTF-8을 파싱 불가로 다루며 Python의 charset 감지·대체 문자 decoding을 재현하지 않습니다. native `Create(ctx, CreateOpts, ...)`는 Name 필수·200/201·typed KeyPair 반환을 유지합니다.

## ConsoleURL의 legacy 경로와 반환

type은 아래6개 문자열을 그대로 받습니다. 빈 값·webmks·알 수 없는 값·다른 case는 HTTP 전에 오류입니다.

| type | action |
|---|---|
| `novnc`, `xvpvnc` | `os-getVNCConsole` |
| `spice-html5`, `spice-direct` | `os-getSPICEConsole` |
| `rdp-html5` | `os-getRDPConsole` |
| `serial` | `os-getSerialConsole` |

서버 GET/LIST 없이 `/servers/{id}/action`에 `{"action-name":{"type":"..."}}`를 POST합니다. 선택된 microversion과 token을 유지하고 spice-direct의 별도 local gate·자동 버전 변경·modern 재시도를 넣지 않습니다. API 지원 여부는 Nova가 판단합니다.

200..399의 실제 root object에서 `console` key를 선택합니다. object/빈 object/null/배열/scalar는 모두 원문 JSON으로 반환하고 type/url을 강제로 맞추지 않습니다. Python의 None은 Go에서 raw `null`입니다. key 누락·empty/malformed body·잘못된 root·invalid UTF-8은 receipt를 가진 오류이고, 생성 API의 관대한 비JSON 정책과 다릅니다. Python의 charset 감지·대체 문자 decoding은 재현하지 않습니다. HTTP403/404도 원래 오류를 유지합니다.

Python `create_console`의 modern/legacy 자동 선택은 별도 계약입니다. modern 생성은 [CreateConsole 가이드](user-actions.md)를 참고하세요. 두 named API의 전체 mutable Resource/session과 discovery는 계속 구현할 공통 범위이며, 현재 호출은 명시 ID·선택된 ServiceClient·독립 raw 결과로 제공합니다.
