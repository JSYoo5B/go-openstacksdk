# 볼륨 타입 목록·검색·조회

`Connection.ListVolumeTypes`, `SearchVolumeTypes`, `GetVolumeType`는 Cinder v3 타입을
공통 raw resource와 nullable JSON view로 제공합니다. 원본 옵션을 한 번 적용한 뒤
cached Cinder 클라이언트를 선택하며, 호출자가 interface builder를 구현할 필요는 없습니다.

| openstacksdk cloud | Go |
|---|---|
| `conn.list_volume_types()` | `conn.ListVolumeTypes(ctx)` |
| `conn.search_volume_types("fast*", filters={...})` | `conn.SearchVolumeTypes(ctx, input, blockstorage.WithVolumeTypeSearchFilters(raw))` |
| `conn.get_volume_type("fast")` | `conn.GetVolumeType(ctx, input)` |
| `conn.get_volume_type("fast", filters={})` | `conn.GetVolumeType(ctx, input, blockstorage.WithVolumeTypeSearchFilters(json.RawMessage("{}")))` |
| JMESPath 문자열 filter | `blockstorage.WithVolumeTypeSearchExpression(expression)` |
| 호출별 location | `WithVolumeTypeReadLocation` 또는 `WithVolumeTypeSearchLocation` |

다음은 Connection과 직접 package 호출을 함께 보여주는 독립 Go 예제입니다.
Connection은 `sdk.Connect` 또는 `sdk.FromProvider`로 준비합니다.

```go
package example

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/gophercloud/gophercloud/v2"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func ReadTypes(ctx context.Context, conn *sdk.Connection) error {
    listed, err := conn.ListVolumeTypes(ctx)
    if err != nil {
        if listed != nil { fmt.Println("admitted list pages:", len(listed.Pages)) }
        return err
    }
    fmt.Println("normalized types:", string(listed.Value))
    for _, item := range listed.Types {
        owned := item.Clone()
        fmt.Println("original extra_specs:", string(owned.Body["extra_specs"]))
        var projection struct {
            ID *string `json:"id"`
            Name *string `json:"name"`
            ExtraSpecs map[string]json.RawMessage `json:"extra_specs"`
        }
        if err := owned.Decode(&projection); err != nil { return err }
    }

    searched, err := conn.SearchVolumeTypes(ctx,
        blockstorage.SearchVolumeTypesRequest{NameOrID: "fast*"},
        blockstorage.WithVolumeTypeSearchFilters(json.RawMessage(`{"is_public":true}`)))
    if err != nil {
        if searched != nil { fmt.Println("admitted search pages:", len(searched.Pages)) }
        return err
    }
    fmt.Println("matched types:", string(searched.Value))

    selected, err := conn.GetVolumeType(ctx,
        blockstorage.GetVolumeTypeRequest{NameOrID: "fast"})
    if err != nil {
        if selected != nil {
            fmt.Println("admitted fallback pages:", len(selected.Pages))
            if selected.Observed != nil {
                fmt.Println("admitted member status:", selected.Observed.StatusCode)
            }
        }
        return err
    }
    if selected.Value == nil { fmt.Println("type absent") } else {
        fmt.Println("selected type:", string(selected.Value))
    }

    expression, err := conn.SearchVolumeTypes(ctx,
        blockstorage.SearchVolumeTypesRequest{},
        blockstorage.WithVolumeTypeSearchExpression("[].name"))
    if err != nil { return err }
    fmt.Println("expression JSON:", string(expression.Value))
    return nil
}

func ReadTypesWithLocation(ctx context.Context, conn *sdk.Connection) error {
    cloud, project := "configured-cloud", "configured-project"
    location := resource.CloudLocation{
        Cloud: &cloud, Zone: json.RawMessage(`"owned-zone"`),
        Project: resource.CloudProject{ID: json.RawMessage(`"scope-id"`), Name: &project},
    }
    result, err := conn.ListVolumeTypes(ctx,
        blockstorage.WithVolumeTypeReadOptions(blockstorage.VolumeTypeReadOpts{Location: &location}))
    if err != nil { return err }
    fmt.Println(string(result.Value))
    return nil
}

func ReadTypesDirect(ctx context.Context, cinder *gophercloud.ServiceClient) error {
    result, err := blockstorage.GetVolumeType(ctx, cinder,
        blockstorage.GetVolumeTypeRequest{NameOrID: "fast"},
        blockstorage.WithVolumeTypeSearchFilters(json.RawMessage(`{}`)))
    if err != nil { return err }
    fmt.Println(string(result.Value))
    return nil
}
```

Python에서는 실제 cloud helper를 사용합니다.

```python
import openstack

conn = openstack.connect(cloud="example")
types = conn.list_volume_types()
matches = conn.search_volume_types("fast*", filters={"is_public": True})
selected = conn.get_volume_type("fast")
names = conn.search_volume_types(filters="[].name")
```

`list_volume_types`와 `search_volume_types`의 deprecated `get_extra`는 non-None일 때
경고만 출력하며 별도의 extra specs 요청을 만들지 않습니다. Go는 이 경고용 no-op
인자를 생략합니다. 기본 목록·검색은 `/types`를 읽으며 `is_public`·`name`·detail·limit
query를 자동으로 넣거나 extra specs를 추가 조회하지 않습니다. 모든 페이지와 view를
완성한 뒤 이름·ID와 filter를 적용합니다. 응답 순서와 중복 행을 보존하며 정확한 이름·ID
일치와 glob 매치를 행별로 적용합니다. malformed filter의 검사도 목록 완성 이후입니다.

필터를 생략하거나 literal JSON `null`을 전달한 `GetVolumeType`는 정확한 이름·ID
조회입니다. 안전한 단일 segment를 member GET으로 먼저 조회하고, clean 400·403·404에서
목록으로 fallback합니다. pinned v3 `find_type`에 맞춰 **양쪽 요청 모두 lowercase
`is_public=none`**을 사용하며 fallback 목록에는 `name` hint를 보내지 않습니다.
URL 경로 구분자·percent·query·fragment·Unicode whitespace가 있는 이름은 Go의 안전한
route 정책에 따라 목록으로 바로 조회합니다. Python Resource가 GET을 먼저 시도하는
경계와 다른 명시적 Go 매핑입니다. 두 번째 정확한 일치에서 ambiguity를 반환하고
사용하지 않는 나머지 행·페이지를 읽지 않습니다. 부재는 완성된 성공 목록 이후에만
확정하며 접근·transport·schema 오류를 부재로 바꾸지 않습니다.

`{}`, `[]`, `false`, `0`, `""` 등 **모든 nonnull filter**를 전달한 get은 전체 Search
경로를 사용합니다. filter 자체가 falsey이면 identifier 선택만 적용되지만 GET-first
경로로 돌아가지 않습니다. Go는 Python의 filter·JMESPath deprecated 경고도 출력하지 않습니다. dictionary filter는
nullable 여섯 필드의 view를 대상으로
중첩 비교를 수행하고, nonempty string은 JMESPath expression입니다. `metadata` 같은
알려지지 않은 Type 속성은 view에 생기지 않습니다. 실제 원본 wire에서 unknown
`metadata`가 있다고 normalized filter 필드로 추가하지 않습니다. 실제 선택된 행에서
unknown filter key를 소비하면 local 오류이며 해당 행이 없을 때는 lazy filter 경계에 따라
소비되지 않을 수 있습니다.

Search expression의 `Value`는 배열·객체·문자열·숫자·boolean·null을 포함한 임의 JSON입니다.
expression 결과에는 `Types`를 합성하지 않습니다. filtered get은 Python 순서대로 외부
값의 truthiness, 길이, index 0을 적용합니다. 결과 배열의 단일 `false`, `0`, `""`는
선택된 값으로 유지하고 null은 부재입니다. 길이 2 이상은 `VolumeTypeSelectionError`이며
`resource.ErrAmbiguous`로 확인할 수 있습니다. truthy 숫자의 길이 검사나 단일 객체의
integer index 접근 같은 실패도 오류로 유지합니다. 따라서 `Type == nil`만으로 expression
부재를 판정하지 않고 먼저 `err`, 이후 `Value`를 확인합니다.

| normalized field | 처리 |
|---|---|
| `id`, `name`, `description` | untyped JSON, 누락은 null |
| `extra_specs` | null은 유지, object는 중첩 JSON·숫자를 보존, nonobject는 Python dict descriptor에 따라 `{}` |
| `is_public` | `os-volume-type-access:is_public`와 normalized alias를 소비하고 Python boolean truthiness 적용 |
| `location` | 아래의 list/member 단계 규칙 적용 |

예를 들어 문자열 `"false"`는 nonempty이므로 `is_public: true`가 됩니다. 이것은
문자열 true/false를 파싱하는 BoolStr 변환이 아닙니다. alias와 반복 wire key는 원래
JSON member 순서대로 소비하고 마지막 값을 사용합니다. Python JSON decoder는 반복 key를 먼저
collapse하고 최초 key 위치를 유지하므로 반복 key와 alias가 교차한 입력에서는 이
last-textual 정책과 다를 수 있습니다. 예를 들어 normalized true → wire false → normalized true는
Go true, Python false가 될 수 있습니다. Go의 exact JSON 숫자 truthiness도 명시적 매핑입니다.
`1e-9999`는 nonzero로 true이며 일반 Python float decoder가 underflow하여 false가 되는
경계와 다릅니다. normalized view에는 여섯 필드를
모두 포함하며, `Types[i].Body` 또는 `Type.Body`에는 알려지지 않은 원본 필드와 정확한
JSON 숫자도 남습니다. `Decode`는 호출자가 선택한 typed projection을 명시적으로 적용하며
raw body를 줄이지 않습니다.

Type에는 project ID나 availability zone descriptor가 없습니다. 일반 list 행에 `id`,
`name`, `description`, `extra_specs`, public alias 중 하나라도 있으면 wire `location`은
discard하고 Connection의 현재 location을 사용합니다. 알려진 key의 null도 presence입니다.
이 알려진 Body 속성이 하나도 없는 list 행은 computed wire `location`을 그대로 보존하므로
location-only 행의 null·false·문자열·객체도 유지합니다. member GET은 wire computed
location을 항상 무시합니다. unknown `project_id`·`availability_zone`는 원본에 남지만
소유 프로젝트나 zone을 추론하는 근거로 사용하지 않습니다.

Connection 기본값은 원본 옵션 적용 후 `CurrentLocation`을 읽은 snapshot입니다. 실행 중
기록된 scope가 바뀌어도 실행은 그 snapshot을 유지하고 다음 호출은 새 scope를 읽습니다.
read/search location factories는 인자와 callback 출력의 복사본을 소유하며 Connection
기본값을 바꾸지 않습니다. 직접 package 호출은 provider에 이미 기록된 project ID를
사용하고 모르는 cloud·region·설정 이름은 null입니다. 완전한 owned location override의
Zone도 유지합니다. Python Type의 기본 current location은 zone을 기본 None으로 계산하므로
이 override는 의도한 Go 확장입니다. caller의 잘못된 Location은 실제 computed location을
소비할 때 local `resource.ErrInvalidOption`이며 기존 proof를 남기지만 HTTP 200
ResponseError를 빌리지 않습니다. 빈 목록이나 wire location-only 행은 사용하지 않은
computed Location을 검사하지 않습니다.

`Pages`는 실제 허용한 목록 200, `Observed`는 실제 허용한 member 200 응답입니다. raw body,
header, status는 resource와 독립적으로 소유합니다. 전체 목록·선택이 실패하면 `Value`와
`Types`·`Type`은 미완성 상태이며 완료된 응답 proof는 유지합니다. read·Close·정규화·source
변경·context cancellation cause도 오류 체인에 남습니다. 물리 row/descriptor 오류는
자신의 accepted 응답 오류를 유지하며, 거절된 HTTP의 native status·header·body도 보존합니다.
SDK는 workflow 전체를 replay하지 않고 native auth·retry를 유지합니다.

collector는 canonical `volume_types` 배열을 사용하며 빈 배열에서 바로 EOF로 끝냅니다.
continuation은 `links` key가 있으면 그 값을 사용하고, 없을 때 plural
`volume_types_links`를 사용합니다. Body에서는 처음 만난 `rel == "next"`이면서 href가 있는
행을 소비하며 그 href가 falsey여도 뒤 body 행을 다시 검색하지 않습니다. 그 경우
다음 우선순위인 top-level `next`, 이어서 HTTP Link next로 진행합니다. native singular
`volume_type_links`는 낮은 우선순위의 Go 호환 경로입니다.

페이지 처리는 다음 Go 차이도 명시적으로 둡니다. pinned source는 Requests가 실제로
제공하는 `response.links["next"]["url"]` 대신 `["uri"]`를 읽어 실패할 수 있으므로 Go는
quoted Link parser로 이 경로를 보완합니다. ([Requests Response.links](https://requests.readthedocs.io/en/latest/_modules/requests/models/#Response.links), [parse_header_links](https://requests.readthedocs.io/en/latest/_modules/requests/utils/#parse_header_links)) 이 parser의 case-insensitive rel·여러 relation
지원도 Go 호환 정책입니다. Python의 body relative `/v...` 링크는 첫 version segment를
제거한 뒤 adapter에 다시 전달하지만 Go는 광고된 relative/absolute URL을 현재 collection의
effective URL로 resolve하고 고정한 origin·escaped path를 검사합니다. query-only 링크의
원래 query가 URI와 params 양쪽에서 반복될 수 있는 Python 경계도 그대로 재현하지 않습니다.
Go는 현재 query에서 marker·limit을 먼저 제거하고 next의 nonblank 값만 한 번 canonical
merge하여 중복 query를 재생성하지 않습니다. 광고된 nonblank key는 같은 current key를
교체하고, 광고하지 않은 nonpagination key는 유지합니다. 초기 limit이나
추정 marker를 새로 넣지 않습니다. cycle은 Python의 같은 marker bookkeeping과 달리
완전한 canonical URL로 검사하며 fragment·source 변경도 검사합니다. canonical
UTF-8의 `volume_types` 배열과 `volume_type` object를 요구하는 strict Go boundary이며
native typed extraction을 전체 페이지 emptiness 검사로 사용하지 않습니다.

Go는 fresh canonical 응답에서 빠진 ID와 public 값을 null로 유지합니다. Python Resource는
member 조회를 위해 seeded 요청 ID와 `is_public='none'` 상태를 보존할 수 있으므로
빠진 response 필드에서 동일한 mutable object state를 재현한다는 주장을 하지 않습니다.
비교 대상은 openstacksdk pin
[`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L66-L79),
[search](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L916-L947),
[get](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L133-L173),
[v3 Type descriptors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/type.py),
[v3 find query](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L474-L496)입니다.
v2 Type, generic Resource/session/cache, 타입 접근·mutation과 전체 SDK의 지원 판정은 각각
독립적입니다. 이 세 helper가 전체 SDK 완료를 뜻하지 않습니다.
