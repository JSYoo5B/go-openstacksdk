# Barbican Secret Stores

이 패키지는 backend 목록, deployment의 global default, 인증된 프로젝트의
preferred backend를 조회합니다. 세 가지 Python proxy 선언에 대응하며,
secret store 생성·수정·삭제·Find·Wait API를 추가하지 않습니다.

```go
package example

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/keymanager/v1/secretstores"
)

func ReadBackends(ctx context.Context, client *gophercloud.ServiceClient) (*secretstores.SecretStore, *secretstores.SecretStore, error) {
	api := secretstores.New(client)
	global, err := api.GetGlobalDefault(ctx)
	if err != nil {
		return nil, nil, err
	}
	preferred, err := api.GetPreferred(ctx)
	return global, preferred, err
}
```

`client.Type`은 `key-manager`이고 ProviderClient가 있어야 합니다. 원래
provider의 인증·재인증·transport와 `MoreHeaders`를 사용하며, endpoint와
`ResourceBase`의 reverse proxy prefix를 유지합니다. 프로젝트나 admin 역할을
추측하거나 microversion을 자동 선택하지 않습니다. Preferred 조회는 현재
토큰 프로젝트를 사용하고, 없는 경우 global default로 대체하지 않습니다.

| 호출 | 경로 | 성공 응답 |
|---|---|---|
| `GetGlobalDefault` | GET `secret-stores/global-default` | 200, 루트 객체 |
| `GetPreferred` | GET `secret-stores/preferred` | 200, 루트 객체 |
| `List` / `All` | GET `secret-stores` | 200, `secret_stores` 배열 |

단건 404는 `resource.ErrNotFound`와 원래 HTTP 오류를 보존합니다. 기능이
비활성화돼 발생한 404도 그대로 오류입니다. Transport·context 오류를
resource absence로 바꾸지 않습니다. 허용된 200의 읽기·JSON·모델 오류는
`resource.ResponseError`에 원래 body/header/status와 원인을 보존하며 재전송하지
않습니다. 목록의 terminal 오류에서는 `All`이 부분 결과를 반환하지 않습니다.

## 목록 옵션

```go
package example

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/keymanager/v1/secretstores"
)

func ListNonDefaultBackends(ctx context.Context, client *gophercloud.ServiceClient) ([]*secretstores.SecretStore, error) {
	return secretstores.New(client).All(ctx,
		secretstores.WithListOptions(secretstores.ListOpts{Status: "ACTIVE", Limit: 10}),
		secretstores.WithListGlobalDefault(false),
		secretstores.WithListMaxItems(5),
		secretstores.WithListPaginated(false),
	)
}
```

`ListOpts`의 `Name`, `Status`, `GlobalDefault`, `CryptoPlugin`, `SecretStorePlugin`,
`Created`, `Updated`, `Limit`, `Marker`는 서버 query입니다. 이름·상태를 로컬에서
다시 비교하지 않습니다. `GlobalDefault`의 nil은 생략이고 false도 명시적으로
전송됩니다. 빈 문자열 필드는 생략합니다. `WithListQuery`는 명시적 빈 문자열이나
추가 wire query를 전달하며 같은 이름의 typed query보다 우선합니다.

`WithListOptions`는 pointer 값을 생성 시점과 적용 시점에 각각 복사합니다.
`List`는 lazy하며 option slice를 보유한 iterator를 재사용할 수 있습니다.
`MaxItems`는 0이면 무제한, 음수면 HTTP 전에 오류입니다. `Paginated`가 nil 또는
true이면 계속 순회하고 false이면 첫 페이지만 읽습니다. 이 두 값은 URL로 보내지
않으며 raw query의 `max_items`·`paginated` 등의 SDK control 이름도 거부합니다.

MaxItems가 있고 limit이 없으면 limit hint를 보냅니다. 명시적 limit이 있을 때만
링크 없는 비어 있지 않은 페이지에서 marker fallback을 사용하며, 짧은 페이지도
계속합니다. Limit이 없으면 광고된 링크만 따르고 빈 페이지는 종료합니다.
이는 pinned Python Resource의 순회 전략이며 Barbican 서버가 marker pagination을
지원한다는 live-cloud 검증은 아닙니다.

Fallback marker는 응답의 정확한 `id`가 존재하면 그 값이고, 아니면 **원래 전체
`secret_store_ref` 문자열**입니다. `id:null`·빈 id는 ref로 대체하지 않으며 필요한
marker가 비어 있으면 오류입니다. 모델의 편의용 `ID`를 marker로 바꾸지 않습니다.
목록은 `next`, `links`, `secret_stores_links`, HTTP Link continuation을 처리하고
같은 origin·정확한 collection 경로를 유지합니다. 필터 변경, 다른 경로·host 및
순환을 거부합니다. 이 경로 보호와 control 종료 직후 링크 검사를 생략하는 정책은
명시적인 Go 정책입니다.

## 응답과 Python 차이

`SecretStore`는 typed name/status/ref, nullable plugin·default flag, 원문 timestamp
문자열을 제공합니다. `Body`는 생략/null과 알 수 없는 JSON 필드 및 큰 숫자를
그대로 보존하고 `Header`·`StatusCode`는 실제 응답입니다. 반환된 모델은 호출자가
소유합니다. Typed pointer는 생략과 null 모두 nil이므로 둘의 구분에는 `Body`를
사용합니다.

편의용 `ID`는 정확한 소문자 `id`가 있으면 그 값을 우선하며 null은 빈 문자열로
표현합니다. id가 없으면 absolute `secret_store_ref`의 마지막 **원문** path
component를 추출합니다. Unicode·공백·기존 `%xx` spelling을 보존하고
query/fragment는 제외하며 trailing slash는 빈 ID입니다. UUID 형식·현재 origin과의
일치를 강제하지 않습니다. 외부 host의 ref도 수동으로 보관하며 **절대 따라가지
않습니다**. 선언된 typed 필드는 정확한 wire 키만 디코딩합니다. Case variant
확장 필드는 원본 Body에 남지만 canonical 필드를 대체하거나 typed decode 오류를
일으키지 않습니다.

Pinned Python의 `Resource.id`는 alternate ref의 formatter를 거치지 않아 전체 URL을
반환하고 `secret_store_id` getter가 마지막 component를 제공합니다. Go의 편의용
ID와 원문 marker 분리는 이 차이를 명시합니다. Go는 typed 문자열·boolean을
검사하고, literal id가 없어 편의용 ID를 추출할 때만 absolute reference 문법을
검사합니다. Literal id가 있으면 ref URL을 파싱하지 않습니다. Malformed 응답은
오류로 반환합니다.
Python descriptor의 값 변환·일부 URI 허용 범위와 동일하다고 주장하지 않습니다.

`WithListQuery`는 Go의 wire extension이므로 Python의 unknown query discard,
Body attribute 로컬 필터·alias, JMESPath와 같지 않습니다. Inherited Resource의
cache/dirty state, model overload, generic per-call base path/version/header
controls도 이 세 API의 구현 범위 밖입니다. 전체 Python Resource parity를
주장하지 않습니다.

검증 근거: [HTTP 계약 테스트](../../../api/keymanager_secretstores_test.go).
Source pin은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의
`openstack/key_manager/v1/_proxy.py:361–392`, `secret_store.py:17–58`,
`_format.py:18–30`과 `Resource.__getattribute__`, `Resource.list`,
`Resource._get_next_link`입니다. REST 상태·응답 형식은
[공식 Secret Stores API](https://docs.openstack.org/barbican/wallaby/api/reference/store_backends.html)를
참조했습니다. 테스트는 격리된 HTTP fixture이며 실제 cloud 테스트가 아닙니다.
