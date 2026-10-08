# 볼륨 타입 접근 권한 조회·추가·삭제

`Connection.GetVolumeTypeAccess`, `AddVolumeTypeAccess`, `RemoveVolumeTypeAccess`는
타입의 이름 또는 ID를 한 번 조회한 뒤 같은 Cinder v3 클라이언트로 접근 API를 호출합니다.
기존 `VolumeTypeReadOption`을 재사용하므로 기본 location과 호출별 옵션을 위해
interface builder를 구현할 필요는 없습니다.

| openstacksdk cloud | Go |
|---|---|
| `conn.get_volume_type_access("fast")` | `conn.GetVolumeTypeAccess(ctx, blockstorage.GetVolumeTypeAccessRequest{NameOrID: "fast"})` |
| `conn.add_volume_type_access("fast", "project-id")` | `conn.AddVolumeTypeAccess(ctx, blockstorage.VolumeTypeAccessRequest{NameOrID: "fast", ProjectID: "project-id"})` |
| `conn.remove_volume_type_access("fast", "project-id")` | `conn.RemoveVolumeTypeAccess(ctx, blockstorage.VolumeTypeAccessRequest{NameOrID: "fast", ProjectID: "project-id"})` |
| 호출별 owned location | `blockstorage.WithVolumeTypeReadLocation(location)` 또는 `WithVolumeTypeReadOptions` |

다음 독립 Go 예제는 Connection과 직접 package 호출을 함께 보여줍니다.
Connection은 `sdk.Connect` 또는 `sdk.FromProvider`로 준비합니다.

```go
package example

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumetypes"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func InspectAccess(ctx context.Context, conn *sdk.Connection) error {
    result, err := conn.GetVolumeTypeAccess(ctx,
        blockstorage.GetVolumeTypeAccessRequest{NameOrID: "fast"})
    if result != nil {
        if result.Resolved != nil {
            fmt.Println("lookup pages:", len(result.Resolved.Pages))
            if result.Resolved.Observed != nil {
                fmt.Println("lookup status:", result.Resolved.Observed.StatusCode)
            }
        }
        if result.Observed != nil {
            fmt.Println("access status:", result.Observed.StatusCode)
        }
    }
    if err != nil { return err }
    fmt.Println("actual type ID:", result.TypeID)
    fmt.Println("raw access JSON:", string(result.Value))
    // Accesses exists only when the entire value is an array of objects.
    for _, row := range result.Accesses {
        owned := row.Clone()
        fmt.Println("original project_id:", string(owned.Body["project_id"]))
        // This optional native projection enforces two string fields.
        // A projection error does not change the successful raw access result.
        var projection volumetypes.VolumeTypeAccess
        if err := owned.Decode(&projection); err != nil { return err }
        fmt.Println("project/type:", projection.ProjectID, projection.VolumeTypeID)
    }
    return nil
}

func ChangeAccess(ctx context.Context, conn *sdk.Connection, projectID string) error {
    input := blockstorage.VolumeTypeAccessRequest{NameOrID: "fast", ProjectID: projectID}
    added, err := conn.AddVolumeTypeAccess(ctx, input)
    if added != nil {
        fmt.Println("resolved target:", added.TypeID)
        if added.Applied != nil {
            fmt.Println("add acknowledgement:", added.Applied.StatusCode,
                "raw bytes:", len(added.Applied.Body))
        }
    }
    if err != nil { return err }
    removed, err := conn.RemoveVolumeTypeAccess(ctx, input)
    if removed != nil && removed.Applied != nil {
        fmt.Println("remove acknowledgement:", removed.Applied.StatusCode)
    }
    return err
}

func AccessWithLocation(ctx context.Context, conn *sdk.Connection) error {
    cloud, project := "configured-cloud", "configured-project"
    location := resource.CloudLocation{
        Cloud: &cloud, Zone: json.RawMessage(`"owned-zone"`),
        Project: resource.CloudProject{ID: json.RawMessage(`"scope-id"`), Name: &project},
    }
    result, err := conn.GetVolumeTypeAccess(ctx,
        blockstorage.GetVolumeTypeAccessRequest{NameOrID: "fast"},
        blockstorage.WithVolumeTypeReadOptions(blockstorage.VolumeTypeReadOpts{Location: &location}))
    if err != nil { return err }
    // Location belongs to the resolved Type view, not to access rows.
    fmt.Println("resolved Type JSON:", string(result.Resolved.Value))
    return nil
}

func DirectAccess(ctx context.Context, cinder *gophercloud.ServiceClient) error {
    result, err := blockstorage.GetVolumeTypeAccess(ctx, cinder,
        blockstorage.GetVolumeTypeAccessRequest{NameOrID: "fast"})
    if err != nil { return err }
    fmt.Println("raw access JSON:", string(result.Value))
    return nil
}
```

실제 Python cloud 사용은 다음과 같습니다. 추가·삭제의 반환값은 `None`입니다.

```python
access = conn.get_volume_type_access("fast")
conn.add_volume_type_access("fast", "project-id")
conn.remove_volume_type_access("fast", "project-id")
```

세 함수 모두 먼저 기본 `get_volume_type(name_or_id)`를 호출합니다. Go도 동일한
[기본 타입 조회](volume-types.md)를 사용하여 안전한 identity는 member GET을 먼저 시도하고,
필요한 fallback 목록까지 exact match를 완료합니다. 두 조회 단계의 query는
`is_public=none`이며, Cinder Type에 없는 `name` query를 추가하지 않습니다.
안전한 ID 경로로 사용할 수 없는 이름은 목록에서만 찾습니다. 이는 고정 URL의 안전성을 위한
Go 정책입니다. 조회가 완료되어도 타입이 없으면 `ErrNotFound`이며 접근 GET·POST를 보내지 않습니다.
[pinned cloud 함수](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L949)를 기준으로 합니다.

접근 요청은 입력 문자열 대신 **실제 canonical 타입 응답의 `id`**를 사용합니다.
Go는 그 값이 비어 있지 않은 안전한 JSON 문자열이어야 합니다. missing·null·숫자·컨테이너·unsafe ID는
조회 증거를 보존하는 local `ErrInvalidOption`이며 접근 요청을 보내지 않습니다.
Python의 seeded Resource ID나 falsey ID를 문자열 경로로 만드는 동작은 재현하지 않습니다.
타입 조회의 normalized six-field `Resolved.Value`와 raw `Resolved.Type`은 그대로 따로 제공합니다.

접근 GET은 `/types/{actualID}/os-volume-type-access`로 한 번만 보내며 query와 body가 없습니다.
UTF-8 JSON object에서 exact `volume_type_access` 필드가 없으면 `Value`는 `[]`입니다.
필드가 존재하면 null·scalar·object·array를 원본 JSON 값으로 유지합니다. Python의 `cast`는
런타임 타입 변환이 아니므로 Go도 이 값을 list로 강제하지 않습니다. advertised next links를 따라가지 않습니다.
`Value`는 원본 JSON 숫자 표기와 정밀도도 유지합니다. Python `response.json()`의 int·float 파싱 이후 값이나
다시 직렬화한 float 표기와는 차이가 날 수 있습니다.
[pinned Type.get_private_access](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/type.py#L119)가 직접 raw 필드를 반환합니다.

`Accesses`는 배열의 **모든** 원소가 object일 때만 제공하는 Go 확장입니다. 빈 배열에는 빈 slice가 있고,
다른 값이나 mixed array에는 nil입니다. 각 row는 전체 raw fields와 실제 접근 응답의 header·status를 소유합니다.
`project_id`·`volume_type_id`의 타입을 강제하거나 location을 계산하지 않습니다.
`RawResource.Clone`과 `Decode`로 필요한 모델을 호출자가 선택할 수 있습니다.
예제의 native `VolumeTypeAccess` projection은 두 string 필드만 해석하므로 원본 값이 다른 타입이면
projection이 실패할 수 있습니다. 이 별도 선택이 raw workflow의 성공 결과를 변경하지는 않습니다.

추가·삭제는 `/types/{actualID}/action`에 각각
`{"addProjectAccess":{"project":ProjectID}}`, `{"removeProjectAccess":{"project":ProjectID}}`를 보냅니다.
`ProjectID`는 literal string입니다. 빈 값·존재하지 않는 프로젝트·유효한 UTF-8 제어문자도 그대로
JSON에 전달하며 Identity client 선택, 프로젝트 이름 해석 또는 존재 확인을 하지 않습니다.
잘못된 UTF-8 문자열은 JSON 대체문자가 전송되는 것을 막기 위해 타입 조회 **이후** local error로 거부합니다.
[pinned 추가·삭제 action](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/type.py#L136)을 따릅니다.

Python의 접근·action 단계는 `raise_from_response`가 HTTP status `<400`이면 반환합니다.
Go의 이 두 단계는 독립적인 `100..399` status policy를 사용하며, 타입 조회의 기존 GET `200` policy와 구분됩니다.
접근 GET의 `204` 빈 body는 실제 `204` 증거를 보존하는 JSON decode error입니다.
action body는 빈 값·non-JSON·binary도 opaque bytes로 보존합니다. native hook이 success codes를 확장해도
원래 policy 밖의 `>=400`을 workflow success로 바꾸지 못합니다.
1xx 최종 응답 노출은 `net/http` 동작을 따르고, 다른 URL·method로 이동하는 redirect는 고정 요청 guard가 거부합니다.
[pinned status 검사](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/exceptions.py#L214)를 기준으로 합니다.
별도의 native `ListAccesses`는 typed pagination transport에서 `200/204/300`을 받으며
`AccessPage`와 두 string 필드 추출을 사용합니다. native `AddAccess`·`RemoveAccess`는 `202`를 받습니다.
따라서 이 raw single-access workflow를 그 native leaf 호출과 같은 응답 추상화로 보지 않습니다.
[native pager status](https://github.com/gophercloud/gophercloud/blob/v2.15.0/pagination/http.go#L59),
[native access 모델](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/volumetypes/results.go#L159),
[native action 요청](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/volumetypes/requests.go#L287)을 각각 유지합니다.

`Resolved`는 타입 조회 결과이며, `Observed`는 접근 GET의 실제 accepted 응답,
`Applied`는 action POST의 실제 accepted 응답입니다. action이 accepted되어도 권한이 실제로 바뀌었거나
서버 처리가 완료되었다고 주장하는 `Changed`·완료 bool은 제공하지 않습니다.
조회 이후 refresh·membership 확인·wait·자동 rollback 또는 전체 workflow replay를 하지 않습니다.
서버 정책이 필요한 경우 호출자가 별도로 조회할 수 있습니다.

accepted 응답의 read·Close·context·선택 source 변경 오류에서는 해당 실제 body·header·status와
이미 완료된 `Resolved` 증거를 유지합니다. 접근 `Value`·`Accesses`는 전체 단계 성공 후에만 확정합니다.
거부된 HTTP 응답은 native 오류의 실제 status·body·header를 유지하며 accepted proof를 합성하지 않습니다.
local 잘못된 target·owned Location 오류도 기존 조회 증거를 유지하지만 이전 HTTP `200`을 그 local 오류의
`ResponseError`로 빌리지 않습니다. 실제 malformed 응답의 decode 오류는 자신의 accepted `ResponseError`를 가집니다.
`errors.Is`로 cancellation·custom context cause를, `errors.As`로 resource operation과 실제 HTTP 증거를 확인할 수 있습니다.

Connection은 원본 옵션을 한 번 적용한 다음 cached Cinder v3와 현재 recorded location을 snapshot합니다.
`WithVolumeTypeReadLocation`과 bulk option은 입력을 소유하며 기본 Connection location을 바꾸지 않습니다.
타입 조회에만 기존 Type location 규칙이 적용되고 access rows에는 computed location이 없습니다.
직접 package 호출은 선택된 client source를 원본 옵션 실행 전에 capture하며 기본값으로 recorded provider scope를 사용합니다.
두 단계는 같은 선택 source와 고정 요청을 사용하고 native authentication·retry를 유지합니다.
Provider·Endpoint·ResourceBase·Type·Microversion 변경은 terminal error입니다.

이 문서는 위 pinned Python cloud 세 선언을 Go의 owned 옵션·fresh canonical response·phase proof로 매핑합니다.
Python의 v2/v3 Resource 객체 상태·session/cache 전체와 native/Proxy/Resource의 별도 선언 지원 상태는 독립적으로 검토합니다.
전체 openstacksdk의 구현 완료를 의미하지 않습니다.
