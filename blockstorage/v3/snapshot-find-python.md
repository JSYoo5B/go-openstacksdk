# Cinder v3 snapshot 찾기

기준은 Gophercloud **v2.15.0**, openstacksdk **ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe**의 [find_snapshot proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L154-L193), [Snapshot descriptor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/snapshot.py), Resource.find입니다.

| openstacksdk | Go | 기본값 |
|---|---|---|
| `find_snapshot(name_or_id)` | `Snapshots.FindRecord(ctx, nameOrID)` | member GET → clean400/403/404만 fallback, 없으면 nil,nil |
| `find_snapshot(name_or_id, details=False)` | `WithRecordFindDetails(false)` | fallback 목록 `/snapshots`; 기본은 `/snapshots/detail` |
| `find_snapshot(name_or_id, all_projects=True)` | `WithRecordFindAllProjects(true)` | fallback query `all_tenants=True`; member GET에는 붙이지 않음 |
| `find_snapshot(name_or_id, ignore_missing=False)` | `WithRecordFindIgnoreMissing(false)` | 전체 검색 후 missing 오류 |

```python
import openstack

conn = openstack.connect(cloud="dev", block_storage_default_microversion="3.65")
snapshot = conn.block_storage.find_snapshot("daily")
summary = conn.block_storage.find_snapshot("daily", details=False)
# all_projects는 서버 정책에 따라 관리자 권한이 필요합니다.
strict = conn.block_storage.find_snapshot(
    "daily", details=True, all_projects=True, ignore_missing=False)
```

## 독립 Go main

[설치 안내](../../docs/install.md)대로 모듈을 준비하고 아래를 `main.go`에 저장합니다. `go run . -cloud dev -snapshot daily`는 찾기만 수행합니다. deployment가 지원하는 microversion을 `-microversion`으로 지정합니다.

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
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    identity := flag.String("snapshot", "daily", "snapshot name or ID")
    details := flag.Bool("details", true, "use detailed fallback list")
    allProjects := flag.Bool("all-projects", false, "search across projects")
    strict := flag.Bool("strict", false, "return an error when missing")
    version := flag.String("microversion", "3.65", "Cinder selected version; empty enables discovery")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
        sdk.WithMicroversion(sdk.BlockStorage, *version))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    value, err := service.Snapshots.FindRecord(ctx, *identity,
        snapshots.WithRecordFindDetails(*details),
        snapshots.WithRecordFindAllProjects(*allProjects),
        snapshots.WithRecordFindIgnoreMissing(!*strict))
    if err != nil { log.Fatal(err) }
    if value == nil { fmt.Println("null"); return }
    data, err := json.MarshalIndent(value.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

## 관찰·검색 계약

`FindRecord`는 name과 ID를 구분하지 않고 literal identity를 한 번 escape해 `GET /snapshots/{identity}`를 보냅니다. member는 항상 query 없이 요청하며 details/all_projects도 붙이지 않습니다. 성공하면 반환하고 list를 호출하지 않습니다. clean400/403/404만 fallback 대상입니다. 다른 HTTP 상태, accepted decode 실패, Read/Close·source·retry callback·취소 원인이 섞인 상태는 fallback하지 않습니다.

Fallback은 details가 생략되거나true이면 `/snapshots/detail`, false이면 `/snapshots`입니다. `name=identity`를 자동으로 넣고 각 행의 **정확한 id 또는 정확한 name**을 비교합니다. 첫 일치 뒤에도 전체 페이지를 관찰해 늦은 중복과 후속 오류를 숨기지 않습니다. 중복은 resource.ErrAmbiguous, 완전한 검색에서 누락은 기본nil,nil, strict는 resource.ErrNotFound입니다. 억제한 최초400/403/404를 논리적 누락의 cause에 섞지 않습니다. all_projects는true일 때만 list에all_tenants=True를 추가합니다. stock policy에서 관리자 권한이 필요하면 실제 거부 상태를 반환합니다.

`SnapshotRecord.Resource`는15개 Body 속성과 nullable location의 view입니다. id/name/consumes_quota/created_at/description/group_snapshot_id/is_forced/progress/project_id/size/status/updated_at/user_id/volume_id/metadata를 제공하며 누락은null입니다. size는 공통 Source int descriptor 변환을 적용해 bool이int subtype인 동작·숫자 Unicode 문자열·정확한 정수를 보존합니다. is_forced는 BoolStr의 bool 또는 대소문자 구분 없는 true/false 문자열을 받고 나머지는 실제 HTTP 증거를 가진 오류입니다. metadata는null을 유지하며 nonnull nonobject는 Source dict descriptor처럼{}로 표시합니다.

Wire에는 unknown keys·원래 descriptor 값·큰 정수·literal timestamp가 있습니다. Envelope는 member 전체 또는 선택한 실제 list page의 본문, Header/StatusCode는 그 실제 응답입니다. view·Wire·Envelope·각 header는 독립 복사합니다. member 응답에서id가 생략되면 요청 identity를 view에만 seed하며 explicit null/number id는 그대로 둡니다. list 행에는id를 seed하지 않습니다. 응답 alias는 JSON member 순서의 마지막 값이 우선합니다. nullable location은 Python Connection location을 합성하지 않습니다.

member는 snapshot singular envelope 또는 flat object를 받습니다. 빈 body·malformed JSON syntax는 Source _translate_response처럼 seed를 유지합니다. 유효한 null/scalar/array 또는 잘못된 UTF-8는 actual ACK를 가진 오류입니다. list는 plural snapshots array 또는 singleton object를 받습니다. null 행·잘못된 descriptor·Resource constructor와 충돌하는 connection/microversion/_synchronized는 실제 page를 가진 오류입니다.

## Go fallback controls

Python의 named find_snapshot proxy는 ignore_missing/details/all_projects 외의kwargs를 받지 않습니다. Go의 `WithRecordFindQuery`·Limit·Marker·MaxItems·Paginated·Headers는 library-owned 추가 기능이며 query는 fallback list에만 적용합니다. Source Resource.list mapping의 name/status/volume_id/project_id/limit/marker/offset/sort_dir/sort_key/sort는 server query, all_projects는 all_tenants로 변환합니다. wire alias도 받고 client alias가null이어도 우선해 wire query를 생략합니다. typed AllProjects가 있으면 raw alias들을 교체하며 false는 둘 다 생략합니다.

다른 declared attribute는 local filter입니다. unknown query·wire-only Body alias는 무시하며 id는 identity와 충돌하므로 거부합니다. 공통 Requests encoder의 True/False·iterable expansion·null 생략과 JSON-domain Python equality·dictionary subset을 재사용합니다. truthy scalar에 빈 object filter는 Source처럼 일치하며, nonempty filter에 mapping.get이 필요하면 오류입니다. Go map에는 kwargs 순서가 없어서 여러 local filter는 key 순서로 평가합니다. name을 명시null로 넘기면 자동 name hint도 넣지 않습니다.

MaxItems는 local filter 이전 raw 행 수를 제한하고, 생략했거나0인 limit에 최초 hint를 줍니다. Paginated(false)는 첫 page만 읽습니다. 기본값은 무제한이며 전체 중복 검색을 수행합니다. cap/single-page를 선택하면 관찰 범위의 결과를 반환하며 미관찰 중복까지 없다고 보장하지 않습니다. 음수 typed limit/cap은 HTTP 전에 오류입니다.

공통 REST pager로 snapshots_links·top-level next·HTTP Link·short-page marker fallback·empty-page 종료를 처리합니다. marker는 반환 view 변경과 독립인 원래 wire id를 사용하며 문자열이 아니면 오류입니다. offset만 명시한 Go fallback은 offset pager로 증가 cursor와 최초 server limit을 검증합니다. 같은 origin/collection과 초기 filter를 유지하고 cross-origin·path/filter 변경·loop를 차단합니다. dictionary links는 Go 호환 확장입니다. Source의 offset 요청 뒤 marker 자동 fallback과 달리 offset-only Go 확장에는 증가하는 offset continuation만 허용합니다. cap/single-page는 next를 해석하기 전에 종료합니다.

## Source·microversion·물리 오류

설정한 microversion은3.65로 clamp하지 않습니다. default가 비어 있으면 공통 Cinder discovery/Negotiate로 광고 범위에서 Source Snapshot ceiling3.65를 선택하고 private client에만 적용합니다. Discovery에는 실제 accepted version 응답을 별도로 보관합니다. discovery 실패는 snapshot ACK를 만들지 않고 Discovery만 가진 proof를 반환합니다. version header 또는 그 부재를 member/list retry와 physical attempt에서도 고정합니다. Discovery GET은 bodyless/versionless이며 유한한 endpoint/root 방식을 사용합니다.

원래200..399 success policy를 적용하고400..599 거부 body의 Read/Close도 공통 observer로 감시합니다. clean native retry hook이 같은 거부를 반환한 경우 fallback 대상이 유지됩니다. accepted member의 부분 body/Read/Close/decode 실패는 SnapshotRecord에 actual ACK를 반환하며 resource.ResponseError에는 status/body/header/cause가 있습니다. list 실패·후속 중복은 논리적 결과를 반환하지 않고 오류 증거를 보존합니다. Source가Read에서 바뀌고Close에서 원복되어도 이미 관측한 실패를 유지합니다. cancel custom cause와 callback cause를 함께 보존합니다. 입력·옵션을 독립 복사하므로 보관한 callback pointer나 동시 옵션 재사용으로 뒤 phase policy가 바뀌지 않습니다.

Python mutable Resource/cache/session/hooks/base_path/Connection location은 별도 공유 SDK 검토 범위입니다. 기존 generated Get/List/ListDetail/Find와 native Resources의 계약은 유지합니다. [계약 테스트](snapshots/python_find_contracts_test.go)는 기존 public Gophercloud testhelper fixture로 lookup·descriptor·query·pagination·물리 cause·negotiation·소유권을 검증합니다. 실제 인증된 Cinder policy/endpoint 확인은 local fixture와 별개입니다.
