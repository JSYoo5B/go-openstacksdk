# Cinder volume 이름 또는 ID 검색

기준은 openstacksdk [`ef55d7d`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py)의 `find_volume`과 [Volume Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py)입니다. `FindRecord`는 SDK가 member GET과 목록 검색을 연결하고, Source descriptor를 적용한 응답과 원래 HTTP 증거를 함께 반환합니다.

| Python | Go |
|---|---|
| `conn.block_storage.find_volume("data")` | `service.Volumes.FindRecord(ctx, "data")` |
| `find_volume("data", ignore_missing=False)` | `FindRecord(ctx, "data", volumes.WithRecordFindIgnoreMissing(false))` |
| `find_volume("data", details=False)` | `FindRecord(ctx, "data", volumes.WithRecordFindDetails(false))` |
| `find_volume("data", all_projects=True)` | `FindRecord(ctx, "data", volumes.WithRecordFindAllProjects(true))` |

```python
import openstack

conn = openstack.connect(cloud="source")
volume = conn.block_storage.find_volume(
    "data", ignore_missing=False, details=False, all_projects=True
)
print(volume.to_dict())
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
)

func main() {
    identity := flag.String("volume", "data", "Volume name or ID")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud("source"))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    volume, err := service.Volumes.FindRecord(ctx, *identity,
        volumes.WithRecordFindIgnoreMissing(false),
        volumes.WithRecordFindDetails(false),
        volumes.WithRecordFindAllProjects(true),
    )
    if err != nil { log.Fatal(err) }
    body, err := json.MarshalIndent(volume.Resource.Body, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(body))
}
```

기본값은 `ignore_missing=true`, `details=true`, `all_projects=false`입니다. 첫 요청은 `GET /volumes/{name_or_id}`이고, 깨끗한 HTTP 400/403/404에만 `GET /volumes/detail?name={name_or_id}`로 fallback합니다. `details=false`는 fallback 경로를 `/volumes`로 바꾸며 direct GET은 동일합니다. `all_projects=true`는 fallback에만 `all_tenants=True`를 붙입니다. 기본 또는 명시한 false는 이 자동 필드를 보내지 않습니다. 프로젝트 간 조회 권한은 서버 정책을 따릅니다.

전체 fallback 페이지에서 정확한 문자열 ID 또는 이름을 찾습니다. 이름과 ID 중 어느 쪽이 일치해도 후보이고, 늦은 페이지의 중복까지 `resource.ErrAmbiguous`로 처리합니다. 미존재는 기본 `nil, nil`, `WithRecordFindIgnoreMissing(false)`는 `resource.ErrNotFound`입니다. late HTTP/decode/descriptor/transport 오류는 앞선 일치가 있어도 실패로 반환합니다. 숫자 ID는 원래 JSON을 유지하고 문자열 identity와 같은 값으로 간주하지 않습니다. 입력 이름은 trim하지 않으며 member 경로를 URL escape합니다. 기존 typed `Find`, `FindIdentity`, `Get`, `List`, `ListSummary`는 계속 사용할 수 있습니다.

Source Volume의 버전 상한은 3.71입니다. 이미 client에 microversion이 있으면 session default처럼 그 값을 그대로 사용하고 discovery를 호출하지 않습니다. 기본 버전이 없으면 공통 Cinder discovery로 서버 범위를 확인하여 최대 3.71을 선택합니다. 공유 client를 바꾸지 않고 GET과 모든 목록 페이지에 같은 선택 버전을 사용합니다. `Discovery`는 실제 받아들인 discovery 응답의 별도 기록이며 volume 응답을 대체하지 않습니다. Go는 필요한 호출마다 제한된 discovery를 수행하고 Python adapter discovery/GET response cache와 mutable Resource 상태는 재현하지 않습니다.

반환값 `VolumeResourceRecord`는 다음 독립 채널을 소유합니다.

- `Resource`는 기존 `internal/cindervolume.View`를 재사용해 직접 Body 34개, 상속 `id`/`name`/`metadata`, computed `location`의 총 38개를 제공합니다. 없는 descriptor는 null입니다. untyped ID/날짜/unknown 형식을 native Go `string`이나 `time.Time`으로 강제하지 않습니다.
- LIST 행에 `connection`, `microversion`, `_synchronized`, `cls`가 존재하면 null/false여도 Source Resource 생성자 인자와 충돌하므로 해당 실제 페이지의 `ResponseError`로 실패합니다. `self`는 Source가 생성 전에 제거하는 필드이므로 LIST에서는 허용하고 declared Resource에서 제외합니다. member GET은 기존 객체에 응답을 merge하므로 이 충돌 검사를 적용하지 않습니다. 원래 unknown 필드는 Wire에 남습니다.
- `Wire`는 변환 전 행과 unknown 필드를 보존합니다. `Envelope`는 member envelope 또는 해당 행이 나온 목록의 원래 전체 JSON이고, `Header`와 `StatusCode`도 실제 응답입니다. query seed나 location override를 원래 HTTP 필드로 꾸미지 않습니다.
- `bootable`/`encrypted`는 BoolStr descriptor이므로 bool 또는 대소문자를 무시한 true/false 문자열만 허용합니다. `multiattach`/`shared_targets`는 JSON truthiness를 적용하여 문자열 `"False"`는 true입니다. `size`는 Source integer descriptor를 적용하며 bool은 bool, digit 문자열은 정수, non-digit 문자열은 0입니다.
- `attachments`의 nonnull 비목록 값은 단일 요소 목록으로 감싸고, `metadata`/`scheduler_hints`의 nonnull 비객체는 `{}`로 바꿉니다. null은 유지합니다. 응답의 canonical/wire alias는 JSON member 순서에서 마지막 값이 우선합니다.

Location은 기존 `resource.CloudLocation.ForResource`를 재사용합니다. 기본은 추가 인증 요청 없이 기록된 provider token scope를 읽고 실제 행의 project/availability_zone을 적용합니다. HTTP body의 `location`은 신뢰하지 않습니다. leaf의 `ServiceClient`에는 Connection의 cloud/region/configured project 이름이 없으므로 이 값은 자동 추정하지 않고 null로 남깁니다. 필요하면 `WithRecordFindLocation(resource.CloudLocation{...})`으로 연결 정보를 제공합니다. 현재 scope와 다른 truthy project ID는 그 원래 JSON을 유지하고 현재 프로젝트 이름을 지웁니다. 옵션의 pointer/map/JSON과 반환 채널은 독립 복사본입니다. Resource.find가 fetch 전에 Volume을 구성하는 순서대로 query seed의 BoolStr/integer descriptor도 discovery/HTTP 전에 검증합니다. 응답이 나중에 seed를 덮어쓸 수 있어도 잘못된 `is_bootable:"yes"`나 `size:"²"`는 먼저 실패합니다.

Pinned proxy가 받는 옵션은 missing/details/all_projects입니다. `WithRecordFindQuery`, `WithRecordFindHeaders`, `WithRecordFindMaxItems`, `WithRecordFindPaginated`, `WithRecordFindLocation`은 underlying Resource 동작을 노출하는 명시적 Go 확장입니다.

- 일반 Query는 member GET에도 Requests 형태로 전달합니다. JSON null은 생략하고, bool은 `True`/`False`, 배열은 반복 query, 객체는 key 배열로 변환합니다. `all_projects`는 별도 Resource 인자처럼 GET에서 제외합니다. raw `all_tenants`는 양쪽 요청의 wire 확장입니다. 명시한 typed AllProjects와 case-insensitive raw all_projects/all_tenants를 함께 주면 순서와 값에 관계없이 사전 오류입니다.
- fallback 서버 query는 `limit`, `marker`, `name`, `status`, `user_id`, `project_id`, `created_at`, `updated_at`, `properties→metadata`, `all_projects→all_tenants`입니다. canonical query key가 wire alias보다 우선하며 명시한 null도 우선권을 유지합니다. name이 없을 때만 identity를 name query로 추가하므로 `WithRecordFindQuery("name", nil)`은 그 query를 생략합니다. offset/sort/volume_id 등 선언되지 않은 kwargs는 fallback 서버 query에서 버립니다.
- 선언된 Body attribute 중 서버 mapping의 client key가 아닌 것은 로컬 필터입니다. `metadata`는 `properties`의 wire alias이자 Body attribute여서 서버 query와 로컬 subset 필터에 동시에 적용됩니다. `properties`를 함께 주면 서버 metadata 값에는 properties가 우선하지만 metadata의 로컬 필터는 유지합니다. object subset은 중첩 필터와 일반 bool/number 동등성을 제공합니다. 공통 `EqualPythonJSON`은 exact-decimal 비교이므로 Source JSON decoder의 binary64 반올림과 다릅니다. 예를 들어 Source에서 같게 비교되는 `9007199254740992`와 `9007199254740993.0`은 Go에서 다르게 비교합니다. 공통 bool descriptor도 exact nonzero를 유지하므로 Source에서 underflow하여 false인 `1e-400`은 Go의 `is_multiattach`/`shared_targets`에서 true입니다. 이 경계는 명시적 Go 정책이며 raw Wire 값은 보존합니다. non-object actual 값에는 불일치를 반환합니다. Go Query map의 descriptor seed는 순서가 없으므로 canonical attribute가 wire alias보다 우선합니다.
- `MaxItems`는 로컬 필터 이전의 원본 row 수를 제한하고 limit이 없거나 0이면 첫 요청에 limit hint를 줍니다. `Paginated(false)`는 첫 페이지에서만 검색합니다. 기본은 전체 페이지 관찰입니다. 공통 pager는 body/HTTP next link와 explicit limit의 short-page marker fallback을 처리합니다. marker는 마지막 원본 행을 다시 읽어 JSON scalar ID를 Source의 Requests 문자열 형태로 변환하므로 큰 정수나 공개 Resource 변경에 영향받지 않습니다. 배열/객체 ID는 Source의 iterable query expansion을 재현하지 않고 공통 guarded pager에 단일 Python repr marker로 전달하는 명시적 Go 정책입니다. 다음 페이지는 고정한 서비스 origin/collection/filter를 유지하고 cycle/음수 limit을 거부합니다. Dictionary links 호환형은 Go 확장입니다.

Source의 `<400` 성공 정책에 맞춰 유효한 HTTP 100–399를 받아들이며, member·fallback HTTP 400–599 rejected body의 Read/Close 원인과 원래 native HTTP proof를 보존합니다. 오류가 취소, 서비스 source 변경, retry callback 실패와 섞이면 fallback/미존재 무시/관찰 재요청을 하지 않습니다. rejected 429/503의 Read+Close 오류와 Read 중 source 변경/Close 중 원복, custom cancellation을 함께 관찰해도 원래 HTTP method/URL/body/header를 유지하며 native retry hook이 nil을 반환해도 재요청하지 않습니다. accepted 응답의 decode/descriptor/Read/Close 실패는 실제 HTTP body/header/status를 `resource.ResponseError`로 보존합니다. 집중 회귀는 기존 Gophercloud public testhelper 기반 `internal/testcloud`, 기존 volume transport/fault fixture, shared FindIdentity/REST/cloudread/microversions/model translator를 재사용합니다. 실제 인증 클라우드 acceptance는 별도입니다.
