# 서버의 이미지·암호 응답

서버 API는 응답의 해석까지 SDK가 처리합니다.

| 작업 | 반환값 |
|---|---|
| `CreateImage` | 생성한 이미지 ID와 error |
| `Evacuate` | 응답의 관리자 암호와 error |
| `GetPassword` | 암호화된 관리자 암호와 error |

openstacksdk의 `conn.compute.get_server_password(server)`처럼 `GetPassword`도 기본적으로 암호화된 문자열을 반환합니다. RSA 개인 키를 전달하면 SDK가 base64와 RSA PKCS#1 v1.5 복호화를 처리합니다.

```go
// ctx context.Context, service *computev2.Service, serverID string을 사용하는 함수 안에서
encrypted, err := service.Servers.GetPassword(ctx, serverID)
if err != nil { return err }
_ = encrypted

// privateKey는 *crypto/rsa.PrivateKey입니다.
password, err := service.Servers.GetPassword(ctx, serverID,
    servers.WithGetPasswordPrivateKey(privateKey))
if err != nil { return err }
_ = password
```

`computev2`는 `gophercloudsdk/compute/v2`, `servers`는 `gophercloudsdk/compute/v2/servers`입니다. Connection의 `ComputeV2(ctx)`가 서비스 객체를 제공합니다.

빈 암호 응답은 빈 문자열을 반환합니다. 잘못된 개인 키나 nil 옵션은 요청 전에 거부합니다. 복호화 오류와 HTTP 오류는 반환한 error로 확인합니다. 옵션을 여러 개 지정하면 마지막 키를 사용하며 `WithGetPasswordPrivateKey(nil)`은 기본 동작으로 돌아갑니다. 개인 키는 호출 동안 변경하지 않습니다.

`CreateImage`는 Nova microversion에 따라 Location 헤더 또는 JSON 본문에서 이미지 ID를 해석합니다. `Evacuate`에서 서버가 관리자 암호를 제공하지 않으면 빈 문자열을 반환합니다. 이 반환값만으로 이미지 생성이나 evacuation 완료를 뜻하지는 않습니다. 완료 확인은 이미지·서버의 공통 `Wait`를 사용합니다.
