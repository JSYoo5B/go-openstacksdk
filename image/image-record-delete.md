# Glance 이미지 레코드 삭제: Python과 Go

`Connection.DeleteImageRecord`와 `image.Service.DeleteImageRecord`는 이미지 ID 또는 SDK가 반환한 `ImageRecord`를 받아 전체 이미지나 특정 store의 복사본을 삭제합니다. 기본값은 전체 삭제와 `ignore_missing=true`입니다. SDK가 identity·옵션 snapshot과 opaque 응답 증거를 관리하며, caller는 concrete `With…` 옵션을 사용합니다.

| 호출 | Python `conn.image` | Go 정상 반환값 |
|---|---|---|
| 전체 이미지 | `delete_image(image, ignore_missing=True)` | 새 `Record`와 실제 삭제 `Acknowledgement` |
| 특정 store | `delete_image(image, store=store, ignore_missing=True)` | `Record=nil`, 실제 삭제 `Acknowledgement` |
| 기본 missing 처리 | 성공·무시한 missing 모두 `None` | 깨끗한 최종 native404를 무시하면 `nil, nil` |

Go 반환형은 `ImageRecordDeleteResult`이며 `Record *ImageRecord`와 `Acknowledgement *DeleteImageResult`를 구분합니다. ACK는 이번 DELETE의 실제 Body/Header/StatusCode와 고정 ImageID/StoreID입니다. 전체 삭제의 ACK StoreID는 빈 문자열입니다. 전체 삭제의 Record는 독립 local Resource 반환이며 삭제 응답을 body로 decode한 결과가 아닙니다. 고정 Python 공개 proxy는 두 branch의 반환값을 모두 버립니다.

기존 [DeleteImage](delete.md)는 `resource.Ref`와 strict204 계약을 유지합니다. [native Delete](v2/images/delete.md)는 upstream raw ID·기본202/204·error-only 정책을 유지합니다. 새 API는 owned ID/Record 입력과 actual200..399 ACK를 제공하는 별도 호출입니다. 이름 검색·자동 GET·store discovery·삭제 완료 wait·별도 Swift object/task cleanup은 실행하지 않습니다. Glance 서버의 backend 동작은 서버가 처리합니다.

## Python 사용과 권한 분기

```python
import openstack

conn = openstack.connect(cloud="dev")
conn.image.delete_image("image-id", ignore_missing=False)

# 별도의 실행 예: 특정 store 복사본만 삭제한다.
admin = openstack.connect(cloud="admin")
admin.image.delete_image("other-image-id", store="store-id", ignore_missing=False)
```

고정 Glance pin `57f7dd9e76ef24e1e9013eceaa703bd442469a24`에서 전체 삭제의 [delete_image 정책38–51행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L38-L51)은 project scope의 `ADMIN_OR_PROJECT_MEMBER`입니다. 기본 전체 경로는 core user 범위입니다.

**특정 store 삭제는 기본 정책에서 admin 분기입니다.** [store route442–446행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/router.py#L442-L446)은 `ImagesController.delete_from_store`로 연결합니다. [controller769–773행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L769-L773)이 `delete_locations()`를 호출하고, [API policy224–229행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/policy.py#L224-L229)은 [delete_image_location 정책159–172행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L159-L172)의 `ADMIN`을 적용합니다. 이 경로도 project scope입니다. 실제 policy override, 소유권, enabled backend, 이미지 상태·location 제한은 서버가 판단하며 SDK에서 role을 검사하지 않습니다. [서비스 우선순위 기준](../docs/glance-policy-priorities.md)을 함께 참고하세요.

## 독립 Go main

[설치 안내](../docs/install.md)를 따라 아래 코드를 `main.go`로 저장합니다. `go run . -cloud dev -image-id ID -ignore-missing=false`는 실제 전체 이미지 삭제를 한 번 수행합니다. `-store STORE`는 특정 store를 선택하며 기본 정책에서 admin 또는 해당 배포가 허용한 cloud를 사용해야 합니다. 문서 검증에서는 빌드만 수행하고 실제 cloud 삭제는 실행하지 않습니다.

기본 ID 입력에는 GET이 없습니다. `-record`를 지정할 때만 예제가 `conn.Image(ctx)`로 얻은 Image Service의 `service.GetImageRecord`를 명시적으로 먼저 호출해 SDK Record를 준비합니다. `-service`는 같은 DELETE를 Service 메서드로 실행하며 Connection 메서드와 중복 호출하지 않습니다. `-store=`처럼 store를 명시하면서 빈 값을 주면 전체 삭제로 바꾸지 않고 오류로 처리합니다. `-record`의 명시 GET에서 발생한 missing·조회 오류는 DELETE의 ignore_missing 옵션으로 숨기지 않습니다.

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    imageID := flag.String("image-id", "", "required literal image ID")
    storeID := flag.String("store", "", "optional explicit store ID; omission deletes whole image")
    ignoreMissing := flag.Bool("ignore-missing", true, "ignore a clean final native404")
    fromRecord := flag.Bool("record", false, "explicitly fetch an SDK record before deletion")
    viaService := flag.Bool("service", false, "use image.Service instead of Connection for the DELETE")
    flag.Parse()
    storeSelected := false
    flag.Visit(func(value *flag.Flag) {
        if value.Name == "store" { storeSelected = true }
    })
    if *imageID == "" { log.Fatal("-image-id is required") }
    if storeSelected && *storeID == "" { log.Fatal("an explicitly selected store must have a nonempty ID") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, *storeID, storeSelected,
        *ignoreMissing, *fromRecord, *viaService); err != nil {
        var response *resource.ResponseError
        if errors.As(err, &response) {
            fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
        }
        log.Fatal(err)
    }
}

func printDelete(result *image.ImageRecordDeleteResult, operationErr error) error {
    value := map[string]any{"partial": result != nil && operationErr != nil}
    if result == nil && operationErr == nil { value["ignored_missing"] = true }
    if result != nil {
        value["acknowledgement"] = result.Acknowledgement
        if result.Record != nil {
            value["record"] = map[string]any{
                "resource": result.Record.Resource,
                "wire": result.Record.Wire,
                "earlier_receipt_status_code": result.Record.StatusCode,
                "earlier_receipt_body_bytes": len(result.Record.Envelope),
                "import_methods": result.Record.ImportMethods,
            }
        }
    }
    if operationErr != nil { value["error"] = operationErr.Error() }
    body, err := json.MarshalIndent(value, "", "  ")
    if err != nil { return errors.Join(operationErr, err) }
    fmt.Println(string(body))
    return operationErr
}

func run(ctx context.Context, cloud, imageID, storeID string,
    storeSelected, ignoreMissing, fromRecord, viaService bool) error {
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    input := image.ImageRecordDeleteRequest{ID: imageID}
    if fromRecord {
        service, err := conn.Image(ctx)
        if err != nil { return err }
        record, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: imageID})
        if err != nil { return err }
        input = image.ImageRecordDeleteRequest{Record: record}
    }
    options := []image.ImageRecordDeleteOption{
        image.WithImageRecordDeleteIgnoreMissing(ignoreMissing),
        image.WithImageRecordDeleteHeader("X-Request-Source", "image-record-delete-example"),
    }
    if storeSelected { options = append(options, image.WithImageRecordDeleteStoreID(storeID)) }
    var result *image.ImageRecordDeleteResult
    if viaService {
        service, err := conn.Image(ctx)
        if err != nil { return err }
        result, err = service.DeleteImageRecord(ctx, input, options...)
        return printDelete(result, err)
    }
    result, err = conn.DeleteImageRecord(ctx, input, options...)
    return printDelete(result, err)
}
```

출력은 두 칸 들여쓰기 JSON입니다. ACK의 `Body []byte`는 Go JSON encoding의 base64 표현이며 opaque bytes를 JSON object로 해석하지 않습니다. `record`의 earlier receipt는 이전 Record에서 복사한 값이고 이번 DELETE의 증거는 `acknowledgement`입니다. ID로 전체 삭제한 경우 earlier StatusCode는0, Wire·Envelope·Header는 nil인 constructor Record를 받습니다. Store 경로는 Record를 반환하지 않습니다. 예제의 1분 context는 caller가 정한 인증·선택적 GET·DELETE 예산입니다.

## 입력과 concrete 옵션

`ImageRecordDeleteRequest{ID string; Record *ImageRecord}`는 두 입력 중 하나를 받습니다. Record에는 SDK가 반환한 private raw state가 있어야 합니다. Get·목록·검색·이전 owned 작업에서 받은 Record를 사용하며, public Resource만 직접 만든 Record는 거부합니다. 요청 ID는 **private current body의 canonical id**에서 읽습니다. public Resource의 projected id, Wire, Envelope, Header, properties의 id, self/file/schema URL을 바꿔도 삭제 대상이 바뀌지 않습니다. 반환값도 입력 Record와 독립 복사본입니다.

이 API는 이름 입력·project lookup을 제공하지 않습니다. ID는 유효한 nonblank UTF-8 문자열이어야 하며 control 문자·정확한 `.`·`..`는 거부합니다. 허용한 ID는 하나의 escaped URI segment로 보존합니다. Python의 falsy 값·임의 object stringification이나 슬래시를 strip하는 `utils.urljoin` 규칙으로 identity를 넓히지 않습니다.

요청 identity로 읽는 raw JSON 문자열은 unpaired UTF-16 surrogate escape(예: `"\uD800"`)를 거부합니다. valid pair `"\uD83D\uDE00"`, 실제 U+FFFD와 `"\uFFFD"`, literal backslash-u 문자열 `"\\ud800"`는 각각의 정상 값을 보존합니다. `encoding/json`의 replacement character 치환으로 다른 ID를 선택하지 않으며, 이 규칙은 일반 Body descriptor 변환과 구분됩니다.

`ImageRecordDeleteOpts`는 `Headers map[string]string`, `StoreID *string`, `StoreRecord *serviceinfo.StoreRecord`, `IgnoreMissing *bool`을 제공합니다. StoreRecord의 package 경로는 `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo`입니다.

- `WithImageRecordDeleteOpts`: pointer·map·Record를 snapshot해 전체 옵션을 교체합니다. StoreID와 StoreRecord가 모두 nil인 full opts는 전체 삭제를 선택합니다.
- `WithImageRecordDeleteHeader/Headers`: 일반 요청 헤더를 추가·교체합니다.
- `WithImageRecordDeleteStoreID`: 명시적 store ID를 선택하고 StoreRecord를 비웁니다.
- `WithImageRecordDeleteStoreRecord`: concrete StoreRecord를 선택하고 StoreID를 비웁니다. nil Record는 두 store 선택을 비워 전체 삭제로 되돌립니다.
- `WithImageRecordDeleteIgnoreMissing`: missing 처리만 변경합니다. nil/default는 true입니다.

full opts에서 StoreID와 StoreRecord를 동시에 지정하면 오류입니다. explicit empty StoreID와 missing/null/invalid StoreRecord id도 오류이며 전체 삭제로 fallback하지 않습니다. StoreRecord는 `Resource.Body["id"]`만 사용합니다. name·properties·wire id를 대체 ID로 삼거나 Store 목록·상세 discovery를 추가 호출하지 않습니다. 나머지 StoreRecord 필드를 image descriptor로 projection하지 않습니다.

helper factory는 map·pointer·Record를 snapshot하고 옵션은 전달 순서대로 한 번 적용합니다. SDK가 입력·옵션 값을 소유하며 각 option·location callback 전후의 source/context 검사를 유지합니다. callback에서 source를 변경했다가 나중 callback에서 되돌려도 앞서 관측한 오류를 없애지 않습니다. nil/error callback, invalid identity/header·보호된 transport 헤더 변경은 HTTP 전 오류입니다. 기존 live 인증 token과 native retry/reauth hooks는 유지합니다. 임의 Resource routing/session 설정을 옵션으로 주입하는 builder를 구현할 필요가 없습니다.

## 전체 삭제와 store 삭제의 local 상태

전체 branch는 `DELETE /images/{id}`입니다. literal ID는 Image constructor raw seed를 만들고, SDK Record는 private current/original/dirty 상태를 복사합니다. HTTP 전에 현재 raw body와 이번 호출의 location으로 65필드 Resource를 projection합니다. DELETE가 body를 보내지 않아도 descriptor conversion 오류는 이 preparation에서 요청을 막을 수 있습니다.

accepted 전체 삭제는 새 Record를 반환합니다. raw current·original·dirty body를 그대로 보존하며, `has_body=False`처럼 응답 Body를 decode하거나 pending body를 clean하지 않습니다. Resource는 준비한 현재 location을 나타내고 Wire·Envelope·Header·StatusCode는 이전 Record의 receipt를 보존합니다. `ImportMethods`는 준비 때 reset하고 accepted 삭제 응답의 `OpenStack-image-import-methods` 헤더를 새로 소비합니다. 이번 응답의 raw bytes·header·status는 ACK에서 확인합니다. 이전 receipt와 ACK의 차이를 새로운 GET 결과나 삭제 완료 상태로 해석하지 않습니다.

store branch는 `DELETE /stores/{store_id}/{image_id}`입니다. Image의 private canonical id와 Store의 canonical id만 선택합니다. **Image body projection·response translation·ImportMethods 갱신을 수행하지 않으며**, 성공이나 accepted 처리 오류에서도 `Record=nil`입니다. Source `Store.delete_image`도 Image를 준비·변경하지 않고 `session.delete(url)`와 응답 오류 검사만 수행합니다. Source whole branch의 Resource headers·microversion preparation과 store leaf의 별도 kwargs 없는 호출은 구분됩니다. Go 일반 헤더 옵션은 명시적인 transport 확장입니다.

## 실제 응답과 오류

두 branch 모두 actual HTTP200..399의 Body를 opaque bytes로 읽어 ACK에 보존하며 JSON을 decode하지 않습니다. null·array·invalid JSON·non-UTF-8 Body도 JSON 형태를 이유로 거부하지 않습니다. 고정 Source의 `raise_from_response`는 status<400을 허용하고, Go final HTTP profile은200..399로 경계를 정한 명시적인 정책입니다. 202 같은 접수 응답은 backend 삭제 완료를 별도로 관찰한 결과가 아닙니다.

기본 `IgnoreMissing=nil/true`는 **깨끗한 최종 direct native404만** `nil result, nil error`로 끝납니다. 명시적인 false는 missing 오류를 반환합니다. 404의 read·Close·context·source/callback 실패는 missing 처리로 숨기지 않습니다. 무시한 missing에는 실제 accepted ACK를 합성하지 않습니다.

accepted read·Close·source/context guard 실패에는 partial `Acknowledgement`와 error를 함께 반환하고 `Record=nil`을 유지합니다. native rejected status나 accepted 응답 없는 transport 실패는 nil result와 원인 오류입니다. `errors.As`로 `resource.ResponseError` 또는 native HTTP 오류를, `errors.Is`로 원인·context 오류를 확인합니다. ACK의 bytes·headers와 오류의 증거는 독립적으로 보존합니다. nonnil 결과만으로 성공을 판단하지 말고 err를 확인합니다. accepted 처리 실패 때문에 DELETE를 replay하거나 이미 접수한 삭제를 되돌리지 않습니다. native retry 정책과 caller의 명시적 재호출은 별도입니다.

## 고정 Source와 비교 범위

openstacksdk pin은 `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`입니다. [Proxy.delete_image970–997행](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L970-L997)은 whole/store branch 뒤 결과를 버립니다. 직접 [Resource.delete2094–2152행](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2094-L2152)은 같은 mutable Resource를 반환하고, [Store.delete_image46–75행](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/service_info.py#L46-L75)은 실제 Response를 반환하며 직접 leaf의 ignore_missing 기본값은 false입니다. 이 Go 메서드는 공개 image proxy의 기본 true를 사용합니다.

Python은 store의 truthiness로 branch를 선택해 `store=""`를 전체 삭제로 처리하지만, Go는 explicit empty store를 거부합니다. Python의 Store Resource는 id가 없어도 truthy일 수 있으며 `utils.urljoin`은 falsy 인자를 빈 문자열로 바꾸고 path를 stringification합니다. Go의 canonical string identity·escaped segment·immutable Record/ACK는 명시적으로 정한 profile이며 모든 Python coercion과 동등하다고 주장하지 않습니다.

Source [Image header 소비429–437행](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L429-L437), [Resource translation1338–1394행](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338-L1394), [Proxy._get_resource563–602·_delete669–714행](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L563-L714)의 local lifecycle을 고정 profile에 적용합니다. arbitrary mutable Resource/Munch/subclass와 dirty Header·base_path/requires_id 변경은 SDK-R1, Adapter cache invalidation은 SDK-C1, 동적 session/microversion/exception·transport kwargs는 SDK-S1의 별도 공통 범위입니다. 실제 cloud auth/policy/backend 실행과 Python runtime 전체의 동등성을 이 문서의 빌드·계약 검증으로 주장하지 않습니다.
