# 볼륨 ID 조회

`Connection.GetVolumeID`는 이름이나 ID로 볼륨을 조회한 뒤 응답의 ID JSON을
반환합니다. 입력한 ID를 그대로 반환하는 shortcut이 아니므로 실제 조회 오류와
부재를 확인할 수 있습니다. Connection은 원본 read 옵션을 한 번 적용한 뒤 cached
Cinder v3 클라이언트를 선택하며 Nova나 별도 project 조회는 필요하지 않습니다.

| openstacksdk | Go |
|---|---|
| `conn.get_volume_id("data")` | `conn.GetVolumeID(ctx, blockstorage.GetVolumeIDRequest{NameOrID: "data"})` |
| 직접 Cinder 클라이언트 사용 | `blockstorage.GetVolumeID(ctx, cinder, input)` |
| 호출별 location 교체 | `blockstorage.WithVolumeReadLocation(location)` |

다음은 Connection과 package 호출을 보여주는 독립 Go 예제입니다. Connection은
`sdk.Connect` 또는 `sdk.FromProvider`로 준비합니다.

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

func ReadVolumeID(ctx context.Context, conn *sdk.Connection, nameOrID string) error {
    result, err := conn.GetVolumeID(ctx,
        blockstorage.GetVolumeIDRequest{NameOrID: nameOrID})
    if err != nil {
        if result != nil {
            fmt.Println("admitted fallback pages:", len(result.Pages))
            if result.Observed != nil {
                fmt.Println("admitted member status:", result.Observed.StatusCode)
            }
        }
        return err
    }
    if result.ID == nil {
        fmt.Println("volume absent")
        return nil
    }
    fmt.Println("ID JSON:", string(result.ID))
    fmt.Println("normalized volume:", string(result.Value))
    copy := result.Volume.Clone()
    fmt.Println("original ID field:", string(copy.Body["id"]))
    return nil
}

func ReadVolumeIDWithLocation(ctx context.Context, conn *sdk.Connection, nameOrID string) error {
    project := "configured-project"
    location := resource.CloudLocation{
        Project: resource.CloudProject{
            ID: json.RawMessage(`"scope-id"`), Name: &project,
        },
    }
    result, err := conn.GetVolumeID(ctx,
        blockstorage.GetVolumeIDRequest{NameOrID: nameOrID},
        blockstorage.WithVolumeReadLocation(location))
    if err != nil { return err }
    fmt.Println(string(result.ID))
    return nil
}

func ReadVolumeIDDirect(ctx context.Context, cinder *gophercloud.ServiceClient, nameOrID string) error {
    result, err := blockstorage.GetVolumeID(ctx, cinder,
        blockstorage.GetVolumeIDRequest{NameOrID: nameOrID})
    if err != nil { return err }
    if result.ID != nil { fmt.Println(string(result.ID)) }
    return nil
}
```

실제 Python cloud helper는 다음과 같이 호출합니다.

```python
import openstack

conn = openstack.connect(cloud="example")
volume_id = conn.get_volume_id("data")
if volume_id is None:
    print("no ID returned")
else:
    print(volume_id)
```

고정 Python 함수는 `get_volume(name_or_id)`를 호출하고, 반환 resource가 truthy이면
그 객체의 `id` 속성을 그대로 반환합니다. Resource는 기본 nullable 필드를 포함한
`to_dict()`로 내부 dict를 채우므로 canonical 빈 Volume `{}`도 일반적으로 truthy입니다.
이 조건은 raw body나 ID의 truthiness를 검사하지 않습니다. 상속한 `Resource.Body("id")`에는 string
coercion이 없으므로 실제 ID 값은 문자열뿐 아니라 `null`, boolean, 숫자나 container도
될 수 있습니다. Go는 이를 `json.RawMessage`로 보존하며 강제로 string으로 바꾸지
않습니다. `false`, `0`, `""`, 배열·객체도 완성된 ID 결과입니다.

Go의 정상 부재는 `err == nil`과 `result.ID == nil`로 구별합니다. 발견한 볼륨의 ID가
없거나 `null`이면 `ID`는 literal JSON `null`이며 `Volume`과 `Value`는 유지됩니다.
따라서 Go는 Python의 반환 `None`에서 구별하기 어려운 부재와 발견한 null ID를
별도의 결과 상태로 확인할 수 있습니다. 오류에서는 `ID`·`Value`·`Volume`이 nil이며
이미 허용한 응답 증거는 남습니다. 호출자가 오류를 정상 부재로 처리하면 조회 실패를
놓칠 수 있으므로 먼저 `err`를 확인합니다.

조회는 필터 없는 정확한 이름·ID 정책을 사용합니다. 안전한 단일 URL segment는
`/volumes/{identity}` GET을 먼저 수행하고, clean 400·403·404에서 `name` query를 가진
`/volumes/detail` 목록으로 fallback합니다. 경로 구분자·percent·query·fragment·Unicode
whitespace를 가진 이름은 Go의 안전한 route 정책에 따라 목록으로 바로 조회합니다.
Python `Resource.find`는 이런 문자열에도 GET을 먼저 시도하므로 명시적인 Go 매핑
차이입니다. glob이나 JMESPath를 적용하지 않으며 이름의 `*`도 정확한 비교에 사용합니다.
두 번째 정확한 일치에서는 `resource.ErrAmbiguous`를 반환합니다. 부재는 전체 fallback
목록이 성공한 뒤에만 확정하며 인증·접근·transport·schema 오류를 억제하지 않습니다.

`ID`, 38개 필드의 정규화된 `Value`, 원래 wire 필드인 `Volume.Body`는 각각 소유하는
독립 복사본입니다. HTTP 응답에서 id를 생략한 경우 Go는 응답 ID를 `null`로 유지합니다.
Python의 기존 Resource는 요청 ID를 seeded state로 보존할 수 있으므로 이 차이를
기록합니다. 응답의 서로 다른 ID를 요청 ID로 덮어쓰거나, 입력 ID라는 이유로 실제
member GET을 생략하지 않습니다. typed 모델이 필요하면 raw resource에 `Decode`를
명시적으로 적용합니다.

`Observed`는 실제 허용한 member 200 응답이고 `Pages`는 실제 허용한 목록 200 응답입니다.
접수 후 read·Close·정규화·source 변경·취소에서 실패해도 완료된 응답은 유지합니다.
거절된 HTTP의 status·header·body는 원래 native 오류에 있으며 성공 증거를 합성하지
않습니다. context의 cancellation cause도 오류 체인에 남습니다. native 인증·retry는
유지하되 SDK가 조회 전체를 다시 실행하지 않습니다.

기본 location은 Connection이 옵션 적용 후 `CurrentLocation`으로 읽은 기록된 scope의
snapshot입니다. 실행 중 scope가 바뀌어도 해당 실행은 고정한 snapshot을 사용하며 다음
실행에서 새 scope를 읽습니다. `WithVolumeReadLocation`과 `WithVolumeReadOptions`는
호출별 owned 복사본을 적용하고 Connection 기본값을 바꾸지 않습니다. 직접 package
호출은 기록된 provider project ID를 사용하고 모르는 cloud·region·설정 이름은 `null`
입니다. caller의 잘못된 location JSON은 실제 row를 소비할 때 local
`resource.ErrInvalidOption`으로 반환하며 허용한 proof는 유지하지만 HTTP 200
ResponseError를 빌리지 않습니다. 성공한 빈 fallback에서는 사용하지 않은 location을
검사하지 않습니다. 물리 row·descriptor 오류는 해당 accepted 응답의 오류를 유지합니다.

이 helper는 Cinder v3와 strict canonical UTF-8 응답·안전한 native continuation을
사용합니다. [조회 문서](volume-reads.md)와 [검색 문서](search-volumes.md)에 raw/normalized
표현, scope·페이지·숫자·alias 경계를 설명합니다. 비교 대상은
openstacksdk [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 직접 `get_volume_id`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L362-L371),
[필터 없는 `get_volume`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L82-L122),
[Resource 조회와 untyped ID](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)입니다.
타입 helper나 generic Resource/session/cache의 지원 판정은 독립적인 작업이며 이 단위가
전체 SDK 완료를 뜻하지 않습니다. Go 예제는 compile 검증 대상으로 관리합니다.
