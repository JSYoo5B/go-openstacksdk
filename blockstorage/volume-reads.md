# 볼륨 목록·ID 조회·존재 확인

`Connection.ListVolumes`, `Connection.GetVolumeByID`, `Connection.VolumeExists`는
Cinder v3 읽기 작업을 같은 옵션·location·오류 방식으로 제공합니다. Connection은
원본 옵션을 한 번 적용한 뒤 cached Cinder 클라이언트를 선택합니다. Nova 클라이언트는
필요하지 않습니다. 직접 `ServiceClient`를 전달하는 같은 이름의 package 함수도 있습니다.

이름·ID에서 응답 ID를 얻는 `GetVolumeID`는 [별도 사용법과 Python 비교](volume-id.md)에 설명합니다.

| openstacksdk cloud helper | Go |
|---|---|
| `conn.list_volumes()` | `conn.ListVolumes(ctx)` |
| `conn.get_volume_by_id("volume-id")` | `conn.GetVolumeByID(ctx, blockstorage.GetVolumeByIDRequest{ID: "volume-id"})` |
| `conn.volume_exists("data")` | `conn.VolumeExists(ctx, blockstorage.VolumeExistsRequest{NameOrID: "data"})` |
| 호출별 location 지정 | `blockstorage.WithVolumeReadLocation(location)` |
| 여러 read 옵션 교체 | `blockstorage.WithVolumeReadOptions(blockstorage.VolumeReadOpts{Location: &location})` |

다음은 Connection과 직접 package 호출을 함께 보여주는 독립 Go 예제입니다.
Connection을 만들 때는 `sdk.Connect` 또는 `sdk.FromProvider`를 사용합니다.

```go
package example

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/gophercloud/gophercloud/v2"
    sdk "github.com/JSYoo5B/gophercloudsdk"
    "github.com/JSYoo5B/gophercloudsdk/blockstorage"
    "github.com/JSYoo5B/gophercloudsdk/resource"
)

func ReadVolumes(ctx context.Context, conn *sdk.Connection, id, name string) error {
    listed, err := conn.ListVolumes(ctx)
    if err != nil {
        if listed != nil { fmt.Println("admitted list pages:", len(listed.Pages)) }
        return err
    }
    fmt.Println(string(listed.Value), len(listed.Volumes))

    found, err := conn.GetVolumeByID(ctx,
        blockstorage.GetVolumeByIDRequest{ID: id})
    if err != nil {
        if found != nil && found.Observed != nil {
            fmt.Println("admitted member response:", found.Observed.StatusCode)
        }
        return err
    }
    fmt.Println(string(found.Value), string(found.Volume.Body["id"]))
    copy := found.Volume.Clone()
    var projection struct { ID *string `json:"id"` }
    if err := copy.Decode(&projection); err != nil { return err }

    exists, err := conn.VolumeExists(ctx,
        blockstorage.VolumeExistsRequest{NameOrID: name})
    if err != nil {
        if exists != nil { fmt.Println("admitted fallback pages:", len(exists.Pages)) }
        return err
    }
    if exists.Exists == nil { return fmt.Errorf("missing completed existence result") }
    fmt.Println("exists:", *exists.Exists)
    return nil
}

func ListWithLocation(ctx context.Context, conn *sdk.Connection) error {
    cloud, project := "configured-cloud", "configured-project"
    location := resource.CloudLocation{
        Cloud: &cloud,
        Project: resource.CloudProject{
            ID: json.RawMessage(`"project-id"`), Name: &project,
        },
    }
    result, err := conn.ListVolumes(ctx, blockstorage.WithVolumeReadLocation(location))
    if err != nil { return err }
    fmt.Println(string(result.Value))
    return nil
}

func ReadDirect(ctx context.Context, cinder *gophercloud.ServiceClient, id string) error {
    listed, err := blockstorage.ListVolumes(ctx, cinder)
    if err != nil { return err }
    fmt.Println(len(listed.Volumes), len(listed.Pages))
    volume, err := blockstorage.GetVolumeByID(ctx, cinder,
        blockstorage.GetVolumeByIDRequest{ID: id})
    if err != nil { return err }
    exists, err := blockstorage.VolumeExists(ctx, cinder,
        blockstorage.VolumeExistsRequest{NameOrID: id})
    if err != nil { return err }
    fmt.Println(string(volume.Value), *exists.Exists)
    return nil
}
```

실제 openstacksdk cloud 호출은 다음과 같습니다.

```python
import openstack

conn = openstack.connect(cloud="example")
volumes = conn.list_volumes()
volume = conn.get_volume_by_id("volume-id")
exists = conn.volume_exists("data")

# cache=False도 경고를 발생시키며 조회 동작을 바꾸지 않습니다.
volumes = conn.list_volumes(cache=False)
```

`ListVolumes`는 `/volumes/detail`을 순회해 전체 목록을 반환합니다. 페이지의 각 row를
변환한 뒤 다음 페이지를 요청하고, 모든 페이지가 성공해야 `Value`와 `Volumes`를
설정합니다. row 순서와 중복을 유지합니다. 잘못된 첫 페이지의 알려진 속성을 만났다면
뒤 페이지는 요청하지 않습니다. canonical `volumes: []`는 광고된 next link와 관계없이
목록을 끝냅니다. Python `cache`는 생략하거나 `None`인 경우 외에는 경고만 발생시키는
폐기 인자이므로 Go 옵션에는 추가하지 않았습니다.

`GetVolumeByID`는 literal ID의 `/volumes/{id}` GET을 사용합니다. 빈 값이나 경로를
바꿀 수 있는 문자는 첫 요청 전에 거절합니다. **404는 오류이며 이름 목록으로
fallback하지 않습니다.** 고정 Python helper도 실제 `Proxy.get_volume`의 NotFound를
그대로 전달합니다. 해당 helper의 docstring에 있는 “else None”과 실제 제어 흐름을
구별해야 합니다. 400·403도 오류이며 native 응답의 status·header·body를 유지합니다.

`VolumeExists`는 필터 없는 단일 이름·ID 조회를 사용합니다. 안전한 ID 경로의 GET을
먼저 사용하고 clean 400·403·404에서 정확한 이름 query를 가진 상세 목록으로
fallback합니다. 경로 구분자·percent·query·fragment·Unicode whitespace를 가진 이름은
Go의 안전한 route 정책에 따라 목록으로 바로 조회합니다. Python Resource.find는
이 문자열에도 GET을 먼저 시도하므로 명시적인 Go 매핑 차이입니다. glob과 JMESPath
필터를 추가하지 않으며 `*` 등의 문자는 정확한 이름 비교의 일부입니다.

존재 여부는 반환 resource의 존재로 판단합니다. `{"volume":{}}`도 성공한 resource이므로
`Exists`는 `true`입니다. 정상적으로 조회를 마친 부재는 `false`이고, ambiguity·접근
거절·잘못된 JSON·취소 등 오류는 `Exists == nil`입니다. 오류를 `false`로 바꾸면 실제
부재와 조회 실패를 구별할 수 없습니다. 이름 조회의 두 번째 정확한 일치에서 ambiguity를
반환하고 사용하지 않을 뒤 row·다음 link는 소비하지 않습니다.

`Value`는 Cinder v3 Volume의 37개 알려진 Body 속성과 계산된 `location`, 총 38개 필드의
정규화된 JSON입니다. 생략한 알려진 필드는 `null`이고, source descriptor의 boolean,
BoolStr·list·dict·size 변환을 적용합니다. `RawResource.Body`에는 원래 wire 필드와
알려지지 않은 필드도 남습니다. raw 값이 `VolumeInfo`와 같은 stricter typed 모델에
맞는지는 `Decode`를 호출할 때 별도로 검사합니다. `Clone`은 body·header를 독립 복사하며,
정규화 값·raw row·응답 증거도 서로 독립적으로 소유합니다. canonical member 응답에서
id가 빠졌으면 정규화된 id는 `null`이며 요청 ID를 응답 필드에 합성하지 않습니다.

Connection의 location은 옵션 적용 후 `CurrentLocation`으로 기록된 인증 scope를
snapshot합니다. 실행 중 token의 scope가 바뀌어도 이미 선택한 location은 유지하고,
다음 실행은 그때 기록된 scope를 읽습니다. cloud·region·설정 project 이름·domain은
알려진 설정만 사용하고 token의 이름을 추측하지 않습니다. 다른 project의 row에는
현재 project의 이름을 붙이지 않습니다. 직접 package 호출은 provider에 기록된 project
ID를 사용하며 cloud·region·설정 이름이 없다면 `null`입니다. `WithVolumeReadLocation`은
호출별 독립 복사본으로 전체 location을 교체하며 `WithCloudLocation`은 Connection의
기본값을 교체합니다. native clouds parser가 project-ID scope에서 제거한 project-domain
설정은 명시적 owned location으로 보완할 수 있습니다.

caller가 제공한 location의 잘못된 JSON은 실제 row나 member를 정규화할 때 local
`resource.ErrInvalidOption`으로 반환합니다. 앞서 허용한 `Pages`·`Observed`는 유지하지만
그 오류를 HTTP 200의 ResponseError로 꾸미지 않습니다. 빈 목록에서 사용하지 않은
location은 검사하지 않습니다. canonical row·descriptor의 물리 응답 오류는 그 응답의
accepted ResponseError를 유지합니다.

`Pages`는 실제로 허용한 HTTP 200 목록 응답이며 `Observed`는 실제로 허용한 200 member
응답입니다. 허용된 응답을 받은 뒤 read·Close·descriptor·source 확인·취소에서 실패해도
이미 받은 증거는 남습니다. 오류 중에는 완성된 `Value`·`Volumes`·`Volume`을 설정하지
않습니다. 거절된 HTTP 응답은 원래 native 오류에 있으며 accepted proof를 합성하지
않습니다. context와 사용자 cancellation cause도 오류 체인에 유지합니다.

세 함수는 Cinder v3 선택과 canonical UTF-8 JSON·`volume(s)` envelope·200 응답을 사용합니다.
페이지 continuation은 native `volumes_links`와 같은 origin·escaped collection path의
안전한 경로만 사용하고 cycle을 검사합니다. 선택된 provider·endpoint·resource base·type·
microversion이 바뀌면 추가 요청을 중단하며, native authentication·retry 동작은 유지합니다.
Python의 v2 협상, mutable Resource/session/cache identity와 generic Resource의 다른
pagination 동작은 독립적인 SDK 검토 대상입니다. 이번 direct helper 세 개의 매핑이
전체 SDK의 완료를 뜻하지 않습니다. 검색의 숫자·alias·location 경계는
[볼륨 검색 문서](search-volumes.md)의 같은 정규화 정책을 따릅니다.

비교 source는 openstacksdk
[`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 cloud helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py),
[v3 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py),
[Resource 조회·목록](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py),
[v3 Volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py)입니다.
Go 예제는 compile 검증 대상으로 관리하며 실제 cloud 실행에는 인증 설정이 필요합니다.
