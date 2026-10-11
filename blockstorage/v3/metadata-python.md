# Volume와 Snapshot metadata: Python Resource 반환에 대응하는 Go 호출

기준은 Gophercloud **v2.15.0**, openstacksdk **ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe**입니다. `Volumes`와 `Snapshots`의 `FetchMetadataRecord`, `GetMetadataRecord`, `SetMetadataRecord`, `DeleteMetadataRecord`는 고정한 Python proxy의 여덟 named metadata 호출을 제공합니다. 기존 `MetadataIn(...).Get/Merge/Replace/DeleteKeys`의 string metadata 결과도 계속 사용할 수 있습니다.

| Python `conn.block_storage` | Go `conn.BlockStorageV3(ctx)`의 accessor | 요청과 결과 |
|---|---|---|
| `fetch_volume_metadata(id)` | `Volumes.FetchMetadataRecord(ctx, id, options...)` | GET, metadata를 갱신한 `VolumeResourceRecord` |
| `get_volume_metadata(id)` | `Volumes.GetMetadataRecord(ctx, id, options...)` | 위 fetch의 deprecated Source alias |
| `set_volume_metadata(id, **metadata)` | `Volumes.SetMetadataRecord(ctx, id, values, options...)` | POST merge, 요청 delta를 캐시한 record |
| `delete_volume_metadata(id, keys=None)` | `Volumes.DeleteMetadataRecord(ctx, id, keys, options...)` | nil이면 PUT clear, 그 외에는 순서대로 DELETE |
| `fetch_snapshot_metadata(id)` | `Snapshots.FetchMetadataRecord(ctx, id, options...)` | GET, metadata를 갱신한 `SnapshotRecord` |
| `get_snapshot_metadata(id)` | `Snapshots.GetMetadataRecord(ctx, id, options...)` | 위 fetch의 deprecated Source alias |
| `set_snapshot_metadata(id, **metadata)` | `Snapshots.SetMetadataRecord(ctx, id, values, options...)` | POST merge, 요청 delta를 캐시한 record |
| `delete_snapshot_metadata(id, keys=None)` | `Snapshots.DeleteMetadataRecord(ctx, id, keys, options...)` | nil이면 PUT clear, 그 외에는 순서대로 DELETE |

Python의 fetch/get/set 반환값은 metadata dict 자체가 아니라 선택한 Volume/Snapshot Resource입니다. Go는 `record.Resource.Body["metadata"]`에서 선언된 값을 읽고, `record.Envelope`, `record.Wire`, `Header`, `StatusCode`에서 실제 metadata 응답을 확인합니다. mutation 응답 본문은 empty·opaque·null·scalar라도 성공 상태이면 캐시 갱신을 막지 않습니다. `Wire`는 본문이 JSON object일 때만 존재하며, 그 필드는 Resource의 상태로 자동 승격되지 않습니다.

다음 두 프로그램은 기본값으로 조회만 실행합니다. 인증 환경변수나 clouds.yaml 설정과 기존 volume 또는 snapshot ID를 제공하세요. metadata 읽기와 변경의 권한은 배포된 Cinder 정책에 따릅니다. 공식 [Cinder API](https://docs.openstack.org/api-ref/block-storage/v3/)와 [정책 목록](https://docs.openstack.org/cinder/latest/configuration/block-storage/policy.html)을 함께 확인할 수 있습니다.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--volume", default="")
parser.add_argument("--snapshot", default="")
parser.add_argument("--action", choices=["fetch", "get", "set", "delete", "clear", "noop"], default="fetch")
parser.add_argument("--metadata", default='{"sdk-example":"python"}')
parser.add_argument("--key", default="sdk-example")
args = parser.parse_args()
if not args.volume and not args.snapshot:
    parser.error("provide --volume or --snapshot")

conn = openstack.connect()
for kind, identity in [("volume", args.volume), ("snapshot", args.snapshot)]:
    if not identity:
        continue
    proxy = conn.block_storage
    if args.action in ("fetch", "get"):
        record = getattr(proxy, f"{args.action}_{kind}_metadata")(identity)
        print(kind, "metadata", record.metadata)
    elif args.action == "set":
        record = getattr(proxy, f"set_{kind}_metadata")(identity, **json.loads(args.metadata))
        print(kind, "cached request delta", record.metadata)
    else:
        keys = None if args.action == "clear" else ([] if args.action == "noop" else [args.key])
        result = getattr(proxy, f"delete_{kind}_metadata")(identity, keys=keys)
        print(kind, "delete return", result)  # Source returns None.
```

Go 예제는 독립된 main 프로그램입니다. `-action set -metadata '{"sdk-example":"go"}'`, `-action delete -key sdk-example`, `-action clear`, `-action noop`으로 동작을 선택합니다.

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func show(kind string, declared *resource.RawResource, envelope json.RawMessage, status int) {
    metadata := "<no declared view>"
    if declared != nil { metadata = string(declared.Body["metadata"]) }
    fmt.Printf("%s metadata=%s status=%d actual_body=%q\n", kind, metadata, status, envelope)
}

func main() {
    volumeID := flag.String("volume", "", "existing volume ID")
    snapshotID := flag.String("snapshot", "", "existing snapshot ID")
    action := flag.String("action", "fetch", "fetch, get, set, delete, clear, or noop")
    metadataJSON := flag.String("metadata", `{"sdk-example":"go"}`, "metadata JSON object")
    key := flag.String("key", "sdk-example", "one metadata key to delete")
    flag.Parse()
    if *volumeID == "" && *snapshotID == "" { log.Fatal("provide -volume or -snapshot") }
    switch *action { case "fetch", "get", "set", "delete", "clear", "noop": default: log.Fatal("unknown action") }
    // RawMessage values preserve JSON null/false/zero and large integers.
    var members map[string]json.RawMessage
    if err := json.Unmarshal([]byte(*metadataJSON), &members); err != nil { log.Fatal(err) }
    values := make(map[string]any, len(members))
    for name, raw := range members { values[name] = raw }
    var keys []string
    if *action == "noop" { keys = []string{} }
    if *action == "delete" { keys = []string{*key} }
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }

    if *volumeID != "" {
        var record *volumes.VolumeResourceRecord
        switch *action {
        case "fetch": record, err = service.Volumes.FetchMetadataRecord(ctx, *volumeID)
        case "get": record, err = service.Volumes.GetMetadataRecord(ctx, *volumeID)
        case "set": record, err = service.Volumes.SetMetadataRecord(ctx, *volumeID, values,
            volumes.WithRecordMetadataHeader("X-Client-Request-Id", "metadata-volume"))
        default:
            var deleted *volumes.MetadataRecordDeletion
            deleted, err = service.Volumes.DeleteMetadataRecord(ctx, *volumeID, keys)
            if deleted != nil {
                record = deleted.Record
                fmt.Printf("volume clear=%t completed_cached_keys=%v accepted_receipts=%d\n", deleted.Cleared, deleted.DeletedKeys, len(deleted.Responses))
            }
        }
        if record != nil { show("volume", record.Resource, record.Envelope, record.StatusCode) }
        if err != nil { log.Fatal(err) }
    }
    if *snapshotID != "" {
        var record *snapshots.SnapshotRecord
        switch *action {
        case "fetch": record, err = service.Snapshots.FetchMetadataRecord(ctx, *snapshotID)
        case "get": record, err = service.Snapshots.GetMetadataRecord(ctx, *snapshotID)
        case "set": record, err = service.Snapshots.SetMetadataRecord(ctx, *snapshotID, values,
            snapshots.WithRecordMetadataHeader("X-Client-Request-Id", "metadata-snapshot"))
        default:
            var deleted *snapshots.MetadataRecordDeletion
            deleted, err = service.Snapshots.DeleteMetadataRecord(ctx, *snapshotID, keys)
            if deleted != nil {
                record = deleted.Record
                fmt.Printf("snapshot clear=%t completed_cached_keys=%v accepted_receipts=%d\n", deleted.Cleared, deleted.DeletedKeys, len(deleted.Responses))
            }
        }
        if record != nil { show("snapshot", record.Resource, record.Envelope, record.StatusCode) }
        if err != nil { log.Fatal(err) }
    }
}
```

## 캐시·입력·반환의 경계

`SetMetadataRecord`는 `map[string]any`를 immutable JSON snapshot으로 만든 뒤 options를 적용합니다. nil map과 빈 map은 모두 `{"metadata":{}}` POST입니다. nested object·array·null·false·zero·`json.Number`·`json.RawMessage`를 임의로 문자열화하지 않습니다. 일반적인 Cinder metadata는 string 값이며, 이 더 넓은 JSON 도메인의 서버 허용 여부는 Cinder가 검증합니다.

Source `set_metadata`는 POST 응답의 merged metadata를 읽지 않고 **요청 delta 자체**를 Resource의 metadata로 저장합니다. 기존 캐시가 `{"old":"cached"}`이고 요청이 `{"new":"value"}`라면 선언된 Go view도 `{"new":"value"}`입니다. 서버가 반환한 `{"old":"server","new":"value"}`는 `Wire`와 `Envelope`에 따로 남습니다. 서버의 최종 전체 값이 필요하면 명시적으로 `FetchMetadataRecord`를 다시 호출하세요.

각 서비스의 `WithRecordMetadataSeed(previous)`는 기존 `Resource`의 알려진 선언 필드만 복사합니다. seed의 ID나 이전 `Wire`·`Envelope`·status는 이번 요청을 선택하거나 실제 응답을 증명하지 못합니다. 필수 method ID가 endpoint와 반환 Resource ID를 결정합니다. GET에서 metadata member가 생략되면 seed의 metadata를 유지하고, 명시적 null은 null로 남습니다. Source dict descriptor처럼 nonnull scalar/list metadata는 선언 view에서 `{}`이며 실제 bytes는 `Wire`에 남습니다. 알려지지 않은 응답 필드나 응답의 다른 id/name은 metadata GET으로 캐시를 갱신하지 않습니다. 두 서비스의 기존 record descriptor 변환을 공유합니다.

Volume은 `WithRecordMetadataLocation(location)`으로 소유한 location을 지정할 수 있습니다. 미지정이면 기존 Volume projection과 기록된 token project 정보를 재사용하고 location 조회 HTTP를 추가하지 않습니다. Snapshot record의 bounded projection은 location을 null로 둡니다. 전체 Python connection location/session 동작은 별도 호환성 항목입니다.

headers는 각 서비스의 `WithRecordMetadataHeader`·`WithRecordMetadataHeaders`로 지정합니다. factory의 map과 custom option의 retained config는 호출 후 별도 소유되므로 다음 option이나 retry가 caller 값을 바꾸어도 요청은 바뀌지 않습니다. 인증·framing·content type·microversion header 변경과 충돌하는 `If-Match`는 사전 오류입니다. nonnil empty keys의 no-op도 header와 seed를 검증합니다. literal ID는 하나의 UTF-8 path component로 escape합니다. blank ID·dot component·제어 문자·invalid UTF-8는 거절하고, ID 내부의 공백·slash·percent는 그대로 하나의 component로 인코딩합니다. 각 metadata key는 비어 있지 않은 UTF-8 component입니다. key의 공백·slash·percent는 인코딩되며 dot component와 제어 문자는 거절합니다.

## 삭제의 순서와 실제 증거

| Go keys | Source 대응 | HTTP와 캐시 |
|---|---|---|
| `nil` | `keys=None` | PUT `metadata:{}`, 성공 후 metadata `{}`, `Cleared=true` |
| `[]string{}` | 빈 iterable | HTTP 없음, `StatusCode=0`, receipts 없음 |
| nonempty slice | iterable | 순서·중복을 유지해 키마다 DELETE, 성공한 캐시 갱신만 `DeletedKeys`에 기록 |

Source는 각 DELETE ACK 뒤 truthy cached metadata에서 `pop(key)`를 호출합니다. 알려진 truthy seed에 해당 key가 없으면 서버가 DELETE를 받아도 Python은 `KeyError`입니다. Go도 `MetadataRecordCacheKeyError`를 돌려주고 그 DELETE의 실제 성공 응답을 `Responses`와 `resource.ResponseError`에 보존합니다. 이 경우 그 key는 `DeletedKeys`에 추가되지 않으며 다음 DELETE는 보내지 않습니다. seed 없는 fresh ID는 기본 metadata가 null이므로 추가 GET 없이 삭제하고 `{}`를 캐시합니다. nil/빈 slice 또는 nested map은 호출 중 caller mutation으로 바뀌지 않습니다.

Python delete proxy는 `None`을 반환합니다. Go는 `MetadataRecordDeletion`으로 마지막 완료된 cached view와 각 실제 accepted 응답을 보존합니다. accepted 응답 처리 중 Read/Close/cancel/source 오류가 생긴 마지막 부분 응답도 `Responses`에 남습니다. rejected HTTP는 native Gophercloud 오류의 status/body/header로 관측되며 `Responses`의 accepted 목록에 섞이지 않습니다. 앞선 성공 ACK는 뒤의 실패가 지우지 않습니다. ACK, `DeletedKeys`, `Cleared`는 그 시점의 요청/캐시 처리 증거이고 server transaction, durable deletion, 복제 완료 또는 이후 GET 결과를 뜻하지 않습니다.

## Source 범위와 검증

고정한 `MetadataMixin`은 `Resource._get_microversion`을 호출하지 않습니다. 이 Go 호출도 이미 선택한 client default를 캡처하고 추가 discovery/Resource GET/HEAD/refresh를 하지 않습니다. client microversion이 있으면 선택한 volume version headers를 유지하고, 없으면 없는 상태를 고정합니다. shared client를 수정하지 않습니다. Source의 generic `raise_from_response`처럼 실제 HTTP 100–399는 성공 범위, 400–599는 native 오류입니다. GET은 성공 JSON object를 읽고, set/delete는 성공 body를 캐시 값으로 해석하지 않습니다.

Source/API 범위 밖인 GET root null/scalar/list는 Go에서 actual response를 가진 decode 오류입니다. Python의 일부 특이한 JSON iterable은 membership 연산을 통과할 수 있지만, metadata endpoint의 object 응답 계약과 동일하다고 세지 않습니다. Go는 deprecated get alias에서 Python warning을 내지 않습니다. mutable Python Resource/subclass/custom session/base path/dirty cache의 전체 동등성을 구현했다는 뜻은 아닙니다. method ID와 소유한 선택 seed에 대한 고정 named 호출을 대응하며 native typed metadata 결과의 기존 계약도 유지합니다.

service source를 custom `MarshalJSON`·option보다 먼저 캡처합니다. route/provider/type/default version 변경, context cause, Read/Close 오류는 복구되더라도 관측된 cause를 유지합니다. 물리 오류가 있는 응답은 cache success·absence 판단이나 재전송 근거로 사용하지 않습니다. 기존 clean native retry는 허용하되 JSON body와 owned headers/선택 version이 바뀌면 원래 HTTP cause를 보존하고 다음 전송을 막습니다. accepted fault의 `Resource`는 조회/set에서 nil이고 actual body/header/status는 그대로 조사할 수 있습니다.

[Volume 테스트](volumes/python_metadata_test.go)와 [Snapshot 테스트](snapshots/python_metadata_test.go)는 각각 여덟 계약 그룹에서 aliases·seed/default·explicit null/zero·delta/cache·opaque mutation ACK·nil/empty/ordered keys·post-ACK cached pop·generic statuses·callback source/cancellation·actual Read/Close·retry/body/version ownership을 확인합니다. 기존 `pythonVolumeAPI`/`pythonSnapshotAPI`, physical fault body와 Gophercloud public testhelper 기반 fixture를 재사용합니다. fixture와 guide 빌드 검증은 인증된 실제 Cinder acceptance와 별도로 기록합니다.
