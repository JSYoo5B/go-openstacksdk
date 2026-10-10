# Cinder v2 native 호출

Cinder v2 API는 Cinder 자체에서 오래전에 deprecated된 legacy API입니다. 이 문서는 Gophercloud `v2.15.0`의 `blockstorage/v2` package를 호출하는 generated 메서드가 v3과 어떻게 다른지만 설명합니다. 같은 동작은 [v3 native 볼륨 호출](../v3/native-volumes.md)을 기준으로 합니다. SDK는 오류에 `resource.OperationError` 문맥만 더하며 경로는 client `ResourceBase`(예: `.../v2/{project}/`) 아래에 붙습니다.

## 볼륨

`service.Volumes`(`blockstorage/v2/volumes`)의 생성·조회·목록·수정·삭제·`WaitForStatus`와 사용자 action 13개(attach·begin detaching·detach·reserve·unreserve·initialize/terminate connection·extend·upload image·image metadata·bootable·retype·reimage)는 v3과 같은 경로·본문·고정 status·decode 규칙을 따릅니다. 차이는 다음과 같습니다.

- `CreateOpts.Size`는 omitempty가 없어 0도 `"size": 0`으로 보내며, 필수 tag가 있어도 native 검사는 정수 0을 거부하지 않습니다. v3은 0이면 생략합니다.
- v2 `CreateOpts`에는 `BackupID`가 없고, 응답 `Volume`에도 `backup_id`·`volume_image_metadata` 필드가 없습니다.
- v2 `ListOpts`에는 `Bootable` 필터가 없습니다.
- `Unmanage`는 v2에 없습니다. force delete와 reset status는 v3과 같이 기본 정책상 관리자 호출이라 이 문서에서 다루지 않습니다.
