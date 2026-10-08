# Backup export

| 기준 API | Go API | 결과 |
|---|---|---|
| deprecated Python Proxy `export_record` | `API.ExportRecord(ctx, id)` | owned opaque Body/Header/StatusCode |
| Python Proxy `export_backup` | `API.ExportBackup(ctx, id)` | owned whole JSON `Value`와 opaque proof |
| Gophercloud native `Export` | 기존 `API.Export(ctx, id)` | native typed `*BackupRecord` |

[실행 가능한 사용법과 Python 비교](../../volume-backup-export-record.md)를 참고하세요.
service는 `conn.BlockStorageV3(ctx)`의 `Backups` accessor 또는 준비한 client의 `backups.New(client)`로
얻습니다. 새 두 export method에는 options/lookup/wait가 없습니다. 선택한 Cinder v3 microversion을
유지하고 malformed JSON을 parse한 경우에도 이미 확보한 실제 응답 proof를 남깁니다.
parsed `Value`는 Python dict 대신 UTF-8 JSON literal이며, `Exported` 존재만으로 성공을 판정하면 안 됩니다.

## Restore 및 status reset

`API.RestoreBackup`은 With 옵션과 nullable Backup 병합 결과를, `API.ResetBackupStatus`는 forced 3.64 action과 owned acknowledgement를 제공합니다.
[실행 가능한 사용법과 Python/native 비교](../../volume-backup-actions.md)를 참고하세요.
기존 `RestoreFromBackup`·`ResetStatus`의 native 계약은 독립적으로 유지됩니다.

`ImportBackup(ctx, service, url)`은 literal record 문자열을 import하고 nullable Backup 및 실제 응답 증거를 반환합니다. [Python/package/Connection 비교](../../volume-backup-import.md)에 microversion 선택과 disconnected 모델을 설명합니다. 기존 native `Import`는 `[]byte` record의 base64 직렬화와 accepted201 ID·Name 계약을 제공합니다.

## Backup 수정과 metadata 전체 교체

기존 `API.Update(ctx, id, opts, options...) (*Backup, error)`와 `UpdateOpts`·`WithUpdateOptions`·`WithUpdateField`를 사용합니다. 일반 Backup 수정은 Cinder v3 microversion 3.9부터 사용할 수 있습니다. SDK adapter는 native의 flat 필드를 Cinder가 요구하는 `{"backup":{...}}` envelope 안에 넣어 `PUT /backups/{id}`로 보냅니다.

`Metadata == nil`이면 metadata를 생략합니다. nonnil 빈 map은 `metadata:{}`를 명시해 전체 metadata를 지우며, nonempty map도 기존 metadata의 **전체 교체**입니다. metadata를 보내려면 준비한 Cinder v3 service에서 3.43 이상의 microversion을 선택해야 합니다. 이 호출이 버전을 자동으로 올리지는 않습니다.

아래 코드는 `ctx`, `backupID`, 이미 3.43 이상을 선택한 `cinder` service를 애플리케이션이 제공하는 예입니다. `WithUpdateOptions`는 opts 전체를 교체하는 기존 factory입니다.

```go
package example

import (
    "context"

    v3 "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups"
)

func clearBackupMetadata(ctx context.Context, cinder *v3.Service, backupID string) (*backups.Backup, error) {
    opts := backups.UpdateOpts{
        Metadata: map[string]string{}, // nil이면 metadata 변경을 요청하지 않음
    }
    return cinder.Backups.Update(ctx, backupID, backups.UpdateOpts{},
        backups.WithUpdateOptions(opts),
    )
}
```

`WithUpdateField`의 확장 필드도 `backup` 내부에 추가합니다. typed core 입력인 `name`·`description`·`metadata`를 확장 필드로 덮어쓸 수 없으며, 해당 값은 `UpdateOpts` 또는 `WithUpdateOptions`로 설정합니다. 확장 필드의 서버 지원 여부는 선택한 API가 판단합니다.

반환하는 native `*Backup`은 실제 서버 응답의 typed 결과입니다. 고정 Cinder 27.0.0의 Update 응답은 id/name/links summary이므로 요청 metadata나 status를 응답에 합성하지 않습니다. 최신 metadata가 필요하면 별도 조회가 필요합니다. [서버 계약](../../../docs/cinder-metadata-server-contracts.md)에 envelope와 응답 범위를 설명합니다.

Python Backup의 `MetadataMixin` childroute는 고정 stock 서버에 없습니다. 이 Update 보정은 `/backups/{id}/metadata`를 생성하거나, 자동 조회·merge·키별 삭제·read-modify-write·동시성 보장을 추가하지 않습니다.
