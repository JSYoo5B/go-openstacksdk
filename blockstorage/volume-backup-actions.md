# Backup restore 및 status reset

`RestoreVolumeBackup`은 알려진 Cinder v3 backup ID로 restore를 요청하고, cached Backup 필드와 응답을 병합한 nullable Backup 값을 반환합니다. `ResetVolumeBackupStatus`는 지정한 status 문자열을 action에 전달하고 실제 HTTP acknowledgement를 보존합니다. 두 함수는 이름 조회나 완료 대기를 수행하지 않습니다.

기준 소스는 openstacksdk commit `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 실제 v3 Proxy 선언입니다. Python 호출은 `conn.block_storage.restore_backup(backup_id, volume=volume_id, name=name)`와 `conn.block_storage.reset_backup_status(backup_id, status)`입니다. 아래 Go package·Connection 함수는 이 Proxy 계약을 사용하는 편의 API입니다. 같은 이름의 Python cloud helper가 존재한다는 의미는 아닙니다.

| 호출 경로 | Restore | Reset |
|---|---|---|
| Connection | `conn.RestoreVolumeBackup(ctx, blockstorage.RestoreVolumeBackupRequest{BackupID: id}, options...)` | `conn.ResetVolumeBackupStatus(ctx, blockstorage.ResetVolumeBackupStatusRequest{BackupID: id, Status: status})` |
| 직접 package 함수 | `blockstorage.RestoreVolumeBackup(ctx, cinder, request, options...)` | `blockstorage.ResetVolumeBackupStatus(ctx, cinder, request)` |
| v3 service | `service.Backups.RestoreBackup(ctx, id, options...)` | `service.Backups.ResetBackupStatus(ctx, id, status)` |
| 기존 native facade | `service.Backups.RestoreFromBackup(ctx, id, backups.RestoreOpts{VolumeID: volumeID, Name: name})` | `service.Backups.ResetStatus(ctx, id, backups.ResetStatusOpts{Status: status})` |

## 실행 가능한 예제

아래 프로그램은 한 번 실행할 때 한 경로와 한 action만 호출합니다. `CLOUD_NAME`은 clouds.yaml의 이름, `BACKUP_ID`는 호출자가 알고 있는 ID입니다. `BACKUP_PATH`는 `connection`·`direct`·`service`·`native` 중 하나이며 기본값은 `connection`입니다. `BACKUP_ACTION`은 `restore` 또는 `reset`이며 기본값은 `restore`입니다. Restore에는 `RESTORE_VOLUME_ID` 또는 `RESTORE_NAME`을 지정합니다. 선택적인 `BACKUP_SEED`에는 cached Backup JSON object를 넣습니다. Reset에는 `BACKUP_STATUS` 환경 변수를 반드시 정의합니다. `BACKUP_STATUS=''`도 정의된 값으로 전달합니다.

예를 들어 `CLOUD_NAME=devstack BACKUP_ID=backup-id RESTORE_NAME=restored-volume go run ./backup-actions-example`은 Connection restore를 호출합니다. `CLOUD_NAME=devstack BACKUP_ID=backup-id BACKUP_ACTION=reset BACKUP_STATUS='' BACKUP_PATH=service go run ./backup-actions-example`은 빈 status를 v3 service action에 전달합니다. 프로그램을 모듈 안의 별도 example 디렉터리에 저장하면 됩니다. 30초 제한은 예제 application의 parent context 정책입니다.

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/blockstorage/v3/backups"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cloudName, backupID := os.Getenv("CLOUD_NAME"), os.Getenv("BACKUP_ID")
	if cloudName == "" || backupID == "" {
		return fmt.Errorf("CLOUD_NAME and BACKUP_ID are required")
	}
	path, action := os.Getenv("BACKUP_PATH"), os.Getenv("BACKUP_ACTION")
	if path == "" {
		path = "connection"
	}
	if action == "" {
		action = "restore"
	}
	switch path {
	case "connection", "direct", "service", "native":
	default:
		return fmt.Errorf("unknown BACKUP_PATH %q", path)
	}
	volumeID, name := os.Getenv("RESTORE_VOLUME_ID"), os.Getenv("RESTORE_NAME")
	status, statusPresent := os.LookupEnv("BACKUP_STATUS")
	switch action {
	case "restore":
		if volumeID == "" && name == "" {
			return fmt.Errorf("RESTORE_VOLUME_ID or RESTORE_NAME is required")
		}
	case "reset":
		if !statusPresent {
			return fmt.Errorf("BACKUP_STATUS must be defined; an empty value is allowed")
		}
	default:
		return fmt.Errorf("unknown BACKUP_ACTION %q", action)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
	if err != nil {
		return err
	}
	if action == "reset" {
		input := blockstorage.ResetVolumeBackupStatusRequest{BackupID: backupID, Status: status}
		if path == "connection" {
			result, err := conn.ResetVolumeBackupStatus(ctx, input)
			inspectReset(result)
			return err
		}
		service, err := conn.BlockStorageV3(ctx)
		if err != nil {
			return err
		}
		switch path {
		case "direct":
			result, err := blockstorage.ResetVolumeBackupStatus(ctx, service.RawClient(), input)
			inspectReset(result)
			return err
		case "service":
			result, err := service.Backups.ResetBackupStatus(ctx, backupID, status)
			inspectReset(result)
			return err
		default:
			return service.Backups.ResetStatus(ctx, backupID, backups.ResetStatusOpts{Status: status})
		}
	}
	options := []blockstorage.RestoreVolumeBackupOption{
		blockstorage.WithRestoreVolumeBackupVolumeID(volumeID),
		blockstorage.WithRestoreVolumeBackupName(name),
	}
	if seed, present := os.LookupEnv("BACKUP_SEED"); present {
		options = append(options, blockstorage.WithRestoreVolumeBackupSeed(json.RawMessage(seed)))
	}
	policy, err := blockstorage.PrepareRestoreVolumeBackupOptions(ctx, options...)
	if err != nil {
		return err
	}
	input := blockstorage.RestoreVolumeBackupRequest{BackupID: backupID}
	if path == "connection" {
		result, err := conn.RestoreVolumeBackup(ctx, input, blockstorage.WithRestoreVolumeBackupOptions(policy))
		inspectRestore(result)
		return err
	}
	service, err := conn.BlockStorageV3(ctx)
	if err != nil {
		return err
	}
	if path == "native" {
		result, err := service.Backups.RestoreFromBackup(ctx, backupID, backups.RestoreOpts{VolumeID: volumeID, Name: name})
		if result != nil {
			fmt.Println("native restore:", result.BackupID, result.VolumeID, result.VolumeName)
		}
		return err
	}
	location, err := conn.CurrentLocation()
	if err != nil {
		return err
	}
	policy, err = blockstorage.PrepareRestoreVolumeBackupOptions(ctx,
		blockstorage.WithRestoreVolumeBackupOptions(policy),
		blockstorage.WithRestoreVolumeBackupLocation(location))
	if err != nil {
		return err
	}
	if path == "direct" {
		result, err := blockstorage.RestoreVolumeBackup(ctx, service.RawClient(), input, blockstorage.WithRestoreVolumeBackupOptions(policy))
		inspectRestore(result)
		return err
	}
	result, err := service.Backups.RestoreBackup(ctx, backupID, backups.WithRestoreBackupOptions(policy))
	inspectRestore(result)
	return err
}

func inspectRestore(result *blockstorage.RestoreVolumeBackupResult) {
	if result == nil {
		return
	}
	fmt.Println("merged logical backup ID JSON:", string(result.BackupID))
	inspectApplied(result.Applied)
	if result.Backup != nil {
		actual := result.Backup.Clone()
		fmt.Println("actual selected fields:", len(actual.Body), "physical backup_id:", string(actual.Body["backup_id"]))
	}
	if result.Value != nil {
		formatted, err := json.MarshalIndent(result.Value, "", "  ")
		if err == nil {
			fmt.Println("merged nullable Backup:", string(formatted))
		}
	}
}

func inspectReset(result *blockstorage.ResetVolumeBackupStatusResult) {
	if result == nil {
		return
	}
	fmt.Println("requested backup ID:", result.BackupID, "helper completed:", result.Completed)
	inspectApplied(result.Applied)
}

func inspectApplied(page *blockstorage.VolumeBackupMutationPage) {
	if page != nil {
		fmt.Println("admitted HTTP:", page.StatusCode, "actual body bytes:", len(page.Body))
	}
}
```

`inspectRestore`와 `inspectReset`은 오류를 검사하기 전에도 실제 응답 증거를 출력합니다. 증거가 존재하더라도 호출자가 반환된 `err`를 반드시 검사해야 합니다. Native 분기는 기존 typed facade 비교 경로이며 `BACKUP_SEED`나 logical Location을 사용하지 않습니다.

## Restore 입력과 결과

`RestoreVolumeBackupOpts`와 `RestoreBackupOpts`는 같은 library-owned 옵션 구조를 사용합니다. `VolumeID`, `Name`은 `*string`, `Seed`는 `json.RawMessage`, `Location`은 `*resource.CloudLocation`입니다. `WithRestoreVolumeBackupOptions`/`WithRestoreBackupOptions`로 complete 옵션을 넘기거나 `With...VolumeID`, `With...Name`, `With...Seed`, `With...Location`으로 설정합니다. `Prepare...Options`는 옵션을 실행하고 pointer·Seed·Location을 복사합니다. Complete 옵션 factory는 전체 값을 교체하며 개별 factory는 해당 필드를 설정하므로 호출 순서가 적용됩니다.

`VolumeID`와 `Name`이 nil이거나 빈 문자열이면 해당 필드를 body에서 생략합니다. 둘 다 생략되면 local 오류입니다. 둘 다 nonempty이면 둘 다 전달합니다. 이 값들은 조회할 참조가 아니라 literal JSON 문자열이며 UTF-8이어야 합니다. `VolumeID`에는 요청 경로용 ID 제한을 적용하지 않습니다. SDK는 Volume이나 Backup을 먼저 조회하지 않습니다.

요청의 `BackupID`는 비어 있지 않은 UTF-8의 안전한 단일 경로 segment입니다. slash·공백·제어 문자·미리 escape한 값 등은 local 오류입니다. 이름처럼 생긴 문자열도 그대로 ID로 사용합니다. 실제 경로는 처음 입력한 ID에 고정한 `POST backups/{escaped ID}/restore`이며 query가 없습니다.

nil Seed는 입력 ID만 가진 cached 상태를 만듭니다. `{}` Seed는 그 ID를 유지합니다. Seed는 UTF-8 JSON object여야 하고 known Backup 필드만 논리 상태에 적용됩니다. Seed에 `id`를 넣으면 요청 ID와 같은 string이어야 합니다. null·다른 ID·비문자열 ID를 가진 Seed는 요청 전 오류입니다. `Location`은 owned cloud view를 지정합니다. Connection restore는 옵션 준비와 ID/body/Seed 검증 후 configured Location을 소유하고 cached Cinder v3를 선택합니다. 직접 package와 service 호출에서 Location을 생략하면 capture한 provider의 기록된 project scope를 사용합니다. 응답 project·availability zone에 따른 Location 계산은 nullable Backup view에 포함됩니다.

| Restore 결과 필드 | 의미 |
|---|---|
| `BackupID` | 병합된 logical `id`의 `json.RawMessage`. 응답의 실제 `id`가 입력 ID를 바꾸거나 null·비문자열이 될 수 있음 |
| `Applied` | admitted HTTP 응답의 실제 전체 body bytes·header·status를 소유한 copy |
| `Backup` | 선택한 실제 response object의 `RawResource`; cached Seed나 요청 필드를 채워 넣지 않음 |
| `Value` | Seed와 응답 known 필드를 병합한 nullable full Backup JSON. 전체 helper가 성공한 경우에만 설정 |

결과에는 별도의 `FixedBackupID` string 필드가 없습니다. 응답의 logical ID가 달라져도 추가 요청은 없고 원래 POST 경로는 바뀌지 않습니다. 일반 restore 응답의 `backup_id`는 Backup descriptor가 아니므로 `Value`와 logical `BackupID`를 갱신하지 않습니다. 실제 응답의 `backup_id`는 `Applied`와 `Backup.Body`에서 확인합니다. `volume_id`, `volume_name`은 known Backup 필드입니다. sparse 응답에 없는 cached name·status·기타 필드는 유지하며 새로운 status를 합성하지 않습니다.

응답은 present `restore` key, 없으면 present `backup` key, 둘 다 없으면 flat object 순으로 선택합니다. 선택된 key가 null·array·scalar이면 다른 key로 fallback하지 않습니다. `{}`는 실제 빈 object를 보존하고 Seed를 유지합니다. UTF-8의 empty 또는 malformed JSON body는 logical Seed를 유지하는 성공 응답이 될 수 있으며 이때 `Backup`은 nil입니다. 유효한 JSON의 wrong shape, invalid UTF-8, descriptor 또는 Location 변환 오류는 실패입니다.

`Value`는 missing/null 필드를 null로 유지하는 23개 Backup 필드와 computed `location`을 포함합니다. Python bool truthiness·list wrapping·dict/int 규칙을 사용하는 기존 Backup normalizer를 재사용하며 날짜 문자열을 Go `time.Time`으로 바꾸지 않습니다. raw response와 logical view는 독립적으로 소유합니다.

Restore는 선택한 client microversion 또는 생략 상태를 유지합니다. Backup의 `_max_microversion=3.64`를 근거로 upgrade·cap·discovery하지 않습니다. Reset과 같은 forced action 정책도 사용하지 않습니다.

## Reset 입력과 결과

`ResetVolumeBackupStatusRequest`의 `BackupID`, `Status`는 required 입력이며 선택 옵션이나 status 기본값이 없습니다. Status는 UTF-8 string이면 빈 값·알려지지 않은 값도 그대로 `{"os-reset_status":{"status":...}}`에 넣습니다. enum이나 현재 backup 상태를 검증하지 않습니다. 요청은 query 없는 `POST backups/{escaped ID}/action`입니다.

Reset은 Source `Backup._action`에 맞춰 operation 전용 client copy의 microversion을 `3.64`로 고정하고 `Openstack-Api-Version: volume 3.64`, `X-Openstack-Volume-Api-Version: 3.64`를 보냅니다. 선택한 원본 client의 microversion과 header는 수정하지 않습니다. Connection reset은 입력을 검증하고 cached Cinder v3만 선택합니다. Resource constructor·descriptor 변환·CurrentLocation 또는 nullable Backup model을 계산하지 않습니다.

`ResetVolumeBackupStatusResult`/`ResetBackupStatusResult`에는 요청 ID string인 `BackupID`, 실제 acknowledgement proof인 `Applied`, bool `Completed`가 있습니다. Source의 implicit `None` 반환을 Go의 owned proof와 오류 모델로 대응한 구조입니다. `Completed=true`는 admitted 응답을 읽고 닫고 context/source guard까지 성공한 helper 호출을 뜻합니다. 서버 상태를 다시 읽어 확인하거나 reset 완료를 기다렸다는 의미가 아닙니다. body는 JSON일 필요가 없고 empty·malformed·binary도 opaque acknowledgement로 보존합니다. cached Backup의 status를 바꾸지 않습니다.

## HTTP 증거와 오류

두 mapping helper의 원래 admission 범위는 유한 HTTP `100..399`입니다. Native retry가 `OkCodes`를 넓혀도 실제 `>=400` 응답을 성공 proof로 바꾸지 않습니다. 404 역시 terminal 오류입니다. Accepted 응답 뒤 read·Close·context·source·restore parsing 오류가 발생하면 이미 확보한 실제 `Applied` 증거가 남을 수 있습니다. Restore의 `Value`와 reset의 `Completed`는 이 경우 성공으로 설정하지 않습니다. Native 자체에서 rejected한 응답의 IO 증거 범위를 모든 body read/Close 오류 보존으로 넓혀 설명하지 않습니다.

이 두 호출과 기존 force delete는 scoped `backupPost`를 공유합니다. 실제 POST의 method·URL·serialized body bytes·ContentLength·TransferEncoding을 physical attempt에서 검사합니다. Reset/force에는 canonical 3.64 header 검사도 적용합니다. Native retry·reauthentication·redirect callback이 policy를 바꾸면 sticky 오류를 유지하고 바뀐 요청의 전송을 차단합니다. Body 검사에서 읽은 body는 닫고 `NoBody`로 교체한 뒤, 일치하는 bytes만 새 owned body로 복구합니다. 이 보장은 이 scoped backup POST에 관한 것이며 다른 SDK operation의 모든 transport 정책을 의미하지 않습니다.

응답을 accepted한 뒤 decoder retry, workflow 재실행, wait, rollback, follow-up GET을 하지 않습니다. 호출자의 parent context cancellation/deadline과 custom cause는 반환 오류에 남습니다.

## Python 및 native 경계

Pinned Python [Proxy.restore_backup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1871)는 local Resource 생성/재사용 후 [Backup.restore](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L200)를 호출하고 같은 mutable Backup 인스턴스를 반환합니다. 별도 Python Restore resource는 없습니다. `volume_id` deprecated keyword는 `volume`으로 바꾸고 warning을 내며 두 keyword가 함께 있으면 old 값이 new 값을 덮어씁니다. Go는 이 warning과 mutable 객체 정체성을 재현하지 않습니다.

Python `_get_resource`는 Resource·dict/Munch·None·literal을 받는 동적 constructor/update 경로와 descriptor/Location 부수효과를 가집니다. Go restore는 safe ID와 copied JSON Seed·owned Location으로 그 상태를 대응합니다. Go reset은 safe ID의 opaque action으로 constructor 부수효과를 의도적으로 생략합니다. Python annotation은 runtime 검사가 아니어서 nonstring status/volume/name도 전달될 수 있지만 Go는 명시한 UTF-8 string 입력 영역을 사용합니다. 임의 object, bool/null/container 인자, permissive URL coercion까지 대응한 API가 아닙니다.

Python Requests의 response encoding·JSON backend·BOM/non-UTF-8 추정·숫자와 duplicate key 표현은 Go JSON/UTF-8 정책과 구분합니다. Go restore의 ordinary UTF-8 malformed JSON tolerance를 모든 Python decoding backend와 동일하다고 주장하지 않습니다. unknown response fields는 logical view에서 제외하면서 physical proof에 보존합니다.

기존 native `RestoreFromBackup`는 202만 받고 `BackupID`, `VolumeID`, `VolumeName`의 세 string을 반환합니다. Native `RestoreOpts`의 두 string은 omitempty이며 둘 다 비어도 empty restore object를 보낼 수 있습니다. nullable full Backup, Seed merge와 response tolerance를 제공하지 않습니다. Native `ResetStatus`도 202만 받고 선택한 client microversion을 쓰는 error-only facade입니다. 새 `ResetBackupStatus`의 forced 3.64·opaque proof 계약과 구분합니다. 기존 `Resources`는 native `Backup` collection이며 restore-result collection이 아닙니다.

현재 변경의 Source 대상은 v3 `restore_backup`, `reset_backup_status` 두 Proxy 선언입니다. Deprecated reset aliases, Python Resource 단독 operation, native 단독 operation, v2, import/export 또는 다른 cloud helper의 지원 상태를 이 사용 가이드로 승격하지 않습니다. 전체 SDK 완료나 모든 Python runtime 객체·backend의 완전한 parity를 주장하지 않습니다.
