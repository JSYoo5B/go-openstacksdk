# Cinder metadata 서버 계약

기준은 Gophercloud v2.15.0, openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, 공식 Cinder **27.0.0** commit `c36f40684e140a1492b00dd4812e8dce17fb2ebf`입니다. [공통 사용법](../blockstorage/metadata/README.md)은 고정 범위와 옵션을 설명합니다. 아래 서버 근거는 고정 소스 비교이며 실제 클라우드 검증은 아닙니다. Go의 네 v2/v3 binding은 HTTP fixture로 검증하며 legacy v2 배포 여부는 클라우드가 결정합니다.

`P`는 선택한 서비스의 project·reverse proxy prefix입니다. SDK는 프로젝트를 URL에서 추정하거나 추가하지 않습니다.

| 작업 | P 뒤 경로 | 입력 | 성공 응답 |
|---|---|---|---|
| Get | `/volumes/{id}/metadata` 또는 `/snapshots/{id}/metadata` | 없음 | 200, 실제 `metadata` 객체 |
| Merge | 같은 경로, POST | `{"metadata":{...}}` | 200, 병합한 전체 객체 |
| Replace | 같은 경로, PUT | `{"metadata":{...}}` | 200, 전체 교체한 객체 |
| DeleteKeys | 같은 경로 + `/{key}`, DELETE | 없음 | 200, 빈 응답 |

[router](https://github.com/openstack/cinder/blob/c36f40684e140a1492b00dd4812e8dce17fb2ebf/cinder/api/v3/router.py#L132-L163), [Volume controller](https://github.com/openstack/cinder/blob/c36f40684e140a1492b00dd4812e8dce17fb2ebf/cinder/api/v2/volume_metadata.py#L46-L142), [Snapshot controller](https://github.com/openstack/cinder/blob/c36f40684e140a1492b00dd4812e8dce17fb2ebf/cinder/api/v3/snapshot_metadata.py#L42-L134)에 근거합니다. metadata 자체에는 새 v3 microversion gate가 없습니다. nil/빈 map은 `metadata:{}`이며 POST는 기존 항목을 유지하고 PUT은 지웁니다. `metadata:null`은 같은 뜻이 아닙니다.

Volume v3는 **3.15 이상**의 GET에서 ETag를 반환하고 collection·단일 키 PUT에서 If-Match를 검사합니다. POST·키 DELETE와 Snapshot은 같은 검사를 제공하지 않습니다. SDK는 실제 ETag·412를 보존하고 사전 조회나 조건 없는 재시도를 자동으로 추가하지 않습니다. [고정 ETag 구현](https://github.com/openstack/cinder/blob/c36f40684e140a1492b00dd4812e8dce17fb2ebf/cinder/api/v3/volume_metadata.py#L33-L77)

Volume의 [metadata schema](https://github.com/openstack/cinder/blob/c36f40684e140a1492b00dd4812e8dce17fb2ebf/cinder/api/schemas/volume_metadata.py#L25-L49)에는 키·값 길이와 키 패턴 제약이 있습니다. Snapshot에 이 패턴을 공통으로 적용하지 않습니다. SDK는 Unicode 문자열과 안전한 삭제 경로를 처리하고 schema·권한·리소스 상태는 서버가 검사합니다. 삭제 키는 한 번 escape하며 `EscapedPath` fixture가 확인하는 것은 SDK의 URL 구성입니다. 실제 proxy/WSGI의 encoded slash 처리는 추가 클라우드 검증 대상입니다.

고정 Python의 [MetadataMixin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/metadata.py)은 fetch 결과를 기존 Resource에 적용하고 set의 요청 map을 cache에 저장합니다. Go 범위는 실제 HTTP metadata·원문·헤더·코드를 반환하며 Volume/Snapshot cache를 만들지 않습니다. 순서별 키 삭제는 첫 오류에서 멈추고 이전 성공을 반환합니다. 트랜잭션이나 rollback을 의미하지 않습니다.

Backup은 별도 계약입니다. [고정 router](https://github.com/openstack/cinder/blob/c36f40684e140a1492b00dd4812e8dce17fb2ebf/cinder/api/v3/router.py#L183-L187)에 `/backups/{id}/metadata` 하위 경로가 없습니다. Python Backup이 MetadataMixin을 상속한다는 이유로 이 경로를 만들지 않습니다. 일반 Backup PUT의 metadata는 3.43부터 전체 교체이며, 성공 응답은 id/name/links summary입니다. [Backup update](https://github.com/openstack/cinder/blob/c36f40684e140a1492b00dd4812e8dce17fb2ebf/cinder/api/v3/backups.py#L37-L61), [summary view](https://github.com/openstack/cinder/blob/c36f40684e140a1492b00dd4812e8dce17fb2ebf/cinder/api/views/backups.py#L37-L46), [metadata 교체](https://github.com/openstack/cinder/blob/c36f40684e140a1492b00dd4812e8dce17fb2ebf/cinder/objects/backup.py#L174-L185)에 근거합니다. API reference의 metadata 응답 설명과 실제 view의 차이도 있으므로 요청 map을 응답으로 합성하지 않습니다. Backup의 조회·교체·merge/delete fallback은 별도 API와 동시 변경 정책이 필요합니다.

일반 Backup 수정은 microversion 3.9부터 제공되며 기존 [`backups.API.Update`](../blockstorage/v3/backups/README.md)를 사용합니다. 공개 signature와 aliases/factories는 유지하며, SDK adapter가 native의 flat 필드를 서버 schema가 요구하는 `{"backup":{...}}` envelope로 보정합니다. 확장 필드도 `backup` 안에서 병합하고 typed core 입력은 덮어쓰지 않습니다. `UpdateOpts.Metadata`가 nil이면 필드를 생략하고, nonnil 빈 map은 `metadata:{}`로 전체 metadata를 지웁니다. nonempty map도 전체 교체이며, metadata를 보내려면 선택한 microversion이 3.43 이상이어야 합니다.

반환하는 native `*Backup`은 실제 200 응답을 decode한 결과입니다. 고정 서버의 summary에는 metadata/status가 없으므로 요청값을 채워 반환하지 않습니다. 이 일반 PUT 보정은 Python `MetadataMixin`의 stock 서버에 없는 childroute 지원을 뜻하지 않으며, implicit GET/merge/write나 키 삭제·동시성·트랜잭션 정책도 추가하지 않습니다.
