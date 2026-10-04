# 볼륨 limits 조회

`Connection.GetVolumeLimits`는 Cinder v3의 현재 limits를 읽습니다. 프로젝트 이름 또는 ID를
지정하면 Identity v3에서 프로젝트를 먼저 찾고, 그 결과의 `id`를 `project_id` query로 전달합니다.
기본값·프로젝트 조회·nullable 응답 변환은 라이브러리가 담당합니다.

| openstacksdk cloud | Go |
|---|---|
| `conn.get_volume_limits()` | `conn.GetVolumeLimits(ctx, blockstorage.GetVolumeLimitsRequest{})` |
| `conn.get_volume_limits("project-name-or-id")` | `conn.GetVolumeLimits(ctx, blockstorage.GetVolumeLimitsRequest{NameOrID: "project-name-or-id"})` |
| Connection의 현재 location | 옵션 생략 |
| 호출별 owned location | `WithGetVolumeLimitsLocation` 또는 `WithGetVolumeLimitsOptions` |
| 프로젝트 조회와 limits 응답의 별도 증거 | `result.Project`와 `result.Observed` |

Connection은 `sdk.Connect` 또는 `sdk.FromProvider`로 준비합니다.
다음은 Connection과 직접 package 호출을 함께 보여주는 독립 Go 예제입니다.

```go
package example

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/gophercloud/gophercloud/v2"
    nativeLimits "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/limits"
    sdk "gophercloudsdk"
    "gophercloudsdk/blockstorage"
    "gophercloudsdk/resource"
)

func Inspect(result *blockstorage.GetVolumeLimitsResult) {
    if result == nil { return }
    if result.Project != nil {
        fmt.Println("project ID JSON:", string(result.Project.ID),
            "seeded:", result.Project.SeededID,
            "lookup pages:", len(result.Project.Pages))
        if result.Project.Observed != nil {
            fmt.Println("project member status:", result.Project.Observed.StatusCode)
        }
    }
    fmt.Println("requested project ID JSON:", string(result.RequestedProjectID))
    if result.Observed != nil {
        fmt.Println("limits status:", result.Observed.StatusCode,
            "raw bytes:", len(result.Observed.Body))
    }
}

func Current(ctx context.Context, conn *sdk.Connection) error {
    result, err := conn.GetVolumeLimits(ctx, blockstorage.GetVolumeLimitsRequest{})
    Inspect(result) // Completed physical proof also remains available on error.
    if err != nil { return err }
    fmt.Println("normalized nullable JSON:", string(result.Value))
    owned := result.Limits.Clone()
    fmt.Println("original absolute JSON:", string(owned.Body["absolute"]))

    // This optional native projection enforces the native int/string schema.
    // A projection error does not invalidate the successful raw workflow.
    var projection nativeLimits.Limits
    if projectionErr := owned.Decode(&projection); projectionErr != nil {
        fmt.Println("native projection:", projectionErr)
    } else {
        fmt.Println("native maximum volumes:", projection.Absolute.MaxTotalVolumes)
    }
    return nil
}

func OtherProject(ctx context.Context, conn *sdk.Connection, nameOrID string) error {
    result, err := conn.GetVolumeLimits(ctx,
        blockstorage.GetVolumeLimitsRequest{NameOrID: nameOrID})
    Inspect(result)
    if err != nil { return err }
    fmt.Println("normalized limits:", string(result.Value))
    return nil
}

func WithLocation(ctx context.Context, conn *sdk.Connection) error {
    cloud, project := "configured-cloud", "authenticated-project"
    location := resource.CloudLocation{
        Cloud: &cloud, Zone: json.RawMessage(`"owned-zone"`),
        Project: resource.CloudProject{ID: json.RawMessage(`"scope-id"`), Name: &project},
    }
    result, err := conn.GetVolumeLimits(ctx,
        blockstorage.GetVolumeLimitsRequest{NameOrID: "other-project"},
        blockstorage.WithGetVolumeLimitsOptions(blockstorage.GetVolumeLimitsOpts{
            Location: &location,
        }))
    Inspect(result)
    if err != nil { return err }
    fmt.Println("limits with owned authenticated location:", string(result.Value))
    return nil
}

func Direct(ctx context.Context, cinder, identity *gophercloud.ServiceClient) error {
    // Empty input needs no Identity client and sends no project filter.
    current, err := blockstorage.GetVolumeLimits(ctx, cinder, nil,
        blockstorage.GetVolumeLimitsRequest{})
    Inspect(current)
    if err != nil { return err }
    selected, err := blockstorage.GetVolumeLimits(ctx, cinder, identity,
        blockstorage.GetVolumeLimitsRequest{NameOrID: "project-name-or-id"},
        blockstorage.WithGetVolumeLimitsLocation(resource.CloudLocation{}))
    Inspect(selected)
    return err
}
```

실제 Python 사용은 `limits = conn.get_volume_limits()` 또는
`limits = conn.get_volume_limits("project-name-or-id")`입니다.
[pinned cloud get_volume_limits](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L348)는
값이 있는 입력을 `find_project(..., ignore_missing=False)`에 전달한 뒤
[pinned Cinder Proxy.get_limits](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1918)를 호출합니다.
Python docstring의 “없으면 None” 설명과 달리, 이 실제 경로에서 완료된 프로젝트 부재는 예외이며
limits GET의 HTTP 실패도 숨기지 않습니다.

빈 Go `NameOrID`는 Identity를 선택하지 않고 query 없는 `/limits`를 한 번 읽습니다.
현재 token의 프로젝트 ID를 query로 보충하지 않습니다. 값이 있는 입력은 ID처럼 보여도
Identity에서 먼저 조회합니다. 문자열을 trim하지 않으며 공백·path 문자를 포함한 이름은
정확한 `name` query 목록으로만 찾습니다. invalid UTF-8·제어 문자는 local 입력 오류입니다.
안전한 identity는 member GET을 먼저 시도하고, 깨끗한 400·403·404 응답에서만 목록으로 fallback합니다.
목록에서 ID 또는 이름이 정확히 일치해야 하며 두 번째 일치가 나타나면 `ErrAmbiguous`,
완료된 부재는 `ErrNotFound`입니다. 프로젝트 조회가 실패하면 Cinder를 선택하지 않습니다.
프로젝트 목록의 raw `self`는 source처럼 constructor 전에 제외합니다.
반면 목록 행에 `connection`·`microversion`·`_synchronized` key가 있으면 값이 null이나 false여도
명시적으로 전달한 runtime keyword와 충돌하므로 accepted descriptor 오류입니다.
이는 nested Limits constructor의 truthy `connection` 규칙과 다른 경로입니다.
현재 physical 페이지와 앞서 완료된 증거는 유지하며 잘못된 행을 건너뛰지 않습니다.

프로젝트 목록은 빈 페이지에서 continuation을 읽지 않고 종료합니다. body의 top-level `links`가
존재하면 `projects_links` 대신 선택하며, source의 rel/href 배열·top-level `next`와 함께
Keystone의 실제 `links.next` object 및 HTTP Link를 읽습니다. 뒤의 두 형식은 Go compatibility repair입니다.
pinned [Resource paging](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2345)는
Keystone dict를 rel/href 형식으로 소비하지 못하고 `response.links['next']['uri']`를 사용하지만,
[Requests의 실제 Link parser](https://requests.readthedocs.io/en/latest/_modules/requests/utils/#parse_header_links)는 `url` key를 제공합니다.
Go의 Keystone `links.next` repair는 top-level `next`와 HTTP Link보다 먼저 선택됩니다.
이 때문에 원본에서 무시되던 dictionary next가 다른 continuation보다 우선하며, truthy nonstring next는 오류입니다.
Go는 relative link를 현재 URL에 대해 해결하고 query를 한 번 합칩니다.
source처럼 이전 `marker`·`limit`를 먼저 제거하고 advertised query에서 빈 값을 제외합니다.
남은 advertised key는 기존 값을 교체하며 생략된 nonpagination key는 유지합니다.
rel/href 배열은 source 순서로 필요할 때만 소비합니다. 앞선 item이 필요한 작업에서 null·scalar이면
오류이지만, 먼저 선택된 exact `next`+`href` 뒤의 사용하지 않는 tail은 검증하지 않습니다.
선택된 falsey href는 뒤의 rel/href item으로 넘어가지 않고 top-level `next`·HTTP Link 순서로 fallback합니다.
같은 origin·escaped collection path를 지키고 전체 canonical URL cycle을 차단하며,
caller가 지정하지 않은 page limit나 marker pagination을 만들어 내지 않습니다.

`Project.ID`와 `RequestedProjectID`는 입력을 string으로 변환한 값이 아니라 실제 조회 결과의 raw JSON입니다.
단, source의 seeded member 상태는 이 query workflow에서 명시적으로 유지합니다.
member 응답이 `id`를 생략했거나 허용된 empty/malformed JSON이면 `Project.ID`는 입력 문자열이고
`SeededID`가 true입니다. `Project.Project.Body`에는 실제 응답 필드만 남으므로 seed된 `id`를 추가하지 않습니다.
명시적 null·false·0·빈 문자열·컨테이너 ID는 그대로 유지하고 seed하지 않습니다.
목록으로 찾은 행은 입력 ID를 seed하지 않으며 `id` 누락은 JSON null입니다.

이 ID는 URL path가 아닌 query 값이므로 안전한 path ID 검사를 적용하지 않습니다.
문자열은 그대로 URL 인코딩하고 boolean은 `True`/`False`로 표현합니다. null은 query를 생략합니다.
컨테이너는 Requests의 첫 iterable 확장과 `urlencode(doseq=True)`의 두 번째 확장을 따라
같은 key를 반복합니다. 예를 들어 ID `[[1,2],[],null]`은 `project_id=1&project_id=2`,
ID `[{"a":1,"b":2}]`는 `project_id=a&project_id=b`, ID `[[null]]`은 `project_id=None`입니다.
그보다 깊은 값은 Python repr 규칙으로 표현합니다. JSON object key는 duplicate collapse 후
첫 삽입 위치의 순서를 유지합니다. 이 query 인코딩은
[Requests의 실제 parameter encoder](https://requests.readthedocs.io/en/latest/_modules/requests/models/#RequestEncodingMixin._encode_params)와
[`urlencode`의 doseq 동작](https://docs.python.org/3/library/urllib.parse.html#urllib.parse.urlencode)을 연결한 정책이며,
Requests 버전 자체는 openstacksdk source pin으로 고정되지 않습니다. Go float의 짧은 repr·Unicode runtime 경계는 별도로 남습니다.

Connection은 original 옵션을 한 번 실행한 직후 명시적 Location 또는 `CurrentLocation`을 복사합니다.
이후 필요한 cached Identity v3 조회를 완료하고 cached Cinder v3를 선택합니다.
Cinder getter가 실패해도 완료된 `Project` 증거를 반환합니다.
직접 package 호출은 선택된 source를 original 옵션보다 먼저 capture하며 빈 입력에서 Identity는
nil·잘못된 설정까지 모두 사용하지 않습니다. 옵션 함수·pointer·raw JSON·원본 옵션 slice는 복사되어
나중의 caller 변경이나 callback이 보관한 pointer가 요청 정책을 바꾸지 않습니다.

이 helper는 선택된 Cinder microversion을 그대로 사용합니다. 프로젝트 query를 보낼 때에도
자동 upgrade·3.39 minimum preflight·cap을 추가하지 않습니다.
empty·3.0·3.38·3.39·3.60과 이미 선택된 native client의 symbolic latest는 transport의 일관된 header와 함께 유지됩니다.
Connection의 `WithMicroversion` exact 옵션은 기존처럼 numeric version을 받습니다.
실제 Python [Resource._get_microversion](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1418)은
`session.default_microversion`이 있으면 그 값을 상한 없이 먼저 사용합니다.
`Limits._max_microversion = "3.39"`는 default가 없을 때 server discovery와 협상하는 상한입니다.
Go helper는 그 자동 server 협상을 수행하지 않고 선택된 native 설정을 보존합니다.
따라서 기존 설명의 “Python limits는 3.39로 cap한다”는 표현은 **default 없는 협상**으로 한정해야 합니다.

기존 [Cinder v3 limits singleton](v3/limits/README.md)의 `Fetch`, `InProject`, `CurrentProject`는
계속 별도 API입니다. 그 프로젝트 스코프는 명시적 `resource.Ref` ID의 Identity 조회를 생략하고
selected numeric 3.39+를 요구합니다. native `Get`은 query 옵션 없이 HTTP 200과 native int/string 모델을 사용합니다.
이 cloud helper를 그 스코프의 단순 wrapper로 설명하지 않습니다.
`RequestedProjectID`는 요청한 filter의 증거이며 서버가 응답의 프로젝트 정체성을 증명한 값은 아닙니다.

`Value`는 [pinned Limits hierarchy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/limits.py)의
nullable normalized JSON입니다. 최상위는 `absolute`, `rate`, `id`, `name`, `location` 다섯 필드입니다.
absolute 객체의 열 가지 numeric attribute는 다음과 같으며 inherited `id`·`name`·`location`도 포함합니다.

| normalized attribute | 원래 wire key |
|---|---|
| `max_total_backup_gigabytes` | `maxTotalBackupGigabytes` |
| `max_total_backups` | `maxTotalBackups` |
| `max_total_snapshots` | `maxTotalSnapshots` |
| `max_total_volume_gigabytes` | `maxTotalVolumeGigabytes` |
| `max_total_volumes` | `maxTotalVolumes` |
| `total_backup_gigabytes_used` | `totalBackupGigabytesUsed` |
| `total_backups_used` | `totalBackupsUsed` |
| `total_gigabytes_used` | `totalGigabytesUsed` |
| `total_snapshots_used` | `totalSnapshotsUsed` |
| `total_volumes_used` | `totalVolumesUsed` |

rate group는 `limits`, `regex`, `uri`, `id`, `name`, `location` 여섯 필드입니다.
`limits`의 wire alias는 `limit`이며, 각 rule은 `next_available`, `remaining`, `unit`, `value`,
`verb`, `id`, `name`, `location` 여덟 필드입니다. `next_available`의 wire alias는 `next-available`입니다.
ID·이름·timestamp·regex·URI·unit·verb는 untyped raw JSON을 유지하며 string·시간 타입으로 강제 변환하지 않습니다.

누락과 null은 null입니다. 실제 nested `{}`는 해당 Resource의 전체 nullable 필드를 얻지만
Resource descriptor의 nonobject 값은 plain `{}`가 됩니다. list descriptor는 nonlist를 한 행으로 감싸고,
list item의 null·scalar·list도 plain `{}`로 변환합니다. descriptor 자체가 null이면 list로 감싸지 않습니다.
numeric descriptor는 boolean과 arbitrary-size integer를 유지하고 finite parsed float64를 정수로 truncation합니다.
unsigned digit-only 문자열은 변환하며 일반적인 signed·공백·nonnumeric 값은 0입니다.
nondecimal digit-only·overflow 변환은 실패할 수 있습니다.
[pinned field conversion](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86)을 기준으로 하며,
Unicode 16.0 숫자 테이블과 CPython의 configurable integer digit-limit는 명시적 runtime 경계입니다.

normalized/wire alias는 Python JSON dict처럼 duplicate key가 마지막 값으로 교체되되 최초 위치를
유지한 뒤 그 순서대로 소비합니다. 이 때문에 마지막 **텍스트** alias가 항상 이기는 정책은 아닙니다.
unknown key는 normalized view에서 제외하고 원본 `Limits.Body`에서는 보존합니다.
raw field·number spelling과 HTTP 증거 보존은 Go가 추가하는 owned 결과입니다.

최상위 location은 authenticated/current snapshot 전체를 유지하며, 요청한 다른 프로젝트나
HTTP `location`·`project_id`·zone에서 추론하지 않습니다. 명시적 Zone도 유지합니다.
nested Resource에는 Connection이 없어 기본 location은 null입니다. nested object에 알려진 Body 필드가
전혀 없을 때만 raw `location`을 유지하고, 알려진 필드가 null인 경우에도 raw location을 제외합니다.
[Resource constructor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L538)의
nested `self`는 거부하고 truthy JSON `connection`도 descriptor 오류입니다.
falsey `connection`은 소비하여 무시하고 `_synchronized`·`microversion`은 normalized Body에 포함하지 않습니다.
숫자 `connection`의 decimal/exponent truthiness는 parsed float64를 따르므로 underflow도 반영합니다.
최상위 HTTP `self`는 normalized view에서 제외합니다.

각 physical lookup/list/limits 응답은 source의 HTTP 400 미만 정책을 유지합니다.
최종 GET은 한 번이며 canonical `{"limits":{...}}`와 flat object를 모두 허용합니다.
UTF-8이 유효한 empty/malformed JSON은 source의 JSON ValueError tolerance처럼 bare nullable Limits를 만듭니다.
invalid UTF-8·유효한 nonobject/null·잘못된 envelope·descriptor 변환 오류는 성공으로 발표하지 않습니다.
`Limits`와 `Value`는 모든 검사가 끝나면 함께 설정합니다.

`Project.Observed`는 실제 accepted Identity member 응답이고 `Project.Pages`는 accepted 목록 응답입니다.
`Observed`는 최종 Cinder 응답입니다. rejected fallback member를 accepted proof로 보충하지 않습니다.
read·Close·취소·decode·source 오류는 실제 현재 physical response와 이전 완료 증거를 유지합니다.
완료된 Project 뒤에 invalid owned Location·query·getter 오류가 발생하면 local 오류와 Project를 유지하되,
이전 Identity 200을 새 `ResponseError`로 가져오지 않습니다. invalid Location은 project 조회 후, 최종 GET 전에 검증됩니다.
400+ 응답은 native status·body·header cause를 유지합니다.

고정된 source provider·endpoint·ResourceBase·type·microversion 변경은 terminal 오류이며,
ordinary header는 capture한 값을, 인증 token은 live provider 값을 사용합니다.
native retry·reauth는 동일한 GET·URL·query를 유지하고 SDK가 별도로 replay·wait·action·rollback하지 않습니다.
Go context는 서비스 선택·프로젝트 조회·limits 읽기에 적용되며 custom cancellation cause도 보존합니다.
Python mutable Resource/session/cache를 제공하는 일반 SDK 의존성의 지원 상태는 이 직접 helper와 별도로 검토합니다.
