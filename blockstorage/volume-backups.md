# 볼륨 backup 조회

`ListVolumeBackups`, `SearchVolumeBackups`, `GetVolumeBackup`는 Cinder v3 backup을
같은 옵션·nullable 값·오류 방식으로 읽습니다. Connection이 인증과 cached Cinder 선택을
맡으며, 직접 package 함수는 준비한 `ServiceClient`를 받습니다. 필요한 값을 `With...`로
지정하거나 complete 옵션을 넘기므로 호출자가 query builder·decoder interface를 구현할 필요가 없습니다.

| 실제 openstacksdk cloud helper | Connection | 직접 package 함수 |
|---|---|---|
| `conn.list_volume_backups()` | `conn.ListVolumeBackups(ctx)` | `blockstorage.ListVolumeBackups(ctx, cinder)` |
| `conn.search_volume_backups("nightly*", filters)` | `conn.SearchVolumeBackups(ctx, blockstorage.SearchVolumeBackupsRequest{NameOrID: "nightly*"}, options...)` | `blockstorage.SearchVolumeBackups(ctx, cinder, request, options...)` |
| `conn.get_volume_backup("nightly", filters=None)` | `conn.GetVolumeBackup(ctx, blockstorage.GetVolumeBackupRequest{NameOrID: "nightly"}, options...)` | `blockstorage.GetVolumeBackup(ctx, cinder, request, options...)` |

`Connect`는 clouds.yaml·환경 인증 설정을 사용하고 `FromProvider`는 이미 인증된 provider를
받습니다. 아래 독립 Go 예제의 `ctx`와 cloud 이름·provider·Cinder client는 호출자가 준비합니다.
각 함수는 실제 HTTP 요청을 하므로 예제를 실행할 때 사용할 cloud 설정과 resource 이름을 지정합니다.
`Configured`·`Adopt`는 Connection 생성 경로이고 `Direct`는 package 대안입니다.

```go
package example

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"

    "github.com/gophercloud/gophercloud/v2"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func Configured(ctx context.Context, cloudName string) (*sdk.Connection, error) {
    return sdk.Connect(ctx, sdk.WithCloud(cloudName))
}

func Adopt(provider *gophercloud.ProviderClient) (*sdk.Connection, error) {
    return sdk.FromProvider(provider, sdk.WithRegion("RegionOne"))
}

func InspectPages(pages []*blockstorage.VolumeBackupsPage) {
    for _, page := range pages {
        fmt.Println("admitted backup page:", page.StatusCode, "bytes:", len(page.Body))
    }
}

func Explain(err error) {
    if err == nil { return }
    var ambiguous *blockstorage.VolumeBackupSelectionError
    if errors.As(err, &ambiguous) {
        fmt.Println("filtered selection length:", ambiguous.Length)
    }
    fmt.Println("ambiguous:", errors.Is(err, resource.ErrAmbiguous), "error:", err)
}

func Inspect(result *blockstorage.GetVolumeBackupResult) {
    if result == nil { return }
    InspectPages(result.Pages)
    if result.Observed != nil {
        fmt.Println("member response:", result.Observed.StatusCode, len(result.Observed.Body))
    }
    fmt.Println("requested ID:", result.RequestedID, "seeded logical ID:", result.SeededID)
    if result.Backup != nil {
        owned := result.Backup.Clone()
        fmt.Println("actual wire ID JSON:", string(owned.Body["id"]))
        var projection struct {
            ID json.RawMessage `json:"id"`
            Name *string `json:"name"`
        }
        if err := owned.Decode(&projection); err != nil {
            fmt.Println("optional wire projection:", err)
        } else {
            fmt.Println("projected wire ID:", string(projection.ID))
        }
    }
}

func ViaConnection(ctx context.Context, conn *sdk.Connection) error {
    listed, err := conn.ListVolumeBackups(ctx,
        blockstorage.WithVolumeBackupListFilters(json.RawMessage(
            `{"status":"available","volume_id":"volume-id","description":"nightly"}`)))
    if listed != nil { InspectPages(listed.Pages) }
    if err != nil { return err }
    fmt.Println("complete normalized list:", string(listed.Value), "raw backups:", len(listed.Backups))

    searched, err := conn.SearchVolumeBackups(ctx,
        blockstorage.SearchVolumeBackupsRequest{NameOrID: "nightly*"},
        blockstorage.WithVolumeBackupSearchFilters(json.RawMessage(`{"status":"available"}`)))
    if searched != nil { InspectPages(searched.Pages) }
    if err != nil { return err }
    fmt.Println("selected normalized backups:", string(searched.Value))

    selected, err := conn.GetVolumeBackup(ctx,
        blockstorage.GetVolumeBackupRequest{NameOrID: "nightly"})
    Inspect(selected) // Inspect actual proof before deciding what to do with err.
    Explain(err)
    if err != nil { return err }
    if selected.Value == nil {
        fmt.Println("completed lookup: no backup found")
    } else {
        fmt.Println("selected normalized JSON:", string(selected.Value))
    }
    return nil
}

func PreparedFirstPage(ctx context.Context, conn *sdk.Connection) error {
    calls := 0
    policy, err := blockstorage.PrepareVolumeBackupListOptions(ctx,
        func(value *blockstorage.VolumeBackupListOpts) error {
            calls++
            return blockstorage.WithVolumeBackupListFilters(
                json.RawMessage(`{"status":"available"}`))(value)
        },
        blockstorage.WithVolumeBackupListDetailed(false),
        blockstorage.WithVolumeBackupListPagination(false),
        blockstorage.WithVolumeBackupListMaxItems(10),
        blockstorage.WithVolumeBackupListHeaders(map[string]string{
            "X-Request-ID": "backup-read-example",
        }))
    if err != nil { return err }
    fmt.Println("original callback calls:", calls)
    result, err := conn.ListVolumeBackups(ctx,
        blockstorage.WithVolumeBackupListOptions(policy))
    if result != nil { InspectPages(result.Pages) }
    return err
}

func ProjectNames(ctx context.Context, conn *sdk.Connection) error {
    result, err := conn.SearchVolumeBackups(ctx,
        blockstorage.SearchVolumeBackupsRequest{},
        blockstorage.WithVolumeBackupSearchExpression("[].name"))
    if result != nil { InspectPages(result.Pages) }
    if err != nil { return err }
    fmt.Println("arbitrary expression JSON:", string(result.Value), "raw associations:", len(result.Backups))
    return nil
}

func FilteredGet(ctx context.Context, conn *sdk.Connection, name string) error {
    // Explicit {} chooses complete Search instead of ordinary GET-first find.
    result, err := conn.GetVolumeBackup(ctx,
        blockstorage.GetVolumeBackupRequest{NameOrID: name},
        blockstorage.WithVolumeBackupSearchFilters(json.RawMessage(`{}`)))
    Inspect(result)
    Explain(err)
    return err
}

func Direct(ctx context.Context, cinder *gophercloud.ServiceClient) error {
    cloud, region, projectName := "owned-cloud", "RegionOne", "configured-project"
    location := resource.CloudLocation{
        Cloud: &cloud, RegionName: &region,
        Project: resource.CloudProject{ID: json.RawMessage(`"project-id"`), Name: &projectName},
    }
    listed, err := blockstorage.ListVolumeBackups(ctx, cinder,
        blockstorage.WithVolumeBackupListLocation(location))
    if listed != nil { InspectPages(listed.Pages) }
    if err != nil { return err }

    policy, err := blockstorage.PrepareVolumeBackupSearchOptions(ctx,
        blockstorage.WithVolumeBackupSearchLocation(location),
        blockstorage.WithVolumeBackupSearchFilters(json.RawMessage(`{"status":"available"}`)))
    if err != nil { return err }
    searched, err := blockstorage.SearchVolumeBackups(ctx, cinder,
        blockstorage.SearchVolumeBackupsRequest{NameOrID: "nightly*"},
        blockstorage.WithVolumeBackupSearchOptions(policy))
    if searched != nil { InspectPages(searched.Pages) }
    if err != nil { return err }

    selected, err := blockstorage.GetVolumeBackup(ctx, cinder,
        blockstorage.GetVolumeBackupRequest{NameOrID: "nightly"},
        blockstorage.WithVolumeBackupSearchLocation(location))
    Inspect(selected)
    return err
}
```

실제 Python cloud 호출은 다음과 같습니다. `filters=None`의 Get과 `filters={}`의 Get은
서로 다른 경로이며, Search의 `filters`는 List의 server query를 대신하지 않습니다.

```python
import openstack

conn = openstack.connect(cloud="my-cloud")
backups = conn.list_volume_backups(
    detailed=True,
    filters={"status": "available", "description": "nightly"},
)
matches = conn.search_volume_backups("nightly*", {"status": "available"})
backup = conn.get_volume_backup("nightly")
filtered_backup = conn.get_volume_backup("nightly", filters={})
names = conn.search_volume_backups(filters="[].name")
```

비교 기준은 고정한 openstacksdk의
[List](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L733),
[Search](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L890),
[Get](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L668)입니다.
Python의 deprecated filter warning은 Go에서 별도 warning으로 재현하지 않습니다.

기본 List는 상세 `/backups/detail`을 끝까지 읽습니다. `Detailed: nil`·`Paginated: nil`은 true이며
`WithVolumeBackupListDetailed(false)`는 `/backups`, `WithVolumeBackupListPagination(false)`는
첫 페이지를 선택합니다. 기본 limit·maximum·호출별 microversion·expression을 만들어 넣지 않습니다.
추가 header가 없어도 목록의 기본 `Accept`는 `application/json`입니다.
`WithVolumeBackupListOptions`는 complete 정책을 교체하고 개별 factory는 해당 값만 지정합니다.
`PrepareVolumeBackupListOptions`·`PrepareVolumeBackupSearchOptions`는 원본 callback을 한 번씩
처리하고 pointer·map·raw JSON·callback slice를 소유하며 서비스 I/O는 하지 않습니다.
Prepare 자체는 nil default를 true로 채우지 않으며, 필터·runtime control의 사용 시점 검증은 실제 호출이 합니다.

List의 raw JSON filters는 server query와 받아온 row의 local 조건으로 나뉩니다.
`name`, `status`, `volume_id`, `project_id`, `limit`, `marker`, `offset`, `sort_dir`, `sort_key`, `sort`는
server query이고 `all_projects`는 wire `all_tenants`로 매핑합니다. Backup의
[Proxy.backups](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1631)는
Snapshot과 달리 `all_projects`를 별도 truthiness 인자로 먼저 소비하지 않습니다. 따라서 raw
false·0·null·문자열 `"false"`도 query 선택에 남으며 `all_tenants`와 함께 있으면 `all_projects`가
우선합니다. JSON 값은 Requests 계열 규칙으로 인코딩하며, null은 실제 query에서 생략합니다.
문자열 `"false"`를 Boolean true로 바꾸지 않습니다. `__conflicting_attrs`로 늦게 들어온 값도
Resource 단계의 literal query 정책을 적용합니다.

`availability_zone`, `container`, `created_at`, `data_timestamp`, `description`, `encryption_key_id`,
`fail_reason`, `force`, `has_dependent_backups`, `is_incremental`, `links`, `metadata`, `object_count`,
`size`, `snapshot_id`, `updated_at`, `user_id`, `volume_name`, `id`는 local Body 조건입니다.
알려지지 않은 key는 source처럼 버립니다. `incremental`은 create request용 이름이고 read의
`is_incremental` alias가 아닙니다. extended project wire key도 local `project_id`나 query alias를
대신하지 않습니다. `AllowUnknownParams`는 concrete control이지만 pinned Resource.list는
query validation에 항상 `allow_unknown_params=True`를 전달하므로 unknown key를 strict error로 바꾸지 않습니다.

List의 local 조건은 source 순서대로 short-circuit하며 중첩 dictionary를 재귀 비교합니다.
falsey actual 값은 빈 expected dictionary에도 일치하지 않습니다. truthy scalar는 빈 expected
dictionary에 일치하지만 nonempty expected dictionary에 도달하면 `.get`이 없는 local 오류가 됩니다.
앞 조건이 불일치하면 사용하지 않은 뒤 조건을 실행하지 않습니다. 다만 backup constructor는
그 전에 알려진 모든 typed descriptor를 변환하므로, 뒤에서 필터로 제외할 row의 잘못된
`size`·`object_count`도 숨기지 않습니다. 실제
[Resource constructor와 list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L540)의
eager `to_dict` 시점에 맞춘 동작입니다.

Search는 query가 없는 상세 목록을 먼저 완성한 뒤 exact ID/name 또는 glob, recursive mapping,
JSON expression을 적용합니다. 빈 `NameOrID`는 이름·ID 조건을 생략합니다. unknown mapping
filter는 실제로 검사할 row에서 오류이며, 빈 목록처럼 사용하지 않은 filter는 별도로 적용할 row가 없습니다.
필터의 JSON 문법·shape·expression 오류는 전체 목록을 읽은 뒤 평가하므로 앞 페이지의 잠정 일치로
뒤 페이지의 오류를 숨기지 않습니다. 이름·ID를 비교할 때의 JSON 값→Python-style string 표현도
ordinary exact find의 string 비교와 구별합니다.

Get의 filter 생략·literal JSON null은 ordinary find입니다. 안전한 literal identity는 먼저
`/backups/{identity}`를 GET하고, clean native 400·403·404 rejection이면 `name` query가 있는 상세
목록에서 exact ID 또는 name을 찾습니다. 500 등 다른 거부, read·Close·cancel·callback/source 오류는
absence로 바꾸지 않습니다. 공백·slash·예약 문자 등 안전한 member segment로 쓸 수 없는 이름은
직접 GET을 생략하고 exact 이름 목록 경로를 사용합니다. identity를 trim하거나 URL-decode하지 않는
Go의 route 경계입니다. 기본 ordinary Get은 nonempty UTF-8 identity를 요구합니다.
한 일치 뒤에도 필요한 목록을 소비하므로 뒤 duplicate·응답 실패가 성공을 취소할 수 있습니다.

반면 `{}`·`[]`·false·0·빈 문자열처럼 **nonnull인 filter는 모두 full Search 경로**를 선택합니다.
filter 자체가 falsey이면 추가 filter는 생략하지만 목록을 읽는 경로는 유지합니다.
expression의 Get은 Python의 truthiness → `len` → index 0 순서를 JSON 도메인에서 적용합니다.
둘 이상이면 `VolumeBackupSelectionError`/`ErrAmbiguous`, 길이가 없는 truthy scalar나 integer key 0이
없는 한-key object는 local 오류입니다. `[false]`·`[0]`·`[""]`는 해당 값을 성공 결과로 유지하고
`[null]`은 nil 선택입니다. expression 결과는 임의 JSON일 수 있어 raw Backup 연결을 만들어 넣지 않습니다.

`Value`는 원본 응답과 별도의 nullable logical JSON입니다.
[v3 Backup descriptors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L26)의
23 Body 필드와 computed `location`, 총 24개를 사용하며 알려진 값이 없거나 null이면 null입니다.
`force`, `has_dependent_backups`, `is_incremental`은 **일반 Boolean**입니다. Snapshot BoolStr과 달리
`"false"`도 nonempty string이므로 true, 빈 string/list/object와 0은 false, null은 null입니다.
`links`는 list를 유지하고 nonnull scalar/object를 한 번 `[value]`로 감싸며 null은 null입니다.
`metadata`는 object를 유지하고 nonobject nonnull을 `{}`로 바꿉니다. nullable `size`·`object_count`는
descriptor integer 변환을 사용합니다. booleans는 그대로 bool이며 정수는 arbitrary precision,
float는 Python float값을 integer로 절삭합니다. digit string은 integer로 변환하고 signed·공백이 있는·
비숫자 string/container는 0이 됩니다. digit이지만 decimal이 아닌 문자나 float overflow는 오류입니다.
나머지 Body 값은 timestamp·ID·name도 문자열로 강제하지 않고 원래 JSON 값을 유지합니다.

`project_id`와 `os-backup-project-attr:project_id`는 같은 descriptor입니다. JSON의 같은 key는
마지막 값으로 collapse하되 첫 insertion 위치를 유지하고, 그 뒤 normalized/wire alias를 소비한 순서의
마지막 값이 선택됩니다. textual 마지막 alias 정책과 다릅니다. case variant는 unknown입니다.
목록 row의 `connection`·`microversion`·`_synchronized`는 false/null이어도 constructor 인자 충돌이며,
member의 같은 raw 필드는 runtime control이 되지 않습니다. `self`는 source처럼 무시합니다.
metadata 안의 같은 이름은 ordinary nested 값으로 유지합니다.

성공한 member 응답의 canonical `id`가 **없으면** requested ID를 logical 값에만 seed합니다.
명시적 null·false·0·빈 문자열·container ID는 응답 값을 그대로 유지합니다. `RequestedID`·`SeededID`는
이 차이를 설명하고 `Backup.Body["id"]`에는 seed를 쓰지 않습니다. member의 malformed/empty JSON은
source가 허용한 empty-resource 경로로 nullable·seeded logical 값이 될 수 있습니다. valid nonobject,
null envelope·nonnull object가 아닌 member·invalid UTF-8은 성공한 빈 Backup으로 바꾸지 않습니다.
`Backup`·`Backups`는 실제 field를 보존한 `RawResource`이고 unknown fields·정확한 raw 숫자 spelling·
response header/status를 소유합니다. `Clone`은 독립 복사이고 `Decode`는 명시적인 atomic typed projection입니다.
typed projection에 실패했다고 이미 완료한 cloud helper가 실패한 것은 아닙니다.

Connection은 원본 옵션 → owned `CurrentLocation` snapshot → cached Cinder v3 선택 순서로 처리합니다.
기본 direct package 호출은 callback 뒤 provider의 recorded project ID를 읽고 나머지 cloud/config 사실을
추정하지 않습니다. `WithVolumeBackupListLocation`·`WithVolumeBackupSearchLocation`으로 전체 owned
cloud/project facts를 지정할 수 있습니다. 연결된 Backup은 row project와 availability zone을 사용해
location을 계산합니다. falsey 또는 현재 scope와 같은 project는 configured 이름/domain을 유지하고
truthy foreign project는 실제 raw ID만 유지하며 이름/domain을 null로 지웁니다. row zone은 arbitrary
JSON 값 그대로이며 missing/null이면 null입니다. **caller Location.Zone도 row zone을 대신하지 않습니다.**
raw HTTP `location`은 계산의 근거가 아니며 location-only row도 다시 계산합니다.
잘못된 owned location은 첫 HTTP 전에 local `ErrInvalidOption`이고 Connection은 Cinder getter 전에 검증합니다.
진행 중 native token/auth scope 변경은 captured logical scope를 바꾸지 않으며 다음 호출은 다시 snapshot합니다.

목록은 source의 100..399 status 정책을 사용합니다. 204 empty member는 nullable/seeded 결과가 될 수
있지만 empty List는 JSON 오류입니다. `Pages`는 실제 admitted list response, `Observed`는 실제 admitted
member response의 body/header/status이고 logical 완료를 보장하지 않습니다. suppressed member 거부 뒤
목록으로 fallback하면 그 member를 list의 `Observed`로 남기지 않습니다. physical read·Close·cancel·
descriptor·pagination/source 실패에도 이미 admitted된 proof를 유지합니다. mapping/expression/local filter·
선택 오류는 local 오류이며 이전 응답을 새로운 HTTP 실패로 빌리지 않습니다.

Gophercloud는 native unexpected-status 응답에서 body `Read`·`Close` 오류를 버립니다.
SDK는 ordinary Cinder member의 400·403·404를 따로 관찰해, IO 실패가 있으면 목록
fallback을 금지하고 native status 오류와 IO cause를 함께 반환합니다. native에서 거부된
이 응답을 admitted `Observed`·`Pages`로 만들지 않습니다. 같은 member 보호는 Snapshot
일반 조회에도 적용됩니다. 목록의 native 거부와 다른 거부 status의 body IO는 기존
Gophercloud 오류 정책을 따릅니다. admitted 응답의 `Read`·`Close`·cancel 보호와는 별도 범위입니다.

일반 빈 List·Search는 `Value: []`와 nonnil 빈 `Backups`, expression 결과는 성공한 `[]`여도
raw association이 없어서 `Backups: nil`입니다. 목록 실패는 `Value`·`Backups`가 nil이며 부분 logical rows를
발표하지 않습니다. normal Get absence는 오류 없이 `Value`·`Backup`가 nil입니다.

pagination은 body top-level `links`의 존재가 `backups_links`보다 우선합니다. 배열의 첫 exact
`rel: "next"`와 존재하는 `href`에서 멈추고 unused tail을 읽지 않습니다. consumed malformed prefix는
오류이며 selected falsey href는 뒤 item 대신 top-level `next`·HTTP Link로 fallback합니다.
dictionary links는 source의 one-key dictionary 순회 때문에 next로 해석하지 않습니다.
canonical empty `backups: []`는 링크를 소비하지 않고 EOF입니다. 이전 marker/limit는 제거하고 advertised
blank 값은 버리며 반복 query 값은 유지합니다. omitted nonpagination key는 유지하고 advertised key는 교체합니다.
초기 truthy limit와 next 부재이면 마지막 소비 row의 normalized ID로 다음 요청을 만듭니다.
짧은 페이지나 local에서 제외한 마지막 row도 marker 계산에서 빠지지 않습니다.

`MaxItems` typed 값은 `*int`이며 raw `max_items`는 JSON number/boolean/null도 받습니다.
fractional·negative 값을 포함하며 false·0·null은 무제한, true는 1입니다. nonzero 음수는 GET과 전체
JSON 파싱 뒤 첫 row 전에 멈춥니다. maximum은 local에서 제외한 raw row도 셉니다. 페이지 마지막 row에서
maximum에 도달하면 continuation을 소비해 다음 GET을 한 뒤 다음 row 전에 멈출 수 있습니다.
same-marker와 complete canonical URL cycle을 검사하므로 필요한 다음 link/응답 오류도 숨기지 않습니다.
Go의 exact decimal counter/truthiness/equality와 Python JSON float rounding·underflow의 경계는 다를 수 있습니다.

HTTP Link 처리는 pinned `response.links['next']['uri']`와 Requests의 실제 `url` 차이를 Go가 교정합니다.
[Requests parser](https://requests.readthedocs.io/en/latest/_modules/requests/utils/#parse_header_links)의
단일 rel dictionary와 달리 Go는 strict quoted parser로 multi-rel·대소문자 없는 next·마지막 next를
처리합니다. unused header는 body next가 있으면 검증하지 않습니다. relative link를 effective URL에 대해
해결하고 같은 origin/escaped collection path를 유지하는 것도 Go의 URI 경계입니다. Python의 `/v` prefix
제거·query-only href 중복을 그대로 재현하지 않습니다.

List의 `WithVolumeBackupListExpression`은 Resource generator를 바로 expression engine에 넘기는
[legacy Proxy 경로](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866)를
완성한 normalized list 평가로 고칩니다. HTTP/평가 시점과 결과가 달라질 수 있습니다.
typed `Expression("")`는 전체 목록 뒤 syntax error, raw falsey `jmespath_filters`는 expression 생략입니다.
typed controls는 해당 raw Resource control을 덮어쓰고 late `__conflicting_attrs`도 source binding 단계에 맞춰
처리합니다. headers nil은 raw headers를 유지하며 nonnil 빈 map은 raw 추가 header를 지우고 기본 Accept를
남깁니다. 이미 capture한 native client ordinary header까지 지우는 옵션은 아닙니다.
raw Proxy/Resource bound 인자의 충돌은 local 오류이며 호출자가 route/decoder를 바꾸는 제어판으로 쓰지 않습니다.

headers는 HTTP token/UTF-8·case alias·인증/framing/서비스 version 규칙을 검증합니다. list override는
복사한 request source에 적용하며 원본 client를 수정하지 않습니다. provider·endpoint·ResourceBase·type·
microversion 변경은 terminal 오류입니다. ordinary headers는 captured 값이고 auth token은 live provider 값이며,
native retry/reauth/redirect에도 고정 GET·URL/query·status 정책을 유지합니다. context/cancellation cause를
보존하고 별도 replay·wait·rollback을 추가하지 않습니다.

Backup의 `_max_microversion = "3.64"`는 Python에서 session default가 없을 때 협상하는 상한입니다.
명시적 default는 우선하며 자동 cap하지 않습니다. Go의 supplied client는 선택한 microversion을 유지하고
자동 3.64 cap·협상·upgrade를 추가하지 않습니다. 필요하면 Connection의
[명시적 microversion·범위 협상](../docs/microversions.md)을 사용합니다. ordinary Boolean/project equality는
exact JSON decimal 정책이므로 Python float rounding·underflow와 차이가 있습니다. untyped raw numeric
spelling, Unicode digit runtime table와 configurable CPython integer digit limit도 명시적인 표현 경계입니다.
Python mutable Resource/session/stale dict cache를 모든 형태로 재현하는 기능은 아닙니다.
이 세 직접 read helper와 [기존 v3 Backup API](v3/README.md), v2·metadata·create/delete·restore·
import/export/action 및 전체 SDK의 지원 검토는 각각 독립적입니다.
