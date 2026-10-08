# Barbican 생성: Python과 Go

세 생성 API는 같은 옵션과 결과 구조를 사용합니다. `CreateRecord`는 입력을 소유하고, 참조만 있는 POST 응답에도 입력 속성을 유지합니다.

| openstacksdk | Go | 요청 |
|---|---|---|
| `conn.key_manager.create_container(**attrs)` | `service.Containers.CreateRecord(ctx, options...)` | POST `containers`, flat JSON |
| `conn.key_manager.create_order(**attrs)` | `service.Orders.CreateRecord(ctx, options...)` | POST `orders`, flat JSON |
| `conn.key_manager.create_secret(**attrs)` | `service.Secrets.CreateRecord(ctx, options...)` | POST `secrets`, flat JSON |

각 leaf의 `WithCreateRecordAttributes(map[string]any)`로 여러 속성을 지정하거나 `WithCreateRecordAttribute(key, value)`로 하나씩 지정합니다. 라이브러리가 선언된 속성과 wire 이름을 연결하므로 builder interface를 구현하지 않아도 됩니다. 별도 type·name·payload 기본값을 넣지 않으며, 옵션이 없으면 `{}`를 보냅니다. 필수 값·enum·payload 조합은 서버가 판정합니다.

## 입력과 옵션

생략한 키는 보내지 않습니다. 지정한 `nil`, `""`, `0`, `false`, 빈 배열과 빈 객체는 그대로 보냅니다. Order `meta`는 중첩 JSON을 받을 수 있고, Secret `expires_at`은 `expiration`으로 보냅니다. 자동 timestamp 파싱이나 포맷 변경은 없습니다.

| 리소스 | 추가 선언 속성 / alias |
|---|---|
| 공통 | `id`, `name`, `status`, `created_at` → `created`, `updated_at` → `updated` |
| Container | `type`, `secret_refs`, `consumers`, `container_ref` / `container_id` |
| Order | `type`, `meta`, `creator_id`, `sub_status`, `sub_status_message`, `order_ref` / `order_id`, `secret_ref` / `secret_id` |
| Secret | `algorithm`, `bit_length`, `mode`, `secret_type`, `content_types`, `payload`, `payload_content_type`, `payload_content_encoding`, `secret_ref` / `secret_id`, `expires_at` / `expiration` |

한 map에 wire 키와 attribute alias를 함께 넣으면 wire 키가 우선합니다. Python은 dict의 입력 순서에 따라 마지막 alias 값이 남을 수 있으므로 이 충돌 정책은 Go의 명시적인 차이입니다. 개별 옵션은 적용 순서대로 같은 wire 속성을 교체합니다. Bulk 옵션과 `WithCreateRecordOptions(CreateRecordOpts{Attributes: ...})`는 선언 속성 묶음만 교체하며 이미 선택한 extension·header는 유지합니다. nil bulk는 속성 묶음을 비웁니다.

알 수 없는 일반 속성은 JSON 변환 전에 버립니다. 배포별 추가 JSON은 `WithCreateRecordField(key, value)`로 명시하며, 선언된 wire 키와 alias를 덮을 수 없습니다. 각 helper는 생성 시 선택된 JSON 값을 snapshot하고 호출마다 독립 복사합니다. 큰 정수는 JSON 숫자로 보존합니다. 알려진 값의 JSON 직렬화 오류는 HTTP 전에 반환하며, 재사용한 옵션이 이후 caller map 변경을 읽지 않습니다.

`WithCreateRecordHeader`는 추가 header를 지정합니다. 인증·라우팅·프로토콜 header와 선택한 microversion은 공통 source 정책으로 보호합니다. 기존 ServiceClient/Provider, 현재 token, ResourceBase와 source header를 사용합니다. 옵션은 한 번 적용하며 source·context를 전후와 실제 HTTP 경계에서 확인합니다. 생성 입력의 reference formatter 오류도 POST 전에 반환합니다. 생성 endpoint를 입력이나 응답 reference로 바꾸지 않습니다.

요청 body에는 raw 입력을 유지합니다. 예를 들어 `secret_refs: "one"`이나 `meta: 3`을 먼저 배열·객체로 바꾸어 보내지 않습니다. Python의 list/dict descriptor 변환은 반환 view에만 적용합니다. JSON으로 표현할 수 없는 Python 객체·set·tuple의 모든 변환을 Go에서 재현하지는 않습니다.

## 반환값과 오류

`CreatedContainer`, `CreatedOrder`, `CreatedSecret`은 다음 값을 구분합니다.

- `Resource`: 선언된 입력을 seed한 뒤 실제 응답의 선언 속성을 병합한 view. 응답 생략은 입력을 유지하고, 명시 null은 입력을 덮습니다. list 속성의 scalar는 한 요소 배열, dict 속성의 non-object는 빈 객체로 읽으며 null은 null입니다.
- `Wire`: 입력을 합성하지 않은 실제 응답 객체. unknown JSON도 보존합니다.
- `Envelope`, `Header`, `StatusCode`: 실제 POST 응답의 원문과 HTTP 증거. 각 결과는 독립적으로 소유합니다.

`Resource.Body["id"]`는 입력/응답 literal id가 있으면 null·empty도 그대로 사용하고, 없으면 전체 alternate reference를 사용합니다. `ContainerID`, `OrderID`, `SecretID` pointer는 별도의 마지막 component formatter 결과입니다. Reference는 수동 데이터입니다. 생성 결과에 executable RequestID나 자동 routing `Ref()`를 만들지 않습니다. `location`은 기존 Go owned view와 같이 null이며 Python Connection location 합성은 별도 공통 SDK 범위입니다.

생성 뒤 metadata GET·Secret payload GET·Order wait·실패 보상 DELETE는 자동 실행하지 않습니다. 필요하면 caller가 후속 작업을 명시합니다. 특히 Order202는 접수이며 작업 완료를 뜻하지 않습니다.

Owned 경로는 최종 HTTP200..399를 받고 non-null UTF-8 JSON root object를 엄격히 디코드합니다. Python generic create가 JSON ValueError를 무시하고 seed를 유지하는 것과 다릅니다. accepted 읽기·Close·JSON·formatter·source·context 실패에는 이미 받은 record와 `resource.ResponseError`의 실제 body/header/status를 반환합니다. HTTP403/404/409와 transport 원인을 보존하며 missing 무시나 fallback을 하지 않습니다. SDK는 이런 실패를 이유로 POST를 다시 조합하지 않습니다. 설정된 Provider의 native retry·재인증은 기존 transport 정책을 유지합니다.

기존 `Create(ctx, CreateOpts, ...CreateOption)`는 native 모델 alias·serializer·응답 코드 정책을 유지합니다. Container/Secret는201, Order는202를 요구하고 응답만 추출합니다. null/생략과 입력 seed가 필요하면 `CreateRecord`를 사용합니다. Mutable Python Resource의 dirty/cache/session lifecycle과 generic adapter override 전체는 별도 SDK 범위로 추적합니다.

## 전체 흐름 예제

환경 인증으로 Secret를 만들고, 반환 reference를 caller가 선택해 Container 생성 입력으로 사용합니다. Order는 별도 비동기 key 생성 요청입니다. 실제 클라우드 실행은 생성 권한과 서버 validation에 따릅니다. 이 예제의 main은 검증 시 컴파일하며 클라우드에서 실행하지 않습니다.

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
)

func main() {
	ctx := context.Background()
	conn, err := go-openstacksdk.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.KeyManagerV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	secret, err := service.Secrets.CreateRecord(ctx, secrets.WithCreateRecordAttributes(map[string]any{
		"name": "stored-demo", "secret_type": secrets.OpaqueSecret,
		"payload": "demo", "payload_content_type": "text/plain",
	}))
	if err != nil {
		log.Fatal(err)
	}
	var secretRef string
	if err := json.Unmarshal(secret.Resource.Body["secret_ref"], &secretRef); err != nil || secretRef == "" {
		log.Fatalf("created secret has no usable reference: %v", err)
	}
	container, err := service.Containers.CreateRecord(ctx,
		containers.WithCreateRecordAttribute("type", containers.GenericContainer),
		containers.WithCreateRecordAttribute("name", "container-demo"),
		containers.WithCreateRecordAttribute("secret_refs", []map[string]any{{"name": "stored", "secret_ref": secretRef}}),
	)
	if err != nil {
		log.Fatal(err)
	}
	order, err := service.Orders.CreateRecord(ctx, orders.WithCreateRecordAttributes(map[string]any{
		"type": orders.KeyOrder, "meta": map[string]any{"algorithm": "aes", "bit_length": 256, "mode": "cbc"},
	}))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("secret=%s container=%s order=%s\n", secret.Resource.Body["id"], container.Resource.Body["id"], order.Resource.Body["id"])
}
```

Source pin은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 `key_manager/v1/_proxy.py`, `container.py`, `order.py`, `secret.py`와 공통 `proxy.py`·`resource.py`·`fields.py`입니다. Gophercloud native 경로는 v2.15.0입니다. 검증과 실제 지원 집계는 [테스트](../../docs/testing.md), [구현 계획](../../docs/implementation-plan.md), [SDK 판정 근거](../../docs/sdk-support-ledger.md)에서 확인합니다.
