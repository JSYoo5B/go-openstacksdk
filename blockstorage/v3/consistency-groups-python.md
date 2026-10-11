# Cinder v3 legacy consistency group

기준은 Gophercloud **v2.15.0**, openstacksdk **ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe**의 [ConsistencyGroup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/consistency_group.py)와 [proxy7개](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2322-L2504)입니다. 이 leaf는 `/consistencygroups`를 사용합니다. [일반 volume Group](groups-python.md)의 `/groups`와 별개입니다.

Cinder **27.0.0 / 2025.2**는 이 legacy API를 deprecated 상태로 제공하고 [기본 extension 구성](https://github.com/openstack/cinder/blob/27.0.0/cinder/common/config.py#L59-L61)과 [v3 router](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/v3/router.py#L165-L171)에 등록합니다. 오래된 disabled policy를 이 pin에 적용하지 않습니다. 배포에서 extension을 제거하거나 backend가 consistency 기능을 지원하지 않으면 실제 서버 오류를 반환할 수 있습니다. 새 설계에는 generic Group을 사용할 수 있으며 legacy 호출의 기존 계약도 제공합니다.

| Python | Go | 기본 동작 |
|---|---|---|
| `get_consistency_group(id)` | `GetRecord(ctx, id)` | literal GET, 추가 lookup 없음 |
| `find_consistency_group(name_or_id)` | `FindRecord(ctx, identity)` | GET 우선; clean400/403/404 뒤 전체 상세 LIST |
| `find_consistency_group(..., ignore_missing=False, details=False)` | `WithRecordFindIgnoreMissing(false)`, `WithRecordFindDetails(false)` | strict missing, summary fallback |
| `consistency_groups()` | `ListRecords(ctx)` | details=true, lazy 전체 페이지 |
| `consistency_groups(details=False, **query)` | `WithRecordListDetails(false)`, `WithRecordListQuery(key, value)` | summary와 declared query/local Body filter |
| `create_consistency_group(**attrs)` | `CreateRecord(ctx, options...)` | known dirty attrs, null/생략 구분 |
| `create_consistency_group_from_source(...)` | `CreateFromSourceRecord(ctx, options...)` | 네 필드 기본 null, source ID lookup 없음 |
| `update_consistency_group(id, **attrs)` | `UpdateRecord(ctx, id, options...)` | PUT; 빈/unknown-only/ID-only 변경은 HTTP 없음 |
| `delete_consistency_group(id, force=False)` | `DeleteRecord(ctx, id)` | POST `{id}/delete`, force=false, missing 오류 유지 |
| 고정 Source 삭제 body 재현 | `WithRecordDeletePinnedSourceWire(true)` | 알려진 Source 형식 오류를 명시적으로 재현 |

`consistencygroups.New(service.RawClient())`를 사용합니다. 서비스 facade의 `ConsistencyGroups`도 같은 leaf입니다. 사용자 interface/builder 구현 없이 SDK가 소유한 기본값과 `WithRecord…` 옵션으로 호출합니다.

## 독립 Python main

`main.py`로 저장합니다. 기본 작업은 목록이며 mutation은 해당 action을 명시해야 실행합니다. 생성의 `volume_types`는 stock legacy controller가 요구하는 쉼표 구분 문자열입니다. 빈 이름을 변경할 때는 `--microversion 3.6` 이상을 선택합니다.

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--microversion", default=None)
    parser.add_argument("--action", choices=["list", "get", "find", "create", "clone", "update", "delete"], default="list")
    parser.add_argument("--group", default="")
    parser.add_argument("--name", default="sample")
    parser.add_argument("--volume-types", default="")
    parser.add_argument("--source-group", default="")
    parser.add_argument("--source-snapshot", default="")
    parser.add_argument("--summary", action="store_true")
    parser.add_argument("--strict", action="store_true")
    parser.add_argument("--force", action="store_true")
    args = parser.parse_args()
    policy = {}
    if args.microversion is not None:
        policy["block_storage_default_microversion"] = args.microversion
    api = openstack.connect(cloud=args.cloud, **policy).block_storage
    if args.action == "list":
        for value in api.consistency_groups(details=not args.summary):
            print(json.dumps(value.to_dict(), indent=2, ensure_ascii=False))
        return
    if args.action == "get":
        value = api.get_consistency_group(args.group)
    elif args.action == "find":
        value = api.find_consistency_group(args.group,
            ignore_missing=not args.strict, details=not args.summary)
    elif args.action == "create":
        value = api.create_consistency_group(volume_types=args.volume_types,
            name=args.name)
    elif args.action == "clone":
        value = api.create_consistency_group_from_source(
            consistency_group=args.source_group or None,
            consistency_group_snapshot=args.source_snapshot or None,
            name=args.name)
    elif args.action == "update":
        value = api.update_consistency_group(args.group, name=args.name)
    else:
        # This exact pinned helper sends consistency_group rather than the
        # stock controller's consistencygroup and can receive HTTP400.
        api.delete_consistency_group(args.group, force=args.force)
        return
    print(json.dumps(None if value is None else value.to_dict(),
                     indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
```

## 독립 Go main

[설치 안내](../../docs/install.md)를 적용한 모듈의 `main.go`로 저장합니다. 기본 `-microversion ""`은 연결에서 이미 선택된 값을 유지하며, 이 class의 새 discovery를 추가하지 않습니다. 예: `go run . -cloud dev -action find -group sample -strict`.

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
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroups"
)

func show(value *consistencygroups.ConsistencyGroupRecord) {
    if value == nil { fmt.Println("null"); return }
    raw, err := json.MarshalIndent(value.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(raw))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    version := flag.String("microversion", "", "optional selected Cinder version")
    action := flag.String("action", "list", "list/get/find/create/clone/update/delete")
    id := flag.String("group", "", "literal ID, or exact identity for find")
    name := flag.String("name", "sample", "name for create/clone/update")
    volumeTypes := flag.String("volume-types", "", "comma-separated legacy volume types")
    sourceGroup := flag.String("source-group", "", "source consistency-group ID")
    sourceSnapshot := flag.String("source-snapshot", "", "source consistency-group-snapshot ID")
    summary := flag.Bool("summary", false, "summary list/find")
    strict := flag.Bool("strict", false, "strict find missing")
    force := flag.Bool("force", false, "also request deletion of member volumes")
    sourceWire := flag.Bool("pinned-source-wire", false, "replay known Source delete spelling")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    policy := []sdk.ConnectionOption{sdk.WithCloud(*cloud)}
    if *version != "" { policy = append(policy, sdk.WithMicroversion(sdk.BlockStorage, *version)) }
    conn, err := sdk.Connect(ctx, policy...)
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    api := consistencygroups.New(service.RawClient())
    var value *consistencygroups.ConsistencyGroupRecord
    switch *action {
    case "list":
        for row, err := range api.ListRecords(ctx, consistencygroups.WithRecordListDetails(!*summary)) {
            if err != nil { log.Fatal(err) }
            show(row)
        }
        return
    case "get":
        value, err = api.GetRecord(ctx, *id)
    case "find":
        value, err = api.FindRecord(ctx, *id,
            consistencygroups.WithRecordFindIgnoreMissing(!*strict),
            consistencygroups.WithRecordFindDetails(!*summary))
    case "create":
        value, err = api.CreateRecord(ctx,
            consistencygroups.WithRecordCreateField("volume_types", *volumeTypes),
            consistencygroups.WithRecordCreateField("name", *name))
    case "clone":
        value, err = api.CreateFromSourceRecord(ctx,
            consistencygroups.WithRecordSourceConsistencyGroupID(*sourceGroup),
            consistencygroups.WithRecordSourceConsistencyGroupSnapshotID(*sourceSnapshot),
            consistencygroups.WithRecordSourceName(*name))
    case "update":
        value, err = api.UpdateRecord(ctx, *id, consistencygroups.WithRecordUpdateField("name", *name))
    case "delete":
        err = api.DeleteRecord(ctx, *id, consistencygroups.WithRecordDeleteForce(*force),
            consistencygroups.WithRecordDeletePinnedSourceWire(*sourceWire))
        if err != nil { log.Fatal(err) }
        return
    default:
        log.Fatal("unknown action")
    }
    if err != nil { log.Fatal(err) }
    show(value)
}
```

## 요청·응답 기본값과 Source 차이

`RecordGet/Create/Source/Update/Delete/List/FindOpts`와 개별 함수는 map·raw JSON·pointer를 복사합니다. 모든 옵션 family는 기본/nil Headers를 빈 owned map으로 준비하여 첫 custom callback도 안전하게 값을 추가할 수 있습니다. callback이 보관한 pointer를 HTTP 중에 변경해도 적용한 스냅샷을 바꾸지 않습니다. unknown Field/Query는 JSON 직렬화 전에 버리고 알려진 invalid JSON·UTF-8, nil option/context/receiver, blank/control/dot-segment identity와 Resource/route control은 HTTP 전에 거부합니다. literal ID는 path segment로 escape합니다.

Create/Update는 declared dirty Body attrs만 보냅니다. null·false·0·빈 문자열·빈 list/object와 생략을 구분합니다. `volume_types` list descriptor는 반환 view에만 적용하며 request의 쉼표 구분 문자열을 list로 바꾸지 않습니다. GET은 seed ID를 보관하고 응답의 explicit id/null로 덮어쓸 수 있습니다. Create/Update는 known request attrs를 seed로 쓰지만 CreateFromSource는 Source classmethod의 fresh Resource대로 response attrs만 보관합니다. 네 source 필드의 기본값은 null입니다. 빈 source ID도 proxy의 falsey 입력처럼 null로 바꾸고 name/description의 명시 빈 문자열은 유지합니다. 정확히 한 source ID가 필요한지는 서버가 검증하고 실제400을 보존합니다. Go reference/name/description 옵션은 nullable string으로 제공하며 arbitrary Python object/Resource 입력 overload는 제공하지 않습니다. 이미 얻은 결과의 실제 ID 문자열을 사용할 수 있습니다.

`ConsistencyGroupRecord.Resource`는 availability_zone/created_at/description/name/status/volume_types/add_volumes/remove_volumes와 inherited id의 declared view입니다. 누락/null은 null이며 volume_types는 scalar/object를 한 원소 list로 감쌉니다. connection에서 계산하는 `location`은 standalone leaf에서 null입니다. Wire는 원래 unknown attrs·타입·큰 숫자를 보관하고 Envelope는 전체 실제 HTTP body입니다. Resource/Wire/Envelope/Header는 독립 복사합니다. 별도 inherited Header descriptor는 없습니다. GET body의 unknown `cls`는 Wire에 유지하지만 LIST constructor의 cls/connection/microversion/_synchronized 충돌은 null/false여도 실제 accepted-page 오류입니다.

accepted empty/malformed JSON은 Source처럼 seed를 유지합니다. JSON null/nonobject/wrong envelope와 invalidUTF8는 실제 status/body/header를 가진 오류입니다. Read/Close 실패에는 partial receipt를 제공하며 거부 status는 native ErrUnexpectedResponseCode의 실제 body/header/code와 원래 cause를 유지합니다. Update가 빈/unknown-only/ID-only이면 status0, Wire/Envelope/actual Header가 없는 local seed를 반환합니다. persistent Resource dirty state를 공유하지 않아 동일한 명시 attr을 다시 주면 다시 요청합니다.

[stock v3 update](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/v3/consistencygroups.py#L34-L100)는 **3.6**부터 name/description의 빈 값을 허용합니다. SDK가 이를 위해 기본 버전을 올리거나 null 변경을 성공으로 꾸미지 않습니다. 필요한 버전을 선택하고 server 응답을 확인합니다. Create/clone/delete와 update 응답은 비동기 요청의 응답이며 backend 완료를 의미하지 않습니다.

고정 Source 삭제는 `{"consistency_group":{"force":false}}`를 보내지만 [stock controller](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/contrib/consistencygroups.py#L61-L76)는 **consistencygroup**을 요구합니다. Go 기본값은 `{"consistencygroup":{"force":false}}`로 고친 형식입니다. `WithRecordDeletePinnedSourceWire(true)`는 원래 오류 형식을 재현하고 실제 stock-shaped400을 보존합니다. 원래 형식이 서버에서 성공하는 Source parity라고 주장하지 않습니다. force=true는 member volume 삭제도 요청합니다. 이 named Python delete는 ignore_missing 인자가 없으므로404를 숨기지 않습니다. 성공 body는 decode하지 않습니다.

## 목록·검색·버전·오류

목록 default details=true는 `/consistencygroups/detail`, false는 `/consistencygroups`입니다. declared server mapping은 **limit/marker/offset/sort_dir/sort_key/sort**입니다. 문서에 언급된 all_tenants나 all_projects, unknown kwargs는 이 고정 class의 실제 mapping에 없으므로 버립니다. name/id/description 같은 나머지 declared Body attrs는 local filter이며 wire query가 아닙니다. Requests의 scalar/bool/iterable/null query 변환을 공통 helper로 처리하고 Go pagination numeric control은 엄격히 검증합니다.

Generic `WithRecordListQuery`와 bulk/callback Query는 LIST 전용 named control(session/headers/paginated/max_items/allow_unknown_params/details/jmespath_filters/__conflicting_attrs 등)을 null/false/0 존재만으로도 사전 오류로 거부합니다. 단일 옵션은 caller Marshaler를 호출하기 전에 판정합니다. details/paginated/max_items/microversion/headers는 각각 typed `WithRecordList…` 옵션을 사용합니다. allow_unknown_params override는 제공하지 않고 고정 Source 기본의 unknown discard를 유지합니다. deprecated JMESPath와 동적 충돌 kwargs 주입은 공통 **SDK-J1** 경계로 남으며 조용히 무시해서 구현한 것처럼 취급하지 않습니다. 이 거부는 LIST에만 적용하고 mutation unknown attrs 규칙을 바꾸지 않습니다. key 판정은 Source처럼 case-sensitive라 uppercase unknown은 계속 discard합니다.

local dict filter는 실제 값이 truthy여야 하며 빈 dict filter는 truthy scalar에도 일치합니다. nonempty dict를 scalar에 적용하면 실제 accepted-page 오류입니다. dict의 값은 얕은 equality로 비교하므로 nested empty dict도 동등 비교할 수 있습니다. Go map에는 순서가 없어 filter key를 정렬하고 첫 불일치에서 멈춥니다. 공통 exact-decimal equality/truthiness는 Python binary64 반올림·underflow를 재현하지 않습니다. `9007199254740992`와 `9007199254740993.0`은 Go에서 다르고, `1e-400`은 Go에서 truthy입니다. 원래 Wire 숫자를 바꾸지 않으며 declared attribute 회귀로 차이를 고정합니다.

top-level `next`의 truthiness는 공통 `PythonJSONTruthy`로 Source JSON float 동작을 따릅니다. `1e-400`/`-1e-400`은 binary64에서0이 되어 continuation을 무시하고, `1e-300`이나 overflow `1e400`/`-1e400`은 truthy nonstring이므로 실제 accepted-page status/body/header를 가진 URL 타입 오류를 반환합니다. Find도 뒤 continuation 오류가 있으면 앞 일치를 성공으로 반환하지 않습니다. 이 페이지 판단은 위의 명시된 local-filter exact-decimal 정책과 별도로 유지합니다.

MaxItems는 local filter 이전 raw row 수를 세고 기본은 lazy 전체 페이지입니다. single-page/cap/consumer break는 다음 요청 전에 멈춥니다. 공통 pager로 top-level links/plural consistencygroups_links/next/HTTP Link, singleton object, marker fallback, forward offset과 cursor 전환을 처리합니다. stock [view builder](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/views/consistencygroups.py)는 singular **consistencygroup_links**를 사용하므로 Go는 이 키도 읽는 개선을 제공합니다. 고정 Source Resource._get_next_link는 plural 키와 일반 links/next/HTTP Link를 직접 읽으며 singular 키는 읽지 않습니다. nonzero limit이 있으면 Source도 marker fallback을 수행하지만 singular key 지원 자체와 같지는 않습니다. origin/project/collection/filter는 잠그고 변경·loop·backward offset·동시 marker+offset은 실제 원인 page의 오류로 차단합니다. dictionary-link 허용도 명시 Go 확장입니다.

Find는 literal GET 후 clean400/403/404에서만 전체 LIST로 넘어갑니다. 선언된 name query가 없어 서버의 `?name=...`를 만들지 않습니다. string id 또는 name과 exact identity를 비교하고 같은 ID의 두 행도 duplicate입니다. 마지막 page까지 확인하므로 뒤 실패가 앞 일치를 덮어 성공으로 반환하지 않습니다. 기본 missing은 성공한 전체 목록의 미존재에만 nil/nil이고 strict는 resource.ErrNotFound입니다. LIST403/404/late failure, accepted decode/constructor와 physical/cancel/source/retry 오류는 숨기지 않습니다. duplicate는 AmbiguousError와 실제 duplicate-page ResponseError를 함께 보존합니다.

[상속된 Resource._get_microversion](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1418-L1445)과 이 legacy class의 max=None대로 선택값이 없으면 **새 discovery HTTP를 수행하지 않고 버전 헤더를 생략**합니다. selected version은 상한 없이 유지합니다. 일반 Group의3.38 cap을 여기 적용하지 않습니다. owned typed override는 original client/version/headers를 바꾸지 않습니다. Find Microversion은 GET 전용 Go 확장이며 fallback은 한 captured source의 원래 버전을 사용합니다. Find의 빈 override는 inherit-default인 Go 정책으로, Source fetch의 explicit empty string과 다릅니다. 일반 Headers도 Go 확장입니다. GET은 captured Accept를 유지하고 LIST는 application/json을 default로 소유하며 per-call Accept만 이를 덮어씁니다.

선택된 두 Cinder version header 또는 absence, Nova version absence와 reserved auth/Host/Cookie/framing absence를 재시도에서도 고정합니다. JSON body의 Content-Type은 명시값 또는 application/json을 소유합니다. owned header를 OmitHeaders로 숨기거나 body를 바꾸면 처음 실제 거부 응답과 오류를 보존한 채 다음 HTTP를 차단합니다. 일반 X-Retry와 provider의 live token은 native 정책을 따릅니다. 물리 Read/Close 실패의401은 willing retry/reauth callback0·HTTP1이며 clean401은 동일 owned POST로 token renewal 후 재시도할 수 있습니다. 관찰한 source 변경을 Close에서 원복해도 오류는 남고 취소의 custom cause도 함께 유지합니다.

## 권한과 검증 범위

stock [group policy](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/groups.py)는 get/list/find를 project reader, create/create_from_src/update를 project member로 허용합니다. [delete policy](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/group_actions.py#L58-L68)는 project member입니다. legacy controller가 같은 group API/policy에 위임하므로 이7개는 core-user 범위입니다. cross-project·policy override·기능 지원은 실제 배포 서버가 결정합니다.

37개 회귀 그룹은 [일반 계약](consistencygroups/python_record_contracts_test.go)21개와 [Find 계약](consistencygroups/python_find_contracts_test.go)16개입니다. 기존 internal/testcloud와 공개 Gophercloud testhelper fixture를 재사용합니다. 검증 결과는 root 통합 기록에서 관리하며 이 문서가 실제 backend 인수 테스트나 아직 실행하지 않은 검증의 통과를 주장하지 않습니다.

Source mutable Resource/session/subclass/dirty cache, Connection location, session 공유 응답 cache와 standalone owned record는 다릅니다. 반복 Get/Find는 실제 HTTP를 다시 수행합니다. backend completion을 자동으로 기다리거나 source-defect body를 성공으로 가정하지 않습니다. 전체 SDK 경계는 [서비스 parity 안내](python-parity.md)를 따릅니다.

Native `*gophercloud.ErrUnableToReauthenticate`는 ErrOriginal/ErrReauth를 Unwrap하지 않습니다. 실제401 receipt는 ErrOriginal에서 복사하고 native pointer와 ErrReauth 원인을 유지합니다. errors.As로 wrapper와 실제 ErrOriginal 응답을 각각 확인합니다.
