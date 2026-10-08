# Compute의 flavor·metadata·키페어·콘솔 조회

`conn.ComputeV2(ctx)`로 얻은 서비스에서 명시 ID 또는 키페어 이름을 조회합니다. 서버 metadata의 두 Python 이름은 같은 Go 메서드로 대응합니다. 콘솔 출력은 조회 작업이지만 Nova action의 POST를 사용합니다. 이 API들은 이름 목록 검색이나 서버 상태 대기를 자동으로 시작하지 않습니다.

## Python과 Go의 대응

| 고정 openstacksdk | go-openstacksdk |
|---|---|
| `conn.compute.get_flavor(flavor_id)` | `service.Flavors.FindIdentity(ctx, flavorID, resource.WithIdentityFindFallback(resource.FindFallbackNever), resource.WithIdentityFindIgnoreMissing(false))` |
| `get_flavor(flavor_id, get_extra_specs=True)` | 위 호출에 `resource.WithIdentityFindExtraSpecs(true)` 추가 |
| `conn.compute.fetch_server_metadata(server_id)` | `service.Servers.Metadata(ctx, serverID)` |
| deprecated `conn.compute.get_server_metadata(server_id)` | 같은 `service.Servers.Metadata(ctx, serverID)` |
| `conn.compute.get_keypair(name)` | `service.KeyPairs.Get(ctx, name)` |
| `get_keypair(name, user_id=owner_id)` | `Get(ctx, name, keypairs.WithGetOptions(keypairs.GetOpts{UserID: ownerID}))` |
| `conn.compute.get_server_console_output(server_id)` | `service.Servers.ConsoleOutput(ctx, serverID)` |
| `get_server_console_output(server_id, length=0)` | `ConsoleOutput(ctx, serverID, servers.WithConsoleOutputLength(0))` |

Python:

```python
import openstack

conn = openstack.connect()
flavor = conn.compute.get_flavor("flavor-id", get_extra_specs=True)
server = conn.compute.fetch_server_metadata("server-id")
keypair = conn.compute.get_keypair("login-key")
console = conn.compute.get_server_console_output("server-id", length=50)
print(flavor.extra_specs, server.metadata, keypair.fingerprint, console)
```

## 독립 실행 Go 예제

`OS_CLOUD`/`clouds.yaml` 또는 `OS_*` 인증 설정을 준비하고 `SERVER_ID FLAVOR_ID KEYPAIR_NAME [OWNER_USER_ID]`를 전달합니다. owner를 생략하면 자신의 키페어를 조회합니다. 예제의 Compute microversion 2.10은 owner query를 사용할 수 있도록 명시한 버전이며, 사용할 cloud에 맞게 선택합니다. 인증과 네 조회는 같은 30초 context를 사용합니다.

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    if err := run(ctx); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context) error {
    if len(os.Args) != 4 && len(os.Args) != 5 {
        return fmt.Errorf("usage: %s SERVER_ID FLAVOR_ID KEYPAIR_NAME [OWNER_USER_ID]", os.Args[0])
    }
    conn, err := sdk.Connect(ctx, sdk.WithMicroversion(sdk.Compute, "2.10"))
    if err != nil {
        return err
    }
    service, err := conn.ComputeV2(ctx)
    if err != nil {
        return err
    }
    flavor, err := service.Flavors.FindIdentity(ctx, os.Args[2],
        resource.WithIdentityFindFallback(resource.FindFallbackNever),
        resource.WithIdentityFindIgnoreMissing(false),
        resource.WithIdentityFindExtraSpecs(true),
    )
    if err != nil {
        return fmt.Errorf("get flavor: %w", err)
    }
    metadata, err := service.Servers.Metadata(ctx, os.Args[1])
    if err != nil {
        return fmt.Errorf("get server metadata: %w", err)
    }
    var keyOptions []keypairs.GetOption
    if len(os.Args) == 5 {
        keyOptions = append(keyOptions,
            keypairs.WithGetOptions(keypairs.GetOpts{UserID: os.Args[4]}))
    }
    pair, err := service.KeyPairs.Get(ctx, os.Args[3], keyOptions...)
    if err != nil {
        return fmt.Errorf("get keypair: %w", err)
    }
    output, err := service.Servers.ConsoleOutput(ctx, os.Args[1],
        servers.WithConsoleOutputLength(50))
    if err != nil {
        return fmt.Errorf("get console output: %w", err)
    }
    data, err := json.MarshalIndent(map[string]any{
        "flavor": flavor,
        "metadata": metadata,
        "keypair": pair,
        "console_output": output,
    }, "", "  ")
    if err != nil {
        return err
    }
    fmt.Println(string(data))
    return nil
}
```

## Flavor의 GET 전용 정책

Python `get_flavor`는 명시 ID의 GET이며 이름 목록 fallback이 없습니다. Go의 `FindFallbackNever`와 `WithIdentityFindIgnoreMissing(false)`를 함께 지정하면 400/403/404를 목록 조회나 정상 미존재로 바꾸지 않습니다. 안전하지 않은 ID 경로도 HTTP 전에 거부합니다. 일반 `FindIdentity`의 기본 fallback 정책과 구분하세요.

ExtraSpecs 기본값은 false입니다. true일 때 반환된 native Flavor의 map이 nil 또는 비어 있으면 반환된 canonical ID의 `/os-extra_specs`를 한 번 조회합니다. inline map에 값이 있으면 추가 GET은 없고, 후속 실패는 strict 오류로 반환합니다. 이름 목록·상세 member 조회·caller query 전달을 추가하지 않습니다. Go는 선택된 microversion을 유지하며 inline specs가 도입된 2.61을 자동 선택하지 않습니다.

반환형은 native `*flavors.Flavor`입니다. `extra_specs`의 생략/null은 nil map이며 Python의 생략 기본 `{}`와 다릅니다. `IsPublic`의 생략은 Go false이고 source descriptor의 기본값 true를 합성하지 않습니다. 안전한 반환 ID를 요구하며 Python Resource의 `id`·`name`·원래 입력 fallback, nullable descriptor·unknown 속성·location·mutable 연결 상태를 복제하지 않습니다. [identity 조회 가이드](../docs/finding-identities.md#nova-flavor와-extra-specs)에 일반 Find와 GET 전용 정책을 함께 설명합니다. `find_flavor`의 상속 pager·query 계약은 별도 지원 판정입니다.

## 서버 metadata의 두 Python 이름

고정 `fetch_server_metadata`는 `/servers/{id}/metadata`를 GET하여 metadata를 넣은 Server Resource를 반환합니다. `get_server_metadata`는 같은 함수로 위임하는 deprecated alias입니다. Go의 `Metadata`는 서버 detail GET 없이 같은 경로에서 `map[string]string`만 반환하고 supplied Server나 cache를 수정하지 않습니다. Python alias 경고도 추가하지 않습니다.

native HTTP200과 문자열 map decoding을 사용합니다. metadata key 생략 또는 null은 nil map, `{}`는 nonnil 빈 map입니다. 문자열 값의 Unicode를 보존하고 native JSON decoding에 따라 개별 null 값은 빈 문자열이 됩니다. 숫자·bool·배열·객체 같은 문자열이 아닌 값은 decode 오류입니다. field-type 오류에는 일부 decode된 map이 함께 반환될 수 있으므로 map의 존재를 성공 증거로 사용하지 말고 error를 먼저 확인하세요. HTTP 실패나 malformed JSON은 native 오류를 반환하며, Python의 동적 dict·seeded Resource·adapter-wide response 정책은 제공하지 않습니다.

## 키페어 이름과 optional owner

키페어의 API 식별자는 numeric wire `id`가 아니라 이름입니다. `KeyPairs.Get(ctx, name)`은 `/os-keypairs/{name}`를 한 번 조회하고 목록 fallback을 하지 않습니다. `GetOpts.UserID`의 빈 문자열은 query를 생략하고 비어 있지 않은 값은 `user_id`로 전달합니다. 선택한 Compute microversion을 그대로 사용하며 Python Keypair의 최대 버전 2.10 기반 discovery를 재현하거나 owner 옵션 때문에 client 버전을 바꾸지 않습니다.

반환형은 native `*keypairs.KeyPair`이며 name·fingerprint·public/private key·user ID·type 필드를 제공합니다. 생략/null 문자열은 빈 문자열이고, keypair envelope가 생략/null이면 native `(nil, nil)` 결과입니다. source의 type 기본값 `ssh`, 기존 Resource의 seed name, created_at·deleted·unknown 속성 등을 합성하지 않습니다. native HTTP200을 사용하고 403/404나 typed decoding 오류를 빈 결과로 숨기지 않습니다. field-type 오류에는 일부 decode된 모델이 함께 올 수 있습니다.

자기 user의 기본 조회와 명시한 타 owner 조회는 구분합니다. 실제 권한과 owner query 지원은 cloud 정책·API 버전이 검사하고 서버의 오류를 그대로 반환합니다. 이 가이드의 로컬 HTTP 검증은 실제 배포의 권한 승인을 뜻하지 않습니다.

## 콘솔 출력의 생략과 0

`ConsoleOutput`은 `/servers/{id}/action`에 `{"os-getConsoleOutput":{}}`를 POST합니다. 옵션을 생략하거나 `ConsoleOutputOpts.Length`가 nil이면 length key를 보내지 않습니다. `WithConsoleOutputLength(0)`은 0을, 양수·음수도 명시한 정수를 그대로 전송합니다. line 제한의 실제 해석은 Nova가 처리합니다.

`WithConsoleOutputOptions(ConsoleOutputOpts{Length: pointer})`는 typed 옵션을 교체하고 `WithConsoleOutputField`는 action 내부의 추가 JSON 필드를 제공합니다. core `length`는 typed 옵션으로 지정하며 extension으로 덮지 않습니다. 입력 pointer·JSON 필드는 소유한 snapshot으로 사용하고 마지막 typed length 값이 우선합니다. 호출별 query·header·argument 옵션은 제공하지 않습니다. 선택된 source·microversion·provider·추가 client header는 유지하며 action의 기본 Accept 값은 Python처럼 빈 문자열입니다. 입력 snapshot과 preflight 오류를 SDK가 처리하므로 caller builder가 필요하지 않습니다.

owned API는 source action처럼 200–399를 허용하되 nonnull UTF-8 JSON object와 문자열 `output`을 검사합니다. root null·배열·빈 body·JSON이 아닌 응답은 오류입니다. `output` 생략/null/빈 문자열은 Go 빈 문자열이고 문자열이 아닌 값은 오류입니다. 필드 선택은 native Go JSON decoder의 대소문자 매칭을 유지합니다. Python은 dict 전체를 반환하며 Go는 출력 문자열만 제공합니다. control 문자는 JSON decoding 후 문자열에 포함되고 `json.MarshalIndent`로 출력하면 다시 JSON으로 escape됩니다. accepted read·Close·decode 오류와 실제 응답 증거는 반환 error에서 확인하며 SDK composition은 그 오류만으로 action을 재전송하지 않습니다. provider의 native retry·재인증은 별도 정책입니다.

기존 `ShowConsoleOutput(ctx, id, ShowConsoleOutputOpts, options...)`는 native signature·HTTP200·int `omitempty`를 그대로 유지합니다. 그 API의 `Length:0`은 생략이므로 source의 explicit0이 필요한 호출은 새 `ConsoleOutput`을 사용합니다.

## 소스와 판정 범위

비교 기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [get_flavor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L246-L266), [metadata 두 이름](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L1876-L1906), [MetadataMixin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/metadata.py#L33-L46), [get_keypair](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L868-L890)와 [콘솔 출력](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server.py#L938-L952)입니다.

이 named 조회들은 명시 입력과 documented typed/raw 응답 정책으로 비교합니다. 전체 mutable Resource/session/descriptor/cache 및 다른 server·keypair·flavor 연산은 전체 SDK 목표에서 별도로 추적합니다. 실제 로컬 HTTP·독립 main 컴파일·전체 gate와 각 함수의 판정은 [지원 판정대장](../docs/sdk-support-ledger.md)에 기록합니다. 이 문서만으로 실행 성공, 인증된 OpenStack 검증이나 `find_flavor`의 pager 완료를 주장하지 않습니다.
