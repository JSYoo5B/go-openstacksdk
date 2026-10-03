# Glance store와 import 정보 조회

`ServiceInfo`는 배포된 Glance의 store 목록과 import method 정보를 읽습니다. `conn.ImageV2(ctx)`의 `ServiceInfo`, 상위 `image.Service.API.ServiceInfo`, 직접 생성한 `serviceinfo.New(client)`에서 같은 API를 사용합니다. 조회 결과로 다른 이미지 연산을 자동 허용하거나 차단하지 않습니다.

## Store 목록과 detail

기본 `ListStores`/`AllStores`는 `GET /info/stores`를 요청합니다. `Details=false`, `Limit=0`, `Marker=""`, `MaxItems=0`, `Paginated=nil`이 기본이며, limit/marker는 생략하고 cap 없이 다음 page를 순회합니다. `Details=true`는 `/info/stores/detail`을 선택하며 query에 details를 넣지 않습니다.

```go
package example

import (
    "context"

    "gophercloudsdk"
    "gophercloudsdk/image/v2/serviceinfo"
)

func discoverStores(ctx context.Context, conn *gophercloudsdk.Connection) ([]*serviceinfo.Store, error) {
    service, err := conn.ImageV2(ctx)
    if err != nil {
        return nil, err
    }
    return service.ServiceInfo.AllStores(ctx,
        serviceinfo.WithListStoresDetails(true),
        serviceinfo.WithListStoresMaxItems(20),
    )
}
```

고정된 Python proxy도 details를 로컬 route 선택에 사용합니다.

```python
for store in conn.image.stores(details=True, max_items=20):
    print(store.id, store.description, store.is_default)
```

[공식 store API](https://docs.openstack.org/api-ref/image/v2/index.html#list-stores)는 v2.8부터 제공되는 multistore discovery이며 v2.7에서는 404를 반환합니다. [detail API](https://docs.openstack.org/api-ref/image/v2/index.html#list-stores-detail)는 type·read-only·weight·properties 등의 정보를 제공하며 접근 권한은 서버 정책을 따릅니다. SDK는 403/404를 숨기거나 detail 실패 뒤 basic 목록으로 fallback하지 않습니다. Limit/Marker는 caller가 명시적으로 보내는 wire 값입니다. Glance의 store discovery는 이 값으로 page를 나누지 않으므로 SDK가 다음 marker를 합성하지 않습니다. `MaxItems`는 로컬 소비 cap이며 wire limit hint를 생성하지 않습니다.

`ListStores`는 생성 시 HTTP를 실행하지 않는 재사용 가능한 iterator입니다. iteration마다 옵션 callback을 한 번씩 적용하고 독립적인 source·query·header·pointer snapshot을 사용합니다. `WithListStoresOptions`는 생성 시 `Paginated` pointer를 복사하고 전체 typed 설정을 교체합니다. 별도 query/header 옵션은 유지하며 뒤의 옵션이 같은 값을 교체합니다. `Paginated=false`는 첫 page만, `MaxItems>0`은 최대 raw 행 수만 소비합니다. cap은 사용하지 않을 행의 decode와 continuation 검사 전에 멈춥니다. 성공한 빈 `AllStores`는 nonnil empty slice이며, 뒤 page나 행이 실패하면 부분 slice 대신 nil과 오류를 반환합니다. iterator에서 이미 받은 행은 caller에게 남습니다.

서버가 광고한 body links·next와 HTTP Link만 continuation으로 사용하고, 빈 page는 종료합니다. link가 없는 nonempty 응답에서 ID로 다음 marker를 만들거나 같은 목록을 반복 요청하지 않습니다. continuation은 capture한 origin·reverse prefix·선택한 collection path와 초기 nonpagination query를 유지해야 하며 반복 link/marker를 거부합니다. pinned Python의 limit 기반 marker fallback과 더 넓은 version-relative next 경로 정규화는 이 facade와 다릅니다.

음수 limit/cap, invalid UTF-8 marker/query, malformed limit/marker, nil/error callback, 지원하지 않는 body·argument·microversion 확장은 HTTP 전에 오류입니다. `WithListStoresQuery`는 실제 wire query 확장입니다. details·max_items·paginated·base_path·session 등의 로컬 제어를 query로 전달할 수 없습니다. 일반 header는 `WithListStoresHeader`로 전달하고 공통 header 검사와 보호 정책을 적용합니다. Python은 선언된 Body attribute를 로컬 filter로 사용하고 unknown query를 버리지만, Go의 wire helper는 이 분류나 properties subset filter를 구현하지 않습니다.

## Import method 정보

`GetImportInfo`는 ID 없이 `GET /info/import`를 한 번 읽습니다. [공식 import discovery](https://docs.openstack.org/api-ref/image/v2/index.html#import-methods-and-values-discovery)는 v2.6부터 제공되며 정상 응답은 200입니다. 응답의 method 목록이 비어 있어도 그대로 반환합니다.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "gophercloudsdk/image"
    "gophercloudsdk/image/v2/serviceinfo"
)

func discoverImportMethods(ctx context.Context, client *gophercloud.ServiceClient) (*serviceinfo.ImportInfo, error) {
    return image.New(client).API.ServiceInfo.GetImportInfo(ctx)
}
```

```python
info = conn.image.get_import_info()
methods = info.import_methods
print(methods)
```

Go의 `ImportInfo.ImportMethods`는 optional pointer입니다. `import-methods`가 생략되거나 null이면 nil이고, nonnull 값은 object여야 합니다. 중첩 `Description`·`Type`은 생략/null이면 빈 문자열이며 nonnull이면 string입니다. `Value`는 생략/null이면 nil, 빈 array이면 nonnil empty slice이고, nonnull array의 각 값은 string이어야 합니다. 새로운 method 이름을 enum으로 제한하지 않으며 중첩 unknown 필드는 `ImportMethods.Body`에 남깁니다. `WithGetImportInfoHeader`는 일반 header만 추가합니다.

기존 native `ImageImport.Get(ctx)`도 `/info/import`를 요청합니다. 그 typed model과 callable은 유지하며, 새 facade는 optional 값과 raw 응답 증거를 추가로 제공합니다.

## Canonical 필드와 실제 응답

Store의 `ID`·`Description`은 canonical `id`·`description`만 읽고 생략/null은 빈 문자열로 남깁니다. ID는 수동 route 선택에 쓰지 않는 passive 정보입니다. `IsDefault`와 `ReadOnly`는 각각 공식 `default`·`read-only` 키를 읽고 생략/null을 nil로 구분합니다. JSON bool과 lexical string `"true"`/`"false"`만 허용합니다. Python의 bool descriptor는 nonempty string `"false"`도 true로 변환하므로 이 해석은 의도적인 차이입니다. `is_default`나 대소문자 변형은 typed 값을 바꾸지 않는 raw unknown 필드입니다.

`Properties`는 생략/null이면 nil이고 nonnull object의 값은 `json.RawMessage`로 보존합니다. detail의 `Type`은 optional string, `Weight`는 optional exact int64이며 숫자 문자열이나 fraction을 coercion하지 않습니다. pinned Python Store는 type·read-only·weight descriptor를 선언하지 않으므로 이 필드들은 공식 wire 응답의 편의 필드입니다.

전체 응답은 valid UTF-8의 flat JSON object여야 합니다. stores는 필수 nonnull array이며 각 행은 object입니다. alternate/capitalized envelope를 자동으로 풀지 않습니다. model의 `resource.Metadata.Body`는 canonical·unknown 원문 JSON 값을 보존하고 `Header`·`StatusCode`는 실제 page/GET 응답을 복사합니다. 큰 unknown 숫자도 float로 변환하지 않습니다. 생략과 null의 차이는 raw Body에서 확인합니다.

실제 200만 허용합니다. accepted body read·Close·JSON·UTF-8·model·context 실패는 `resource.ResponseError`의 전체 raw Body·Header·StatusCode와 원인을 보존합니다. body를 한 번 닫고 실패 뒤 replay하지 않습니다. 원래 read·Close·transport·native 오류와 `ctx.Err()`·`context.Cause(ctx)`는 request operation 오류를 통해 검사할 수 있습니다. pinned Python은 `<400`을 허용하고 일부 fetch decode 실패를 삼키므로 strict schema와 status는 동일한 Python coercion/response 계약이 아닙니다.

context·image service·provider·endpoint·headers를 옵션 전에 검사하고 callback 뒤와 각 page 전에 다시 확인합니다. route·reverse prefix·source headers·provider를 capture하되 원래 provider의 live auth token을 사용합니다. source/provider 교체로 요청을 다른 대상에 보내지 않습니다. 기존 redirect callback 정책은 유지하며 method·origin·path·query가 바뀐 대상은 다음 transport 전에 거부합니다. configured pre-body reauth·retry·backoff는 유지하고 accepted body 실패 후에는 재요청하지 않습니다. 공통 `DoJSON`은 retry callback의 body 소유권 변경을 거부하고 원래 expected 200을 다시 검사합니다. native가 거부한 응답의 body 처리와 reauth wrapper는 native 정책을 유지합니다. `ErrUnableToReauthenticate`의 `ErrOriginal`·`ErrReauth`는 `errors.As`로 얻은 wrapper의 필드에서 확인합니다.

Resources·Find·CRUD·Wait·info/usage·자동 discovery gate/cache는 제공하지 않습니다. Python의 mutable Resource·cache·객체 overload·session·microversion·base_path override·coercion·로컬 Body filter 전체도 이 두 조회의 범위에 포함되지 않습니다.

Python 비교는 openstacksdk commit `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [Proxy.stores/get_import_info](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L2273-L2295), [Store/Import](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/service_info.py#L22-L44), native 비교는 Gophercloud `v2.15.0`을 기준으로 합니다.

실제 route·schema·source·pagination·오류 계약은 [공개 HTTP 테스트](contracts_test.go), [core 증거 테스트](core_test.go), [옵션 소유권 테스트](options_test.go)에서 확인합니다. [Connection 테스트](../../../connection_image_serviceinfo_test.go)는 shared client·prefix·live auth를, [생성기 회귀 테스트](../../../internal/cmd/sdkgen/glance_serviceinfo_test.go)는 registry와 native Get의 유지·drift 거부를 검증합니다. [응답 검증 hook 테스트](../../../internal/rest/response_validation_test.go)는 전체 body UTF-8 검사와 page별 실행·lazy break 경계를 확인합니다.
