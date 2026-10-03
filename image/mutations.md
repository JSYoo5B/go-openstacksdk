# 이미지의 단일 태그와 활성화 상태 변경

`image.Service`의 `AddTag`·`RemoveTag`·`DeactivateImage`·`ReactivateImage`는 각각 전용 endpoint에 body/query 없는 요청을 보내고 실제 204 acknowledgement를 반환합니다. ID를 지정하면 metadata 조회 없이 해당 mutation을 준비합니다. `resource.Name`은 기존 native list 경로에서 정확한 이름을 찾아 선택한 ID로 요청합니다.

| Python 호출 | Go 호출 | 요청 |
|---|---|---|
| `conn.image.add_tag(image, tag)` | `service.AddTag(ctx, ref, tag)` | `PUT images/{id}/tags/{tag}` |
| `conn.image.remove_tag(image, tag)` | `service.RemoveTag(ctx, ref, tag)` | `DELETE images/{id}/tags/{tag}` |
| `conn.image.deactivate_image(image)` | `service.DeactivateImage(ctx, ref)` | `POST images/{id}/actions/deactivate` |
| `conn.image.reactivate_image(image)` | `service.ReactivateImage(ctx, ref)` | `POST images/{id}/actions/reactivate` |

다음 예시는 네 독립 요청을 순서대로 호출합니다. 앞선 요청이 성공한 뒤 다음 요청이 실패해도 앞선 변경을 되돌리는 transaction·cleanup은 수행하지 않습니다.

```go
package example

import (
    "context"

    "gophercloudsdk/image"
    "gophercloudsdk/resource"
)

type MutationProof struct {
    Added       *image.ImageTagResult
    Removed     *image.ImageTagResult
    Deactivated *image.ImageActionResult
    Reactivated *image.ImageActionResult
}

func mutateImage(ctx context.Context, service *image.Service, ref resource.Ref, tag string) (proof MutationProof, err error) {
    option := image.WithImageMutationHeader("X-Request-Source", "example")
    proof.Added, err = service.AddTag(ctx, ref, tag, option)
    if err != nil {
        return proof, err
    }
    proof.Removed, err = service.RemoveTag(ctx, ref, tag, option)
    if err != nil {
        return proof, err
    }
    proof.Deactivated, err = service.DeactivateImage(ctx, ref, option)
    if err != nil {
        return proof, err
    }
    proof.Reactivated, err = service.ReactivateImage(ctx, ref, option)
    return proof, err
}
```

옵션은 필수가 아닙니다. `ImageMutationOpts`는 `Headers map[string]string`만 가지며 `WithImageMutationOpts`는 전체 설정을 교체합니다. `WithImageMutationHeader`·`WithImageMutationHeaders`는 일반 header를 추가하고 canonical 이름이 같은 이전 일반 header를 덮어씁니다. helper 생성 시 입력 map을 복사하고 각 callback에 별도 설정을 한 번 전달한 뒤 다시 복사하므로 보관한 map·config·option slice가 준비된 요청을 바꾸지 않습니다. 보호된 auth·Host·cookie·framing·version·representation header와 충돌하는 map alias는 HTTP 전에 거부합니다.

## Python의 반환값과 Resource 캐시

```python
def mutate_image(conn, image_id, tag):
    conn.image.add_tag(image_id, tag)
    conn.image.remove_tag(image_id, tag)
    conn.image.deactivate_image(image_id)
    conn.image.reactivate_image(image_id)
```

고정된 openstacksdk commit `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 네 Proxy 메서드는 ID string 또는 Image Resource를 받으며 `None`을 반환합니다. string은 ID-only Resource로 준비되고 자동 Name search를 뜻하지 않습니다. Go의 `resource.Name`은 명시적인 추가 편의입니다.

Python `TagMixin.add_tag`·`remove_tag`는 HTTP 성공 뒤 Resource의 local tags list를 append/remove합니다. supplied Resource·subclass·connection·descriptor·dirty/cache·session/microversion 정책은 Go acknowledgement보다 넓은 동작입니다. Go는 tag set을 읽거나 PATCH로 교체하지 않고 caller의 Image/태그 캐시도 수정하지 않습니다. 기존 native `ReplaceImageTags`는 전체 tag set PATCH API로 별도로 유지됩니다.

Python tag helper의 `raise_from_response`는 400 이상을 거부하며, action leaf는 `session.post`를 호출하고 해당 helper를 직접 적용하지 않습니다. 따라서 adapter의 실패 정책과 Go의 실제 204 제한은 같은 계약이 아닙니다. Python remove의 local `list.remove`에 대한 `ValueError` 억제는 성공한 HTTP 뒤의 캐시 처리입니다. HTTP 404를 무시한다는 의미가 아니며 Go에는 `IgnoreMissing` 옵션이 없습니다.

## 태그의 문자와 경로

Go는 nonempty valid UTF-8 태그를 최대 255 Unicode code point까지 받습니다. 대소문자·앞뒤 공백·Unicode·`%`·`?`·`#`를 그대로 태그 데이터로 유지하고 `url.PathEscape`로 한 번 escape합니다. `%2F`는 literal percent 문자열이므로 wire에서는 `%252F`가 되며 SDK가 slash로 decode하지 않습니다. trim·dedupe·URL decode는 하지 않습니다.

empty, 정확히 `.` 또는 `..`, literal slash/backslash, Unicode control character, invalid UTF-8, 256 rune 이상은 Name lookup·callback·HTTP 전에 거부합니다. 이는 pinned Python의 raw URL 조립보다 엄격한 Go 선택입니다. encoded slash가 reverse proxy·WSGI route를 통과해 태그로 남는다고 보장할 수 없으므로 해당 모호한 입력을 제한합니다. 공백만 있는 태그는 nonempty literal 데이터입니다.

[공식 tag API](https://docs.openstack.org/api-ref/image/v2/index.html#add-image-tag)는 v2.0의 단일 태그 PUT/DELETE, 255자 제한과 정상 204를 설명합니다. 고정된 서버의 [tag controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/image_tags.py#L47-L101)는 set에 추가하므로 반복 Add는 전용 PUT을 다시 보내도 다른 태그를 교체하지 않습니다. 존재하지 않는 태그의 Remove는 서버 404입니다. 허용 태그 수·권한·실제 저장 성공은 서버 정책에 따릅니다.

## 선택한 ID와 실제 응답의 증거

`resource.ID`는 선택한 ID를 검증한 뒤 바로 전용 route로 보냅니다. `resource.Name`은 UUID처럼 보여도 정확한 이름이며 기존 native pagination의 모든 페이지에서 한 번 resolve합니다. absent·ambiguous 이름, surfaced native list/decode/HTTP/context 오류, 잘못된 resolved ID는 mutation을 보내지 않습니다. Name LIST의 body ownership·typed model·continuation은 기존 native 구현을 유지하며 새로운 일반 continuation guard를 뜻하지 않습니다.

`ImageTagResult`는 고정된 `ImageID`·`Tag`, `ImageActionResult`는 고정된 `ImageID`·`Action`과 각각 독립적인 `Body []byte`·`Header`·`StatusCode`를 가집니다. 실제 204의 Body는 빈 값·non-JSON·invalid UTF-8도 opaque bytes로 보존합니다. response의 ID·Location·status·tag 필드가 request target이나 결과 identity를 바꾸지 않습니다. acknowledgement만으로 현재 tag set이나 최종 image status를 추정하지 않으며 필요하면 caller가 별도 조회합니다.

실제 accepted 204의 read·Close·context 실패에서도 partial/full bytes를 가진 result와 `resource.ResponseError`를 함께 반환합니다. 두 값의 Body/Header는 서로 독립적으로 복사되고 원래 read/Close·`ctx.Err()`·custom cause를 유지합니다. body는 한 번 닫으며 accepted 실패 뒤 요청을 replay하지 않습니다. transport·native rejected status·pre-body callback 실패처럼 SDK response가 없는 오류는 nil result이고, 실제 200/201/202/206/404도 성공 result를 만들지 않습니다.

## Source·재시도·활성화 정책

context·source type·provider·origin·UTF-8 base·headers·version·Ref·태그를 옵션 전에 검사합니다. client/provider identity와 prefix·일반 headers를 먼저 capture하며 Name LIST와 mutation 모두 준비한 headers를 사용합니다. callback·lookup 뒤와 HTTP 전에 원래 source/provider를 다시 검사합니다. 안전한 source base/header 변경에도 capture한 route/map을 사용하고 원래 provider의 최신 auth token은 유지합니다. provider 교체나 foreign retarget은 차단합니다.

공통 `DoJSON`의 native pre-body reauth·retry·backoff와 동일 target의 configured redirect 정책은 유지합니다. 변경된 method·origin·path·query는 transport 전에 차단합니다. custom retry 정책에서는 accepted body 전에 여러 network attempt가 있을 수 있습니다. RetryFunc가 body ownership(`KeepResponseBody`·`JSONResponse`·`RawBody`·nil body와 JSON null)을 변경하면 원래 오류와 hook/encoding cause를 보존하고 다음 요청 전에 거부합니다. callback이 native `OkCodes`를 늘려도 처음 선택한 실제 204 제한은 유지합니다. native reauth wrapper의 `ErrOriginal`·`ErrReauth`는 wrapper 필드에서 확인합니다.

[공식 action API](https://docs.openstack.org/api-ref/image/v2/index.html#deactivate-image)는 v2.3의 deactivate/reactivate와 정상 204를 설명합니다. Go는 사전 metadata/status GET, capability discovery, wait·local status gate를 추가하지 않습니다. 인증·권한과 active/deactivated 전이 검증은 [고정된 서버 action controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/image_actions.py#L47-L111)가 결정하며 API 도입 버전을 speculative microversion header로 바꾸지 않습니다. local HTTP 테스트는 배포의 권한·reverse proxy·persistence·full Python Resource 정책을 증명하지 않습니다.
