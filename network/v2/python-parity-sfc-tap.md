# SFC·TaaS: openstacksdk와 Go

고정 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 network Proxy 가운데 networking-sfc 5종(flow classifier, port chain, port pair, port pair group, service graph)과 tap-as-a-service 3종(tap flow, tap mirror, tap service)의 메서드 48개를 Go SDK 호출과 비교합니다. Gophercloud pin은 `v2.15.0`입니다. 이 pin에는 TaaS 중 `networking/v2/extensions/taas/tapmirrors` 패키지만 있고 networking-sfc, tap flow, tap service 패키지는 없습니다. 그래서 Go 쪽 대응은 생성된 [`tapmirrors`](extensions/taas/tapmirrors) facade에 한정됩니다.

예제의 `api`는 `tapmirrors.New(client)`로 만든 `*tapmirrors.API`이며, Connection의 network 서비스에서 같은 값을 얻을 수 있습니다. 판정 근거가 되는 요청과 응답은 [`python_parity_test.go`](extensions/taas/tapmirrors/python_parity_test.go)가 고정합니다.

| Python 메서드 | Go 호출 | 판정 |
|---|---|---|
| [`create_tap_mirror`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9066) | `api.Create(ctx, tapmirrors.CreateOpts{...}, tapmirrors.WithCreateField(k, v)...)` | 대응 |
| [`get_tap_mirror`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9110) | `api.GetWithFields(ctx, id, fields...)` | 대응 |
| [`find_tap_mirror`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9096) | `api.Find(ctx, resource.ID(x), resource.WithIgnoreMissing())` 뒤 `api.Find(ctx, resource.Name(x))` | 미해결 |
| [`tap_mirrors`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9138) | `api.All(ctx, resource.WithQuery(k, v)...)`, `api.List(ctx, tapmirrors.WithListQuery(k, v)...)` | 미해결 |
| [`update_tap_mirror`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9130) | `api.Update(ctx, id, tapmirrors.UpdateOpts{...}, tapmirrors.WithUpdateField(k, v)...)` | 대응 |
| [`delete_tap_mirror`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9070) | `api.Remove(ctx, resource.ID(id))` 또는 `resource.IgnoreMissing(api.Delete(ctx, id))` | 대응 |
| [`create_tap_flow`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L8987) | SDK 소유 package | 대응 |
| [`get_tap_flow`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9029) | SDK 소유 package | 대응 |
| [`find_tap_flow`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9015) | SDK 소유 package | 대응 |
| [`tap_flows`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9057) | SDK 소유 package | 대응 |
| [`update_tap_flow`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9049) | SDK 소유 package | 대응 |
| [`delete_tap_flow`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L8991) | SDK 소유 package | 대응 |
| [`create_tap_service`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9147) | SDK 소유 package | 대응 |
| [`get_tap_service`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9191) | SDK 소유 package | 대응 |
| [`find_tap_service`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9177) | SDK 소유 package | 대응 |
| [`tap_services`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9219) | SDK 소유 package | 대응 |
| [`update_tap_service`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9211) | SDK 소유 package | 대응 |
| [`delete_tap_service`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9151) | SDK 소유 package | 대응 |
| [`create_sfc_flow_classifier`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9228) | SDK 소유 package | 대응 |
| [`get_sfc_flow_classifier`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9310) | SDK 소유 package | 대응 |
| [`find_sfc_flow_classifier`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9283) | SDK 소유 package | 대응 |
| [`sfc_flow_classifiers`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9347) | SDK 소유 package | 대응 |
| [`update_sfc_flow_classifier`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9329) | SDK 소유 package | 대응 |
| [`delete_sfc_flow_classifier`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9242) | SDK 소유 package | 대응 |
| [`create_sfc_port_chain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9369) | SDK 소유 package | 대응 |
| [`get_sfc_port_chain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9450) | SDK 소유 package | 대응 |
| [`find_sfc_port_chain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9423) | SDK 소유 package | 대응 |
| [`sfc_port_chains`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9483) | SDK 소유 package | 대응 |
| [`update_sfc_port_chain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9467) | SDK 소유 package | 대응 |
| [`delete_sfc_port_chain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9382) | SDK 소유 package | 대응 |
| [`create_sfc_port_pair`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9502) | SDK 소유 package | 대응 |
| [`get_sfc_port_pair`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9580) | SDK 소유 package | 대응 |
| [`find_sfc_port_pair`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9554) | SDK 소유 package | 대응 |
| [`sfc_port_pairs`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9613) | SDK 소유 package | 대응 |
| [`update_sfc_port_pair`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9597) | SDK 소유 package | 대응 |
| [`delete_sfc_port_pair`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9513) | SDK 소유 package | 대응 |
| [`create_sfc_port_pair_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9632) | SDK 소유 package | 대응 |
| [`get_sfc_port_pair_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9713) | SDK 소유 package | 대응 |
| [`find_sfc_port_pair_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9686) | SDK 소유 package | 대응 |
| [`sfc_port_pair_groups`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9750) | SDK 소유 package | 대응 |
| [`update_sfc_port_pair_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9732) | SDK 소유 package | 대응 |
| [`delete_sfc_port_pair_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9645) | SDK 소유 package | 대응 |
| [`create_sfc_service_graph`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9770) | SDK 소유 package | 대응 |
| [`get_sfc_service_graph`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9851) | SDK 소유 package | 대응 |
| [`find_sfc_service_graph`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9824) | SDK 소유 package | 대응 |
| [`sfc_service_graphs`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9886) | SDK 소유 package | 대응 |
| [`update_sfc_service_graph`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9868) | SDK 소유 package | 대응 |
| [`delete_sfc_service_graph`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9783) | SDK 소유 package | 대응 |

판정이 "대응"이면 Python 인수의 효과를 공개 Go API로 재현할 수 있고 남은 차이는 아래에 적었습니다. "미해결"이면 Go에 경로가 없거나 Python 인수 일부를 보낼 방법이 없습니다.

## Tap mirror

Python 자원 정의는 [`tap_mirror.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/tap_mirror.py)입니다. 경로는 `/taas/tap_mirrors`이고 요청과 응답 envelope는 `tap_mirror`, 목록 envelope는 `tap_mirrors`입니다. Neutron에는 microversion이 없으므로 두 SDK 모두 버전 header를 보내지 않습니다.

### 생성과 수정

`create_tap_mirror(**attrs)`는 호출자가 넘긴 속성만 `{"tap_mirror": {...}}`에 담아 POST합니다. 정의되지 않은 속성도 그대로 보내도록 `_allow_unknown_attrs_in_body`가 켜져 있습니다. Go `Create`는 `CreateOpts`의 `name`, `port_id`, `mirror_type`, `remote_ip`, `directions`를 값이 비어 있어도 항상 보내고 `description`, `tenant_id`만 비었을 때 생략합니다. 서버가 `port_id`, `mirror_type`, `remote_ip`, `directions`를 요구하고 `name` 기본값이 빈 문자열이므로 유효한 호출의 결과는 같습니다. `project_id`나 확장 속성은 `tapmirrors.WithCreateField`로 envelope 안에 넣을 수 있지만 `CreateOpts`가 이미 가진 키를 덮어쓰려 하면 HTTP 전에 `resource.ErrInvalidOption`으로 거부됩니다.

`directions`는 차이가 큽니다. Python은 넘긴 dict를 그대로 보내고 응답도 그대로 보관합니다. Go `Directions`는 `IN`, `OUT` tunnel ID를 정수로 받아 `"101"` 같은 JSON 문자열로 보내고, 0은 생략하며, 응답에서도 문자열로 감싼 정수만 해석합니다. 따라서 서버가 따옴표 없는 숫자를 돌려주면 Go `Get`과 `Create`는 decode 오류를 냅니다.

`update_tap_mirror(tap_mirror, **attrs)`는 바뀐 속성만 PUT합니다. 바뀐 속성이 없으면 Python은 요청을 보내지 않지만 Go `Update`는 `UpdateOpts{}`여도 `{"tap_mirror":{}}`를 PUT합니다. 이름과 설명은 `UpdateOpts`의 포인터 필드로, 나머지 속성은 `tapmirrors.WithUpdateField`로 보냅니다.

Python 결과는 `TapMirror` Resource이고 `project_id`가 비어 있으면 `tenant_id` 값을 대신 읽습니다. Go는 `*tapmirrors.TapMirror`를 돌려주며 `ProjectID`와 `TenantID`를 따로 둡니다. 생성은 201과 202, 조회와 수정은 200만 성공으로 받으므로 Python보다 성공 코드가 좁고, 오류는 `openstack.exceptions` 대신 gophercloud 오류를 감싼 `resource.OperationError`입니다.

### 삭제

`delete_tap_mirror`의 기본값 `ignore_missing=True`는 `api.Remove(ctx, resource.ID(id))`가 그대로 재현합니다. 요청은 `DELETE /taas/tap_mirrors/{id}` 한 번뿐이고 404는 nil, 409 같은 다른 오류는 그대로 반환됩니다. `ignore_missing=False`는 `resource.WithMissingError()`를 붙이면 되고, 이때 404는 `resource.ErrNotFound`로 확인할 수 있습니다. 직접 호출을 선호하면 `resource.IgnoreMissing(api.Delete(ctx, id))`도 같은 결과를 냅니다. Go는 202와 204만 성공으로 받습니다.

### 조회와 검색

`get_tap_mirror(tap_mirror)`처럼 `fields`를 생략한 호출은 `api.Get(ctx, id)`와 같은 GET을 보냅니다. `fields=["name", ...]`를 주면 Python은 Body 속성 이름을 wire 이름으로 바꿔 `?fields=name&fields=...`를 붙이고 일부 필드만 담긴 Resource를 돌려줍니다. Go에서는 sdkgen이 만든 `GetWithFields(ctx, id, fields...)`는 Python `_get_with_fields`처럼 Resource 상속 체인의 Body 속성 이름을 wire 이름으로 바꾼 뒤 `fields` query를 같은 키로 반복해서 보냅니다. 선언되지 않은 이름, 중복, 순서는 그대로 두고, 필드를 주지 않으면 query 없이 `Get`과 같은 GET을 보냅니다. 성공 상태는 Python 읽기처럼 200부터 399까지 받고, 일부 필드만 담긴 응답은 Gophercloud 모델로 decode되어 빠진 필드가 zero 값이 됩니다. 실패는 `resource.OperationError`로 감싼 native 오류로 돌아오므로 `gophercloud.ResponseCodeIs(err, 404)`로 Python의 `NotFoundException`을 구분합니다.

`find_tap_mirror(name_or_id)`는 먼저 `GET /taas/tap_mirrors/{name_or_id}`를 보내고 404, 400, 403이면 `?name=`으로 목록을 조회해 ID나 이름이 같은 행 하나를 고릅니다. 기본값은 `ignore_missing=True`라서 찾지 못하면 None을 돌려줍니다. Go에서 가장 가까운 방법은 `api.Find(ctx, resource.ID(x), resource.WithIgnoreMissing())`가 nil이면 `api.Find(ctx, resource.Name(x), resource.WithIgnoreMissing())`를 부르는 것입니다. 이렇게 하면 404 뒤의 이름 검색과 중복 이름 거부(`resource.ErrAmbiguous`)는 같지만 400과 403에서는 Go가 목록으로 넘어가지 않고 오류를 반환합니다. Python의 `**query`는 GET과 목록 요청 모두에 붙는데, `Find`는 `LookupOption`만 받으므로 이 query를 보낼 수 없습니다. 이 binding은 `FindIdentity`에도 참여하지 않습니다.

### 목록

`tap_mirrors(**query)`는 `name`, `project_id`, `sort_key`, `sort_dir`, `fields`, `limit`, `marker`만 서버 query로 보냅니다. 나머지 Body 속성(`port_id`, `remote_ip` 등)은 받은 행을 로컬에서 거르고, 알 수 없는 키는 버립니다. Go `api.All`과 `api.List`는 `resource.WithQuery`나 `tapmirrors.WithListQuery`로 준 키를 모두 서버에 보내며 Body 속성 필터는 호출자가 직접 처리해야 합니다. 이름은 `resource.WithName`으로 `?name=`과 정확한 로컬 비교를 함께 적용할 수 있습니다.

두 SDK 모두 `tap_mirrors_links`의 next 링크를 따라갑니다. Python은 링크가 없어도 `limit`만큼 찬 페이지 뒤에 마지막 ID를 marker로 다시 요청하지만 Go는 링크가 없으면 멈춥니다. 무엇보다 Python은 `fields`, `sort_key`, `sort_dir`에 list를 주면 같은 키를 반복해서 보내는데, Go의 두 query 옵션은 키마다 값 하나만 남기고 `ListOpts`에는 이 키들이 없습니다.

## Tap flow와 Tap service

Python [`tap_flow.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/tap_flow.py)는 `/taas/tap_flows`(`tap_flow`), [`tap_service.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/tap_service.py)는 `/taas/tap_services`(`tap_service`)를 사용합니다. 두 자원 모두 tap mirror와 같은 생성, 조회(`fields` 포함), 수정, 삭제, 검색, 목록 메서드를 가집니다. Gophercloud v2.15.0의 `taas` 아래에는 `tapmirrors`만 있으므로 이 SDK에는 두 경로를 호출하는 Go API가 없습니다.

## SFC

networking-sfc 자원은 `/sfc` 아래에 있습니다. Python 정의는 [`sfc_flow_classifier.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/sfc_flow_classifier.py), [`sfc_port_chain.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/sfc_port_chain.py), [`sfc_port_pair.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/sfc_port_pair.py), [`sfc_port_pair_group.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/sfc_port_pair_group.py), [`sfc_service_graph.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/sfc_service_graph.py)입니다.

| 자원 | 경로 | envelope |
|---|---|---|
| flow classifier | `/sfc/flow_classifiers` | `flow_classifier` |
| port chain | `/sfc/port_chains` | `port_chain` |
| port pair | `/sfc/port_pairs` | `port_pair` |
| port pair group | `/sfc/port_pair_groups` | `port_pair_group` |
| service graph | `/sfc/service_graphs` | `service_graph` |

다섯 자원 모두 POST 생성, `GET {id}` 조회, PUT 수정, DELETE 삭제(기본 `ignore_missing=True`), GET-then-list 검색, 목록을 제공합니다. Gophercloud v2.15.0에는 networking-sfc 패키지가 없고 이 SDK도 해당 경로를 호출하는 Go API를 두지 않습니다.

## 미해결 메서드

networking-sfc의 flow classifier·port pair·port pair group·port chain·service graph와 TaaS tap flow·tap service 메서드는 SDK 소유 package가 추가되어 대응으로 바뀌었습니다. 각 package README([flow classifier](extensions/sfc/flowclassifiers/README.md), [port pair](extensions/sfc/portpairs/README.md), [port pair group](extensions/sfc/portpairgroups/README.md), [port chain](extensions/sfc/portchains/README.md), [service graph](extensions/sfc/servicegraphs/README.md), [tap flow](extensions/taas/tapflows/README.md), [tap service](extensions/taas/tapservices/README.md))에 요청과 차이를 설명합니다. 아래 목록에서 이 메서드들은 더 이상 미해결이 아닙니다.

- `find_tap_mirror`: Go 검색은 GET 400과 403 뒤에 목록으로 넘어가지 않고, `**query`를 GET과 목록 요청에 붙일 수 없습니다.
- `tap_mirrors`: `fields`, `sort_key`, `sort_dir`의 반복 값을 보낼 수 없습니다.
- `create_tap_flow`, `get_tap_flow`, `find_tap_flow`, `tap_flows`, `update_tap_flow`, `delete_tap_flow`: `/taas/tap_flows`에 대한 Gophercloud binding과 SDK 소유 호출이 없습니다.
- `create_tap_service`, `get_tap_service`, `find_tap_service`, `tap_services`, `update_tap_service`, `delete_tap_service`: `/taas/tap_services`에 대한 Gophercloud binding과 SDK 소유 호출이 없습니다.
- SFC 30개(`*_sfc_flow_classifier`, `*_sfc_port_chain`, `*_sfc_port_pair`, `*_sfc_port_pair_group`, `*_sfc_service_graph`와 각 목록 메서드): `/sfc` 아래 다섯 경로에 대한 Gophercloud binding과 SDK 소유 호출이 없습니다.
