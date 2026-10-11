# Cinder group snapshot: ID 또는 이름 찾기

기준은 openstacksdk [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2690-L2719), Gophercloud **v2.15.0**입니다. `groupsnapshots.API.FindRecord`는 generic `/group_snapshots`를 찾습니다. 일반 volume snapshot `/snapshots`, legacy `/cgsnapshots`, Manila share-group snapshot은 다른 자원입니다.

| Python | Go | 기본값 |
|---|---|---|
| `conn.block_storage.find_group_snapshot(name_or_id)` | `api.FindRecord(ctx, identity)` | `ignore_missing=True`, `details=True` |
| `ignore_missing=False` | `WithRecordFindIgnoreMissing(false)` | 완료한 검색에서 없으면 오류 |
| `details=False` | `WithRecordFindDetails(false)` | fallback만 summary `/group_snapshots` |

고정한 named Python proxy는 `name_or_id`, `ignore_missing`, `details`만 받습니다. Go의 Header, Location, Microversion, Query, Paginated, MaxItems 옵션은 아래에 범위를 적은 추가 기능입니다. Python named 호출에 대응하지 않는 `**query`를 예제에 붙이지 않습니다.

```python
import argparse
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("identity")
parser.add_argument("--strict", action="store_true")
parser.add_argument("--summary", action="store_true")
args = parser.parse_args()
conn = openstack.connect()
value = conn.block_storage.find_group_snapshot(
    args.identity,
    ignore_missing=not args.strict,
    details=not args.summary,
)
print(None if value is None else value.to_dict())
```

아래 Go 코드는 standalone main 프로그램입니다. 환경변수 또는 clouds.yaml의 기존 `Connect` 설정으로 인증합니다. 조회·목록의 project reader 규칙과 관리자 `all_tenants` 규칙은 [Cinder 공식 정책](https://docs.openstack.org/cinder/latest/configuration/block-storage/policy.html)을 기준으로 하며, 실제 배포 정책이 접근 권한을 결정합니다.

```go
package main

import (
    "context"
    "errors"
    "flag"
    "fmt"
    "log"
    "time"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    identity := flag.String("identity", "group-snapshot-id", "group snapshot ID or exact name")
    strict := flag.Bool("strict", false, "report completed search absence as an error")
    summary := flag.Bool("summary", false, "use summary list only on fallback")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    api := groupsnapshots.New(cinder.RawClient())
    value, err := api.FindRecord(ctx, *identity,
        groupsnapshots.WithRecordFindIgnoreMissing(!*strict),
        groupsnapshots.WithRecordFindDetails(!*summary))
    if err != nil {
        if value != nil {
            fmt.Printf("actual_http=%d actual_body=%q discovery=%d\n", value.StatusCode, value.Envelope, len(value.Discovery))
        }
        if errors.Is(err, resource.ErrNotFound) { log.Fatal("completed search found no group snapshot") }
        if errors.Is(err, resource.ErrAmbiguous) { log.Fatal("more than one ID/name match") }
        log.Fatal(err)
    }
    if value == nil { fmt.Println("no matching group snapshot"); return }
    fmt.Printf("id=%s name=%s status=%s\n", value.Resource.Body["id"], value.Resource.Body["name"], value.Resource.Body["status"])
    fmt.Printf("actual_http=%d actual_body=%q discovery=%d\n", value.StatusCode, value.Envelope, len(value.Discovery))
}
```

`FindRecord`는 먼저 identity를 한 경로 segment로 escape해 GET합니다. 내부 공백, slash, percent는 그대로 식별자로 보존합니다. 빈 값, dot segment, control 문자, 잘못된 UTF-8은 HTTP 전에 거부합니다. GET이 성공하면 응답 ID가 입력과 다르거나 null·숫자라도 재검색하지 않습니다. GroupSnapshot의 여덟 Body descriptor는 Source에서도 untyped입니다. 누락 값은 null, 반환하지 않은 GET seed 값은 보존하며, 서버 null은 seed를 덮어씁니다.

오류 없는 native 400·403·404만 fallback을 허용합니다. 기본 `/group_snapshots/detail`, summary 옵션은 `/group_snapshots`를 조회합니다. GroupSnapshot의 Source query mapping에는 `name`이 없으므로 입력 이름을 자동 서버 query나 로컬 name filter로 추가하지 않습니다. 모든 행에서 정확히 `id == identity` 또는 `name == identity`를 검사합니다. ID만 맞는 행도 후보입니다. 유일한 후보는 목록을 끝까지 확인한 후 반환하고, 두 번째 후보를 만나면 `ErrAmbiguous`를 반환합니다. 후보 뒤의 HTTP·decode·paging·physical 오류는 그 후보로 숨기지 않습니다. fallback LIST 404는 missing으로 무시하지 않습니다. 완전히 끝난 검색에서만 기본 `(nil, nil)`, strict `ErrNotFound`가 나오며, 논리적 missing 오류에 이미 억제한 GET 오류를 붙이지 않습니다.

`Resource`, `Wire`, `Envelope`, `Header`, `StatusCode`는 서로 다른 carrier입니다. `Resource`는 여덟 선언 descriptor와 계산한 location, `Wire`는 실제 객체의 unknown JSON까지, `Envelope`는 실제 member 응답 또는 후보가 나온 전체 page body를 보존합니다. 터미널 HTTP 실패의 반환 record는 실제 응답 근거만 가지며 성공한 `Resource`를 만들지 않습니다. 논리적 missing·duplicate에는 가짜 응답이나 마지막 성공 후보를 반환하지 않습니다. `Discovery`는 실제 버전 조회 응답이며 자원 조회의 성공 근거와 분리됩니다. 조회 결과는 현재 backend 완료·내구성·복제 상태의 추가 증거가 아닙니다.

소유 옵션의 범위는 다음과 같습니다.

- `WithRecordFindHeaders`는 두 단계에 적용하며, 기본 LIST `Accept: application/json`보다 명시한 Accept가 우선합니다. auth·route·framing·version headers는 공통 검증을 따릅니다. native retry에서도 고정한 route/header/version이 유지됩니다.
- `WithRecordFindMicroversion`은 underlying `Resource.find`처럼 **GET 전용**입니다. fallback은 원래 선택한 서비스 버전을 사용하고, 없으면 별도로 Source ceiling **3.29**까지의 유한 Cinder discovery를 합니다. GET에서 선택한 버전이나 explicit empty는 LIST에 넘기지 않습니다. explicit empty는 GET version headers 부재를 뜻하는 Go 추가 옵션입니다. 원래 서비스 버전은 변경하지 않습니다. Python session/Adapter discovery cache와 달라 default 두 단계가 각각 실제 discovery를 할 수 있습니다.
- `WithRecordFindLocation`은 소유한 CloudLocation을 사용합니다. 없으면 이미 기록한 인증 project만 재사용하며 cloud/region 이름이나 추가 프로젝트 조회를 추측하지 않습니다. constructor descriptor/location projection은 discovery와 GET 전에 검증합니다.
- `WithRecordFindQuery`는 underlying `Resource.find`의 고정 경로 JSON parameter profile을 제공합니다. `fields` 등 ordinary unknown key는 GET query로 전달하고 LIST에서는 Source query mapping에 따라 버립니다. JSON null은 query에서 생략하고 bool은 `True`/`False`, 배열은 반복 값, 객체는 key 반복으로 펼칩니다. 선언 Body attrs는 GET constructor seed·query와 LIST 로컬 predicate로 동작합니다. `all_projects`는 bool/null을 받으며 GET에서 제외하고, false를 포함한 값은 LIST `all_tenants`로 전달합니다. null은 이 special parameter 부재입니다. Source server alias·`sort`·`limit`·`marker`·`offset`은 기존 목록 구현을 재사용합니다.
- `WithRecordFindPaginated(false)`는 첫 fallback page만 검색하고, `WithRecordFindMaxItems(n)`은 local filter 전에 raw 행 수를 제한합니다. 둘 다 검색 범위를 줄이므로 전체 cloud의 유일성을 확인하는 옵션이 아닙니다. zero cap은 무제한, 음수는 HTTP 전에 거부합니다. raw `paginated`/`max_items`는 이 named 옵션을 사용하도록 거부합니다.

Query factory는 생성 시 JSON을 snapshot합니다. custom Marshaler도 이 시점에 호출되므로 caller가 옵션을 만들기 전·후에 별도로 한 변경은 호출의 captured source 구간에 포함되지 않습니다. API는 옵션 callback 전에 서비스를 한 번 capture하고 callback 양쪽에서 context와 sticky source guard를 검사합니다. callback이 받은 map/raw bytes/header/location은 다음 callback 및 GET/LIST 단계와 분리됩니다. reserved constructor `id`/`cls`/`self`/`connection`/`_synchronized`는 값이 null·false·zero라도 충돌하며 custom Marshaler를 실행하지 않습니다. 임의 base path, method control, subclass, mutable Resource/session/Adapter/cache/dirty 상태, Python 경고 체계는 이 named mapping에 포함하지 않습니다.

목록과 local predicate는 기존 그룹 snapshot 구현의 공통 엔진을 재사용합니다. Go stable query-key order와 exact-decimal 숫자 비교는 Python kwargs 순서·JSON float 반올림과 다릅니다. 예를 들어 실제 integer `9007199254740992`와 filter `9007199254740993.0`은 Go에서 다르고 Python binary64 처리에서는 같아질 수 있습니다. recursive dictionary subset, truthiness, bool/number equality는 shared Source predicate를 유지합니다. 페이지 링크는 origin·collection·filter·cursor·cycle 검증을 하며 advertised marker/offset을 지원합니다. offset은 자동 증가하지 않고, 하나의 cursor family와 nonnegative uint64 offset을 요구합니다. Requests의 표준 Link `url`과 pinned Source의 `uri` 조회 차이는 shared pager의 문서화한 개선 범위입니다.

버전 discovery는 공통 identity 검색 엔진에 들어가기 전에 완료합니다. discovery의 clean 400·403도 자원 GET의 rejection으로 바꾸지 않으며, 실제 native body/header/status를 가진 오류와 receipt를 반환하고 member GET·LIST를 실행하지 않습니다. Read/Close/context/선택 서비스 변경은 400·403·404에도 fallback·retry·missing 성공을 허용하지 않습니다. 실제 부분 body와 native rejection을 보존하고 source가 Close에서 복구돼도 sticky 오류를 유지합니다. shared public Gophercloud testhelper 기반 fixture와 기존 fault injector로 GET/LIST, 후속 페이지, 버전, ownership, retry를 검증합니다. 이것은 실제 인증 cloud나 generic group snapshot 지원 backend에서의 인수 테스트와 별개의 근거입니다.
