# Nova native 서버 호출

`servers.New(client)`(또는 `service.Servers`)의 generated 메서드는 Gophercloud `v2.15.0`의 [servers 요청](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/servers/requests.go)을 바꾸지 않고 호출합니다. SDK는 오류에 `resource.OperationError{Resource: "servers"}` 문맥만 더하며, 다른 status는 native `gophercloud.ErrUnexpectedResponseCode`로 남습니다. ID는 escape 없이 경로 segment로 이어 붙입니다.

## 조회·목록·생성·수정·삭제

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Get(ctx, id)` | `GET servers/{id}` | 200, 203 |
| `List(ctx, options...)` | `GET servers/detail` | native pager |
| `ListSimple(ctx, options...)` | `GET servers` | native pager |
| `Create(ctx, opts, options...)` | `POST servers` | 200, 202 |
| `Update(ctx, id, opts, options...)` | `PUT servers/{id}` | 200 |
| `Delete(ctx, id)` | `DELETE servers/{id}` | 202, 204 |

`Server`의 `image`가 빈 문자열이면(볼륨 부팅) `Image`는 nil이 됩니다. 두 목록은 `ListOpts`(이름·상태·태그·`all_tenants` 등)와 `WithListQuery`/`WithListSimpleQuery` 확장 query를 보내고, `servers_links`의 `rel=next` href를 받은 그대로 따라갑니다. 호출자가 순회를 멈추면 다음 페이지를 요청하지 않습니다.

`Create`는 `CreateOpts`를 `{"server": {...}}`로 감쌉니다. `Name`은 필수이며, `UserData`는 base64가 아니면 인코딩하고, `SecurityGroups`는 `{"name": ...}` 목록으로, `Networks`는 `[]servers.Network` 또는 `"auto"`/`"none"` 문자열로 보냅니다. 다른 문자열은 HTTP 전에 오류입니다. `WithCreateField`의 확장 필드는 단일 `server` envelope 안에 들어가고 기존 key와 겹치면 HTTP 전에 거부됩니다. `WithCreateHintOpts`의 scheduler hints는 최상위 `os:scheduler_hints`로 보내며, UUID가 아닌 `Group` 같은 잘못된 hint는 HTTP 전에 오류입니다.

`Update`는 이름·access IP·`Hostname`을 `{"server": {...}}`로 보내고 비어 있는 값은 생략합니다. 생성·수정 응답은 서버가 돌려준 `Server`이며 SDK가 요청 값을 합성하지 않습니다. Python의 mutable Server Resource와 대기·주소 helper는 [서버 생성 사용법](../../create-with-floating-ip.md)과 [서버 대기](../../../docs/service-waits.md)를 참고합니다.

## metadata

| 메서드 | 요청 | 성공 status | 반환 |
|---|---|---|---|
| `Metadata(ctx, id)` | `GET servers/{id}/metadata` | 200 | `metadata` map |
| `ResetMetadata(ctx, id, opts, options...)` | `PUT servers/{id}/metadata`, 전체 교체 | 200 | 응답의 `metadata` map |
| `UpdateMetadata(ctx, id, opts, options...)` | `POST servers/{id}/metadata`, 병합 | 200 | 응답의 `metadata` map |
| `Metadatum(ctx, id, key)` | `GET servers/{id}/metadata/{key}` | 200 | `meta` map |
| `CreateMetadatum(ctx, id, opts, options...)` | `PUT servers/{id}/metadata/{key}` | 200 | 응답의 `meta` map |
| `DeleteMetadatum(ctx, id, key)` | `DELETE servers/{id}/metadata/{key}` | 202, 204 | error |

Reset·Update는 `{"metadata": {...}}`, CreateMetadatum은 `{"meta": {...}}`를 보냅니다. `MetadatumOpts`는 정확히 한 쌍이어야 하며 그 key가 경로가 됩니다. 비어 있거나 두 쌍 이상이면 HTTP 전에 오류입니다. typed metadata map은 일반 object envelope가 아니므로 `With...Field` 확장 필드는 `metadata`/`meta` 옆 최상위에 붙고, 이미 있는 key와 겹치면 HTTP 전에 거부됩니다. 반환 map은 서버 응답이며 요청 값을 합성하지 않습니다. key는 escape 없이 경로에 이어 붙입니다.


## 사용자 action

모든 action은 `POST servers/{id}/action`으로 보내며 본문의 최상위 key가 action 이름입니다. 성공 status는 따로 적지 않은 경우 Gophercloud POST 기본값인 201, 202입니다.

| 메서드 | 본문 | 성공 status | 반환 |
|---|---|---|---|
| `ChangeAdminPassword(ctx, id, newPassword)` | `{"changePassword": {"adminPass": ...}}` | 201, 202 | error |
| `Reboot(ctx, id, opts, options...)` | `{"reboot": {"type": "SOFT"\|"HARD"}}` | 201, 202 | error |
| `Rebuild(ctx, id, opts, options...)` | `{"rebuild": {...}}` | 201, 202 | 응답의 `server` |
| `Resize(ctx, id, opts, options...)` | `{"resize": {"flavorRef": ...}}` | 201, 202 | error |
| `ConfirmResize(ctx, id)` | `{"confirmResize": null}` | 201, 202, 204 | error |
| `RevertResize(ctx, id)` | `{"revertResize": null}` | 201, 202 | error |
| `CreateImage(ctx, id, opts, options...)` | `{"createImage": {"name": ..., "metadata": ...}}` | 202 | image ID |
| `Rescue(ctx, id, opts, options...)` | `{"rescue": {"adminPass": ..., "rescue_image_ref": ...}}` | 200 | 응답의 `adminPass` |
| `Unrescue(ctx, id)` | `{"unrescue": null}` | 201, 202 | error |
| `Start`·`Stop(ctx, id)` | `{"os-start": null}`·`{"os-stop": null}` | 201, 202 | error |
| `Pause`·`Unpause(ctx, id)` | `{"pause": null}`·`{"unpause": null}` | 201, 202 | error |
| `Suspend`·`Resume(ctx, id)` | `{"suspend": null}`·`{"resume": null}` | 201, 202 | error |
| `Lock`·`Unlock(ctx, id)` | `{"lock": null}`·`{"unlock": null}` | 201, 202 | error |
| `Shelve`·`ShelveOffload(ctx, id)` | `{"shelve": null}`·`{"shelveOffload": null}` | 201, 202 | error |
| `Unshelve(ctx, id, opts, options...)` | `{"unshelve": null}` 또는 `{"unshelve": {"availability_zone": ...}}` | 201, 202 | error |

`Reboot`의 `Type`, `Resize`의 `FlavorRef`, `CreateImage`의 `Name`은 필수이며 비어 있으면 HTTP 전에 오류입니다. `Rebuild`·`Resize`의 `DiskConfig`는 `AUTO`나 `MANUAL`만 받고 다른 값은 HTTP 전에 거부합니다. `Rebuild`의 `imageRef`는 생략하지 않으므로 빈 `RebuildOpts`도 `{"rebuild": {"imageRef": ""}}`를 보냅니다.

`With...Field` 확장 필드는 action 객체 안에 들어갑니다. 다만 `Unshelve`에 가용 영역을 주지 않으면 action 값이 null이라 확장 필드는 `unshelve` 옆 최상위에 붙습니다. 옵션 struct가 가진 key(예: `type`, `name`, `availability_zone`)와 같은 확장 필드나 nil 옵션은 HTTP 전에 거부됩니다.

`CreateImage`는 응답의 `X-OpenStack-Nova-API-Version` header로 image ID 위치를 고릅니다. 2.45 미만이면 `Location` header 경로의 마지막 segment를, 2.45 이상이면 본문의 `image_id`를 돌려줍니다. 다만 2.45 이상에서 `Content-Type`이 정확히 `application/json`이 아니면 본문을 읽지 않으므로 오류 없이 빈 ID를 돌려줍니다. version header가 없거나 형식이 틀리거나, 2.45 미만에서 `Location`이 없거나 `/`이면 202를 받은 뒤에도 오류입니다. 이때 서버 쪽 image 생성 요청은 이미 접수됐을 수 있습니다.

evacuate·force delete·live migrate·migrate·reset state·reset network·inject network info 같은 관리자 action은 이 절에서 다루지 않습니다.

## 주소·console output·상태 대기

| 메서드 | 요청 | 성공 status | 반환 |
|---|---|---|---|
| `ListAddresses(ctx, id)` | `GET servers/{id}/ips` | native pager 200, 204, 300 | 단일 페이지의 `addresses` map 하나 |
| `ListAddressesByNetwork(ctx, id, network)` | `GET servers/{id}/ips/{network}` | native pager 200, 204, 300 | 그 network의 `Address` 행 |
| `ShowConsoleOutput(ctx, id, opts, options...)` | `POST servers/{id}/action`, `{"os-getConsoleOutput": {...}}` | 200 | 응답의 `output` 문자열 |
| `WaitForStatus(ctx, id, status)` | `GET servers/{id}` 반복 | 200, 203 | error |

두 주소 목록은 다음 페이지가 없는 단일 페이지 stream입니다. 주소가 하나도 없으면 아무 값도 내보내지 않습니다. 본문 없는 204는 native pager가 JSON을 먼저 읽기 때문에 빈 목록이 아니라 `io.EOF` 오류 하나로 끝납니다. network별 응답은 최상위 key 하나를 network 이름으로 보고 그 배열을 꺼내므로, 최상위 key가 여럿이면 어느 배열을 고를지 정해져 있지 않습니다. 목록 오류는 operation 문맥 없이 native 오류 그대로 전달됩니다. network 이름은 escape 없이 경로에 이어 붙입니다.

`ShowConsoleOutput`은 `Length`가 0이면 `{"os-getConsoleOutput": {}}`를 보내 전체 출력을 요청합니다. `WithShowConsoleOutputField` 확장 필드는 action 객체 안에 들어가며 `length`와 겹치거나 nil 옵션이면 HTTP 전에 거부됩니다.

`WaitForStatus`는 첫 GET을 바로 보내고, 일치하지 않으면 1초마다 다시 조회합니다. 상태 문자열이 정확히 같을 때만 끝나므로 `ERROR`도 종료 조건이 아니며 context가 끝나면 그 오류를 돌려줍니다. GET 오류는 즉시 대기를 멈춥니다. Python `wait_for_server`의 실패 상태·간격·timeout 정책은 [서버 대기](../../../docs/service-waits.md)를 참고합니다.
