# Python openstacksdk identity v3 policy·federation 대응

이 문서는 openstacksdk [`identity/v3/_proxy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py)의 policy, federation protocol, mapping, identity provider, service provider Proxy 메서드를 이 SDK의 Go 호출과 맞춰 봅니다. 요청 경로, query, 본문과 decode 결과는 [`policies/python_parity_test.go`](policies/python_parity_test.go)와 [`federation/python_parity_test.go`](federation/python_parity_test.go)가 고정합니다. generated 메서드 자체의 status와 본문 규칙은 [native-admin-b.md](native-admin-b.md)에 있습니다.

판정은 두 가지입니다. `대응`은 Python 인자의 효과를 공개 Go API로 모두 재현할 수 있다는 뜻이고, 아래 절에 적은 차이는 남습니다. `미해결`은 Go에 해당 요청이나 Python이 돌려주는 정보, 기본 동작이 없다는 뜻입니다.

| Python 메서드 | Go 호출 | 판정 |
|---|---|---|
| `create_policy(**attrs)` | `Policies.Create(ctx, CreateOpts{Type, Blob, Extra})` | 대응 |
| `delete_policy(policy, ignore_missing=True)` | `Policies.Remove(ctx, resource.ID(id))` 또는 `resource.IgnoreMissing(Policies.Delete(ctx, id))` | 대응 |
| `find_policy(name_or_id, ignore_missing=True)` | `Policies.Find(ctx, resource.ID(v), resource.WithIgnoreMissing())`만 가능 | 미해결 |
| `get_policy(policy)` | `Policies.Get(ctx, id)`, `Policies.Find(ctx, resource.ID(id))` | 대응 |
| `policies(**query)` | `Policies.List(ctx, ...)`, `Policies.All(ctx, resource.WithQuery(...))` | 미해결 |
| `update_policy(policy, **attrs)` | `Policies.Update(ctx, id, UpdateOpts{...}, WithUpdateField(...))` | 대응 |
| `create_mapping(**attrs)` | `Federation.CreateMapping(ctx, id, CreateMappingOpts{Rules}, WithCreateMappingField(...))` | 미해결 |
| `delete_mapping(mapping, ignore_missing=True)` | `resource.IgnoreMissing(Federation.DeleteMapping(ctx, id))` | 대응 |
| `find_mapping(name_or_id, ignore_missing=True)` | 없음 | 미해결 |
| `get_mapping(mapping)` | `Federation.GetMapping(ctx, id)` | 미해결 |
| `mappings(**query)` | `Federation.ListMappings(ctx)` | 미해결 |
| `update_mapping(mapping, **attrs)` | `Federation.UpdateMapping(ctx, id, UpdateMappingOpts{Rules}, ...)` | 미해결 |
| `create_identity_provider(**attrs)` | 없음 | 미해결 |
| `delete_identity_provider(identity_provider, ignore_missing=True)` | 없음 | 미해결 |
| `find_identity_provider(name_or_id, ignore_missing=True)` | 없음 | 미해결 |
| `get_identity_provider(identity_provider)` | 없음 | 미해결 |
| `identity_providers(**query)` | 없음 | 미해결 |
| `update_identity_provider(identity_provider, **attrs)` | 없음 | 미해결 |
| `create_federation_protocol(idp, **attrs)` | 없음 | 미해결 |
| `delete_federation_protocol(idp, protocol, ignore_missing=True)` | 없음 | 미해결 |
| `find_federation_protocol(idp, protocol, ignore_missing=True)` | 없음 | 미해결 |
| `get_federation_protocol(idp, protocol)` | 없음 | 미해결 |
| `federation_protocols(idp, **query)` | 없음 | 미해결 |
| `update_federation_protocol(idp, protocol, **attrs)` | 없음 | 미해결 |
| `create_service_provider(**attrs)` | 없음 | 미해결 |
| `delete_service_provider(service_provider, ignore_missing=True)` | 없음 | 미해결 |
| `find_service_provider(name_or_id, ignore_missing=True)` | 없음 | 미해결 |
| `get_service_provider(service_provider)` | 없음 | 미해결 |
| `service_providers(**query)` | 없음 | 미해결 |
| `update_service_provider(service_provider, **attrs)` | 없음 | 미해결 |

표의 `Policies`는 `policies.New(client)`, `Federation`은 `federation.New(client)`가 돌려준 API입니다. 30개 중 대응은 5개, 미해결은 25개입니다.

## 공통 차이

Python은 Resource 인스턴스나 ID 문자열을 받고, Go는 ID 문자열이나 `resource.Ref`만 받습니다. Keystone identity API에는 microversion이 없으므로 양쪽 모두 microversion 헤더를 보내지 않습니다. Python은 400 미만의 모든 응답을 성공으로 받지만, Go generated 호출은 native가 정한 status만 받습니다. 그래서 생성은 201, 조회와 수정은 200, 삭제는 202와 204만 성공이고 나머지는 `gophercloud.ErrUnexpectedResponseCode`가 됩니다. Python의 실패는 `NotFoundException` 같은 openstacksdk 예외이고, Go는 Gophercloud 오류를 `resource.OperationError`로 감쌉니다.

## policy

출처는 [`_proxy.py` Policies 절](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L912-L1015)과 [`policy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/policy.py)입니다.

`create_policy`는 `POST policies`에 `{"policy": {...}}`로 지정한 속성만 보냅니다. Go `Create`는 `Type`과 `Blob`을 같은 envelope에 넣고 `project_id`, `user_id` 같은 나머지 속성은 `Extra`로 보냅니다. Keystone이 `type`과 `blob`을 필수로 요구하므로 Go는 둘 중 하나가 없으면 HTTP 전에 거부하고, `type`이 255자를 넘을 때도 같습니다. Python은 이런 요청도 서버로 보낸 뒤 400을 받습니다. 응답의 `project_id`와 `user_id`는 Python에서 속성이지만 Go에서는 `Policy.Extra` map 안에 있습니다.

`get_policy`는 `Get(ctx, id)`와 같은 `GET policies/{id}`를 보냅니다. 없을 때 Python은 `NotFoundException`을 내고, `Get`은 404 Gophercloud 오류를, `Find(ctx, resource.ID(id))`는 `resource.ErrNotFound`를 돌려줍니다.

`update_policy`는 바뀐 속성만 `PATCH policies/{id}`에 담습니다. Go `UpdateOpts`는 빈 `Type`과 빈 `Blob`을 생략하므로 빈 문자열을 보내려면 `WithUpdateField("blob", "")`나 `Extra`를 씁니다. `Extra`에 nil을 넣으면 JSON null이 됩니다. 바꿀 속성이 하나도 없으면 Python은 요청을 보내지 않지만 Go는 `{"policy": {}}` PATCH를 보냅니다.

`delete_policy`의 기본값 `ignore_missing=True`는 `Remove`가 그대로 따릅니다. 404는 nil이 되고 409 같은 다른 오류는 남으며, `resource.WithMissingError()`를 주면 `ignore_missing=False`처럼 `resource.ErrNotFound`를 돌려줍니다. 직접 `Delete`를 쓸 때는 `resource.IgnoreMissing`으로 감싸야 같은 기본값이 됩니다.

`policies(**query)`는 미해결입니다. Policy의 query mapping에는 `limit`과 `marker`만 있어서 Python은 `type`, `project_id`, `user_id`, `blob` 같은 Body 속성을 server query로 보내지 않고, `GET policies`의 결과를 받은 뒤 로컬에서 거릅니다. Go는 인자가 없는 호출과 `links.next` 추적, `limit`과 `marker` query는 똑같이 할 수 있고, `type`은 Keystone이 지원하는 server 필터로 같은 결과를 얻습니다. 그러나 policy collection에는 로컬 Body 필터가 없어 `resource.WithBodyFilter`와 `resource.WithName`이 `resource.ErrUnsupported`를 돌려주고, Keystone은 `project_id`나 `user_id` query를 무시합니다.

`find_policy`도 미해결입니다. Python은 `GET policies/{name_or_id}`가 400, 403, 404이면 `GET policies` 전체를 보고 ID나 name이 같은 항목을 고르며, 기본으로 없으면 None을 돌려줍니다. 이 메서드는 deprecated 경고도 냅니다. Go `Find(ctx, resource.ID(v), resource.WithIgnoreMissing())`는 404만 nil로 바꾸고 목록으로 넘어가지 않으며, name 조회와 `FindIdentity`는 policy binding이 지원하지 않습니다.

## federation mapping

출처는 [`_proxy.py` Mappings 절](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2913-L3020)과 [`mapping.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/mapping.py)입니다.

`delete_mapping`만 대응합니다. Go generated `DeleteMapping`은 404를 오류로 돌려주므로 `resource.IgnoreMissing(api.DeleteMapping(ctx, id))`로 감싸면 Python 기본값과 같아집니다. `ignore_missing=False`는 감싸지 않은 호출이고, 이때 404는 Gophercloud 오류로 남습니다.

나머지 mapping 메서드는 경로가 같아도 미해결입니다. Gophercloud v2.15.0 `Mapping`에는 `schema_version` 필드가 없고 규칙을 고정된 `MappingRule` 구조체로 decode합니다. 그래서 Python이 돌려주는 `schema_version`과 구조체 밖의 규칙 key는 Go 결과에서 사라집니다. 요청 쪽에서도 `rules`는 typed 필드라 Python처럼 임의의 JSON 목록을 보낼 수 없고, 확장 필드로 `rules`를 넣으면 HTTP 전에 거부됩니다. `UpdateMappingOpts.Rules`에는 omitempty가 없어 `update_mapping(m, schema_version="2.0")`과 같은 요청을 만들 수 없습니다. Go는 항상 `"rules": null`을 함께 보냅니다. `create_mapping`은 Python처럼 본문에 `id`를 넣으려면 `WithCreateMappingField("id", id)`를 쓰면 되지만, 결과에서 `schema_version`이 빠지는 문제는 같습니다. `mappings(**query)`에 대응하는 `ListMappings`는 옵션을 받지 않아 `limit`, `marker`와 로컬 Body 필터를 표현할 수 없습니다. `find_mapping`은 GET 다음 목록으로 넘어가는 helper나 mapping collection이 없습니다.

## identity provider, federation protocol, service provider

출처는 [`_proxy.py` Federation Protocols 절](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2725-L2912), [Identity Providers 절](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L3022-L3150), [Service Providers 절](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L3218-L3340)과 [`identity_provider.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/identity_provider.py), [`federation_protocol.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/federation_protocol.py), [`service_provider.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/service_provider.py)입니다.

Gophercloud v2.15.0 `identity/v3/federation` 패키지는 mapping만 다루므로 이 18개 메서드에는 Go API가 없습니다. Python이 보내는 요청은 다음과 같고, 지원하려면 SDK가 직접 호출을 만들어야 합니다.

- identity provider: `PUT OS-FEDERATION/identity_providers/{id}`에 `{"identity_provider": {...}}`를 보내고 본문에서 `id`를 뺍니다. 조회, 수정, 삭제는 같은 member 경로의 `GET`, `PATCH`, `DELETE`이고 목록은 `GET OS-FEDERATION/identity_providers`에 `id`, `enabled`(Python 이름 `is_enabled`) query를 붙입니다.
- federation protocol: `PUT OS-FEDERATION/identity_providers/{idp_id}/protocols/{id}`에 `{"protocol": {"mapping_id": ...}}`를 보내고 본문에서 `id`를 뺍니다. 조회, 수정, 삭제는 같은 경로의 `GET`, `PATCH`, `DELETE`이고 목록은 `GET OS-FEDERATION/identity_providers/{idp_id}/protocols`에 `id` query를 붙입니다. `idp`를 생략하면 Python은 FederationProtocol 인스턴스의 `idp_id`를 씁니다.
- service provider: `PUT OS-FEDERATION/service_providers/{id}`에 `{"service_provider": {...}}`를 보내고 본문에서 `id`를 뺍니다. 조회, 수정, 삭제는 같은 경로의 `GET`, `PATCH`, `DELETE`이고 목록은 `GET OS-FEDERATION/service_providers`에 `id`, `enabled` query를 붙입니다.

세 리소스의 `delete_*`는 기본으로 404를 무시하고, `find_*`는 GET이 400, 403, 404이면 목록에서 ID나 name을 찾으며 없으면 None을 돌려줍니다. 세 리소스 모두 `name`이 `id`와 같은 Body 필드라서 목록 단계는 ID 비교와 같습니다. `find_identity_provider`는 deprecated 경고를 냅니다.

## 미해결 목록

- `find_policy`: GET 400/403 뒤의 목록 fallback과 목록에서 ID나 name을 고르는 동작이 policy collection에 없습니다.
- `policies`: Python이 로컬에서 거르는 `project_id`, `user_id`, `blob` 같은 Body 필터를 Go policy collection이 지원하지 않습니다.
- `create_mapping`, `get_mapping`, `update_mapping`: 결과에서 `schema_version`과 구조체 밖의 규칙 key가 빠지고, 규칙을 임의 JSON으로 보낼 수 없습니다. `update_mapping`은 `rules` 없이 보내는 요청도 만들 수 없습니다.
- `mappings`: `ListMappings`가 query와 로컬 필터를 받지 않고 결과에서 `schema_version`이 빠집니다.
- `find_mapping`: GET 다음 목록 fallback을 하는 Go helper가 없습니다.
- identity provider, federation protocol, service provider의 18개 메서드: Gophercloud v2.15.0에 해당 경로의 binding이 없습니다.
