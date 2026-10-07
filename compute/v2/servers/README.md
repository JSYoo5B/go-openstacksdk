# 서버의 이미지·암호 응답

서버 API는 응답의 해석까지 SDK가 처리합니다.

`Metadata(ctx, serverID)`와 `ConsoleOutput(ctx, serverID, options...)`의 Python 대응·문자열 반환·length 생략과0은 [Compute 조회 가이드](../../user-read-apis.md)를 참고하세요. 기존 native `ShowConsoleOutput`은 별도로 유지합니다.

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

`computev2`는 `github.com/JSYoo5B/gophercloudsdk/compute/v2`, `servers`는 `github.com/JSYoo5B/gophercloudsdk/compute/v2/servers`입니다. Connection의 `ComputeV2(ctx)`가 서비스 객체를 제공합니다.

암호가 누락되거나 null·빈 문자열이면 Go는 빈 문자열을 반환합니다. 고정 Python 구현은 누락/null을 `None`으로 반환하며 `password` 값을 그대로 꺼냅니다. Go는 숫자·boolean 등 문자열이 아닌 `password` 값을 decode 오류로 반환합니다. 잘못된 개인 키나 nil 옵션은 요청 전에 거부합니다. 복호화 오류와 HTTP 오류는 반환한 error로 확인합니다. 옵션을 여러 개 지정하면 마지막 키를 사용하며 `WithGetPasswordPrivateKey(nil)`은 기본 동작으로 돌아갑니다. 개인 키는 호출 동안 변경하지 않습니다.

`GetPassword`는 선택한 Compute ServiceClient의 microversion을 유지하며 추가 discovery나 자동 버전 선택을 하지 않습니다. 고정 [Python Server.get_password](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server.py)는 [Resource._get_microversion](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)에 따라 session default를 우선 사용하고, 없으면 서버 지원 최대 버전과 Server의 최대 버전 2.100 중 작은 값을 선택할 수 있습니다.

GetPassword는 native HTTP200만 받아들이며, 다른 status와 JSON decode 오류의 원래 cause를 보존합니다. HTTP203도 성공 문자열로 처리하지 않습니다.

`CreateImage`는 Nova microversion에 따라 Location 헤더 또는 JSON 본문에서 이미지 ID를 해석합니다. `Evacuate`에서 서버가 관리자 암호를 제공하지 않으면 빈 문자열을 반환합니다. 이 반환값만으로 이미지 생성이나 evacuation 완료를 뜻하지는 않습니다. 완료 확인은 이미지·서버의 공통 `Wait`를 사용합니다.

## 서비스별 상태·삭제 대기

`WaitForServer(ctx, ref, options...)`는 ACTIVE/ERROR와 120초·2초 기본값을
SDK가 적용합니다. `WaitForState`는 caller가 대상 상태를 지정하며 SDK 시간
제한이 없습니다. `WaitForServerState`는 caller 대상과 120초 제한을 사용합니다. `WaitForDelete`는 삭제 요청 없이 실제
미존재를 최대 120초 기다립니다. 호출별 `resource.WaitOption`이 기본값을
순서대로 재정의합니다. 기존 `WaitFor`/`WaitForDeletion`과 네이티브
`WaitForStatus`의 계약은 유지됩니다. 전체 예제와 Python Resource/cache의
차이는 [서비스 대기 가이드](../../../docs/service-waits.md)를 참고하세요.
