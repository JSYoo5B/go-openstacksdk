# Port의 Python 속성 생성·조건부 수정

OpenStackSDK `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 `create_port`와
`update_port`에 대응하는 호출은 `API.CreateWithAttributes`와
`API.UpdateWithAttributes`입니다. 기존 typed native `Create`·`Update`는 유지됩니다.

| 호출 | HTTP | 본문·반환 |
|---|---|---|
| `CreateWithAttributes(ctx, options...)` | POST `ports` | `{"port": {지정한 속성}}`·native Port 모델 |
| 속성 없는 `CreateWithAttributes(ctx)` | POST `ports` | `{"port": {}}`·서버가 필수값 판단 |
| `UpdateWithAttributes(ctx, id)` | 요청 없음 | ID만 담긴 새 native 모델 |
| 속성 또는 조건이 있는 `UpdateWithAttributes(ctx, id, options...)` | PUT `ports/{id}` | 변경 Body만 감싼 envelope·native 모델 |

`WithAttribute`는 옵션 생성 시 JSON을 복사해 미지정·null·false·0·빈 문자열·빈 목록을
구분합니다. 반복된 wire key에는 뒤의 옵션이 적용됩니다. canonical Body 이름은 기존
pinned Source MRO 표를 재사용합니다: `binding_host_id -> binding:host_id`·`is_admin_state_up -> admin_state_up`·`security_group_ids -> security_groups`·`is_port_security_enabled -> port_security_enabled`.

Port 생성의 native NetworkID 필수 검사를 거치지 않습니다. `security_group_ids=[]`와 nil 목록은 각각 `[]`와 `null`이고, ValueSpecs flatten 같은 native 전용 변환은 이 속성 API에서 수행하지 않습니다.

알 수 없는 extension은 Body에 유지하며 query-only 이름도 생성·수정의 속성으로 지정하면
본문에 남습니다. `base_path`, `microversion`, `connection`, `_synchronized`,
`resource_type`, `__conflicting_attrs`, `location`은 이 Body API에서 거부합니다. 수정 ID는
메서드 인자로 전달하고 `WithAttribute("id", ...)`·`WithAttribute("if_revision", ...)`
수정 옵션은 거부합니다. revision은 `WithRevision`을 사용합니다.

## 조건과 Header

`WithRevision(int64)`은 Source의 update `if_revision` 인자에 대응합니다. 미지정이면
조건을 추가하지 않고 `WithRevision(0)`은 `If-Match: revision_number=0`을 보냅니다.
조건만 지정해도 Source Header dirty처럼 `{"port": {}}` PUT을 보냅니다.

`WithIfMatch("a", "b")`와 `WithAttribute("if_match", []string{"a", "b"})`는
`If-Match: a, b`를 보냅니다. Header wire 이름 `If-Match`도 인식합니다. 빈 목록은 빈
Header를 보내고 `WithAttribute("if_match", nil)`은 Source의 문자열 `None`을 보냅니다.
Header는 Body에 들어가지 않습니다. string·string list·null을 지원하며 임의 Python 객체의
`str()` 변환은 제공하지 않습니다.

`WithRevision`이 있으면 Source `_update`처럼 raw If-Match보다 우선합니다. 두 옵션의
상대 순서에 영향받지 않고, revision끼리는 마지막 값이 적용됩니다. Source처럼 revision으로 버려지는 raw Header는
JSON 타입·CRLF 검증 대상이 되지 않습니다. 선택된 최종 Header만 검증합니다. 조건이 있으면
configured client의 대소문자가 다른 If-Match key를 복사본에서 제거한 뒤 요청하며 원본
client는 바꾸지 않습니다. 조건 미지정 시 configured Header를 유지합니다. `WithRevision`은
생성 호출에서 거부하며, Source create의 별도 Header에는 `WithIfMatch`를 사용할 수 있습니다.

```python
conn.network.create_port(network_id=network_id, is_admin_state_up=False, security_group_ids=[])
conn.network.update_port(port_id, name="", binding_host_id="", if_revision=0)
conn.network.update_port(resource_id)  # 변경·조건 없으면 HTTP 없음
conn.network.update_port(resource_id, if_revision=0)  # 빈 Body + 조건 Header PUT
```

## Connection main

`NEUTRON_UPDATE_ID`가 있으면 수정합니다. `NEUTRON_REVISION`은 미지정 또는 실제 서버의
revision 숫자로 설정합니다. `NEUTRON_CREATE=1`이면 생성합니다. 환경 변수가 없으면
사용법만 출력합니다. 동일 API를 `ports.New(client)`에서도 사용합니다.

```go
package main

import (
    "context"
    "fmt"
    "os"
    "strconv"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(os.Getenv("OS_CLOUD")))
    if err != nil { panic(err) }
    service, err := conn.Network(ctx)
    if err != nil { panic(err) }
    api := service.API.Ports
    if id := os.Getenv("NEUTRON_UPDATE_ID"); id != "" {
        options := []ports.AttributeOption{ports.WithAttribute("name", ""), ports.WithAttribute("binding_host_id", "")}
        if value := os.Getenv("NEUTRON_REVISION"); value != "" {
            revision, err := strconv.ParseInt(value, 10, 64)
            if err != nil { panic(err) }
            options = append(options, ports.WithRevision(revision))
        }
        value, err := api.UpdateWithAttributes(ctx, id, options...)
        if err != nil { panic(err) }
        fmt.Printf("%#v\n", value)
        return
    }
    if os.Getenv("NEUTRON_CREATE") == "1" {
        value, err := api.CreateWithAttributes(ctx, ports.WithAttribute("network_id", os.Getenv("NETWORK_ID")), ports.WithAttribute("is_admin_state_up", false), ports.WithAttribute("security_group_ids", []string{}))
        if err != nil { panic(err) }
        fmt.Printf("%#v\n", value)
        return
    }
    fmt.Println("set NEUTRON_UPDATE_ID (optionally NEUTRON_REVISION) or NEUTRON_CREATE=1")
}
```

새 호출의 성공 status 범위는 HTTP 200..399입니다. 모델은 native decoder를 유지하며,
204는 HTTP 본문을 decode하지 않아 ID 없는 zero-model 포인터와 nil 오류를 반환합니다.
이를 요청 ID·속성을 가진 Resource로 합성하지 않습니다. 200 등 JSON decode 경로에서 빈/해석 불가 본문은 오류로
반환합니다. 오류는 `resource.OperationError`와 원래 HTTP status/body/header·decode 오류·
`context.Canceled`/custom `context.Cause`를 보존합니다.

전체 Python Resource 인스턴스 입력·같은 객체 반환·기존 상태의 dirty/cache 비교·응답에
없는 요청값과 unknown 응답값 저장, 동적 microversion·computed metadata·adapter/session
정책과 Python 예외 동등성은 별도로 남습니다. 빈 수정의 ID-only 반환 외에는 native 모델에
요청 ID나 속성을 합성하지 않습니다. 기존 native 옵션의 required/default 변환은 새 속성
프로필에 적용하지 않습니다.

`python_mutations_contracts_test.go`는 기존 Gophercloud public testhelper 기반
`internal/testcloud`와 native recorder를 재사용합니다. 기본·별칭·null/빈 값,
조건만 있는 수정·revision 우선순위·client Header 분리·snapshot·잘못된 옵션,
204 native 경계·HTTP/decode 오류·취소를 검증합니다. 실제 OpenStack 실행은 별도 환경에서
검증해야 합니다.
