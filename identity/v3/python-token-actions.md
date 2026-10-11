# Keystone v3 Python token actions

`conn.IdentityV3(ctx).Tokens`는 `ValidateToken`, `CheckToken`, `RevokeToken`을 제공합니다. 기존 generated `Get`, `Validate`, `Revoke`의 Gophercloud 호출과 결과는 유지합니다.

| openstacksdk | Go | 기본 동작 |
|---|---|---|
| `conn.identity.validate_token(token, nocatalog=False, allow_expired=False)` | `Tokens.ValidateToken(ctx, token, options...)` | GET `/auth/tokens`; raw Token descriptor view 반환 |
| `conn.identity.check_token(token, allow_expired=False)` | `Tokens.CheckToken(ctx, token, options...)` | HEAD; 실제 200만 true, 다른 HTTP status는 false |
| `conn.identity.revoke_token(token)` | `Tokens.RevokeToken(ctx, token)` | DELETE; 실제 200..399 허용, 본문을 JSON으로 해석하지 않음 |

`WithValidateTokenNoCatalog(true)`, `WithValidateTokenAllowExpired(true)`, `WithCheckTokenAllowExpired(true)`는 각각 `nocatalog=True`, `allow_expired=True`를 query에 넣습니다. false는 parameter를 생략합니다. 옵션은 순서대로 적용하며 마지막 값이 이깁니다. `WithValidateTokenOpts`와 `WithCheckTokenOpts`는 전체 설정을 교체합니다. nil 옵션·실패한 callback·취소된 context는 HTTP 전에 실패합니다. callback에 전달한 pointer를 보관해 변경해도 적용이 끝난 policy는 바뀌지 않습니다.

세 호출은 제출 token을 `X-Subject-Token`에 넣고 provider의 인증 token을 `X-Auth-Token`에 유지합니다. 제출 token은 route에 들어가지 않으며 ID 조회나 이름 검색을 하지 않습니다. 이 호출의 subject header는 service header보다 우선하고 native retry로 변경할 수 없습니다. 호출별 client snapshot과 공통 guard가 endpoint·provider·서비스·microversion의 변경을 검출합니다. 공유 client는 수정하지 않습니다.

```python
validated = conn.identity.validate_token(token, nocatalog=True)
print(validated.expires_at, validated.subject_token)
valid = conn.identity.check_token(token, allow_expired=True)
# 필요한 경우: conn.identity.revoke_token(token)
```

## 독립 Go main

[설치 안내](../../docs/install.md)대로 모듈을 준비하고 아래를 `main.go`에 저장합니다. `go run . -cloud dev -token TOKEN`으로 조회·확인하고, `-revoke`를 명시한 경우에만 마지막에 폐기합니다.

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
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/tokens"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    token := flag.String("token", "", "subject token")
    allowExpired := flag.Bool("allow-expired", false, "allow expired subject token")
    revoke := flag.Bool("revoke", false, "revoke after validation and check")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    record, err := service.Tokens.ValidateToken(ctx, *token,
        tokens.WithValidateTokenNoCatalog(true),
        tokens.WithValidateTokenAllowExpired(*allowExpired))
    if err != nil { log.Fatal(err) }
    data, err := json.MarshalIndent(record.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
    valid, err := service.Tokens.CheckToken(ctx, *token,
        tokens.WithCheckTokenAllowExpired(*allowExpired))
    if err != nil { log.Fatal(err) }
    fmt.Printf("valid: %t\n", valid)
    if *revoke {
        if err := service.Tokens.RevokeToken(ctx, *token); err != nil { log.Fatal(err) }
    }
}
```

## 결과와 오류 경계

`TokenRecord.Resource`는 declared Token view이고 `Wire`는 해석한 token 객체의 모든 원래 JSON 필드입니다. `Envelope`는 전체 HTTP 본문, `Header`·`StatusCode`는 실제 응답입니다. 네 채널은 독립 복사합니다. 알려지지 않은 필드·큰 정수·날짜 원문은 Wire에 보존하며, expires_at/issued_at를 RFC3339나 `time.Time`으로 강제하지 않습니다. inherited id/name은 body에서 가져오며 subject token으로 id를 채우지 않습니다. `SubjectToken`은 응답 header에서만 가져오므로 응답에 없으면 nil입니다. Source가 `Token()`을 연결 없이 만드는 것처럼 location은 null입니다.

Source descriptor와 같이 audit_ids/methods는 scalar를 한 원소 list로, catalog/roles는 list 원소를 dict로, user/project/domain/system은 non-dict를 빈 dict로 변환합니다. null은 null로 유지하고 is_domain은 JSON truthiness를 적용합니다. 누락된 declared 값은 null입니다. 빈 본문·잘못된 JSON syntax는 Source의 ValueError 처리처럼 빈 view로 허용합니다. 유효한 nonobject JSON이나 `token` 안의 nonobject는 실제 accepted 응답을 담은 오류입니다. 원래 view를 수정해 다음 요청 대상을 바꿀 수 없습니다.

Validate/Revoke는 실제 200..399를 허용하고 400 이상은 원래 status·body·header를 유지하는 오류입니다. Check는 204·403·404·500도 false,nil을 반환합니다. 세 호출 모두 transport·read·close·취소 오류를 숨기지 않으며, `resource.OperationError`가 연산 이름과 원래 cause를 보존합니다. accepted response 처리 실패 때 `resource.ResponseError`에 실제 응답 증거가 남고 Validate는 부분 `TokenRecord`도 반환합니다.

공통 fixed-request 정책에 따라 redirect나 retry로 method·URL·선택 subject를 바꾸면 오류이며, Python의 임의 adapter/cache/subclass/동일 Resource 객체 재사용은 제공하지 않습니다. Go 문자열 header와 JSON은 유효한 UTF-8 범위를 사용합니다. 구현 검증은 [token action 계약 테스트](tokens/token_actions_test.go)에서 기존 Gophercloud testhelper 기반 fixture를 재사용합니다. 실제 인증된 Keystone 배포 동작은 로컬 계약 테스트와 별개입니다.
