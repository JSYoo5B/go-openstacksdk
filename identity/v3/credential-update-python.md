# 일반 Keystone Credential 업데이트

Pinned openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 `conn.identity.update_credential`은 일반 `/credentials/{id}` 리소스에 PATCH를 보냅니다. EC2 전용 `/users/{user_id}/credentials/OS-EC2` 경로와 별개입니다. 기존 [Credential 생성·조회·목록·검색·삭제](credentials-python.md)와 같은 `CredentialRecord` 응답을 사용합니다.

| Python | Go |
|---|---|
| `conn.identity.update_credential(id, blob=blob)` | `api.UpdateRecord(ctx, id, credentials.WithRecordUpdateField("blob", blob))` |
| `conn.identity.update_credential(id, project_id=None)` | `api.UpdateRecord(ctx, id, credentials.WithRecordUpdateField("project_id", nil))` |
| `conn.identity.update_credential(id)` | `api.UpdateRecord(ctx, id)`; HTTP 없이 ID-only view 반환 |

```python
import openstack

conn = openstack.connect(cloud="dev")
updated = conn.identity.update_credential(
    credential_id,
    blob='{"access":"example","secret":"replace-me"}',
)
print(updated.id, updated.blob)
```

## 독립 Go main

[설치 안내](../../docs/install.md)에 따라 모듈을 준비하고 아래 코드를 `main.go`로 저장합니다. `go run . -cloud dev -id CREDENTIAL_ID -blob '{"access":"example","secret":"replace-me"}'`로 일반 Credential의 blob을 변경합니다. `-clear-project`를 지정하면 project_id를 명시적 null로 전송합니다. 실제 Keystone 정책과 Credential type별 검증은 서버가 수행합니다.

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
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    id := flag.String("id", "", "generic Credential ID")
    blob := flag.String("blob", "", "replacement credential blob string")
    clearProject := flag.Bool("clear-project", false, "send project_id=null")
    flag.Parse()
    if *id == "" { log.Fatal("-id is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    options := []credentials.RecordUpdateOption{
        credentials.WithRecordUpdateField("blob", *blob),
    }
    if *clearProject {
        options = append(options, credentials.WithRecordUpdateField("project_id", nil))
    }
    updated, err := service.Credentials.UpdateRecord(ctx, *id, options...)
    if err != nil { log.Fatal(err) }
    data, err := json.MarshalIndent(updated.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

## 속성·기본 동작·응답

`WithRecordUpdateField`와 `WithRecordUpdateOpts`는 Source에 선언된 `name/blob/project_id/type/user_id`만 `credential` envelope에 넣습니다. 이 Resource에는 별도 Body alias나 Header descriptor가 없습니다. 생략한 값은 본문에서 빠지고, 명시한 null·false·0·빈 문자열·빈 array·빈 object는 JSON 타입 그대로 보존합니다. 기존 native `UpdateOpts`가 string만 받거나 빈 값을 생략하는 동작에 의존하지 않습니다. 필드별 required 검사나 blob string 변환을 추가하지 않습니다.

Source는 미등록 attrs를 버리므로 이 프로필도 unknown attrs를 버립니다. 일반 Gophercloud extension body를 의도적으로 보낼 때는 기존 native `Update`와 `WithUpdateField`를 사용할 수 있습니다. `id`는 PATCH body에 들어가지 않으며 Go 프로필의 literal member target을 바꾸지 않습니다. Source `commit`도 id를 dirty set에서 제거합니다. Go 옵션으로 id를 전달하면 무시하며, Python의 literal ID 인자와 중복 id kwarg에서 발생하는 TypeError까지 재현하지 않습니다.

변경할 선언된 Body 속성이 없으면 HTTP를 호출하지 않습니다. 결과의 `Resource`에는 literal id와 나머지 null descriptor가 있고, 실제 요청이 없으므로 `StatusCode=0`, 빈 `Envelope/Header`, nil `Wire`를 유지합니다. `WithRecordUpdateHeaders`는 명시적 Go 요청 정책이며 단독으로 Resource를 dirty 처리하지 않습니다. 빈 변경에서도 유효한 context·client·literal ID·header 정책을 검증합니다.

변경이 있으면 사전 GET 없이 고정 `/credentials/{id}` 경로에 PATCH를 한 번 준비합니다. 응답의 선언된 속성은 requested seed를 덮어쓰고 응답에서 생략한 속성은 요청 값과 literal ID를 유지합니다. singular `credential` 객체가 있으면 unpack하고 없으면 root 객체를 해석합니다. 빈 본문이나 JSON syntax 오류는 pinned Source처럼 seed view를 유지합니다. 유효한 nonobject JSON 또는 null envelope는 실제 accepted 응답 증거를 보존하는 오류입니다.

`Resource`는 선언된 7개 속성(`id/name/blob/project_id/type/user_id/location`)의 view이고 누락 descriptor와 location은 null입니다. `Wire`는 unknown response 속성과 큰 정수를 포함한 원래 객체, `Envelope`는 원래 HTTP 본문, `Header/StatusCode`는 실제 응답입니다. view·wire·envelope·header는 독립 복사합니다. 200..399 응답을 허용하며 204 응답에서도 합성 HTTP 본문을 만들지 않습니다.

## 소유권·오류·검증 경계

옵션은 map·raw JSON·header 값을 복사합니다. 전체 정책 옵션 이후의 필드 옵션은 해당 값을 덮어씁니다. callback에 전달된 pointer를 보관하고 나중에 변경해도 적용이 완료된 정책은 바뀌지 않습니다. nil option·알려진 필드의 잘못된 UTF-8/JSON·보호된 auth header·유효하지 않은 ID는 HTTP 전에 실패합니다. literal ID는 UTF-8 문자열을 한 URL segment로 한 번 escape합니다.

공통 Source capture·고정 요청 REST engine·rejected response observer를 재사용합니다. accepted 응답의 Read/Close 오류는 `resource.ResponseError`에 실제 status/body/header와 원인이 남습니다. rejected 응답은 native `gophercloud.ErrUnexpectedResponseCode`의 status/body/header를 유지하고 Read/Close·취소 cause·Source 변경·retry callback 오류를 함께 보존합니다. 404를 무시하거나 다른 조회로 fallback하지 않습니다. Read에서 변한 Source가 Close에서 원복되어도 이미 관측한 실패는 유지합니다. 기존 native retry hook은 정상 동작을 유지하지만 SDK가 소유한 PATCH method/URL/JSON을 바꾼 retry는 다음 물리 요청 전에 실패합니다.

이 API는 literal-ID call과 요청/응답 Resource view를 제공하며 Python의 mutable Resource 객체·Munch 인자, same-instance dirty 비교, subclass/cache/session hooks, Connection 기반 location, 임의 adapter/microversion을 구현했다고 주장하지 않습니다. 기존 native `Update`와 이 explicit Source 프로필은 각각의 데이터 표현을 유지합니다.

[집중 계약 테스트](credentials/python_update_test.go)는 기존 credential fixture, Gophercloud public testhelper를 사용하는 `testcloud`, fault body와 transport를 재사용합니다. omitted/null/false/zero, ID-only no-op, response overlay와 복사, 실제 accepted/rejected Read/Close/cancel/Source 오류, retry ownership 및 escaped member를 검증합니다. 실제 인증된 Keystone 배포 테스트는 로컬 계약 테스트와 별도입니다.
