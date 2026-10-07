# SDK 지원 판정 검증

`paritycheck`는 생성된 API 함수의 존재와 SDK 지원 판정을 구분합니다. SDK 루트에서 실행합니다.

```sh
go run ./internal/cmd/paritycheck
go run ./internal/cmd/paritycheck -sync
```

기본 명령은 두 고정 inventory, 전체 catalog와 수작업 판정이 일치하는지 확인합니다. `-sync`는 새 연산을 catalog에 추가하며 수작업 판정 파일을 변경하지 않습니다. 판정이 없는 연산은 `unresolved`입니다. 검토한 원본이 변경되거나 기존 연산이 사라졌으면 저장 전에 실패하고 기존 catalog를 유지합니다. 미검토 연산도 조용히 삭제하지 않습니다. 저장은 임시 파일을 작성한 뒤 rename하여 write 실패에도 기존 근거를 보존합니다. `-root /path/to/gophercloudsdk`로 다른 위치의 checkout을 검사할 수도 있습니다.

| 파일 | 역할 |
|---|---|
| [gophercloud_inventory.json](../../../api/gophercloud_inventory.json) | native 호출 ID, source와 transport 정보 |
| [openstacksdk/manifest.json](../../../api/openstacksdk/manifest.json) | 고정 Python 직접 선언 목록과 개수 |
| [sdk_support_catalog.json](../../../api/sdk_support_catalog.json) | 모든 직접 선언 연산의 ID와 source fingerprint |
| [sdk_reviews.json](../../../api/sdk_reviews.json) | 재생성과 분리한 판정·세부 계약·근거·남은 기능 |

ID는 `gophercloud:compute/v2/servers.GetPassword` 또는 `python:compute/v2/get_server_password` 형태입니다. catalog의 fingerprint를 판정의 `source_fingerprint`로 복사한 뒤 소스 계약을 검토합니다. 서로 다른 버전의 근거를 재사용하지 않도록 source pin과 fingerprint를 모두 검증합니다. fingerprint에는 해당 source pin과 inventory의 선언 metadata를 포함하며 후보 Go 패키지·review marker·SDK 반환 정책은 제외합니다. 원본 함수 본문은 source pin으로 고정되고, 이 명령이 소스 checkout의 함수 본문을 직접 분석하지는 않습니다. source fingerprint 오류에는 현재 값도 출력되지만 원본을 검토하지 않고 그 값만 복사하여 지원 판정을 유지하지 않습니다.

판정에는 다음 정보를 기록합니다.

```json
{
  "operation": "python:service/v1/get_resource",
  "source_fingerprint": "catalog에 기록된 SHA-256",
  "status": "go_mapping",
  "go_api": ["github.com/JSYoo5B/gophercloudsdk/service/v1/resources.API.Get"],
  "contracts": [{
    "behavior": "ID 조회, 응답 값과 HTTP 오류 보존",
    "tests": ["api/resource_contracts_test.go:TestResourceGet"]
  }],
  "differences": ["context와 error를 명시적으로 반환"],
  "documentation": ["service/v1/resources/README.md"]
}
```

이 구조 예제의 연산·파일은 실제 항목이 아닙니다. 실제 형식과 검증 근거는 [판정 JSON](../../../api/sdk_reviews.json)을 확인합니다. Go API는 module import path와 함수 이름 또는 명시적 receiver method 이름입니다. generic receiver도 검사하지만 승격된 method·alias·interface 구현을 자동으로 추론하지는 않습니다. 그런 경우 실제 구현된 공개 method를 근거로 연결합니다. 내부 함수는 공개 SDK API 근거로 허용하지 않습니다.

`supported`와 `go_mapping`은 API·하나 이상의 계약·계약별 테스트·사용 문서가 필요하고 남은 기능을 기록할 수 없습니다. `go_mapping`은 기본값·타입·결과 등 관찰 가능한 차이를 설명해야 합니다. `unsupported`와 `unresolved`는 구체적인 `remaining`을 기록합니다. 일부 계약만 검증했으면 전체 연산은 `unresolved`로 유지하면서 `contracts`에 완료한 세부 근거를 연결합니다.

검증기는 판정·catalog의 모르는 필드, 중복 ID·JSON key, 누락된 catalog 연산, source drift, 사라진 API·테스트·문서를 거부합니다. 어느 위치의 `internal` 패키지도 공개 SDK API 근거로 허용하지 않습니다. Go AST에서 테스트 이름과 `*testing.T` signature를 확인하며, 테스트의 의미나 실행 성공을 자동 판정하지 않습니다. 근거를 추가할 때 계약 테스트를 실행하고 Python/native 구현과 실제 assertion을 검토합니다.

현재 조사 범위는 직접 선언 3,362개입니다. 상속·descriptor·Resource·외부 Adapter와 사용자가 등록한 서비스는 [지원대장](../../../docs/sdk-support-ledger.md)에 별도 조사 대상으로 기록합니다. 출력의 상태 개수는 이 직접 선언 목록의 검토 상태이며 전체 SDK 완료율로 사용하지 않습니다.
