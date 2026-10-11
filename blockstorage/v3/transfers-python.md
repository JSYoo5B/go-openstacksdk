# Cinder volume transfer: Python proxy와 Go

기준은 openstacksdk [`ef55d7d`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py)의 transfer proxy와 [Transfer Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/transfer.py)입니다. 새 `Record` API는 SDK가 기본값, 요청 옵션, 경로, 페이지 순회와 이름 검색을 관리합니다. 기존 Gophercloud 타입 API는 그대로 사용할 수 있습니다.

| Python | Go |
|---|---|
| `create_transfer(volume_id="v1", name="handoff", no_snapshots=True)` | `Transfers.CreateRecord(ctx, WithRecordCreateVolumeID("v1"), WithRecordCreateName("handoff"), WithRecordCreateNoSnapshots(true))` |
| `get_transfer("t1")` | `Transfers.GetRecord(ctx, "t1")` |
| `transfers()` | `Transfers.ListRecords(ctx)` |
| `transfers(details=False)` | `Transfers.ListRecords(ctx, WithRecordListDetails(false))` |
| `find_transfer("handoff")` | `Transfers.FindRecord(ctx, "handoff")` |
| `find_transfer("handoff", ignore_missing=False)` | `Transfers.FindRecord(ctx, "handoff", WithRecordFindIgnoreMissing(false))` |
| `accept_transfer("t1", "key")` | `Transfers.AcceptRecord(ctx, "t1", WithRecordAcceptAuthKey("key"))` |
| `delete_transfer("t1")` | `Transfers.DeleteRecord(ctx, "t1")` |
| `delete_transfer("t1", ignore_missing=False)` | `Transfers.DeleteRecord(ctx, "t1", WithRecordDeleteIgnoreMissing(false))` |

아래 Python 호출과 Go 호출은 독립된 예제입니다. Transfer 생성은 volume 소유권 이전을 준비하며, 반환된 `auth_key`는 대상 프로젝트에 안전하게 전달해야 합니다. 생성 이후 실제 이전은 대상 프로젝트의 인증으로 `accept_transfer`를 호출할 때 이루어집니다.

```python
import openstack

conn = openstack.connect(cloud="source")
transfer = conn.block_storage.create_transfer(
    volume_id="v1", name="handoff", no_snapshots=True
)
for candidate in conn.block_storage.transfers(details=False):
    print(candidate.id, candidate.name)
found = conn.block_storage.find_transfer("handoff", ignore_missing=False)
print(found.id)
```

```go
package main

import (
    "context"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers"
)

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx, openstack.WithCloud("source"))
    if err != nil { log.Fatal(err) }
    storage, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    created, err := storage.Transfers.CreateRecord(ctx,
        transfers.WithRecordCreateVolumeID("v1"),
        transfers.WithRecordCreateName("handoff"),
        transfers.WithRecordCreateNoSnapshots(true),
        transfers.WithRecordCreateSuppressLegacySnapshots(true),
    )
    if err != nil { log.Fatal(err) }
    fmt.Println(string(created.Resource.Body["id"]))
    for candidate, err := range storage.Transfers.ListRecords(ctx,
        transfers.WithRecordListDetails(false),
    ) {
        if err != nil { log.Fatal(err) }
        fmt.Println(string(candidate.Resource.Body["id"]), string(candidate.Resource.Body["name"]))
    }
    found, err := storage.Transfers.FindRecord(ctx, "handoff",
        transfers.WithRecordFindIgnoreMissing(false),
    )
    if err != nil { log.Fatal(err) }
    fmt.Println(string(found.Resource.Body["id"]))
}
```

버전과 경로 선택은 서비스에 귀속됩니다. 각 Record 호출은 공통 Cinder 버전 탐색으로 서버의 `min_version`/`max_version`을 확인합니다. 두 경계가 있고 3.55가 범위 안에 있으며 명시된 기본 버전도 3.55를 지원하면 `/volume-transfers`를 선택하고, 그 외에는 `/os-volume-transfer`를 선택합니다. 기본 버전이 없으면 Resource의 상한 3.55와 서버 최대 중 작은 값을 HTTP 버전으로 사용합니다. 명시한 기본 버전은 3.55보다 높아도 유지합니다. 공유 client의 microversion은 변경하지 않습니다. Go는 호출마다 제한된 버전 탐색을 수행하며 Python의 adapter discovery cache를 흉내 내지 않습니다. `TransferRecord.Discovery`에는 실제 탐색 응답을, `Envelope`/`Header`/`StatusCode`에는 실제 transfer 응답을 따로 보존합니다.

`TransferRecord.Resource`는 Python이 선언한 `id`, `name`, `created_at`, `volume_id`, `auth_key`, `links`, `no_snapshots`와 nullable `location`을 제공합니다. 이 필드는 모두 JSON 값을 그대로 보존합니다. 날짜에 시간대가 붙어도 native `time.Time` 해석에 실패하지 않으며, 큰 정수도 float64로 바뀌지 않습니다. 생성 시 입력값에 서버가 반환한 선언 필드를 덮어씁니다. `Wire`에는 서버의 확장 필드를 포함한 원본 resource JSON이 있고 `Envelope`에는 실제 전체 응답이 있습니다. accept 결과는 Source와 같이 입력 transfer ID를 임의로 채워 넣지 않습니다.

`WithRecordCreateField`는 선언된 필드에 한해 omission과 explicit `null`/`false`/`0`/빈 문자열/빈 목록을 구분합니다. 알 수 없는 일반 attribute는 Source처럼 버립니다. 새 서버 필드를 보내려면 `WithRecordCreateExtension("future", value)`를 사용합니다. 이는 Go 확장이며 선언된 필드를 덮어쓸 수 없습니다. Options와 결과의 JSON 및 header는 소유한 복사본으로 관리합니다.

목록은 기본 `details=true`, `paginated=true`입니다. limit/marker만 Source의 서버 query이고 나머지 선언된 Body field는 받은 Resource에 대한 로컬 필터입니다. `WithRecordListQuery("links", map[string]any{"owner": "p1"})` 같은 중첩 필터도 지원합니다. unknown query는 버립니다. `WithRecordListMaxItems`는 로컬 필터를 적용하기 전의 원본 row 수를 제한하며 limit이 없거나 0이면 HTTP limit hint도 설정합니다. 페이지는 소비할 때만 요청하고, 조기 break나 `WithRecordListPaginated(false)` 뒤에는 다음 요청을 하지 않습니다. Body next link, HTTP Link, marker fallback을 공통 pager로 처리하며 실제 다음 페이지 오류의 body/header/status를 보존합니다. 다음 URL은 고정 서비스 origin과 collection에 제한합니다. Source의 dictionary links 처리보다 넓은 `links:{next:...}` 호환형도 지원하지만 임의 다른 서비스로 이동하지 않습니다.

고정한 Python 소스에는 세 가지 주의할 동작이 있습니다.

- `transfers(all_projects=True)`는 Transfer가 `all_projects` Body/query를 선언하지 않아 실제로 그 값을 버립니다. Go `WithRecordListAllProjects(true)`도 해당 Source 호출을 재현합니다. 서버에 명시적으로 admin query를 보내려면 Go 확장 `WithRecordListServerQuery("all_tenants", "True")`를 사용합니다.
- 3.55 미만에서 create는 `params` 안의 `no_snapshots`만 제거합니다. Proxy는 이 값을 Resource attribute에 미리 넣으므로 실제 `create_transfer(no_snapshots=True)`는 legacy body에도 그대로 보냅니다. 기본 Go profile이 이 실제 동작을 보존합니다. 예제의 `WithRecordCreateSuppressLegacySnapshots(true)`는 3.55 미만에서는 본문에서 해당 key를 제거하는 명시적 Go 개선입니다. 응답 view는 Source 입력 Resource처럼 그 입력값을 기억합니다.
- `find_transfer`는 먼저 discovered route로 GET을 하지만 fallback은 Resource.find가 Transfer.list를 직접 호출하므로 3.55 미만에도 `/volume-transfers` 요약 경로입니다. Go `FindRecord`는 이 정확한 Source 경로를 유지합니다. 직접 GET에 깨끗한 400/403/404가 있을 때만 전체 목록에서 정확한 ID 또는 이름을 찾으며, 중복은 `resource.ErrAmbiguous`, 기본 미존재는 `nil, nil`입니다.

GET fallback과 default delete의 미존재 처리는 깨끗한 HTTP 오류에만 적용합니다. rejected body 읽기/Close 실패, 취소, retry callback 실패 또는 서비스 source 변경이 섞이면 이를 무시하거나 검색을 계속하지 않습니다. Strict delete의 404는 원래 Gophercloud 오류로 반환하므로 `gophercloud.ResponseCodeIs(err, 404)`로 검사합니다. 이 테스트는 공통 `internal/testcloud`가 재사용하는 Gophercloud testhelper fixture와 HTTP transport로 검증하며 인증된 실제 클라우드 acceptance 실행은 별도입니다.
