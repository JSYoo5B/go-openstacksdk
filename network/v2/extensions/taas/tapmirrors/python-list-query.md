# Tap mirror 목록의 반복 query

openstacksdk의 [`tap_mirrors(**query)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)는 [`TapMirror`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/tap_mirror.py)의 `sort_key`, `sort_dir`, `fields`, `name`, `project_id` query를 requests에 넘깁니다. requests는 목록 값을 같은 키의 반복으로 보내므로 `fields=["id", "name"]`은 `fields=id&fields=name`이 됩니다. Gophercloud v2.15.0 `ListOpts`에는 이 키들이 없고 `WithListQuery`와 `resource.WithQuery`는 키마다 값 하나만 남기기 때문에, `WithListQueryValues`가 반복 키를 보냅니다.

## 사용법

```go
for mirror, err := range api.List(ctx,
	tapmirrors.WithListQuery("name", "mirror"),
	tapmirrors.WithListQueryValues("fields", "id", "name"),
	tapmirrors.WithListQueryValues("sort_key", "name", "id"),
	tapmirrors.WithListQueryValues("sort_dir", "asc", "desc"),
) {
	...
}
```

- `WithListQueryValues(key, values...)`는 키를 값 순서대로 반복해 `GET taas/tap_mirrors`에 붙입니다. 같은 키에 나중에 준 `WithListQuery`나 `WithListQueryValues`가 이전 값을 대신하고, 값 없이 부르면 그 키를 뺍니다.
- `List`는 `tap_mirrors_links`의 next href를 따라가며, Neutron이 돌려주는 href에 반복 키가 그대로 들어 있으면 다음 요청에도 남습니다.
- 빈 키는 HTTP 전에 `resource.ErrInvalidOption`을 담은 `resource.OperationError`(Operation `List`)로 실패합니다. 400 같은 오류 응답은 Gophercloud 상태 오류로 돌려줍니다.

## Python과 다른 점

- Python은 선언된 query 키만 보내고 `port_id` 같은 다른 Body 속성은 받은 뒤 걸러 내며 알 수 없는 키는 버립니다. Go는 `WithListQuery`, `WithListQueryValues`, `resource.WithQuery`로 준 키를 모두 서버에 보냅니다.
- 반복 키는 `API.List`에서만 쓸 수 있습니다. `API.All`과 `resource.WithQuery`는 키마다 값 하나를 보내므로, 반복 `fields`와 `max_items`를 함께 쓰려면 `List` iterator를 원하는 개수에서 멈춥니다.
- Python은 link 없이 limit만큼 찬 page 뒤에 `marker=<마지막 id>`로 이어 가지만 Go는 `tap_mirrors_links`만 따라갑니다.
- Python은 `project_id`가 `tenant_id`로 대체되는 TapMirror Resource를 돌려주고, Go는 `ProjectID`와 `TenantID`를 따로 가진 `*tapmirrors.TapMirror`를 돌려줍니다.
- Python은 `directions`를 그대로 넘기지만 Go `Directions`는 IN/OUT tunnel ID를 정수 문자열 JSON으로 읽으므로 따옴표 없는 숫자에서는 디코드가 실패합니다.
- Neutron에는 microversion이 없어 두 SDK 모두 버전 헤더를 보내지 않습니다.
