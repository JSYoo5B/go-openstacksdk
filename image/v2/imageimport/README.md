# Glance interoperable image import

`API.ImportImage(ctx, ref, options...)`는 Image를 새로 조회한 뒤 `POST /images/{id}/import`를 보냅니다. 기본 method는 `glance-direct`입니다. 명시적인 ID는 조회 응답의 다른 ID로 바뀌지 않고, Name은 기존 Image collection의 정확한 name 비교로 한 번 해석한 뒤 그 ID를 조회합니다. 새 import를 시작하려면 image record의 `container_format`과 `disk_format`이 필요합니다.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "gophercloudsdk/image/v2/imageimport"
    "gophercloudsdk/resource"
)

func importStagedImage(ctx context.Context, client *gophercloud.ServiceClient, id string) (*imageimport.ImportResult, error) {
    return imageimport.New(client).ImportImage(ctx, resource.ID(id))
}
```

이 호출 전에 이미지 데이터를 stage해야 합니다. 이 API는 stage·생성·format 갱신을 수행하지 않습니다. Python의 대응 호출은 이미 형식 정보가 있는 Image Resource를 받습니다.

```python
image = conn.image.get_image(image_id)
response = conn.image.import_image(image)  # method='glance-direct'
```

## 조회 없는 KnownImage와 옵션

`ImportKnownImage(ctx, image, options...)`는 supplied native Image의 ID·두 format 문자열만 복사해 사용하고 GET을 보내지 않습니다. 이 모델의 상태·store 위치·임의 Properties를 요청 본문에 섞지 않습니다. 옵션 처리 후 caller의 모델이 바뀌어도 이미 선택한 입력은 바뀌지 않습니다.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "gophercloudsdk/image/v2/imageimport"
    "gophercloudsdk/image/v2/images"
)

func importKnownFromWeb(ctx context.Context, client *gophercloud.ServiceClient, image *images.Image, uri string) (*imageimport.ImportResult, error) {
    return imageimport.New(client).ImportKnownImage(ctx, image,
        imageimport.WithImportMethod("web-download"),
        imageimport.WithImportURI(uri),
        imageimport.WithImportStores("fast", "reliable"),
        imageimport.WithImportAllStoresMustSucceed(false),
    )
}
```

```python
response = conn.image.import_image(
    image, method="web-download", uri=uri,
    stores=["fast", "reliable"], all_stores_must_succeed=False)
```

`ImportOpts`의 `Method`, `URI`, `RemoteRegion`, `RemoteImageID`, `RemoteServiceInterface`, `Store`, `Stores`, `AllStores`, `AllStoresMustSucceed`, `Headers`, `Fields`, `MethodFields`를 `WithImportOpts`로 전달할 수 있습니다. bulk 옵션은 pointer·slice·header·JSON extension 값을 snapshot하고 전체 설정을 교체합니다. 개별 `WithImport...` 옵션은 순서대로 적용되며 마지막 값이 유효합니다. 같은 snapshot 옵션을 독립 호출에서 재사용할 수 있습니다.

`WithImportStore(id)`는 `X-Image-Meta-Store` 호환 헤더만 보냅니다. `WithImportStores(ids...)`는 최상위 `stores` 배열만 보냅니다. 둘을 함께 지정하지 않습니다. 명시적인 빈 Store나 빈 Stores 항목은 사전 거부하며 nil/empty Stores 배열은 생략합니다. `AllStores == true`도 개별 store 선택과 함께 지정할 수 없습니다. 기존 client의 store 헤더가 있으면 plural Stores 또는 true AllStores와의 충돌을 HTTP 전에 거부합니다. 명시적인 `WithImportStore`는 기존 store 헤더를 덮어씁니다. nil boolean은 생략하고 명시적인 false는 보존합니다. `AllStoresMustSucceed`의 nil은 서버 기본값에 맡기며 SDK가 true를 넣지 않습니다.

`web-download`는 nonempty URI가 필요하며 다른 method에는 URI를 지정할 수 없습니다. `glance-download`는 remote region과 image ID를 모두 지정합니다. interface는 선택 사항이고 생략하면 서버 기본값에 맡깁니다. incomplete·orphan·다른 method의 remote 값은 사전 거부합니다. 알려지지 않은 nonempty method는 서버로 전달하므로 deployment가 지원하는 method 목록을 SDK가 추측하지 않습니다.

`WithImportField`는 최상위 JSON, `WithImportMethodField`는 `method` 객체의 JSON extension을 추가합니다. 각 namespace의 SDK 소유 키를 extension으로 덮어쓰지 못합니다. 최상위 method·store·boolean과 method 객체의 name·URI·remote 키가 해당합니다. 헤더는 `WithImportHeader` / `WithImportHeaders`로 설정하며 auth·transport·version과 typed store control은 SDK가 소유합니다. 임의 request builder·query·base path를 받지 않습니다.

## 202 응답과 실패

`ImportResult`는 요청한 `ImageID`와 실제 받아들인 202의 `Body`, `Header`, `StatusCode`를 반환합니다. 빈 본문과 임의 acknowledgement 본문을 실제 응답 그대로 보존합니다. 받아들인 본문을 읽는 도중 실패해도 `result, err`를 함께 반환해 수신한 증거를 볼 수 있습니다. HTTP 상태가 202라는 사실은 image가 `active`가 되었거나 모든 store에 복사가 끝났다는 뜻이 아닙니다. Task·Image 모델이나 Location 기반 ID를 합성하지 않습니다.

HTTP·전송·조회 decode·context 오류는 import workflow의 재조회·재생성·정리 요청을 일으키지 않습니다. 동일 Provider를 공유해 live token과 기존 configured transport/reauth 정책을 사용하고 method/URL을 고정합니다. 조회와 POST에 snapshot한 헤더·client 설정을 사용하며 공유 ServiceClient를 변경하지 않습니다. 응답 body는 자동 JSON model로 해석하거나 URL로 따라가지 않습니다.

## 원본 SDK와 서버 계약

기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [Proxy.import_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L440-L536)와 [Image.import_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L368-L427)입니다. Python의 `_get_resource`는 ID만으로 GET을 하지 않으며 supplied Image에서 format을 검사합니다. `ImportKnownImage`는 이 조회 없는 형태에 대응하고, `ImportImage(Ref)`의 fresh 조회와 Name 해석은 Go 편의 API입니다. mutable Resource·Store 객체·dict 입력·cache/default/descriptor·session injection 전체를 재현하지 않습니다.

Python singular `store`는 호환 헤더와 one-element `stores` 배열을 함께 보냅니다. Go는 서버 호환 경로를 명확하게 구분해 header-only로 보냅니다. Python의 incomplete remote tuple은 interface가 없으면 생략될 수 있지만 Go는 필요한 region·ID를 검사하고 선택적인 interface만 생략합니다. Python이 URI를 누락한 web method를 보내는 경우도 Go는 사전 거부합니다. Go의 canonical format·native 전체 Image decode·UTF-8·안전한 route 검사도 별도 정책입니다.

[공식 Image Import API](https://docs.openstack.org/api-ref/image/v2/index.html#import-an-image)는 POST 202와 method별 precondition을 정의합니다. `glance-direct`는 stage된 `uploading` image, web/remote download는 `queued` image, `copy-image`는 데이터가 있는 `active` image를 대상으로 합니다. 이 상태·URL 허용·store 존재·권한·quota·backend 성공은 서버가 판단합니다. import가 API 2.6, multistore가 API 2.8에 도입되었다는 문서상 사실을 client microversion 최소값 검사로 바꾸지 않습니다. 자동 discovery·stage·wait·cleanup과 전체 create_image/upload workflow는 별도입니다.

기존 native `Create`는 기존 `CreateOpts`의 `method` envelope와 error-only 결과를, `Get`은 import discovery 결과를 유지합니다. 새 workflow의 최상위 stores/boolean과 actual acknowledgement가 그 기존 선언을 바꾸지는 않습니다.

실제 계약은 [HTTP import 테스트](../../../api/glance_import_contracts_test.go), [옵션 snapshot·본문 namespace 테스트](import_options_test.go), [조회·승인 응답 증거 테스트](import_core_test.go), [native binding 경계 테스트](../../../internal/cmd/sdkgen/glance_import_test.go)에서 확인합니다.
