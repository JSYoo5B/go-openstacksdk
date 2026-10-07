# Glance image staging

`API.StageImage(ctx, ref, data, options...)`는 기존 Image를 새로 조회해 정확한 `queued` 상태를 확인하고, `PUT /images/{id}/stage` 승인 뒤 같은 ID를 다시 조회합니다. 명시적인 ID는 응답의 다른 ID로 바뀌지 않습니다. Name은 기존 Images collection의 정확한 name 비교와 native 페이지 정책으로 한 번 해석한 뒤 그 ID를 조회합니다.

```go
package example

import (
    "context"
    "io"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/gophercloudsdk/image/v2/imagedata"
    "github.com/JSYoo5B/gophercloudsdk/resource"
)

func stageExistingImage(ctx context.Context, client *gophercloud.ServiceClient, id string, data io.Reader) (*imagedata.StageImageResult, error) {
    return imagedata.New(client).StageImage(ctx, resource.ID(id), data)
}
```

Python의 대응 호출은 `queued` 상태를 가진 Image Resource를 받습니다. ID만 넘긴 Python 호출은 자동 GET으로 그 상태를 채우지 않습니다.

```python
image = conn.image.get_image(image_id)
image = conn.image.stage_image(image, data=data)
```

## 조회 없는 KnownImage와 크기

`StageKnownImage(ctx, image, data, options...)`는 supplied native Image의 ID·status만 옵션 처리 전에 복사하고 초기 GET을 보내지 않습니다. 이 상태는 caller가 보유한 값이므로 서버의 현재 상태가 바뀌었으면 PUT에서 거부될 수 있습니다. 성공한 PUT 뒤 fresh GET은 이 경로에도 적용됩니다. 두 경로 모두 exact lowercase `queued`를 요구하고 Image 생성·import·완료 대기를 자동 실행하지 않습니다.

```go
package example

import (
    "context"
    "io"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/gophercloudsdk/image/v2/imagedata"
    "github.com/JSYoo5B/gophercloudsdk/image/v2/images"
)

func stageKnownImage(ctx context.Context, client *gophercloud.ServiceClient, image *images.Image, data io.Reader, remainingBytes int64) (*imagedata.StageImageResult, error) {
    return imagedata.New(client).StageKnownImage(ctx, image, data,
        imagedata.WithStageSize(remainingBytes),
        imagedata.WithStageHeader("X-Trace", "stage-image"),
    )
}
```

```python
image = conn.image.stage_image(image, data=data, size=remaining_bytes)
```

`StageOpts`는 `Size *int64`와 `Headers map[string]string`을 제공합니다. `WithStageOpts`는 pointer·header map을 snapshot해 전체 설정을 교체하고, `WithStageSize`, `WithStageHeader`, `WithStageHeaders`는 순서대로 적용됩니다. 개별 helper의 마지막 값이 유효하며 같은 snapshot 옵션을 독립 호출에서 재사용할 수 있습니다.

Size의 nil은 생략하고 명시적인 0은 보존하며 음수는 HTTP 전에 거부합니다. Size는 `X-OpenStack-Image-Size`에만 전달합니다. SDK는 reader의 Len·Stat·Seek를 검사하거나 HTTP Content-Length를 추론하지 않습니다. reader의 현재 위치부터 읽는 데이터의 예상 크기를 caller가 선택합니다. 크기를 생략하면 서버가 실제 데이터를 기준으로 계산합니다.

binary PUT은 `Content-Type: application/octet-stream`과 빈 `Accept`를 사용합니다. 이 헤더와 typed Size 헤더는 SDK가 소유하므로 일반 header extension으로 덮어쓰지 못합니다. snapshot한 일반 헤더는 metadata GET에도 전달되지만 binary Content-Type을 metadata 조회에 강제로 적용하지 않습니다. 공유 ServiceClient의 헤더·Provider 설정을 변경하지 않습니다.

## reader와 전송

caller의 `io.Reader`를 빌려 현재 위치부터 스트리밍합니다. SDK는 caller reader를 Close·Seek·buffer하지 않으며 재사용 가능한 파일이나 요청 본문을 만들지 않습니다. 호출이 끝난 뒤 파일을 닫는 책임과 blocking Read를 해제하는 책임은 caller에게 있습니다. 같은 옵션은 재사용할 수 있지만 동일 reader의 동시 사용 안전성까지 보장하지 않습니다.

binary PUT의 private Provider에서는 reauth·retry·backoff를 끄고 redirect를 따르지 않습니다. 401·429·전송 오류 등이 발생해도 SDK가 소비된 본문을 다시 보내지 않습니다. 원래 Provider의 token은 PUT dispatch 시 읽고 원래 HTTPClient의 transport·timeout을 유지합니다. caller가 제공한 RoundTripper 내부 동작까지 한 번의 네트워크 전송으로 제한하는 것은 아닙니다. metadata 조회는 기존 JSON 요청 정책을 사용합니다.

## 실제 승인과 조회 결과

`StageImageResult.ImageID`는 고정한 요청 ID입니다. `Acknowledgement`는 실제 받아들인 PUT 204의 `Body`, `Header`, `StatusCode`를 독립적으로 보존합니다. `Image`는 후속 GET 200의 native decode까지 성공했을 때만 채웁니다. outer `Body`, `Header`, `StatusCode`는 실제 받아들인 후속 GET 200의 증거이며 read·decode 오류가 나도 수신한 값을 보존합니다. SDK가 `uploading` 또는 `active` 모델을 합성하지 않습니다. 반환 Image는 native GetResult extractor로 해석하므로 import method·store ID 헤더의 모델 projection도 적용합니다. 이 model projection은 실제 metadata `Body`와 `Header`를 수정하지 않습니다.

204를 받아들인 뒤 body read나 후속 GET이 실패하면 `result, err`를 함께 반환해 staging 승인 증거를 확인할 수 있습니다. PUT body read가 실패하면 추가 GET을 보내지 않습니다. PUT이 승인되기 전에 실패한 경우에는 결과가 nil입니다. 조회 decode·HTTP·전송·context 오류는 자동 restage·import·DELETE를 실행하지 않습니다. 후속 응답의 Location·ID도 요청 경로를 바꾸지 않습니다.

## 원본 SDK와 지원 범위

기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [Proxy.stage_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L538-L581), [Image.stage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L325-L366), [get_file_size](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L462-L491)입니다. Python은 mutable Image의 기존 data 또는 filename을 사용하고, 크기가 없으면 seek 가능한 파일의 전체 길이를 구한 뒤 cursor를 복구합니다. 현재 위치부터 남은 길이와 다를 수 있습니다. Go는 filename을 열거나 size를 추론하지 않으며 nonnegative int64를 사용합니다. Python의 bool·negative int 허용, Resource dirty/cache/default/coercion과 injected session 전체를 재현하지 않습니다.

[공식 Stage API](https://docs.openstack.org/api-ref/image/v2/index.html#stage-binary-image-data)는 queued image의 binary PUT과 204, optional size·quota 조건을 정의합니다. stage된 데이터는 import가 끝나기 전에는 다운로드할 수 없습니다. method 활성화·권한·quota·저장 공간·size 불일치 여부는 서버가 판단하며 SDK가 배포 버전이나 기능을 추측하지 않습니다. 이 호출은 [import 제출](../imageimport/README.md)과 분리되어 있습니다.

기존 native `Stage(ctx, id, data) error`, `Upload`, `Download`와 상위 `image.Service.Upload`는 그대로 제공합니다. 새 workflow의 queued 검사·optional size·단일 PUT 정책·fresh 조회와 승인 증거가 native ABI를 바꾸지 않습니다.

실제 계약은 [전체 HTTP workflow 테스트](../../../api/glance_stage_contracts_test.go), [reader·승인 증거·native decode 테스트](stage_core_test.go), [옵션 snapshot·header ownership 테스트](stage_options_test.go), [native binding 경계 테스트](../../../internal/cmd/sdkgen/glance_stage_test.go)에서 확인합니다.
