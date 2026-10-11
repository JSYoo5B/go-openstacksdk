# Cinder v3 일반 볼륨 그룹

기준은 Gophercloud **v2.15.0**, openstacksdk **ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe**의 [Group 모델](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group.py)과 [그룹 proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2071-L2250)입니다. `/groups`는 일반 볼륨과 별도의 자원입니다. 새 수동 leaf는 인증된 Cinder client를 공유하며, GroupType의 관리자 CRUD나 GroupSnapshot을 제공하지 않습니다.

| openstacksdk | Go | 기본값 |
|---|---|---|
| `get_group(id)` | `GetRecord(ctx, id)` | literal ID GET, fallback 없음 |
| `get_group(id, list_volume=False)` | `WithRecordGetListVolume(false)` | 생략과 구분하여 `list_volume=False` 전송 |
| `groups()` | `ListRecords(ctx)` | details=True, lazy 전체 페이지 |
| `groups(details=False)` | `WithRecordListDetails(false)` | `/groups` summary route |
| `create_group(**attrs)` | `CreateRecord(ctx, options...)` | known attrs만 전송, 생략/null 구분 |
| `create_group_from_source(**attrs)` | `CreateFromSourceRecord(ctx, options...)` | 두 nullable source ID 필수, name/description=null |
| `update_group(id, **attrs)` | `UpdateRecord(ctx, id, options...)` | 빈 변경은 HTTP 없음 |
| `delete_group(id)` | `DeleteRecord(ctx, id)` | POST action, delete-volumes=false, clean404 무시 |
| `delete_group(id, delete_volumes=True, ignore_missing=False)` | `WithRecordDeleteVolumes(true)`, `WithRecordDeleteIgnoreMissing(false)` | strict 오류, volume 삭제 요청 |

`api := groups.New(service.RawClient())`로 사용할 수 있습니다. interface/builder 구현 없이 `WithRecord…` 함수와 owned 옵션 struct를 사용합니다. 생성된 transport wrapper의 존재를 이 여섯 Python 동작의 완료 근거로 삼지 않습니다.

## 독립 Python main

`main.py`로 저장합니다. 기본은 목록이고 create/clone/update/delete는 해당 작업을 명시한 경우 실행합니다. 기존 group type은 배포 관리자가 미리 준비해야 합니다.

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--microversion", default="3.38")
    parser.add_argument("--action", choices=["list", "get", "create", "clone", "update", "delete"], default="list")
    parser.add_argument("--group", default="")
    parser.add_argument("--name", default="sample")
    parser.add_argument("--group-type", default="")
    parser.add_argument("--volume-types", default="")
    parser.add_argument("--source-group", default="")
    parser.add_argument("--source-snapshot", default="")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud,
        block_storage_default_microversion=args.microversion)
    api = conn.block_storage
    if args.action == "list":
        for value in api.groups():
            print(json.dumps(value.to_dict(), indent=2, ensure_ascii=False))
        return
    if args.action == "get":
        value = api.get_group(args.group, list_volume=True)
    elif args.action == "create":
        value = api.create_group(group_type=args.group_type,
            volume_types=args.volume_types.split(","), name=args.name)
    elif args.action == "clone":
        # Both IDs are required parameters of the pinned classmethod.
        value = api.create_group_from_source(
            group_snapshot_id=args.source_snapshot or None,
            source_group_id=args.source_group or None,
            name=args.name)
    elif args.action == "update":
        value = api.update_group(args.group, name=args.name)
    else:
        api.delete_group(args.group, delete_volumes=False, ignore_missing=False)
        return
    print(json.dumps(value.to_dict(), indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
```

## 독립 Go main

[설치 안내](../../docs/install.md)를 적용해 `main.go`로 저장합니다. `-action list`는 목록, `-action get -group ID`는 단일 조회입니다. 작업의 생성·완료 대기를 자동으로 추가하지 않습니다.

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

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func printGroup(value *resource.RawResource) {
    raw, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(raw))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    version := flag.String("microversion", "3.38", "Cinder selected version")
    action := flag.String("action", "list", "list/get/create/clone/update/delete")
    id := flag.String("group", "", "group ID")
    name := flag.String("name", "sample", "name for create/clone/update")
    groupType := flag.String("group-type", "", "existing group-type name or ID")
    volumeTypes := flag.String("volume-types", "", "comma-separated volume-type IDs")
    sourceGroup := flag.String("source-group", "", "source group ID for clone")
    sourceSnapshot := flag.String("source-snapshot", "", "source group-snapshot ID for clone")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
        sdk.WithMicroversion(sdk.BlockStorage, *version))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    api := groups.New(service.RawClient())
    var value *groups.GroupRecord
    switch *action {
    case "list":
        for row, err := range api.ListRecords(ctx) {
            if err != nil { log.Fatal(err) }
            printGroup(row.Resource)
        }
        return
    case "get":
        value, err = api.GetRecord(ctx, *id, groups.WithRecordGetListVolume(true))
    case "create":
        value, err = api.CreateRecord(ctx,
            groups.WithRecordCreateField("group_type", *groupType),
            groups.WithRecordCreateField("volume_types", strings.Split(*volumeTypes, ",")),
            groups.WithRecordCreateField("name", *name))
    case "clone":
        var snapshot, source any
        if *sourceSnapshot != "" { snapshot = *sourceSnapshot }
        if *sourceGroup != "" { source = *sourceGroup }
        value, err = api.CreateFromSourceRecord(ctx,
            groups.WithRecordSourceGroupSnapshotID(snapshot),
            groups.WithRecordSourceSourceGroupID(source),
            groups.WithRecordSourceName(*name))
    case "update":
        value, err = api.UpdateRecord(ctx, *id, groups.WithRecordUpdateField("name", *name))
    case "delete":
        err = api.DeleteRecord(ctx, *id, groups.WithRecordDeleteIgnoreMissing(false))
        if err != nil { log.Fatal(err) }
        return
    default:
        log.Fatal("unknown action")
    }
    if err != nil { log.Fatal(err) }
    printGroup(value.Resource)
}
```

## 소유 옵션과 Source 응답

`RecordGet/Create/Source/Update/Delete/ListOpts`와 `WithRecord…Opts`는 map·raw JSON·nullable pointer를 복사합니다. 개별 Field/Query 함수도 생성 시 JSON을 복사하고 unknown key는 JSON 직렬화 전에 버립니다. 알려진 attr의 null·false·0·빈 문자열·빈 list·빈 object는 생략과 구분됩니다. create/update는 Source의 dirty body 값을 전송하며 list descriptor 변환 때문에 request 값을 바꾸지 않습니다.

Get의 `ListVolume`은 기본 nil이라 query가 없습니다. false/true를 명시하면 Python Requests처럼 `False`/`True`를 전송합니다. `WithRecordGetField`는 deprecated `get_group(**attrs)`의 known Body attr을 응답 seed로 보관하며 GET query나 body로 보내지 않습니다. 조회 ID를 attr으로 덮어쓰는 것과 Resource/route controls는 사전 오류입니다. `cls`도 Source `new`/`list` classmethod의 이미 bound된 인자이므로 get/create/update/list 입력에서 null·false를 포함한 모든 명시 값을 discovery HTTP 전에 거부합니다. Python deprecation warning은 Go에서 발생시키지 않습니다.

Source Group의 14개 direct Body attr과 inherited id/name을 Resource에 제공합니다. 누락과 명시 null은 null입니다. `replication_targets`, `volumes`, `volume_types`는 Source list descriptor대로 scalar/object를 한 원소 list로 감쌉니다. 다른 Body attr의 JSON 타입과 큰 정수는 그대로 유지합니다. `location`은 이 standalone leaf에 Connection 객체가 없어 null입니다. Group은 별도의 inherited Header descriptor가 없습니다.

Wire는 unknown attr·`self`·원래 숫자·원래 타입을 보관하고, Envelope는 전체 HTTP body입니다. direct GET 응답의 unknown `cls`는 Wire에 보존하고 Resource에서는 제외합니다. 이 응답 번역은 목록의 `existing` constructor 호출과 구분합니다. Resource·Wire·Envelope·Header·Discovery는 서로 독립 복사하며 status는 실제 응답값입니다. member/create/update의 known seed에 응답 known attrs를 병합합니다. server의 explicit null은 seed를 덮어씁니다. accepted empty body·malformed JSON은 Source처럼 seed를 유지하고, JSON null/nonobject/envelope 오류·invalidUTF8는 actual response 증거를 가진 오류입니다.

CreateFromSource는 `/groups/action`에 `create-from-src`를 보냅니다. 두 source ID를 각각 명시해야 하며 불필요한 쪽은 nil/null로 지정합니다. name/description은 기본 null입니다. Source classmethod가 fresh Group을 생성하므로 CreateRecord와 달리 request attrs를 response seed로 남기지 않습니다. 입력 ID의 배타성·group-type 존재·backend 지원은 Cinder가 검증합니다.

Update는 빈 attr·unknown-only·ID-only인 경우 mutation과 discovery HTTP 모두 생략하고 status=0, Wire/Envelope/Discovery가 없는 local seed를 반환합니다. update body의 id는 SDK 소유 literal selector에서 제외합니다. Python Resource 객체의 dirty id를 버리는 동작에 대응하지만, Python에 string ID와 중복 id kwargs를 주면 발생하는 동적 TypeError까지 재현하지 않습니다. 서로 다른 호출에 mutable Resource dirty state를 공유하지 않으므로 동일 attr을 다시 UpdateRecord에 지정하면 다시 전송합니다.

Delete는 `/groups/{id}/action`에 `{"delete":{"delete-volumes":false}}`를 POST합니다. `WithRecordDeleteVolumes(true)`를 명시해야 소속 volume 삭제를 요청합니다. 성공 body는 decode하지 않고, default ignore_missing은 clean404에만 적용합니다. Physical read/close·cancel·source·callback 원인이 함께 있는404 또는 native retry가 최초 admission을 바꾼404는 오류로 남습니다.

## 목록·페이지·버전

details 기본 true는 `/groups/detail`, false는 `/groups`입니다. server mapping은 limit/marker/offset/sort_dir/sort_key/sort와 all_projects→all_tenants입니다. 두 alias가 함께 있으면 canonical all_projects가 우선하며 bool·iterable·null의 query 문자열은 공통 Requests 변환을 따릅니다. Source 문서에 언급된 list_volume/search_opts/detailed는 이 고정 Group의 실제 QueryParameters에 없으므로 ListRecords에서도 버립니다. list_volume은 GetRecord 옵션으로만 사용합니다. name/id/project_id 등 나머지 declared Body attrs는 local filter입니다.

local 비교는 descriptor 변환 이후이며 boolean/number 교차 비교를 포함한 공통 exact-decimal equality를 사용합니다. 숫자는 Python binary64 반올림·underflow를 재현하지 않습니다. 예를 들어 raw JSON `9007199254740992`와 `9007199254740993.0`은 Source JSON float에서는 같지만 Go local filter에서는 다릅니다. 반대로 `9007199254740993`과 `9007199254740993.0`은 Go에서는 같고 Source int/float 비교에서는 다릅니다. `1e-400`은 Go에서 truthy이므로 빈 dict filter와 일치하지만 Source float에서는 0.0으로 underflow하여 불일치합니다. 이 차이는 declared Body attr의 raw Query 회귀로 고정하며 원래 Wire/Envelope 숫자는 바꾸지 않습니다. 재귀 dict subset은 truthy scalar+빈 dict를 일치시키고, nonempty dict가 scalar의 mapping.get을 필요로 하면 actual page 오류를 반환합니다. Go map의 순서가 없으므로 filter key를 정렬하고 첫 불일치에서 멈춥니다. unknown kwargs와 reserved route override를 임의 wire query로 확장하지 않습니다.

기본은 lazy 전체 페이지이며 MaxItems는 local filter 이전 raw row 수를 셉니다. 생략한 limit 또는 0에는 MaxItems hint를 적용합니다. cap·single-page·consumer break는 unconsumed row나 next link 처리 전에 멈춥니다. groups_links·top-level next·HTTP Link·short-page marker fallback·빈 페이지 종료·singleton object를 지원합니다. marker는 retained Wire ID에서 구하고 loop·origin·collection·initial filter 변경은 공통 REST가 차단합니다. dictionary links는 Go 호환 확장입니다. 공통 `MarkerOrOffsetPagination`은 initial marker/offset 또는 cursor 생략에서 시작하여 서버가 나중에 광고한 forward offset도 따릅니다. next가 없고 nonzero limit이 있으면 local filter에서 제외된 마지막 raw row의 ID로 marker fallback하며 이전 offset을 제거합니다. Go pager는 요청당 cursor family 하나를 허용하고, offset이 앞서 관찰한 모든 offset보다 커야 한다는 명시된 제한을 둡니다. list row의 `cls`/connection/microversion/_synchronized constructor 충돌은 값이 null·false여도 실제 accepted page의 status/body/header 증거를 가진 오류입니다.

[고정 Group ceiling은 3.38](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group.py#L65)입니다. explicit selected version 또는 `WithRecord…Microversion`은 상한보다 높아도 유지합니다. default가 없으면 기존 공통 Cinder discovery를 사용하여 server maximum과3.38 중 지원 가능한 값을 선택합니다. discovery 증거는 실제 group response와 분리합니다. Source처럼 list_volume의3.25 minimum이나 group capability를 임의로 올리거나 새로운 support gate를 만들지 않습니다. 낮은 버전·미지원 backend의 실제 HTTP 오류는 Cinder가 결정합니다.

## 권한·재시도·검증 범위

고정 Cinder **27.0.0/2025.2** [정책](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/groups.py)은 get/list를 project reader, create/update를 project member에 허용합니다. [group:delete](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/group_actions.py)는 project member이며, [create_from_src도 group:create를 실제 authorize](https://github.com/openstack/cinder/blob/27.0.0/cinder/group/api.py#L183-L200)합니다. cross-project 접근과 privileged project attrs, GroupType 관리와 reset_status는 별도의 관리자 조건입니다. 이 leaf가 기존 token의 권한을 높이지 않으며 정책 override나 배포 구성은 서버가 정합니다.

Source `raise_from_response`의 status<400 기준에 맞춰 유효한100..399를 받고 모든400..599 physical rejection을 관측합니다. 보통 net/http는 interim1xx를 처리합니다. accepted Read/Close 실패는 partial GroupRecord와 ResponseError를, rejected 실패는 native ErrUnexpectedResponseCode의 actual status/body/header와 원래 read/close/callback 원인을 유지합니다. 실패 응답을 이전 discovery나 이전 page 성공 증거로 대체하지 않습니다. 응답 Close는 한 번입니다.

선택한 Cinder version 두 header 또는 absence, 다른 서비스의 Nova version absence, Authorization/X-Service-Token/Cookie/Host/framing header absence를 native retry/physical attempts에서도 고정합니다. body가 있으면 Content-Type은 선택한 명시 값 또는 application/json으로 소유합니다. owned header의 OmitHeaders, body/decoder/retention 변경은 첫 실제 native 오류와 callback 원인을 보존하고 재전송 전에 차단합니다. 일반 X-Retry 등은 native 정책대로 바뀔 수 있고 X-Auth-Token은 provider의 live token/reauth에 맡깁니다. shared client의 version·route·headers를 변경하지 않습니다.

nil receiver/context/option, invalid identity·known JSON·version·음수 typed control·보호 header는 HTTP 전 오류입니다. Source가 Read에서 바뀌고 Close에서 원복해도 관측한 실패는 유지합니다. Go는 numeric pagination controls를 엄격히 검증하고 Source 임의 dynamic objects/kwargs와 Resource subclass·shared lifecycle·Connection-derived location을 구현하지 않습니다. local record snapshot과 Proxy/session 공유 HTTP cache도 별개입니다. 반복 Get은 HTTP를 다시 수행하며 Source의 zero-HTTP shared cache hit/invalidation을 재현한다고 주장하지 않습니다.

[기능 회귀](groups/python_records_test.go)와 [물리 오류·재시도 회귀](groups/python_policy_test.go)는 기존 internal/testcloud와 공개 Gophercloud testhelper fixture를 재사용합니다. 검증 결과는 root 통합 검증에서 기록하며 이 문서는 실행 전 통과나 실제 Cinder 인증 integration 성공을 주장하지 않습니다. 전체 typed/generic parity 범위는 [서비스 안내](python-parity.md)를 따릅니다.
