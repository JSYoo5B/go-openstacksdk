# Cloud 이미지 목록·검색·조회: Python과 Go

`conn.Image(ctx)`의 `AllCloudImageRecords`, `SearchImageRecords`, `GetCloudImageRecord`, `GetImageRecordByID`는 고정 openstacksdk Cloud 계층의 `list_images`, `search_images`, `get_image`, `get_image_by_id`에 대응합니다. `GetImageRecordExclude`, `GetImageRecordName`, `GetImageRecordID`는 같은 검색 결과를 쓰는 `get_image_exclude`, `get_image_name`, `get_image_id`에, `WaitForCloudImageRecord`는 `wait_for_image`에, `DownloadCloudImageRecord`는 `download_image`에, `UpdateCloudImageProperties`는 `update_image_properties`에, `DeleteCloudImageRecord`는 `delete_image`에 대응합니다. SDK가 기존 [ImageRecord 조회·목록·Find](image-records.md) 엔진과 Connection location, 공통 Cloud 필터를 조합하므로 애플리케이션이 builder interface나 페이지 순회를 구현하지 않아도 됩니다. Proxy 계층의 lazy `ListImageRecords`, `FindImageRecord`, `GetImageRecord`는 그대로 유지합니다.

| 고정 openstacksdk | Go | 실제 기본값과 순서 |
|---|---|---|
| `conn.list_images(filter_deleted=True, show_all=False)` | `service.AllCloudImageRecords(ctx, options...)` | 기본 이미지 목록 전체를 eager 수집한 뒤 status가 소문자 `deleted`인 행 제외 |
| `conn.search_images(name_or_id=None, filters=None)` | `service.SearchImageRecords(ctx, nameOrID, options...)` | 기본 `list_images`를 끝낸 뒤 이름/ID glob과 dictionary 또는 JMESPath 선택 |
| `conn.get_image(name_or_id, filters=None)` | `service.GetCloudImageRecord(ctx, nameOrID, options...)` | filters가 None이면 Proxy `find_image`, 그 밖에는 검색 후 truthiness·`len`·`[0]` |
| `conn.get_image_by_id(id)` | `service.GetImageRecordByID(ctx, id, options...)` | literal GET 한 번; 404는 nil이 아닌 원래 오류 |
| `conn.get_image_exclude(name_or_id, exclude)` | `service.GetImageRecordExclude(ctx, nameOrID, exclude, options...)` | `search_images(name_or_id)` 순서대로 name에 exclude가 없는 첫 행 |
| `conn.get_image_name(image_id, exclude=None)` | `service.GetImageRecordName(ctx, imageID, exclude, options...)` | 위 선택 행의 raw `name` |
| `conn.get_image_id(image_name, exclude=None)` | `service.GetImageRecordID(ctx, imageName, exclude, options...)` | 위 선택 행의 raw `id` |
| `conn.wait_for_image(image, timeout=3600)` | `service.WaitForCloudImageRecord(ctx, record, options...)` | 2초 간격으로 Proxy find 반복; 정확한 `active` 성공, 정확한 `error` 실패 |
| `conn.download_image(name_or_id, output_path=None, output_file=None, chunk_size=1MiB, stream=False)` | `service.DownloadCloudImageRecord(ctx, request, options...)` | 출력 하나를 HTTP 전에 확인; `find_image(ignore_missing=False)` 뒤 Proxy download |
| `conn.update_image_properties(image=None, name_or_id=None, meta=None, **properties)` | `service.UpdateCloudImageProperties(ctx, request, options...)` | `image or name_or_id`를 고른 뒤 Proxy helper에 위임; 이름 조회 없음 |
| `conn.delete_image(name_or_id, wait=False, timeout=3600, delete_objects=True)` | `service.DeleteCloudImageRecord(ctx, nameOrID, options...)` | Find(ignore_missing) 뒤 owned 삭제; Task 업로드 객체 정리; 선택적 부재 대기 |

`show_all=True`이면 Source가 `filter_deleted`를 False로 덮어쓴 뒤 v2 query `member_status=all`을 추가합니다. Go Service는 Glance v2 전용이므로 `supports_version(image, '2')` 분기는 항상 v2 경로입니다. `search_images`와 `get_image`의 검색 경로는 `list_images()`를 인자 없이 호출하므로 항상 deleted 필터를 적용하고 `member_status`를 보내지 않습니다. 이름이나 Cloud 필터는 목록 query나 Proxy의 Body 필터로 먼저 보내지 않습니다.

`get_image`는 `filters is not None`으로 경로를 고릅니다. 따라서 `{}`, `false`, `0`, `""`처럼 falsey인 값도 Proxy find가 아닌 검색 경로를 사용하며, 이 경우 `_filter_list`의 `if not filters`에 따라 이름/ID 선택만 적용합니다. 이 점은 falsey 필터를 무필터 Find로 바꾸는 Cloud `get_flavor`와 다릅니다. Find 경로는 Cloud deleted 필터가 없는 Proxy 엔진이므로 deleted 상태 이미지도 반환할 수 있습니다.

```python
import openstack

conn = openstack.connect(cloud="dev")
visible = conn.list_images()
everything = conn.list_images(show_all=True)
ubuntu = conn.search_images("ubuntu-*", filters={"visibility": "public"})
image = conn.get_image("ubuntu-24.04")  # find_image, ignore_missing=True
by_id = conn.get_image_by_id("4c1e7d5a-0b6f-4a28-9a5e-7b0f3c1d2e9f")
print(len(visible), len(everything), [i.id for i in ubuntu])
if image is not None:
    print(image.to_dict(), by_id.status)
```

## 독립 Go main

[설치 안내](../docs/install.md)로 모듈을 준비하고 아래 내용을 `main.go`에 저장합니다. 이 API를 포함한 revision 이후를 사용합니다. `clouds.yaml`의 cloud를 `-cloud`로 고르고 `-pattern`의 기본값 `*`로 검색합니다. `-image`를 주면 Cloud get을, `-id`를 주면 strict literal get을 추가로 수행합니다. 문서 빌드는 실제 cloud 인증이나 권한 검증을 뜻하지 않습니다.

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/image"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    pattern := flag.String("pattern", "*", "image name or ID glob")
    showAll := flag.Bool("show-all", false, "include shared images that are not accepted")
    identity := flag.String("image", "", "optional image name or ID for Cloud get_image")
    id := flag.String("id", "", "optional image ID for strict get_image_by_id")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *pattern, *showAll, *identity, *id); err != nil {
        log.Fatal(err)
    }
}

func recordValue(record *image.ImageRecord) any {
    if record == nil {
        return nil
    }
    return map[string]any{
        "resource": record.Resource, "wire": record.Wire,
        "status_code": record.StatusCode, "import_methods": record.ImportMethods,
    }
}

func printQuery(operation string, result *image.ImageRecordQueryResult) error {
    if result == nil {
        return nil
    }
    data, err := json.MarshalIndent(map[string]any{
        "operation": operation, "value": result.Value,
        "image_count": len(result.Images), "inventory_count": len(result.Inventory),
    }, "", "  ")
    if err != nil {
        return err
    }
    fmt.Println(string(data))
    return nil
}

func run(ctx context.Context, cloud, pattern string, showAll bool, identity, id string) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil {
        return err
    }
    service, err := conn.Image(ctx)
    if err != nil {
        return err
    }
    all, err := service.AllCloudImageRecords(ctx, image.WithImageRecordQueryShowAll(showAll))
    if printErr := printQuery("list_images", all); printErr != nil {
        return errors.Join(err, printErr)
    }
    if err != nil {
        return fmt.Errorf("list images: %w", err)
    }
    matched, err := service.SearchImageRecords(ctx, pattern)
    if printErr := printQuery("search_images", matched); printErr != nil {
        return errors.Join(err, printErr)
    }
    if err != nil {
        return fmt.Errorf("search images: %w", err)
    }
    if identity != "" {
        found, err := service.GetCloudImageRecord(ctx, identity)
        if err != nil {
            return fmt.Errorf("get image: %w", err)
        }
        data, err := json.MarshalIndent(map[string]any{"operation": "get_image", "value": found.Value, "record": recordValue(found.Image)}, "", "  ")
        if err != nil {
            return err
        }
        fmt.Println(string(data))
    }
    if id != "" {
        record, err := service.GetImageRecordByID(ctx, id)
        if err != nil {
            return fmt.Errorf("get image by id: %w", err)
        }
        data, err := json.MarshalIndent(map[string]any{"operation": "get_image_by_id", "record": recordValue(record)}, "", "  ")
        if err != nil {
            return err
        }
        fmt.Println(string(data))
    }
    return nil
}
```

error와 결과가 함께 오면 예제는 부분 증거를 출력한 뒤 error를 반환합니다. 네 호출은 각각 별개의 logical call이며 호출마다 location과 옵션을 한 번씩 준비합니다.

## concrete 옵션

`ImageRecordQueryOpts`는 `Headers`, `FilterDeleted *bool`, `ShowAll *bool`, `Filters *json.RawMessage`를 갖습니다. 각 함수는 Source 선언에 있는 인자만 받습니다. 그래서 다른 함수의 인자가 present이면 HTTP 전에 `resource.ErrInvalidOption`을 반환합니다.

| 함수 | 받는 옵션 | 기본값 |
|---|---|---|
| `AllCloudImageRecords` | `WithImageRecordQueryFilterDeleted`, `WithImageRecordQueryShowAll`, header | nil FilterDeleted는 true, nil ShowAll은 false |
| `SearchImageRecords` | `WithImageRecordQueryFilters`, `WithImageRecordQueryExpression`, header | 필터 없음 |
| `GetCloudImageRecord` | `WithImageRecordQueryFilters`, `WithImageRecordQueryExpression`, header | 필터 없음이면 Proxy find |
| `GetImageRecordByID` | header | 없음 |
| `GetImageRecordExclude`, `GetImageRecordName`, `GetImageRecordID` | header | 기본 `search_images` |

`WithImageRecordQueryFilters(nil)`은 필터를 비우고 `json.RawMessage("null")`은 명시 null을 보존합니다. 목록 함수는 명시 null도 present 필터로 보고 거부합니다. `get_image`에서 명시 null은 Python None과 같으므로 Find 경로입니다. `WithImageRecordQueryExpression(expression)`은 JSON 문자열 필터를 만들고 JMESPath로 실행합니다. 옵션 bytes와 header는 SDK가 소유 복사하며, callback은 logical call마다 한 번 실행하고 callback 사이마다 context와 source binding을 확인합니다. header는 Python 함수에 없는 Go 확장으로 한 logical call의 모든 요청에 적용합니다.

검색 dictionary는 Image Resource의 선언 필드65개와 `location`에 대해 공통 `cloudfilter.Select` 계약을 사용합니다. `os_distro`처럼 Image가 선언한 필드는 값이 없어도 null attribute로 존재하고, 서버의 알 수 없는 key는 `properties` 안에 있으므로 최상위 key로 지정하면 Python의 AttributeError에 대응하는 입력 오류입니다. 이름/ID는 Source의 문자열 표현과 exact-or-fnmatch 단계를 따르고 Proxy Find의 exact 비교와 구별합니다.

## 반환·부분 결과

`AllCloudImageRecords`와 `SearchImageRecords`는 `ImageRecordQueryResult{Value, Images, Inventory}`를 반환합니다. Inventory는 실제로 소비한 모든 행이며 deleted 필터로 제외한 행도 포함합니다. Value와 Images는 완료된 결과에만 채우고 빈 성공 결과의 Value는 `[]`입니다. JMESPath 결과는 scalar, object, null을 포함한 임의 JSON일 수 있어 Images를 합성하지 않습니다.

deleted 필터가 켜져 있으면 Source의 `image.status.lower()`처럼 status는 문자열이어야 합니다. null, 숫자, 배열 status는 그 행의 실제 목록 응답 증거를 담은 `resource.ResponseError`로 실패하고, 그 행까지의 Inventory를 보존합니다. `FilterDeleted(false)`나 `ShowAll(true)`는 status를 검사하지 않습니다. 이 Go profile은 lone surrogate가 없는 Unicode JSON 문자열을 요구합니다. 뒤 페이지 실패와 로컬 필터 오류도 Inventory를 보존하고 Value와 Images를 완료 결과로 채우지 않습니다.

`GetCloudImageRecord`는 `CloudImageRecordResult{Value, Image, Inventory}`를 반환합니다. Find 경로에서 Image는 찾은 record, Value는 그 Resource view이며 찾지 못하면 둘 다 nil입니다. Find의 GET, 일반 목록, 숨김 목록 순서와 clean400/403/404 fallback은 기존 `FindImageRecord`와 같습니다. 필터 경로는 검색 Value에 Python의 truthiness, `len`, 정수 index 0을 적용합니다. 둘 이상이면 `resource.ErrAmbiguous`를 감싼 `ImageRecordSelectionError`이고, 문자열 결과는 code point 길이를 사용합니다. 선택한 JSON null은 결과 없음이며 false, 0, 빈 문자열은 present Value로 남습니다. Image는 ordinary 선택에서 정확히 한 행이 남을 때만 채웁니다.

`GetImageRecordByID`는 `GetImageRecord(ImageRecordRequest{ID: id})`에 위임합니다. Find fallback이나 missing-as-nil이 없고 native 오류를 그대로 반환하며 operation 이름만 `GetImageRecordByID`로 표시합니다. 빈 ID, 공백, `.`/`..`, control 문자는 HTTP 전에 거부합니다.

## exclude 선택과 이름·ID 반환

세 helper는 필터 없는 `search_images(name_or_id)`를 먼저 끝냅니다. 그래서 deleted 행은 후보가 아니고 빈 nameOrID는 전체 목록을 뜻합니다. exclude가 빈 문자열이면 Python의 None이나 빈 문자열처럼 첫 행을 반환합니다. 그렇지 않으면 행 순서대로 Python `exclude not in image.name`을 평가해 처음 통과한 행을 고르고, 그 뒤 행은 검사하지 않습니다.

`in`의 의미는 name의 JSON 형태를 따릅니다. 문자열은 부분 문자열, 배열은 문자열 원소의 동등 비교, 객체는 key 멤버십입니다. null, 숫자, bool name에 도달하면 Python TypeError에 대응해 그 행의 실제 목록 응답 증거를 담은 입력 오류를 반환하고 Inventory를 보존합니다. 이 Go profile은 lone surrogate가 없는 Unicode JSON 문자열을 요구합니다.

결과는 `CloudImageRecordResult`입니다. 선택한 행이 없으면 Image와 Value가 nil이며 Python의 None에 해당합니다. `GetImageRecordExclude`의 Value는 선택 행의 declared view이고 `GetImageRecordName`과 `GetImageRecordID`의 Value는 그 행의 raw `name`, `id` JSON입니다. Python은 None name과 결과 없음을 구별하지 못하지만 Go는 선택된 null name을 Value `null`과 non-nil Image로 구별합니다. 식별자 단계는 숫자 ID도 문자열 표현으로 비교합니다.

## 이미지 대기

`WaitForCloudImageRecord`는 전달한 record의 문자열 `id`만 읽습니다. record에 이미 기록된 status는 사용하지 않고, 반복마다 `get_image(image_id)`와 같은 Proxy find 경로(ignore_missing true)를 호출합니다. 찾지 못하면 계속 기다리고, status가 정확히 `"active"`이면 그 record를 반환하며 정확히 `"error"`이면 `resource.FailedStateError`로 실패합니다. 비교는 Python `==`이므로 대소문자가 다른 값, null, 숫자 status는 계속 대기합니다. 이 점은 소문자 비교를 쓰는 `WaitForImageRecordStatus`와 다릅니다. find 자체의 오류는 즉시 반환합니다.

시간 정책은 `iterate_timeout`을 따릅니다. 기본 timeout은 3600초이고 `WithImageRecordCloudWaitUnlimited`가 Python None입니다. 각 조회 전에 deadline을 확인하므로 0이나 음수 timeout은 HTTP 없이 `context.DeadlineExceeded`를 감싼 오류입니다. 진행 중인 조회는 deadline으로 취소하지 않으며 조회 뒤 간격 전체를 기다립니다. 기본 간격은 2초이고 `WithImageRecordCloudWaitPollInterval`은 Go 확장이며 양수여야 합니다. caller context 취소는 대기 중에도 즉시 반영합니다.

결과 `ImageRecordCloudWaitResult`의 Image는 성공한 active record이고 Last는 마지막으로 찾은 record, Lookups는 수행한 조회 수입니다. timeout, error 상태, 취소는 Last의 실제 응답 증거를 오류에 붙이고 결과를 함께 반환합니다. Python의 timeout 메시지 문자열과 SDKException 계층은 Go 오류 타입으로 대체합니다.

## 이미지 다운로드

`DownloadCloudImageRecord`는 `ImageRecordCloudDownloadRequest{NameOrID, Output, Filename}`를 받습니다. Source처럼 출력이 없거나 둘 다 있으면 HTTP 전에 입력 오류입니다. 그다음 `find_image(name_or_id, ignore_missing=False)`와 같은 Find 경로로 이미지를 찾고, 없으면 `resource.ErrNotFound` 오류를 반환합니다. 찾은 record는 [owned 다운로드](image-record-download.md)의 `DownloadImageRecord`에 그대로 넘기므로 필수 metadata GET, binary GET, 기본 checksum 검증, 파일 생성·writer 차용 규칙이 같습니다.

옵션은 Cloud 서명의 `chunk_size`, `stream`만 concrete하게 받습니다(`WithImageRecordCloudDownloadChunkSize`, `WithImageRecordCloudDownloadStream`). 출력이 필수이므로 stream 값은 전달해도 출력 모드가 우선합니다. store 선호·checksum 검증 끄기·hash factory 같은 Proxy 전용 옵션은 Cloud 함수에 없어 기본값을 사용합니다. header는 Find와 다운로드 단계 모두에 적용하는 Go 확장입니다.

결과 `ImageRecordCloudDownloadResult`는 Find 결과 `Found`와 하위 `Download` 결과를 분리합니다. 다운로드 단계가 실패하면 Found와 이미 받은 metadata·binary 증거를 함께 반환하고 operation 이름은 `DownloadCloudImageRecord`로 표시합니다. Python에서 빈 문자열 경로는 None이 아니어서 검사를 통과한 뒤 메모리 모드로 바뀌지만, Go는 빈 Filename을 출력 없음으로 보고 HTTP 전에 거부합니다.

## 이미지 속성 갱신

`UpdateCloudImageProperties`는 `ImageRecordCloudPropertiesRequest{Record, ID, NameOrID}`를 받습니다. Python의 `image`는 Image 객체나 문자열일 수 있어 Go에서는 Record와 ID 두 필드로 나눴습니다. Record가 있으면 그것을, 없으면 비어 있지 않은 ID를, 둘 다 없으면 NameOrID를 고릅니다. 이는 Source의 `image or name_or_id`에 해당합니다. 고른 값은 [속성 helper](image-record-properties.md)의 `UpdateImagePropertiesRecord`에 그대로 넘기며 옵션도 같은 `ImageRecordPropertiesOption`을 씁니다.

Source wrapper는 이름을 조회하지 않으므로 NameOrID도 literal identity로 전달됩니다. literal identity에는 캐시된 properties가 없어 하위 helper가 HTTP 전에 실패하는 경계도 그대로 유지합니다. 실제 사용은 Get이나 Find로 얻은 Record를 넘기는 경로입니다. 반환값과 오류는 하위 helper와 같고 operation 이름만 `UpdateCloudImageProperties`로 표시합니다.

## 이미지 삭제

`DeleteCloudImageRecord`는 먼저 `get_image(name_or_id)`와 같은 ignore_missing Find를 수행합니다. 찾지 못하면 HTTP 삭제 없이 `Deleted=false`를 반환합니다. 찾으면 그 record를 [owned 삭제](image-record-delete.md)의 `DeleteImageRecord`로 전체 삭제합니다. 삭제 단계 오류는 Found와 함께 반환하고 Swift 정리와 대기는 수행하지 않습니다.

cloud 설정의 `image_api_use_tasks`가 Python 기준 truthy이면 찾은 record의 properties에서 `owner_specified.openstack.object`, 없으면 `owner_specified.shade.object`를 확인합니다. 값은 `container/name` 형식이어야 하며 첫 `/`에서 나눈 뒤 Proxy `delete_object`와 같은 `objects.DeleteObject`(SLO HEAD 확인, 404 무시)로 객체를 지웁니다. 이 확인은 이미지 삭제 뒤에 일어나므로, 이름 목록으로 찾은 행의 null properties, 문자열이 아닌 값, `/`가 없는 값은 이미지가 이미 삭제된 상태에서 입력 오류가 됩니다. 직접 GET으로 찾은 record는 properties를 항상 dictionary로 투영합니다. 설정이 false이거나 key가 없으면 Swift를 호출하지 않습니다. 이미지 header는 Swift 요청에 붙이지 않습니다.

`WithImageRecordCloudDeleteWait(true)`이면 [이미지 대기](#이미지-대기)와 같은 시간 정책으로 찾은 ID를 반복 Find하고 결과가 없을 때 끝냅니다. timeout은 마지막으로 찾은 record의 응답 증거와 함께 `context.DeadlineExceeded`를 감싼 오류입니다. 모든 단계를 마치면 `Deleted=true`입니다. 고정 소스는 `delete_objects` 인자를 받지만 본문에서 읽지 않으므로 Go는 이 옵션을 제공하지 않습니다.

## 남은 범위와 근거

Python의 deprecated warning, mutable Resource와 Munch 객체 동일성, adapter cache·session·임의 transport는 Go concrete 옵션과 소유 record로 대체하며 동등성을 주장하지 않습니다. Cloud `create_image`는 별도 단위입니다. 실제 이미지 가시성·member 상태·deleted 행 노출과 권한은 서버가 판단합니다.

고정 소스는 [Cloud list_images·search_images·get_image·get_image_by_id](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L84-L166), [get_image_exclude·get_image_name·get_image_id](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L221-L246), [wait_for_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L248-L263), [download_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L168-L219), [update_image_properties](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L432-L445), [delete_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L265-L313), [iterate_timeout](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53-L101)와 [_filter_list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py#L43-L145)입니다. 테스트와 지원 판정은 [판정대장](../docs/sdk-support-ledger.md#glance-cloud-이미지-목록검색조회-완료)에 기록합니다.
