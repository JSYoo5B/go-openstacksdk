# Glance의 현재 project quota·usage 조회

`ServiceInfo.GetUsageInfo`는 인증된 current project의 `GET /info/usage`를 한 번 읽고 실제 200 응답을 반환합니다. project·store·image selector, query, request body, pagination, cache와 quota enforcement는 추가하지 않습니다. Go의 `context.Context`는 취소·시간 제한에 사용하며 조회할 project를 선택하지 않습니다.

```go
package example

import (
    "context"

    "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo"
)

func discoverUsage(ctx context.Context, conn *openstack.Connection) (*serviceinfo.UsageInfo, error) {
    service, err := conn.ImageV2(ctx)
    if err != nil {
        return nil, err
    }
    return service.ServiceInfo.GetUsageInfo(ctx)
}
```

상위 `image.Service.API.ServiceInfo`와 직접 생성한 `serviceinfo.New(client)`에서도 같은 singleton을 사용합니다. 현재 authenticated image client·prefix·provider를 공유하며 quota 조회를 위해 별도 Keystone client나 endpoint discovery를 요청하지 않습니다.

## Python SDK와 API의 경계

고정된 openstacksdk commit `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 Image proxy·service_info에는 usage 연산이 없고, Gophercloud `v2.15.0`의 Image v2에도 대응 native 연산이 없습니다. 따라서 이 기능에 대응하는 Python/native operation이나 parity review를 만들지 않습니다. 다음은 이미 알고 있는 authenticated image endpoint와 token으로 같은 HTTP 조회를 수행하는 Python 표준 라이브러리 예시입니다.

```python
import json
from urllib.request import Request, urlopen

def get_usage_http(image_endpoint, token):
    request = Request(
        image_endpoint.rstrip("/") + "/info/usage",
        headers={"X-Auth-Token": token, "Accept": "application/json"},
        method="GET",
    )
    with urlopen(request) as response:
        if response.status != 200:
            raise RuntimeError(f"unexpected usage status: {response.status}")
        return json.loads(response.read())
```

이 HTTP 예시는 SDK method 호출이나 Go의 source guard·raw error evidence·retry ownership 정책과 같은 계약을 뜻하지 않습니다.

[공식 quota usage API](https://docs.openstack.org/api-ref/image/v2/index.html#quota-usage)는 request parameter/body 없이 200으로 quota와 사용량을 제공합니다. [Yoga 24.0.0 release notes](https://docs.openstack.org/releasenotes/glance/yoga.html)는 이 기능의 release 도입을 설명합니다. release 도입과 최소 API minor version/microversion은 별개입니다. 확인한 Glance [router](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/router.py#L607-L610)는 v2 route를 등록하며 SDK는 usage-specific version header나 사전 discovery gate를 요구하지 않습니다.

## 단위와 optional 정수

`UsageInfo`는 `resource.Metadata`를 포함하며 `Usage`는 resource 이름에서 `*UsageResource`로 이어지는 map입니다. 알려진 resource의 단위는 다음과 같습니다.

| Resource 이름 | 단위·서버 계산 |
|---|---|
| `image_size_total` | 저장된 이미지 bytes를 MiB로 나눈 정수 사용량 |
| `image_stage_total` | staging bytes를 MiB로 나눈 정수 사용량 |
| `image_count_total` | 삭제되지 않은 현재 project 이미지 수 |
| `image_count_uploading` | uploading/importing 상태의 이미지 수 |

[고정된 quota helper](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/quota/keystone.py#L147-L170)는 bytes를 `2^20`으로 나눠 MiB 정수 사용량을 계산합니다. SDK가 bytes로 다시 바꾸거나 반올림하지 않습니다. 다른 resource 이름도 passive map data로 보존하며 이름에서 unit·route·지원 여부를 추정하지 않습니다.

`UsageResource.Limit`·`Usage`는 `*int64`입니다. canonical `limit`·`usage`가 생략/null이면 nil이고 명시적인 0은 nonnil zero입니다. signed int64 범위의 JSON integer를 정확하게 읽으므로 `9007199254740993` 같은 float64 정밀도 밖 값도 유지합니다. string·bool·fraction·overflow는 decode 오류이며 음수를 clamp하거나 unlimited/overquota로 해석하지 않습니다. `UsageResource.Body`는 생략/null·unknown metric·원문 숫자를 `json.RawMessage`로 남깁니다.

전체 body는 valid UTF-8의 flat object이고 canonical `usage`는 필수 nonnull object입니다. 각 resource 값도 nonnull object여야 합니다. `Usage`·`Limit` 같은 대소문자 decoy는 raw unknown 필드로 남고 typed 값을 바꾸지 않습니다. alternate quota envelope를 자동으로 풀지 않으며 unknown root 값은 `UsageInfo.Body`에 보존합니다. `Header`·`StatusCode`는 실제 HTTP 응답에서 복사합니다.

고정된 서버의 [usage schema](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/discovery.py#L235-L236)는 array를 선언하지만 이 schema의 validation은 응답에 적용되지 않으므로, DTO는 controller·serializer·실제 response sample의 object를 따릅니다.

## Disabled 결과와 요청·오류 정책

확인한 [Glance 설정](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/common/config.py#L479-L480)의 `use_keystone_limits` 기본값은 false입니다. 비활성화된 서버는 정상 200의 `{"usage": {}}`를 반환하며 Go는 nonnil empty map으로 받습니다. 빈 map을 404·unsupported·무제한 quota·모두 0으로 해석하지 않고, 응답이 비었다는 사실만으로 서버 설정을 단정하지 않습니다. 다른 project의 limit 조회나 quota 변경, upload/import 자동 차단도 수행하지 않습니다.

옵션 없는 호출은 query/body 없이 한 번의 singleton GET을 준비합니다. `GetUsageInfoOpts`는 empty struct이며 `WithGetUsageInfoHeader`는 일반 header만 추가합니다. nil/error callback, query·body field·argument·microversion 확장과 보호된 auth·framing·version header는 HTTP 전에 거부합니다. 각 callback은 소유한 config를 한 번 전달받고 보관한 map/slice가 다음 callback이나 최종 요청을 바꾸지 않습니다.

context·image source·provider·base·headers를 옵션 전에 검사하고 callback 뒤와 HTTP 전에 다시 확인합니다. capture한 prefix·headers·원래 provider를 사용하면서 해당 provider의 live auth token을 유지합니다. configured native pre-body reauth·retry·backoff와 같은 scope의 redirect callback은 유지하며 method·origin·path·query 변경은 transport 전에 차단합니다.

실제 200만 허용하며 404나 disabled 응답을 다른 discovery/static quota로 fallback하지 않습니다. accepted body read·Close·context·UTF-8·JSON·schema·numeric 실패는 전체 raw Body·Header·StatusCode를 가진 `resource.ResponseError`와 원래 원인을 보존합니다. body는 한 번 닫고 accepted read/Close 실패 뒤 replay하지 않습니다. 공통 `DoJSON`은 retry callback의 body 소유권 변경을 거부하고 원래 expected 200을 다시 검사합니다. native가 거부한 status의 body 처리와 reauth wrapper는 기존 native 정책을 따르며 `ErrUnableToReauthenticate.ErrOriginal`·`ErrReauth`는 `errors.As`로 얻은 wrapper의 필드에서 확인합니다.

서버 비교는 Glance commit `57f7dd9e76ef24e1e9013eceaa703bd442469a24`의 [usage controller·serializer](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/discovery.py#L197-L213), quota helper와 실제 response example을 기준으로 합니다. 이 조회의 local HTTP 테스트는 실제 배포의 authentication·권한·limit 설정·quota enforcement 결과를 대신하지 않습니다.

실제 route·disabled empty 결과·optional 정수·schema 오류·옵션 소유권은 [core 테스트](usage_core_test.go), 외부 호출의 accepted evidence·strict status·provider/retry 경계는 [계약 테스트](usage_contracts_test.go)에서 확인합니다. [Connection 테스트](../../../connection_image_usage_test.go)는 공유 client·prefix·live token 경로를, [generator 테스트](../../../internal/cmd/sdkgen/glance_serviceinfo_test.go)는 기존 native API를 유지하면서 UsageInfo를 SDK 소유 모델로 등록하는 경계를 확인합니다.
