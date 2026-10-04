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
