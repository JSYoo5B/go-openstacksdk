# Glance 이미지와 store의 이미지 복사본 삭제

`image.Service.DeleteImage`는 전체 이미지 또는 특정 store의 복사본을 삭제합니다. 기본값은 전체 이미지 삭제와 `IgnoreMissing=true`입니다. 명시적인 `resource.ID`는 metadata GET 없이 바로 `DELETE /images/{id}`를 요청합니다. 실제 204만 삭제 응답으로 기록하며, 정상적으로 확인한 부재를 무시하면 result와 error가 모두 nil입니다.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "gophercloudsdk/image"
    "gophercloudsdk/resource"
)

func deleteWholeImage(ctx context.Context, client *gophercloud.ServiceClient, id string) (*image.DeleteImageResult, error) {
    return image.New(client).DeleteImage(ctx, resource.ID(id))
}
```

고정된 Python proxy의 전체 이미지 호출도 기본적으로 부재를 무시합니다. proxy는 성공과 무시한 부재 모두 `None`을 반환합니다.

```python
conn.image.delete_image(image_id)
```

## Store 선택과 옵션

`DeleteImageOpts`의 `StoreID == nil`은 전체 이미지 삭제입니다. nonnil StoreID는 `DELETE /stores/{storeID}/{imageID}`를 선택합니다. 명시적인 빈 store ID는 첫 HTTP 전에 오류이며 전체 삭제로 바뀌지 않습니다. image ID와 store ID는 유효한 UTF-8의 안전한 route segment여야 합니다.

`IgnoreMissing == nil`은 true이고 명시적인 false는 부재를 오류로 반환합니다. `WithDeleteImageOpts`는 생성 시 pointer를 snapshot하여 전체 설정을 교체합니다. `WithDeleteImageStore`와 `WithDeleteImageIgnoreMissing`은 해당 값을 교체합니다. 옵션은 전달 순서대로 적용하며 custom callback은 preparation에서 한 번 실행합니다. 옵션 slice와 각 callback의 설정을 복사하므로 caller가 보관한 pointer가 뒤의 설정이나 진행 중인 요청을 바꾸지 않습니다.

다음 예는 지정한 store의 복사본만 삭제하고 부재를 오류로 받습니다.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "gophercloudsdk/image"
    "gophercloudsdk/resource"
)

func deleteStoreCopy(ctx context.Context, client *gophercloud.ServiceClient, id, storeID string) (*image.DeleteImageResult, error) {
    ignoreMissing := false
    return image.New(client).DeleteImage(ctx, resource.ID(id),
        image.WithDeleteImageOpts(image.DeleteImageOpts{
            StoreID: &storeID,
            IgnoreMissing: &ignoreMissing,
        }),
    )
}
```

```python
conn.image.delete_image(image_id, store=store_id, ignore_missing=False)
```

Python proxy는 truthy store만 store 삭제로 선택합니다. 따라서 Python의 빈 store는 전체 삭제로 해석되지만 Go의 명시적인 빈 StoreID는 거부됩니다. Python의 직접 `Store.delete_image` 호출은 `ignore_missing=False`가 기본이며, proxy는 자신의 기본 true를 전달합니다. 해당 leaf는 성공 시 HTTP Response를 반환하고 전체 이미지의 `Resource.delete`는 Resource를 반환하지만, proxy는 두 반환값을 모두 버립니다.

[공식 store 삭제 API](https://docs.openstack.org/api-ref/image/v2/index.html#delete-image-from-store)는 Image API v2.10부터 제공되며 마지막 location의 삭제를 허용하지 않습니다. store 지원·권한·마지막 location 여부와 409 판단은 서버가 담당합니다. SDK는 store discovery나 location count 조회를 수행하거나 실패한 store 삭제를 전체 삭제로 바꾸지 않습니다.

## 실제 응답과 부재 처리

`DeleteImageResult`는 실제 DELETE 204의 `ImageID`, `StoreID`, raw `Body`, 복사한 `Header`, `StatusCode`를 보존합니다. 전체 이미지 삭제의 StoreID는 빈 문자열입니다. body를 JSON으로 decode하거나 metadata에서 결과를 합성하지 않습니다. 204의 body read·Close·context 오류에도 받아들인 result와 `resource.ResponseError`를 함께 반환하며, result와 error의 bytes/header는 별도로 소유합니다.

404는 body를 읽고 Close한 뒤 부재 정책을 판단하기 위해 내부적으로 처리합니다. body read·Close·context가 모두 정상일 때만 기본 정책이 `(nil, nil)`을 반환합니다. false 정책이면 `resource.NotFoundError`와 원래 404의 method·고정 URL·expected 204·actual 404·raw body·header를 보존합니다. 404 body read·Close·context 실패는 기본 정책에서도 오류이며 원래 404와 실패 원인을 함께 유지합니다. nested error에 404가 포함됐다는 이유로 다른 실패를 무시하지 않습니다. 무시한 404는 삭제 성공이나 store API 지원의 증거가 아닙니다.

`resource.Name`은 기존 Images pager로 모든 page에서 정확한 이름을 검색하고 중복을 검사한 뒤 선택한 ID로 DELETE합니다. 기존 pager의 continuation 정책을 유지하며, 이름 LIST body의 소유권과 오류 표면도 기존 native pager 계약을 따릅니다. 성공적으로 완료한 검색에서 발생한 직접적인 logical NotFound만 기본 정책으로 무시합니다. pager가 반환한 LIST 404·body/decode/transport 실패·ambiguity·cause가 있는 NotFound는 무시하지 않으며 logical absence에 HTTP 응답을 합성하지 않습니다.

이 workflow는 실제 204만 받아들입니다. [공식 전체 이미지 삭제 API](https://docs.openstack.org/api-ref/image/v2/index.html#delete-image)는 204를 정상 응답으로 정의합니다. 고정된 Python의 error translator는 status `<400`을 허용하며, native Gophercloud `images.Delete`는 202/204를 허용합니다. 새 workflow의 204 검사는 이 두 동작보다 엄격합니다. 기존 `Images.Delete`, `Collection.Delete`/`Remove`와 native wrapper의 서명·동작은 유지합니다.

## 요청 경계와 오류 원인

context·service·provider·image endpoint·headers·Ref·옵션을 HTTP 전에 검사합니다. prefix와 headers를 capture하고 원래 Provider의 live token을 사용합니다. 옵션 적용과 이름 검색 뒤에 원래 source/provider와 target을 다시 검사합니다. redirect는 따라가지 않으며 DELETE route는 선택한 ID와 store로 고정합니다.

configured reauth·retry·backoff는 받아들인 응답 body를 소유하기 전의 요청 정책으로 유지합니다. 다만 내부적으로 처리하는 DELETE 404는 native `RetryFunc`를 거치지 않습니다. 따라서 모든 status에 대한 native retry 동작과 동일하지 않으며, configured policy가 받아들이기 전 DELETE를 재전송할 수 있어 HTTP packet 한 번만을 보장하지 않습니다. 소유한 204/404 body의 read·Close 실패 이후에는 retry나 replay를 수행하지 않습니다.

DELETE의 빈 요청 body와 응답 body 소유권은 workflow가 고정합니다. `RetryFunc`가 `JSONBody`·`RawBody`·`JSONResponse`·`KeepResponseBody`를 바꾸면 추가 DELETE 전에 `resource.ErrInvalidOption`과 원래 요청 실패, callback이 반환한 오류를 함께 보존해 반환합니다.

받아들인 204/404 body는 한 번 닫고, 나머지 status의 body는 native Request가 소유합니다. error는 `request.Wrap("DeleteImage", "image", ...)`를 통해 read·Close·transport·native status 원인과 `ctx.Err()`·`context.Cause(ctx)`를 보존하여 `errors.Is`/`errors.As`로 검사할 수 있습니다. native reauth 실패의 `ErrOriginal`/`ErrReauth`는 `errors.As`로 얻은 `gophercloud.ErrUnableToReauthenticate`의 필드에서 확인합니다.

추가 metadata GET, protected/status의 client gate, 삭제 대기, cache·Swift object·task cleanup은 제공하지 않습니다. Python Resource 객체 overload, cloud `delete_objects`와 session 정책 전체의 parity도 이 workflow 범위에 포함되지 않습니다.

Python 비교는 openstacksdk commit `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [Proxy.delete_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L970-L997)와 [Store.delete_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/service_info.py#L46-L75), native 비교는 Gophercloud `v2.15.0`의 구현을 기준으로 합니다.

실제 whole/store route·preflight·snapshot·Name·404·204 증거·provider·native compatibility 계약은 [공개 HTTP 테스트](delete_contracts_test.go), [core 증거 테스트](delete_core_test.go), [옵션 소유권 테스트](delete_options_test.go)에서 확인합니다. [생성기 회귀 테스트](../internal/cmd/sdkgen/glance_delete_test.go)는 native Delete signature·result·Image ID와 기존 binding의 보존을 검사합니다.
