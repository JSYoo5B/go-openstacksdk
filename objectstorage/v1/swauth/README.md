# Swift swauth native 호출

`swauth` package는 Keystone 대신 Swift의 TempAuth·swauth 미들웨어로 인증하는 배포를 위한 고정 Gophercloud v2.15.0 [swauth](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/objectstorage/v1/swauth/requests.go) 호출을 그대로 감쌉니다. `conn.ObjectStorageV1(ctx)`의 `Swauth` 필드로 접근합니다.

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Auth(ctx, opts, options...)` | `GET {IdentityBase}auth/v1.0`, `X-Auth-User`·`X-Auth-Key` header | 200 |
| `NewObjectStorageV1(ctx, opts, options...)` | 위 `Auth` 한 번 | 200 |

## 요청과 결과

두 호출은 object-store endpoint가 아니라 ProviderClient의 `IdentityBase` 뒤에 `auth/v1.0`을 붙인 주소로 `GET`을 보냅니다. `AuthOpts.User`(`account:user` 형식)와 `Key`는 필수라 비어 있으면 HTTP 전에 오류입니다. 요청에는 swauth header와 함께 ProviderClient가 가진 기존 `X-Auth-Token`도 실립니다.

`Auth`의 결과는 본문이 아니라 응답 header에서 읽습니다. `X-Auth-Token`은 `Token`, `X-Storage-Url`은 `StorageURL`, `X-CDN-Management-Url`은 `CDNURL`이 됩니다. `WithAuthHeader`로 header를 더할 수 있지만 `X-Auth-User`·`X-Auth-Key`와 겹치는 key나 nil 옵션은 HTTP 전에 거부합니다. 200 밖의 status는 `Auth`/`swauth` 문맥을 가진 오류입니다.

## Swift client 생성의 부작용

`NewObjectStorageV1`은 `Auth` 결과의 `StorageURL`을 끝에 `/`를 붙여 endpoint로 쓰는 새 ServiceClient를 돌려줍니다. 이 client는 같은 ProviderClient를 공유하고, native 함수가 받은 token을 ProviderClient의 `TokenID`에 직접 씁니다. 그래서 호출 뒤에는 같은 Connection에서 만든 다른 서비스 client도 swauth token을 보냅니다. Keystone 인증과 섞어 쓰려면 swauth 전용 ProviderClient를 따로 만들어야 합니다. 새 client의 `Type`은 비어 있습니다.

이 호출은 native 오류를 operation 문맥 없이 그대로 돌려줍니다. 필수 header 누락과 200 밖의 status가 모두 이에 해당하고, nil 옵션만 `NewObjectStorageV1`/`swauth` 문맥으로 감쌉니다. 실패하면 ProviderClient의 token은 바뀌지 않습니다.
