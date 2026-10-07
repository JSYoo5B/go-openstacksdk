# 볼륨의 서버별 device 조회

`blockstorage.GetVolumeAttachDevice(volume, serverID)`는 전달한 볼륨의 attachment를 순서대로 읽고, `ServerID`가 정확히 일치하는 첫 항목의 device를 반환합니다. HTTP나 이름 조회를 하지 않으며 볼륨 상태를 새로 읽지 않습니다. 이미 받은 `CreateVolumeResult.Ready`, `DeleteVolumeResult.Ready`, attachment workflow의 observation 또는 native Cinder 모델에 사용할 수 있습니다. 호출자가 adapter나 interface 구현을 만들 필요가 없습니다.

타입 함수는 `*blockstorage.AttachVolumeObservation`, `*blockstorage.VolumeInfo`, native Cinder v2/v3의 `*volumes.Volume` 네 타입을 받습니다. `blockstorage.GetVolumeAttachDeviceFields(fields, serverID)`는 `map[string]json.RawMessage`에서 같은 선택을 수행하며, device의 원래 JSON 값을 반환합니다. 두 함수 모두 package 함수로 호출합니다.

## HTTP 없이 실행하는 예제

아래 `main`은 네 가지 모델 타입, 현재 typed 필드와 원래 raw evidence, 빈 문자열·null, precision을 유지하는 raw 값과 local 오류를 보여줍니다.

```go
package main

import (
    "encoding/json"
    "errors"
    "fmt"

    volumesv2 "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v2/volumes"
    volumesv3 "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"

    "github.com/JSYoo5B/gophercloudsdk/blockstorage"
    "github.com/JSYoo5B/gophercloudsdk/resource"
)

func text(value string) *string { return &value }

func device(value *string, err error) string {
    if err != nil { panic(err) }
    if value == nil { return "nil" }
    return fmt.Sprintf("%q", *value)
}

func rawDevice(value json.RawMessage, err error) string {
    if err != nil { panic(err) }
    if value == nil { return "nil" }
    return string(value)
}

func invalid(label string, err error) {
    var operation *resource.OperationError
    if !errors.As(err, &operation) || !errors.Is(err, resource.ErrInvalidOption) {
        panic(fmt.Sprintf("unexpected error: %v", err))
    }
    fmt.Printf("%s %s %s %t\n", label, operation.Operation,
        operation.Resource, errors.Is(err, resource.ErrInvalidOption))
}

func main() {
    owned := &blockstorage.VolumeInfo{Attachments: []*blockstorage.AttachVolumeRecord{
        {ServerID: text("server"), Device: text("")},
        {ServerID: text("server"), Device: text("/dev/vdb")},
    }}
    fmt.Println("owned first", device(blockstorage.GetVolumeAttachDevice(owned, "server")))
    fmt.Println("owned no match", device(blockstorage.GetVolumeAttachDevice(owned, "SERVER")))
    empty := &blockstorage.VolumeInfo{Attachments: []*blockstorage.AttachVolumeRecord{}}
    fmt.Println("owned empty", device(blockstorage.GetVolumeAttachDevice(empty, "server")))
    null := &blockstorage.VolumeInfo{Attachments: []*blockstorage.AttachVolumeRecord{
        {ServerID: text("server"), Device: nil}, // Manual Body:nil means explicit null.
    }}
    fmt.Println("owned null", device(blockstorage.GetVolumeAttachDevice(null, "server")))

    var current blockstorage.AttachVolumeObservation
    if err := json.Unmarshal([]byte(`{"attachments":[{"server_id":"server","device":"/dev/original"}]}`), &current); err != nil {
        panic(err)
    }
    current.Attachments[0].Device = text("/dev/current")
    fmt.Println("typed current", device(blockstorage.GetVolumeAttachDevice(&current, "server")))
    fmt.Println("raw original", rawDevice(blockstorage.GetVolumeAttachDeviceFields(current.Body, "server")))

    nativeV2 := &volumesv2.Volume{Attachments: []volumesv2.Attachment{
        {ServerID: "server", Device: ""},
    }}
    nativeV3 := &volumesv3.Volume{Attachments: []volumesv3.Attachment{
        {ServerID: "server", Device: "/dev/vdc"},
    }}
    fmt.Println("native v2", device(blockstorage.GetVolumeAttachDevice(nativeV2, "server")))
    fmt.Println("native v3", device(blockstorage.GetVolumeAttachDevice(nativeV3, "server")))

    raw := map[string]json.RawMessage{
        "attachments": json.RawMessage(`[{"server_id":"server","device":false}]`),
        "status": json.RawMessage("unrelated invalid JSON"), // This field is not consumed.
    }
    fmt.Println("raw false", rawDevice(blockstorage.GetVolumeAttachDeviceFields(raw, "server")))
    raw["attachments"] = json.RawMessage(`[{"server_id":"server","device":9007199254740993123456789}]`)
    fmt.Println("raw number", rawDevice(blockstorage.GetVolumeAttachDeviceFields(raw, "server")))
    raw["attachments"] = json.RawMessage(`[{"server_id":"server","device":{"path":"/dev/vdb"}}]`)
    fmt.Println("raw object", rawDevice(blockstorage.GetVolumeAttachDeviceFields(raw, "server")))
    raw["attachments"] = json.RawMessage(`[{"server_id":"server","device":null}]`)
    fmt.Println("raw null", rawDevice(blockstorage.GetVolumeAttachDeviceFields(raw, "server")))

    tail := []byte(`{"attachments":[{"server_id":"server","device":"/dev/vdb"},null]}`)
    var fields map[string]json.RawMessage
    if err := json.Unmarshal(tail, &fields); err != nil { panic(err) }
    fmt.Println("raw short circuit", rawDevice(blockstorage.GetVolumeAttachDeviceFields(fields, "server")))
    var whole blockstorage.VolumeInfo
    fmt.Println("whole decoder rejects tail", json.Unmarshal(tail, &whole) != nil)

    _, err := blockstorage.GetVolumeAttachDevice((*blockstorage.VolumeInfo)(nil), "server")
    invalid("typed nil error", err)
    fields["attachments"] = json.RawMessage(`[{"server_id":"server"}]`)
    _, err = blockstorage.GetVolumeAttachDeviceFields(fields, "server")
    invalid("raw missing error", err)
}
```

## openstacksdk와 비교

실제 Python 대응은 cloud helper인 `conn.get_volume_attach_device`입니다. `self`는 이 선언에서 사용되지 않으며, 전달한 `volume['attachments']`만 읽습니다. 이미 조회한 Resource를 전달하거나 아래처럼 plain dict를 전달합니다.

```python
volume = {"attachments": [
    {"server_id": "server", "device": ""},
    {"server_id": "server", "device": "/dev/vdb"},
]}
device = conn.get_volume_attach_device(volume, "server")  # "": first match
```

이 구현의 `typing.cast`는 device를 문자열로 바꾸거나 검사하지 않습니다. 따라서 dict의 device가 `False`, 큰 정수, list 또는 dict라면 Python도 그 값을 그대로 반환합니다. Go typed 함수는 이미 표현 가능한 문자열·null을 반환하고, raw 함수는 모든 nonnull JSON device를 그대로 반환합니다. 이름·공백·대소문자·경로 모양을 정규화하지 않으며 `serverID`가 빈 문자열인 호출도 그대로 비교합니다.

## 반환값과 첫 일치

| 전달한 첫 일치 device | typed 결과 | raw 결과 |
|---|---|---|
| 빈 문자열 | nonnil `*string`이며 값은 `""` | `json.RawMessage`의 `""` |
| null | `nil, nil` | `nil, nil` |
| nonempty 문자열 | 복사한 `*string` | 복사한 JSON 문자열 |
| bool·number·array·object | owned 전체 decoder가 거부하는 타입 | 복사한 원래 JSON 값 |
| canonical `device` 누락 | consumed input 오류 | consumed input 오류 |
| 일치하는 항목 없음 | `nil, nil` | `nil, nil` |

첫 일치 device가 빈 문자열이나 null이어도 즉시 반환합니다. 누락된 device가 첫 일치 항목에 있으면 오류를 반환하며 뒤의 중복 항목을 대체 값으로 찾지 않습니다. null과 일치 없음은 반환값만으로 구별하지 않습니다. device 유무만 보고 실제 서버의 연결 완료를 새로 확인한 것으로 해석하지 않습니다.

## 현재 typed 값과 원래 raw evidence

Owned 모델의 현재 `Attachments`, `ServerID`, `Device`가 typed 함수의 입력입니다. `Metadata.Body`와 record의 `Body`는 원래 응답 evidence를 유지합니다. nonnil typed 포인터는 원래 Body의 누락·이전 값에 우선하며, canonical key가 있는 Body와 nil typed 포인터 조합은 현재 null입니다. helper가 원래 raw 값을 다시 decode해 caller의 수정을 되돌리지 않습니다.

nil 포인터만으로 직접 모델을 만들면 누락과 null을 구별할 수 없습니다. 이 경우 record `Body == nil`은 명시적인 typed null로 취급합니다. `Body`가 nonnil이고 정확한 `server_id` 또는 선택된 `device` key가 없으면 해당 필드를 읽는 순간 오류입니다. nonnil 빈 map과 nil map의 차이가 여기서 중요합니다. `ServerID`가 null인 row는 nonmatch이며 device를 검사하지 않습니다. 방문한 nil row는 오류지만 첫 일치 뒤의 nil row는 읽지 않습니다.

nil volume 또는 nil `Attachments`는 불완전한 입력으로 거부합니다. 명시적인 nonnil 빈 attachment slice는 정상적인 일치 없음입니다. 상위 Body의 `attachments`가 없거나 이전 값이어도 현재 nonnil typed slice를 사용합니다. raw 함수에 `volume.Body`를 직접 전달하면 원래 raw evidence를 읽습니다. 예제의 `/dev/current`와 `/dev/original`은 이 두 입력의 차이를 보여줍니다.

Native v2/v3 모델의 attachment는 `ServerID`와 `Device`를 plain string으로 보관합니다. omission·null·빈 문자열이 native decoder에서 같은 zero string이 될 수 있으므로, matching device는 항상 nonnil 포인터이며 빈 문자열도 그대로 반환합니다. native `ServerID == ""`는 빈 `serverID`와 일치합니다. 이 모델에서 사라진 presence/null 정보는 복원하지 않습니다.

## raw 입력과 전체 decoder의 경계

Raw 함수는 정확한 `attachments`, 방문한 row의 `server_id`, 첫 일치 row의 `device` key만 읽습니다. 잘못된 대소문자의 key는 canonical key를 대신하지 않습니다. `server_id`가 null이나 nonstring JSON이면 문자열 `serverID`와 일치하지 않으며 그 row의 device는 읽지 않습니다. id·status·timestamp·extension 등 다른 필드는 검사하지 않습니다.

Raw `attachments`의 유효한 JSON `[]`, `{}`, `""`는 빈 iterable로서 정상적인 일치 없음입니다. null·bool·number는 오류이며 nonempty object/string은 첫 row가 object가 아니므로 오류입니다. nil map, `attachments` 누락과 key가 있지만 nil RawMessage인 경우도 오류입니다.

Array 전체 JSON은 먼저 올바른 UTF-8과 JSON 문법이어야 합니다. 이 조건을 만족하면 row의 shape와 key는 순회 중 실제 방문할 때만 검사합니다. 예제의 첫 일치 뒤 null row처럼 schema가 맞지 않는 tail은 raw helper가 읽지 않습니다. tail에 JSON 문법 오류나 invalid UTF-8이 있으면 전체 파싱 단계에서 거부됩니다. 기존 owned 모델의 전체 decoder는 모든 row와 알려진 필드 타입을 검사하므로 같은 null tail도 거부합니다. native decoder 역시 먼저 전체 모델을 읽으며 case folding·타입 변환·timestamp 검사에 따른 별도 제약이 있습니다.

Python Resource의 inherited descriptor와 plain mapping도 같지 않습니다. 고정 v2 Volume의 `attachments`는 untyped Body이고, v3 Volume은 `type=list`입니다. v3 descriptor는 nonnull object/string을 singleton list로 변환할 수 있으므로 `{}`·`""`를 그대로 전달한 plain dict와 결과가 다를 수 있습니다. Go raw API는 ordinary JSON plain-mapping 입력을 나타냅니다. Python custom mapping, 사용자 정의 equality/iterable와 Resource descriptor conversion은 별도 모델 기능입니다.

## 오류와 입력 소유권

Malformed consumed input은 `*resource.OperationError`를 반환합니다. `Operation`은 실제 호출한 함수 이름이고 `Resource`는 `"volume"`입니다. `errors.Is(err, resource.ErrInvalidOption)`으로 local 입력 오류를 판별하며 visited row index와 canonical key를 오류 설명에서 확인할 수 있습니다. JSON parsing cause가 있으면 오류 chain에 유지합니다. HTTP response나 존재하지 않는 resource를 나타내는 오류를 만들지 않습니다.

함수는 입력과 원래 Body·header를 수정하지 않습니다. nonnil 반환 포인터와 raw bytes는 복사본이므로 반환값 수정이 다음 조회에 영향을 주지 않습니다. 입력을 다른 goroutine에서 동시에 수정하는 사용은 지원하지 않습니다.

이 문서는 고정된 direct cloud helper의 Go mapping을 설명합니다. Resource 전체의 descriptor·session·cache, 다른 API 선언과 전체 SDK 구현 범위는 각각의 지원 리뷰에서 독립적으로 관리합니다. [서비스 README](README.md)와 [전체 README](../README.md)를 함께 참고하세요.

원본은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [cloud helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L381-L396), [v2 Volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/volume.py#L44), [v3 Volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L54)과 [field conversion](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86-L111)을 확인했습니다. Native 입력은 Gophercloud `v2.15.0`의 [v2 모델](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v2/volumes/results.go#L11-L31), [v3 모델](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/volumes/results.go#L12-L33)을 기준으로 합니다.
