# 볼륨 snapshot 조회

`ListVolumeSnapshots`, `SearchVolumeSnapshots`, `GetVolumeSnapshot`, `GetVolumeSnapshotByID`는
Cinder v3 snapshot을 같은 옵션·nullable 값·오류 방식으로 읽습니다. Connection은 인증과
cached Cinder 선택을 담당하고, 직접 package 함수는 이미 준비한 `ServiceClient`를 받습니다.
호출자는 Gophercloud 옵션 builder를 구현하지 않고 필요한 `With...` 옵션만 지정합니다.

| 실제 openstacksdk cloud helper | Connection | 직접 package 함수 |
|---|---|---|
| `conn.list_volume_snapshots()` | `conn.ListVolumeSnapshots(ctx)` | `blockstorage.ListVolumeSnapshots(ctx, cinder)` |
| `conn.search_volume_snapshots("nightly*", filters)` | `conn.SearchVolumeSnapshots(ctx, blockstorage.SearchVolumeSnapshotsRequest{NameOrID: "nightly*"}, options...)` | `blockstorage.SearchVolumeSnapshots(ctx, cinder, request, options...)` |
| `conn.get_volume_snapshot("nightly", filters=None)` | `conn.GetVolumeSnapshot(ctx, blockstorage.GetVolumeSnapshotRequest{NameOrID: "nightly"}, options...)` | `blockstorage.GetVolumeSnapshot(ctx, cinder, request, options...)` |
| `conn.get_volume_snapshot_by_id("snapshot-id")` | `conn.GetVolumeSnapshotByID(ctx, blockstorage.GetVolumeSnapshotByIDRequest{ID: "snapshot-id"}, options...)` | `blockstorage.GetVolumeSnapshotByID(ctx, cinder, request, options...)` |

Connection 생성은 실제 `sdk.Connect` 또는 `sdk.FromProvider`를 사용합니다. `WithCloud`는
clouds.yaml의 항목을 선택합니다. 이미 인증된 provider를 넘기는 `FromProvider`에는
필요한 region·endpoint·microversion·owned location 옵션을 지정할 수 있습니다.
다음 독립 Go 예제는 Connection의 네 함수와 같은 이름의 package 대안을 함께 보여줍니다.

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

func Configured(ctx context.Context, cloudName string) (*sdk.Connection, error) {
    return sdk.Connect(ctx, sdk.WithCloud(cloudName))
}

func Adopt(provider *gophercloud.ProviderClient) (*sdk.Connection, error) {
    return sdk.FromProvider(provider, sdk.WithRegion("RegionOne"))
}

func InspectPages(pages []*blockstorage.VolumeSnapshotsPage) {
    for _, page := range pages {
        fmt.Println("admitted snapshot page:", page.StatusCode, "bytes:", len(page.Body))
    }
}

func Inspect(result *blockstorage.GetVolumeSnapshotResult) {
    if result == nil { return }
    InspectPages(result.Pages)
    if result.Observed != nil {
        fmt.Println("member response:", result.Observed.StatusCode, len(result.Observed.Body))
    }
    fmt.Println("requested ID:", result.RequestedID, "seeded logical ID:", result.SeededID)
    if result.Snapshot != nil {
        owned := result.Snapshot.Clone()
        fmt.Println("actual response ID JSON:", string(owned.Body["id"]))
        var wire map[string]json.RawMessage
        if err := owned.Decode(&wire); err != nil {
            fmt.Println("optional wire projection:", err)
        } else {
            fmt.Println("wire fields:", len(wire))
        }
    }
}

func ViaConnection(ctx context.Context, conn *sdk.Connection) error {
    listed, err := conn.ListVolumeSnapshots(ctx,
        blockstorage.WithVolumeSnapshotListFilters(json.RawMessage(
            `{"status":"available","volume_id":"volume-id","description":"nightly"}`)))
    if listed != nil { InspectPages(listed.Pages) }
    if err != nil { return err }
    fmt.Println("complete normalized list:", string(listed.Value), "raw rows:", len(listed.Snapshots))

    searched, err := conn.SearchVolumeSnapshots(ctx,
        blockstorage.SearchVolumeSnapshotsRequest{NameOrID: "nightly*"},
        blockstorage.WithVolumeSnapshotSearchFilters(json.RawMessage(`{"status":"available"}`)))
    if searched != nil { InspectPages(searched.Pages) }
    if err != nil { return err }
    fmt.Println("selected normalized snapshots:", string(searched.Value))

    selected, err := conn.GetVolumeSnapshot(ctx,
        blockstorage.GetVolumeSnapshotRequest{NameOrID: "nightly"})
    Inspect(selected) // Physical proof remains useful when err is nonnil.
    if err != nil { return err }
    fmt.Println("selected normalized JSON:", string(selected.Value))

    byID, err := conn.GetVolumeSnapshotByID(ctx,
        blockstorage.GetVolumeSnapshotByIDRequest{ID: "snapshot-id"})
    Inspect(byID)
    return err
}

func ProjectNames(ctx context.Context, conn *sdk.Connection) error {
    result, err := conn.SearchVolumeSnapshots(ctx,
        blockstorage.SearchVolumeSnapshotsRequest{},
        blockstorage.WithVolumeSnapshotSearchExpression("[].name"))
    if result != nil { InspectPages(result.Pages) }
    if err != nil { return err }
    fmt.Println("arbitrary expression JSON:", string(result.Value))
    return nil
}

func FirstPage(ctx context.Context, conn *sdk.Connection) error {
    detailed, paginated, max := false, false, 10
    result, err := conn.ListVolumeSnapshots(ctx,
        blockstorage.WithVolumeSnapshotListOptions(blockstorage.VolumeSnapshotListOpts{
            Detailed: &detailed, Paginated: &paginated, MaxItems: &max,
            Headers: map[string]string{"X-Request-ID": "snapshot-read-example"},
        }))
    if result != nil { InspectPages(result.Pages) }
    return err
}

func Direct(ctx context.Context, cinder *gophercloud.ServiceClient) error {
    location := resource.CloudLocation{}
    listed, err := blockstorage.ListVolumeSnapshots(ctx, cinder,
        blockstorage.WithVolumeSnapshotListLocation(location))
    if listed != nil { InspectPages(listed.Pages) }
    if err != nil { return err }

    searched, err := blockstorage.SearchVolumeSnapshots(ctx, cinder,
        blockstorage.SearchVolumeSnapshotsRequest{NameOrID: "nightly*"},
        blockstorage.WithVolumeSnapshotSearchLocation(location))
    if searched != nil { InspectPages(searched.Pages) }
    if err != nil { return err }

    // Explicit {} filters choose full Search, rather than ordinary GET-first find.
    selected, err := blockstorage.GetVolumeSnapshot(ctx, cinder,
        blockstorage.GetVolumeSnapshotRequest{NameOrID: "nightly"},
        blockstorage.WithVolumeSnapshotSearchFilters(json.RawMessage(`{}`)))
    Inspect(selected)
    if err != nil { return err }

    byID, err := blockstorage.GetVolumeSnapshotByID(ctx, cinder,
        blockstorage.GetVolumeSnapshotByIDRequest{ID: "snapshot-id"},
        blockstorage.WithVolumeSnapshotReadOptions(blockstorage.VolumeSnapshotReadOpts{Location: &location}))
    Inspect(byID)
    return err
}
```

실제 Python 호출은 `snapshots = conn.list_volume_snapshots(filters={"status": "available"})`,
`matches = conn.search_volume_snapshots("nightly*", {"status": "available"})`,
`snapshot = conn.get_volume_snapshot("nightly")`,
`snapshot = conn.get_volume_snapshot_by_id("snapshot-id")`입니다. 비교 기준은
[고정한 cloud 구현](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L546)이며,
[목록 helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L710)와
[Search helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L864)는
서로 다른 필터 경로를 사용합니다.

기본 List는 상세 `/snapshots/detail`을 끝까지 읽습니다. `Detailed: nil`과 `Paginated: nil`은
true이며 `WithVolumeSnapshotListDetailed(false)`는 `/snapshots`,
`WithVolumeSnapshotListPagination(false)`는 첫 페이지를 선택합니다. 기본 page limit·maximum·
호출별 microversion·추가 header·expression을 만들어 넣지 않습니다. 목록 Accept 기본값은
`application/json`입니다. typed `Headers: nil`은 raw filters의 `headers`를 덮어쓰지 않습니다.
반면 nonnil 빈 map은 raw header control을 `{}`로 교체해 해당 raw 추가 header를 지우고
기본 `Accept: application/json`을 유지합니다. 이는 selected native client의 이미 capture한
ordinary header까지 지우는 옵션은 아닙니다. `WithVolumeSnapshotListOptions`는 complete 정책을 교체하고,
개별 `WithVolumeSnapshotList...` factory는 해당 값만 지정합니다. Search와 ByID도 각각
`WithVolumeSnapshotSearchOptions`, `WithVolumeSnapshotReadOptions`를 사용합니다.
`PrepareVolumeSnapshotListOptions`·`PrepareVolumeSnapshotSearchOptions`·
`PrepareVolumeSnapshotReadOptions`는 context와 원본 callback을 처리하며 서비스 I/O는 하지 않습니다.

List의 `WithVolumeSnapshotListFilters`는 raw JSON dictionary를 source 순서대로 분류합니다.
`name`, `status`, `volume_id`, `project_id`, `limit`, `marker`, `offset`, `sort_dir`, `sort_key`,
`sort`와 `all_projects`/`all_tenants`는 서버 query입니다. `all_projects`는 wire `all_tenants`로
변환하고 source의 Proxy-bound 값과 query alias 우선순위를 적용합니다. 값은 string으로 강제하지
않으며 Requests 계열의 query 인코딩 규칙을 사용합니다.
`consumes_quota`, `created_at`, `description`, `group_snapshot_id`, `is_forced`, `progress`, `size`,
`updated_at`, `user_id`, `id`, `metadata`는 받아온 logical row에 적용하는 local 조건입니다.
중첩 dictionary는 재귀 비교하며, falsey actual dictionary는 빈 expected dictionary에도 일치하지 않습니다.
이름이 같은 값은 JSON-domain의 Python equality 정책으로 비교합니다.

List에서 알려지지 않은 key는 source처럼 버립니다. raw wire `force`나 extended progress key는
local filter 속성 `is_forced`·`progress`를 대신하지 않습니다. `project_id`는 server query이고
extended project wire key를 별도 server query alias로 간주하지 않습니다.
`AllowUnknownParams`는 concrete 옵션으로 제공하지만 실제
[Resource.list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2165)는
query validation에 항상 `allow_unknown_params=True`를 전달하므로 unknown key를 strict error로
바꾸는 스위치가 아닙니다. 이는 Search의 unknown filter 규칙과 다릅니다.

Search는 상세 목록을 query 없이 먼저 완료한 뒤 normalized 16-field view에서 이름·ID와 필터를
적용합니다. `NameOrID`는 각 row의 정확한 ID·이름 또는 glob 일치를 선택하며 원래 순서와 중복을 유지합니다.
Search의 `status` 필터도 server query로 이동하지 않습니다. 도달한 unknown mapping key는 local 오류이고,
앞선 조건이 일치하지 않거나 이름·ID 선택에 row가 남지 않으면 사용하지 않는 조건은 소비하지 않습니다.
`force` 대신 normalized `is_forced`를 지정합니다. expression은 JMESPath를 normalized JSON list에 적용합니다.
결과는 배열뿐 아니라 null·숫자·문자열·object일 수 있으므로 `Value`를 확인하며, expression에
서버 row의 `RawResource` 연결을 만들어 붙이지 않습니다. 빈 raw Search filter 문자열 `""`은
source의 falsey filter처럼 추가 표현식을 평가하지 않습니다. `WithVolumeSnapshotSearchExpression("")`도
같은 Search filter 표현을 사용하므로 이 규칙을 유지합니다.

Get의 필터를 생략하거나 JSON null로 지정하면 ordinary exact find입니다. 안전한 문자열은
`/snapshots/{identity}`를 먼저 읽고, 깨끗한 400·403·404에서만 상세 목록의 `name` query로 fallback합니다.
이름과 ID가 정확히 같아야 하며 glob 검색은 하지 않습니다. 두 번째 일치에서 `ErrAmbiguous`로
종료하되, 한 개만 일치하면 뒤 페이지까지 성공적으로 끝나야 그 값을 반환합니다. 따라서 뒤 페이지의
HTTP·descriptor·pagination 오류를 먼저 찾은 한 row로 숨기지 않습니다. 완료된 부재는 오류 없는
nil `Value`·`Snapshot`입니다. unsafe path 이름은 literal `name` query 목록으로만 찾는 Go route 경계이며
입력을 trim하지 않습니다.

`GetVolumeSnapshot`에 `{}`, `[]`, false, 0, 빈 문자열처럼 **nonnull** filter를 명시하면 전체 Search를
수행합니다. Search 결과에는 Python helper의 truthiness·`len`·index 0 규칙을 적용합니다.
falsey 결과는 부재, length가 1보다 크면 ambiguity, 한 값은 index 0 선택입니다.
expression이 항상 Snapshot을 반환한다고 가정하지 않습니다. 숫자나 dictionary처럼 index/length가
맞지 않는 결과는 local 선택 오류일 수 있으며 선택된 JSON false·0·빈 문자열도 raw 값으로 구분합니다.
Python의 해당 filters 인자는 deprecated warning을 내지만 Go는 library 옵션으로 유지합니다.

ByID는 literal ID의 고정 member GET만 수행하고 이름 검색·목록 fallback을 하지 않습니다.
404도 오류입니다. Python ByID docstring의 “없으면 None”와 달리 실제
[Proxy.get_snapshot](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L119)
호출은 HTTP 오류를 반환합니다. Go ID는 UTF-8의 단일 unescaped path segment여야 하며 공백·
제어 문자·route 변경 문자를 거부합니다. 임의 Python 객체·mutable Resource 입력과 같은 route coercion은 제공하지 않습니다.

member의 canonical `{"snapshot":{...}}`와 flat object를 읽습니다. UTF-8이 유효한 empty/malformed JSON은
source의 `response.json()` ValueError tolerance처럼 bare nullable Snapshot을 만들 수 있습니다.
유효한 null·scalar·array, 잘못된 envelope, invalid UTF-8와 descriptor 실패는 성공이 아닙니다.
member 응답에서 `id`가 누락되면 source 입력 ID를 **logical `Value`에만** 유지하고 `RequestedID`와
`SeededID`로 표시합니다. 실제 `Snapshot.Body`에는 없는 `id`를 추가하지 않습니다.
응답의 명시적 null·false·0·빈 문자열·컨테이너 ID는 그대로 유지하며 seed하지 않습니다.
응답 ID가 요청 ID와 같아야 성공하는 규칙도 추가하지 않습니다. List row에는 입력 ID를 seed하지 않습니다.

List는 canonical `snapshots` key를 요구합니다. 그 값이 배열이 아니면 source처럼 한 row로 감싸며,
사용할 row가 object가 아니면 실패합니다. 전체 response JSON은 먼저 파싱하고 row·link는 실제
소비할 때 검증합니다. 빈 배열은 즉시 EOF이며 advertised next를 읽지 않습니다.
raw `self`는 logical view에서 제외하지만 실제 proof에는 남깁니다. List row의
`connection`, `microversion`, `_synchronized`는 값이 null·false여도 source constructor keyword와
충돌하므로 오류입니다. member 응답의 같은 key는 runtime argument가 아닌 unknown raw field입니다.
nested `metadata`의 같은 이름은 평범한 dictionary 값으로 유지합니다.

`Value`의 nullable schema는 아래 16개입니다. 누락과 null은 null이며 native Snapshot의
string·time.Time·int 모델로 canonical 결과를 좁히지 않습니다.

| normalized 속성 | 변환 |
|---|---|
| `consumes_quota`, `created_at`, `description`, `group_snapshot_id` | untyped raw JSON |
| `is_forced` ← `force` | strict BoolStr |
| `progress` ← `os-extended-snapshot-attributes:progress` | untyped raw JSON |
| `project_id` ← `os-extended-snapshot-attributes:project_id` | untyped raw JSON |
| `size` | source integer descriptor |
| `status`, `updated_at`, `user_id`, `volume_id`, `id`, `name` | untyped raw JSON |
| `metadata` | nullable plain dictionary |
| `location` | independently computed current/foreign project location |

`is_forced`에는 descriptor의 implicit false 기본값이 없습니다. null은 null이며 bool 또는
대소문자 구분 없는 정확한 `"true"`·`"false"` 문자열만 변환합니다. 0·1, padded 문자열,
다른 문자열과 container는 descriptor 오류입니다. `size`는 boolean과 arbitrary integer를
유지하고 finite parsed float64를 0 방향으로 truncate합니다. 예를 들어 `1e30`은
`1000000000000000019884624838656`이며 int64 범위로 제한하지 않습니다. unsigned Unicode decimal
문자열은 변환하고 signed·공백·nonnumeric 문자열과 container는 0으로 변환합니다.
nondecimal digit-only 문자열과 무한대로 overflow하는 float는 실패할 수 있습니다.
metadata는 null 그대로, object는 그대로, 그 밖의 nonnull 값은 plain `{}`입니다.
이 규칙은 [pinned Snapshot descriptors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/snapshot.py#L62)와
[실제 field conversion](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86)을 기준으로 합니다.

JSON duplicate key는 마지막 값으로 교체하되 최초 삽입 위치를 유지한 뒤, 그 dictionary 순서로
normalized/wire alias를 소비합니다. `{"force":true,"is_forced":false,"force":true}`의 logical
`is_forced`는 false입니다. 사용하지 않는 overwritten alias의 잘못된 값은 변환하지 않습니다.
알려지지 않은 응답 속성과 원래 number spelling은 raw `Snapshot`에 남고 normalized view에서는 제외합니다.
source의 최종 constructor `dict.update(self, self.to_dict())`가 모든 descriptor를 변환하므로
List local filter나 Find의 이름·ID 검사 전에 invalid size/force가 실패합니다. 제외될 row라는
이유로 이 오류를 건너뛰지 않습니다. Search도 목록을 완료한 뒤 필터를 처리하므로 같은 순서를 유지합니다.

Connection은 원본 옵션을 한 번 실행하고 명시적 Location 또는 `CurrentLocation`을 복사한 뒤
cached Cinder v3를 선택합니다. 다른 서비스와 프로젝트 lookup을 추가하지 않습니다. 직접 package 함수는
selected source를 original callback보다 먼저 capture합니다. factory·원본 옵션 slice·pointer·map·raw JSON을
소유하며 나중의 caller 변경이 진행 중인 정책을 바꾸지 않습니다.
Snapshot에는 project descriptor가 있으므로 빈/unknown-only/location-only 응답도 location을 다시 계산합니다.
Snapshot에는 zone descriptor가 없으므로 **zone은 항상 null**입니다. caller의 `Location.Zone`도
Snapshot의 source zone을 대신하지 않습니다. raw `location`·`availability_zone`로 current scope를 추론하지 않습니다.
falsey 또는 현재 scope와 같은 project는 configured 이름·domain을 유지하고, truthy foreign project는
실제 raw ID만 유지하며 configured 이름·domain을 null로 지웁니다. 잘못된 owned Location은
local `ErrInvalidOption`입니다. 현재 helper는 첫 HTTP 전에 location을 검증하고, Connection은
Cinder getter 전에 검증합니다. 필터·선택처럼 응답 뒤 발생하는 local 오류는 이미 admitted된
`Pages`·`Observed`를 보존하면서 그 응답을 새 `ResponseError`로 빌리지 않습니다.

pagination은 body top-level `links`의 **존재**를 먼저 보고, 없을 때 `snapshots_links`를 봅니다.
배열은 앞에서부터 exact `rel: "next"`와 존재하는 `href`를 만나면 선택하고 뒤의 unused tail을 읽지 않습니다.
소비한 malformed prefix는 오류이며 selected falsey href는 뒤 item 대신 body top-level `next`,
HTTP Link 순서로 fallback합니다. body `links`가 dictionary이면 source가 각 항목을 one-key
object로 순회하므로 `rel`과 `href`가 함께 성립하지 않습니다. `{"next":"..."}`도 next 링크로
해석하지 않으며 이후 top-level `next`·HTTP Link 경로를 사용합니다.
HTTP Link는 pinned Resource가 `response.links['next']['uri']`를 사용하는 반면
[Requests의 실제 parser](https://requests.readthedocs.io/en/latest/_modules/requests/utils/#parse_header_links)는
`url`을 제공하는 차이를 Go가 교정합니다. [Requests Response.links](https://requests.readthedocs.io/en/latest/_modules/requests/models/#Response.links)의
단일 `rel` 문자열 dictionary와도 달리, Go는 quoted delimiter·escape를 처리하는 strict header parser로
여러 relation token을 읽고 대소문자를 무시해 `next`를 찾습니다. 여러 Link header와 next가 있으면
마지막 next를 선택합니다. 실제로 사용하는 잘못된 문법·UTF-8·제어 문자는 오류이고, body에서
truthy next를 선택했으면 사용하지 않는 HTTP Link는 검증하지 않습니다. 이는 source와 동일한
parser라는 주장이 아니라 사용할 수 있게 만든 Go repair입니다. relative link는 effective URL에 대해 해결하며,
Python의 relative `/v` prefix 제거와 query-only href 중복 대신 실제 path와 canonical query를 한 번 유지합니다.
source처럼 이전 marker·limit를 제거하고 advertised blank 값을 제외합니다. 남은 advertised key는
이전 key를 교체하고 생략된 nonpagination key는 유지합니다. 같은 origin·escaped collection path를
검사하며 source의 same-marker 검사와 함께 complete canonical URL cycle을 차단합니다.

처음 선택한 raw `limit`가 truthy이고 next가 없으면, source처럼 마지막 소비 row의 normalized ID를
marker로 사용해 collection의 다음 요청을 만듭니다. 짧은 페이지도 이 동작을 적용합니다.
typed `MaxItems`는 `*int`이며 raw filters의 `max_items`는 JSON number·boolean·null을 받습니다.
소수 maximum도 지원합니다. 예를 들어 1.5이면 소비 counter 2부터 멈추며 true는 1,
false·0·null은 무제한입니다. 음수의 nonzero 값은 GET과 전체 JSON 파싱을 수행한 뒤 첫 row 전에
멈춥니다. maximum이 truthy이고 기존 limit가 falsey이면 initial query limit도 그 raw maximum으로
설정합니다. 서버가 소수·음수 등의 query를 거부하면 실제 HTTP 오류를 반환합니다.
maximum은 local filter로 반환하지 않은 row를 포함한 **소비한 raw row 수**를 셉니다.
Go는 raw maximum의 decimal 값·truthiness를 정확하게 유지해 정수 counter와 비교합니다.
Python JSON float의 rounding/underflow 때문에 경계가 달라질 수 있습니다. 예를 들어
`1.0000000000000001`은 Go에서 counter 1을 넘기지만 Python float는 1로 반올림할 수 있고,
아주 작은 nonzero maximum은 Python에서 0으로 underflow해 무제한이 될 수 있습니다.

maximum 검사는 HTTP 전이 아니라 **각 다음 raw row 전**입니다. 같은 페이지의 unused tail이
있으면 그 전에 멈추지만, 페이지의 마지막 row에서 정확히 maximum에 도달하면 continuation을
소비하고 다음 페이지를 읽은 뒤 첫 row 전에 멈출 수 있습니다. 따라서 그 경로의 잘못된 link나
다음 응답 JSON 실패를 숨기지 않습니다. marker 누락/null과 이전 marker가 같으면 source의 loop 오류가
발생할 수도 있습니다. `Paginated: false`는 첫 페이지 뒤 이 추가 요청을 막습니다.
[실제 Resource counter와 paging](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2256)을 비교 기준으로 삼습니다.

List의 inherited controls는 `WithVolumeSnapshotListPagination`, `MaxItems`, `Microversion`, `Headers`,
`Expression`, `AllowUnknownParams`, `ConflictingAttrs` 또는 complete `ListOpts`로 선택합니다.
raw filter의 control보다 해당 explicit typed 값이 우선합니다. `__conflicting_attrs`는 source처럼
Proxy-bound 인자 선택 뒤 Resource 단계의 속성을 overlay합니다. late `headers`·`microversion`·
`max_items`·`allow_unknown_params`는 그 단계의 controls이며 late `paginated`·`base_path`·`session`·
`cls`는 이미 전달한 인자와 충돌합니다. late `details`·`resource_type`·`self`·`jmespath_filters`는
unknown Resource parameter여서 이미 선택한 route/expression을 다시 바꾸지 않습니다.
원래 top-level raw filters의 `details`·`base_path`·`resource_type`·`self` 충돌도 오류로 처리합니다.

List expression은 명시적인 Go repair입니다. pinned
[Proxy._list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866)는
Resource generator를 곧바로 JMESPath에 넘기므로 일반 projection이 HTTP 전에 null/오류가 되거나
generator 자체를 반환할 수 있습니다. Go는 완성한 normalized JSON list에 expression을 적용하고
finite arbitrary JSON `Value`를 반환합니다. 결과·오류·HTTP 시점이 이 legacy 경로와 달라집니다.
이는 eager list 뒤 expression을 평가하는 실제 cloud **Search** 경로와 구별합니다.
`WithVolumeSnapshotListExpression("")`는 명시적인 expression이며 전체 목록을 읽은 뒤 engine의
빈 표현식 syntax error를 반환합니다. 이때 `Value`·`Snapshots`는 nil이고 실제 `Pages`는 유지됩니다.
반면 raw legacy `jmespath_filters: ""` 등 falsey control은 표현식을 건너뜁니다. 따라서 typed
빈 List 표현식과 raw falsey legacy control, Search의 빈 raw filter는 각각의 경로로 이해해야 합니다.

read status는 source의 observable 400 미만, 즉 100..399를 받습니다. native leaf의 HTTP 200만으로
이 helper를 좁히지 않습니다. 204 empty member는 nullable/seeded Snapshot이 될 수 있지만,
204 empty **List**는 JSON 파싱 오류입니다. 400+는 native body·header·status cause를 유지합니다.
`Pages`는 실제 admitted list 응답, `Observed`는 실제 admitted member 응답이며 성공 논리 결과를
보증하지 않습니다. read·Close·cancel·descriptor·pagination·source 실패에도 현재 실제 증거와
앞서 완료된 증거를 유지합니다. `Value`·`Snapshots`·`Snapshot`은 complete 성공 때만 발표하므로
실패한 목록의 부분 logical rows를 성공 결과로 사용하지 않습니다. 일반 List·Search가 정상적으로
빈 목록을 완료하면 `Value`는 JSON `[]`이고 `Snapshots`는 nonnil 빈 slice입니다. 표현식으로
성공한 결과에는 raw row 연결을 만들지 않으므로 결과가 `[]`여도 `Snapshots`는 nil입니다.
목록의 atomic 실패는 `Value`·`Snapshots`가 nil이며, 일반 Get의 정상 부재는 `Value`·`Snapshot`가 nil입니다.
필터·expression·owned Location·선택 오류는 local 오류이며 이전 physical 응답을 새 HTTP 실패로 만들지 않습니다.

header 이름은 ASCII HTTP token이고 값은 UTF-8이어야 하며, 대소문자 alias가 다른 값으로 충돌하면 거부합니다.
인증·Host·Cookie·hop-by-hop/framing header는 라이브러리가 소유합니다. Cinder version header는
selected effective microversion과 일치해야 하며 다른 서비스의 version header를 받지 않습니다.
호출별 microversion과 headers는 복사한 request source에 적용하고 원본 native client를 변경하지 않습니다.
원본 provider·endpoint·ResourceBase·type·microversion이 바뀌면 terminal 오류입니다. ordinary header는
capture한 값, auth token은 live provider 값을 사용하고 native retry/reauth/redirect도 고정 GET·URL·
query와 original status policy를 유지해야 합니다. Go context와 custom cancellation cause를 보존하며
SDK가 별도 orchestration replay·wait·rollback을 추가하지 않습니다.

Snapshot의 `_max_microversion = "3.65"`는 Python에서 default microversion이 **없을 때** server와
협상하는 상한입니다. 명시적 session default는 우선하며 자동 cap하지 않습니다. Go supplied-client
helper는 selected 설정을 유지하며 자동 3.65 협상·upgrade를 추가하지 않습니다. 필요하면 Connection의
[명시적 microversion 또는 범위 협상](../docs/microversions.md)을 선택합니다.
Go의 concrete `*bool`, `*int`, `*string` controls는 각 typed 필드의 도메인을 사용합니다.
raw JSON `max_items`의 소수·음수·number/boolean/null 지원은 별도로 유지하며, 이를 float control을
지원하지 않는다는 의미로 좁히지 않습니다. 임의 Python runtime 객체·mutable Resource·custom session과
JSON 도메인 밖의 control은 대체하지 않습니다. Unicode 16.0 digit table와
CPython의 configurable integer digit-limit도 runtime 경계입니다. untyped number spelling·exact decimal
truthiness/equality는 Python JSON float의 rounding/underflow와 다를 수 있습니다. raw proof·Clone·Decode·
단계별 owned 결과는 Go 표현이며 Python mutable Resource의 stale dict cache를 재현하지 않습니다.
일반 Resource/session/cache와 별도 snapshot create/delete/action의 지원 상태는 이 네 직접 read helper와
독립적으로 검토합니다. [기존 v3 snapshot API](v3/snapshots/README.md)와
[metadata 범위](metadata/README.md)는 계속 별도 기능입니다.
