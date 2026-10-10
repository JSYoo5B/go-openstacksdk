# Nova v2 Python proxy 대응: 서버 사용자 호출

이 문서는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [compute v2 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py) 가운데 서버 조회·수정·삭제와 사용자 action, server metadata·tag·IP·security group, server group, interface, volume attachment, instance action, Nova image proxy 메서드 62개를 Go 호출과 비교합니다. Python 요청은 [Proxy 공통 helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py), [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py), [Server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server.py)의 action 메서드, [MetadataMixin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/metadata.py)과 [TagMixin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/tag.py)에서 따라갔고, 각 package의 `python_parity_test.go`가 Go 쪽 요청을 고정합니다.

`go_mapping`은 모든 Python 인자의 효과를 공개 Go API로 재현할 수 있다는 뜻입니다. 이때도 아래 절의 차이는 남습니다. `unresolved`는 Go에 경로, 본문 key나 helper 동작이 없어서 같은 요청을 만들 수 없다는 뜻입니다. 표의 Go 호출은 `conn.ComputeV2(ctx)`가 돌려준 service의 필드를 기준으로 적었고, Glance 호출은 `conn.ImageV2(ctx)`의 `Images`입니다.

| Python 메서드 | Go 호출 | 판정 |
|---|---|---|
| `get_server` | `Servers.Get(ctx, id)` | go_mapping |
| `update_server` | `Servers.Update(ctx, id, opts, WithUpdateField)` | go_mapping |
| `delete_server` | `Servers.Remove(ctx, resource.ID(id))`, `Servers.ForceDelete(ctx, id)` | go_mapping |
| `change_server_password` | `Servers.ChangeAdminPassword(ctx, id, password)` | go_mapping |
| `clear_server_password` | 없음 | unresolved |
| `reboot_server` | `Servers.Reboot(ctx, id, RebootOpts{Type})` | go_mapping |
| `rebuild_server` | `Servers.Rebuild(ctx, id, opts, WithRebuildField)` | go_mapping |
| `resize_server` | `Servers.Resize(ctx, id, ResizeOpts{FlavorRef})` | go_mapping |
| `confirm_server_resize` | `Servers.ConfirmResize(ctx, id)` | go_mapping |
| `revert_server_resize` | `Servers.RevertResize(ctx, id)` | go_mapping |
| `create_server_image` | `Servers.CreateImage`, `Images.FindIdentity`, `Images.WaitFor` | go_mapping |
| `backup_server` | 없음 | unresolved |
| `pause_server` | `Servers.Pause(ctx, id)` | go_mapping |
| `unpause_server` | `Servers.Unpause(ctx, id)` | go_mapping |
| `suspend_server` | `Servers.Suspend(ctx, id)` | go_mapping |
| `resume_server` | `Servers.Resume(ctx, id)` | go_mapping |
| `lock_server` | `Servers.Lock(ctx, id)` (사유 없이만 가능) | unresolved |
| `unlock_server` | `Servers.Unlock(ctx, id)` | go_mapping |
| `rescue_server` | `Servers.Rescue(ctx, id, RescueOpts{...})` | go_mapping |
| `unrescue_server` | `Servers.Unrescue(ctx, id)` | go_mapping |
| `start_server` | `Servers.Start(ctx, id)` | go_mapping |
| `stop_server` | `Servers.Stop(ctx, id)` | go_mapping |
| `restore_server` | 없음 | unresolved |
| `shelve_server` | `Servers.Shelve(ctx, id)` | go_mapping |
| `shelve_offload_server` | `Servers.ShelveOffload(ctx, id)` | go_mapping |
| `unshelve_server` | `Servers.Unshelve(ctx, id, opts, WithUnshelveField)` (일부 조합만) | unresolved |
| `set_server_metadata` | `Servers.UpdateMetadata(ctx, id, MetadataOpts)` | go_mapping |
| `delete_server_metadata` | key마다 `Servers.DeleteMetadatum`, keys가 없으면 `Servers.ResetMetadata(ctx, id, MetadataOpts{})` | go_mapping |
| `server_ips` | `Servers.ListAddresses(ctx, id)`, `Servers.ListAddressesByNetwork(ctx, id, label)` | go_mapping |
| `add_tag_to_server` | `Tags.InServer(ctx, resource.ID(id))`의 `Add(ctx, tag)` | go_mapping |
| `remove_tag_from_server` | scope의 `Remove(ctx, tag, tags.WithMissingError())` | go_mapping |
| `remove_tags_from_server` | scope의 `RemoveAll(ctx, tags.WithMissingError())` | go_mapping |
| `fetch_server_security_groups` | `SecurityGroups.ListByServer(ctx, id)` | go_mapping |
| `add_security_group_to_server` | `SecurityGroups.AddServer(ctx, id, name)` | go_mapping |
| `remove_security_group_from_server` | `SecurityGroups.RemoveServer(ctx, id, name)` | go_mapping |
| `add_fixed_ip_to_server` | 없음 | unresolved |
| `remove_fixed_ip_from_server` | 없음 | unresolved |
| `add_floating_ip_to_server` | `compute.Service.AddIPList`는 여러 요청을 묶은 다른 workflow | unresolved |
| `remove_floating_ip_from_server` | 없음 | unresolved |
| `create_server_group` | `ServerGroups.Create(ctx, opts, WithCreateField)` | go_mapping |
| `get_server_group` | `ServerGroups.Get(ctx, id)` | go_mapping |
| `delete_server_group` | `ServerGroups.Remove(ctx, resource.ID(id))` | go_mapping |
| `server_groups` | `ServerGroups.List(ctx, WithListOptions, WithListQuery)` | go_mapping |
| `find_server_group` | `ServerGroups.Find(ctx, resource.ID(x))`와 `resource.Name(x)`를 따로 호출 | unresolved |
| `create_server_interface` | `AttachInterfaces.Create(ctx, serverID, opts, WithCreateField)` | go_mapping |
| `get_server_interface` | `AttachInterfaces.Get(ctx, serverID, portID)` | go_mapping |
| `server_interfaces` | `AttachInterfaces.List(ctx, serverID)` | go_mapping |
| `delete_server_interface` | `AttachInterfaces.InServer(ctx, resource.ID(serverID))`의 `Delete(ctx, resource.ID(portID))` | go_mapping |
| `create_volume_attachment` | `VolumeAttachments.Create(ctx, serverID, opts)` | go_mapping |
| `get_volume_attachment` | `VolumeAttachments.Get(ctx, serverID, volumeID)` | go_mapping |
| `volume_attachments` | `VolumeAttachments.List(ctx, serverID)` (query 없이만 가능) | unresolved |
| `update_volume_attachment` | 없음 | unresolved |
| `delete_volume_attachment` | `VolumeAttachments.InServer(ctx, resource.ID(serverID))`의 `Delete(ctx, resource.ID(volumeID))` | go_mapping |
| `get_server_action` | `InstanceActions.InServer(ctx, resource.ID(serverID))`의 `Get(ctx, requestID)` | go_mapping |
| `get_image` | 없음 | unresolved |
| `find_image` | 없음 | unresolved |
| `images` | 없음 | unresolved |
| `delete_image` | 없음 | unresolved |
| `fetch_image_metadata` | 없음 | unresolved |
| `get_image_metadata` | 없음 | unresolved |
| `set_image_metadata` | 없음 | unresolved |
| `delete_image_metadata` | 없음 | unresolved |

## 공통 차이

Python은 Resource 객체나 ID 문자열을 받지만 Go generated 호출은 ID 문자열을 받고, `Remove`, `Find`, `InServer`, scope의 `Delete`는 `resource.Ref`를 받습니다. 결과는 Python Resource 대신 Gophercloud typed model이라서 모델에 없는 응답 field는 남지 않습니다. 예를 들어 `attachinterfaces.Interface`에는 2.70의 `tag`가 없고, `volumeattach.VolumeAttachment`에는 2.89의 `attachment_id`와 `bdm_uuid`가 없습니다.

Python Server action과 Server·ServerGroup·ServerInterface·VolumeAttachment·ServerAction 요청은 resource의 `_max_microversion`(Server 2.100, ServerGroup 2.64, ServerInterface 2.70, VolumeAttachment 2.89, ServerAction 2.84)과 서버 최대값 가운데 작은 값을 골라 보냅니다. Go는 Connection에 설정하거나 협상한 client microversion 하나를 모든 호출에 씁니다. 그래서 같은 header가 필요하면 `RawClient()`의 복사본에 `Microversion`을 바꿔 새 API를 만들어야 합니다. MetadataMixin, `fetch_server_security_groups`, `server_ips`는 Python이 microversion을 지정하지 않는 요청이고 Go는 이때도 client microversion을 보냅니다.

Python delete 계열은 `ignore_missing=True`가 기본이라 404를 무시합니다. Go에서는 `Remove(ctx, resource.ID(id))`나 scope의 `Delete(ctx, resource.ID(id))`가 같은 기본값을 갖고 `resource.WithMissingError()`가 `ignore_missing=False`에 해당합니다. generated `Delete`를 직접 부르면 404가 오류로 돌아옵니다.

Python `_action`은 `Accept` header를 비워 보내고 400 미만의 모든 status를 성공으로 봅니다. Go action은 Gophercloud 기본 Accept와 action별 성공 status(대부분 201, 202)를 씁니다. Python update는 바뀐 attribute가 없으면 요청을 보내지 않지만 Go `Update`는 빈 envelope라도 PUT을 보냅니다.

## 서버 조회·수정·삭제

`get_server`는 같은 `GET servers/{id}`를 보냅니다. `update_server`는 Python이 넘긴 attribute만 `{"server": {...}}`에 담고, Go `UpdateOpts`의 이름·access IP·hostname 밖의 key(예: `description`)는 `WithUpdateField`로 넣습니다. `UpdateOpts`는 빈 문자열을 생략하므로 빈 이름처럼 빈 값을 보내는 Python 호출과는 본문이 다릅니다.

`delete_server`는 `Remove`가 404를 무시하며 같은 DELETE를 보냅니다. `force=True`는 Python이 `ignore_missing`을 적용하지 않고 `{"forceDelete": null}`을 보내는 경로입니다. Go `ForceDelete`는 같은 action을 `{"forceDelete": ""}`로 보내고 404를 오류로 돌려줍니다.

## 서버 action

`change_server_password`, `reboot_server`, `resize_server`, `confirm_server_resize`, `revert_server_resize`, `pause_server`, `unpause_server`, `suspend_server`, `resume_server`, `unlock_server`, `unrescue_server`, `start_server`, `stop_server`, `shelve_server`, `shelve_offload_server`는 같은 `POST servers/{id}/action` 본문을 보냅니다. `reboot_server`의 문자열은 `servers.RebootMethod("HARD")`처럼 그대로 바꿔 넣고, `resize_server`의 Flavor 객체는 ID만 `FlavorRef`에 줍니다. `rescue_server`는 인자가 없으면 두 쪽 모두 `{"rescue": {}}`를 보내고, `image`는 `RescueImageRef`, `admin_pass`는 `AdminPass`가 됩니다. Python은 None을 돌려주지만 Go는 응답의 `adminPass`를 돌려줍니다.

`rebuild_server`는 Python이 넘긴 key만 보내며 Go도 `RebuildOpts`의 빈 값을 생략합니다. `preserve_ephemeral`, `user_data`, `key_name`, `description`, `trusted_image_certificates`, `hostname`은 `WithRebuildField`로 action 안에 넣고, `key_name=None`처럼 null로 지우는 값도 그대로 보낼 수 있습니다. 다만 `name`, `adminPass`, `accessIPv4`, `accessIPv6`, `metadata`는 typed field라서 확장 field로 덮을 수 없으므로 이 key에 null이나 빈 문자열을 보낼 수 없습니다. Nova는 이 key의 null을 받지 않으므로 실제로 쓰는 인자는 모두 표현됩니다. 결과는 두 쪽 모두 응답의 `server`입니다.

`create_server_image`는 Python이 서버가 2.45를 지원하면 그 version을 고정해 `createImage`를 보내고, 응답 본문의 `image_id`나 `Location` header에서 ID를 꺼냅니다. Go `CreateImage`는 client microversion을 보내고 응답의 version header로 같은 위치를 고릅니다. 그 뒤 Python은 Glance에서 `find_image(image_id)`로 image를 읽으므로 Go도 `Images.FindIdentity(ctx, id)`를 부릅니다. `wait=True`는 `Images.WaitFor(ctx, resource.ID(id), "active", resource.WithTimeout(120*time.Second), resource.WithFailureStates("error"))`에 해당합니다. Python은 조회 결과가 없으면 계속 기다리지만 Go는 404를 바로 오류로 돌려주고, Python의 시간 초과는 `ResourceTimeout`이며 Go는 context deadline 오류입니다.

`unshelve_server`는 인자가 없을 때 `{"unshelve": null}`, 가용 영역을 줄 때 `{"unshelve": {"availability_zone": ...}}`를 보내며 Go도 같습니다. 가용 영역과 함께 `host`를 주면 Go도 `WithUnshelveField("host", ...)`로 같은 본문을 만듭니다. 하지만 `host`만 주면 Go action 값이 null이라 `host`가 `unshelve` 옆 최상위에 붙고, `availability_zone=None`으로 고정 영역을 푸는 null은 typed key라서 보낼 수 없습니다. `lock_server(locked_reason=...)`도 Go `Lock`이 옵션을 받지 않아 `{"lock": null}`만 보냅니다.

## 서버 metadata와 IP

`set_server_metadata`는 `POST servers/{id}/metadata`로 병합하며 Go `UpdateMetadata`가 같은 본문을 보냅니다. Python은 넘긴 key만 담은 Server를 돌려주고 Go는 응답의 병합된 metadata 전체를 돌려줍니다. Go `MetadataOpts`는 문자열 값만 받습니다. `delete_server_metadata(keys=...)`는 key마다 DELETE를 보내므로 Go도 `DeleteMetadatum`을 반복하고, keys를 주지 않으면 `PUT servers/{id}/metadata`에 `{"metadata": {}}`를 보내므로 `ResetMetadata(ctx, id, MetadataOpts{})`를 씁니다. nil map을 넘기면 Go는 `"metadata": null`을 보내므로 빈 map을 넘겨야 합니다.

`server_ips`는 label이 없으면 `GET servers/{id}/ips`, 있으면 `GET servers/{id}/ips/{label}`을 보냅니다. Python은 주소마다 `network_label`, `address`, `version`을 가진 ServerIP를 만들고, Go는 label별 map 하나 또는 label 없는 `Address` 행을 돌려줍니다. Python은 이 요청의 HTTP status를 검사하지 않아서 404 응답이 KeyError나 TypeError로 드러나지만, Go는 Gophercloud HTTP 오류를 그대로 돌려줍니다.

## Tag와 security group

Python TagMixin은 tag를 URL에 그대로 이어 붙이고 discovery로 2.26 이상을 보냅니다. Go `tags.API.InServer`는 client microversion이 2.26 이상이어야 하고, tag를 path escape하며 쉼표나 slash가 든 tag는 HTTP 전에 거부합니다. `remove_tag_from_server`와 `remove_tags_from_server`는 404를 예외로 올리므로 Go에서는 `tags.WithMissingError()`를 줘야 같습니다. Go 기본값은 404를 무시합니다.

`fetch_server_security_groups`는 `GET servers/{id}/os-security-groups`이며 Go `ListByServer`가 같은 경로를 읽습니다. Python은 Server의 `security_groups`에 응답 dict를 넣고 Go는 typed `SecurityGroup`을 하나씩 돌려줍니다. `add_security_group_to_server`와 `remove_security_group_from_server`는 문자열을 조회 없이 group 이름으로 보내고, SecurityGroup 객체는 `name`이 있으면 이름을, 없으면 ID를 씁니다. Go `AddServer`와 `RemoveServer`에는 그 문자열을 직접 넘깁니다.

## Server group

`create_server_group`은 Python이 2.64 이상이면 `policies`의 첫 값을 `policy`로 바꾸고, 그보다 낮으면 `policy`를 `policies`로 바꾸며 `rules`를 거부합니다. Go는 이 변환을 하지 않으므로 caller가 client microversion에 맞춰 `Policy`와 `Rules` 또는 `Policies`를 고릅니다. `CreateOpts` 밖의 key는 `WithCreateField`로 넣습니다. `get_server_group`과 `delete_server_group`은 같은 요청을 보내며 `Remove`가 404를 무시합니다.

`server_groups(all_projects=True)`는 Python이 `all_projects=True`를 보냅니다. Go typed `AllProjects`는 `all_projects=true`를 보내므로 같은 문자열이 필요하면 `WithListQuery("all_projects", "True")`를 쓰고, `limit`은 `ListOpts.Limit`, `marker`는 `WithListQuery`로 보냅니다. Go 목록은 한 페이지만 읽고, Python은 `limit`을 채운 페이지 뒤에 marker로 다음 페이지를 더 요청할 수 있습니다.

`find_server_group`은 ID GET 뒤 404·400·403이면 전체 목록에서 ID나 이름이 같은 항목을 하나 고르고, `all_projects=True`를 목록 query로 보냅니다. Go에는 이 fallback을 하는 helper가 없어서 `Find(ctx, resource.ID(x), resource.WithIgnoreMissing())`와 `Find(ctx, resource.Name(x))`를 이어 붙여야 하고, `Find`에는 `all_projects`를 넘길 수 없습니다.

## Interface와 volume attachment

`create_server_interface`는 `port_id`, `net_id`, `fixed_ips`를 typed field로, 2.49의 `tag`를 `WithCreateField("tag", ...)`로 보냅니다. `get_server_interface`, `server_interfaces`는 같은 경로를 읽습니다. Python은 `server_interfaces`에 `limit`과 `marker` query를 받지만 Nova의 interface 목록은 이를 쓰지 않으며, Go `List`는 query 없이 한 번 요청합니다. `delete_server_interface`는 `InServer(ctx, resource.ID(serverID))`가 돌려준 scope의 `Delete(ctx, resource.ID(portID))`가 404를 무시합니다. Python은 interface 객체의 `server_id`로 서버를 정할 수 있고 Go는 서버 ID를 따로 받습니다.

`create_volume_attachment`는 `volumeId`, `device`, `tag`, `delete_on_termination`을 같은 envelope로 보냅니다. Go는 false인 `delete_on_termination`을 생략하며 서버 기본값은 false입니다. Python의 legacy `volume_id`, `volumeId` 인자는 Go `CreateOpts.VolumeID`에 해당합니다. `delete_volume_attachment`는 인자 순서 확인을 위해 서버 ID로 `find_server`를 먼저 부르므로 `GET servers/{id}`가 한 번 더 나갑니다. 그 조회가 실패하면 Python은 두 인자를 바꿔 삭제하지만 Go scope의 `Delete`는 DELETE 하나만 보내고 인자를 바꾸지 않습니다.

`volume_attachments`는 기본 호출이 같은 GET이지만 Python이 받는 `limit`과 `offset` query를 Go 목록에 넘길 수 없습니다. scope의 `List`에 query를 주면 HTTP 전에 `resource.ErrUnsupported`가 돌아옵니다. `update_volume_attachment`의 `PUT servers/{id}/os-volume_attachments/{volume}`은 Go 호출이 없습니다.

## Instance action

`get_server_action`은 `GET servers/{id}/os-instance-actions/{request_id}`이며 Go `InstanceActions.InServer` scope의 `Get`이 같은 요청을 보내고 2.84 event의 `details`까지 `ActionEvent`에 남깁니다. Python의 `ignore_missing` 인자는 `_get`에 쓰이지 않아 404가 그대로 예외가 되고, Go도 `resource.ErrNotFound`를 돌려줍니다.

## unresolved 목록

- `clear_server_password`: `DELETE servers/{id}/os-server-password`를 보내는 Go API가 없습니다.
- `backup_server`: `createBackup` action과 그 뒤의 Glance image 조회·대기를 묶은 Go API가 없습니다.
- `restore_server`: soft delete된 서버의 `restore` action을 보내는 Go API가 없습니다.
- `lock_server`: Go `Lock`은 옵션을 받지 않아 2.73의 `{"lock": {"locked_reason": ...}}`를 보낼 수 없습니다.
- `unshelve_server`: `host`만 줄 때 `host`를 action 안에 넣을 수 없고, `availability_zone=None`의 null을 보낼 수 없습니다.
- `add_fixed_ip_to_server`, `remove_fixed_ip_from_server`: `addFixedIp`, `removeFixedIp` action을 보내는 Go API가 없습니다.
- `add_floating_ip_to_server`: Go에는 `addFloatingIp`만 따로 보내는 호출이 없습니다. `compute.Service.AddIPList`는 Nova floating IP 목록 조회와 재검증을 거치는 다른 workflow이고 그 목록 API가 있는 2.35 이하에서만 동작합니다.
- `remove_floating_ip_from_server`: `removeFloatingIp` action을 보내는 Go API가 없습니다.
- `find_server_group`: ID GET 뒤 목록 fallback을 하는 helper가 없고 `all_projects`를 `Find`에 전달할 수 없습니다.
- `volume_attachments`: `limit`과 `offset` query를 보낼 수 없습니다.
- `update_volume_attachment`: volume attachment PUT 호출이 없습니다.
- `get_image`, `find_image`, `images`, `delete_image`, `fetch_image_metadata`, `get_image_metadata`, `set_image_metadata`, `delete_image_metadata`: Nova의 deprecated `/images` proxy와 그 metadata 하위 경로를 호출하는 Go API가 없습니다. Go에서는 Glance `ImageV2().Images`를 쓰지만 경로, 응답 모델과 metadata 표현이 달라서 같은 요청으로 보지 않았습니다.
