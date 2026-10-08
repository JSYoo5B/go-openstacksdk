# Trove root access

`service.Instances.IsRootEnabled(ctx, instanceID)`는 해당 Trove instance의 루트
계정 활성화 여부를 조회하고 `bool, error`를 반환합니다. instance ID를 직접
받으며 서비스의 현재 base URL에서 `GET /instances/{instanceID}/root`를
요청합니다.

```go
package example

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk"
)

func rootEnabled(ctx context.Context, conn *openstack.Connection, instanceID string) (bool, error) {
	service, err := conn.DatabaseV1(ctx)
	if err != nil {
		return false, err
	}
	return service.Instances.IsRootEnabled(ctx, instanceID)
}
```

먼저 `error`를 확인한 뒤 반환한 boolean을 사용합니다. 성공한 JSON 객체의
`rootEnabled`가 literal `true`일 때만 `true, nil`입니다. `false`, 누락, null,
문자열·숫자·배열·객체 필드 값은 기존 Gophercloud 비교와 같이 `false, nil`을
반환합니다. 문자열 `"true"`나 숫자 `1`을 boolean으로 변환하지 않습니다.

HTTP 200만 허용합니다. HTTP 상태 오류, transport·context 오류, 빈 응답이나
잘못된 JSON은 `false`와 원래 오류를 반환합니다. 성공한 JSON의 최상위 값이
null·배열·문자열·숫자·boolean이면 응답 형태 오류를 반환합니다. 이 오류에는
native 결과가 보존하지 않은 성공 상태 코드나 원문 응답을 만들어 넣지
않습니다. 모든 오류는 `IsRootEnabled`/`instances`의 `resource.OperationError`로
감싸며 `errors.Is`와 `errors.As`로 원래 cause를 검사할 수 있습니다.

이 조회는 instance 또는 사용자 목록을 추가로 읽지 않고 루트 계정을
활성화하지 않습니다. 활성화는 별도의 `Instances.EnableRootUser` 연산입니다.
현재 native client의 인증·서비스 헤더·transport·재인증·retry·redirect 정책을
사용합니다. 그 정책에 따른 재요청은 가능하지만 결과 추출 오류에 대한 SDK
재시도나 다른 조회로의 fallback은 추가하지 않습니다. 안전한 boolean 처리는
이 facade 메서드에 적용됩니다. 직접 노출된 native `IsRootEnabledResult`의
`Extract`는 고정 Gophercloud의 계약을 따릅니다.

## OpenStackSDK comparison

OpenStackSDK `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`에는 실제
[`Instance.is_root_enabled(session)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/database/v1/instance.py#L73-L85)
Resource 메서드가 있습니다. Database Proxy에 같은 작업을 위한 직접 메서드는
없으며, instance에 세션으로 `conn.database`를 전달합니다.

```python
from openstack.database.v1.instance import Instance

instance = Instance(id=instance_id)
enabled = instance.is_root_enabled(conn.database)
```

Python 메서드도 루트 endpoint를 GET하지만 `resp.json()['rootEnabled']`를
`typing.cast(bool, ...)`로 반환합니다. `cast`는 실행 중 boolean 변환이나 타입
검사를 하지 않습니다. 필드가 누락되면 Python dictionary 조회 오류가 나고,
null이나 비boolean 값은 그대로 돌아갈 수 있습니다. Go facade는 위에 명시한
Gophercloud의 literal-true 비교를 유지하면서 실패 경로의 panic을 오류 반환으로
바꿉니다. 이 비교는 해당 루트 권한 조회의 입력과 결과에 한정합니다.

[HTTP 계약 테스트](../../../api/trove_root_enabled_contracts_test.go)와
[Database 서비스 가이드](../README.md)에서 호출과 오류 처리를 확인할 수 있습니다.
