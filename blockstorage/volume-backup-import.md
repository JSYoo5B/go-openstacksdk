# Backup import

`ImportVolumeBackup`은 Cinder v3에 backup service와 export locator를 전달하고, 실제 HTTP 응답과 nullable Backup 값을 반환합니다. 이름이나 ID를 조회하지 않고 완료 대기를 수행하지 않습니다. 요청에는 `BackupService`, `BackupURL` 두 문자열이 있으며 옵션은 없습니다. 빈 문자열도 그대로 전달합니다.

비교 기준은 openstacksdk commit `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 실제 [`Proxy.import_backup`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1802)입니다. Python cloud helper라는 별도 선언을 추가한 것이 아니라 이 Proxy 계약을 Go package·Connection에서도 제공하는 것입니다.

| 호출 경로 | 사용법 | 결과 |
|---|---|---|
| Python v3 Proxy | `conn.block_storage.import_backup(service, url)` | 연결이 없는 새 Backup Resource |
| Go Connection | `conn.ImportVolumeBackup(ctx, request)` | actual proof, raw Backup, nullable24 Value |
| Go package | `blockstorage.ImportVolumeBackup(ctx, cinder, request)` | 같은 library-owned 결과 |
| Go v3 service | `cinder.Backups.ImportBackup(ctx, service, url)` | 같은 결과를 service API에서 제공 |
| 기존 native facade | `cinder.Backups.Import(ctx, backups.ImportOpts{BackupService: service, BackupURL: recordBytes})` | accepted201, ID·Name 문자열 DTO |

## 한 번의 import 요청

아래 프로그램은 한 번 실행할 때 선택한 경로로 import를 한 번 요청합니다. `CLOUD_NAME`은 clouds.yaml에 등록된 cloud이고 `BACKUP_IMPORT_PATH`는 `connection`·`direct`·`service` 중 하나이며 기본값은 `connection`입니다. `BACKUP_SERVICE`와 `BACKUP_URL`은 환경 변수에 정의합니다. 빈 값도 정의된 값으로 전달하며 locator를 URL로 파싱하거나 base64로 변환하지 않습니다. 30초 parent context는 예제 application이 authentication·discovery·import 전체에 적용하는 제한입니다. SDK의 wait 정책은 아닙니다.

예를 들어 `CLOUD_NAME=devstack BACKUP_SERVICE=cinder.backup.drivers.swift.SwiftBackupDriver BACKUP_URL=opaque-record BACKUP_IMPORT_PATH=connection go run ./backup-import-example`로 실행합니다. 프로그램은 SDK를 사용하는 모듈의 별도 example 디렉터리에 저장합니다. 실제 service와 locator는 해당 cloud에서 export한 값을 사용합니다.

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "time"

    sdk "gophercloudsdk"
    "gophercloudsdk/blockstorage"
    "gophercloudsdk/resource"
)

func main() {
    if err := run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run() error {
    cloudName := os.Getenv("CLOUD_NAME")
    if cloudName == "" {
        return fmt.Errorf("CLOUD_NAME is required")
    }
    backupService, servicePresent := os.LookupEnv("BACKUP_SERVICE")
    backupURL, urlPresent := os.LookupEnv("BACKUP_URL")
    if !servicePresent || !urlPresent {
        return fmt.Errorf("BACKUP_SERVICE and BACKUP_URL must be defined; empty values are allowed")
    }
    path := os.Getenv("BACKUP_IMPORT_PATH")
    if path == "" {
        path = "connection"
    }
    switch path {
    case "connection", "direct", "service":
    default:
        return fmt.Errorf("unknown BACKUP_IMPORT_PATH %q", path)
    }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil {
        return err
    }
    input := blockstorage.ImportVolumeBackupRequest{
        BackupService: backupService,
        BackupURL: backupURL,
    }
    var result *blockstorage.ImportVolumeBackupResult
    if path == "connection" {
        result, err = conn.ImportVolumeBackup(ctx, input)
    } else {
        cinder, selectErr := conn.BlockStorageV3(ctx)
        if selectErr != nil {
            return selectErr
        }
        if path == "direct" {
            result, err = blockstorage.ImportVolumeBackup(ctx, cinder.RawClient(), input)
        } else {
            result, err = cinder.Backups.ImportBackup(ctx, backupService, backupURL)
        }
    }
    inspect(result, err)
    return err
}

func inspect(result *blockstorage.ImportVolumeBackupResult, err error) {
    if result != nil {
        fmt.Printf("logical backup ID JSON: %s\n", result.BackupID)
        fmt.Printf("import microversion: %q\n", result.Microversion)
        for i, page := range result.Discovery {
            if page != nil {
                fmt.Println("admitted discovery response:", i, page.StatusCode, len(page.Body))
            }
        }
        if result.Applied != nil {
            fmt.Println("admitted import response:", result.Applied.StatusCode, len(result.Applied.Body))
        }
        if result.Backup != nil {
            actual := result.Backup.Clone()
            fmt.Println("actual selected fields:", len(actual.Body))
            fmt.Printf("actual response ID JSON: %s\n", actual.Body["id"])
        }
    }
    if err != nil {
        fmt.Println("import error:", err)
        if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
            fmt.Println("parent context interrupted import")
        }
        var physical *resource.ResponseError
        if errors.As(err, &physical) {
            fmt.Println("current admitted error response:", physical.StatusCode, len(physical.Body))
        }
        return
    }
    if result != nil && result.Value != nil {
        pretty, formatErr := json.MarshalIndent(result.Value, "", "  ")
        if formatErr == nil {
            fmt.Println("nullable Backup:", string(pretty))
        }
    }
}
```

`Discovery`와 `Applied`는 다른 HTTP 단계의 증거입니다. 오류를 검사하기 전에도 이미 관찰한 증거를 확인할 수 있지만, 증거가 존재한다는 이유로 호출 성공을 판단하지 않습니다. 예제는 오류가 없을 때만 logical `Value`를 출력합니다. 옵션이나 Prepare 함수는 없으며, native bytes를 새 요청으로 자동 변환하는 builder도 없습니다.

## Python 호출과 두 URL 보정

Python의 실제 호출은 다음과 같습니다.

```python
import openstack

conn = openstack.connect(cloud="devstack")
backup = conn.block_storage.import_backup(
    service="cinder.backup.drivers.swift.SwiftBackupDriver",
    url="opaque-record",
)
print(backup.id, backup.to_dict())
```

다만 pinned [`Backup.import_record`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L146)는 caller `url`을 `backups/export_record`로 덮어씁니다. 이 잘못된 값으로 POST를 보내고 body의 `backup_url`에도 같은 값을 넣습니다. 따라서 Python 예제는 실제 원본 API 사용법을 보여주지만 이 pinned 버전의 wire 동작은 import 목적과 다릅니다.

Go는 두 문제를 명시적으로 보정합니다. POST 경로는 query 없는 **`backups/import_record`**로 고정하고, body는 **`{"backup-record":{"backup_service":service,"backup_url":callerURL}}`**로 caller locator를 보존합니다. 이 보정을 원본 Python과 동일한 경로·body라고 설명하지 않습니다.

두 body 값은 모든 유효한 UTF-8 문자열을 허용하며 empty·공백·control·Unicode도 caller 값 그대로 JSON 문자열로 전달합니다. `BackupURL`은 URI일 필요가 없는 opaque locator입니다. service enum, URI scheme, base64 또는 JSON record 유효성을 검사하거나 locator가 가리키는 주소에 별도 HTTP 요청을 보내지 않습니다. 서버가 입력을 판정합니다. Python annotation이 허용하는 동적 null·수치·컨테이너 입력은 Go string 영역과 다릅니다.

## 선택 microversion과 discovery

이미 선택된 Cinder client microversion이 있으면 문자열을 그대로 사용하고 discovery를 생략합니다. `3.64`로 낮추거나 reset/force action처럼 `3.64`를 강제하지 않으며 원본 client 설정도 변경하지 않습니다. 이 선택 단계는 Source의 session default 우선 규칙에 대응하지만, 실제 keystoneauth Session이 수행하는 version 문자열 canonicalization은 재현하지 않습니다. 예를 들어 선택한 숫자 문자열의 prefix·padding 등을 dependency가 정규화하는 동작까지 동일하다고 설명하지 않습니다.

선택 version이 없으면 captured endpoint에서 유한한 후보에 bodyless guarded GET discovery를 수행합니다. 끝의 catalog project와 version 부분만 제거하므로 encoded reverse-proxy 경로를 보존합니다. HTTP200·300으로 받아들인 응답은 flat version object, `version` object, `versions` array 또는 `versions.values` array 형태를 지원합니다.

처음 사용할 수 있는 v3 advertisement를 선택합니다. status는 빈 문자열 또는 CURRENT·SUPPORTED·STABLE·DEPRECATED이며 비교는 대소문자를 구분하지 않습니다. accepted `{}` 또는 사용할 수 있는 v3 행이 없는 응답은 다음 유한 후보로 넘어갑니다. usable v3 행을 찾으면 `max_version`을 우선하고 비어 있으면 `version`을 사용합니다. 그 행의 maximum이 없으면 즉시 version 없이 import하며 뒤 후보에서 새로운 maximum을 찾지 않습니다. 모든 후보에 usable 행이 없어도 version을 생략합니다.

maximum이 있으면 서버 maximum을 먼저 파싱한 뒤 선택적인 minimum을 파싱합니다. Source utility처럼 class maximum3.64와 server maximum 중 작은 tuple을 사용하며, minimum이3.64보다 높으면 version을 생략합니다. maximum이 없을 때는 unused minimum을 파싱하지 않습니다. max/min tuple의 major를3으로 제한하거나 min≤max 조건을 추가하지 않고, tuple 길이를 보존한 비교를 사용합니다. discovery에서 선택된 tuple에 finite major 아래 `latest` component가 남으면 header 문자열을 만들 수 없어 POST 전에 오류가 됩니다. `Microversion`은 이 호출에 선택된 import version이며 빈 문자열은 version 생략입니다. 실패한 단계에서는 실제 POST가 이루어졌음을 의미하지 않습니다.

[`Resource._get_microversion`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1418)의 truthy session default 우선 및 [`maximum_supported_microversion`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L358)의 max/min/cap 계산을 Go의 소유된 정책으로 옮깁니다. Source Adapter·keystoneauth discovery cache와 모든 배포별 HTTP taxonomy를 복제한 것은 아닙니다.

Go 후보 탐색에서 clean native404·405는 다음 후보로 넘어가는 제한적인 fallback입니다. body Read/Close·callback·source·context 오류가 join된404·405나 확장한 native OkCodes로 받은 응답은 clean fallback이 아닙니다. accepted response의 parse·Read·Close 오류, source drift·context interruption 및 그 밖의 native failure는 terminal입니다. `Discovery`에는 각각의 accepted discovery HTTP bytes/header/status를 보존하며 rejected404·405를 accepted proof로 만들지 않습니다. 후보 URL/method·credential routing과 bodyless framing·microversion header omission은 실제 physical 요청에서도 guard의 범위 안에 있습니다.

이 terminal parse 정책은 실제 keystoneauth discovery 일부 경로가 malformed JSON을 DiscoveryFailure로 바꾸고 Source utility가 이를 None으로 처리하는 동작과 다릅니다. Go는 그 accepted physical failure 증거를 유지하는 명시적 mapping입니다. utility에서 max를 min보다 먼저 파싱하는 순서는 유지하지만 keystoneauth 내부 endpoint-data 단계가 min을 먼저 처리하는 모든 예외 순서까지 복제하지 않습니다. finite candidate 형태·허용 JSON shape/status·clean404/405 fallback과 dependency HTTP taxonomy 전체가 동일하다고 주장하지 않습니다.

## 연결이 없는 nullable Backup

Source는 import 응답을 받은 뒤 `cls()`로 **connection 없는 새 Backup**을 만듭니다. Go도 `CurrentLocation`이나 project lookup을 사용하지 않고 disconnected logical view를 반환합니다. request에 ID가 없으므로 생성·seed ID를 넣지 않습니다. 누락 id는 `null`이고, 반환된 null·false·0·빈 문자열·컨테이너 ID도 untyped JSON 값으로 유지합니다.

`Value`는 알려진23 Body 필드와 computed `location`을 포함한24-field JSON object입니다. missing/null 필드는 nullable이며 **`location`은 항상 `null`**입니다. 응답 project_id·availability_zone·location이 있어도 연결 위치로 승격하지 않습니다.

- force·has_dependent_backups·is_incremental은 ordinary Boolean descriptor입니다. 예를 들어 문자열 `"false"`는 truthy이므로 logical true이며 BoolStr 규칙을 적용하지 않습니다.
- links는 nonlist 값을 한 번 list로 감싸고, metadata는 nullable dict이며 nonobject 값은 `{}`입니다.
- size·object_count는 descriptor integer 변환을 따릅니다. 예를 들어 numeric6.9는6, container `[]`는0이며 문자열 `"²"`처럼 변환에 실패하는 입력은 오류입니다.
- ID·name·status·date 등 type 없는 필드는 JSON 원값을 유지합니다. `incremental`은 create request의 wire 변환이며 import response에서는 알려진 `is_incremental`을 대신하지 않습니다.

응답의 `backup` key가 있으면 그 값을 선택하고, 없으면 flat object입니다. 존재하는 null·잘못된 shape는 fallback하지 않습니다. ordinary UTF-8 empty/malformed JSON은 Source의 JSON ValueError tolerance에 맞춰 새 Resource의 all-null24 view를 유지하지만 actual `{}`를 만들어 내지 않습니다. genuine `{}`는 실제 선택한 raw empty object로 보존합니다. valid wrong shape·invalid UTF-8·known descriptor 변환 실패는 실제 response 증거와 함께 오류가 됩니다.

응답 `self`·connection·microversion·_synchronized·computed location은 logical Body 필드가 아닙니다. 응답을 constructor kwargs로 취급해 collision 오류를 내거나 위치를 바꾸지 않습니다. project attribute/wire alias는 parsed object의 소비 순서를 따르며 duplicate key는 마지막 값과 최초 삽입 위치를 사용합니다.

## 실제 응답과 실패 단계

`Applied`는 import HTTP의 실제 body bytes·header·status입니다. `Backup`은 실제 선택한 raw object이며 nullable fields, request 값, generated ID를 채워 넣은 가상 응답이 아닙니다. unknown response fields도 raw 증거에는 남고 logical `Value`에서는 제외합니다. `BackupID`는 logical response id JSON으로 제공하며 native `backup_id`를 id로 바꾸지 않습니다.

`BackupID`는 import 전에는 literal `null`로 시작하며 완료나 HTTP 발생 여부를 판단하는 flag가 아닙니다. 응답 번역에서 nullable response id를 얻으면 그 원값을 보존합니다.

`Value`는 응답 번역·descriptor 변환·마지막 captured-source 검사를 통과한 뒤에만 게시합니다. accepted Read/Close·source·context·model 오류에서도 이미 완료한 `Discovery`와 `Applied`를 보존하고, 실제 object를 파싱한 단계까지 도달했다면 raw `Backup`도 분리해 유지합니다. 각 결과와 ResponseError의 bytes/header는 독립적인 소유 값입니다.

import 교환은 Source의400미만 success 영역을 finite HTTP100..399로 옮깁니다. >=400은 native HTTP error와 body/header/code를 유지하고 `Applied`를 만들지 않습니다. native rejection에서 원래 Gophercloud가 버리는 body Read/Close 오류까지 전부 보존한다고 주장하지 않습니다. accepted IO와 rejected IO의 범위는 구분됩니다. parser 오류는 HTTP 재시도나 rollback을 일으키지 않습니다. 기존 native authentication·reauthentication·bounded retry는 유지하되 fixed route·method·exact serialized POST body/framing·source 정책을 바꿔 resend할 수는 없습니다.

## 기존 native API와 한계

Gophercloud `v2.15.0`의 [native Import](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/requests.go#L306)는 이미 `backups/import_record`를 사용하지만 accepted201과 ID·Name 두 문자열 projection입니다. `BackupURL []byte`는 표준 Go JSON base64 문자열로 직렬화합니다. native `ImportBackup` DTO는 opaque locator 내부를 caller가 구성할 때 쓰는 별도 타입이며, Source가 반환하는 nullable full Backup 모델이 아닙니다. 새 `ImportBackup` method는 literal string URL과 fuller result를 사용하고 기존 native `Import` 동작은 유지합니다.

Go JSON representation은 Python Resource의 mutable dict/session/descriptor identity와 다릅니다. raw proof는 숫자 spelling·precision·unknown fields를 보존하지만 logical conversion의 exact-decimal truthiness/equality는 Python JSON float rounding/underflow와 경계가 다릅니다. UTF-8 parser·string surrogate·number/depth 한계와 선택 Python JSON backend의 encoding 정책을 완전히 복제하지 않습니다. discovery200/300, finite 후보, typed advertisement, clean404/405 fallback, malformed accepted JSON terminal 정책과 selected version literal 보존도 명시적인 Go mapping입니다. Source utility의 tuple decimal integer parsing은 부호·공백·digit separator·Unicode decimal digits를 포함하지만 CPython interpreter 버전/구성의 arbitrary integer digit 한계와 고정 Unicode table 경계를 자동으로 동일시하지 않습니다.

이 문서는 실제 v3 Proxy import 하나의 사용과 차이를 설명합니다. native Import·v2·Resource·다른 backup 기능을 자동으로 지원 완료 처리하거나 authenticated OpenStack/Python runtime 및 전체 SDK 완료를 주장하지 않습니다.

인접 기능은 [Backup 조회·검색](volume-backups.md), [Backup export record](volume-backup-export-record.md), [Backup restore/reset](volume-backup-actions.md)를 참고하세요.
