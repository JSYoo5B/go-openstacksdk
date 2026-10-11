# Keystone federation service provider

이 패키지는 pinned openstacksdk의 service provider Proxy 메서드를 Go로 제공합니다. Gophercloud v2.15.0 `identity/v3/federation`은 `OS-FEDERATION/mappings`만 다루므로, 이 패키지가 `OS-FEDERATION/service_providers` 요청을 SDK에서 직접 보냅니다. 기준 소스는 [`_proxy.py` Service Providers 절](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L3218-L3340)과 [`service_provider.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/service_provider.py)입니다.

| Python `conn.identity` | Go `serviceproviders.API` | 요청 |
|---|---|---|
| `create_service_provider(**attrs)` | `Create(ctx, id, options...)` | `PUT OS-FEDERATION/service_providers/{id}` |
| `get_service_provider(service_provider)` | `Get(ctx, id, options...)` | `GET OS-FEDERATION/service_providers/{id}` |
| `update_service_provider(service_provider, **attrs)` | `Update(ctx, id, options...)` | `PATCH OS-FEDERATION/service_providers/{id}` |
| `delete_service_provider(service_provider, ignore_missing=True)` | `Delete(ctx, id, options...)` | `DELETE OS-FEDERATION/service_providers/{id}` |
| `service_providers(**query)` | `List(ctx, options...)`, `All(ctx, options...)` | `GET OS-FEDERATION/service_providers` |
| `find_service_provider(name_or_id, ignore_missing=True)` | `Find(ctx, nameOrID, options...)` | GET 뒤 목록 |

```python
sp = conn.identity.create_service_provider(
    id="cloud2", auth_url="https://cloud2/v3/OS-FEDERATION/identity_providers/corp/protocols/saml2/auth",
    sp_url="https://cloud2/Shibboleth.sso/SAML2/ECP", is_enabled=True)
conn.identity.update_service_provider("cloud2", description="두 번째 리전")
conn.identity.delete_service_provider("cloud2")
```

```go
api := serviceproviders.New(identityClient)
sp, err := api.Create(ctx, "cloud2",
    serviceproviders.WithCreateField("auth_url", "https://cloud2/v3/OS-FEDERATION/identity_providers/corp/protocols/saml2/auth"),
    serviceproviders.WithCreateField("sp_url", "https://cloud2/Shibboleth.sso/SAML2/ECP"),
    serviceproviders.WithCreateField("is_enabled", true))
if err != nil { return err }
_, err = api.Update(ctx, sp.ID, serviceproviders.WithUpdateField("description", "두 번째 리전"))
if err != nil { return err }
err = api.Delete(ctx, sp.ID) // 없는 provider는 무시합니다
```

## 요청과 응답

생성은 Python `Resource.create`와 같이 PUT을 씁니다. Python은 `id`(또는 같은 Body 키를 쓰는 `name`)로 member 경로를 만들고 본문에서 `id`를 지우므로, Go는 ID를 인자로 받고 `id`와 `name` 필드는 본문에 넣지 않습니다. 나머지 속성은 `{"service_provider": {...}}`에 들어가며, `WithCreateField`의 키로 Python 속성 이름(`is_enabled`)과 Body 키(`enabled`)를 모두 쓸 수 있습니다. 선언되지 않은 키는 Python `Resource.new`처럼 버립니다. 값은 JSON 그대로 보내므로 null, false, 빈 목록도 그대로 전달됩니다. 같은 Body 키에 서로 다른 값을 주면 Python은 둘 중 하나를 임의로 고르지만 Go는 HTTP 전에 거부합니다.

수정은 `Resource.commit`처럼 지정한 속성만 PATCH 본문에 담습니다. ID는 dirty 속성이 될 수 없으므로 `id`와 `name` 필드는 무시하고, 보낼 속성이 없으면 요청 없이 ID만 채운 결과를 돌려줍니다.

Python은 400 미만의 모든 status를 성공으로 받으므로 Go도 200부터 399까지를 받습니다. 응답 본문이 `service_provider` 객체를 가지면 그 안을, 아니면 최상위 객체를 읽어 선언된 키만 요청 값 위에 덮어씁니다. 비어 있거나 JSON이 아닌 본문은 요청 값을 그대로 두고, JSON이지만 객체가 아니면 실제 응답을 담은 `resource.ResponseError`로 실패합니다. 400 이상은 Gophercloud `ErrUnexpectedResponseCode`이고, 모든 오류는 `resource.OperationError`(Resource `service_providers`, Operation `Create`, `Get`, `Update`, `Delete`, `List`, `Find`)로 감쌉니다.

`ServiceProvider`의 typed 필드는 JSON 타입이 맞을 때만 채워집니다. `Body`는 선언된 Body 키 6개(`id`, `auth_url`, `description`, `enabled`, `relay_state_prefix`, `sp_url`)를 raw JSON으로 보관하고 값이 없으면 null입니다. `Wire`는 실제 응답 객체 전체, `Envelope`는 실제 응답 본문입니다.

## 삭제

`Delete`는 Python 기본값 `ignore_missing=True`를 따라 깨끗한 404를 nil로 바꿉니다. `WithDeleteIgnoreMissing(false)`를 주면 404도 오류로 돌려주고, 409나 500 같은 다른 status는 언제나 오류입니다.

## 목록

`List`는 lazy iterator이며 반복할 때마다 요청을 다시 보냅니다. Python의 `**query`는 `WithListQuery(key, value)`로 넘깁니다. `id`와 `is_enabled`(또는 `enabled`)는 Requests 방식으로 인코딩한 server query가 되고, 그래서 `true`는 `enabled=True`로 나갑니다. `name`, `auth_url`, `description`, `relay_state_prefix`, `sp_url`은 Python처럼 응답 행을 로컬에서 거르고, 그 밖의 이름은 무시합니다. `WithListID`와 `WithListEnabled`는 자주 쓰는 query의 편의 함수입니다. 요청에는 `Accept: application/json`이 붙습니다.

`WithListLimit`, `WithListMarker`, `WithListMaxItems`, `WithListPaginated(false)`는 `limit`, `marker`, `max_items`, `paginated`와 같습니다. limit을 주면 Python처럼 빈 페이지가 나올 때까지 마지막 ID를 marker로 삼아 다음 페이지를 요청합니다. max_items만 주면 그 값을 limit으로 보내고, 로컬 필터보다 먼저 행 수를 셉니다.

## 찾기

`Find`는 `Resource.find`처럼 먼저 `GET OS-FEDERATION/service_providers/{name_or_id}`를 보냅니다. 응답이 깨끗한 400, 403, 404이면 전체 목록을 읽어 ID가 정확히 같은 항목을 고릅니다. 이 리소스의 `name`은 `id`와 같은 Body 키라서 이름 비교도 ID 비교와 같습니다. 없으면 기본으로 `nil, nil`이고, `WithFindIgnoreMissing(false)`이면 `resource.NotFoundError`입니다. 같은 ID가 둘 이상이면 오류이며, 500 같은 다른 실패에서는 목록으로 넘어가지 않습니다.

## Python과 다른 점

Python은 Resource 인스턴스나 ID 문자열을 받지만 Go는 ID 문자열만 받습니다. Go는 ID를 URL segment 하나로 escape하고 빈 ID, `.`과 `..`, 제어 문자를 HTTP 전에 거부합니다. Python은 ID를 escape하지 않고 경로에 그대로 붙입니다. Python `update_service_provider(id, name=...)`는 `name`이 같은 Body 키라서 요청 경로를 바꿀 수 있지만, Go는 인자로 받은 ID만 경로로 씁니다. 목록의 로컬 필터는 JSON 값을 비교하므로 bool과 숫자를 구분하고, Python의 타입 변환(`is_enabled`의 bool 변환 등)은 하지 않습니다. 목록은 Keystone `links.next`가 문자열이면 따라가는데, pinned Python 페이지네이터는 이 dict 형태를 무시합니다. `find_service_provider`의 deprecated 경고와 Resource의 `location` 속성은 재현하지 않습니다. 헤더 옵션은 Go 확장이며 인증과 content 헤더는 SDK가 소유하므로 거부합니다.
