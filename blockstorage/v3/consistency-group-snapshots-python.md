# Cinder legacy consistency group snapshot

이 가이드는 openstacksdk [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2503-L2640)와 Gophercloud **v2.15.0**을 기준으로 합니다. `consistencygroupsnapshots.API`는 legacy `/cgsnapshots` 자원입니다. `/group_snapshots`의 generic group snapshot, `/snapshots`의 volume snapshot과 별개입니다.

| Python named proxy | Go | 기본 동작 |
|---|---|---|
| `get_consistency_group_snapshot(id)` | `GetRecord(ctx, id)` | 한 literal ID의 GET |
| `find_consistency_group_snapshot(identity)` | `FindRecord(ctx, identity)` | `ignore_missing=True`, `details=True` |
| `consistency_group_snapshots(**query)` | `ListRecords(ctx, options...)` | 지연 실행, detail 목록 |
| `create_consistency_group_snapshot(**attrs)` | `CreateRecord(ctx, attrs)` | 알려진 attrs를 새 `cgsnapshot`으로 POST |
| `delete_consistency_group_snapshot(id)` | `DeleteRecord(ctx, id)` | body 없는 DELETE, clean404 무시 |

아래 예제는 각각 독립 실행 프로그램입니다. `OS_*` 환경변수 또는 clouds.yaml의 인증 설정을 사용합니다. create는 서버에 작업을 요청하고 delete는 실제 응답을 받는 것까지 수행합니다. 별도 대기나 backend 완료 검증은 하지 않습니다.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("operation", choices=["get", "find", "list", "create", "delete"])
parser.add_argument("--identity")
parser.add_argument("--group")
parser.add_argument("--name")
parser.add_argument("--attrs", default="{}")
parser.add_argument("--query", default="{}")
parser.add_argument("--strict", action="store_true")
parser.add_argument("--summary", action="store_true")
args = parser.parse_args()
conn = openstack.connect()
if args.operation in {"get", "find", "delete"} and args.identity is None:
    parser.error("--identity is required")
if args.operation == "get":
    value = conn.block_storage.get_consistency_group_snapshot(args.identity)
    print(value.to_dict())
elif args.operation == "find":
    value = conn.block_storage.find_consistency_group_snapshot(
        args.identity, ignore_missing=not args.strict, details=not args.summary
    )
    print(None if value is None else value.to_dict())
elif args.operation == "list":
    for value in conn.block_storage.consistency_group_snapshots(
        details=not args.summary, **json.loads(args.query)
    ):
        print(value.to_dict())
elif args.operation == "create":
    attrs = json.loads(args.attrs)
    if args.group is not None:
        attrs["consistencygroup_id"] = args.group
    if args.name is not None:
        attrs["name"] = args.name
    print(conn.block_storage.create_consistency_group_snapshot(**attrs).to_dict())
else:
    conn.block_storage.delete_consistency_group_snapshot(
        args.identity, ignore_missing=not args.strict
    )
    print("delete returned")
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "strings"
    "time"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots"
)

func main() {
    operation := flag.String("operation", "list", "get, find, list, create or delete")
    identity := flag.String("identity", "", "literal ID, or exact ID/name for find")
    group := flag.String("group", "", "source consistency group ID for create")
    name := flag.String("name", "", "create name, including an explicit empty string")
    attrsJSON := flag.String("attrs", "{}", "create JSON object")
    queryJSON := flag.String("query", "{}", "list JSON query object")
    strict := flag.Bool("strict", false, "report missing find/delete as an error")
    summary := flag.Bool("summary", false, "list summary, also on find fallback")
    flag.Parse()
    supplied := make(map[string]bool)
    flag.Visit(func(f *flag.Flag) { supplied[f.Name] = true })
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    api := consistencygroupsnapshots.New(cinder.RawClient())
    show := func(value *consistencygroupsnapshots.ConsistencyGroupSnapshotRecord, err error) {
        if value != nil {
            fmt.Printf("actual_http=%d actual_body=%q\n", value.StatusCode, value.Envelope)
        }
        if err != nil { log.Fatal(err) }
        if value == nil { fmt.Println("no matching consistency group snapshot"); return }
        raw, err := json.MarshalIndent(value.Resource.Body, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Println(string(raw))
    }
    switch *operation {
    case "get":
        show(api.GetRecord(ctx, *identity))
    case "find":
        show(api.FindRecord(ctx, *identity,
            consistencygroupsnapshots.WithRecordFindIgnoreMissing(!*strict),
            consistencygroupsnapshots.WithRecordFindDetails(!*summary)))
    case "list":
        var query map[string]json.RawMessage
        if err := json.Unmarshal([]byte(*queryJSON), &query); err != nil { log.Fatal(err) }
        for value, err := range api.ListRecords(ctx,
            consistencygroupsnapshots.WithRecordListOpts(consistencygroupsnapshots.RecordListOpts{Query: query}),
            consistencygroupsnapshots.WithRecordListDetails(!*summary)) {
            show(value, err)
        }
    case "create":
        var attrs map[string]any
        decoder := json.NewDecoder(strings.NewReader(*attrsJSON))
        decoder.UseNumber()
        if err := decoder.Decode(&attrs); err != nil { log.Fatal(err) }
        if attrs == nil { attrs = make(map[string]any) }
        if supplied["group"] { attrs["consistencygroup_id"] = *group }
        if supplied["name"] { attrs["name"] = *name }
        show(api.CreateRecord(ctx, attrs))
    case "delete":
        receipt, err := api.DeleteRecord(ctx, *identity,
            consistencygroupsnapshots.WithRecordDeleteIgnoreMissing(!*strict))
        if receipt != nil {
            fmt.Printf("actual_http=%d actual_body=%q ignored_missing=%t\n",
                receipt.StatusCode, receipt.Envelope, receipt.IgnoredMissing)
        }
        if err != nil { log.Fatal(err) }
    default:
        log.Fatal("unknown operation: ", *operation)
    }
}
```

예를 들어 Python 프로그램을 `consistency_snapshots.py`, Go 프로그램을 module 안의 `cmd/consistency-snapshots/main.go`에 저장하면 다음과 같이 실행합니다.

```sh
python consistency_snapshots.py list --query '{"limit": 20, "status": "available"}'
go run ./cmd/consistency-snapshots -operation list -query '{"limit": 20, "status": "available"}'
python consistency_snapshots.py create --group source-consistency-group-id --name nightly
go run ./cmd/consistency-snapshots -operation create -group source-consistency-group-id -name nightly
python consistency_snapshots.py find --identity nightly --strict
go run ./cmd/consistency-snapshots -operation find -identity nightly -strict
```

`ConsistencyGroupSnapshotRecord.Resource`는 Source의 여섯 untyped Body 속성 `id`, `consistencygroup_id`, `created_at`, `description`, `name`, `status`와 계산한 `location`입니다. 누락 속성은 null이며 명시한 false·zero·null·nested JSON은 해당 JSON 값으로 보존합니다. `Wire`는 실제 객체의 unknown 필드까지, `Envelope`는 실제 member 응답 또는 행이 나온 전체 page body, `Header`와 `StatusCode`는 실제 응답을 보존합니다. 각 carrier는 별도 소유 copy입니다. 이 클래스에는 `project_id` Body descriptor가 없으므로 unknown wire project_id는 location을 바꾸지 않습니다. location은 기록한 인증 project 또는 명시한 `WithRecord*Location` 값을 사용합니다. 추가 프로젝트 조회나 cloud·region 이름 추측은 없습니다.

GET은 ID를 한 경로 segment로 escape합니다. slash·percent·내부 공백은 문자 그대로 보존하고, 빈 값·dot segment·control·잘못된 UTF-8은 HTTP 전에 거부합니다. `WithRecordReadSeed`는 선언된 view만 snapshot하고 입력 ID가 경로와 seed ID를 결정합니다. 응답의 명시적 null은 seed를 덮어씁니다. 정상 HTTP의 빈 body 또는 잘못된 JSON은 pinned Source의 JSON ValueError 처리처럼 seed 기반 Resource로 남습니다. 이미 파싱되는 null·배열·scalar·null wrapper 및 잘못된 UTF-8은 실제 응답 근거와 오류를 반환합니다. passive 응답 ID가 null·숫자·다른 ID여도 새 조회나 route 변경을 하지 않습니다.

이 클래스의 [Source 정의](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/consistency_group_snapshot.py#L16-L49)에는 microversion ceiling이 없으며, 상속한 `_max_microversion=None`은 `maximum_supported_microversion`에서 **discovery 없이 None**을 반환합니다. 따라서 이 leaf도 선택한 서비스 버전이 없으면 version header를 추가하거나 discovery를 하지 않습니다. 서비스 버전이 있으면 class ceiling 없이 그대로 사용하고 원래 client는 변경하지 않습니다. 일반 GroupSnapshot의 3.29 ceiling이나 Group의 3.38 ceiling을 적용하지 않습니다. `WithRecordRead/Create/Delete/ListMicroversion`은 명시적 Go 추가 옵션이며, empty string은 해당 요청의 version headers 부재입니다. `WithRecordFindMicroversion`은 underlying Resource.find처럼 GET 전용이고 fallback LIST는 원래 서비스 default로 돌아갑니다. 버전 값은 형식을 검증하지만 서버 지원을 만들어 내거나 독립 discovery로 추측하지 않습니다. 현재 Cinder 27.0.0 전역 API 상한 3.71보다 큰 명시 버전이 실제 배포에서 성공한다는 의미는 아닙니다.

목록 기본 경로는 `/cgsnapshots/detail`이고 `WithRecordListDetails(false)`는 `/cgsnapshots`입니다. server query는 **limit, marker, offset, sort_dir, sort_key, sort**만입니다. 여섯 선언 Body 속성은 로컬 filter, 일반 unknown은 무시됩니다. 이 클래스에는 `all_projects`/`all_tenants` mapping이 없으므로 관리자 목록 옵션으로 전달하지 않습니다. `WithRecordListQuery`는 unknown 값을 marshal하지도 않습니다. 알려진 query의 JSON null은 생략, bool은 `True`/`False`, 배열은 반복 값, 객체는 key 반복으로 변환합니다. 요청 query와 로컬 filter 입력은 소유 copy입니다. 공통 로컬 predicate는 dict subset·truthiness·bool/number 동작을 재사용하되 **exact decimal** 비교를 사용합니다. Python JSON binary64로 반올림되는 `9007199254740993.0` filter와 `9007199254740992` 행은 Go에서 일치하지 않을 수 있습니다. filter 순서는 안정된 Go key 순서입니다.

일반 unknown과 Source control은 구분합니다. raw list query의 `cls`, `self`, `session`, `base_path`, `list_base_path`, `microversion`, `headers`, `__conflicting_attrs`, `jmespath_filters`, `resource_type`, `details`, `connection`, `_synchronized`는 null·false·zero를 포함해 존재만으로 marshal과 HTTP 전에 거부합니다. Headers/Microversion/Details/Paginated/MaxItems는 해당 `WithRecordList*` 소유 옵션을 사용합니다. pinned `Proxy._list`의 truthy 문자열 JMESPath branch는 실제로 도달 가능한 기능이지만 이 Resource iterator의 반환 shape에는 포함되지 않습니다. falsey `jmespath_filters`도 raw control로 명시 거부합니다. JMESPath의 임의 result shape는 SDK-J1, dynamic session/class/conflicting attrs/custom route/control은 SDK-S1의 공통 추적 범위이며 이 목록 mapping으로 완료 처리하지 않습니다.

공통 pager는 body next, dictionary links, RFC HTTP Link와 retained ID marker를 지원합니다. body `next`의 truthiness는 Python JSON처럼 decimal/exponent를 binary64로 판단하므로 `1e-400`·`-1e-400`은 false, 정수는 arbitrary precision의 zero 여부를 유지합니다. falsey next는 link가 없는 것으로 처리하고 별도 Link 또는 limit fallback이 없으면 종료합니다. truthy bool·숫자·객체 등 잘못된 link type은 실제 page 오류로 남습니다. 이 continuation 판단은 로컬 predicate의 exact decimal 정책과 별개입니다. advertised offset은 한 cursor family의 nonnegative uint64 값으로 scope·origin·collection·filter·limit·cycle·forward 진행을 검증합니다. marker와 offset 전환도 기존 query defaults와 로컬 filter를 유지합니다. caller가 반환된 Resource/Wire ID를 변경해도 retained marker는 바뀌지 않습니다. unsafe link와 late decode 오류는 실제 page 근거와 함께 종료합니다. RFC Link의 표준 `url` 지원은 pinned Source의 Requests `uri` 접근보다 개선한 동작입니다. 명시적 offset을 자동 증가시키지 않습니다. pagination 입력 limit은 하나의 nonnegative integer, marker는 하나의 nonblank 값이며 HTTP 전에 검증합니다. `WithRecordListPaginated(false)`는 한 page, `WithRecordListMaxItems(n)`은 filter 전 raw 행 수를 제한합니다. zero cap은 무제한입니다. 이 유한 pager의 domain과 scope 보호는 Python generic pager의 모든 느슨한 입력과 같다고 주장하지 않습니다.

Find는 literal GET을 먼저 수행하며 **physical 오류 없는 member 400·403·404**만 LIST fallback을 허용합니다. policy·constructor·version preparation은 shared identity engine에 들어가기 전에 끝나므로 member absence로 오인하지 않습니다. Source query mapping에 name이 없어 자동 name query/filter를 붙이지 않습니다. 정확한 `id == identity` 또는 `name == identity`로 모든 page를 확인하고, 두 번째 후보는 `resource.ErrAmbiguous`, 완전히 끝난 missing은 기본 `(nil, nil)` 또는 `WithRecordFindIgnoreMissing(false)`의 `resource.ErrNotFound`입니다. 후보 뒤의 오류는 후보로 숨기지 않고 LIST404도 무시하지 않습니다. 논리적 missing·duplicate에는 가짜 HTTP receipt나 억제한 GET 오류를 붙이지 않습니다.

named Python Find는 identity·ignore_missing·details만 받습니다. Go의 Find Headers/Location/GET-only Microversion/Query/Paginated/MaxItems는 범위를 한정한 underlying Resource.find 추가 옵션입니다. `WithRecordFindQuery`의 ordinary unknown `fields` 등은 GET query로만 전달하고 fallback에서는 클래스 mapping에 따라 버립니다. 선언 Body attrs는 GET seed·query와 LIST 로컬 filter로 사용합니다. Resource.find의 special `all_projects`는 bool/null로 제한하고 GET에서 제외하며, 이 클래스의 LIST에서도 mapping이 없어 무시됩니다. constructor/control key, custom route, cache/session/subclass control은 HTTP 전에 거부합니다. Paginated=false 또는 MaxItems를 명시한 Find는 검색 범위를 제한하므로 전체 cloud 유일성의 근거가 아닙니다.

Create는 capture 뒤 알려진 attrs를 immutable JSON으로 만든 후 옵션을 적용하고 `{"cgsnapshot": {...}}`를 POST합니다. Source의 fresh dirty 상태처럼 supplied ID도 body에 남으며 명시한 null·false·zero를 생략하지 않습니다. nil attrs는 empty object입니다. `consistencygroup_id`의 필수 여부·내용·서버 schema는 실제 Cinder가 검증하며 SDK가 prefetch하거나 server400을 가짜 preflight 결과로 대체하지 않습니다. unknown attrs는 버리고 bound constructor `self`/`cls`/`connection`/`_synchronized`의 존재는 값이 null·false·zero여도 거부합니다. Source constructor의 microversion 등 generic control은 이 고정 attrs profile에 포함되지 않으며 소유 옵션을 사용합니다. LIST row의 bound constructor `cls`/`connection`/`microversion`/`_synchronized` 충돌은 실제 page 오류이며, GET의 unknown cls는 Wire에 보존됩니다.

Delete는 추가 GET 없이 body 없는 DELETE를 수행하며 generic HTTP100..399를 acknowledgement로 받습니다. 기본 missing 처리는 오류 없는 native404에만 적용합니다. `DeleteRecordResult`는 실제 ACK 또는 ignored404 receipt를 반환하는 Go 추가 기능입니다. Python 반환은 None입니다. 잘못된 JSON이어도 DELETE ACK body는 그대로 보존하며, 이 receipt 자체가 backend의 즉시 삭제·내구성·존재하지 않음을 추가로 증명하지는 않습니다.

Headers/options/attrs는 capture와 guard 및 소유 copy로 처리합니다. nil header map은 functional option callback이 바로 사용할 수 있게 초기화합니다. LIST는 서비스의 Accept보다 기본 `application/json`을 우선하고 명시한 method Accept만 override합니다. auth·route·framing·version headers는 SDK가 관리하며 native clean retry에서도 고정한 headers/body/version/route와 protected absence를 유지합니다. Read·Close·취소·sticky source 변경 오류는 실제 partial body/status/header 및 원인을 보존하고 native retry callback·fallback·ignore_missing·성공 투영으로 숨기지 않습니다. physical401도 willing retry/reauth callback을 호출하지 않고 HTTP 한 번으로 끝납니다. native `*gophercloud.ErrUnableToReauthenticate` pointer와 `ErrOriginal`/`ErrReauth` public fields를 보존합니다. 이 native type은 두 원인을 Unwrap하지 않으므로 실제401 status/body/header는 `ErrOriginal`에서, physical 원인은 `ErrReauth`에서도 확인합니다. 별도 Go receipt는 오직 실제 `ErrOriginal` native 응답으로부터 copy합니다. 오류 없는401은 기존 live 인증 token을 갱신하고 동일한 body/route/version으로 재요청할 수 있습니다. preflight 오류에는 실제로 없었던 HTTP 근거를 붙이지 않습니다. JSON option factory는 생성할 때 snapshot하며 API 호출 밖에서 실행한 caller Marshaler의 부작용은 API capture 범위 밖입니다.

서버 적용 여부는 [Cinder 27.0.0 cgsnapshots controller와 extension 등록](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/contrib/cgsnapshots.py), [default standard_extensions](https://github.com/openstack/cinder/blob/27.0.0/cinder/common/config.py), [현재 group backend API](https://github.com/openstack/cinder/blob/27.0.0/cinder/group/api.py), [group snapshot reader/member 정책](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/group_snapshots.py)을 확인했습니다. legacy route의 등록·지원 버전 범위와 current backend 위임은 존재하며, 조회·목록은 project reader, 생성·삭제는 project member 정책을 거칩니다. 옛 policy 이름이 없다는 이유로 route가 비활성이라고 판정하지 않습니다. 실제 cloud의 정책 override, driver consistency-group capability와 runtime acceptance는 fixture 검증과 별도입니다. 다른 legacy ConsistencyGroup delete의 Source wrapper spelling 문제는 이 snapshot5 구현에 적용하지 않습니다.

이 mapping은 위 다섯 named API의 fixed route/default/descriptor/error behavior입니다. Python의 전체 mutable Resource·dirty/cache·subclass·session/Adapter·arbitrary routing/constructor overload parity나 live cloud/backend 완료를 뜻하지 않습니다. 테스트는 Gophercloud public `testhelper.SetupHTTP`를 사용하는 공통 `internal/testcloud` fixture, 공통 REST/reader/identity/pager/filter와 최소 receipt/fault injection을 재사용합니다.
