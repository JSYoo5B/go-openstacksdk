# Compute console auth-token 조회

`conn.Compute(ctx)`의 `ValidateConsoleAuthToken(ctx, token)`은 openstacksdk의 `conn.compute.validate_console_auth_token(token)`에 대응합니다. console URL에서 얻은 token을 명시적으로 전달하면 Nova의 연결 정보를 조회합니다. provider 인증 token과 별개의 값이며 SDK가 서버·console을 생성하거나 URL에서 token을 추출하지 않습니다.

권한 분류는 **핵심 admin API**입니다. 2026-10-08에 확인한 [Nova 기본 정책](https://docs.openstack.org/nova/latest/configuration/policy.html)의 `os_compute_api:os-console-auth-tokens`는 `rule:context_is_admin`을 요구합니다. 배포의 policy override는 서버가 판단하며, SDK가 로컬 role 검사로 권한을 대신 결정하지 않습니다. 일반 사용자 console 생성과 토큰 연결 정보 조회의 권한은 다릅니다.

| openstacksdk | go-openstacksdk |
|---|---|
| `conn.compute.validate_console_auth_token(token)` | `service.ValidateConsoleAuthToken(ctx, token)` |
| ConsoleAuthToken Resource의 known 속성 | `record.Resource.Body`의 passive JSON 필드 |
| 실제 응답 객체 | `record.Wire` |
| HTTP status/header와 원문 | `record.StatusCode`, `Header`, `Envelope` |

고정 Python 구현은 `_get(ConsoleAuthToken, token)`을 호출하고 Resource를 반환합니다. 함수 이름의 validate는 로컬 유효성 bool 검사를 뜻하지 않습니다. 실제 HTTP403/404는 오류이며 missing-ignore나 다른 console API fallback을 적용하지 않습니다.

```python
import sys
import openstack

conn = openstack.connect()
console = conn.compute.validate_console_auth_token(sys.argv[1])
print(console.to_dict())
```

## 독립 Go main

인증 환경을 준비하고 `-console-token TOKEN`을 전달합니다. 예제는 연결 정보 GET만 수행하고 반환한 host/port에 접속하지 않습니다. 문서 검증은 이 main의 빌드이며 실제 cloud 호출 실행은 별도입니다.

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
)

func main() {
    token := flag.String("console-token", "", "console token to look up")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *token); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, token string) error {
    if token == "" {
        return fmt.Errorf("-console-token is required")
    }
    conn, err := sdk.Connect(ctx)
    if err != nil {
        return err
    }
    service, err := conn.Compute(ctx)
    if err != nil {
        return err
    }
    record, err := service.ValidateConsoleAuthToken(ctx, token)
    if record != nil {
        data, marshalErr := json.MarshalIndent(map[string]any{
            "resource": record.Resource,
            "wire": record.Wire,
            "status_code": record.StatusCode,
            "header": record.Header,
        }, "", "  ")
        if marshalErr != nil {
            return marshalErr
        }
        fmt.Println(string(data))
    }
    return err
}
```

## 필드·seed·응답

Resource는 `instance_uuid`, `host`, `port`, `tls_port`, `internal_access_path`, `id`, `name`과 computed `location`의8개 필드를 갖습니다. 다섯 리소스 필드와 inherited id/name에는 Source가 type converter를 지정하지 않았습니다. 따라서 Go도 숫자·문자열·null·배열·객체를 JSON 그대로 보존하며 port를 강제로 int로 바꾸지 않습니다.

요청 token은 최초 id seed입니다. 실제 응답 id가 있으면 null·큰 수·다른 형태도 seed를 덮고, 생략하면 요청 token을 유지합니다. 다른 missing known 필드는 null입니다. 응답의 unknown 필드는 Resource에 넣지 않고 Wire에 보존합니다. `console` key가 있으면 그 객체를, 없으면 flat root 객체를 Wire로 선택합니다. Wire에는 token seed·missing 기본값·computed location을 합성하지 않습니다. Resource·Wire·Envelope·Header는 독립 snapshot입니다.

Connection은 설정된 cloud/project/region의 location을 제공합니다. 이 값은 discovery 이전에 snapshot하고 zone은 null입니다. 응답의 location은 Wire에만 남으며 computed 값을 덮지 않습니다. `compute.New(client, compute.Dependencies{})`로 직접 만들면 Resource location은 null입니다.

Source의 Resource.fetch는 accepted nonJSON·빈 응답의 JSON parsing 오류를 허용합니다. 같은 경우 Go는 seeded Resource와 nil Wire를 반환하고 실제 원문/header/status를 유지합니다. 반면 유효 JSON의 root 또는 console key가 null/scalar/array이면 processing 오류입니다. 수락한 응답의 read/Close/source/context 오류도 실제 receipt와 원래 cause를 보존하며 재조회하지 않습니다. 이런 실패에서는 record가 있어도 Resource가 nil일 수 있으므로 항상 error를 확인합니다.

## 버전·경로·Go 차이

selected microversion이 있으면 discovery 없이 그대로 조회합니다. resource ceiling2.99보다 큰2.100과 raw client의 `latest`·`2.latest`도 자동 낮추거나 로컬 거부하지 않습니다. 미선택일 때만 [공통 Nova discovery](console-selection.md)를 실행하고 max를2.99 한도로 선택해 private GET client에 적용합니다. 광고 max가 없거나 광고 min이2.99보다 높으면 version header 없이 token GET을 계속합니다. 2.6 minimum이나 console type gate는 없습니다. 원래 공유 service client를 수정하지 않습니다.

member GET은 선택된 ResourceBase의 `/os-console-auth-tokens/{token}`에 고정하며 body·query·server lookup을 추가하지 않습니다. 명시 token은 한 path segment로 검증·escape합니다. Go의 UTF-8/control/path 검사와 Python generic Resource의 더 넓은 식별자 입력은 구별합니다. 조회 token·반환 id/host/port는 provider 인증이나 다른 요청의 route를 변경하지 않습니다.

Go는 유한 catalog 경로·첫 exact API 행·status/self-link·captured authenticated token·operation-owned discovery를 사용합니다. Python의 전체 sorting/URL selection·anonymous401 전환·session discovery cache와 동일하다고 주장하지 않습니다. Go의 clean404/405·valid unmatched root fallback 외 discovery JSON/physical/source/context 오류는 terminal이고, Python maximum_supported_microversion은 일부 DiscoveryFailure 후 version 없이 계속할 수 있습니다. Go는 invalid UTF-8를 unavailable JSON parsing으로 처리하며 Python response charset/replacement decoding은 재현하지 않습니다. 반환은 mutable Python Resource/session 대신 독립 raw record입니다.

## 테스트 재사용

공통 `microversions.MemberGet`은 기존 Cinder bodyless GET·private version header·retry/physical guard를 공유합니다. 응답 parsing·receipt·raw field/clone는 기존 REST/Resource 엔진을 사용합니다. 조회 고유3그룹은 public Gophercloud testhelper와 기존 testcloud·flavorIdentityClient·payloadContractTrack·secretFetchRoundTripFunc를 사용합니다. 전체 discovery 또는 transport fault matrix를 서비스마다 복제하지 않습니다. 실행 결과와 API 판정은 [검증 기록](../docs/sdk-support-ledger.md), [구현 계획](../docs/implementation-plan.md)에 남깁니다.
