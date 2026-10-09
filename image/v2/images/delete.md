# Glance native 이미지 삭제

`service.API.Images.Delete(ctx, imageID)`는 Gophercloud `v2.15.0`의 native Delete와 `ExtractErr()`를 호출하고 **error만** 반환합니다. 기본 성공 status는202·204이며 Record·ACK·응답 Body/Header를 성공 반환값으로 제공하지 않습니다. 준비한 client는 `images.New(client).Delete`를 사용합니다. 전체 v2 facade는 아래처럼 반환값과 오류를 먼저 처리한 뒤 `api.Images.Delete(ctx, imageID)`로 같은 메서드를 사용합니다.

```go
api, err := conn.ImageV2(ctx)
if err != nil { return err }
```

이미 준비한 Image Service와 context에서 호출합니다. 이 코드는 실제 이미지를 삭제합니다.

```go
err := service.API.Images.Delete(ctx, imageID)
if err != nil {
    var native gophercloud.ErrUnexpectedResponseCode
    if errors.As(err, &native) {
        fmt.Printf("HTTP %d, expected=%v, response bytes=%d\n",
            native.Actual, native.Expected, len(native.Body))
    }
    return err
}
```

위 오류 검사에는 Go `errors`, `fmt`와 `github.com/gophercloud/gophercloud/v2`를 사용합니다. Connection 설정·독립 main은 [owned 삭제 가이드](../../image-record-delete.md#독립-go-main)와 [설치 안내](../../../docs/install.md)를 참고하세요. owned main을 native Delete 예제로 바꾸려면 삭제 호출과 반환 처리를 위 error-only 형태로 바꿉니다.

## 입력과 기본 요청

ID string을 [upstream Delete](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/images/requests.go#L222-L226)의 `ServiceURL` 경로로 그대로 전달합니다. 이름 조회·GET·store 선택·ignore_missing·wait 옵션이 없으며 SDK가 ID를 검증하거나 하나의 URI segment로 escape하지 않습니다. 빈 ID·slash·query delimiter도 upstream URL 처리에 맡깁니다. owned 레코드의 raw JSON ID 검증과 private current identity 정책을 이 native 메서드의 보장으로 옮기지 않습니다.

DELETE Body는 nil이고 accepted Body는 JSON을 decode하지 않고 읽어 버립니다. 기존 client의 ResourceBase/Endpoint, MoreHeaders, microversion과 live auth token을 사용합니다. 기본 [OkCodes](https://github.com/gophercloud/gophercloud/blob/v2.15.0/provider_client.go#L582-L597)는202·204입니다. 200·201·301·404 등은 기본 거부이며 **404를 자동으로 무시하지 않습니다**. client hook이 바꾸는 native 정책은 그대로 따릅니다.

## 오류와 native 재시도

SDK는 원래 error를 `resource.OperationError`의 `Operation="Delete"`, `Resource="images"`로 감쌉니다. 원인을 다른 오류 형식으로 번역하지 않습니다. `errors.As`로 `gophercloud.ErrUnexpectedResponseCode`의 Actual/Expected, Method/URL, Body/ResponseHeader를 읽고 `errors.Is`로 transport·context·read 원인을 확인합니다. callback이 최종 오류를 교체하면 그 callback 오류가 원인으로 반환됩니다.

- native `RetryFunc`의 transport·거부 status 재시도를 유지합니다. 재시도 요청은 당시 live token을 읽습니다.
- 401은 provider `ReauthFunc` 설정과 기존 재인증 조건을 따릅니다. 실패한 재인증의 `ErrUnableToReauthenticate`에는 `ErrOriginal`과 `ErrReauth`가 남습니다.
- 429는 설정된 `RetryBackoffFunc`와 native retry 제한을 따릅니다. SDK가 새 backoff·자동 성공·owned source guard를 추가하지 않습니다.

고정 [native accepted drain](https://github.com/gophercloud/gophercloud/blob/v2.15.0/provider_client.go#L570-L580)은 Read 오류를 반환하지만 `Close()` 오류는 버립니다. accepted Read 오류로 `RetryFunc`를 새로 호출하지 않으며 부분 ACK도 만들지 않습니다. [거부 응답 처리](https://github.com/gophercloud/gophercloud/blob/v2.15.0/provider_client.go#L458-L535)는 Read·Close 오류를 별도 cause로 반환하지 않고, 읽은 partial Body와 실제 status/header를 native HTTP 오류에 보존합니다. 이 경계는 accepted Read/Close 실패에서 ACK와 `resource.ResponseError`를 함께 보존하는 owned API와 다릅니다.

## Python과 owned 삭제의 범위

| API | 기본 status·missing | 결과·범위 |
|---|---|---|
| native `API.Images.Delete` | 202·204, 404 오류 | error만, raw ID upstream 전달 |
| 기존 `service.DeleteImage` | strict204, default missing 무시 | Ref·concrete store 옵션·실제204 결과 |
| `DeleteImageRecord` | actual200..399, 깨끗한 최종 native404 기본 무시 | owned ID/Record, whole Record+ACK 또는 store ACK |
| Python `conn.image.delete_image` | Source status<400, ignore_missing=True | whole/store proxy, 공개 반환값 None |

[owned 삭제의 Python/Go 비교](../../image-record-delete.md)는 private identity·descriptor/local 상태·store admin 정책·immutable 반환과 actual receipt를 설명합니다. native mapping은 고정 Gophercloud 전송·결과 계약의 mapping이며 Python mutable Image·Store lifecycle이나 cache/session 전체의 동등성 판정이 아닙니다. [native 계약 테스트](delete_contracts_test.go)는 경로·status·opaque Body·read/Close·재시도 경계를 검증하며 실제 cloud 권한·backend 삭제 완료를 증명하지 않습니다.
