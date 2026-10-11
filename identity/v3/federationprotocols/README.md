# Keystone federation protocol

이 패키지는 pinned openstacksdk의 federation protocol Proxy 메서드를 Go로 제공합니다. Gophercloud v2.15.0 `identity/v3/federation`은 `OS-FEDERATION/mappings`만 다루므로, 이 패키지가 `OS-FEDERATION/identity_providers/{idp_id}/protocols` 요청을 SDK에서 직접 보냅니다. 기준 소스는 [`_proxy.py` Federation Protocols 절](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2725-L2912)과 [`federation_protocol.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/federation_protocol.py)입니다.

| Python `conn.identity` | Go `federationprotocols.API` | 요청 |
|---|---|---|
| `create_federation_protocol(idp, **attrs)` | `Create(ctx, idpID, id, options...)` | `PUT .../identity_providers/{idp_id}/protocols/{id}` |
| `get_federation_protocol(idp, protocol)` | `Get(ctx, idpID, id, options...)` | `GET .../protocols/{id}` |
| `update_federation_protocol(idp, protocol, **attrs)` | `Update(ctx, idpID, id, options...)` | `PATCH .../protocols/{id}` |
| `delete_federation_protocol(idp, protocol, ignore_missing=True)` | `Delete(ctx, idpID, id, options...)` | `DELETE .../protocols/{id}` |
| `federation_protocols(idp, **query)` | `List(ctx, idpID, options...)`, `All(ctx, idpID, options...)` | `GET .../identity_providers/{idp_id}/protocols` |
| `find_federation_protocol(idp, protocol, ignore_missing=True)` | `Find(ctx, idpID, nameOrID, options...)` | `GET .../protocols/{id}?idp_id={idp_id}` 뒤 목록 |

```python
conn.identity.create_federation_protocol("corp", id="saml2", mapping_id="corp-mapping")
protocol = conn.identity.find_federation_protocol("corp", "saml2")
conn.identity.delete_federation_protocol("corp", "saml2")
```

```go
api := federationprotocols.New(identityClient)
_, err := api.Create(ctx, "corp", "saml2",
    federationprotocols.WithCreateField("mapping_id", "corp-mapping"))
if err != nil { return err }
protocol, err := api.Find(ctx, "corp", "saml2")
if err != nil { return err }
err = api.Delete(ctx, "corp", protocol.ID) // 없는 protocol은 무시합니다
```

## 요청과 응답

모든 메서드는 identity provider ID를 첫 인자로 받습니다. Python은 `idp`를 생략하고 FederationProtocol 인스턴스의 `idp_id`를 쓸 수 있지만, ID 문자열만 넘기면서 `idp`를 생략하면 요청 전에 실패합니다. Go는 인스턴스 입력이 없으므로 항상 ID를 받습니다. `IdentityProviderID`는 이 인자로 채우며 응답에서 읽지 않습니다.

생성은 Python `Resource.create`와 같이 PUT을 씁니다. protocol ID가 member 경로가 되고 본문 `{"protocol": {...}}`에서 `id`(또는 같은 Body 키를 쓰는 `name`)는 빠집니다. 선언된 Body 속성은 `mapping_id` 하나라서 그 밖의 키는 Python `Resource.new`처럼 버리고, `idp_id` 필드도 본문에 넣지 않습니다. 수정은 지정한 `mapping_id`만 PATCH로 보내며, 보낼 속성이 없으면 요청 없이 ID만 채운 결과를 돌려줍니다.

Python은 400 미만의 모든 status를 성공으로 받으므로 Go도 200부터 399까지를 받습니다. 응답 본문이 `protocol` 객체를 가지면 그 안을, 아니면 최상위 객체를 읽어 선언된 키만 요청 값 위에 덮어씁니다. 비어 있거나 JSON이 아닌 본문은 요청 값을 그대로 두고, JSON이지만 객체가 아니면 실제 응답을 담은 `resource.ResponseError`로 실패합니다. 400 이상은 Gophercloud `ErrUnexpectedResponseCode`이고, 모든 오류는 `resource.OperationError`(Resource `federation_protocols`)로 감쌉니다. `Body`는 선언된 Body 키 `id`와 `mapping_id`를 raw JSON으로, `Wire`는 실제 응답 객체 전체를 보관합니다.

`Delete`는 Python 기본값 `ignore_missing=True`를 따라 깨끗한 404를 nil로 바꾸고, `WithDeleteIgnoreMissing(false)`이면 404도 오류로 돌려줍니다.

## 목록과 찾기

`List`는 lazy iterator입니다. `WithListQuery`에서 `id`는 server query가 되고, `name`과 `mapping_id`는 Python처럼 응답 행을 로컬에서 거르며, 그 밖의 이름은 무시합니다. 요청에는 `Accept: application/json`이 붙고, `WithListLimit`을 주면 빈 페이지가 나올 때까지 마지막 ID를 marker로 삼아 다음 페이지를 요청합니다. `WithListMarker`, `WithListMaxItems`, `WithListPaginated(false)`도 Python `marker`, `max_items`, `paginated`와 같습니다.

`Find`는 `Resource.find`가 URI 속성을 fetch에 query로 넘기는 동작까지 따라서 `GET .../protocols/{name_or_id}?idp_id={idp_id}`를 먼저 보냅니다. 이 응답이 깨끗한 400, 403, 404이면 query 없이 protocol 목록 전체를 읽어 ID가 정확히 같은 항목을 고릅니다. `name`은 `id`와 같은 Body 키라서 이름 비교도 ID 비교와 같습니다. 없으면 기본으로 `nil, nil`이고, `WithFindIgnoreMissing(false)`이면 `resource.NotFoundError`입니다.

## Python과 다른 점

Go는 identity provider ID와 protocol ID를 각각 URL segment 하나로 escape하고, 빈 ID, `.`과 `..`, 제어 문자를 HTTP 전에 거부합니다. Python은 두 값을 escape하지 않고 경로에 그대로 붙입니다. Python `update_federation_protocol(idp, id, name=...)`는 `name`이 같은 Body 키라서 요청 경로를 바꿀 수 있지만, Go는 인자로 받은 ID만 경로로 씁니다. 목록은 Keystone `links.next`가 문자열이면 따라가는데, pinned Python 페이지네이터는 이 dict 형태를 무시합니다. 로컬 필터는 JSON 값을 비교합니다. Resource의 `location` 속성은 재현하지 않고, 헤더 옵션은 Go 확장이며 인증과 content 헤더는 SDK가 소유하므로 거부합니다.
