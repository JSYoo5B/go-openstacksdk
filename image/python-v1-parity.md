# openstacksdk image v1 Proxy 대응

고정한 openstacksdk 커밋 `ef55d7d`의 [image v1 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v1/_proxy.py)는 Glance v1 API를 호출합니다. Gophercloud v2.15.0에는 Glance v1 패키지가 없고 이 SDK도 Glance v1용 Go API를 만들지 않았기 때문에, 아래 11개 메서드는 모두 대응하는 Go 호출이 없는 `unresolved`입니다. Glance v1 API는 Newton 릴리스에서 폐기되고 Rocky 릴리스에서 Glance에서 제거되었으므로, 이 SDK에서는 [image README](README.md)의 Glance v2 호출을 사용합니다.

| Python 메서드 | Go 호출 | 판정 |
|---|---|---|
| [`create_image`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v1/_proxy.py#L63) | 없음 | unresolved |
| [`upload_image`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v1/_proxy.py#L254) | 없음 | unresolved |
| [`delete_image`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v1/_proxy.py#L328) | 없음 | unresolved |
| [`find_image`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v1/_proxy.py#L359) | 없음 | unresolved |
| [`get_image`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v1/_proxy.py#L378) | 없음 | unresolved |
| [`images`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v1/_proxy.py#L390) | 없음 | unresolved |
| [`update_image`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v1/_proxy.py#L400) | 없음 | unresolved |
| [`download_image`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v1/_proxy.py#L414) | 없음 | unresolved |
| [`update_image_properties`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v1/_proxy.py#L477) | 없음 | unresolved |
| [`wait_for_status`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v1/_proxy.py#L515) | 없음 | unresolved |
| [`wait_for_delete`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v1/_proxy.py#L554) | 없음 | unresolved |

## Python이 보내는 요청

v1 [Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v1/image.py) Resource는 `/images`를 기준 경로로 쓰고 `image` resource key로 본문을 감쌉니다. Go로 옮길 때 기준이 되도록 메서드별 요청을 정리합니다.

- `get_image`는 `GET /images/{id}`, `delete_image`는 `DELETE /images/{id}`를 보내고 삭제의 기본값은 `ignore_missing=True`입니다.
- `images(**query)`는 `GET /images/detail`을 보내고 `name`, `container_format`, `disk_format`, `status`, `size_min`, `size_max`, `limit`, `marker`를 query로 보내며 `images_links`를 따라갑니다.
- `find_image`는 `GET /images/{name_or_id}`가 404이면 `name` query를 붙인 `GET /images/detail`로 다시 찾습니다. 다른 v1 Resource와 달리 400과 403에서는 목록 단계로 넘어가지 않습니다.
- `update_image`는 `PUT /images/{id}`에 `{"image": {...}}` 본문을 보냅니다.
- `update_image_properties`는 cloud 계층의 `get_image`로 현재 이미지를 읽은 뒤, 값이 달라진 속성만 `x-image-meta-{key}` header로 담아 본문 없는 `PUT /images/{id}`를 보냅니다. 바뀐 값이 없으면 요청 없이 `False`를 돌려줍니다.
- `create_image`는 중복 이름 검사, checksum 계산, vendor agent 속성 같은 cloud 기본값을 적용한 뒤, 데이터가 있으면 `POST /images`로 메타데이터를 만들고 `PUT /images/{id}`로 데이터를 올립니다. 업로드가 실패하면 만든 이미지를 지우려고 `DELETE /images/{id}`를 보냅니다.
- `upload_image`는 폐기 경고를 남기고 `POST /images`만 보냅니다.
- `download_image`는 `GET /images/{id}`로 checksum 메타데이터를 읽은 뒤 `GET /images/{id}/file`로 데이터를 받고 checksum을 검사합니다.
- `wait_for_status`와 `wait_for_delete`는 Image Resource의 fetch를 반복하며, 삭제 대기의 기본 제한 시간은 120초입니다.

## 남은 부분

모든 메서드에 같은 이유가 남아 있습니다. 이 SDK에는 Glance v1 서비스 client, `images`와 `images/detail` 경로, `x-image-meta-*` header 메타데이터, v1 업로드와 다운로드, v1 Image 대기를 보내는 공개 Go API가 없습니다. Glance v2의 `image.Service`와 `image/v2/images` 호출은 경로와 본문 형식이 달라서 v1 요청을 대신하지 않습니다.
