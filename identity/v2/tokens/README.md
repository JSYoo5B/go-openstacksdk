# Keystone v2 인증 응답

`Create`와 `Get`은 `(*Authentication, error)`를 반환합니다. 호출자가 Gophercloud의 여러 `Extract...` 메서드를 조합할 필요 없이 토큰, 사용자, 서비스 catalog를 함께 읽습니다.

```go
// ctx context.Context, service *identityv2.Service를 사용하는 오류 반환 함수 안에서
auth, err := service.Tokens.Get(ctx, tokenID)
if err != nil { return err }
fmt.Println(auth.Token.ID, auth.Token.ExpiresAt, auth.User.Name)
for _, entry := range auth.Catalog.Entries {
    fmt.Println(entry.Type, entry.Endpoints)
}
requestID := auth.Header.Get("X-Openstack-Request-Id")
_ = requestID
```

`identityv2`는 `github.com/JSYoo5B/gophercloudsdk/identity/v2`, `fmt`는 표준 라이브러리입니다. Connection의 `IdentityV2(ctx)`가 이 서비스 객체를 제공합니다. 일반적인 최초 인증은 프로젝트의 `Connect(ctx, ...)`를 사용합니다.

`Authentication.Body`는 추가 응답 필드도 포함하는 JSON snapshot입니다. 원래 HTTP 본문의 공백이나 필드 순서를 보존하지 않습니다. `Header`는 복사된 HTTP 응답 헤더입니다. 사용자나 catalog가 없는 unscoped 응답은 해당 필드가 비어 있습니다. 날짜나 모델 필드가 잘못되면 결과 대신 오류를 반환하고, HTTP 오류는 `errors.As`로 원래 Gophercloud 오류를 확인할 수 있습니다.

openstacksdk에서는 Connection의 인증 세션을 통해 토큰과 catalog에 접근합니다. 이 Go API는 고정한 Gophercloud의 명시적인 Keystone v2 token 생성·조회 응답을 다룹니다. 응답 객체를 수정해도 인증 세션이나 원격 토큰은 변경되지 않습니다.
