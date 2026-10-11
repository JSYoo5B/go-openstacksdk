# Cinder backup 이름 또는 ID 검색

기준은 openstacksdk [`ef55d7d`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py)의 `find_backup`과 [Backup Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py)입니다. `FindRecord`는 GET과 목록 검색을 SDK가 연결하고 Python에 선언된 응답 view와 원래 HTTP 증거를 함께 반환합니다.

| Python | Go |
|---|---|
| `conn.block_storage.find_backup("nightly")` | `service.Backups.FindRecord(ctx, "nightly")` |
| `find_backup("nightly", ignore_missing=False)` | `FindRecord(ctx, "nightly", backups.WithRecordFindIgnoreMissing(false))` |
| `find_backup("nightly", details=False)` | `FindRecord(ctx, "nightly", backups.WithRecordFindDetails(false))` |

```python
import openstack

conn = openstack.connect(cloud="source")
backup = conn.block_storage.find_backup(
    "nightly", ignore_missing=False, details=False
)
print(backup.id, backup.name, backup.created_at)
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
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups"
)

func main() {
    identity := flag.String("backup", "nightly", "Backup name or ID")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud("source"))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    backup, err := service.Backups.FindRecord(ctx, *identity,
        backups.WithRecordFindIgnoreMissing(false),
        backups.WithRecordFindDetails(false),
    )
    if err != nil { log.Fatal(err) }
    body, err := json.MarshalIndent(backup.Resource.Body, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(body))
}
```

기본 호출은 먼저 `GET /backups/{name_or_id}`를 보냅니다. 깨끗한 HTTP 400/403/404에 한해 `GET /backups/detail?name={name_or_id}`로 fallback합니다. `details=false`는 fallback 목록을 `/backups?name=...`로 바꿉니다. direct GET의 경로는 동일합니다. 받은 모든 페이지에서 정확한 문자열 ID 또는 이름을 비교하고, 둘 이상의 일치는 `resource.ErrAmbiguous`, 일치가 없으면 기본 `nil, nil`입니다. `WithRecordFindIgnoreMissing(false)`는 미존재를 `resource.ErrNotFound`로 반환합니다. 이전 native `Find(ctx, resource.ID(...))`와 `Find(ctx, resource.Name(...))`의 동작은 그대로 유지됩니다.

Backup의 Source 상한은 3.64입니다. 이미 client에 microversion이 명시돼 있으면 Source의 session default처럼 그것을 그대로 사용하며 discovery를 호출하지 않습니다. 기본 버전이 없으면 공통 Cinder discovery로 서버 최대를 확인하여 최대 3.64를 선택합니다. 공유 client 설정을 바꾸지 않고 GET과 모든 fallback 페이지에 같은 선택 버전을 사용합니다. `Discovery`는 실제 받아들인 탐색 응답의 별도 기록입니다. Go는 필요한 호출마다 제한된 탐색을 수행하므로 Python adapter discovery cache를 재현하지 않습니다.

반환 타입 `BackupResourceRecord`의 `Resource`는 Source가 선언한 Body 23개와 nullable `location`을 제공합니다. unknown 응답 필드는 `Wire`에 보존하고 `Envelope`, `Header`, `StatusCode`는 실제 GET 또는 일치 행이 나온 목록 페이지의 증거입니다. direct GET의 ID는 입력값을 seed로 쓰되 실제 응답 ID가 있으면 그것으로 덮어씁니다. `created_at` 등의 날짜는 native `time.Time`으로 강제 변환하지 않습니다. Body의 untyped ID도 큰 정수 등 원래 JSON을 유지합니다. fallback에서 숫자 ID는 문자열 identity와 같다고 간주하지 않습니다.

bool descriptor는 JSON truthiness를, `size`/`object_count`는 Source의 integer descriptor를 적용합니다. 따라서 문자열 `"False"`의 bool 결과는 true이고, `object_count:true`는 Python의 bool/int 관계를 따라 true 그대로입니다. `metadata`는 null을 유지하며 nonnull 비객체는 `{}`로, `links`의 nonnull 비목록은 단일 요소 목록으로 변환합니다. project ID의 `project_id`/`os-backup-project-attr:project_id` 응답 별칭은 JSON dictionary 순서대로 마지막 값이 우선합니다. 원본 `Wire`에는 변환 전 값과 별칭이 모두 남습니다. 채널별 JSON/header와 옵션의 map/slice는 독립된 복사본입니다.

Pinned proxy의 입력은 `ignore_missing`과 `details`입니다. 다음 옵션들은 underlying Resource 동작을 노출하는 명시적 Go 확장입니다.

- `WithRecordFindQuery`는 JSON 값을 Requests query 형태로 바꿉니다. 일반 kwargs는 member GET에도 보내며 unknown 값은 fallback 목록에서는 버립니다. `all_projects`는 Resource.find의 별도 인자처럼 member GET에서 제외하고 fallback의 `all_tenants`로 바꿉니다.
- fallback 서버 query는 `limit`, `marker`, `offset`, `project_id`, `name`, `status`, `volume_id`, `sort_key`, `sort_dir`, `sort`, `all_projects`입니다. canonical client key가 wire alias보다 우선합니다. `name`이 제공되지 않았을 때만 identity를 name query로 추가합니다. 명시한 `name:null`은 그 query를 보내지 않고 전체 결과에서 검색합니다.
- 서버 query에 없는 선언된 Body attribute는 받은 Resource에 로컬 필터를 적용합니다. `WithRecordFindQuery("metadata", map[string]any{"job": "nightly"})`는 중첩 subset 필터입니다. Python의 숫자/bool 동등성도 유지합니다. non-object 응답에 대한 object 필터는 불일치로 처리하여 Source에서 생길 수 있는 타입 오류를 예방합니다.
- `WithRecordFindMaxItems(n)`은 로컬 Body 필터 이전의 원본 row 수를 제한합니다. limit이 없거나 0이면 첫 HTTP limit hint도 설정합니다. `WithRecordFindPaginated(false)`는 첫 페이지까지만 검색하므로 일부 결과에서 검색한다는 선택입니다. 제한을 지정하지 않으면 전체 페이지를 관찰해 late duplicate와 late 오류를 감지합니다.
- marker 페이지는 Body/HTTP next link와 explicit limit의 short-page marker fallback을 공통 pager로 처리합니다. fallback marker는 공통 pager가 보존한 마지막 원본 row를 다시 읽어 Source의 untyped ID를 Requests 문자열로 변환하며, 공개 Resource view나 float64 변환에 의존하지 않습니다. 초기 offset을 명시하면 shared offset pager를 사용하고 marker 혼합·역행을 거부하며 marker fallback은 사용하지 않습니다. 이 offset 선택은 Source의 marker fallback과 구별되는 Go 정책입니다. 다음 페이지는 고정한 서비스 origin과 collection에만 갈 수 있고 필터를 바꿀 수 없습니다. dictionary links 호환형은 명시적인 Go 확장입니다.

직접 GET·fallback 목록은 HTTP 400–599 rejected body의 Read/Close 원인도 보존합니다. 이것이 취소, retry callback 실패, 서비스 source 변경과 섞이면 미존재로 무시하거나 다른 요청을 보내지 않습니다. 받아들인 응답의 decode/descriptor/Read/Close 오류는 실제 response body/header/status를 함께 돌려줍니다. default missing은 깨끗한 GET 오류 뒤 목록 관찰이 성공한 경우에만 성립합니다. 옵션 콜백에서 source를 바꾸거나 취소하면 HTTP 전에 실패합니다.

이 API는 기존 stock `/backups` 조회 및 목록 경로를 사용합니다. `fetch_backup_metadata`, `get_backup_metadata`, `set_backup_metadata`, `delete_backup_metadata`의 `/backups/{id}/metadata` child 경로 네 개는 stock Cinder 라우터에 없어 unresolved인 별도 항목이며, `FindRecord`를 구현했다고 지원으로 판정하지 않습니다. 집중 회귀는 기존 Gophercloud public testhelper 기반 `internal/testcloud` fixture와 shared FindIdentity/cloudread/microversions/REST pager를 재사용합니다. 실제 인증 클라우드 acceptance는 별도입니다.
