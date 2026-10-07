# Backup export: raw response와 JSON record

Backup export는 같은 Cinder v3 경로를 사용해도 반환하는 값이 다릅니다.
`ExportRecord`는 실제 응답 bytes·header·status를 돌려주고, `ExportBackup`과
`ExportVolumeBackupRecord`는 그 응답을 한 개의 JSON 값으로 읽습니다.
호출자는 backup ID를 전달하며, 세 함수에는 옵션이나 wait 정책이 없습니다.

비교 기준은 openstacksdk commit `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`와
Gophercloud `v2.15.0`입니다. Python의 deprecated
[`Proxy.export_record`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1824)는
raw response를 반환하지만,
[`Proxy.export_backup`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1812)는
JSON을 반환합니다. 실제 cloud `export_volume_backup`은 앞의 raw 경로입니다.

| 기준 선언과 결과 | Go service API | package / Connection |
|---|---|---|
| Python `conn.block_storage.export_record(id)` → raw `requests.Response` | `cinder.Backups.ExportRecord(ctx, id)` → `*backups.ExportRecordResult` | 기존 `ExportVolumeBackup(ctx, request)` → raw proof |
| Python `conn.block_storage.export_backup(id)` → `response.json()` 값 | `cinder.Backups.ExportBackup(ctx, id)` → `*backups.ExportBackupResult` | `ExportVolumeBackupRecord(ctx, request)` → JSON `Value`와 raw proof |
| Gophercloud native `backups.Export` → typed `BackupRecord` | 기존 `cinder.Backups.Export(ctx, id)` → `*backups.BackupRecord` | native `backup-record` envelope/string service/base64 `[]byte` projection |

아래 예제에서 `cloudName`은 clouds.yaml에 등록한 cloud이고 `backupID`는 호출자가 알고 있는
ID입니다. 함수들은 각각 별개의 사용 경로입니다. 각 export 호출은 독립적인 HTTP 요청이며,
raw 응답을 확인한 뒤 parsed API를 호출하면 export를 두 번 요청하게 됩니다.
한 번의 parsed 호출에서 `Value`와 `Exported`를 함께 확인할 수 있습니다.

```go
package example

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "time"

    "github.com/gophercloud/gophercloud/v2"
    sdk "github.com/JSYoo5B/gophercloudsdk"
    "github.com/JSYoo5B/gophercloudsdk/blockstorage"
    "github.com/JSYoo5B/gophercloudsdk/blockstorage/v3/backups"
    "github.com/JSYoo5B/gophercloudsdk/resource"
)

func Inspect(response *blockstorage.VolumeBackupExportResponse, value json.RawMessage, err error) {
    if response != nil {
        fmt.Println("export HTTP:", response.StatusCode, "bytes:", len(response.Body))
        fmt.Println("content type:", response.Header.Get("Content-Type"))
    }
    if err != nil {
        fmt.Println("export error:", err)
        if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
            fmt.Println("parent context interrupted export")
        }
        var physical *resource.ResponseError
        if errors.As(err, &physical) {
            fmt.Println("accepted error proof:", physical.StatusCode, len(physical.Body))
        }
        return
    }
    if value != nil {
        // Literal null is successful nonnil bytes "null"; nil means no Value.
        fmt.Printf("JSON record: %s\n", value)
    }
}

func ConfiguredRecord(parent context.Context, cloudName, backupID string) (*blockstorage.ExportVolumeBackupRecordResult, error) {
    ctx, cancel := context.WithTimeout(parent, 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil { return nil, err }
    result, err := conn.ExportVolumeBackupRecord(ctx,
        blockstorage.ExportVolumeBackupRecordRequest{BackupID: backupID})
    if result != nil { Inspect(result.Exported, result.Value, err) }
    return result, err
}

func ServiceRecord(ctx context.Context, conn *sdk.Connection, backupID string) (*backups.ExportBackupResult, error) {
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { return nil, err }
    result, err := cinder.Backups.ExportBackup(ctx, backupID)
    if result != nil { Inspect(result.Exported, result.Value, err) }
    return result, err
}

func ServiceRaw(ctx context.Context, conn *sdk.Connection, backupID string) (*backups.ExportRecordResult, error) {
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { return nil, err }
    result, err := cinder.Backups.ExportRecord(ctx, backupID)
    if result != nil { Inspect(result.Exported, nil, err) }
    return result, err
}

func DirectRecord(ctx context.Context, cinder *gophercloud.ServiceClient, backupID string) (*blockstorage.ExportVolumeBackupRecordResult, error) {
    result, err := blockstorage.ExportVolumeBackupRecord(ctx, cinder,
        blockstorage.ExportVolumeBackupRecordRequest{BackupID: backupID})
    if result != nil { Inspect(result.Exported, result.Value, err) }
    return result, err
}

func DirectServiceRecord(ctx context.Context, cinder *gophercloud.ServiceClient, backupID string) (*backups.ExportBackupResult, error) {
    return backups.New(cinder).ExportBackup(ctx, backupID)
}

// Native typed Export is an alternative for the native BackupRecord schema.
func NativeTyped(ctx context.Context, conn *sdk.Connection, backupID string) (*backups.BackupRecord, error) {
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { return nil, err }
    return cinder.Backups.Export(ctx, backupID)
}
```

실제 Python Proxy 사용은 다음과 같습니다. 두 호출의 transport 경로는 같지만 반환 정책은
다릅니다. raw의 deprecated warning 때문에 환경의 warning filter가 오류를 낼 수도 있습니다.

```python
import openstack

conn = openstack.connect(cloud="devstack", block_storage_api_version="3")
value = conn.block_storage.export_backup(backup_id)
print(value)  # response.json()이 읽은 Python 값

response = conn.block_storage.export_record(backup_id)  # deprecated raw 경로
print(response.status_code, response.headers, response.content)
```

## JSON Value

parsed 결과는 `BackupID string`, `Value json.RawMessage`,
`Exported *VolumeBackupExportResponse`를 가집니다. `Value`는 object만을 요구하지 않습니다.
array·string·number·boolean·`null`·빈 object·빈 array가 모두 유효한 전체 JSON 값입니다.
falsey 값을 정상 부재로 바꾸거나 Backup descriptor/model로 변환하지 않습니다.
`backup-record` envelope나 base64 값을 검사하지 않습니다.

Go 표현은 UTF-8의 일반 JSON 한 document를 보존하는 owned `json.RawMessage`입니다.
앞뒤 document 공백만 제외하고 내부 공백·숫자의 표기·escape·중복 member literal을 유지합니다.
숫자를 float64나 `json.Number`로 바꾼 뒤 다시 serialize하지 않습니다.
예를 들어 `{ "x": 1e3, "x": 2 }`의 두 member와 `1e3` 표기는 `Value`에 그대로 남습니다.
이 값은 semantic Python dict가 아닙니다. 호출자가 Go map으로 decode하면 중복 key는
decoder의 정책에 따라 처리되고, 다시 marshal한 표기도 원문과 달라질 수 있습니다.

empty body, malformed JSON, 뒤따르는 두 번째 값이나 쓰레기, invalid UTF-8,
UTF-16/UTF-32 document, BOM, `NaN`·`Infinity` token은 parsed 오류입니다.
원래 raw API에는 JSON 판정이 없으므로 동일한 empty 204나 malformed/binary 응답도
raw status 정책에 admitted될 수 있습니다. HTTP `Content-Type`은 이 parser의 입력 정책을
바꾸지 않습니다. 엄격한 UTF-8 JSON 값과 실제 opaque 응답 bytes를 별도로 제공합니다.

`Value == nil`은 값을 공개하지 않은 상태입니다. 성공한 JSON `null`은 nonnil bytes `null`입니다.
`Value`는 전체 JSON parse와 마지막 captured Source guard가 모두 성공한 뒤에만 공개합니다.
parse가 실패했다고 HTTP를 재전송하거나 raw API를 다시 호출하지 않습니다.

## 요청·선택·오류 증거

ID는 안전하고 비어 있지 않은 literal UTF-8 단일 member segment입니다.
이름처럼 생긴 ID도 조회하지 않으며, slash·공백·제어 문자·미리 escape한 입력 등은 local
오류입니다. SDK가 유효한 literal을 한 번 escape하여 body·query 없는
`GET backups/{ID}/export_record`를 요청합니다. Volume·Snapshot·Identity나 backup member/list를
선행 조회하지 않습니다. lookup, wait, model/location 변환, rollback 정책이 없습니다.

직접 package/service API는 선택한 client의 source를 먼저 capture합니다.
Connection은 context/provider/ID를 검사한 뒤 cached Cinder v3를 선택합니다.
`CurrentLocation`을 평가하지 않으며 export 한 호출에서 source를 다시 capture하지 않습니다.
선택된 microversion 또는 생략 상태를 유지합니다. 3.64 강제 적용, cap, 자동 discovery나
default negotiation을 추가하지 않습니다. 기존 provider live token, native reauthentication와
호출자가 설정한 retry를 사용합니다.

원래 finite HTTP `100..399` 응답을 admitted로 처리합니다. 실제 `Body []byte`, `Header`,
`StatusCode`는 `Exported`의 독립적인 snapshot이며, parse 전에 확보합니다.
accepted body Read·Close·source·context 오류나 JSON 오류가 있어도 이미 확보한 실제 proof는
남습니다. `Exported != nil`만으로 호출 성공을 판정하면 안 됩니다. `err`를 먼저 확인하세요.
부분 bytes가 남을 수도 있으며 result와 오류의 `ResponseError` 증거는 서로 alias하지 않습니다.
parent context cancellation/deadline과 custom cause도 오류에 남습니다.

native 또는 원래 정책이 reject한 `>=400` 응답에는 `Exported`와 `Value`가 없습니다.
404도 terminal 오류이고 정상 부재나 이름 fallback으로 바꾸지 않습니다.
native retry가 `OkCodes`를 넓혀도 원래 정책에 없는 응답을 admitted proof로 만들지 않습니다.
rejected 응답 body의 Read·Close 오류 보존은 native의 기존 제한을 유지합니다.
다른 member lookup/삭제 poll의 response-body observer를 export에 적용한 것은 아닙니다.

SDK는 요청 method·URL·native body/decoder/retention 정책과 captured service facts를 유지합니다.
redirect hook이 같은 GET/URL에 physical Body·ContentLength·TransferEncoding을 추가해도
전송 전에 거부하며, retry에서 값을 복구해도 최초 정책 오류가 남아 재전송을 막습니다.
다른 경로나 origin으로 이동해 token을 전송하는 redirect도 막습니다.
이 guard는 export 범위입니다. 일반 native header hook의 기존 동작을 유지하며,
force action의 별도 microversion header 검증을 뜻하지 않습니다.

## pinned Python과 Go 표현의 경계

Python [`Backup.export_record`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L173)는
status 오류를 확인한 뒤 `response.json()`을 반환합니다. 반환 annotation의 dict cast가
실제 JSON object 검사나 Backup schema validation을 추가하지 않습니다.
deprecated [`Backup.export`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L184)는
같은 경로의 raw response를 반환합니다. Python raw Proxy와 Backup의 warning, warning-filter-as-error
동작은 Go API에 옮기지 않습니다. parsed Proxy 경로에는 이 두 deprecated warning 호출이 없습니다.

두 Python Proxy 모두 `_get_resource`로 연결된 mutable Backup을 생성하거나 갱신합니다.
임의 Resource/dict/Munch, falsey·container ID, permissive `urljoin`, constructor의 descriptor/
location/cache/객체 정체성 부수효과는 Go의 안전한 ID-only domain에 포함하지 않습니다.
Go 결과에는 Python `requests.Response`의 mutable request/history/cookies/encoding/elapsed와
편의 메서드 대신 실제 body/header/status의 owned proof가 있습니다.

openstacksdk pin 자체가 Requests/simplejson/interpreter의 정확한 버전이나 JSON backend를
고정하지는 않습니다. 참고로 Requests `v2.32.5`의
[Response.json](https://github.com/psf/requests/blob/v2.32.5/src/requests/models.py#L947)은
encoding detection과 text fallback을 사용하고,
[compat](https://github.com/psf/requests/blob/v2.32.5/src/requests/compat.py#L57-L64)는
설치한 simplejson 또는 표준 json을 선택합니다.
[guess_json_utf](https://github.com/psf/requests/blob/v2.32.5/src/requests/utils.py#L947)은
BOM과 UTF-16/UTF-32도 다룹니다. 이 참고 버전은 SDK의 dependency pin이 아닙니다.
Go는 위의 UTF-8 literal JSON domain을 사용합니다. Python semantic 숫자의 float rounding,
중복 key 처리, string/surrogate 표현, backend의 nonfinite 정책 및 interpreter digit/depth 제한을
그대로 복제하지 않습니다. 긴 숫자도 float64로 변환하지 않지만 Go JSON parser의 구조적 제한은
적용됩니다.

native [`backups.Export`](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/requests.go#L281)는
기본 GET 200과 native JSON result extraction을 사용하며,
[`BackupRecord`](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/results.go#L201)의
`backup_service string`·`backup_url []byte`에 typed/base64 규칙을 적용합니다.
그 typed API와 v2, Resource 메서드, 다른 backup action의 지원 검토는 각각 독립적입니다.
이 문서는 전체 SDK 또는 실제 cloud runtime 검증의 완료를 뜻하지 않습니다.

[Cloud raw export](volume-backup-export.md), [Backup 조회](volume-backups.md),
[생성·삭제](volume-backup-mutations.md), [전체 사용법](../README.md)에서 다른 정책을 확인할 수 있습니다.
