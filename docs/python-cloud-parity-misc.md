# Python cloud 계층 기타 helper 대응

openstacksdk의 cloud 계층에서 object store helper, Connection 상태 helper, `FileSegment`, inventory, `Connection` 메서드를 Go SDK와 비교한 결과입니다. 비교 기준은 고정 commit `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 다음 소스입니다.

- [openstack/cloud/_object_store.py](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_object_store.py)
- [openstack/cloud/openstackcloud.py](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/openstackcloud.py)
- [openstack/cloud/_utils.py](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py)
- [openstack/cloud/inventory.py](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/inventory.py)
- [openstack/connection.py](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/connection.py)

판정은 공개 Go API가 기본 인자에서 helper의 동작 전체를 직접 수행할 때만 대응으로 봅니다. 호출자가 행을 거르거나, 결과에 따라 다음 호출을 고르거나, 404를 직접 성공으로 바꿔야 한다면 미해결로 둡니다. 아래 예제의 `containers`, `objects`는 `objectstorage/v1/containers`, `objectstorage/v1/objects` 패키지이고 `resource`는 SDK의 `resource` 패키지입니다.

## 대응 표

| Python 메서드 | Go 호출 | 판정 |
| --- | --- | --- |
| `list_containers` | `containers.API.ListRecords(ctx, containers.WithRecordListPrefix(p))` | 대응 |
| `search_containers` | 없음 (`ListRecords` 뒤 호출자 필터) | 미해결 |
| `get_container` | `containers.API.Find(ctx, resource.ID(name), resource.WithIgnoreMissing())` | 대응 |
| `create_container` | `Find`, `CreateRecord`, `Find`를 호출자가 이어 붙임 | 미해결 |
| `delete_container` | `containers.API.DeleteRecord(ctx, name)` | 대응 |
| `update_container` | `containers.API.SetMetadataAndRefresh(ctx, name, headers, containers.WithMetadataRefresh(false))` | 대응 |
| `set_container_access` | `SetMetadataAndRefresh`에 ACL 문자열 직접 전달 | 미해결 |
| `get_container_access` | `Find` 결과의 `X-Container-Read`를 호출자가 해석 | 미해결 |
| `get_object_capabilities` | `objectstorage/v1.Service.GetInfo` | 대응 |
| `get_object_segment_size` | `objectstorage/v1.Service.GetObjectSegmentSize` | 대응 |
| `is_object_stale` | `objects.API.IsObjectStale` | 대응 |
| `create_object` | `objects.API.CreateObject` | 미해결 |
| `update_object` | `objects.API.SetMetadataFromFresh(ctx, container, name, values)` | 대응 |
| `list_objects` | `objects.API.InContainer` 뒤 `Collection.List` | 미해결 |
| `search_objects` | 없음 | 미해결 |
| `delete_object` | `objects.API.DeleteObject(ctx, container, name)` | 대응 |
| `delete_autocreated_image_objects` | 없음 | 미해결 |
| `get_object_metadata` | `objects.API.HeadMetadata(ctx, container, name)` | 대응 |
| `get_object_raw` | 없음 (`GetObject`는 임의 query를 받지 않음) | 미해결 |
| `stream_object` | `objects.API.StreamObject` | 미해결 |
| `get_object` | `objects.API.GetObject`, `objects.API.DownloadObject` | 미해결 |
| `current_location` | `Connection.CurrentLocation()` | 대응 |
| `current_project` | `Connection.CurrentLocation()`의 `Project` | 대응 |
| `current_project_id` | `Connection.CurrentLocation()`의 `Project.ID` | 대응 |
| `get_name` | `Connection.CurrentLocation()`의 `Cloud` | 대응 |
| `auth_token` | 없음 | 미해결 |
| `current_user_id` | 없음 | 미해결 |
| `service_catalog` | 없음 | 미해결 |
| `session` | 없음 | 미해결 |
| `close` | 없음 | 미해결 |
| `set_global_request_id` | 없음 | 미해결 |
| `global_request` | 없음 | 미해결 |
| `pprint` | 없음 | 미해결 |
| `pformat` | 없음 | 미해결 |
| `range_search` | 없음 | 미해결 |
| `search_resources` | 없음 | 미해결 |
| `project_cleanup` | 없음 | 미해결 |
| `has_service` | 없음 | 미해결 |
| `get_session_endpoint` | 없음 | 미해결 |
| `FileSegment.read` | 없음 | 미해결 |
| `FileSegment.reset` | 없음 | 미해결 |
| `FileSegment.seek` | 없음 | 미해결 |
| `FileSegment.tell` | 없음 | 미해결 |
| `OpenStackInventory.list_hosts` | 없음 | 미해결 |
| `OpenStackInventory.search_hosts` | 없음 | 미해결 |
| `OpenStackInventory.get_host` | 없음 | 미해결 |
| `Connection.add_service` | 없음 | 미해결 |
| `Connection.authorize` | 없음 | 미해결 |
| `Connection.connect_as` | 없음 | 미해결 |
| `Connection.connect_as_project` | 없음 | 미해결 |
| `Connection.endpoint_for` | 없음 | 미해결 |

51개 가운데 11개가 대응이고 40개가 미해결입니다.

## Container helper

`list_containers`는 account root에 `GET`을 보내고 `prefix`가 있을 때만 query에 넣습니다. Python Container는 `X-Account-Container-Count`가 지금까지 받은 개수보다 크면 마지막 이름을 `marker`로 다음 쪽을 요청합니다. `ListRecords`도 같은 규칙으로 이어 받으며 `prefix`를 유지하고 빈 쪽에서 멈춥니다. Python은 list를 돌려주지만 Go는 `ContainerRecord`를 차례로 내놓는 iterator를 돌려줍니다. `full_listing`은 Python에서 경고만 내므로 Go 대응 인자가 없습니다.

`get_container`는 container에 `HEAD`를 보내고 404이면 `None`을 돌려줍니다. Go에서는 `Find(ctx, resource.ID(name), resource.WithIgnoreMissing())`가 같은 `HEAD`를 보내고 404에서 `nil, nil`을 돌려줍니다. 403 같은 다른 실패는 두 쪽 모두 오류입니다. 결과 `ContainerResource`에는 개수, 바이트, 사용자 metadata, 원본 응답 header가 들어 있습니다. Go는 Gophercloud의 container 이름 검사를 먼저 하므로 `/`가 들어간 이름이나 256바이트를 넘는 이름은 요청 전에 거부합니다. `HEAD` 응답은 200과 204만 받는데, Python은 2xx 전체를 받습니다.

`delete_container`는 Python에서 성공하면 `True`, 404이면 `False`, 409이면 비어 있지 않다는 `SDKException`을 냅니다. `DeleteRecord`는 기본으로 깨끗한 404를 무시하고 `IgnoredMissing`을 참으로 표시하므로 Python 결과는 `!IgnoredMissing`입니다. 409는 Go에서 상태와 본문이 남은 HTTP 오류로 돌아옵니다.

`update_container`는 `set_container_metadata(..., refresh=False)`로 `POST` 한 번을 보냅니다. `SetMetadataAndRefresh`에 `WithMetadataRefresh(false)`를 주면 같은 요청이 나갑니다. 접미사 key는 `X-Container-Meta-`를 붙이고, 이미 붙은 key는 그대로 두며, `read_ACL`이나 `write_ACL` 같은 system 이름은 `X-Container-Read`, `X-Container-Write`로 바꿉니다. Python은 keyword 인자의 마지막 값이 이기지만 Go는 같은 header로 가는 key가 둘이면 요청 전에 실패합니다.

`create_container`는 `HEAD`로 먼저 확인하고, 없을 때만 `PUT`과 두 번째 `HEAD`를 보냅니다. Go의 `Find`, `CreateRecord`, `Find`로 같은 요청 순서를 만들 수 있다는 점은 테스트로 확인했습니다. 그래도 있는지 보고 만들지 정하는 분기는 호출자 몫이고, `public=True`도 호출자가 `read_ACL`에 `.r:*,.rlistings`를 직접 넣어야 하므로 미해결로 둡니다. `set_container_access`와 `get_container_access`도 같은 이유입니다. Go에는 `public`, `private`를 ACL 문자열로 바꾸거나 거꾸로 해석하는 helper가 없습니다. 그래서 호출자가 `SetMetadataAndRefresh`에 ACL 값을 직접 넘기거나 `Find` 결과의 `X-Container-Read`를 직접 비교해야 합니다. 비공개 ACL은 빈 `X-Container-Read` 값으로 나가는 것까지 확인했습니다.

`search_containers`는 container 목록 뒤에 `_utils._filter_list`로 이름 glob, 중첩 dictionary 조건, JMESPath 문자열을 적용합니다. 이 필터를 container에 적용하는 공개 Go helper가 없으므로 미해결입니다.

## Object helper

`get_object_metadata`는 object에 `HEAD`를 보내고 `X-Object-Meta-` 뒤 이름을 소문자로 바꾼 dict를 돌려줍니다. `HeadMetadata`는 query 없는 `HEAD`를 보내고 `Metadata.Values`에 같은 key를 담습니다. 404는 두 쪽 모두 오류입니다.

`update_object`는 `metadata`와 keyword header를 하나로 합친 뒤 `set_object_metadata`를 부릅니다. Python Object setter는 먼저 `HEAD`로 현재 metadata를 읽고, 기존 값에 새 값을 더한 다음 `POST`를 한 번 보냅니다. 빈 값은 빼고 마지막 `HEAD`는 하지 않습니다. `SetMetadataFromFresh`도 같은 순서를 따르며 `content_type` 같은 system 이름은 `Content-Type`으로 바꿉니다. Go 호출자는 두 dict를 합친 map 하나를 넘깁니다.

`delete_object`는 `ignore_missing=False`로 지운 뒤 성공이면 `True`, `SDKException`이면 `False`를 돌려줍니다. Python Object 삭제는 static large object 여부를 알려고 먼저 `HEAD`를 보내고, 해당하면 `DELETE`에 `multipart-manifest=delete`를 붙입니다. `DeleteObject`도 같은 `HEAD`와 `DELETE`를 보냅니다. `HEAD`나 `DELETE`에서 깨끗한 404를 받으면 오류 없이 `IgnoredMissing`이 참이 되므로 Python의 `False`와 같습니다. 다만 Python은 409나 5xx도 `False`로 삼키는데 Go는 이 실패를 오류로 돌려줍니다.

object 이름에 `/`가 있으면 Python은 이름을 그대로 경로에 붙이고 Go는 `%2F`로 escape합니다. Swift는 두 경로를 같은 object로 해석합니다.

`get_object`, `stream_object`, `create_object`, `is_object_stale`, `get_object_capabilities`, `get_object_segment_size`, `list_objects`는 같은 이름의 Proxy 메서드에 그대로 위임합니다. 그래서 cloud helper의 판정은 위임하는 Proxy 메서드의 판정을 따릅니다. `get_object_capabilities`, `get_object_segment_size`, `is_object_stale`는 Proxy 쪽 `GetInfo`, `GetObjectSegmentSize`, `IsObjectStale`이 기본 인자에서 같은 `/info` `GET`이나 object `HEAD`를 보내고 같은 값을 돌려주므로 대응입니다. 나머지 Proxy 메서드는 미해결로 기록되어 있고, cloud helper의 기본 경로도 그 차이를 그대로 지나갑니다. 예를 들어 `get_object`는 기본값에서 `response.text`를 저장하므로 Requests가 추측한 문자 집합으로 디코딩한 문자열을 돌려주는데, Go `GetObject`는 원본 바이트를 돌려줍니다. `stream_object`는 `resp_chunk_size` 단위 iterator이고 404에서 빈 iterator가 되지만 Go `StreamObject`는 `io.ReadCloser`를 돌려주고 404를 오류로 냅니다.

## Connection 상태

`current_location`, `current_project`, `current_project_id`, `get_name`은 모두 `Connection.CurrentLocation()` 하나로 얻습니다. 이 메서드는 Connect 때 기록한 token scope와 설정값을 요청 없이 읽습니다. JSON으로 바꾸면 `cloud`, `region_name`, `zone`, `project`의 `id`, `name`, `domain_id`, `domain_name` 이름이 Python Munch와 같습니다. project ID는 token에서 오고 project 이름과 domain은 설정에서 옵니다. scope가 없는 token이면 ID가 `null`이고 Python도 `None`입니다.

차이도 있습니다. Python은 `OS_*` 환경 변수로 만든 연결의 이름을 `envvars`(또는 `OS_CLOUD_NAME`)로 붙이지만 Go는 identity endpoint의 host를 씁니다. `FromProvider`로 만든 연결은 `WithCloudLocation`을 주지 않으면 이름이 `null`입니다. region을 설정하지 않으면 Python은 빈 문자열이고 Go는 `null`입니다. Python `current_project_id`는 `session.get_project_id()`를 부르므로 필요하면 인증을 하지만 Go는 기록된 값만 읽습니다.

## 미해결 목록

- `search_containers`, `search_objects`, `OpenStackInventory.search_hosts`: `_utils._filter_list`를 대신하는 공개 Go helper가 없습니다.
- `create_container`: 있으면 돌려주고 없으면 만드는 helper가 없고, `public=True`는 호출자가 ACL 값을 직접 넣어야 합니다.
- `set_container_access`, `get_container_access`: `public`, `private`와 ACL 문자열 사이의 변환과 잘못된 값 거부가 없습니다.
- `get_object`: 기본 경로의 문자열 디코딩, `outfile` 경로를 열고 닫는 동작, 404를 `None`으로 바꾸는 동작이 없습니다.
- `stream_object`: chunk iterator와 404를 빈 iterator로 바꾸는 동작이 없습니다.
- `get_object_raw`: 임의 query 문자열로 `GET`을 보내고 원본 응답을 돌려주는 호출이 없습니다.
- `list_objects`: Python과 같은 object record 목록 호출이 없고, 기존 Collection 목록은 `format=json` query와 Python 이어 받기 규칙을 기준으로 비교하지 않았습니다.
- `create_object`: 위임하는 Proxy `create_object`가 미해결인 이유가 그대로 남습니다. large object segment 이름과 `If-None-Match` header, DLO manifest prefix가 Python 요청과 다릅니다.
- `delete_autocreated_image_objects`: `image_api_use_tasks` 확인, `images` container 목록, `x-sdk-autocreated` metadata 확인과 삭제를 묶은 helper가 없습니다.
- `auth_token`, `Connection.authorize`: Connection이 token을 노출하지 않습니다. token은 Gophercloud `ProviderClient`에 있고, 만료 전에 미리 다시 인증하지도 않습니다.
- `current_user_id`, `service_catalog`, `session`: 인증 응답의 user ID, service catalog, keystoneauth Session에 해당하는 공개 접근자가 없습니다.
- `close`: Connection에 `Close`가 없습니다. 인증 cache 저장이나 executor 종료 같은 정리 대상도 없고 `http.Client`는 호출자가 소유합니다.
- `set_global_request_id`, `global_request`: 모든 요청에 `X-OpenStack-Request-ID`를 붙이는 설정이 없습니다.
- `pprint`, `pformat`: SDK에 출력 helper가 없으며 Go에서는 `fmt`나 `encoding/json`을 씁니다.
- `range_search`: 여러 dictionary에 정수 범위 조건을 적용하는 공개 helper가 없습니다.
- `search_resources`: `service.resource` 문자열로 resource를 찾아 ID 조회 뒤 목록 조회로 넘어가는 registry가 없습니다.
- `project_cleanup`: 서비스 의존 순서, dry run, filter, callback, 상태 queue를 갖춘 정리 작업이 없습니다.
- `has_service`, `get_session_endpoint`, `Connection.endpoint_for`: service type 문자열로 endpoint를 찾거나 존재 여부를 bool로 알려 주는 공개 API가 없습니다. 호출별 `interface`, `region_name`, version 범위도 지정할 수 없습니다.
- `Connection.connect_as`, `Connection.connect_as_project`: 현재 설정에서 인증 인자만 바꾼 새 Connection을 만드는 기능이 없어서 `Connect`를 다시 불러야 합니다.
- `Connection.add_service`: 실행 중에 새 서비스 proxy를 등록하는 기능이 없습니다.
- `FileSegment`의 `read`, `reset`, `seek`, `tell`: 공개 segment reader가 없습니다. Python의 이 class는 내부 upload helper이고 Go segment upload는 `CreateObject` 안의 내부 reader를 씁니다.
- `OpenStackInventory.list_hosts`, `OpenStackInventory.get_host`: `clouds.yaml`의 모든 cloud에 연결해 server를 모으는 inventory가 없습니다.

## 검증

- [objectstorage/v1/containers/python_parity_test.go](../objectstorage/v1/containers/python_parity_test.go): container 목록, `HEAD` 조회, 생성 순서, 삭제 결과, metadata `POST`.
- [objectstorage/v1/objects/python_parity_test.go](../objectstorage/v1/objects/python_parity_test.go): object metadata `HEAD`, setter의 `HEAD` 뒤 `POST`, 삭제 결과.
- [python_parity_test.go](../python_parity_test.go): `CurrentLocation`의 cloud, region, project 값과 요청 없는 조회.
