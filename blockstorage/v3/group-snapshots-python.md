# Cinder group snapshot: 조회·목록·생성·삭제

기준은 openstacksdk [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group_snapshot.py), Gophercloud **v2.15.0**입니다. generic group snapshot은 `/group_snapshots` 자원입니다. 일반 volume snapshot `/snapshots`, legacy consistency-group snapshot `/cgsnapshots`, Manila share-group snapshot은 각각 다른 자원입니다. 고정한 Gophercloud에는 generic group snapshot package가 없으므로 `groupsnapshots`의 소유한 record API를 사용합니다.

| Python `conn.block_storage` | Go `groupsnapshots.API` | 기본 동작 |
|---|---|---|
| `get_group_snapshot(id)` | `GetRecord(ctx, id, options...)` | literal member GET, 검색/refresh 없음 |
| `group_snapshots(details=True, **query)` | `ListRecords(ctx, options...)` | lazy `/group_snapshots/detail` 목록 |
| `create_group_snapshot(**attrs)` | `CreateRecord(ctx, attrs, options...)` | wrapped `group_snapshot` POST |
| `delete_group_snapshot(id, ignore_missing=True)` | `DeleteRecord(ctx, id, options...)` | bodyless DELETE, clean404만 무시 |

아래 프로그램은 기본값으로 조회만 합니다. 실제 group snapshot 생성에는 이를 지원하는 group/backend가 필요합니다. Cinder의 [공식 정책](https://docs.openstack.org/cinder/latest/configuration/block-storage/policy.html)은 일반 조회·목록에 project reader, 생성·삭제에 project member를 허용하는 규칙을 제공합니다. `all_tenants` 조회와 project attribute 노출은 별도 관리자 정책을 따르며 배포 정책이 최종 권한을 결정합니다. [공식 API](https://docs.openstack.org/api-ref/block-storage/v3/#group-snapshots-group-snapshots)의 요청 접수와 backend 처리 완료를 구분하세요.

```python
import argparse
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--action", choices=["get", "list", "create", "delete"], default="get")
parser.add_argument("--id", default="group-snapshot-id")
parser.add_argument("--group", default="group-id")
args = parser.parse_args()
conn = openstack.connect()
if args.action == "get":
    print(conn.block_storage.get_group_snapshot(args.id).to_dict())
elif args.action == "list":
    for item in conn.block_storage.group_snapshots(details=False, status="available", limit=10):
        print(item.to_dict())
elif args.action == "create":
    print(conn.block_storage.create_group_snapshot(group_id=args.group, name="sdk-example").to_dict())
else:
    print(conn.block_storage.delete_group_snapshot(args.id, ignore_missing=False))
```

Go 예제는 standalone main 프로그램입니다. 인증에는 기존 SDK `Connect` 환경변수/clouds.yaml 구성을 사용하고, group snapshot leaf에도 같은 authenticated Cinder client를 전달합니다.

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "time"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots"
)

func show(value *groupsnapshots.GroupSnapshotRecord) {
    if value == nil { return }
    if value.Resource != nil {
        fmt.Printf("id=%s name=%s status=%s\n", value.Resource.Body["id"], value.Resource.Body["name"], value.Resource.Body["status"])
    }
    fmt.Printf("actual_http=%d discovery=%d actual_body=%q\n", value.StatusCode, len(value.Discovery), value.Envelope)
}

func main() {
    action := flag.String("action", "get", "get, list, create, or delete")
    id := flag.String("id", "group-snapshot-id", "existing group snapshot ID")
    group := flag.String("group", "group-id", "existing source group ID")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    api := groupsnapshots.New(cinder.RawClient())
    switch *action {
    case "get":
        value, err := api.GetRecord(ctx, *id)
        show(value)
        if err != nil { log.Fatal(err) }
    case "list":
        for value, err := range api.ListRecords(ctx,
            groupsnapshots.WithRecordListDetails(false),
            groupsnapshots.WithRecordListQuery("status", "available"),
            groupsnapshots.WithRecordListLimit(10)) {
            if err != nil { log.Fatal(err) }
            show(value)
        }
    case "create":
        value, err := api.CreateRecord(ctx, map[string]any{"group_id": *group, "name": "sdk-example"})
        show(value)
        if err != nil { log.Fatal(err) }
    case "delete":
        receipt, err := api.DeleteRecord(ctx, *id, groupsnapshots.WithRecordDeleteIgnoreMissing(false))
        if receipt != nil { fmt.Printf("actual_http=%d ignored404=%t actual_body=%q\n", receipt.StatusCode, receipt.IgnoredMissing, receipt.Envelope) }
        if err != nil { log.Fatal(err) }
    default:
        log.Fatal("unknown action")
    }
}
```

`GroupSnapshotRecord.Resource`는 여덟 untyped Body field `created_at`, `description`, `group_id`, `group_type_id`, `id`, `name`, `project_id`, `status`와 computed location의 아홉 필드를 제공합니다. 없는 값은 null이고, bool·0·array·nested object·큰 정수·timestamp 문자열을 native string/time 타입으로 강제하지 않습니다. metadata는 이 Source class의 descriptor가 아니므로 선언된 view에 추가하지 않습니다. `Wire`는 JSON object 행/본문의 unknown 값까지 보존하고, `Envelope`는 실제 member/create envelope 또는 행이 나온 전체 페이지입니다. 각 채널과 Header는 독립 복사본입니다.

GET과 create는 `group_snapshot` wrapper 또는 flat object를 읽습니다. 성공한 empty/malformed JSON body는 Source `_translate_response`처럼 seed 값을 유지합니다. JSON으로 파싱된 root 또는 wrapper의 null/scalar/list, invalid UTF-8와 descriptor/projection 오류는 실제 HTTP status·body·header를 가진 오류입니다. mutation ACK나 `status:"creating"`만으로 완료/복제/일관성을 보장하지 않습니다. delete는 opaque body를 해석하지 않고 실제 DELETE receipt를 돌려줍니다. Python이 delete에서 None을 반환하는 것과 다른 소유한 Go 증거입니다.

`CreateRecord`는 service source를 먼저 캡처하고 알려진 attrs만 JSON snapshot한 뒤 options를 적용합니다. nil/empty attrs는 `{"group_snapshot":{}}`이며 Source의 `Resource.new` dirty 상태대로 **명시한 id도 POST 본문에 남습니다**. 이름/description/group_id의 생략·null·false·zero·empty를 구분하고 unknown attrs는 직렬화하지 않습니다. source constructor와 충돌하는 `self`, `cls`, `connection`, `_synchronized` 입력은 HTTP 전에 거절합니다. `microversion`, `base_path`, mutable Resource/session control과 `__conflicting_attrs` overload는 Body attr로 보내지 않습니다. 서버가 허용하는 생성 필드와 필요한 group/backend를 최종 검증합니다.

각 method의 concrete `WithRecordRead/Create/Delete/ListHeaders`와 `WithRecordRead/Create/Delete/ListMicroversion`으로 옵션을 지정합니다. 모든 source/option/header snapshot은 shared client를 변경하지 않습니다. `RecordReadOpts`·`RecordListOpts`를 사용하거나 `WithRecordReadSeed(previous)`로 GET의 알려진 declared fields만 seed할 수 있습니다. seed의 기존 ID/receipt는 이번 literal route를 선택하거나 새 HTTP 증거를 만들지 않습니다. header maps, option의 retained config와 raw JSON은 독립 소유합니다. JSON create의 Content-Type은 application/json으로 소유하며 잘못된 입력은 discovery보다 먼저 거절합니다. factory가 JSON 값을 받는 `WithRecordListQuery`는 factory 생성 시 snapshot하며 호출 안의 custom options와 Create attrs `MarshalJSON`은 source 캡처 후 처리합니다.

Location은 공통 `CloudLocation.ForResource`를 재사용합니다. leaf 기본은 기록된 token project와 실제 row project_id를 사용하며 인증/프로젝트 조회를 추가하지 않습니다. Connection의 cloud/region/configured name은 leaf에서 추정하지 않고, `WithRecordReadLocation`, `WithRecordCreateLocation`, `WithRecordListLocation`으로 소유한 연결 정보를 지정할 수 있습니다. truthy foreign project는 원래 JSON ID를 유지하며 현재 scope의 이름을 제거합니다. HTTP `location`은 선언된 computed view에 승격하지 않습니다.

## 버전·목록 query·paging

Source `GroupSnapshot._max_microversion`은 **3.29**입니다. 선택한 client default가 있으면 그 값으로 discovery 없이 요청하고, default가 없으면 공통 Cinder discovery로 서버 범위와 ceiling3.29에서 선택합니다. 더 높은 explicit default도 그대로 유지합니다. 명시적 per-call version 옵션은 underlying Resource read의 Go 확장이며 빈 문자열은 이번 요청의 version header를 생략합니다. discovery의 실제 응답은 `Discovery`로 따로 보관하고 group snapshot ACK를 대신하지 않습니다. finite per-call discovery는 Python adapter discovery/cache/session 전체 동등성을 의미하지 않습니다.

`ListRecords`는 iterator를 실행할 때 service를 캡처합니다. Accept는 service에 저장된 기본값보다 Source의 application/json이 우선하며 명시적인 `WithRecordListHeaders` Accept만 이를 바꿉니다. 기본 detail 경로에서 `WithRecordListDetails(false)`는 `/group_snapshots`로 변경합니다. Source query mapping은 limit·marker·offset·sort_dir·sort_key·sort와 `all_projects→all_tenants`입니다. wire alias all_tenants도 받을 수 있고, Go map에서는 canonical all_projects가 우선하며 null도 우선권을 유지합니다. typed AllProjects는 raw scope query와 충돌하면 사전 오류이고, 명시한 false는 `all_tenants=False`를 보냅니다. Requests 형태처럼 null은 생략하고 bool은 True/False, array는 반복 query, object는 key query로 전달합니다. sort와 offset은 서버 값으로 전달하며 임의의 offset 증가나 schema 추정을 추가하지 않습니다. 서버 next link가 offset을 광고하면 공통 pager가 따라가고 초기 sort·limit·all_tenants 같은 값은 보존합니다. Go는 요청마다 marker/offset 한 cursor family만 사용하며 advertised family 전환 시 이전 cursor를 제거하고 offset은 nonnegative uint64로 제한하며 반복되거나 뒤로 가는 link를 실제 page 증거와 함께 거절합니다. Python의 임의 조합 cursor 전달과 다른 명시적 scope 규칙입니다.

여덟 Body descriptor 중 server query가 아닌 값은 Source Resource.list처럼 로컬 필터입니다. 따라서 name·status·group_id·project_id는 목록 서버 query로 보내지 않습니다. unknown query는 버리고, predicate는 Go map의 안정된 정렬 순서로 적용합니다. 기존 shared `cloudsnapshot.MatchLocal`의 recursive dictionary subset·truthiness·bool/number equality를 사용합니다. 숫자는 공통 exact decimal 정책이며 Python JSON float rounding은 재현하지 않습니다. 예를 들어 integer9007199254740992와 raw filter9007199254740993.0은 Go에서 다르고, Python float decode 후에는 같을 수 있습니다. 이 차이는 tests와 review에 명시합니다.

`WithRecordListMaxItems`는 로컬 필터 전에 원본 행 수를 제한합니다. limit이 없거나0이면 cap이 첫 요청의 limit hint가 되고, `WithRecordListPaginated(false)`는 한 페이지만 읽습니다. body/HTTP next link와 explicit limit의 short-page marker를 공통 REST pager로 처리합니다. marker는 retained 원본 행의 scalar ID를 Requests 문자열로 바꾸며 공개 Resource/Wire 변경에 영향받지 않습니다. marker의 JSON 값은 Requests query expansion에서 정확히 하나의 nonempty 값으로 변환되어야 합니다. 따라서 여러 항목/키 또는 빈 array/object는 오류이고, 한 항목/키는 공통 query expansion 결과를 사용합니다. Go의 finite pager는 limit을 하나의 nonnegative Go integer, marker를 하나의 nonblank 문자열로 제한하며 이 입력은 discovery 전에 검증합니다. max_items는 nonnegative Go integer/zero-unbounded, paginated는 bool 옵션으로 제공하여 Python의 임의 control-value truthiness/coercion 전체를 재현하지 않습니다. Go는 continuation의 origin/collection/filter 변경·cycle을 거절하고 Dictionary links도 처리합니다. Source LIST constructor의 cls/connection/microversion/_synchronized collision은 실제 페이지 오류로 반환하며, member GET의 동일한 unknown field는 Wire에 보존합니다.

## 실패·권한·호환성 범위

Source generic raise_from_response대로 HTTP100–399를 받아들이고400–599는 native Gophercloud 오류를 보존합니다. 기본 delete는 끝까지 깨끗한404만 무시하며, false 옵션은404도 실패입니다. Read/Close/context cause·sticky source 변경이나 retry callback 실패와 섞인404는 무시하지 않습니다. clean native retry는 허용하지만 JSON body·owned header·선택 version/absence·fixed route를 바꾸면 원래 native HTTP cause를 남기고 다음 전송을 막습니다. accepted 물리 실패의 record는 선언된 Resource가 nil이며 actual bytes/header/status가 남고, rejected native 증거는 성공/미존재 proof로 꾸미지 않습니다.

이 네 named mapping은 fixed literal-ID 또는 fresh attrs 호출과 소유한 옵션을 제공합니다. find_group_snapshot, admin reset_status/state, legacy consistency-group snapshot, full mutable Resource/subclass/dirty/session/Adapter/cache/base_path·constructor overload는 별도 항목입니다. server 정책과 version이 project_id를 감추면 그 선언 값은 null이며, SDK가 이를 관리자 권한으로 확대하지 않습니다. authenticated cloud acceptance와 backend 완료 확인은 fixture 검증과 별도입니다.

[계약 테스트](groupsnapshots/python_records_test.go)는 shared Gophercloud public-testhelper `internal/testcloud` fixture와 REST/source/discovery/filter/pager를 사용해 literal route·envelopes·untyped/null/zero·dirty id·owned attrs/options·detail/query/local filters·raw cap·scalar marker·default version·clean404·Read/Close/cancel/source 원복·retry ownership을 확인합니다. 별도 HTTP server harness나 자체 pagination/response 엔진을 만들지 않습니다.
