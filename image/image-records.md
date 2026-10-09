# Glance 이미지 레코드 조회·목록·검색: Python과 Go

`conn.Image(ctx)`의 `GetImageRecord`, `ListImageRecords`, `AllImageRecords`, `FindImageRecord`는 이미지 메타데이터를 고정 Python Image의 declared Resource로 제공합니다. SDK가 descriptor 기본값·별칭·변환, 확장 properties, 로컬 조건, 페이지 순회와 ID/이름 검색을 처리합니다. 이미지 데이터 다운로드는 별도 [Download API](download.md)입니다.

| Python `conn.image` | Go `image.Service` | 반환·동작 |
|---|---|---|
| `get_image(id_or_resource)` | `GetImageRecord(ctx, ImageRecordRequest{...}, options...)` | ID 경로 GET, seed를 새 record에 반영 |
| `images(**query)` | `ListImageRecords(ctx, options...)` | lazy `iter.Seq2[*ImageRecord, error]` |
| `list(images(**query))` | `AllImageRecords(ctx, options...)` | 같은 순회의 slice와 부분 결과 |
| `find_image(name_or_id, ignore_missing=True)` | `FindImageRecord(ctx, nameOrID, options...)` | GET → 일반 목록 → 숨김 목록, 유일한 ID/이름 일치 |

기존 `GetImage/ListImages/AllImages`의 `ImageInfo`, `service.Images`의 native Gophercloud Image와 `API.Images`의 호출은 유지합니다. 기존 typed 조회의 nullable·형태 검증, strict200와 페이지 정책은 [기존 조회 가이드](images.md)에 설명하며 이 record 계층의 Source 변환·범용 pager 정책과 구분합니다.

## Python 서비스·연결 사용

```python
import openstack

conn = openstack.connect(cloud="dev")
image = conn.image.get_image("image-id")
print(image.to_dict(), image.image_import_methods)

for image in conn.image.images(status="active", limit=10, max_items=20):
    print(image.to_dict())

image = conn.image.find_image("ubuntu", ignore_missing=False)
print(image.to_dict())
```

Python의 [공개 proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1013-L1054)는 `get_image`에 ID/Resource, `images`에 query, `find_image`에 이름/ID와 ignore_missing을 받습니다. Go의 concrete `With...` 옵션은 호출자가 request builder를 구현하지 않아도 같은 선택을 표현하도록 제공합니다. `FindImageRecord`의 별도 List 옵션은 공개 Python find signature보다 넓은 Go 확장이며 초기 GET의 query나 seed로 보내지 않습니다.

## 독립 Go main

[설치 안내](../docs/install.md)로 모듈을 준비한 뒤 아래 내용을 `main.go`에 저장합니다. 신규 ImageRecord API가 없는 과거 설치 pin 대신 이 가이드와 같은 revision 또는 API를 포함한 후속 revision을 사용합니다. `clouds.yaml`의 cloud와 실제 이미지 ID를 준비해 `go run . -cloud dev -image-id ID -find ubuntu`처럼 실행합니다. 문서 빌드는 실제 cloud 인증·권한 검증을 의미하지 않습니다.

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
    imageID := flag.String("image-id", "", "required image ID for literal GET")
    find := flag.String("find", "", "optional exact image name or ID")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, *find); err != nil {
        var response *resource.ResponseError
        if errors.As(err, &response) {
            fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
        }
        log.Fatal(err)
    }
}

func printRecord(value *image.ImageRecord) error {
    body, err := json.MarshalIndent(map[string]any{
        "resource": value.Resource, "wire": value.Wire,
        "status_code": value.StatusCode, "import_methods": value.ImportMethods,
    }, "", "  ")
    if err != nil { return err }
    fmt.Println(string(body))
    return nil
}

func run(ctx context.Context, cloud, imageID, nameOrID string) error {
    if imageID == "" { return fmt.Errorf("-image-id is required") }
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }

    value, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: imageID},
        image.WithImageRecordHeader("X-Request-Source", "image-records-example"))
    if err != nil { return err }
    if err := printRecord(value); err != nil { return err }

    for value, readErr := range service.ListImageRecords(ctx,
        image.WithImageRecordListFilter("status", "active"),
        image.WithImageRecordListLimit(10), image.WithImageRecordListMaxItems(20)) {
        if readErr != nil { return readErr }
        if err := printRecord(value); err != nil { return err }
    }

    if nameOrID != "" {
        found, err := service.FindImageRecord(ctx, nameOrID,
            image.WithFindImageRecordIgnoreMissing(false))
        if err != nil { return err }
        if err := printRecord(found); err != nil { return err }
    }
    return nil
}
```

이 main은 전체 연결에서 이미지 서비스를 얻는 예제입니다. 이미 준비한 Gophercloud client는 `image.New(client)`로 같은 API를 사용합니다. `NewWithDependencies(client, image.Dependencies{CloudLocation: getter})`는 현재 cloud 사실을 공급합니다. 호출·순회마다 현재 location을 옵션 callback 전에 한 번 읽고 복사하며 operation 중 변경한 Connection facts를 후속 page에 섞지 않습니다. dependency 없는 직접 New의 computed location은 null입니다.

예제는 Resource·Wire·상태와 import methods만 JSON으로 출력합니다. 빈/invalid JSON을 허용한 GET의 Envelope는 JSON이 아닐 수 있으므로 record 전체를 JSON으로 marshal할 때 raw bytes를 별도로 처리합니다. 1분 제한은 예제의 caller context이며 library가 설정한 기본 timeout이 아닙니다.

## 옵션·기본값

`ImageRecordRequest`는 `ID string`, `Resource *resource.RawResource`, `Attributes map[string]any`를 받습니다. ID와 Resource를 동시에 지정하지 않습니다. literal ID를 지정하면 request/option Attributes의 id key는 null·같은 값도 포함해 거부합니다. Resource 또는 Attributes에서 얻은 seed id는 후속 Attribute 옵션으로 교체할 수 있고 최종 literal id로 GET합니다. Resource 입력은 새 record의 seed로 복사하며 Python Resource 같은 객체를 변경하지 않습니다. Go seed는 Resource.Body → Request.Attributes → option Attributes를 raw name별로 overlay한 뒤 constructor packing을 한 번 수행합니다. unrelated unknown 속성은 유지하고 같은 raw key는 뒤 값이 교체합니다. 이는 명시적 Go 입력 정책이며 Python mutable Resource._update의 component 교체·dirty state를 재현한 것으로 해석하지 않습니다. ID/Find 문자열은 nonblank UTF-8이고 control 문자와 정확한 `.`·`..`를 거부합니다. 공백·slash·backslash를 포함한 허용 문자열은 원문을 유지해 단일 escaped URI segment로 보냅니다. Get은 이름 검색을 자동 수행하지 않습니다.

요청 identity로 읽는 raw JSON 문자열은 unpaired UTF-16 surrogate escape(예: `"\uD800"`)를 거부합니다. valid pair `"\uD83D\uDE00"`, 실제 U+FFFD와 `"\uFFFD"`, literal backslash-u 문자열 `"\\ud800"`는 각각의 정상 값을 보존합니다. `encoding/json`의 replacement character 치환으로 다른 ID를 선택하지 않으며, 이 규칙은 일반 Body descriptor 변환과 구분됩니다.

`ImageRecordOpts`는 `Headers map[string]string`, `Attributes map[string]any`를 제공합니다. `WithImageRecordOpts`는 설정 전체를 교체하며 `WithImageRecordHeader/Headers`, `WithImageRecordAttribute/Attributes`로 순서대로 값을 추가·교체합니다. Go map seed의 alias는 deterministic sorted key 순서로 처리하며 JSON 응답의 parsed insertion 순서와 구분합니다. 입력 RawResource는 컴포넌트를 가진 Python Resource의 same-instance fetch를 재현하는 타입이 아닙니다.

`ImageRecordListOpts`는 `Headers map[string]string`, `Filters map[string]json.RawMessage`, `Limit *int`, `Marker string`, `MaxItems int`, `Paginated *bool`를 제공합니다. `WithImageRecordListOpts`는 전체 설정을 교체하고 `WithImageRecordListHeader/Headers/Filter/Filters/Limit/Marker/MaxItems/Paginated`는 순서대로 적용합니다.

- 기본 목록 query는 없고 Accept는 application/json입니다. nil Limit·빈 Marker는 생략하며 explicit Limit0은 `limit=0`을 보냅니다.
- nil Paginated는 true, false는 첫 page까지만 읽습니다. MaxItems0은 무제한입니다.
- 양수 MaxItems는 필터 전 소비한 원본 행 수를 제한합니다. 초기 limit이 없거나 0이면 wire limit을 cap hint로 설정하며 양수 explicit limit은 유지합니다.
- input map·pointer·raw bytes와 option slice는 복사합니다. 각 순회에서 callback은 한 번 적용하고 재순회는 새 operation입니다.
- nil/error callback, 잘못된 identity·header·marker, 음수 limit/cap과 선택한 조건의 invalid JSON은 HTTP 전에 오류입니다. auth·framing·version·representation header를 일반 Header 옵션으로 덮어쓰지 못합니다.

`FindImageRecordOpts`는 `Headers map[string]string`, `IgnoreMissing *bool`, `List ImageRecordListOpts`를 제공합니다. `WithFindImageRecordOpts/Header/Headers/IgnoreMissing`과 `WithFindImageRecordListOptions(...ImageRecordListOption)`으로 선택합니다. nil IgnoreMissing은 true이고 `WithFindImageRecordIgnoreMissing(false)`는 끝까지 일치가 없으면 ErrNotFound를 반환합니다. List의 controls·filters는 두 fallback 목록에 적용하고 초기 GET에는 query를 전달하지 않습니다. nested List.Headers도 GET과 목록에 적용하며 같은 이름의 Find.Headers가 우선합니다. `WithFindImageRecordListOptions`의 callback은 preparation에서 각각 한 번 실행하고 callback 사이 source guard를 검사합니다.

## 서버 query와 로컬 조건

[Image query 선언](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L45-L63)의 `id`, `name`, `visibility`, `member_status`, `owner`, `status`, `size_min`, `size_max`, `protected`, `is_hidden`, `sort_key`, `sort_dir`, `sort`, `tag`, `created_at`, `updated_at`는 서버로 전달합니다. `is_hidden`만 `os_hidden`으로 옮기고 remote query spelling `os_hidden`도 허용합니다. 둘 다 같은 map에 있으면 canonical is_hidden이 우선하며 이후 단일 Filter helper는 해당 alias 선택을 교체합니다. 이는 [QueryParameters validation·transpose](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L301-L382)의 규칙입니다. limit·marker는 typed 옵션으로 지정합니다. `WithImageRecordListFilter`의 같은 이름을 SDK가 서버 query로 분류하므로 별도 builder가 필요하지 않습니다.

나머지 선언된 canonical Body 이름은 descriptor 변환 뒤 로컬 비교합니다. 예를 들어 `is_protected`, `owner_id`, `tags`, `properties`, `disk_format`, `size`는 로컬 조건이며 `protected`, `owner`, `tag`, `size_min`과 용도가 다릅니다. ordinary unknown 이름·선언되지 않은 remote spelling은 encoding 전에 무시합니다. raw map의 선택한 값은 JSON-domain query expansion과 반복 값을 유지하며 null은 생략합니다. boolean query의 True/False 표기는 Python Requests 경로를 따릅니다. Filter map에 `limit`, `marker`, `max_items`, `paginated`, `base_path`, `session`, `microversion`, `headers`, `jmespath_filters`를 넣으면 ErrUnsupported입니다. paging/header는 dedicated concrete 옵션을 쓰고 동적 base path/session/microversion/JMESPath를 이 호출에 섞지 않는 Go 경계입니다.

로컬 object 조건은 nonempty 실제 object에 recursive subset을 적용하고 scalar·array는 값 비교를 합니다. missing은 null로 비교하며 empty 실제 object는 object 조건에 맞지 않습니다. caller 조건 자체를 descriptor 타입으로 변환하지 않습니다. Go는 bool/number를 구분하고 숫자를 exact decimal로 비교하므로 Python의 `True == 1`, binary float rounding과 일부 nested 오류에 차이가 있습니다. 서버 조건의 유효성·가시성·정렬·default limit은 Glance가 판단합니다.

## 65필드 Resource와 실제 응답

`ImageRecord`는 `Resource`, `Wire *resource.RawResource`, `Envelope json.RawMessage`, `Header http.Header`, `StatusCode int`, `ImportMethods []string`를 제공합니다. Resource의64 Body 필드와 computed location은 [고정 Image 선언](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L79-L260), inherited id·tags와 location을 반영합니다. 선언하지 않은 wire 필드는 properties에 저장하며 arbitrary declared 필드를 새로 만들지 않습니다.

| 변환 | canonical 필드 |
|---|---|
| untyped 기본 속성 | `id`, `name`, `checksum`, `container_format`, `created_at`, `disk_format`, `hash_algo`, `hash_value`, `min_disk`, `min_ram`, `owner`, `owner_id`, `properties`, `store`, `status`, `updated_at`, `virtual_size`, `visibility`, `file`, `locations`, `direct_url`, `url` |
| untyped 공통 image 속성 | `architecture`, `hypervisor_type`, `instance_uuid`, `needs_config_drive`, `kernel_id`, `os_distro`, `os_version`, `needs_secure_boot`, `ramdisk_id`, `vm_mode`, `hw_disk_bus`, `hw_cpu_policy`, `hw_cpu_thread_policy`, `hw_rng_model`, `hw_machine_type`, `hw_scsi_model`, `hw_video_model`, `hw_watchdog_action`, `os_command_line`, `hw_vif_model`, `vmware_adaptertype`, `vmware_ostype`, `has_auto_disk_config`, `os_type`, `os_admin_user`, `schema` |
| nullable bool | `is_hidden`, `is_protected`, `is_hw_boot_menu_enabled`, `os_require_quiesce` |
| nullable int | `size`, `os_shutdown_timeout`, `hw_cpu_sockets`, `hw_cpu_cores`, `hw_cpu_threads`, `hw_serial_port_count`, `hw_video_ram` |
| nullable float·dict·str·BoolStr | `instance_type_rxtx_factor`, `metadata`, `hw_qemu_guest_agent`, `is_hw_vif_multiqueue_enabled` 각각 |
| list·computed | `tags`, `location` 각각 |

untyped 생략/null은 null이고 다른 JSON 값·큰 숫자·nested 값은 보존합니다. bool은 nonnull Python truthiness를 사용하므로 nonempty string `"false"`도 true입니다. int는 bool을 보존하고 숫자를 정수로 절삭하며 digit 문자열을 변환하고 나머지는0으로 처리합니다. float는 유한 값만 허용하고 invalid numeric string은 오류입니다. metadata는 nonnull nonobject를 `{}`로 바꿉니다. tags는 기본 `[]`, explicit null은 null, 배열은 임의 요소를 유지하고 nonarray는 singleton list입니다. BoolStr은 string 표현을 lower해 정확한 true/false만 허용하며 whitespace를 trim하지 않습니다. str 변환은 JSON dump가 아닌 Python 형태의 scalar/container 표현입니다. string은 원문을 유지하고 container는 insertion 순서의 Python repr을 사용합니다. float spelling은 Go의 shortest float64와 Python의 fixed/scientific 경계를 사용하며 정확한 Python float repr·Unicode database version은 비교 한계입니다. 이 규칙과 finite JSON·숫자 범위는 [descriptor 구현](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86-L128)의 동작을 확인한 Go 범위에 적용합니다.

별칭은 `os_hidden→is_hidden`, `protected→is_protected`, `os_hash_algo/value→hash_algo/value`, `img_config_drive→needs_config_drive`, `os_secure_boot→needs_secure_boot`, `hw_vif_multiqueue_enabled→is_hw_vif_multiqueue_enabled`, `hw_boot_menu→is_hw_boot_menu_enabled`, `auto_disk_config→has_auto_disk_config`입니다. `owner`와 `owner_id`는 같은 owner component의 두 view입니다. canonical/remote spelling이 함께 있으면 parsed dictionary insertion 순서의 나중 key가 우선하며 중복 key의 마지막 값은 첫 insertion 위치를 유지합니다.

[properties packing](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1221-L1230)은 explicit properties object에 unknown attrs를 overlay하고 nonobject properties는 `{"properties": value}`로 감싼 뒤 unknown을 추가합니다. consumed canonical/remote 필드는 properties에 중복하지 않습니다. `self`는 제거하고 case-decoy는 ordinary unknown 속성으로 유지합니다. 응답의 `location`도 properties에 들어가며 declared location은 captured current Connection location입니다. owner를 project_id로 해석하거나 URI·schema·location으로 후속 요청을 만들지 않습니다. caller Attributes/RawResource.Body의 location도 unknown property로 취급하며 입력 Python Resource의 computed component를 그대로 재사용하는 동작과 구분합니다.

Literal GET의 id는 seed이며 응답 id 생략은 seed를 유지하고 explicit null도 seed를 교체합니다. List에는 alternate ID가 없으므로 name-only 행의 id는 null입니다. GET valid `{}`는 properties `{}`를 만들지만 empty/invalid JSON은 translation을 건너뛰므로 기본 properties null을 유지합니다. 반면 List의 `{}` 또는 known fields만 있는 행은 unknown packing이 없으면 properties null을 유지합니다. 이 차이는 [fetch translation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1356-L1394)과 [constructor collector](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L808-L839)의 차이입니다.

`ImportMethods`는 Resource descriptor가 아닌 별도 ordinary 속성입니다. GET은 실제 OpenStack-image-import-methods 헤더를 case-insensitive로 읽고 반복 HTTP 값을 comma+space로 합친 뒤 comma로 split합니다. trim하지 않으므로 앞 공백·빈 항목도 유지합니다. List는 실제 page header를 각 행에 적용하지 않으며 행의 정확한 `OpenStack-image-import-methods` key가 constructor에서 소비될 수 있습니다. 이 값은 falsey이면 빈 slice, truthy string이면 comma split이며 truthy nonstring은 오류입니다. unknown `image_import_methods` Body key는 descriptor로 승격하지 않습니다. 원래 header·Body 값은 Header·Wire에 남습니다. [고정 header 소비](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L429-L437)

Resource·Wire·Envelope·Header와 각 행의 raw bytes는 독립 복사합니다. GET Wire에는 요청 seed를 합성하지 않으며 List의 Envelope는 실제 전체 page body입니다. 반환값을 변경해 다음 page target이나 다른 행의 evidence를 바꾸지 못합니다.

## 응답·페이지·부분 결과

Get은 actual HTTP200..399를 받아 fresh seed에 Body attrs를 overlay합니다. empty/invalid JSON은 기본 record와 실제 receipt를 반환하고 Wire는 nil입니다. valid JSON null·array·scalar, invalid UTF-8과 descriptor 실패는 처리 오류입니다. accepted 응답 처리 오류는 bounded body·실제 status/header를 가진 ResponseError와 nil record를 반환하며 partial Resource를 성공으로 반환하지 않습니다.

List는 valid UTF-8의 nonnull object와 `images` key를 요구합니다. 배열 외 single row object도 generic Source처럼 받습니다. 소비하는 행은 nonnull object이며 `connection`, `microversion`, `_synchronized`는 Source constructor의 중복 인자 오류에 대응해 거부합니다. break/cap 뒤의 미소비 행은 projection하지 않지만 전체 page JSON·UTF-8은 먼저 검증합니다.

Body links·images_links·next, HTTP Link와 초기 양수 limit의 마지막 소비 id marker fallback을 처리합니다. top-level falsey next는 다른 continuation을 허용하며 truthy nonstring next는 오류입니다. `links:{next:"URL"}`는 Go 확장입니다. short nonempty page도 Source처럼 이어가고 empty page는 끝납니다. initial limit/max-items hint가 없는 목록은 후속 link가 준 limit만으로 fallback을 새로 활성화하지 않습니다. marker fallback에는 마지막 소비 행의 nonblank string id가 필요하며 name이나 wire URL로 marker를 합성하지 않습니다. local filter가 제외한 행도 cap·marker에 기여합니다.

계속 요청할 endpoint·route를 고정하고 같은 origin의 승인된 versioned alias만 허용합니다. cross-origin·다른 route·반복 marker·cycle은 terminal 오류입니다. cap 또는 caller break 뒤 continuation을 검증·fetch하지 않습니다. AllImageRecords는 오류 전 이미 반환한 행과 오류를 함께 돌려주므로 caller가 partial slice 사용 여부를 결정합니다. 기본 query·반복 값은 Source pager를 따라 처리하며 기존 typed ListImages의 collapsed query 복원 정책을 이 API의 보장으로 옮기지 않습니다.

context 취소와 captured client/provider/endpoint·identity source 변경은 operation의 sticky guard로 관찰합니다. native 재인증·retry callback을 유지하고 accepted 응답의 Read/Close·projection 실패를 보존합니다. 보호된 source·header와 응답 크기·finite JSON 경계는 Go의 명시적 계약이며 Python mutable Resource/session 전체 동작의 동등성을 뜻하지 않습니다.

## Find의 세 단계

[고정 Image.find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L494-L531)와 [Resource.find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2524-L2582)의 순서입니다.

1. `GET /images/{nameOrID}`를 단일 escaped segment로 요청합니다. actual200..399의 seed/default record를 반환하면 검색을 끝내고 응답 ID와 입력이 다르다는 이유로 재검색하지 않습니다.
2. clean 최종 native400·403·404일 때 일반 List를 순회합니다. List 옵션에 name이 없으면 첫 검색에만 `name=nameOrID`를 자동 추가합니다.
3. 일반 List가 정상 종료해 일치가 없을 때만 원래 List 옵션에 `is_hidden=True`를 설정해 `os_hidden=True` 목록을 순회합니다. 첫 검색에서 자동 추가한 name을 넘기지 않습니다. caller가 직접 지정한 name 조건은 유지합니다.

일치는 Resource id 또는 name의 정확한 문자열 비교입니다. 같은 행이 둘 다 맞으면 한 후보이며 두 번째 후보를 만나면 즉시 ErrAmbiguous를 반환합니다. 유일한 후보를 확인하려면 순회를 끝내야 합니다. 후보 뒤에 page·projection·continuation 오류가 발생하면 nil record와 오류를 반환하며 hidden 검색으로 넘어가지 않습니다. 목록에서 찾은 행을 다시 GET하지 않고 그 page의 실제 receipt를 반환합니다.

선택한 GET400·403·404에는 native rejected body의 Read/Close·source 실패 관찰을 추가해 faulty rejection 뒤 List로 넘어가지 않습니다. native retry/재인증이 먼저 실행되며 hook·transport·source·context·body 원인이 추가된 rejection은 fallback하지 않습니다. RetryFunc가 같은 plain native input error를 그대로 반환하면 추가 원인으로 만들지 않습니다. 다른 rejected GET status와 List rejection은 기존 native Gophercloud I/O 정책을 유지합니다. 모든 rejection의 I/O 정책을 교체한 것으로 해석하지 않습니다.

기본 미존재는 `nil, nil`이고 IgnoreMissing false는 ErrNotFound입니다. 거부·visibility 때문에 접근 가능한 결과가 없을 수도 있으므로 최종 nil/ErrNotFound가 cloud 전체의 실제 부재를 증명하지 않습니다. Find의 explicit List cap·single-page·local 조건은 선택한 탐색 범위를 줄이는 Go 확장입니다.

## 권한·검증 범위

[공식 Glance 기본 정책](../docs/glance-policy-priorities.md#이미지-조회목록검색의-기본-정책)은 get_image와 get_images에 project reader 경로를 허용하므로 이 작업을 핵심 user에 배치합니다. 목록 controller는 get_images 검사 후 각 행의 get_image 정책도 검사합니다. SDK가 role preflight나 server policy를 재구현하지 않으며 실제 auth·소유권·visibility·override는 서버가 결정합니다.

이 계층은 고정 Source의 literal/fresh record·descriptor·목록·검색을 구현하는 범위입니다. Python mutable Resource same-instance/dirty state, configured GET cache, dynamic Adapter/session·microversion·custom Resource/Munch, deprecated jmespath_filters와 임의 parser 범위는 별도 비교 대상입니다. generic Resource.find의 arbitrary kwargs를 초기 seed/GET에 전달하는 동작을 public proxy find의 완료 범위로 확대하지 않습니다. Go는 각 호출에 새 record를 반환하고 configured Python response cache를 재현하지 않습니다.

GET의65필드 projection·packing·seed·실제 receipt와 local/IO 경계는 [GET record 테스트](image_record_get_test.go), query/local 분류·lazy 순회·raw cap·paging·partial·constructor 오류는 [목록 record 테스트](image_record_lists_test.go)에 계약을 둡니다. [Find record 테스트](image_record_find_test.go)는 일반/숨김 query 분리·중복·late 실패·선택한 rejected I/O를, [Connection 테스트](../connection_image_records_test.go)는 서비스 생성과 현재 location을 확인합니다. [공통 Python string 테스트](../internal/cloudfilter/python_string_test.go)는 scalar/container 표현의 범위를 다룹니다. 기존 image/task/member transport·body fixtures와 공개 Gophercloud testhelper를 재사용하며 신규 HTTP mock 서버나 별도 fault 엔진을 만들지 않습니다. 여기서 설명한 계약 테스트와 문서 빌드는 실제 cloud 또는 Python runtime 실행을 대신하지 않습니다.
