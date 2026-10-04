# 볼륨 검색과 단일 선택

`Connection.SearchVolumes`와 `Connection.GetVolume`은 Cinder v3 클라이언트와
공통 옵션·오류·페이지 증거를 연결합니다. 애플리케이션은 builder interface를
구현하지 않고 `WithVolumeSearchFilters` 또는 `WithVolumeSearchExpression`을 사용합니다.

| openstacksdk cloud helper | Go |
|---|---|
| `conn.search_volumes("data*")` | `conn.SearchVolumes(ctx, blockstorage.SearchVolumesRequest{NameOrID: "data*"})` |
| `conn.search_volumes(filters={"metadata": {"owner": "worker"}})` | `WithVolumeSearchFilters(json.RawMessage(...))` |
| ``conn.search_volumes(filters="[?size >= `10`].id")`` | ``WithVolumeSearchExpression("[?size >= `10`].id")`` |
| `conn.get_volume("data")` | `conn.GetVolume(ctx, blockstorage.GetVolumeRequest{NameOrID: "data"})` |
| `conn.get_volume("data", filters={})` | 같은 호출에 `WithVolumeSearchFilters`로 JSON `{}` 전달 |

다음 예제는 연결을 전달받는 독립 Go 함수입니다. `Value`를 출력하면 Python의
정규화된 cloud 검색 결과를 확인할 수 있습니다. `Volumes`는 표현식으로 변환되지
않은 검색에서만 원본 응답 필드와 HTTP metadata를 제공합니다.

```go
package example

import (
    "context"
    "encoding/json"
    "fmt"

    sdk "gophercloudsdk"
    "gophercloudsdk/blockstorage"
)

func Search(ctx context.Context, conn *sdk.Connection) error {
    result, err := conn.SearchVolumes(ctx,
        blockstorage.SearchVolumesRequest{NameOrID: "data*"},
        blockstorage.WithVolumeSearchFilters(json.RawMessage(
            `{"metadata":{"owner":"worker"},"is_bootable":false}`)))
    if err != nil { return err }
    fmt.Println(string(result.Value), len(result.Volumes), len(result.Pages))

    ids, err := conn.SearchVolumes(ctx, blockstorage.SearchVolumesRequest{},
        blockstorage.WithVolumeSearchExpression("[?size >= `10`].id"))
    if err != nil { return err }
    fmt.Println(string(ids.Value))

    volume, err := conn.GetVolume(ctx,
        blockstorage.GetVolumeRequest{NameOrID: "data"})
    if err != nil { return err }
    if volume.Value == nil { return nil }
    fmt.Println(string(volume.Value))
    return nil
}
```

Python:

```python
volumes = conn.search_volumes(
    "data*", filters={"metadata": {"owner": "worker"}, "is_bootable": False}
)
ids = conn.search_volumes(filters="[?size >= `10`].id")
volume = conn.get_volume("data")
```

검색은 `/volumes/detail`의 전체 목록을 먼저 읽습니다. 각 row의 알려진 필드를
페이지를 넘기기 전에 변환하고, 모든 페이지를 읽은 다음 이름·ID의 정확한 일치나
대소문자를 구별하는 glob을 적용합니다. `*`, `?`, bracket class는 경로의 `/`,
점, 줄바꿈, Unicode에도 적용됩니다. 같은 pattern의 정확한 일치가 있더라도
다른 row의 glob 일치는 유지하며 응답 순서와 중복을 보존합니다.

필터 객체는 JSON에 쓴 key 순서대로 처리합니다. 중첩 객체는 부분 일치이며,
이전 key의 불일치로 종료했다면 나중의 존재하지 않는 key를 검사하지 않습니다.
알려진 생략 필드는 정규화 결과에 `null`로 존재합니다. 알려지지 않은 필터 key를
실제로 읽으면 오류입니다. 일반 JSON equality는 Python처럼 boolean과 숫자를
재귀적으로 비교하므로 `true`와 `1`도 일치할 수 있습니다. 기존 `resource.WithBody`
필터는 각 binding의 별도 정책을 유지합니다.
중복 필터 key는 처음 등장한 위치를 유지하면서 마지막 값을 사용합니다.

JMESPath는 projection·filter·pipe·함수·expression reference 등 전체 문법을
제공합니다. 반환값은 목록, 객체, 문자열, 숫자, boolean, `null` 모두 가능합니다.
`Value`가 기본 결과이며 표현식 결과에 원본 row나 응답 header를 합성하지 않습니다.
문법 오류와 실제로 평가한 함수·runtime 오류는 전체 목록을 읽고 식별자 검색을
마친 후 반환합니다. 빈 projection이나 short circuit으로 방문하지 않은 runtime
오류는 발생하지 않습니다. JMESPath의 숫자 `0`은 truthy이며 일반 Python 결과의
숫자 `0`은 falsey입니다.

`GetVolume`의 필터 생략과 JSON `null`은 안전한 ID 경로의 GET을 먼저 사용하고,
clean 400·403·404에서 이름 query를 가진 상세 목록으로 fallback합니다. `{}`, `[]`,
`false`, `0`, `""`를 명시하면 전체 검색 분기입니다. 이 분기는 먼저 검색 결과의
truthiness를 검사하고, 길이가 1보다 크면 `resource.ErrAmbiguous`, 그 후 index 0을
선택합니다. `[false]`, `[0]`, `[""]`의 선택값은 그대로 유지하며 `[null]`은 부재입니다.
문자열 결과는 Unicode 문자 수를 사용합니다. 숫자·boolean의 길이나 객체의 integer
index는 Python과 마찬가지로 오류입니다. 잘못된 JSON 필터도 원본 HTTP 오류로 꾸미지
않고 `resource.ErrInvalidOption`과 실제 원인을 반환합니다.
기본 GET도 canonical 응답의 원본 row만 사용합니다. 응답에서 id를 생략하면
정규화된 id는 null이며 요청 ID를 응답 필드로 합성하지 않습니다. Python Resource가
요청 ID를 기존 객체에 보관하는 상태 동작과 strict envelope/JSON decoder는 별도 경계입니다.

38개 정규화 필드는 Cinder v3 Volume의 37개 알려진 Body 속성과 계산된 `location`입니다.
wire alias는 JSON member 순서대로 읽으며 마지막 소비 값이 적용됩니다. `bootable`과
`encrypted`는 boolean 또는 정확한 대소문자 무관 `true`/`false` 문자열을 받습니다.
attachments의 nonnull 단일 값은 목록으로, metadata의 nonobject는 `{}`로 변환합니다.
size는 Python descriptor의 bool·정수·float 절삭·digit 문자열 변환을 따릅니다. 원본의
알려지지 않은 필드와 실제 wire 값은 `RawResource.Body`에 보존됩니다. typed 모델이
필요할 때 `RawResource.Decode`로 명시적으로 변환합니다.

`location`은 HTTP body의 location을 사용하지 않습니다. Connection은 선택된 cloud·region,
이미 기록된 token의 project ID, 설정의 project 이름·domain을 사용하고 row의 project와
availability zone을 반영합니다. 다른 project의 row에는 현재 project의 이름을 붙이지
않습니다. `CurrentLocation`은 추가 인증이나 API 호출 없이 snapshot을 반환합니다.
직접 ServiceClient를 전달한 호출은 기록된 project ID를 사용하고 cloud·region·설정 이름은
`null`입니다. `FromProvider`의 수동 token·custom auth result에도 모르는 정보를 추측하지
않습니다. 필요한 설정은 owned `WithCloudLocation` 또는 `WithVolumeSearchLocation`으로
제공합니다. native clouds parser가 project ID scoping에서 제거한 project-domain 설정은
반환된 AuthOptions로 복구할 수 없으므로 이 명시적 옵션으로 보완합니다.

각 result의 JSON·header·페이지 증거는 독립 복사본입니다. 결과 Value와 Volumes는 전체
성공 후에만 설정하고, 오류 시 Pages와 실제 member 응답 Observed는 남습니다. source의
provider·endpoint·resource base·type·microversion 변경과 취소는 다음 요청을 막습니다.
페이지는 같은 origin과 escaped collection path의 continuation만 받고 cycle을 검사합니다.
HTTP Link, 임의 marker, 다른 envelope의 continuation을 추가로 추측하지 않습니다.
빈 `volumes: []`는 사용하지 않을 next link를 검사하기 전에 목록을 끝냅니다.
접수된 페이지·member 증거는 실제로 허용한 HTTP 200에만 생성하며, 뒤의 거절된
응답이나 로컬 필터 오류에 앞선 성공 응답을 붙이지 않습니다.

Go 경계도 명시합니다. 필터의 원시 key 순서는 보존하지만 JMESPath의 JSON 객체 key·value
순서는 lexical로 고정합니다. Python의 사용자 정의 객체 equality나 mutable Resource cache는
이 JSON API에 포함되지 않습니다. 숫자 equality·ordering은 exact decimal이고, 비정수 산술과
avg는 유한 IEEE float 결과를 사용합니다. 정수 산술의 coefficient 확장이 1,048,576 자리를
넘으면 오류를 반환합니다. 표현식 문자열·reverse·length는 Unicode 문자 단위입니다.
Volume size의 Unicode digit 변환은 감사한 Unicode 16 범위를 사용하며 Python의 설정 가능한 정수 문자열
자릿수 제한을 모방하지 않습니다. duplicate JSON key와 alias의 textual last-wins는 Go API의
명시적 정책입니다. 고정 openstacksdk는 `jmespath>=0.9.0`만 선언하며, 테스트의 Python
1.0.1은 별도 참조 버전입니다.
식별자 stringify는 정수의 정확한 값과 float64의 표현을 구분합니다. float 표현의
overflow는 inf, underflow는 ±0.0이 될 수 있고 Unicode printability·최단 문자열 표현은
Go 경계입니다. 일반 JSON truthiness의 exact decimal `1e-9999`는 계속 true입니다.
엔진의 숫자 문자열·식별자 문자열의 Unicode 분류는 Go toolchain의 표를 사용합니다.
반복 slice bounds와 생략·trailing function comma 같은 일부 Python parser의 허용 구문은
표준 JMESPath 문법에 따라 오류입니다. Connection의 기본 location은 옵션 적용 후 기록된
인증 scope의 독립 snapshot입니다. 각 `SearchVolumes`/`GetVolume` 실행은 해당 snapshot을
유지하고, 다음 `CurrentLocation` 또는 새 workflow 호출은 그때 기록된 scope를 읽습니다.

이 문서의 Go 예제는 compile 검증 대상으로 관리합니다. 실제 클라우드 실행은 인증 설정이
필요하며 로컬 HTTP fixture 테스트와 구별합니다.
