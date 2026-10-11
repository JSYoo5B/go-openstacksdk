# Cinder v3 Type encryption과 ID/name 검색

기준은 Gophercloud **v2.15.0**, openstacksdk **ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe**입니다. `conn.BlockStorageV3(ctx)`의 `VolumeTypes`와 `QoS`에 추가된 Source-shaped API는 서비스가 반환한 JSON 타입과 HTTP 증거를 보존하고, library-owned concrete option으로 생략·null·zero를 구분합니다. Cinder가 적용하는 조회·encryption 변경 권한은 배포 정책에 따릅니다.

[공식 Cinder 2025.2 policy 문서](https://docs.openstack.org/cinder/2025.2/configuration/block-storage/policy.html)의 기본 정책에서 type GET/list는 `xena_system_admin_or_project_reader`, encryption create/get/update와 QoS GET/list는 `admin_api`입니다. 따라서 이 묶음의 신규 mapping 4개는 **core user 1개(find_type), core admin 3개(create/update_type_encryption, find_qos_spec)**로 구분합니다. 조회라는 이유로 QoS를 user API로 세지 않습니다. 이 정책 근거는 현재 27.0.1.dev25 문서이며 SDK의 Python/Gophercloud source pin이나 실제 배포 policy.yaml을 대체하지 않습니다.

| openstacksdk | Go | 기본 동작 |
|---|---|---|
| `create_type_encryption(volume_type, **attrs)` | `VolumeTypes.CreateEncryptionWithAttributes(ctx, typeID, options...)` | 명시한 Body 속성만 `encryption` envelope로 POST |
| `update_type_encryption(volume_type=type, **attrs)` | `VolumeTypes.UpdateEncryptionFromType(ctx, typeID, options...)` | 현재 encryption GET 후 변경된 속성만 PUT |
| `update_type_encryption(encryption=record, **attrs)` | `VolumeTypes.UpdateEncryptionRecord(ctx, record, options...)` | 받은 record를 복사하고 변경된 속성만 PUT |
| `find_type(name_or_id, ignore_missing=True)` | `VolumeTypes.FindRecord(ctx, nameOrID, options...)` | GET와 fallback 목록 모두 `is_public=none` |
| `find_qos_spec(name_or_id, ignore_missing=True, **query)` | `QoS.FindRecord(ctx, nameOrID, options...)` | GET kwargs와 목록의 서버 query·Body filter를 분리 |

기존 generated `CreateEncryption`·`UpdateEncryption`·`Find`와 native `Resources`는 유지합니다. generated encryption builder는 cipher/key_size/control_location의 기본 값을 넣으므로 Python의 부분 속성과 같은 wire body가 필요할 때 위 메서드를 사용합니다.

```python
volume_type = conn.block_storage.find_type("fast")
qos = conn.block_storage.find_qos_spec("gold", project_id=project_id)

created = conn.block_storage.create_type_encryption(
    volume_type, provider="luks", control_location="front-end")
updated = conn.block_storage.update_type_encryption(
    volume_type=volume_type, key_size=512, cipher=None)
# 이미 받은 TypeEncryption을 쓰면 다시 GET하지 않습니다.
updated = conn.block_storage.update_type_encryption(
    encryption=updated, key_size=0)
```

## 독립 Go main

[설치 안내](../../docs/install.md)를 적용하고 아래를 `main.go`로 저장합니다. `go run . -cloud dev -type fast -qos gold`는 두 리소스를 검색합니다. `-create-encryption` 또는 `-key-size`를 지정한 경우에만 encryption을 변경합니다. `-key-size 0`도 실제 0으로 전송하며 생략과 구분됩니다.

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "strconv"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func printResource(v *resource.RawResource) {
    if v == nil { fmt.Println("null"); return }
    data, err := json.MarshalIndent(v, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    typeName := flag.String("type", "fast", "volume type ID or exact name")
    qosName := flag.String("qos", "", "optional QoS spec ID or exact name; stock policy needs admin")
    project := flag.String("project", "", "optional QoS project filter")
    create := flag.Bool("create-encryption", false, "create encryption on the found type")
    keySize := flag.String("key-size", "", "update encryption key size; 0 is explicit")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    volumeType, err := service.VolumeTypes.FindRecord(ctx, *typeName,
        volumetypes.WithRecordFindIgnoreMissing(false))
    if err != nil { log.Fatal(err) }
    printResource(volumeType.Resource)
    var typeID string
    if err := json.Unmarshal(volumeType.Resource.Body["id"], &typeID); err != nil { log.Fatal(err) }
    if *qosName != "" {
        options := []qos.RecordFindOption{}
        if *project != "" { options = append(options, qos.WithRecordFindQuery("project_id", *project)) }
        specification, err := service.QoS.FindRecord(ctx, *qosName, options...)
        if err != nil { log.Fatal(err) }
        if specification == nil { fmt.Println("QoS: null") } else { printResource(specification.Resource) }
    }
    if *create {
        encryption, err := service.VolumeTypes.CreateEncryptionWithAttributes(ctx, typeID,
            volumetypes.WithEncryptionAttribute("provider", "luks"),
            volumetypes.WithEncryptionAttribute("control_location", "front-end"))
        if err != nil { log.Fatal(err) }
        printResource(encryption.Resource)
    }
    if *keySize != "" {
        size, err := strconv.Atoi(*keySize)
        if err != nil { log.Fatal(err) }
        encryption, err := service.VolumeTypes.UpdateEncryptionFromType(ctx, typeID,
            volumetypes.WithEncryptionAttribute("key_size", size))
        if err != nil { log.Fatal(err) }
        printResource(encryption.Resource)
    }
}
```

## Encryption의 속성과 응답

`WithEncryptionAttribute(key, value)`와 `WithEncryptionAttributeOpts`는 id/name/cipher/control_location/created_at/deleted/deleted_at/encryption_id/key_size/provider/updated_at만 Body로 보냅니다. 미등록 속성은 Source처럼 버립니다. `volume_type_id`는 별도의 URI 인자로 받습니다. map·raw JSON bytes·headers는 옵션 생성 시와 적용 후에 각각 복사하며, 뒤에서 호출자의 값을 바꿔도 요청이 달라지지 않습니다.

Create는 `provider`만 명시하면 `{"encryption":{"provider":"luks"}}`를 보내며, 옵션을 생략해도 `{"encryption":{}}`를 한 번 POST합니다. `nil`은 JSON null, `0`·`false`·빈 문자열은 명시된 값입니다. Cipher·key_size·control_location 기본 값을 만들어 넣지 않습니다. `key_size`와 날짜 등의 untyped Body descriptor도 string/int/time.Time으로 강제 변환하지 않습니다.

`UpdateEncryptionFromType`은 `/types/{typeID}/encryption`을 GET해 flat 또는 `encryption` envelope 응답을 읽습니다. 다음 PUT는 `/types/{typeID}/encryption/{encryptionID}`로 보내며, 전달한 속성 중 기존 값과 달라진 것만 포함합니다. 옵션이 없거나 모든 값이 같으면 GET만 보냅니다. Source와 같이 `False == 0` 등 Python의 recursive bool/number equality를 변경 판정에 적용하고, 같을 때 원래 값의 타입도 유지합니다. 실제 필드가 없던 경우와 null은 구분합니다. `id`는 commit dirty body에서 제외합니다. 명시한 `encryption_id` 변경은 alternate ID가 선택되는 경우 요청 경로에도 반영합니다.

`GetEncryptionRecord`로 받은 record를 `UpdateEncryptionRecord`에 넘기면 GET 없이 update할 수 있습니다. Go API는 입력 record를 복사하며 같은 pointer를 수정하지 않습니다. 변경할 속성은 concrete option으로 전달합니다. Python Resource를 직접 수정한 뒤 `commit()`하는 전체 mutable Resource surface는 공통 SDK 단계에서 추적합니다.

`EncryptionRecord.Resource`는 선언된 descriptor view이며 누락된 Body 값과 `location`은 null입니다. URI parent는 `volume_type_id`로 포함합니다. 실제 id가 없을 때 id는 encryption_id alternate를 읽습니다. `Wire`는 unknown field를 포함한 원래 JSON 객체, `Envelope`는 전체 HTTP 본문이며 `Header`·`StatusCode`는 실제 응답입니다. response 속성은 요청/조회 seed에 덮어쓰고 생략한 속성은 유지합니다. 응답 body가 없거나 JSON syntax가 잘못된 경우 Source의 ValueError 정책처럼 seed view를 유지합니다. 유효한 nonobject JSON은 accepted 응답 증거를 가진 오류입니다.

## GET-first 검색

직접 GET가 성공하면 목록을 호출하지 않습니다. 정상적으로 읽고 닫은 400·403·404만 목록 fallback으로 이어지며, 정확한 ID 또는 name이 일치하는 행을 전체 페이지에서 찾습니다. 기본 missing은 `nil, nil`, `WithRecordFindIgnoreMissing(false)`는 `resource.NotFoundError`입니다. 중복은 `resource.AmbiguousError`이며 앞 페이지에서 찾은 뒤의 후속 페이지 오류도 반환합니다. type와 QoS의 Source query mapping에 name이 없으므로 자동 wire name hint를 넣지 않습니다.

Type 검색은 `/types/{nameOrID}?is_public=none`과 `/types?is_public=none`을 사용합니다. 대문자 `None`으로 변환하지 않습니다. Type의 Source-shaped view는 extra_specs의 dict descriptor와 is_public의 bool descriptor를 적용하며, unknown wire 속성은 `Wire`에 보존합니다. Type proxy에는 없는 추가 Go query/header 옵션은 명시적 확장입니다. `is_public`는 항상 Source의 `none`으로 고정합니다.

QoS의 `WithRecordFindQuery(key, value)`는 GET에 kwargs를 보냅니다. null은 Requests query encoder에서 생략하고 array/object/bool은 공통 encoder의 확장·표기 규칙을 사용합니다. 목록에서는 project_id/limit/marker/offset/sort_dir/sort_key/sort만 서버 query입니다. id/name/consumer/specs는 로컬 Body filter이며, 그 외 kwargs는 무시합니다. dict는 중첩 subset, 다른 값은 완전한 값으로 비교하고 recursive bool/number equality를 적용합니다. QoS는 Source의 unknown Body 허용 정책을 따라 Resource view에도 unknown 속성을 보존합니다. QoS id의 string descriptor는 큰 정수를 float64로 손실 없이 변환하고 `Wire`에는 원래 숫자를 남깁니다.

QoS `max_items`와 `paginated` kwargs는 GET query에도 포함하고 목록의 raw-row cap/첫 페이지 정책으로 사용합니다. cap은 로컬 filter 전의 행 수를 제한하며 양수 cap은 생략되거나 0인 initial limit의 힌트가 됩니다. Go max_items는 음수가 아닌 int, paginated는 bool만 허용합니다. 페이지는 array next link·top-level next·HTTP Link와 명시한 limit의 short-page marker fallback을 지원합니다. `links:{next:"URL"}`는 Go 호환 확장입니다. 다른 origin/path, filter 변경, 서로 충돌하는 continuation, marker 반복은 공통 고정 REST 정책에서 오류입니다.

## 오류와 검증 경계

현재 선택한 서비스의 provider·Endpoint·ResourceBase·Type·microversion을 공통 Source capture/guard로 고정합니다. 옵션 callback, authentication/retry, 실제 response Read/Close에서 바뀐 source와 취소 cause를 보존합니다. clean GET 거부 응답만 FindIdentity fallback 대상이며, read/close/transport/retry/source/cancellation 오류가 섞인 HTTP404를 missing으로 지우지 않습니다. accepted 처리 실패는 실제 status/body/header를 가진 `resource.ResponseError`, native HTTP 상태는 Gophercloud error cause로 관찰합니다. 신규 엔진이나 별도 HTTP fixture를 추가하지 않고 기존 guarded REST·FindIdentity·Requests encoder·Gophercloud public testhelper 기반 `testcloud`를 재사용합니다.

Go는 유효한 UTF-8 문자열 identity segment를 한 번 escape하고, 검색 결과의 id를 nonempty string으로 요구합니다. Python 임의 Resource/Munch 객체와 subclass/cache/session hooks, 자동 microversion 협상과 연결 기반 location은 이 profile의 호출 계약과 별도로 공통 SDK 범위에 남습니다. 숫자 equality는 exact decimal 비교이며 Python float rounding이나 사용자 정의 객체를 흉내 내지 않습니다. 적용하는 microversion은 Connection에서 선택한 값입니다.

회귀는 [encryption 계약](volumetypes/python_encryption_contracts_test.go), [type 검색](volumetypes/python_find_records_test.go), [QoS 검색](qos/python_find_records_test.go)의 20개 그룹입니다. wire body·옵션 snapshot·Source dirty 판정·raw response·GET/list kwargs·전체 검색·missing·중복·pagination cap·물리적 오류를 검증합니다. 인증된 Cinder 실행과 전체 SDK parity 완료는 로컬 fixture 검증과 별도의 단계입니다.
