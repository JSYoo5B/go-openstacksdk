# Cinder v3 native backup 호출

`backups.New(client)`(또는 `service.Backups`)의 generated 메서드는 Gophercloud `v2.15.0`의 [v3 backups 요청](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/requests.go)을 호출합니다. SDK는 오류에 `resource.OperationError{Resource: "backups"}` 문맥을 더하고, `Update`만 아래처럼 본문 envelope를 보정합니다. export·import·restore·status reset의 SDK 소유 호출은 [Backup 사용법](backups/README.md)을 참고합니다.

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create(ctx, opts, options...)` | `POST backups`, `{"backup": {...}}` | 202 |
| `Get(ctx, id)` | `GET backups/{id}` | 200 |
| `List(ctx, options...)` | `GET backups` | native pager 200, 204, 300 |
| `ListDetail(ctx, options...)` | `GET backups/detail` | native pager 200, 204, 300 |
| `Update(ctx, id, opts, options...)` | `PUT backups/{id}`, `{"backup": {...}}` | 200 |
| `RestoreFromBackup(ctx, id, opts, options...)` | `POST backups/{id}/restore`, `{"restore": {...}}` | 202 |
| `Delete(ctx, id)` | `DELETE backups/{id}` | 202, 204 |

`CreateOpts`의 `VolumeID`는 필수라 비어 있으면 HTTP 전에 오류입니다. `Force`·`Incremental`은 bool이라 true일 때만 보냅니다. `RestoreOpts`의 빈 값은 모두 생략하므로 빈 옵션은 `{"restore": {}}`이고, 응답은 `restore` key의 backup·volume ID와 volume 이름입니다.

`List`는 `ListOpts`(이름·상태·volume·`all_tenants` 등), `ListDetail`은 별도 `ListDetailOpts`(`all_tenants`, 정렬, paging, `with_count`)를 받습니다. 두 목록 모두 `backups_links`의 next href를 따라가며, 상세 목록은 이름·상태 필터를 typed 옵션으로 제공하지 않으므로 필요하면 `WithListDetailQuery`로 더합니다.

Gophercloud native `Update`는 `name`·`description`·`metadata`를 envelope 없이 최상위에 보냅니다. Cinder는 `{"backup": {...}}`를 요구하므로 SDK adapter가 그 값을 `backup` 안으로 옮기고, 확장 필드도 같은 envelope 안에 넣습니다. 이 보정과 빈 metadata 처리는 [Backup 수정](backups/README.md#backup-수정과-metadata-전체-교체)에 설명합니다.

응답은 `backup` key를 직접 찾아 `{}`·null을 빈 backup으로 돌려주고, 다른 key만 있으면 오류입니다. `metadata`는 map pointer라 null이면 nil이고, `availability_zone`도 pointer입니다. 생성·수정·데이터 시각은 시간대 없는 형식만 받아서 끝에 `Z`가 붙으면 decode 오류입니다.

## 관리자 호출

기본 Cinder 정책상 관리자 호출인 네 메서드입니다.

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Export(ctx, id)` | `GET backups/{id}/export_record` | 200 |
| `Import(ctx, opts, options...)` | `POST backups/import_record`, `{"backup-record": {...}}` | 201 |
| `ResetStatus(ctx, id, opts, options...)` | `POST backups/{id}/action`, `{"os-reset_status": {"status": ...}}` | 202 |
| `ForceDelete(ctx, id)` | `POST backups/{id}/action`, `{"os-force_delete": {}}` | 202 |

`Export`의 `BackupRecord.BackupURL`은 `[]byte`라 응답의 `backup_url` 문자열을 base64로 해석합니다. base64가 아니면 decode 오류이고, `backup-record` key가 없으면 빈 record를 돌려줍니다. `Import`는 같은 byte 값을 다시 base64 문자열로 보내고 응답 `backup` key의 ID·이름을 돌려줍니다. record 원문을 그대로 다루는 SDK 소유 호출은 [Backup 사용법](backups/README.md)의 `ExportRecord`·`ImportBackup`을 참고합니다.
