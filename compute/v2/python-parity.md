# Nova v2 Python proxy 대응: 서버 호출과 권한 조건

이 문서는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [compute v2 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py) 가운데 서버 조회·수정·삭제와 사용자 action, server metadata·tag·IP·security group, server group, interface, volume attachment, instance action, Nova image proxy 메서드 63개를 Go 호출과 비교합니다. Python 요청은 [Proxy 공통 helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py), [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py), [Server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server.py)의 action 메서드, [MetadataMixin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/metadata.py)과 [TagMixin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/tag.py)에서 따라갔고, 기존 `python_parity_test.go`와 아래 개별 가이드가 참조하는 계약 테스트가 각 Go 요청의 비교 기준입니다.

`go_mapping`은 고정한 named Python 호출의 HTTP 동작·기본값·옵션에 대응하는 공개 Go API가 있다는 뜻입니다. 이때도 아래 절의 차이는 남습니다. `unresolved`는 Go에 경로, 본문 key나 helper 동작이 없어서 같은 요청을 만들 수 없다는 뜻입니다. 표의 Go 호출은 `conn.ComputeV2(ctx)`가 돌려준 service의 필드를 기준으로 적었고, Glance 호출은 `conn.ImageV2(ctx)`의 `Images`입니다. `backup_server` workflow만 `conn.Compute(ctx)`의 cloud service를 사용합니다. 표는 공개 API의 구현 대응을 설명하며 전체 검증 완료나 Python Resource/session 전체 동등성을 뜻하지 않습니다. 최종 실행 결과는 [검증 기록](../../docs/sdk-support-ledger.md)에서 별도로 확인합니다.

| Python 메서드 | Go 호출 | 판정 |
|---|---|---|
| `get_server` | `Servers.Get(ctx, id)` | go_mapping |
| `update_server` | `Servers.Update(ctx, id, opts, WithUpdateField)` | go_mapping |
| `delete_server` | `Servers.Remove(ctx, resource.ID(id))`, `Servers.ForceDelete(ctx, id)` | go_mapping |
| `change_server_password` | `Servers.ChangeAdminPassword(ctx, id, password)` | go_mapping |
| `clear_server_password` | `Servers.ClearPassword(ctx, id)` ([가이드](servers/python-user-actions.md)) | go_mapping |
| `reboot_server` | `Servers.Reboot(ctx, id, RebootOpts{Type})` | go_mapping |
| `rebuild_server` | `Servers.Rebuild(ctx, id, opts, WithRebuildField)` | go_mapping |
| `resize_server` | `Servers.Resize(ctx, id, ResizeOpts{FlavorRef})` | go_mapping |
| `confirm_server_resize` | `Servers.ConfirmResize(ctx, id)` | go_mapping |
| `revert_server_resize` | `Servers.RevertResize(ctx, id)` | go_mapping |
| `create_server_image` | `Servers.CreateImage`, `Images.FindIdentity`, `Images.WaitFor` | go_mapping |
| `backup_server` | `conn.Compute(ctx)`의 `BackupServer(ctx, id, request, options...)` ([가이드](../backup-server-python.md)) | go_mapping |
| `pause_server` | `Servers.Pause(ctx, id)` | go_mapping |
| `unpause_server` | `Servers.Unpause(ctx, id)` | go_mapping |
| `suspend_server` | `Servers.Suspend(ctx, id)` | go_mapping |
| `resume_server` | `Servers.Resume(ctx, id)` | go_mapping |
| `lock_server` | `Servers.LockWithOptions(ctx, id, servers.WithLockedReason(reason))` ([가이드](servers/python-user-actions.md)) | go_mapping |
| `unlock_server` | `Servers.Unlock(ctx, id)` | go_mapping |
| `rescue_server` | `Servers.Rescue(ctx, id, RescueOpts{...})` | go_mapping |
| `unrescue_server` | `Servers.Unrescue(ctx, id)` | go_mapping |
| `start_server` | `Servers.Start(ctx, id)` | go_mapping |
| `stop_server` | `Servers.Stop(ctx, id)` | go_mapping |
| `restore_server` | `Servers.Restore(ctx, id)` ([가이드](servers/python-user-actions.md)) | go_mapping |
| `shelve_server` | `Servers.Shelve(ctx, id)` | go_mapping |
| `shelve_offload_server` | `Servers.ShelveOffload(ctx, id)` | go_mapping |
| `unshelve_server` | `Servers.UnshelveWithOptions(ctx, id, options...)` ([가이드](servers/python-user-actions.md)) | go_mapping |
| `set_server_metadata` | `Servers.UpdateMetadata(ctx, id, MetadataOpts)` | go_mapping |
| `delete_server_metadata` | key마다 `Servers.DeleteMetadatum`, keys가 없으면 `Servers.ResetMetadata(ctx, id, MetadataOpts{})` | go_mapping |
| `server_ips` | `Servers.ListAddresses(ctx, id)`, `Servers.ListAddressesByNetwork(ctx, id, label)` | go_mapping |
| `add_tag_to_server` | `Tags.InServer(ctx, resource.ID(id))`의 `Add(ctx, tag)` | go_mapping |
| `remove_tag_from_server` | scope의 `Remove(ctx, tag, tags.WithMissingError())` | go_mapping |
| `remove_tags_from_server` | scope의 `RemoveAll(ctx, tags.WithMissingError())` | go_mapping |
| `fetch_server_security_groups` | `SecurityGroups.ListByServer(ctx, id)` | go_mapping |
| `add_security_group_to_server` | `SecurityGroups.AddServer(ctx, id, name)` | go_mapping |
| `remove_security_group_from_server` | `SecurityGroups.RemoveServer(ctx, id, name)` | go_mapping |
| `add_fixed_ip_to_server` | `Servers.AddFixedIP(ctx, id, networkID)` ([가이드](servers/python-user-actions.md)) | go_mapping |
| `remove_fixed_ip_from_server` | `Servers.RemoveFixedIP(ctx, id, address)` ([가이드](servers/python-user-actions.md)) | go_mapping |
| `add_floating_ip_to_server` | `Servers.AddFloatingIP(ctx, id, address, options...)` ([가이드](servers/python-user-actions.md)) | go_mapping |
| `remove_floating_ip_from_server` | `Servers.RemoveFloatingIP(ctx, id, address)` ([가이드](servers/python-user-actions.md)) | go_mapping |
| `create_server_group` | `ServerGroups.Create(ctx, opts, WithCreateField)` | go_mapping |
| `get_server_group` | `ServerGroups.Get(ctx, id)` | go_mapping |
| `delete_server_group` | `ServerGroups.Remove(ctx, resource.ID(id))` | go_mapping |
| `server_groups` | `ServerGroups.List(ctx, WithListOptions, WithListQuery)` | go_mapping |
| `find_server_group` | `ServerGroups.FindRecord(ctx, nameOrID, options...)` ([가이드](python-attachments-groups.md)) | go_mapping |
| `create_server_interface` | `AttachInterfaces.Create(ctx, serverID, opts, WithCreateField)` | go_mapping |
| `get_server_interface` | `AttachInterfaces.Get(ctx, serverID, portID)` | go_mapping |
| `server_interfaces` | `AttachInterfaces.List(ctx, serverID)` | go_mapping |
| `delete_server_interface` | `AttachInterfaces.InServer(ctx, resource.ID(serverID))`의 `Delete(ctx, resource.ID(portID))` | go_mapping |
| `create_volume_attachment` | `VolumeAttachments.Create(ctx, serverID, opts)` | go_mapping |
| `get_volume_attachment` | `VolumeAttachments.Get(ctx, serverID, volumeID)` | go_mapping |
| `volume_attachments` | `VolumeAttachments.ListRecords(ctx, serverID, options...)` ([가이드](python-attachments-groups.md)) | go_mapping |
| `update_volume_attachment` | `VolumeAttachments.UpdateRecord(ctx, serverID, volumeID, options...)` ([가이드](python-attachments-groups.md)) | go_mapping |
| `delete_volume_attachment` | `VolumeAttachments.InServer(ctx, resource.ID(serverID))`의 `Delete(ctx, resource.ID(volumeID))` | go_mapping |
| `get_server_action` | `InstanceActions.InServer(ctx, resource.ID(serverID))`의 `Get(ctx, requestID)` | go_mapping |
| `server_actions` | `InstanceActions.ProxyListRecords(ctx, serverID, options...)` ([가이드](instanceactions/python-records.md)) | go_mapping |
| `get_image` | `Images.GetRecord` ([legacy 버전·사용법](nova-images-python.md)) | go_mapping |
| `find_image` | `Images.FindRecord` ([legacy 버전·사용법](nova-images-python.md)) | go_mapping |
| `images` | `Images.ListRecords` ([legacy 버전·사용법](nova-images-python.md)) | go_mapping |
| `delete_image` | `Images.DeleteRecord` ([legacy 버전·사용법](nova-images-python.md)) | go_mapping |
| `fetch_image_metadata` | `Images.FetchMetadata` ([legacy 버전·사용법](nova-images-python.md)) | go_mapping |
| `get_image_metadata` | `Images.GetMetadata` ([legacy 버전·사용법](nova-images-python.md)) | go_mapping |
| `set_image_metadata` | `Images.SetMetadata` ([legacy 버전·사용법](nova-images-python.md)) | go_mapping |
| `delete_image_metadata` | `Images.DeleteMetadata` ([legacy 버전·사용법](nova-images-python.md)) | go_mapping |

## 공통 차이

Python은 Resource 객체나 ID 문자열을 받지만 Go generated 호출은 ID 문자열을 받고, `Remove`, `Find`, `InServer`, scope의 `Delete`는 `resource.Ref`를 받습니다. generated 결과는 Python Resource 대신 Gophercloud typed model이라서 모델에 없는 응답 field는 남지 않습니다. 예를 들어 `attachinterfaces.Interface`에는 2.70의 `tag`가 없고, native `volumeattach.VolumeAttachment`에는 2.89의 `attachment_id`와 `bdm_uuid`가 없습니다. 새 attachment/group `Record` API는 선언된 Python 속성의 `Resource`와 unknown field를 포함한 `Wire`, 실제 `Envelope`·`Header`·`StatusCode`를 제공합니다. mutable Resource, dirty/cache/session hooks와 전체 cloud location은 이 record의 범위에 포함되지 않습니다.

Python Server action과 Server·ServerGroup·ServerInterface·VolumeAttachment·ServerAction 요청은 resource의 `_max_microversion`(Server 2.100, ServerGroup 2.64, ServerInterface 2.70, VolumeAttachment 2.89, ServerAction 2.84)과 서버 최대값 가운데 작은 값을 골라 보냅니다. 기존 generated Go 호출은 Connection에 설정하거나 협상한 client microversion을 씁니다. 같은 header가 필요하면 `RawClient()`의 복사본에 `Microversion`을 바꿔 새 API를 만듭니다. 새 attachment/group record는 명시된 version을 유지하고 미선택 시 공통 discovery로 각각 2.89·2.64 ceiling을 적용합니다. `server_actions`의 owned 목록도 iterator를 소비할 때만 discovery를 수행하고 2.84 ceiling을 적용하며, Proxy opaque JMESPath 표현식은 HTTP/discovery 없이 generator 값으로 평가합니다. `BackupServer`는 Server ceiling 2.100으로 요청별 선택을 하고, 직접 사용자 action 8개는 선택한 version을 바꾸거나 추가 discovery를 하지 않습니다. MetadataMixin, `fetch_server_security_groups`, `server_ips`는 Python이 microversion을 지정하지 않는 요청이고 Go는 이때도 client microversion을 보냅니다.

Python delete 계열은 `ignore_missing=True`가 기본이라 404를 무시합니다. Go에서는 `Remove(ctx, resource.ID(id))`나 scope의 `Delete(ctx, resource.ID(id))`가 같은 기본값을 갖고 `resource.WithMissingError()`가 `ignore_missing=False`에 해당합니다. generated `Delete`를 직접 부르면 404가 오류로 돌아옵니다.

Python `_action`은 `Accept` header를 비워 보내고 400 미만의 모든 status를 성공으로 봅니다. 기존 generated Go action은 Gophercloud 기본 Accept와 action별 성공 status(대부분 201, 202)를 씁니다. 새 직접 action과 `CreateBackupAction`의 POST는 Source의 빈 Accept를 SDK가 소유하고 최종 200–399를 허용하며, 실제 거부 status·body·headers와 물리 read/Close 원인을 보존합니다. Python update는 바뀐 attribute가 없으면 요청을 보내지 않지만 Go `Update`는 빈 envelope라도 PUT을 보냅니다.

## 서버 조회·수정·삭제

`get_server`는 같은 `GET servers/{id}`를 보냅니다. `update_server`는 Python이 넘긴 attribute만 `{"server": {...}}`에 담고, Go `UpdateOpts`의 이름·access IP·hostname 밖의 key(예: `description`)는 `WithUpdateField`로 넣습니다. `UpdateOpts`는 빈 문자열을 생략하므로 빈 이름처럼 빈 값을 보내는 Python 호출과는 본문이 다릅니다.

`delete_server`는 `Remove`가 404를 무시하며 같은 DELETE를 보냅니다. `force=True`는 Python이 `ignore_missing`을 적용하지 않고 `{"forceDelete": null}`을 보내는 경로입니다. Go `ForceDelete`는 같은 action을 `{"forceDelete": ""}`로 보내고 404를 오류로 돌려줍니다.

## 서버 action

`change_server_password`, `reboot_server`, `resize_server`, `confirm_server_resize`, `revert_server_resize`, `pause_server`, `unpause_server`, `suspend_server`, `resume_server`, `unlock_server`, `unrescue_server`, `start_server`, `stop_server`, `shelve_server`, `shelve_offload_server`는 같은 `POST servers/{id}/action` 본문을 보냅니다. `reboot_server`의 문자열은 `servers.RebootMethod("HARD")`처럼 그대로 바꿔 넣고, `resize_server`의 Flavor 객체는 ID만 `FlavorRef`에 줍니다. `rescue_server`는 인자가 없으면 두 쪽 모두 `{"rescue": {}}`를 보내고, `image`는 `RescueImageRef`, `admin_pass`는 `AdminPass`가 됩니다. Python은 None을 돌려주지만 Go는 응답의 `adminPass`를 돌려줍니다.

`rebuild_server`는 Python이 넘긴 key만 보내며 Go도 `RebuildOpts`의 빈 값을 생략합니다. `preserve_ephemeral`, `user_data`, `key_name`, `description`, `trusted_image_certificates`, `hostname`은 `WithRebuildField`로 action 안에 넣고, `key_name=None`처럼 null로 지우는 값도 그대로 보낼 수 있습니다. 다만 `name`, `adminPass`, `accessIPv4`, `accessIPv6`, `metadata`는 typed field라서 확장 field로 덮을 수 없으므로 이 key에 null이나 빈 문자열을 보낼 수 없습니다. Nova는 이 key의 null을 받지 않으므로 실제로 쓰는 인자는 모두 표현됩니다. 결과는 두 쪽 모두 응답의 `server`입니다.

`create_server_image`는 Python이 서버가 2.45를 지원하면 그 version을 고정해 `createImage`를 보내고, 응답 본문의 `image_id`나 `Location` header에서 ID를 꺼냅니다. Go `CreateImage`는 client microversion을 보내고 응답의 version header로 같은 위치를 고릅니다. 그 뒤 Python은 Glance에서 `find_image(image_id)`로 image를 읽으므로 Go도 `Images.FindIdentity(ctx, id)`를 부릅니다. `wait=True`는 `Images.WaitFor(ctx, resource.ID(id), "active", resource.WithTimeout(120*time.Second), resource.WithFailureStates("error"))`에 해당합니다. Python은 조회 결과가 없으면 계속 기다리지만 Go는 404를 바로 오류로 돌려주고, Python의 시간 초과는 `ResourceTimeout`이며 Go는 context deadline 오류입니다.

`UnshelveWithOptions`는 기본 `{"unshelve": null}`, host 단독·가용 영역 문자열·명시적 null을 action 안에 보냅니다. `WithUnshelveHost`는 2.91 이상, `WithUnshelveAvailabilityZone`는 2.77 이상, `WithUnshelveUnpinnedAvailabilityZone`는 2.91 이상에서 사용합니다. `LockWithOptions`는 기본 `{"lock": null}`을 유지하고 `WithLockedReason`으로 2.73 이상의 `locked_reason`을 보냅니다. 생략과 빈 사유를 구분합니다. generated `Unshelve`·`Lock`의 기존 입력 한계는 유지되므로 이 경우 새 API를 사용합니다. [직접 action 8개의 옵션·버전·오류 비교](servers/python-user-actions.md)에 사용법을 설명합니다.

`ClearPassword`는 Nova에 저장된 암호를 bodyless DELETE로 지우며 guest 암호를 바꾸지 않습니다. `Restore`는 `{"restore": null}`을 보내고 상태 변경을 기다리지 않습니다. 네 직접 fixed/floating IP action은 목록 조회 없이 요청을 보내며 2.1–2.43에 적용됩니다. 2.44 이상은 HTTP 전에 거부합니다. `compute.Service.AddIPList`의 legacy inventory 조회 workflow와 경로가 다릅니다.

`backup_server`는 cloud `BackupServer(ctx, id, compute.BackupServerRequest{...}, options...)`로 대응합니다. SDK가 `createBackup` action, image ID 추출, Glance cloud lookup과 선택적 active 대기를 묶습니다. 기본 wait=false·timeout=120초이며 Nova action 수락 증거와 Glance 조회/대기 결과를 따로 반환합니다. 하위 action만 필요하면 `Servers.CreateBackupAction`을 사용합니다. [Python/Go workflow 비교](../backup-server-python.md)에 connection의 Glance binding, 직접 client 옵션과 부분 결과를 설명합니다.

## 서버 metadata와 IP

`set_server_metadata`는 `POST servers/{id}/metadata`로 병합하며 Go `UpdateMetadata`가 같은 본문을 보냅니다. Python은 넘긴 key만 담은 Server를 돌려주고 Go는 응답의 병합된 metadata 전체를 돌려줍니다. Go `MetadataOpts`는 문자열 값만 받습니다. `delete_server_metadata(keys=...)`는 key마다 DELETE를 보내므로 Go도 `DeleteMetadatum`을 반복하고, keys를 주지 않으면 `PUT servers/{id}/metadata`에 `{"metadata": {}}`를 보내므로 `ResetMetadata(ctx, id, MetadataOpts{})`를 씁니다. nil map을 넘기면 Go는 `"metadata": null`을 보내므로 빈 map을 넘겨야 합니다.

`server_ips`는 label이 없으면 `GET servers/{id}/ips`, 있으면 `GET servers/{id}/ips/{label}`을 보냅니다. Python은 주소마다 `network_label`, `address`, `version`을 가진 ServerIP를 만들고, Go는 label별 map 하나 또는 label 없는 `Address` 행을 돌려줍니다. Python은 이 요청의 HTTP status를 검사하지 않아서 404 응답이 KeyError나 TypeError로 드러나지만, Go는 Gophercloud HTTP 오류를 그대로 돌려줍니다.

## Tag와 security group

Python TagMixin은 tag를 URL에 그대로 이어 붙이고 discovery로 2.26 이상을 보냅니다. Go `tags.API.InServer`는 client microversion이 2.26 이상이어야 하고, tag를 path escape하며 쉼표나 slash가 든 tag는 HTTP 전에 거부합니다. `remove_tag_from_server`와 `remove_tags_from_server`는 404를 예외로 올리므로 Go에서는 `tags.WithMissingError()`를 줘야 같습니다. Go 기본값은 404를 무시합니다.

`fetch_server_security_groups`는 `GET servers/{id}/os-security-groups`이며 Go `ListByServer`가 같은 경로를 읽습니다. Python은 Server의 `security_groups`에 응답 dict를 넣고 Go는 typed `SecurityGroup`을 하나씩 돌려줍니다. `add_security_group_to_server`와 `remove_security_group_from_server`는 문자열을 조회 없이 group 이름으로 보내고, SecurityGroup 객체는 `name`이 있으면 이름을, 없으면 ID를 씁니다. Go `AddServer`와 `RemoveServer`에는 그 문자열을 직접 넘깁니다.

## Server group

`create_server_group`은 Python이 2.64 이상이면 `policies`의 첫 값을 `policy`로 바꾸고, 그보다 낮으면 `policy`를 `policies`로 바꾸며 `rules`를 거부합니다. Go는 이 변환을 하지 않으므로 caller가 client microversion에 맞춰 `Policy`와 `Rules` 또는 `Policies`를 고릅니다. `CreateOpts` 밖의 key는 `WithCreateField`로 넣습니다. `get_server_group`과 `delete_server_group`은 같은 요청을 보내며 `Remove`가 404를 무시합니다.

`server_groups(all_projects=True)`는 Python이 `all_projects=True`를 보냅니다. Go typed `AllProjects`는 `all_projects=true`를 보내므로 같은 문자열이 필요하면 `WithListQuery("all_projects", "True")`를 쓰고, `limit`은 `ListOpts.Limit`, `marker`는 `WithListQuery`로 보냅니다. Go 목록은 한 페이지만 읽고, Python은 `limit`을 채운 페이지 뒤에 marker로 다음 페이지를 더 요청할 수 있습니다.

`ServerGroups.FindRecord`는 literal ID GET부터 시작하고 깨끗한 400·403·404일 때 전체 목록에서 정확한 ID 또는 이름을 찾습니다. 기본 미존재는 nil,nil이며 `WithGroupFindIgnoreMissing(false)`로 오류를 선택합니다. `WithGroupFindAllProjects(true)`는 fallback 목록에만 `all_projects=True`를 전송합니다. 뒤 페이지 오류와 중복을 확인하며 물리 read/Close·취소·source 변경이 섞인 GET 오류로는 fallback하지 않습니다. [attachment/group 비교](python-attachments-groups.md)에 옵션과 raw record 차이를 설명합니다. 기존 `Find(resource.Ref)`의 분리된 검색 계약은 그대로 유지됩니다.

## Interface와 volume attachment

`create_server_interface`는 `port_id`, `net_id`, `fixed_ips`를 typed field로, 2.49의 `tag`를 `WithCreateField("tag", ...)`로 보냅니다. `get_server_interface`, `server_interfaces`는 같은 경로를 읽습니다. Python은 `server_interfaces`에 `limit`과 `marker` query를 받지만 Nova의 interface 목록은 이를 쓰지 않으며, Go `List`는 query 없이 한 번 요청합니다. `delete_server_interface`는 `InServer(ctx, resource.ID(serverID))`가 돌려준 scope의 `Delete(ctx, resource.ID(portID))`가 404를 무시합니다. Python은 interface 객체의 `server_id`로 서버를 정할 수 있고 Go는 서버 ID를 따로 받습니다.

`create_volume_attachment`는 `volumeId`, `device`, `tag`, `delete_on_termination`을 같은 envelope로 보냅니다. Go는 false인 `delete_on_termination`을 생략하며 서버 기본값은 false입니다. Python의 legacy `volume_id`, `volumeId` 인자는 Go `CreateOpts.VolumeID`에 해당합니다. `delete_volume_attachment`는 인자 순서 확인을 위해 서버 ID로 `find_server`를 먼저 부르므로 `GET servers/{id}`가 한 번 더 나갑니다. 그 조회가 실패하면 Python은 두 인자를 바꿔 삭제하지만 Go scope의 `Delete`는 DELETE 하나만 보내고 인자를 바꾸지 않습니다.

`VolumeAttachments.ListRecords`는 고정 server의 GET에 `WithAttachmentLimit`·`WithAttachmentOffset`을 보내고 lazy page 순회·Body 필터·필터 전 raw cap을 제공합니다. native `List`와 scope의 기존 query 제약은 유지됩니다. `UpdateRecord`는 기존 volume ID를 PUT 경로에 쓰고 기본 `volumeId`도 그 ID로 유지합니다. `WithAttachmentAttribute("delete_on_termination", false)`는 명시 false를, `WithAttachmentSwapTo`는 교체 volume ID를 보냅니다. `InServer` scope에서도 `ListRecords`·`UpdateRecord`를 사용합니다. [별도 가이드](python-attachments-groups.md)에 omission/null, immutable route, discovery·반환 seed·실제 HTTP 증거를 설명합니다.

## Instance action

`server_actions`는 owned `ListRecords/AllRecords/ProxyListRecords`로 lazy 페이지·query·parent URI·Source event descriptor를 처리합니다. object event는 Resource 값이라 plain JSON dict와 같지 않으며 nonobject event는 plain `{}`가 됩니다. version 선택, 실제 Wire/page 증거와 재시도 policy는 [목록 사용 비교](instanceactions/python-records.md)에 설명합니다. 조회 정책은 instance project의 reader 또는 admin이고, 목록이 event detail을 자동 조회하지 않습니다.

`get_server_action`은 `GET servers/{id}/os-instance-actions/{request_id}`이며 Go `InstanceActions.InServer` scope의 `Get`이 같은 요청을 보내고 2.84 event의 `details`까지 `ActionEvent`에 남깁니다. Python의 `ignore_missing` 인자는 `_get`에 쓰이지 않아 404가 그대로 예외가 되고, Go도 `resource.ErrNotFound`를 돌려줍니다.

## 권한 조건과 남은 공통 한계

이번에 추가한 action 8개, attachment/group 3개와 backup workflow 1개는 위의 공개 API로 대응합니다. 이 12개를 모두 순수 user API로 세지는 않습니다. [Nova 2025.2 기본 정책](https://docs.openstack.org/nova/2025.2/configuration/policy.html)과 [32.0.0 attachment controller](https://github.com/openstack/nova/blob/32.0.0/nova/api/openstack/compute/volume_attachments.py#L319-L347)를 기준으로 다음 조건을 구분합니다.

- 직접 action의 일반 경로와 `createBackup`은 `project_member_or_admin`, attachment 목록과 일반 group GET/list는 `project_reader_or_admin`입니다. 실제 Glance lookup은 해당 Image 서비스 정책도 적용받습니다.
- 명시한 nonempty unshelve host는 2.91 이상에서 추가 `unshelve_to_host` 관리자 정책이 필요합니다. zone-only 또는 지원 버전의 null-zone 요청은 일반 unshelve 정책을 사용합니다.
- `UpdateRecord`는 2.85 이상에서 같은 volumeId의 일반 속성을 변경할 때 user 경로입니다. 2.85 미만 또는 다른 volumeId로 교체하면 `service_or_admin` swap 경로이며, stock controller는 Cinder migration 용도로 제한합니다.
- group Find의 `all_projects=True` fallback은 관리자 정책을 사용합니다. 첫 direct GET까지 관리자 목록으로 바꾸지는 않습니다.

서버 정책의 기준 release는 SDK source pin과 별개이며 실제 배포의 override·ownership·상태 검사는 서버가 판단합니다. 표의 mapping은 named 호출의 대응이며 mutable Python Resource/session, 전체 cloud location, 모든 action 및 인증된 실제 클라우드 acceptance 동등성은 별도 공통 범위입니다. 구현별 차이와 계약 테스트 참조는 [사용자 action](servers/python-user-actions.md), [attachment/group](python-attachments-groups.md), [backup workflow](../backup-server-python.md)에 남깁니다.

Nova legacy `/images`와 metadata 8개는 [별도 비교 가이드](nova-images-python.md)의 owned API로 제공합니다. 일반 2.35·metadata 2.38 버전 상한과 raw Resource/응답·Source 차이는 해당 프로파일에 적용됩니다. Find fallback은 Source의 `/images?name=...` summary 경로입니다.
