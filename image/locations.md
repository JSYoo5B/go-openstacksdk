# 전용 image location 추가·조회

`image.Service.AddImageLocation`은 선택한 이미지의 `POST images/{id}/locations`를 준비하고 실제 202 acknowledgement를 반환합니다. `GetImageLocations`는 같은 endpoint에 body/query 없는 GET을 보내 실제 200의 유한한 location 배열을 읽습니다. ID는 직접 사용하며 명시적인 `resource.Name`만 기존 exact-name native paging으로 한 번 resolve합니다.

```go
package example

import (
    "context"

    "gophercloudsdk/image"
    "gophercloudsdk/resource"
)

func addDefaultLocation(ctx context.Context, service *image.Service, ref resource.Ref, locationURL string) (*image.AddImageLocationResult, error) {
    return service.AddImageLocation(ctx, ref, locationURL)
}

func addValidatedThenGetLocations(ctx context.Context, service *image.Service, ref resource.Ref, locationURL, algorithm, value string) (ack *image.AddImageLocationResult, locations *image.ImageLocationsResult, err error) {
    ack, err = service.AddImageLocation(ctx, ref, locationURL,
        image.WithAddImageLocationOpts(image.AddImageLocationOpts{
            Headers: map[string]string{"X-Request-Source": "example"},
        }),
        image.WithImageLocationValidation(algorithm, value),
        image.WithAddImageLocationHeader("X-Location-Request", "add"))
    if err != nil {
        return ack, nil, err
    }
    locations, err = service.GetImageLocations(ctx, ref,
        image.WithGetImageLocationsOpts(image.GetImageLocationsOpts{
            Headers: map[string]string{"X-Request-Source": "example"},
        }),
        image.WithGetImageLocationsHeader("X-Location-Request", "get"))
    return ack, locations, err
}
```

첫 함수의 기본 JSON body는 `{"url": "입력 문자열", "validation_data": {}}`입니다. 두 번째 함수는 hash pair를 전달한 뒤 별도 GET을 수행합니다. POST와 GET은 독립 요청이고 202 직후 GET에 새 location이 나타난다고 보장하지 않습니다. POST accepted body 실패에서도 nonnil acknowledgement를 보존하므로 오류와 result를 함께 확인합니다.

## Python Resource와 실제 서버 응답

```python
def add_and_read_locations(conn, image_id, location_url, validation_data=None):
    created = conn.image.add_image_location(
        image_id, location_url, validation_data=validation_data
    )
    locations = list(conn.image.image_locations(image_id))
    return created, locations
```

고정된 openstacksdk commit `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 `add_image_location`은 Image ID 또는 Resource에서 ID를 얻고, 기본 `validation_data=None`을 `{}`로 바꿔 `ImageLocation`을 create합니다. `image_locations`는 같은 Resource의 generic list generator를 반환합니다. string은 자동 Name search를 뜻하지 않으며 Go의 명시적인 Name은 추가 편의입니다.

Python의 inherited create는 비어 있는 응답의 JSON `ValueError`를 무시하고 입력 URL·validation data로 준비한 mutable Resource를 반환할 수 있습니다. Go의 Add result는 고정된 request `ImageID`·`URL`과 실제 `Body []byte`·`Header`·`StatusCode`입니다. 서버가 location 객체나 stable location ID를 돌려줬다고 추정하지 않고, seeded Resource·descriptor·dirty/cache·session 동작을 재현하지 않습니다.

[공식 Add Location 문서](https://docs.openstack.org/api-ref/image/v2/index.html#add-location)는 정상 200을 기재하지만, 고정된 Glance commit `57f7dd9e76ef24e1e9013eceaa703bd442469a24`의 [serializer](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L1882-L1883)는 body 없는 202를 설정합니다. Go는 이 실제 소스의 202만 받으며 문서의 200이나 다른 2xx로 성공 범위를 넓히지 않습니다. [controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L1161-L1257)는 비동기 `location_import`를 접수하므로 202는 hash 검증·활성화·완료 증거가 아닙니다.

## URL·validation data·옵션

location URL은 nonempty valid UTF-8 문자열 데이터입니다. scheme·대소문자·공백·percent·Unicode·query·fragment·route처럼 보이는 내용을 그대로 JSON에 넣으며 client target으로 사용하지 않습니다. SDK가 이 URL을 parse·normalize·escape하거나 직접 fetch하지 않고 scheme/길이 whitelist도 추가하지 않습니다. 실제 URL·store·HTTP URI 정책은 서버가 검사합니다.

`AddImageLocationOpts`는 `ValidationData *ImageLocationValidation`·`Headers map[string]string`만 가집니다. nil validation은 `{}`이고 supplied pointer는 `OSHashAlgo`·`OSHashValue` 둘 다 nonempty valid UTF-8이어야 합니다. 명시적인 zero/partial pair는 Name lookup·HTTP 전에 거부합니다. 이 pair preflight는 Python의 arbitrary dict보다 제한적인 Go 선택입니다. 알고리즘·hex·digest 길이를 local whitelist로 검사하지 않고 trim·대소문자·공백을 바꾸거나 hash를 계산/검증하지 않습니다. 서버 schema·controller·비동기 작업이 지원 여부와 값을 판단합니다.

`WithAddImageLocationOpts`·`WithGetImageLocationsOpts`는 전체 concrete 설정을 교체합니다. `WithImageLocationValidation`은 pair를, 각 Add/Get의 `Header`·`Headers` helper는 일반 header를 추가합니다. 입력 map·pointer와 option slice를 복사하고 각 callback에 독립 config를 한 번 전달한 뒤 다시 복사합니다. 보관한 config·map·pointer가 다음 callback이나 직렬화한 body를 바꾸지 않습니다. 보호된 auth·framing·version·representation headers, nil/error callback과 충돌하는 header alias는 HTTP 전에 거부합니다.

request에는 `checksum`·`metadata`·unknown body field·`do_secure_hash`·query·wait·verify·IgnoreMissing 옵션을 추가하지 않습니다. `do_secure_hash`는 확인한 서버의 [설정](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/common/config.py#L414)이며 request field가 아닙니다. Source의 version 정책은 유지하지만 Python `ImageLocation._max_microversion='2.17'`이라는 framework capability hint를 새 microversion header나 사전 discovery gate로 바꾸지 않습니다.

## 유한 배열과 passive location 데이터

GET result는 선택한 `ImageID`, `Locations []*ImageLocation`과 실제 전체 `Body []byte`·`Header`·`StatusCode`를 가집니다. valid UTF-8 JSON의 bare array만 받고 `[]`는 nonnil empty slice입니다. null·object·envelope·scalar·empty body와 null/nonobject row는 오류입니다. Python generic list의 단일 object→한 row 변환이나 Link·marker·limit fallback을 추가하지 않습니다.

각 `ImageLocation.URL`은 canonical `url`의 optional `*string`입니다. missing/null은 nil이고 명시적인 empty string은 nonnil입니다. canonical `metadata`는 missing/null이면 nil, `{}`면 nonnil raw map이며 nonnull일 때 object여야 합니다. 대소문자 decoy·unknown·`id`·`status`·`image_id`·`next`는 `Body map[string]json.RawMessage`에 passive 데이터로 보존합니다. metadata의 null·0·false·nested value·`9007199254740993`·fraction도 float64로 바꾸지 않습니다. row URL·ID·metadata가 route·Name·marker·지원 capability를 만들지 않습니다.

[공식 GET 문서](https://docs.openstack.org/api-ref/image/v2/index.html#get-location)와 실제 [controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L1259-L1280)는 200 location 배열을 기준으로 합니다. 이 handler는 전체 목록에서 `id`·`status`를 제거하고 pagination/query나 `show_multiple_locations` gate를 추가하지 않습니다. metadata 이미지 응답에서 locations가 노출되는 정책과 전용 GET의 권한 정책을 동일시하지 않습니다.

## 응답 실패와 기존 source 정책

POST actual 202의 empty/non-JSON/invalid UTF-8 bytes는 opaque acknowledgement입니다. accepted read·Close·context 실패에서도 독립적인 Body/Header 복사본을 가진 result와 `resource.ResponseError`를 함께 반환합니다. GET은 accepted read·Close·context·UTF-8·JSON·schema 실패에서 nil typed result를 반환하고 전체 실제 200 bytes/header/status는 `ResponseError`에 남깁니다. partial typed row 성공으로 바꾸지 않습니다. 원래 read/Close·`ctx.Err()`·custom cause를 보존하며 body는 한 번 닫고 accepted 실패 뒤 replay하지 않습니다.

기존 mutation의 source 준비·header 규칙을 유지합니다. source/client/provider·prefix·일반 headers를 callback 전에 capture하고 body/header 옵션을 준비한 뒤 정확한 Name lookup을 수행합니다. absent·ambiguous·surfaced native list/decode/HTTP/context 실패나 잘못된 chosen ID는 location 요청을 보내지 않습니다. Name의 native typed model·body ownership·continuation은 기존 경계입니다. provider 교체·foreign retarget은 차단하고 원래 provider의 live auth token을 사용합니다.

공통 `DoJSON`은 native pre-body retry·reauth·backoff와 같은 target의 configured redirect를 유지합니다. method·origin·path·query 변경은 transport 전에 차단하고 실제 202/200 acceptance를 고정합니다. RetryFunc의 동일 serialized JSON 교체는 허용하되 URL/hash·response ownership 변경, RawBody·KeepResponseBody·JSONResponse 변경과 GET nil body→JSON null은 다음 요청 전에 거부합니다. 원래 pre-body 오류·hook/encoding cause를 유지하며 native reauth의 `ErrOriginal`·`ErrReauth`는 wrapper 필드에서 확인합니다. configured 정책으로 accepted body 전에 여러 attempt가 있을 수 있습니다.

queued 상태·권한·store·import lock·실제 hash 검증·backend persistence는 서버 소유 정책입니다. SDK는 metadata/status gate, location ID 합성, cache mutation, task 조회, polling·wait·rollback·cleanup을 수행하지 않습니다. local HTTP 계약은 배포의 기능·권한·비동기 완료나 full Python Resource/cache/session parity를 증명하지 않습니다.

실제 local 계약은 [외부 HTTP tests](locations_contracts_test.go), [core tests](locations_core_test.go), [option tests](locations_options_test.go)에서 검증합니다. [Connection test](../connection_image_locations_test.go)는 공유 native client와 passive result를, [generator test](../internal/cmd/sdkgen/glance_locations_test.go)는 기존 native API 보존과 전용 workflow 문서를 확인합니다.
