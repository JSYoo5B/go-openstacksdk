# Backup metadata Python compatibility profile

기준은 Gophercloud **v2.15.0**, openstacksdk **`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`**입니다. 이 프로파일은 고정 Python SDK의 메타데이터 요청을 명시적으로 재현합니다. 일반 Backup API의 stock Cinder 지원 목록을 늘리는 기능은 아닙니다.

고정 Cinder **27.0.0** commit `c36f40684e140a1492b00dd4812e8dce17fb2ebf`의 [v3 router](https://github.com/openstack/cinder/blob/c36f40684e140a1492b00dd4812e8dce17fb2ebf/cinder/api/v3/router.py#L183-L187)는 `/backups/{id}/metadata` 하위 경로를 등록하지 않습니다. 따라서 stock 서버에서 아래 호출은 **404를 반환할 수 있습니다**. 기존 [서버 계약 문서](../../docs/cinder-metadata-server-contracts.md)의 일반 Volume/Snapshot metadata scope와 Backup 경계는 유지합니다. 이 문서와 HTTP fixture는 실제 stock 클라우드 성공 검증을 의미하지 않습니다.

## 선택과 사용

Python의 [Backup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L27)은 [MetadataMixin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/metadata.py)을 상속합니다. [Proxy 네 함수](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1772-L1867)는 이 mixin에 위임합니다. Go는 `API.PythonMetadata()`로 선택한 프로파일에만 대응 메서드를 제공합니다.

| Python 호출 | 명시적 Go 프로파일 | 실제 요청 |
|---|---|---|
| `conn.block_storage.fetch_backup_metadata(id)` | `profile.FetchMetadata(ctx, id)` | GET `/backups/{id}/metadata` |
| `conn.block_storage.get_backup_metadata(id)` | `profile.GetMetadata(ctx, id)` | 같은 GET, deprecated alias |
| `conn.block_storage.set_backup_metadata(id, owner="ops")` | `profile.SetMetadata(ctx, id, map[string]string{"owner":"ops"})` | POST 같은 경로, `{"metadata":{"owner":"ops"}}` |
| `conn.block_storage.delete_backup_metadata(id)` | `profile.DeleteMetadata(ctx, id, nil)` | PUT 같은 경로, `{"metadata":{}}` |
| `conn.block_storage.delete_backup_metadata(id, keys=[])` | `profile.DeleteMetadata(ctx, id, []string{})` | HTTP 없음 |
| `conn.block_storage.delete_backup_metadata(id, keys=["owner"])` | `profile.DeleteMetadata(ctx, id, []string{"owner"})` | DELETE `/backups/{id}/metadata/owner` |

```go
package example

import (
    "context"
    "encoding/json"

    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups"
    "github.com/gophercloud/gophercloud/v2"
)

// 이 child route를 제공하는 배포에 한해 Python 호환 호출을 선택한다.
func fetchBackupMetadata(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]string, error) {
    profile := backups.New(client).PythonMetadata()
    result, err := profile.FetchMetadata(ctx, id,
        backups.WithMetadataHeader("X-Request-Id", "application-request"))
    if err != nil {
        return nil, err // stock 서버의 404를 포함한 원인을 그대로 반환한다.
    }
    var values map[string]string
    if result.Metadata != nil {
        if err := json.Unmarshal(result.Metadata, &values); err != nil {
            return nil, err
        }
    }
    return values, nil
}
```

stock Cinder의 일반 Backup metadata 전체 교체는 기존 `Backups.Update(ctx, id, backups.UpdateOpts{Metadata: completeMap})`를 사용합니다. 서버의 update는 3.9부터, metadata 필드는 **3.43부터** 제공됩니다. 이 프로파일은 child route 실패를 일반 Backup GET/PUT으로 바꾸거나 implicit read/merge/write를 하지 않습니다. 일반 PUT을 merge/delete로 바꾸려면 별도의 동시 변경 정책이 필요합니다.

## 결과와 Go 차이

`MetadataResult`는 요청한 `BackupID`, 관측한 `Metadata json.RawMessage`, 실제 `Response` 원문·헤더·상태를 분리합니다. fetch는 응답 JSON 객체의 정확한 소문자 `metadata` 값을 읽습니다. 생략은 nil, 명시적 null은 nonnil `"null"`이며, 큰 정수·배열·확장 값은 원래 JSON 표현을 유지합니다. JSON 객체가 아닌 루트와 잘못된 UTF-8는 accepted-response 오류가 됩니다. Python은 root 형태에 따라 membership 검사를 수행하지만 Go는 응답 객체 계약을 명시합니다.

Python fetch/set는 기존 mutable Backup Resource를 갱신하고 반환합니다. Go는 기존 native `Backup` alias와 ABI를 유지하며 Resource cache를 만들지 않습니다. set는 요청 map을 cache 또는 응답 metadata로 합성하지 않고 **불투명한 실제 응답**만 보존합니다. set/clear/key-delete는 성공 응답의 JSON decode를 요구하지 않습니다. 고정 Python `raise_from_response`와 같은 최종 **200..399** 응답 범위를 사용하고, 원래 HTTP 오류·본문 Read/Close 오류·취소 cause를 보존합니다. deprecated Get 별칭은 GoDoc으로 표시하고 Python의 runtime warning은 발생시키지 않습니다.

set 입력은 Python signature의 문자열 metadata에 대응하는 `map[string]string`입니다. nil/빈 map도 `metadata:{}`로 POST합니다. map·keys·헤더 옵션은 callback 전에 소유 복사본을 만들어 재사용합니다. 인증·버전·routing은 SDK가 소유하며 일반 헤더에만 `WithMetadataHeader(s)`를 허용합니다. 프로파일은 configured microversion을 사용하고 3.43 gate나 자동 negotiation을 추가하지 않습니다. MetadataMixin 자체가 이러한 gate 또는 요청별 microversion을 지정하지 않기 때문입니다.

키 삭제는 입력 순서와 중복을 유지하고 첫 오류에서 중단합니다. `MetadataDeleteResult.Deleted`는 이전에 실제 성공한 요청의 key·원문·헤더·상태를 반환합니다. nil clear의 `Cleared`는 실제 PUT 근거이며 입력의 빈 map으로 metadata 응답을 채우지 않습니다. 성공 상태여도 Read/Close/취소 오류가 있으면 완료를 주장하지 않습니다. 삭제는 트랜잭션이 아니며 rollback하지 않습니다. Python mutable cache의 key 제거/KeyError 여부는 재현하지 않습니다.

Go의 ID는 기존 SDK의 단일 UTF-8 path segment 검증을 재사용합니다. 삭제 key는 literal UTF-8 값을 한 번 escape하므로 `/`, `%`, `?`, `#`가 route나 query를 바꾸지 않습니다. Python `utils.urljoin`의 raw key 처리와는 이 점에서 다릅니다. 실제 reverse proxy의 encoded slash 전달은 배포별 검증 대상입니다.

## Source 계약과 검증 근거

아래 fingerprint는 현재 `api/sdk_support_catalog.json`과 unresolved review 행에 기록된 값입니다. 네 행 모두 **Source 프로파일 구현 후보**이며 stock API 지원으로 자동 집계하지 않습니다. 중앙 리뷰 상태와 지원 개수는 별도로 결정합니다.

| Source operation | Source fingerprint | Go API | 계약 회귀 테스트 |
|---|---|---|---|
| `python:block_storage/v3/fetch_backup_metadata` | `44366618ad521a9e7ed3242721117672c9493dc716038b7dcf53f9a1f6866387` | `backups.PythonMetadataProfile.FetchMetadata` | `TestPythonBackupMetadataProfileFetchAndLegacyAliasKeepObservedJSON`, `TestPythonBackupMetadataProfileFetchOmissionNullAndDecodeFailures` |
| `python:block_storage/v3/get_backup_metadata` | `568f7e2447af6c70a2f5973592c8e2c6dd1da8a668c6f77a9e23d3f63016b1ff` | `backups.PythonMetadataProfile.GetMetadata` | `TestPythonBackupMetadataProfileFetchAndLegacyAliasKeepObservedJSON` |
| `python:block_storage/v3/set_backup_metadata` | `a5642b88578610dbf4b7cf27834d987b53375a8821e59aa762d28325179e057d` | `backups.PythonMetadataProfile.SetMetadata` | `TestPythonBackupMetadataProfileSetOwnsInputAndOpaqueAcknowledgement` |
| `python:block_storage/v3/delete_backup_metadata` | `90645fff2c7de7eb341b7818e87543fd0e9aec7f53a8c479feaf2fa22c8b97eb` | `backups.PythonMetadataProfile.DeleteMetadata` | `TestPythonBackupMetadataProfileDeleteNilEmptyLiteralKeysAndPartialFailure` |

공통 회귀는 `TestPythonBackupMetadataProfileKeepsSourceStatusesAndStock404WithoutFallback`, `TestPythonBackupMetadataProfileAcceptedAndRejectedBodyFaultsKeepPhysicalCauses`, `TestPythonBackupMetadataProfilePreflightOwnsOptionsAndPreservesWireCauses`, `TestPythonBackupMetadataProfileSnapshotsHeadersConcurrentReuseAndSourceChanges`입니다. 기존 public Gophercloud testhelper 기반 `internal/testcloud`, 동일 패키지 Python backup HTTP fixture, 공통 REST/source guard와 Cinder metadata 옵션을 재사용합니다. 별도 HTTP 엔진은 없습니다. 테스트 파일은 [metadata_python_test.go](backups/metadata_python_test.go), 구현은 [metadata.go](backups/metadata.go)입니다.

남은 범위는 stock Cinder 미지원 child route의 실제 배포 가능성 검증과 중앙 review의 지원 분류입니다. 프로파일 호출은 원래 404를 보존하며 서버를 지원한다고 추론하지 않습니다. 중앙 통합 runner에서 새 회귀 8개 그룹(하위 사례 포함106개)을 `go test -race -count=1 -timeout 120s`로 실행해 PASS를 확인했습니다. 실행 기록은 `/private/tmp/go-openstacksdk-oct11-backup-profile-focused.jsonl`입니다. 이 검증은 HTTP fixture 기반이며 실제 stock 클라우드의 child route 지원을 입증하지 않습니다.
