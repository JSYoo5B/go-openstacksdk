# Cinder v2 native 호출

Cinder v2 API는 Cinder 자체에서 오래전에 deprecated된 legacy API입니다. 이 문서는 Gophercloud `v2.15.0`의 `blockstorage/v2` package를 호출하는 generated 메서드가 v3과 어떻게 다른지만 설명합니다. 같은 동작은 [v3 native 볼륨 호출](../v3/native-volumes.md)을 기준으로 합니다. SDK는 오류에 `resource.OperationError` 문맥만 더하며 경로는 client `ResourceBase`(예: `.../v2/{project}/`) 아래에 붙습니다.

## 볼륨

`service.Volumes`(`blockstorage/v2/volumes`)의 생성·조회·목록·수정·삭제·`WaitForStatus`와 사용자 action 13개(attach·begin detaching·detach·reserve·unreserve·initialize/terminate connection·extend·upload image·image metadata·bootable·retype·reimage)는 v3과 같은 경로·본문·고정 status·decode 규칙을 따릅니다. 차이는 다음과 같습니다.

- `CreateOpts.Size`는 omitempty가 없어 0도 `"size": 0`으로 보내며, 필수 tag가 있어도 native 검사는 정수 0을 거부하지 않습니다. v3은 0이면 생략합니다.
- v2 `CreateOpts`에는 `BackupID`가 없고, 응답 `Volume`에도 `backup_id`·`volume_image_metadata` 필드가 없습니다.
- v2 `ListOpts`에는 `Bootable` 필터가 없습니다.
- `Unmanage`는 v2에 없습니다. force delete와 reset status는 v3과 같이 기본 정책상 관리자 호출이라 이 문서에서 다루지 않습니다.

## snapshot

`service.Snapshots`(`blockstorage/v2/snapshots`)의 `Create`·`Get`·`Delete`는 [v3 snapshot 호출](../v3/native-snapshots.md)과 같은 경로·202/200/202·204 status·필수 volume ID·nil envelope decode를 따릅니다. v2에는 `Update`·`ListDetail`이 없고, 응답에 진행률·프로젝트 같은 `os-extended-snapshot-attributes:` 필드가 없습니다.

v2 `List`는 요약 경로 `snapshots`를 **한 페이지로만** 읽습니다. page가 `SinglePageBase`라 `snapshots_links`를 따라가지 않으며, `ListOpts`에도 `Limit`·`Marker`·`Sort` 같은 paging 필드가 없습니다.

`WaitForStatus`는 v3과 같은 native 구현이라 `snapshot` key가 없는 응답에서 panic합니다. SDK가 막지 않으므로 이 판정은 미완료이고, SDK의 `WaitForState`·`WaitForAvailable`을 사용합니다.

## backup·transfer·availability zone

`service.Backups`(`blockstorage/v2/backups`)의 `Create`·`Get`·`List`·`ListDetail`·`RestoreFromBackup`·`Delete`, `service.Transfers`의 다섯 호출, `service.AvailabilityZones.List`는 v3과 같은 Gophercloud 구현을 v2 경로로 호출합니다. 계약은 [v3 backup 호출](../v3/native-backups.md)과 [v3 transfer·availability zone 호출](../v3/native-transfers.md)을 따르며, availability zone 목록도 v3과 같이 한 페이지를 바로 추출합니다.

v2 backup `Update`는 v3과 다르게 SDK가 본문을 보정하지 않아 native의 envelope 없는 본문을 그대로 보냅니다. Cinder는 backup 수정을 microversion 3.9의 v3 API에서 추가했으므로 v2에서는 이 호출을 사용할 수 없고, 판정도 미완료로 둡니다.
