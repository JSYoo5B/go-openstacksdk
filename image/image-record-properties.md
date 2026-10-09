# Glance 이미지 속성 갱신: Python과 Go

`Connection.UpdateImagePropertiesRecord`와 `image.Service.UpdateImagePropertiesRecord`는 SDK의 ImageRecord를 받아 기존 custom properties를 보존하고, 값 변환·kernel/ramdisk 이미지 검색·meta overlay·변경 비교·자동 PATCH를 처리합니다. 호출자는 builder나 참조 이미지 resolver를 작성하지 않고 concrete `With` 옵션을 전달합니다.

대상 이미지는 이름으로 자동 조회하지 않습니다. 먼저 `service.GetImageRecord`로 레코드를 가져온 뒤 `ImageRecordPropertiesRequest{Record: record}`를 전달하는 사용법을 권장합니다. literal ID도 입력 형태로 받지만 Python의 ID constructor에는 복사할 properties가 없어 helper가 실패합니다. Go도 이 경로에 detail GET을 자동 추가하지 않고 오류를 반환합니다. truthy kernel/ramdisk 입력이 있으면 그 목록 조회가 properties 복사 실패보다 먼저 일어날 수 있습니다.

| 작업 | Python | Go |
|---|---|---|
| 대상 명시적 조회 | `record = conn.image.get_image(id)` | `record, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: id})` |
| 변환하는 속성 | `conn.image.update_image_properties(record, team="platform", min_ram="512")` | `UpdateImagePropertiesRecord(ctx, request, image.WithImageRecordProperties(...))` |
| 변환을 건너뛰는 overlay | `meta={"example_enabled": False}` | `image.WithImageRecordPropertyMeta("example_enabled", false)` 또는 `WithImageRecordPropertiesMetaJSON` |
| 참조 이미지 해석 | `kernel="linux-*"` | `image.WithImageRecordProperty("kernel", "linux-*")` |

기존 [SetImageProperties](update.md)는 정렬한 key별 `add` PATCH를 보내고 `ImageInfo`를 반환하는 별도 API입니다. 이 helper의 기존 properties 복사·변환·전체 목록 검색·bool/no-op 동작을 수행하지 않습니다. 고정 Gophercloud `images` 패키지에 이 helper의 native 선언은 없으므로 `API.Images.UpdateImageProperties` 같은 호출은 제공하지 않습니다.

## Python 사용

```python
import openstack

conn = openstack.connect(cloud="dev")
record = conn.image.get_image("image-id")
updated = conn.image.update_image_properties(
    record,
    team="platform",
    min_ram="512",
    meta={"example_enabled": False},
)
print(updated)
```

`updated`는 갱신 helper가 nonempty property map을 `update_image`에 넘겨 성공했는지를 나타냅니다. 같은 값만 남아 있으면 `True`여도 HTTP를 생략할 수 있습니다. 서버 저장·필드 변경·PATCH 전송 여부를 나타내는 boolean이 아닙니다. 반환값이 `False`이면 helper가 Update를 호출하지 않습니다.

## 독립 Go main

[설치 안내](../docs/install.md)를 따라 아래 코드를 `main.go`로 저장합니다. `go run . -cloud dev -image-id ID`는 대상 레코드를 명시적으로 조회하고 속성 helper를 호출합니다. `-service`는 같은 작업을 Service 메서드로 실행하며 Connection 메서드를 추가로 호출하지 않습니다.

`-properties '{"kernel":"linux-*","min_ram":"512"}'`는 kernel 후보를 전체 이미지 목록에서 찾습니다. `-meta '{"example_enabled":false}'`는 raw boolean을 전달합니다. 아래 예제의 기본 입력은 team·min_ram·example_enabled를 수정할 수 있으므로 사용할 이미지와 값을 정해 실행합니다. 문서 검증은 외부 consumer 빌드이며 실제 cloud 인증·수정을 실행한 검증은 아닙니다.

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
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    imageID := flag.String("image-id", "", "required literal image ID")
    properties := flag.String("properties", `{"team":"platform","min_ram":"512"}`, "ordered JSON keyword object")
    meta := flag.String("meta", `{"example_enabled":false}`, "raw JSON overlay object")
    viaService := flag.Bool("service", false, "call image.Service instead of Connection")
    flag.Parse()
    if *imageID == "" { log.Fatal("-image-id is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, json.RawMessage(*properties), json.RawMessage(*meta), *viaService); err != nil {
        var native gophercloud.ErrUnexpectedResponseCode
        if errors.As(err, &native) {
            fmt.Fprintf(os.Stderr, "native HTTP %d, response bytes=%d\n", native.Actual, len(native.Body))
        }
        var response *resource.ResponseError
        if errors.As(err, &response) {
            fmt.Fprintf(os.Stderr, "accepted HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
        }
        log.Fatal(err)
    }
}

func run(ctx context.Context, cloud, imageID string, properties, meta json.RawMessage, viaService bool) error {
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    record, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: imageID})
    if err != nil { return err }
    input := image.ImageRecordPropertiesRequest{Record: record}
    options := []image.ImageRecordPropertiesOption{
        image.WithImageRecordPropertiesOpts(image.ImageRecordPropertiesOpts{
            Headers: map[string]string{"X-Request-Source": "image-record-properties-example"},
            Properties: properties,
            Meta: meta,
        }),
        image.WithImageRecordPropertiesHeader("X-SDK-Example", "owned-properties"),
    }
    var result *image.ImageRecordPropertiesResult
    if viaService {
        result, err = service.UpdateImagePropertiesRecord(ctx, input, options...)
    } else {
        result, err = conn.UpdateImagePropertiesRecord(ctx, input, options...)
    }
    value := map[string]any{"partial": result != nil && err != nil}
    if result != nil {
        value["updated"] = result.Updated
        if result.Record != nil {
            value["record"] = map[string]any{
                "resource": result.Record.Resource,
                "wire": result.Record.Wire,
                "receipt_status_code": result.Record.StatusCode,
                "receipt_body_bytes": len(result.Record.Envelope),
                "import_methods": result.Record.ImportMethods,
            }
        }
    }
    if err != nil { value["error"] = err.Error() }
    body, marshalErr := json.MarshalIndent(value, "", "  ")
    if marshalErr != nil { return errors.Join(err, marshalErr) }
    fmt.Println(string(body))
    return err
}
```

출력은 두 칸 들여쓰기 JSON입니다. 1분 context는 caller가 정한 인증·명시적 GET·참조 검색·속성 갱신 전체 예산입니다. 새로운 HTTP가 없는 결과의 StatusCode·Header·Envelope는 이전 receipt를 유지합니다. nonnil 결과나 `Updated`만 확인하지 말고 항상 `err`를 확인합니다. 반환한 Record를 다음 호출에 전달하며 입력 Record의 public 필드를 직접 바꿔 변경을 제출하지 않습니다.

## 입력과 concrete 옵션

`ImageRecordPropertiesRequest`는 `ID string` 또는 `Record *ImageRecord` 중 하나를 받습니다. Record는 `GetImageRecord`, owned 목록·검색·수정·태그 등의 SDK 반환값이어야 합니다. public Resource만 직접 만든 Record에는 private commit baseline이 없어 거부합니다. 둘을 함께 지정하거나 빈 selector를 전달하면 local 오류입니다. ID는 [owned 수정](image-record-update.md)의 UTF-8·단일 escaped segment 규칙을 따릅니다.

`ImageRecordPropertiesOpts`는 일반 `Headers map[string]string`, 변환 전 `Properties json.RawMessage`, 변환을 건너뛰는 `Meta json.RawMessage`를 받습니다. Properties와 Meta의 기본값은 빈 object이며 raw object는 멤버 순서를 유지합니다. Meta의 null·false·0·빈 string·빈 array 같은 falsy JSON도 빈 object로 처리합니다. truthy Meta는 JSON object여야 합니다. 일반 Headers는 참조 목록의 각 페이지와 최종 PATCH에 적용합니다.

| 옵션 | 입력·동작 |
|---|---|
| `WithImageRecordPropertiesOpts` | Headers·Properties·Meta 전체 설정 교체 |
| `WithImageRecordPropertiesHeader` / `WithImageRecordPropertiesHeaders` | 일반 요청 헤더 하나 / map |
| `WithImageRecordProperty(name, value)` | 변환할 keyword 하나; `any`를 Go JSON으로 캡처 |
| `WithImageRecordProperties(map[string]any)` | 정렬한 map key를 기존 keyword object에 upsert; 기존 key의 첫 위치 유지 |
| `WithImageRecordPropertiesJSON(json.RawMessage)` | keyword object 전체 교체; raw 멤버 순서 유지 |
| `WithImageRecordPropertyMeta(name, value)` | 변환 없이 반영할 meta 하나 |
| `WithImageRecordPropertiesMetaJSON(json.RawMessage)` | meta object 전체 교체; raw 멤버 순서 유지 |

옵션은 호출 순서대로 적용합니다. 단일 Property/PropertyMeta는 해당 object의 key를 upsert하고, 기존 key는 첫 위치를 유지하면서 마지막 값으로 바꿉니다. bulk Properties map도 정렬한 incoming key를 같은 규칙으로 병합하므로 앞선 key를 보존합니다. PropertiesJSON은 keyword object를, MetaJSON은 meta object를 전체 교체하므로 앞선 단일 key도 제거할 수 있습니다. WithImageRecordPropertiesOpts는 Headers·Properties·Meta 설정 전체를 교체합니다. raw object의 duplicate key도 첫 위치와 마지막 값을 사용합니다. map·slice·raw bytes·option slice와 private Record를 복사하므로 옵션 생성 후 caller 값을 변경해도 이미 캡처한 입력은 바뀌지 않습니다. header만 추가해서 no-op을 PATCH로 강제하지 않습니다. nil/error 옵션·encoding 오류·invalid header·보호된 auth·framing·version·representation override는 오류입니다.

준비 단계는 private current body를 descriptor-backed canonical view로 materialize하고 현재 location을 적용하며 plain ImportMethods를 `[]`로 reset합니다. 이 실패는 kernel/ramdisk 조회나 PATCH보다 앞섭니다. public `Resource`, `Wire`, `Envelope`, 이전 header를 편집해 target ID나 custom property seed를 바꾸지 못합니다. pending raw body를 이때 clean하거나 서버에 동기화되었다고 가정하지 않습니다.

## keyword 변환과 meta 우선순위

| keyword | 변환 |
|---|---|
| `min_disk`, `min_ram`, `size`, `virtual_size` | Python `int(v)`에 대응하는 helper 변환 |
| `is_protected`, `protected`, `tags` | raw JSON 유지 |
| 나머지 nonnull 값 | 전체 값을 Python `str(v)`에 대응하는 문자열로 변환 |
| 나머지 null | null 유지 |
| meta의 모든 값 | 위 변환 완료 후 raw overlay |

helper의 integer 변환은 Resource getter의 descriptor 변환과 다릅니다. boolean은 1/0, finite float는 소수 부분을 버린 정수이며 큰 integer token은 값을 보존합니다. signed·공백·underscore·Unicode decimal 문자열을 처리합니다. null·container·잘못된 문자열·표현 범위를 벗어난 수치 변환은 PATCH 전에 오류입니다. meta가 같은 key를 덮어쓸 예정이어도 잘못된 keyword의 변환 오류를 생략하지 않습니다.

ordinary 값은 `false`가 `"False"`로, 중첩 배열·object도 전체 Python 표현에 대응하는 문자열로 바뀝니다. 일부 요소만 JSON으로 변환하는 방식이 아닙니다. `[]byte`와 사용자 marshaler 등 `any` 입력은 먼저 Go `encoding/json`의 값으로 캡처합니다. exact raw JSON과 큰 숫자가 필요하면 raw object 또는 `json.RawMessage`를 사용합니다. 임의 Python object의 `__str__`, 사용자 mapping·iterable, 모든 Python float/Unicode runtime 표현을 재현하는 계약은 아닙니다.

meta는 keyword 변환 이후 같은 key를 덮어씁니다. keyword `{"example_enabled":false}`는 문자열 `"False"`, meta `{"example_enabled":false}`는 raw boolean을 전달합니다. meta의 `kernel`·`ramdisk`는 참조 검색을 하지 않습니다.

## kernel·ramdisk 참조 검색

keyword의 원래 key가 `kernel` 또는 `ramdisk`이고 값이 Python JSON truthiness에서 truthy이면 SDK가 참조 이미지를 찾아 각각 `kernel_id`·`ramdisk_id`로 바꿉니다. false·null·0·빈 string·빈 array/object는 검색하지 않고 원래 key를 ordinary 변환합니다. 검색 결과가 없으면 null을 사용하며 별도 `ErrNotFound`나 ambiguous 오류를 합성하지 않습니다.

각 참조 검색은 일반 owned 이미지 목록의 모든 페이지를 읽고, status를 소문자로 바꿨을 때 `deleted`인 행을 제외한 뒤 exact ID/name 또는 Python fnmatch pattern을 적용합니다. 유지한 행의 순서대로 첫 일치를 선택합니다. 두 참조가 있으면 각각 별도로 전체 목록을 읽으며 cache를 공유하지 않습니다. status가 string이 아니거나 뒷페이지에 오류가 있으면 앞페이지에 일치가 있었어도 helper를 중단합니다.

조회 행의 status는 완전한 UTF-8 JSON string이며 escaped UTF-16 surrogate는 올바른 쌍이어야 합니다. Go는 lone surrogate를 거부하므로 Python의 임의 string `.lower()` 동작보다 엄격한 경계입니다. 같은 strict identity 규칙은 direct/meta `id`를 dirty 비교하기 전에도 적용합니다.

이 검색은 GET-first·exact uniqueness·hidden retry를 사용하는 `FindImageRecord`와 다릅니다. 대상 detail GET, 자동 name query, hidden-image fallback, 중복 거부를 추가하지 않습니다. 모든 페이지를 수집한 뒤 선택하므로 목록을 읽는 도중 첫 일치에서 멈추지 않습니다. keyword `kernel`과 직접 `kernel_id`가 함께 있으면 실제 처리 순서의 마지막 값이 이기며, 순서를 지정하려면 raw Properties object를 사용합니다.

## 기존 properties와 bool·PATCH

helper는 private current의 전체 cached properties를 복사해 시작합니다. null·missing·scalar처럼 `.copy()`할 수 없는 seed는 오류입니다. JSON array seed는 Source의 특이한 경계를 유지합니다. 빈 array와 변경 없는 입력은 False일 수 있고 string-key 삽입이나 nonempty array의 keyword 확장은 오류입니다. 이 shape 오류에 앞서 truthy kernel/ramdisk 검색이 실행될 수 있습니다.

변환과 meta overlay 결과는 fresh canonical descriptor-backed `image.get(k, None)` 값과 비교합니다. cached custom properties나 flat Wire의 값을 대신 비교하지 않습니다. bool/numeric의 Python 동등성과 object·array 재귀 비교를 사용합니다. 예를 들어 custom `foo`가 cached properties 안에만 있으면 canonical `get("foo", None)`은 null입니다. `foo=None`을 지정해도 기존 cached foo를 제거하지 않을 수 있습니다.

| 준비된 상태 | 성공 결과·HTTP |
|---|---|
| cached properties가 비고 새 변경도 없음 | `Updated:false`; Update/PATCH 없음 |
| cached properties가 nonempty이며 새 변경 없음 | `Updated:true`; clean component이면 PATCH 없음 |
| canonical 값과 다른 keyword/meta | 전체 seed를 shared Update에 전달; dirty이면 자동 PATCH |
| 이전 호출에서 unrelated pending body가 남아 있고 prepared properties는 nonempty | pending body도 함께 commit할 수 있음 |
| dirty flag는 남았지만 original/current diff는 비어 있음 | `[]` PATCH를 제출할 수 있음 |

기존 custom keys를 보존한 전체 seed를 [UpdateImageRecord](image-record-update.md)의 준비·commit 경로에 전달합니다. cached key가 declared canonical/wire alias와 같으면 일반 Body 필드로 승격되고 unknown keys는 properties로 pack합니다. nested `properties`와 별칭의 우선순위도 같은 shared 규칙을 사용하므로 deep merge나 key별 add PATCH로 바꾸지 않습니다.

direct ID dirtiness는 commit gate에서 제외하지만 private current ID는 유지합니다. raw meta의 `id`로 새 ID를 넣고 다른 dirty 변경이 있으면 새 경로에 PATCH할 수 있습니다. id만 바뀌고 다른 dirty 변경이 없으면 HTTP를 생략할 수 있습니다. 실제 readonly field 수정 허용 여부는 서버가 판단합니다.

Go는 고정 route/source를 유지하며 bound argument·constructor·routing/runtime 제어와 충돌하는 direct keys를 local 오류로 처리합니다. `image`, `self`, `resource_type`, `value`, `base_path`, `microversion`, `connection`, `_synchronized` 등을 임의 URL·session·version 설정으로 소비하지 않습니다. 이 fixed-v2 owned profile은 Python의 동적 Resource·Proxy argument collision이나 Adapter 설정 전체와 구분됩니다.

## 응답·부분 오류·호출 소유권

HTTP가 필요하면 shared Update가 `PATCH /images/{escaped-id}`를 사용합니다. Content-Type은 `application/openstack-images-v2.1-json-patch`, Accept는 빈 값이며 accepted profile은 actual `200..399`입니다. `400..599`는 native HTTP 오류와 실제 응답 증거를 유지합니다. helper 내부의 자동 detail GET·polling은 없습니다. 실제 PATCH operation의 diff 순서·wire alias·property flattening·response overlay와 pending baseline은 [owned 수정 가이드](image-record-update.md)를 따릅니다.

준비·참조 목록·properties 복사·변환 실패와 accepted 응답이 없는 transport/native 거부는 nil result와 오류를 반환합니다. 목록 조회가 먼저 일어났다고 갱신 성공을 합성하지 않습니다. accepted PATCH의 Read/Close·context/source·decode/projection 실패에는 `Updated:false`와 가능한 partial Record를 오류와 함께 반환합니다. 이 Record에는 submitted 상태와 실제 body·header·status 증거가 남을 수 있으며 서버는 이미 요청을 접수했을 수 있습니다.

정상 object 응답은 shared translation으로 current 값을 overlay하고 baseline을 clean합니다. 빈 body나 invalid JSON syntax는 tolerated 성공으로 pending 상태를 남길 수 있습니다. 따라서 성공 `Updated:true`가 baseline이 clean하다는 뜻도 아닙니다. 같은 반환 Record를 다시 전달하면 caller가 명시적으로 pending PATCH를 재시도하는 경로가 될 수 있습니다. accepted 처리를 복구하려고 PATCH를 자동 replay하지 않습니다.

호출 전체는 하나의 캡처한 Glance v2 source·provider·route·header·location과 sticky guard를 사용합니다. 참조 조회 뒤 다른 mutable 서비스로 다시 source를 선택하지 않습니다. callback·retry·body 처리 중 source/context가 바뀌면 원인 오류를 유지하며 중단합니다. 원래 provider의 live 인증·정상 native retry 정책을 유지하되 고정 method/route/representation과 입력 소유권을 지킵니다. `errors.As`로 `gophercloud.ErrUnexpectedResponseCode`와 `resource.ResponseError`, `errors.Is`로 context·custom cause를 확인할 수 있습니다.

## 권한·비교 범위

고정 Glance pin `57f7dd9e76ef24e1e9013eceaa703bd442469a24`의 [property policy213–229행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/policy.py#L213-L229)은 일반 속성에 `modify_image`를 적용합니다. [modify 정책80–93행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L80-L93)의 project scope `ADMIN_OR_PROJECT_MEMBER`에 따라 일반 property 갱신은 핵심 user 경로입니다. public visibility는 [publicize 정책94–103행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L94-L103), location removal은 [delete location 정책159–172행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L159-L172)의 admin 분기입니다. 실제 role·소유권·visibility·property protection·readonly/schema·policy override·backend 허용 여부는 서버가 판단합니다. [권한 근거](../docs/glance-policy-priorities.md)를 함께 참고하세요.

고정 openstacksdk revision `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [public helper1139–1179행](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1139-L1179), [변환159–176행](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L159-L176), [cloud 목록·검색84–120행](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L84-L120), [첫 참조 ID 선택221–246행](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L221-L246)을 대조한 JSON owned profile입니다. SDK가 properties·meta와 resolver를 소유하지만 Python의 arbitrary Resource/subclass·공유 mutable lifecycle·동적 Adapter/session/version·모든 custom object와 Python runtime 표현은 공통 SDK-R1/C1/S1 비교 범위로 남습니다. Go는 입력을 직접 mutate하지 않고 독립 결과를 반환하며 exception 후 부분 제출 Record를 반환할 수 있습니다.

local HTTP/fault fixture와 공개 Gophercloud testhelper의 재사용, Source 검토와 외부 main 빌드는 Python runtime 실행·실제 cloud 인증·서버 API 검증과 별개입니다. 현재 검증 결과와 남은 operation·공통 계약은 [지원 판정대장](../docs/sdk-support-ledger.md)에서 확인합니다.
