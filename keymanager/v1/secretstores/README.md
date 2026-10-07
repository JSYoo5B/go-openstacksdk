# Barbican Secret Stores

이 패키지는 backend 목록, deployment의 global default, 인증된 프로젝트의
preferred backend를 조회합니다. 세 가지 Python proxy 선언에 대응하며,
secret store 생성·수정·삭제·Find·Wait API를 추가하지 않습니다.

```go
package example

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/JSYoo5B/gophercloudsdk/keymanager/v1/secretstores"
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

| Python | Go 호출 | 경로 / 성공 응답 |
|---|---|---|
| `conn.key_manager.get_global_default_secret_store()` | `service.SecretStores.GetGlobalDefault(ctx)` | GET `secret-stores/global-default` / 200 루트 객체 |
| `conn.key_manager.get_preferred_secret_store()` | `service.SecretStores.GetPreferred(ctx)` | GET `secret-stores/preferred` / 200 루트 객체 |
| `conn.key_manager.secret_stores(**query)` | `service.SecretStores.List` / `All` | GET `secret-stores` / 200 `secret_stores` 배열 |

단건 404는 `resource.ErrNotFound`와 원래 HTTP 오류를 보존합니다. 기능이
비활성화돼 발생한 404도 그대로 오류입니다. Transport·context 오류를
resource absence로 바꾸지 않습니다. 허용된 200의 읽기·JSON·모델 오류는
`resource.ResponseError`에 원래 body/header/status와 원인을 보존합니다. SDK는 이 오류를 이유로 조합 단계의 재전송을 추가하지 않으며, configured Provider의 native retry·재인증은 별도 transport 정책을 유지합니다. 목록의 terminal 오류에서는 `All`이 부분 결과를 반환하지 않습니다.

## 목록 옵션

```go
package example

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/JSYoo5B/gophercloudsdk/keymanager/v1/secretstores"
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

## 선언된 속성 필터

Python의 `secret_stores(**query)`처럼 선언된 속성을 전달하려면 `WithListFilter` 또는 `WithListFilters`를 사용합니다. SDK가 서버 query와 응답의 로컬 조건을 구분합니다. `name`/`status`는 서버 query이며 로컬 exact-match를 추가하지 않습니다. Unknown semantic 속성은 버리고, `WithListQuery`는 미선언 이름도 서버에 전달하는 별도의 wire 확장입니다.

| 분류 | 속성 |
|---|---|
| 서버 query | `name`, `status`, `global_default`, `crypto_plugin`, `secret_store_plugin`, `created`, `updated`, `limit`, `marker` |
| 로컬 Body 비교 | `id`, `created_at`, `updated_at`, `secret_store_ref`, `secret_store_id` |

```python
stores = conn.key_manager.secret_stores(
    name="backend-pattern", secret_store_id="store-id", limit=10,
)
```

```go
stores, err := service.SecretStores.All(ctx,
    secretstores.WithListFilter("name", "backend-pattern"),
    secretstores.WithListFilter("secret_store_id", "store-id"),
    secretstores.WithListOptions(secretstores.ListOpts{Limit: 10}),
)
```

`created_at`/`updated_at`은 원문 `created`/`updated`를 비교하며 `created`/`updated` 서버 query와 구분합니다. `id`는 literal 원문 id가 있으면 null/empty를 포함해 그 값을 사용하고, 없으면 전체 원래 `secret_store_ref`를 비교합니다. `secret_store_id`만 ref의 마지막 원문 path component를 비교하므로 반환 모델의 편의용 `ID`와 다른 값일 수 있습니다. 생략/null은 null 조건이고 selected formatter가 해석할 수 없는 ref는 로컬 조건 평가 오류입니다. 이 오류는 원인을 보존하지만 새 HTTP ResponseError를 합성하지 않습니다. 참조는 HTTP 대상이 되지 않습니다.

개별 속성의 마지막 옵션이 우선합니다. `WithListFilters`는 semantic 집합만 교체하고 nil/empty는 비웁니다. 값은 옵션 생성 시 복사하며 검증은 순회할 때 합니다. typed/raw query와 같은 wire 키의 semantic query를 함께 지정하면 값이 같아도 HTTP 전 오류입니다. 로컬 Body와 raw wire query는 서로 다른 namespace입니다. Query boolean은 소문자로 보내며 scalar 배열은 반복 query, null/빈 배열은 URL 값 생략입니다. query 객체는 오류입니다. Go의 공통 JSON 비교는 boolean과 number를 구분하고 원문 큰 숫자·생략/null을 보존합니다. Python의 `True == 1` 비교와 다릅니다.

로컬 필터에서 제외된 행도 `MaxItems`를 소모하며 결과 수를 채우는 추가 페이지를 요청하지 않습니다. 링크·marker와 첫 페이지 옵션은 기존 목록 엔진을 사용합니다. 별도 builder나 predicate를 구현할 필요가 없습니다.

`WithListOptions`는 pointer 값을 생성 시점과 적용 시점에 각각 복사합니다.
`List`는 lazy하며 option slice를 보유한 iterator를 재사용할 수 있습니다.
`MaxItems`는 0이면 무제한, 음수면 HTTP 전에 오류입니다. `Paginated`가 nil 또는
true이면 계속 순회하고 false이면 첫 페이지만 읽습니다. 이 두 값은 URL로 보내지
않으며 raw query의 `max_items`·`paginated` 등의 SDK control 이름도 거부합니다.

MaxItems가 있고 limit이 없으면 limit hint를 보냅니다. 명시적 limit 또는 이 hint가 있을 때만
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
반환하고 `secret_store_id` getter가 마지막 component를 제공합니다. 단건 Python getter는 요청 selector를 id로 seed하므로 응답에 literal id가 없으면 `Resource.id`가 `global-default` 또는 `preferred`로 남습니다. Go는 selector를 응답 Body에 추가하지 않고 실제 literal id 또는 ref에서 추출한 편의용 ID를 반환합니다. Python `secret_store_id`는 별도의 ref formatter 속성입니다. Go의 편의용
ID와 원문 marker 분리는 이 차이를 명시합니다. Go는 typed 문자열·boolean을
검사하고, literal id가 없어 편의용 ID를 추출할 때만 absolute reference 문법을
검사합니다. Literal id가 있으면 ref URL을 파싱하지 않습니다. Malformed 응답은
오류로 반환합니다.
Python descriptor의 값 변환·일부 URI 허용 범위와 동일하다고 주장하지 않습니다.

`WithListQuery`의 wire extension과 `WithListFilter(s)`의 declared query/Body 분류는 서로 다른 옵션입니다. deprecated JMESPath·동적 conflicting-attribute 복구는 지원하지 않습니다. Inherited Resource의 cache/dirty state, model overload, generic per-call base path/version/header controls는 별도 SDK 범위입니다. 반환 모델의 strict typed decode와 canonical-key 정책도 위에 설명한 Go 매핑을 따릅니다.

검증 근거: [HTTP 계약 테스트](../../../api/keymanager_secretstores_test.go).
Source pin은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의
`openstack/key_manager/v1/_proxy.py:361–392`, `secret_store.py:17–58`,
`_format.py:18–30`과 `Resource.__getattribute__`, `Resource.list`,
`Resource._get_next_link`입니다. REST 상태·응답 형식은
[공식 Secret Stores API](https://docs.openstack.org/barbican/wallaby/api/reference/store_backends.html)를
참조했습니다. 테스트는 격리된 HTTP fixture이며 실제 cloud 테스트가 아닙니다.

## 권한

[Barbican2024.1 정책](https://docs.openstack.org/barbican/2024.1/configuration/policy.html)의 `secretstores:get`·`secretstores:get_global_default`·`secretstores:get_preferred`는 new defaults에서 project reader의 목록·단건 조회를 허용합니다. [Train 기본값](https://docs.openstack.org/barbican/train/configuration/policy.html)은 admin이며 운영 정책에 따라 달라질 수 있습니다. SDK는 역할을 추측하거나 전환하지 않고 실제403을 반환합니다.

## 단독 목록 예제

`-cloud dev -name backend-pattern -id store-id`는 name 서버 query와 secret_store_id 로컬 조건을 함께 사용합니다. 빈 flag는 그 조건을 생략합니다. caller timeout은 인증·발견·전체 순회를 포함하며 SDK가 timeout을 새로 시작하지 않습니다.

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "net/http"
    "os"
    "time"

    sdk "github.com/JSYoo5B/gophercloudsdk"
    "github.com/JSYoo5B/gophercloudsdk/keymanager/v1/secretstores"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml entry")
    name := flag.String("name", "", "server name query")
    id := flag.String("id", "", "local secret_store_id condition")
    timeout := flag.Duration("timeout", time.Minute, "authentication and list timeout")
    flag.Parse()
    if err := run(*cloud, *name, *id, *timeout); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(cloud, name, id string, timeout time.Duration) error {
    if timeout <= 0 {
        return fmt.Errorf("timeout must be positive")
    }
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil {
        return err
    }
    service, err := conn.KeyManagerV1(ctx)
    if err != nil {
        return err
    }
    options := []secretstores.ListOption{
        secretstores.WithListOptions(secretstores.ListOpts{Limit: 10}),
    }
    if name != "" {
        options = append(options, secretstores.WithListFilter("name", name))
    }
    if id != "" {
        options = append(options, secretstores.WithListFilter("secret_store_id", id))
    }
    stores, err := service.SecretStores.All(ctx, options...)
    if err != nil {
        return err
    }
    type item struct {
        ID         string                     `json:"id"`
        StatusCode int                        `json:"status_code"`
        Header     http.Header                `json:"header"`
        Body       map[string]json.RawMessage `json:"body"`
    }
    output := make([]item, 0, len(stores))
    for _, store := range stores {
        output = append(output, item{
            ID: store.ID,
            StatusCode: store.StatusCode,
            Header: store.Header,
            Body: store.Body,
        })
    }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    return encoder.Encode(output)
}
```

출력의 `id`는 Go 모델의 편의용 ID입니다. `body`는 해당 목록 row 원문이며 header/status_code는 실제 GET 응답입니다. SDK 모듈 안 별도 디렉토리의 main.go로 저장합니다. 실제 OpenStack·Python 실행은 별도 환경에서 수행합니다.
