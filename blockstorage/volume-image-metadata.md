# Volume image metadata 설정·삭제

`SetVolumeImageMetadata`와 `DeleteVolumeImageMetadata`는 명시한 현재 volume ID의 Cinder v3 image metadata action을 실행합니다. SDK가 옵션 기본값·복사·microversion·부분 응답을 처리하며 caller builder는 필요하지 않습니다. 일반 volume `metadata`의 subresource API와는 별도입니다.

기준 Source는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [Proxy set/delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1164-L1197)와 [Volume image metadata actions](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L189-L207)입니다. Go의 기본 전체 삭제는 이 pin의 잘못된 metadata field 순회를 보정합니다.

| 작업 | 실제 Python Proxy | Go Connection / package | 전송·결과 |
|---|---|---|---|
| image metadata 설정 | `conn.block_storage.set_volume_image_metadata(volume_id, **values)` | `SetVolumeImageMetadata(ctx, request, options...)` | `os-set_image_metadata` 1회; `VolumeActionResult` |
| 명시한 key 삭제 | `conn.block_storage.delete_volume_image_metadata(volume_id, keys=[...])` | `DeleteVolumeImageMetadata(ctx, request, WithVolumeImageMetadataDeleteKeys(...))` | 순서·중복을 유지한 key별 POST; `VolumeImageMetadataDeleteResult` |
| 기본 전체 삭제 | `conn.block_storage.delete_volume_image_metadata(volume)` | `DeleteVolumeImageMetadata(ctx, request)` | Go: image metadata GET→정렬한 key별 POST; Source 결함 보정 |
| 명시한 빈 삭제 목록 | `conn.block_storage.delete_volume_image_metadata(volume_id, keys=[])` | `DeleteVolumeImageMetadata(ctx, request, WithVolumeImageMetadataDeleteKeys())` | HTTP·discovery 없음 |

package 함수는 `ctx` 다음에 선택한 `*gophercloud.ServiceClient`를 받습니다. `conn.BlockStorageV3(ctx)`의 `Volumes`는 API **필드**이며 service 메서드는 request 대신 `volumeID string`을 받습니다. `VolumeActionRequest{VolumeID: ...}`는 안전한 현재 route ID이며 name finder, Glance·Nova·Identity lookup을 실행하지 않습니다.

## 한 작업 실행

다음 독립 프로그램은 clouds.yaml의 `CLOUD_NAME`, 현재 `VOLUME_ID`, `IMAGE_METADATA_ACTION=set|delete`를 요구합니다. `IMAGE_METADATA_PATH=connection|direct|service` 기본값은 connection이며, 한 실행에서 작업 하나만 수행합니다. 30초 parent context는 application의 인증·discovery·GET·POST 전체 제한입니다.

set의 `IMAGE_METADATA_JSON`은 JSON object 또는 null이며 생략/null은 metadata `{}`를 보냅니다. `IMAGE_METADATA_KEY`가 존재하면 `IMAGE_METADATA_VALUE`의 빈 값도 string member로 추가합니다. delete의 `IMAGE_METADATA_KEYS_JSON`을 **생략하면 전체 삭제**, `[]` 또는 `null`을 명시하면 **삭제 없음**, string 배열을 주면 그 순서대로 삭제합니다. JSON 파싱은 예제 application의 입력 처리입니다.

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    if err := run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run() error {
    cloudName, volumeID := os.Getenv("CLOUD_NAME"), os.Getenv("VOLUME_ID")
    if cloudName == "" || volumeID == "" { return fmt.Errorf("CLOUD_NAME and VOLUME_ID are required") }
    action, path := os.Getenv("IMAGE_METADATA_ACTION"), os.Getenv("IMAGE_METADATA_PATH")
    if action != "set" && action != "delete" { return fmt.Errorf("IMAGE_METADATA_ACTION must be set or delete") }
    if path == "" { path = "connection" }
    if path != "connection" && path != "direct" && path != "service" { return fmt.Errorf("unknown IMAGE_METADATA_PATH %q", path) }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    input := blockstorage.VolumeActionRequest{VolumeID: volumeID}
    var set blockstorage.VolumeImageMetadataOpts
    var deletion blockstorage.VolumeImageMetadataDeleteOpts
    if action == "set" {
        var values map[string]json.RawMessage
        if text, supplied := os.LookupEnv("IMAGE_METADATA_JSON"); supplied {
            if err := json.Unmarshal([]byte(text), &values); err != nil { return fmt.Errorf("IMAGE_METADATA_JSON: %w", err) }
        }
        options := []blockstorage.VolumeImageMetadataOption{blockstorage.WithVolumeImageMetadataRaw(values)}
        if key, supplied := os.LookupEnv("IMAGE_METADATA_KEY"); supplied {
            options = append(options, blockstorage.WithVolumeImageMetadataValue(key, os.Getenv("IMAGE_METADATA_VALUE")))
        }
        prepared, err := blockstorage.PrepareVolumeImageMetadataOptions(ctx, options...)
        if err != nil { return err }
        set = prepared
    } else {
        var options []blockstorage.VolumeImageMetadataDeleteOption
        if text, supplied := os.LookupEnv("IMAGE_METADATA_KEYS_JSON"); supplied {
            var keys []string
            if err := json.Unmarshal([]byte(text), &keys); err != nil { return fmt.Errorf("IMAGE_METADATA_KEYS_JSON: %w", err) }
            options = append(options, blockstorage.WithVolumeImageMetadataDeleteKeys(keys...))
        }
        prepared, err := blockstorage.PrepareVolumeImageMetadataDeleteOptions(ctx, options...)
        if err != nil { return err }
        deletion = prepared
    }
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil { return err }
    if action == "set" {
        result, err := applySet(ctx, conn, path, input, set)
        inspectSet(result)
        inspectError(err)
        return err
    }
    result, err := applyDelete(ctx, conn, path, input, deletion)
    inspectDelete(result)
    inspectError(err)
    return err
}

func applySet(ctx context.Context, conn *sdk.Connection, path string, input blockstorage.VolumeActionRequest, policy blockstorage.VolumeImageMetadataOpts) (*blockstorage.VolumeActionResult, error) {
    option := blockstorage.WithVolumeImageMetadataOptions(policy)
    if path == "connection" { return conn.SetVolumeImageMetadata(ctx, input, option) }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { return nil, err }
    if path == "direct" { return blockstorage.SetVolumeImageMetadata(ctx, cinder.RawClient(), input, option) }
    return cinder.Volumes.SetVolumeImageMetadata(ctx, input.VolumeID, option)
}

func applyDelete(ctx context.Context, conn *sdk.Connection, path string, input blockstorage.VolumeActionRequest, policy blockstorage.VolumeImageMetadataDeleteOpts) (*blockstorage.VolumeImageMetadataDeleteResult, error) {
    option := blockstorage.WithVolumeImageMetadataDeleteOptions(policy)
    if path == "connection" { return conn.DeleteVolumeImageMetadata(ctx, input, option) }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { return nil, err }
    if path == "direct" { return blockstorage.DeleteVolumeImageMetadata(ctx, cinder.RawClient(), input, option) }
    return cinder.Volumes.DeleteVolumeImageMetadata(ctx, input.VolumeID, option)
}

func inspectSet(result *blockstorage.VolumeActionResult) {
    if result == nil { return }
    fmt.Printf("volume ID: %q; microversion: %q\n", result.VolumeID, result.Microversion)
    for index, page := range result.Discovery {
        if page != nil { fmt.Println("admitted discovery:", index, page.StatusCode, len(page.Body)) }
    }
    if result.Applied != nil { fmt.Printf("admitted set: HTTP %d; opaque body %q\n", result.Applied.StatusCode, result.Applied.Body) }
    fmt.Println("helper acknowledgement completed:", result.Completed)
}

func inspectDelete(result *blockstorage.VolumeImageMetadataDeleteResult) {
    if result == nil { return }
    fmt.Printf("volume ID: %q; microversion: %q\n", result.VolumeID, result.Microversion)
    for index, page := range result.Discovery {
        if page != nil { fmt.Println("admitted discovery:", index, page.StatusCode, len(page.Body)) }
    }
    if result.Observed != nil { fmt.Printf("admitted member GET: HTTP %d; body %q\n", result.Observed.StatusCode, result.Observed.Body) }
    for _, entry := range result.Deleted {
        if entry.Response != nil { fmt.Printf("acknowledged key %q: HTTP %d; opaque body %q\n", entry.Key, entry.Response.StatusCode, entry.Response.Body) }
    }
    if result.Failed != nil {
        fmt.Printf("failed key: %q\n", result.Failed.Key)
        if page := result.Failed.Response; page != nil { fmt.Printf("admitted failed-key response: HTTP %d; opaque body %q\n", page.StatusCode, page.Body) }
    }
    fmt.Println("helper acknowledgement completed:", result.Completed)
}

func inspectError(err error) {
    if err == nil { return }
    var proof *resource.ResponseError
    if errors.As(err, &proof) { fmt.Println("current admitted error response:", proof.StatusCode, len(proof.Body)) }
}
```

일반 호출은 `conn.SetVolumeImageMetadata(ctx, request, blockstorage.WithVolumeImageMetadata(map[string]string{"hw_disk_bus": "scsi"}))`처럼 factory를 바로 전달하면 충분합니다. Prepare는 HTTP나 service selection 없이 originals를 한 번 실행하여 독립 policy를 반환하고, 오류에는 zero policy를 반환합니다. 위 예제는 policy를 전체 교체 factory로 재사용합니다. `blockstorage/v3/volumes` 패키지도 같은 opts/option alias·factory·Prepare 이름을 제공합니다.

Python에서는 다음처럼 실제 Proxy를 호출합니다. 각 줄은 별도 작업입니다.

```python
import openstack

conn = openstack.connect(cloud="devstack")
conn.block_storage.set_volume_image_metadata(volume_id, hw_disk_bus="scsi")
conn.block_storage.set_volume_image_metadata(volume_id, **{})
conn.block_storage.delete_volume_image_metadata(volume_id, keys=["z", "a", "z"])
conn.block_storage.delete_volume_image_metadata(volume_id, keys=[])
# Pinned all-path calls Volume.delete_image_metadata, which loops ordinary metadata.
conn.block_storage.delete_volume_image_metadata(volume)
```

## 기본값·옵션 소유권

set의 `Metadata`는 owned `map[string]json.RawMessage`입니다. options 생략, nil map, 빈 map은 모두 `{"os-set_image_metadata":{"metadata":{}}}` 1회를 보냅니다. `WithVolumeImageMetadata`는 string map을 **교체**하고, `WithVolumeImageMetadataRaw`는 raw map을 **교체**합니다. `WithVolumeImageMetadataValue`만 기존 policy에 string member를 **병합**합니다. 이 factory의 교체·병합은 요청 policy 구성 의미이며 Cinder가 저장된 image metadata 전체를 교체한다는 뜻은 아닙니다.

`WithVolumeImageMetadataOptions`는 완전한 policy를 교체합니다. 실제 최종 key의 UTF-8와 raw value의 UTF-8·완전한 JSON을 검사하며, null·false·0·빈 string·배열·object·큰 숫자도 JSON 값으로 포함할 수 있습니다. 잘못된 초기 member를 뒤 factory에서 교체하면 최종 member를 기준으로 검증합니다. Python datetime/UUID/custom serializer·임의 runtime object 변환은 제공하지 않습니다. raw map은 숫자를 float로 미리 변환하지 않지만, 전송 JSON 직렬화가 raw value의 whitespace·escape 표현까지 그대로 유지한다는 보장은 없습니다.

delete의 `Keys`는 nullable slice pointer입니다. **pointer nil**은 전체 삭제이고, **nonnil pointer**는 explicit keys이며 nil slice도 삭제 없음입니다. `WithVolumeImageMetadataDeleteKeys()`는 zero args여도 nonnil pointer를 만들어 HTTP·discovery 없이 마칩니다. `WithVolumeImageMetadataDeleteAll()`은 이전 explicit keys를 다시 전체 삭제로 바꿉니다. 전체 교체 factory도 pointer presence를 유지합니다. explicit keys는 먼저 모두 UTF-8를 검증하며 빈 값·control·path-like key를 literal body string으로 보냅니다. 순서와 중복을 유지하고 key를 route에 붙이지 않습니다.

factories는 입력 map·slice·pointer·각 RawMessage bytes를 복사합니다. SDK는 original 옵션 slice를 실행 전에 복사하고 callback 경계마다 policy를 복사하므로 retained config나 입력 map의 나중 변경으로 요청 policy를 바꿀 수 없습니다. callback을 native retry마다 다시 실행하지 않습니다. package/service는 selected Source와 안전한 현재 volume ID를 캡처·검사한 뒤 originals를 실행합니다. Connection은 context/provider 검사→originals Prepare 한 번→현재 ID 검사→cached Cinder v3 선택 순서이며 internal engine에 완전한 owned policy를 전달합니다.

## 전체 삭제·부분 결과

Go 전체 삭제는 fixed ID `GET /volumes/{id}`의 canonical `{"volume":{...}}`에서 **`volume_image_metadata`만** 읽습니다. ordinary `metadata`, 반환된 다른 ID·URL·route hint와 unrelated typed descriptor를 사용하지 않습니다. image map이 missing/null/빈 object면 POST 없이 마칩니다. nonobject image map, 잘못된 envelope, malformed 또는 invalid-UTF-8 JSON은 `Observed` proof를 남기고 실패합니다. map의 value는 삭제 대상 선택에 사용하지 않습니다. key 목록을 한 번 캡처하여 Go string 순서로 정렬한 뒤 action을 보내며, 조회 이후 다른 caller가 새로 추가한 key는 이 batch에 포함되지 않습니다. 원자적 clear·read/merge/write·동시성 보장을 제공하지 않습니다.

이 pin의 [전체 삭제 구현](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L196-L200)은 image field 대신 inherited ordinary `self.metadata`를 순회합니다. ordinary metadata가 있으면 잘못된 key 집합을 삭제하며, ID-only Resource에서 ordinary metadata가 absent이면 descriptor default None을 순회하여 TypeError가 날 수 있습니다. 이는 Source 정적 추론이며 Python 실행 결과를 주장하지 않습니다. Go의 fresh image lookup은 이 결함을 의도적으로 보정합니다.

set의 `Applied`, delete의 `Observed`·`Deleted[i].Response`·`Failed.Response`, 각 결과의 `Discovery`는 해당 단계에서 admitted된 실제 Body/Header/StatusCode를 독립적으로 소유합니다. package/service의 `VolumeImageMetadataResponse` alias로 이 응답 형식을 사용할 수 있습니다. 기본 전체 삭제의 member GET은 admitted HTTP100..399라도 canonical JSON 검증이 필요합니다. POST 응답은 opaque이므로 empty/malformed/invalid-UTF-8 body를 그대로 보존합니다. `Deleted`는 성공한 이전 key의 acknowledgement이며 실패한 현재 key는 `Failed`에 분리됩니다. accepted Read/Close·source·custom context 오류에는 Failed.Response가 남을 수 있고, native rejected HTTP에는 해당 Response가 nil일 수 있습니다. 요청 전 batch guard 실패에는 아직 시도한 key가 없어 Failed가 nil일 수 있습니다. 뒤 오류의 현재 proof로 이전 Observed·Discovery·Deleted를 빌리지 않습니다.

첫 오류에서 멈추고 이전 action을 되돌리지 않습니다. `Completed=true`는 선택한 helper의 전송·응답 처리 완료이며 Cinder backend의 최종 image metadata 상태를 재조회한 증거가 아닙니다. explicit none이나 빈 image map은 POST 없이 Completed가 true일 수 있습니다. SDK는 상태 확인·poll·wait·rollback·cleanup·accepted-fault replay를 실행하지 않습니다.

## Source·기존 API 경계

두 helper에는 minimum microversion gate가 없습니다. nonempty selected version은 literal로 사용하며, 없으면 유한 discovery에서 server maximum/optional minimum과 Volume cap3.71을 적용합니다. 선택한 version을 batch GET/POST에 일관되게 사용하고 original client의 microversion·cache를 변경하지 않습니다. explicit none은 discovery도 하지 않습니다. Source는 각 key의 `_action`에서 session microversion을 다시 선택하므로 mutable session의 중간 변경과 Go의 고정 batch는 다른 계약입니다. finite discovery·Keystoneauth cache/canonicalization/예외 순서 전체를 재현하지 않습니다.

Python explicit keys는 arbitrary lazy Iterable이며 string의 문자 순회나 중간 generator 오류도 가능합니다. Go는 owned concrete UTF-8 string slice를 eager 검증하므로 중간 mutation 이전에 입력 오류를 발견합니다. 안전한 현재 ID·literal JSON·independent response 결과는 Python Resource/dict/Munch·dynamic ID·constructor/descriptor/location side effect·mutable requests.Response와 다른 Go 매핑입니다. 모델이나 CurrentLocation을 생성하지 않습니다. Source action은 Resource metadata를 refresh하지 않지만 Proxy의 HTTP cache invalidation은 별도 session 동작입니다.

retry/redirect에서는 fixed method·URL·serialized body·framing·version과 captured Source를 검사하고 provider의 live authentication을 사용할 수 있습니다. native rejected HTTP body IO 전체를 보존하거나 Python dependency HTTP taxonomy 전체가 같다고 주장하지 않습니다.

기존 native `cinder.Volumes.SetImageMetadata(ctx, id, ImageMetadataOpts, ...)`는 `map[string]string`, HTTP200/error-only 계약이며 nil map은 metadata:null을 인코딩할 수 있습니다. `cinder.Volumes.MetadataIn(...)`의 Get/Merge/Replace/DeleteKeys는 ordinary `/volumes/{id}/metadata` 경로입니다. 그 API의 nil key 삭제 PUT `{metadata:{}}`는 새 image action의 전체 삭제와 다릅니다. 새 두 helper는 이 native·ordinary metadata·v2·Resource API의 지원 승격 근거가 아닙니다. Source 정적 비교·예제 컴파일·로컬 HTTP 검증은 인증된 OpenStack/Python runtime이나 전체 SDK 완성을 뜻하지 않습니다.

[일반 volume metadata](v3/volumes/README.md), [직접 Cinder action](volume-actions.md), [block storage 전체 API](README.md)도 참고하세요.
