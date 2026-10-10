# Keystone v3 native OAuth1·EC2 token 호출

`service.Oauth1`(`identity/v3/oauth1`)과 `service.EC2Tokens`(`identity/v3/ec2tokens`)의 generated 메서드는 Gophercloud `v2.15.0`의 [oauth1](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/oauth1/requests.go)과 [ec2tokens](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/ec2tokens/requests.go) 요청을 바꾸지 않고 호출합니다. SDK는 단건 호출의 오류에 `resource.OperationError` 문맥을 더하고 문서화한 확장 필드·header를 병합할 뿐이며, 목록 stream의 오류는 감싸지 않습니다. 목록은 Keystone `links.next` 문자열을 따라갑니다.

## OAuth1 consumer

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `CreateConsumer(ctx, opts, options...)` | `POST OS-OAUTH1/consumers`, `{"consumer": {"description": ...}}` | 201 |
| `GetConsumer(ctx, id)` | `GET OS-OAUTH1/consumers/{id}` | 200 |
| `UpdateConsumer(ctx, id, opts, options...)` | `PATCH OS-OAUTH1/consumers/{id}`, `{"consumer": {...}}` | 200 |
| `DeleteConsumer(ctx, id)` | `DELETE OS-OAUTH1/consumers/{id}` | 202, 204 |
| `ListConsumers(ctx)` | `GET OS-OAUTH1/consumers` | native pager |

consumer 관리는 기본 정책상 관리자 호출입니다. `Description`에는 omitempty가 없어서 빈 문자열도 항상 보냅니다. `WithCreateConsumerField`·`WithUpdateConsumerField`의 확장 필드는 `consumer` 객체 안에 들어가고, `description`과 겹치면 HTTP 전에 오류입니다. 응답 `Consumer`는 `id`·`secret`·`description`을 그대로 읽으며 `secret`은 생성 응답에만 들어 있습니다.

## OAuth1 위임 흐름

| 메서드 | 요청 | 결과 | 성공 status |
|---|---|---|---|
| `RequestToken(ctx, opts, options...)` | `POST OS-OAUTH1/request_token`, 본문 없음, OAuth `Authorization` | `*Token` | 201 |
| `AuthorizeToken(ctx, id, opts, options...)` | `PUT OS-OAUTH1/authorize/{id}`, `{"roles": [...]}` | `*AuthorizedToken` | 200 |
| `CreateAccessToken(ctx, opts, options...)` | `POST OS-OAUTH1/access_token`, 본문 없음, OAuth `Authorization` | `*Token` | 201 |
| `Create(ctx, opts, options...)` | `POST auth/tokens`, OAuth `Authorization` | `*tokens.Token` | 201 |

`RequestToken`·`CreateAccessToken`·`Create`는 `Authorization: OAuth ...` header로 서명합니다. header에는 `oauth_consumer_key`, `oauth_signature_method`, `oauth_timestamp`, `oauth_nonce`, `oauth_version="1.0"`이 이름순으로 들어가고, 마지막에 `oauth_signature`가 붙습니다. `RequestToken`은 `oauth_callback="oob"`를 더하고 `CreateAccessToken`은 `oauth_token`·`oauth_verifier`를, `Create`는 `oauth_token`을 더합니다. 서명 대상은 method, 전체 요청 URL, 정렬한 oauth 인자이며 `HMAC-SHA1`은 consumer secret과 token secret으로 만든 key의 HMAC 값을, `PLAINTEXT`는 그 key 자체를 씁니다. `OAuthTimestamp`와 `OAuthNonce`를 비우면 현재 시각과 난수로 채우므로 매 요청의 서명이 달라집니다.

`OAuthConsumerKey`·`OAuthSignatureMethod`와 각 호출의 `OAuthToken`·`OAuthVerifier`는 비어 있으면 HTTP 전에 오류입니다. 반면 `OAuthConsumerSecret`·`OAuthTokenSecret`은 필수 표시가 있어도 검사하지 않아서 빈 secret으로 서명합니다. `RequestTokenOpts.RequestedProjectID`는 `Requested-Project-Id` header로 보냅니다. `With...Header`의 확장 header는 `Authorization`이나 `Requested-Project-Id`와 겹치면 대소문자와 상관없이 거부합니다.

`RequestToken`과 `CreateAccessToken`의 응답은 JSON이 아니라 `application/x-www-form-urlencoded` 본문입니다. `Content-Type`이 이 값과 정확히 같지 않으면 charset 인자가 붙은 경우까지 포함해 오류가 나고, `oauth_token`·`oauth_token_secret`·`oauth_expires_at`을 읽습니다. `oauth_expires_at`은 `2006-01-02T15:04:05.999999Z` 형식만 받습니다. 두 요청 모두 OAuth 서명과 별개로 provider token이 `X-Auth-Token`으로 함께 전송됩니다.

`AuthorizeToken`은 사용자 token으로 호출하며 본문에 envelope이 없습니다. 그래서 `WithAuthorizeTokenField`의 확장 필드는 `roles` 옆 최상위에 들어갑니다. `Roles`에 omitempty가 없어 빈 옵션은 `{"roles":null}`을 보내고, ID와 이름이 모두 빈 role이 있으면 HTTP 전에 오류입니다. 응답의 `token.oauth_verifier`를 `AuthorizedToken`으로 돌려줍니다.

`Create`는 `{"auth": {"identity": {"methods": ["oauth1"], "oauth1": {}}}}`를 보내고, `WithCreateField`의 확장 필드는 `identity` 안이 아니라 `auth` 객체에 들어갑니다. scope 객체는 만들지 않습니다. 결과 `tokens.Token`의 ID는 응답 `X-Subject-Token` header에서, 만료 시각은 `token.expires_at`에서 읽습니다. 응답의 `OS-OAUTH1` 확장(`TokenExt`)은 반환값에 포함되지 않습니다. native 구현은 빈 `X-Auth-Token`을 요청하지만 Gophercloud가 그 뒤에 provider token을 다시 붙이므로, 인증된 client에서는 현재 token이 함께 전송되고 token이 없는 client에서는 빈 header가 나갑니다. 이 호출은 발급한 token을 ProviderClient에 저장하지 않고 재인증 함수도 바꾸지 않습니다. `WithCreateHeader`로 `Authorization`·`X-Auth-Token`을 덮어쓸 수 없습니다.

## OAuth1 access token

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `GetAccessToken(ctx, userID, id)` | `GET users/{user}/OS-OAUTH1/access_tokens/{id}` | 200 |
| `RevokeAccessToken(ctx, userID, id)` | `DELETE users/{user}/OS-OAUTH1/access_tokens/{id}` | 202, 204 |
| `ListAccessTokens(ctx, userID)` | `GET users/{user}/OS-OAUTH1/access_tokens` | native pager |
| `ListAccessTokenRoles(ctx, userID, id)` | `GET users/{user}/OS-OAUTH1/access_tokens/{id}/roles` | native pager |
| `GetAccessTokenRole(ctx, userID, id, roleID)` | `GET users/{user}/OS-OAUTH1/access_tokens/{id}/roles/{role}` | 200 |

access token 조회와 폐기는 기본 정책상 token을 승인한 사용자 본인이나 관리자가 호출합니다. `AccessToken.ExpiresAt`은 `2006-01-02T15:04:05.999999Z` 형식으로 읽고 값이 없으면 nil입니다. role 목록은 `roles` 배열, 단건 role은 `role` 객체에서 `id`·`name`·`domain_id`를 읽습니다.

세 목록은 `links.next`가 비거나 null이면 멈추고 빈 배열은 결과 없이 끝납니다. 본문 없는 204 응답은 Gophercloud가 JSON을 읽다가 `io.EOF` 오류를 내며, 404는 200·204·300을 기대한 `ErrUnexpectedResponseCode`입니다.

## EC2·S3 token

| 메서드 | 요청 | 결과 | 성공 status |
|---|---|---|---|
| `EC2Tokens.Create(ctx, opts, options...)` | `POST ec2tokens`, `{"credentials": {...}}` | `*tokens.Token` | 200 |
| `EC2Tokens.ValidateS3Token(ctx, opts, options...)` | `POST s3tokens`, `{"credentials": {...}}` | `*tokens.Token` | 200 |

두 호출 모두 `*AuthOptions`를 받으며 nil이거나 `Access`가 비어 있으면 HTTP 전에 오류입니다. `Signature`를 주면 서명을 계산하지 않고 omitempty 없는 `host`·`path`·`verb`·`headers`·`params`·`body_hash`를 빈 값이나 null 그대로 보냅니다. `[]byte` 값인 `Signature`와 `Token`은 base64 문자열로 바뀝니다.

`Signature`가 없으면 `params.SignatureVersion`이 `"2"`일 때 AWS 서명 V2를 씁니다. 이때 `SignatureMethod`는 `HmacSHA1`이나 `HmacSHA256`이어야 하고, verb·host·path·정렬한 params를 secret으로 서명하며 `body_hash`와 `headers`는 본문에서 뺍니다. 다른 버전이나 빠진·지원하지 않는 method는 HTTP 전에 오류입니다. 버전 지정이 없으면 AWS 서명 V4를 쓰며 `headers.X-Amz-SignedHeaders`로 고른 header와 `Region`·`Service`·`Timestamp`·`BodyHash`로 서명합니다. 계산한 `X-Amz-Date`와 `Authorization`은 본문의 `headers`에 들어가고 호출자가 넘긴 map은 바뀌지 않습니다. `Timestamp`와 `BodyHash`를 비우면 현재 시각과 무작위 64바이트 hash를 쓰므로 매번 서명이 달라집니다.

`Create`는 본문에서 `token`을 지우고, `ValidateS3Token`은 `access`·`signature`·`token`만 남깁니다. 확장 필드는 `credentials` 안에 들어가며 `token`은 `Create`에서 지워지는 필드여도 core 필드라서 확장 키로 쓸 수 없습니다. 결과 token의 ID는 `X-Subject-Token` header에서 읽고, provider token은 `X-Auth-Token`으로 함께 전송되지만 ProviderClient의 token은 바뀌지 않습니다.

native `Create`와 `ValidateS3Token`은 header builder를 호출하지 않습니다. 그래서 두 연산은 header 확장 옵션을 제공하지 않고, 설정에 남은 raw header 확장은 HTTP 전에 거부합니다.

## Python openstacksdk와의 차이

이 문서는 Gophercloud native 호출만 다루며, Python openstacksdk identity Resource와의 대응과 keystoneauth의 OAuth1·EC2 인증 plugin은 별도로 추적합니다.
