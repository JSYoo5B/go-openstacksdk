# 볼륨 backup export

`ExportVolumeBackup`은 Cinder v3 backup의 export 응답을 bytes·header·status로 반환합니다.
Connection은 인증된 cached Cinder v3를 선택하고, 직접 package 함수는 준비된
`*gophercloud.ServiceClient`를 받습니다. 호출자는 backup ID와 context를 전달합니다.
이 helper에는 함수 옵션이 없습니다.

비교 기준은 openstacksdk commit `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의
[실제 cloud `export_volume_backup`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L648)입니다.
native transport 기준은 Gophercloud `v2.15.0`입니다.

| 실제 Python cloud 호출 | Connection | 직접 package 함수 |
|---|---|---|
| `conn.export_volume_backup(backup_id)` | `conn.ExportVolumeBackup(ctx, blockstorage.ExportVolumeBackupRequest{BackupID: backupID})` | `blockstorage.ExportVolumeBackup(ctx, cinder, request)` |

Python의 이 cloud helper는 v3에서 `requests.Response`를 반환합니다.
[deprecated v3 `Proxy.export_record`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1824)가
[`Backup.export`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L184)를 호출하고,
그 메서드는 HTTP 오류를 확인한 뒤 응답을 그대로 돌려줍니다. `response.json()`을 호출하지 않습니다.
Go도 export record의 schema나 base64 값을 해석하지 않고 실제 응답 bytes를 보존합니다.

다음 예제의 `cloudName`은 clouds.yaml의 항목이며, `backupID`는 호출자가 알고 있는 ID입니다.
`ConfiguredExport`는 전체 Connection 경로, `DirectExport`는 이미 준비한 Cinder client 경로를
보여줍니다. `Adopt`는 기존 인증 provider를 재사용하는 대안입니다. 출력과 선택적 JSON 해석은
호출자의 application 코드입니다.

```go
package example

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "time"

    "github.com/gophercloud/gophercloud/v2"
    sdk "gophercloudsdk"
    "gophercloudsdk/blockstorage"
    "gophercloudsdk/resource"
)

func Adopt(provider *gophercloud.ProviderClient) (*sdk.Connection, error) {
    return sdk.FromProvider(provider, sdk.WithRegion("RegionOne"))
}

func Inspect(result *blockstorage.ExportVolumeBackupResult, err error) {
    if result != nil {
        fmt.Println("requested backup ID:", result.BackupID)
        if result.Exported != nil {
            response := result.Exported
            fmt.Println("export HTTP:", response.StatusCode, "bytes:", len(response.Body))
            fmt.Println("content type:", response.Header.Get("Content-Type"))
        }
    }
    if err == nil { return }
    fmt.Println("export error:", err)
    if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
        fmt.Println("parent context interrupted export")
    }
    if errors.Is(err, resource.ErrInvalidOption) {
        fmt.Println("invalid local input or changed request source")
    }
}

func ConfiguredExport(parent context.Context, cloudName, backupID string) (*blockstorage.ExportVolumeBackupResult, error) {
    ctx, cancel := context.WithTimeout(parent, 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil { return nil, err }
    result, err := conn.ExportVolumeBackup(ctx,
        blockstorage.ExportVolumeBackupRequest{BackupID: backupID})
    Inspect(result, err)
    return result, err
}

func DirectExport(ctx context.Context, cinder *gophercloud.ServiceClient, backupID string) (*blockstorage.ExportVolumeBackupResult, error) {
    result, err := blockstorage.ExportVolumeBackup(ctx, cinder,
        blockstorage.ExportVolumeBackupRequest{BackupID: backupID})
    Inspect(result, err)
    return result, err
}

// Optional application policy: call after a successful export only when
// the application expects JSON. An opaque export need not contain JSON.
func ApplicationJSON(result *blockstorage.ExportVolumeBackupResult) (json.RawMessage, error) {
    if result == nil || result.Exported == nil {
        return nil, fmt.Errorf("no admitted export response")
    }
    var value json.RawMessage
    if err := json.Unmarshal(result.Exported.Body, &value); err != nil {
        return nil, err
    }
    return value, nil
}
```

실제 Python v3 cloud 사용은 다음과 같습니다. 이 예제의 API version은 3입니다.

```python
import openstack

conn = openstack.connect(cloud="devstack", block_storage_api_version="3")
response = conn.export_volume_backup(backup_id)
print(response.status_code, response.headers, response.content)
# JSON이 필요할 때 response.json()을 호출하는 것은 호출자의 선택이다.
```

## ID와 요청

`BackupID`는 이름을 조회하는 입력이 아니라 그대로 사용할 ID입니다. SDK는 member 조회나
목록 검색, Volume·Snapshot·Identity lookup을 하지 않습니다. 유효한 ID로 body와 query 없는
`GET backups/{BackupID}/export_record`를 요청합니다. ID는 비어 있지 않은 UTF-8의 안전한 단일
경로 segment여야 합니다. slash·제어 문자·공백·미리 escape한 ID 등은 local 오류이고,
허용한 literal ID는 SDK가 URL에 escape합니다. 일반 이름처럼 생긴 문자열을 전달해도 이름
검색으로 전환하지 않습니다.

context와 ID를 먼저 검증하므로 Connection의 잘못된 입력은 Cinder getter를 호출하지 않습니다.
유효한 Connection 호출은 cached Cinder v3만 선택합니다. `CurrentLocation`, nullable Backup
모델이나 readiness를 평가하지 않으며, 이미 Connection에 기록한 location의 값도 export에서
사용하지 않습니다. 직접 함수는 선택한 client의 provider·endpoint·resource base·service type·
microversion과 기본 source header를 capture하고 요청을 진행합니다.

선택한 service client의 microversion 값 또는 생략 상태를 유지합니다. export는 force action과 다르게
3.64로 바꾸지 않으며, Backup의 최대 version을 근거로 자동 cap·upgrade·discovery를 수행하지
않습니다. provider의 live token과 native reauthentication·호출자가 설정한 retry는 유지됩니다.
요청의 method·URL·bodylessness와 source 사실은 SDK가 고정하며, native retry의 `OkCodes` 변경도
helper의 최종 원래 status 판정을 바꾸지 않습니다. native RequestOpts header hook의 기존 동작은
유지됩니다. force action처럼 모든 retry header에 별도 3.64 검증을 추가한 API는 아닙니다.
이는 새로운 SDK workflow 재실행이나 lookup 추가를 의미하지 않습니다.

## 응답과 오류

| 결과 | 의미 |
|---|---|
| `result.BackupID` | 요청에 사용한 고정 ID string |
| `result.Exported` | 원래 정책에 admitted된 실제 응답의 owned snapshot |
| `Exported.Body` | 실제 body bytes; JSON이나 UTF-8이어야 할 필요 없음 |
| `Exported.Header` | 실제 응답 header의 owned copy |
| `Exported.StatusCode` | 실제 응답 status |

원래 status 정책은 유한 HTTP 범위 `100..399`입니다. empty·malformed JSON·JSON scalar나
array·invalid UTF-8·binary body도 opaque bytes로 받습니다. 빈 204 역시 정상 admitted 응답입니다.
`backup-record` envelope, service string, backup URL/base64, Backup descriptor를 요구하지 않습니다.
필요한 JSON 해석이나 record validation은 성공 응답을 받은 application이 명시적으로 선택합니다.

`Exported`는 admitted 응답을 실제로 확보한 단계에서 만들며, 그 뒤 Read·Close·source·context
오류가 나더라도 해당 응답 bytes·header·status를 보존합니다. 이때 `err`는 nil이 아니며
`Exported`가 있다는 것만으로 전체 호출 성공을 판단하면 안 됩니다. 실제 응답의 부분 bytes가
있을 수 있습니다. result의 응답과 오류에 담긴 응답 증거는 독립적으로 소유합니다.

context·ID·getter 등 요청 전 실패에는 result가 nil일 수 있습니다. native 또는 원래 정책이
reject한 `>=400` 응답은 `Exported`가 nil이며 원래 native HTTP 오류 증거를 유지합니다.
404도 terminal 오류입니다. 이름 fallback, 정상 부재나 별도 `Absent` 결과로 바꾸지 않습니다.
native retry가 `OkCodes`를 넓혀도 helper의 원래 정책이 reject한 응답을 admitted proof로 만들지
않습니다. rejected 응답 body의 Read·Close 보존은 native의 기존 범위이며, 다른 member lookup이나
삭제 poll에 둔 IO observer를 export까지 적용한 것으로 설명하지 않습니다.

이 helper에는 wait나 SDK timeout 정책이 없습니다. 위 30초는 호출자의 parent context에서
인증·선택·export HTTP를 제한하는 예제입니다. context cancellation/deadline과 custom cause는
오류에 남습니다. 응답을 받은 뒤 별도의 poll·rollback·cleanup을 하지 않습니다.

## Python과 Go의 경계

Python v3 cloud export의 raw response를 Go의 owned bytes·header·status로 대응합니다.
`requests.Response`의 mutable 객체 정체성, request/history/cookies/encoding/elapsed와 편의 메서드,
deprecated warning은 복제하지 않습니다. Source의 같은 이름 v2 경로는 dict를 반환하지만 이
helper는 선택한 Cinder v3 범위입니다.

Python의 `_get_resource`는 ID로 연결된 Backup을 로컬 생성하거나 기존 resource를 업데이트합니다.
이 과정의 location 계산·descriptor 변환·mutable Resource 부수효과가 export 전에 오류를 낼 수
있습니다. Go의 안전한 ID-only helper는 이 construction을 실행하지 않습니다. 임의 Python
resource/dict/object나 permissive `urljoin`의 falsey·container·leading slash 입력도 Go ID domain에
포함하지 않습니다. 이 차이는 export 응답을 normalized Backup으로 바꾼다는 의미가 아닙니다.

[`Proxy.export_backup`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1812)와
[`Backup.export_record`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L173)는
`response.json()`을 반환하는 별도 선언입니다. Gophercloud의 native `backups.Export`도
[JSON result](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/requests.go#L281)와
[typed `BackupRecord`](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/results.go#L201)를
제공하며, `backup_url`의 `[]byte` projection에는 JSON base64 규칙이 적용됩니다. 이 문서의 cloud helper는 해당 parsed/typed API의 지원 검토와 독립적입니다.
전체 SDK 구현 완료를 주장하지 않습니다.

[Backup 조회](volume-backups.md)와 [생성·삭제](volume-backup-mutations.md),
[전체 사용법](../README.md)에서 다른 작업의 정책을 확인할 수 있습니다.
