# SecurityGroupRule의 Python 속성 단일·일괄 생성

OpenStackSDK `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의
`create_security_group_rule`은 `API.CreateWithAttributes`,
`create_security_group_rules`는 `API.CreateBulkWithAttributes`로 호출합니다.
기존 typed `Create`·`CreateBulk`의 ABI와 native decoder는 유지됩니다.

| 호출 | HTTP | 본문 |
|---|---|---|
| `CreateWithAttributes(ctx, options...)` | POST `security-group-rules` | `{"security_group_rule": {지정한 속성}}` |
| 속성 없는 `CreateWithAttributes(ctx)` | 같은 POST | `{"security_group_rule": {}}` |
| `CreateBulkWithAttributes(ctx, row1, row2, ...)` | 같은 POST 한 번 | `{"security_group_rules": [{row1}, {row2}, ...]}` |

`WithAttribute`는 옵션 생성 시 JSON을 복사합니다. native Direction·EtherType·SecGroupID
필수 검사를 거치지 않아, 미지정 값은 생략하고 null·0·false·빈 문자열·빈 목록은 보존합니다.
서버가 실제 유효성을 판단합니다. canonical `ether_type`은 `ethertype`으로 변환하고 나머지
Body 이름은 pinned Source MRO 표를 재사용합니다. unknown extension과 query-only `fields`
같은 이름도 속성으로 전달하면 Body에 남습니다. 반복된 wire key에는 뒤의 옵션이 적용됩니다.

## 단일과 일괄 요청의 Source 차이

단일 생성은 `SecurityGroupRule._prepare_request`처럼 **falsy**
`remote_address_group_id`를 제거합니다: null·false·숫자 0·빈 문자열·빈 목록·빈 객체.
문자열 `"0"`·`"False"`와 nonempty 목록·객체는 유지합니다. 숫자의 truth는 float 반올림 없이
판단하므로 JSON number `1e-999`처럼 nonzero 값도 보존합니다.

일괄 생성은 row별 `_prepare_request`가 singular envelope 없이 실행되므로 위 제거 조건이
실행되지 않습니다. 따라서 일괄 row의 빈/null `remote_address_group_id`는 Body에 유지합니다.
입력 row는 `[]rules.AttributeOption`이며, 전체 row 옵션 목록을 callback 실행 전에 복사합니다.
각 값은 `WithAttribute`에서 이미 snapshot됩니다. 빈 전체 목록은 Source처럼 요청 전 오류이고,
빈 row 하나는 `{}` row로 전송할 수 있습니다. 모든 row를 준비한 뒤 한 번 POST하며 순서를 유지합니다.

Source `Resource.bulk_create`는 **마지막 row의 Header**를 사용합니다. 이 호출도 앞 row의
Header를 합치지 않습니다. 앞 row의 잘못된 Header는 준비 중 오류가 나고 요청하지 않습니다.
마지막 row에 Header가 없으면 configured client Header를 유지합니다.

`WithIfMatch("a", "b")` 또는 `WithAttribute("if_match", []string{"a", "b"})`는
`If-Match: a, b`입니다. string·string list·null을 지원하며 null은 Source 문자열 `None`,
빈 목록은 빈 Header가 됩니다. Header는 Body에 넣지 않습니다. 임의 Python 객체의 `str()`
변환은 제공하지 않고 null 목록 요소·숫자 목록 요소·CRLF Header는 요청 전에 거부합니다.
선택한 Header는 대소문자가 다른 configured If-Match key를 client 복사본에서 제거한 뒤
보내며 원본 client를 수정하지 않습니다.

`base_path`, `microversion`, `connection`, `_synchronized`, `resource_type`,
`__conflicting_attrs`, `location` 제어 속성 및 generic query/임의 Header option은 거부합니다.
Source SecurityGroupRule은 수정 capability가 없고 Proxy update 선언도 없으므로 별도
`UpdateWithAttributes`나 `WithRevision`은 제공하지 않습니다.

```python
conn.network.create_security_group_rule(
    security_group_id=group_id, direction="ingress", ether_type="IPv4",
    protocol=None, port_range_min=0, remote_address_group_id="",
)  # remote_address_group_id 제거

conn.network.create_security_group_rules([
    dict(security_group_id=group_id, direction="ingress", ether_type="IPv4", remote_address_group_id=""),
    dict(security_group_id=group_id, direction="egress", ether_type="IPv6", remote_address_group_id=None),
])  # 두 remote_address_group_id 값 유지
```

## Connection main

`NEUTRON_CREATE=1`이면 `SECURITY_GROUP_ID`에 단일 rule을 생성합니다.
`NEUTRON_BULK=1`도 지정하면 row 두 개를 한 번의 요청으로 생성합니다. 환경 변수가 없으면
사용법만 출력합니다. 같은 API를 `rules.New(client)`에서도 사용할 수 있습니다.

```go
package main

import (
    "context"
    "fmt"
    "os"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(os.Getenv("OS_CLOUD")))
    if err != nil { panic(err) }
    service, err := conn.Network(ctx)
    if err != nil { panic(err) }
    api := service.API.SecurityRules
    if os.Getenv("NEUTRON_CREATE") != "1" {
        fmt.Println("set NEUTRON_CREATE=1 and SECURITY_GROUP_ID (optionally NEUTRON_BULK=1)")
        return
    }
    first := []rules.AttributeOption{
        rules.WithAttribute("security_group_id", os.Getenv("SECURITY_GROUP_ID")),
        rules.WithAttribute("direction", "ingress"), rules.WithAttribute("ether_type", "IPv4"),
        rules.WithAttribute("protocol", nil), rules.WithAttribute("remote_address_group_id", ""),
    }
    if os.Getenv("NEUTRON_BULK") == "1" {
        second := []rules.AttributeOption{
            rules.WithAttribute("security_group_id", os.Getenv("SECURITY_GROUP_ID")),
            rules.WithAttribute("direction", "egress"), rules.WithAttribute("ether_type", "IPv6"),
            rules.WithAttribute("remote_address_group_id", nil),
        }
        values, err := api.CreateBulkWithAttributes(ctx, first, second)
        if err != nil { panic(err) }
        fmt.Printf("%#v\n", values)
        return
    }
    value, err := api.CreateWithAttributes(ctx, first...)
    if err != nil { panic(err) }
    fmt.Printf("%#v\n", value)
}
```

성공 HTTP 범위는 SDK profile 200..399입니다. 단일 결과는 native `*SecGroupRule`, 일괄
결과는 native `[]SecGroupRule`을 즉시 반환합니다. Source는 POST 후 Resource generator를
반환하지만 Go는 native 전체 slice decoder를 사용합니다. 204는 HTTP Body를 decode하지
않아 단일·일괄 모두 nil 결과와 nil 오류를 반환합니다. Source bulk의 unconditional
`response.json()`은 bodyless 204에서 오류가 나는 차이가 남습니다. 이를 요청 ID·속성을 가진
Resource로 합성하지 않습니다. JSON 경로의 빈 Body·잘못된 타입은 native decode 오류,
400 이상은 원래 HTTP status/body/header 오류로 반환하고 custom context Cause도 보존합니다.

전체 mutable Resource/Munch 입력·객체 identity·dirty/cache·unknown 응답 필드·요청값 fallback,
동적 microversion·computed metadata·Python descriptor 접근/변환·adapter/session/예외 동등성은
별도 범위입니다. 생성 결과의 ID나 attrs를 요청에서 합성하지 않습니다.

`python_mutations_contracts_test.go`는 기존 Gophercloud public testhelper 기반
`internal/testcloud`와 native recorder를 재사용합니다. 단일 omission·zero/null·alias와 falsy
변환, 일괄 Body/순서·마지막 Header·전체 row snapshot·빈 row와 잘못된 row, HTTP/decode/취소
경계를 검증합니다. 실제 OpenStack 실행은 별도 환경에서 검증해야 합니다.
