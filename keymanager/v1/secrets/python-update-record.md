# Secret JSON commit: Python update_secret와 Go UpdateRecord

`secrets.API.UpdateRecord`는 pinned openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 `key_manager.update_secret` 중 **문자열 ID + JSON 속성** 경로를 제공합니다. Gophercloud `v2.15.0`의 raw payload 업로드 `API.Update`와 별도 API입니다. 옵션과 seed/본문/응답 복사는 SDK가 소유하므로 호출자 interface builder가 필요하지 않습니다.

이 named 행은 기존 Resource/dictionary seed의 dirty 상태·재커밋·생성자 및 Requests JSON loader 전체 잔여 때문에 **unresolved**, 완료 집계 **0**입니다. 일반 Barbican 서버가 JSON metadata 변경을 지원한다는 주장은 하지 않습니다. payload 바이트를 실제 Barbican에 추가하려는 목적이면 기존 `API.Update(ctx,id,UpdateOpts{ContentType:...,Payload:...})`를 사용합니다.

| 목적 | Python | Go |
|---|---|---|
| 기본 no-op | `update_secret("s1")` | `UpdateRecord(ctx,"s1")` |
| JSON 속성 commit | `update_secret("s1",name="new")` | `WithUpdateRecordAttribute("name","new")` |
| 명시적 null | `name=None` | `WithUpdateRecordAttribute("name",nil)` |
| wire/attribute alias | `expires_at=...` 또는 `expiration=...` | 같은 attribute factory key |
| 입력과 응답 | mutable Secret에 응답 필드 병합 | `Seed`/`Request`와 `Resource`/`Wire`/실제 receipt 분리 |
| dirty 없음 | PUT 없이 Resource 반환 | `Committed=false`, `StatusCode=0`, `Wire=nil`, seed Resource 반환 |
| 실제 JSON PUT | ID dirty 제거 후 속성만 전송 | `PUT /secrets/{id}`의 unwrapped object에 **id 없음** |

ID는 route/초기 seed이며 commit 본문에 무조건 추가하지 않습니다. 알려진 Body 속성만 dirty body가 됩니다. unknown, `location`, `microversion`, `if_match` 등만 주면 Source와 같이 PUT이 없습니다. Secret의 inherited Header mapping은 실제로 비어 있습니다. nullable 값은 누락과 다릅니다. 새 문자열 seed에서 `name=null`은 dirty이고 PUT에 들어갑니다.

다음 두 main은 `clouds.yaml`의 `devstack`을 사용합니다. 기본 `--attrs-json '{}'`는 no-op입니다. Python은 Resource를, Go는 최종 Resource와 PUT 관찰 정보를 출력합니다. 예제에서 인증/카탈로그 연결은 일어날 수 있지만 no-op의 secret PUT은 발생하지 않습니다.

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="devstack")
    parser.add_argument("--secret", default="s1")
    parser.add_argument("--attrs-json", default="{}")
    args = parser.parse_args()
    attrs = json.loads(args.attrs_json)
    with openstack.connect(cloud=args.cloud) as conn:
        result = conn.key_manager.update_secret(args.secret, **attrs)
        print(json.dumps(result.to_dict(), ensure_ascii=False))


if __name__ == "__main__":
    main()
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
)

func run(ctx context.Context, cloud, id, attributes string) error {
    var attrs map[string]json.RawMessage
    if err := json.Unmarshal([]byte(attributes), &attrs); err != nil { return err }
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.KeyManagerV1(ctx)
    if err != nil { return err }
    result, err := service.Secrets.UpdateRecord(ctx, id,
        secrets.WithUpdateRecordOpts(secrets.UpdateRecordOpts{Attributes: attrs}))
    if err != nil { return err }
    return json.NewEncoder(os.Stdout).Encode(struct {
        Committed bool `json:"committed"`
        StatusCode int `json:"status_code"`
        Resource map[string]json.RawMessage `json:"resource"`
    }{result.Committed, result.StatusCode, result.Resource.Body})
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    id := flag.String("secret", "s1", "literal secret ID")
    attributes := flag.String("attrs-json", "{}", "closed JSON attributes; empty object makes no PUT")
    timeout := flag.Duration("timeout", time.Minute, "whole request deadline")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), *timeout)
    defer cancel()
    if err := run(ctx, *cloud, *id, *attributes); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

```sh
python main.py --cloud devstack --secret s1 --attrs-json '{}'
go run main.go --cloud devstack --secret s1 --attrs-json '{}'
python main.py --cloud devstack --secret s1 --attrs-json '{"name":"new","expiration":null}'
go run main.go --cloud devstack --secret s1 --attrs-json '{"name":"new","expiration":null}'
```

선택된 client가 이미 있으면 `secrets.New(client).UpdateRecord(ctx,"s1",...)`로 직접 호출합니다. `Connection.KeyManagerV1(ctx).Secrets`가 같은 helper를 노출하므로 별도 Connection wrapper나 caller builder를 만들지 않습니다.

`WithUpdateRecordOpts`는 Attributes raw JSON, Headers map, BasePath/Microversion pointer를 생성 시점과 적용 시점에 복사합니다. attribute factory의 호출 순서는 alias 충돌에 적용됩니다. `expiration` 다음 `expires_at`을 주면 마지막 값이 canonical `expiration`으로 전송됩니다. Go map으로 제공한 Attributes는 정렬된 key 순서를 사용하므로 충돌하는 alias를 넣을 때는 순차 factory를 쓰는 편이 명확합니다. 응답은 JSON object insertion order의 마지막 alias가 이깁니다. `content_types`는 request raw 값을 유지하며 descriptor view에서 Source처럼 null 또는 object를 유지하고, 그 밖의 값은 `{}`로 보입니다.

`WithUpdateRecordBasePath("/custom")` 또는 `base_path` JSON attribute는 선택한 서비스 내부의 literal 경로를 사용합니다. Python `%` URI template/임의 URI attrs 전체는 지원하지 않습니다. explicit `WithUpdateRecordHeaders`와 `WithUpdateRecordMicroversion`은 Go transport 옵션입니다. Python의 일반 attrs `microversion`은 session의 기본 commit version을 덮지 않으며, 이 helper에서도 wire body/commit override로 사용하지 않습니다. 기존 selected client의 microversion을 유지하고 Barbican discovery GET은 추가하지 않습니다.

성공 status는 Source의 최종 100..399입니다. 199/204 등에 본문을 넣은 Go 상태 정책 테스트는 synthetic RoundTripper에 한정하며 실제 net/http의 informational/204 body 처리와 구분합니다. 정상 object response는 request seed 위에 알려진 Body 필드를 병합하고, unknown/`self`는 Resource view에 넣지 않습니다. `Wire`와 `Envelope`에는 실제 필드를 보존합니다. 빈 응답과, UTF-8로 올바르게 인코딩됐지만 JSON 문법이 깨진 응답은 원본 `ValueError` 허용처럼 seed를 유지합니다. 정상 JSON이어도 root list/null/scalar는 병합 불가 오류입니다. 기존 HREF formatter를 재사용하여 `secret_ref`/`secret_id`의 마지막 component를 노출하며 반환 ref를 따라가서 GET하지 않습니다.

Read/Close, context 취소, 변경됐다 복원된 selected route, native retry/auth/backoff 오류는 실제 상태·헤더·본문과 함께 유지합니다. 이 실패 뒤 추가 hook 실행을 막고 route/JSON body/header ownership을 지킵니다. clean native retry는 configured policy를 따르며 변조된 retry body는 두 번째 PUT 전에 거부합니다. `Committed=true`는 accepted HTTP body/Close 관찰이 끝났다는 뜻이며 Resource 병합은 뒤에 실패할 수 있습니다. **false도 원격 실행이 없었다는 보장이 아닙니다.** 오류 반환에서 `Resource`가 nonnil이면 request seed일 수 있으며 성공한 원격 상태로 사용하면 안 됩니다. 응답 없는 transport 실패나 본문 처리 실패 후 mutation을 무조건 다시 실행하지 마세요.

남은 named 경계는 다음과 같습니다.

1. 기존 Python Secret/Resource 또는 dictionary seed의 synchronized/dirty 상태, `True==1` 등 변경 비교, 기존 pending dirty 값, 같은 객체 반환과 재커밋, 전체 MRO/constructor/kwargs binding/URI template. 현재 API는 매번 새 문자열 seed를 만듭니다. `id`/`connection`/`_synchronized`/`cls` collision은 preflight 오류지만, Python wrapper의 `self`/`secret`/`resource_type`/`value` 중복 binding은 현재 unknown으로 버리므로 Source의 TypeError/HTTP0와 다릅니다. empty string 등 falsey ID의 PUT route는 Go의 안전한 nonblank literal ID 정책으로 제한합니다. Source no-op의 잘못된 base_path 타입과 같은 지연 검증 세부 순서도 완전히 재현하지 않습니다.
2. Requests `Response.json()`의 encoding/charset/BOM/text fallback, Python의 permissive float/NaN/Infinity loader 전체. 비UTF-8 응답은 `ErrUnsupported`와 실제 raw body/status/header를 유지합니다. Source가 일부 비UTF-8 malformed JSON을 허용하거나 정상 legacy-charset JSON을 파싱할 수 있으므로 이 차이를 단순 Go 안전 개선으로 완료 처리하지 않습니다. 유한 raw JSON 숫자는 원래 literal/큰 정수를 보존하는 Go 정책입니다.

SDK-R1/C1/S1 Resource class/session/cache foundation도 별도 잔여입니다. 이 helper를 추가했다고 generated transport나 raw payload upload만으로 `update_secret` 전체를 완료했다고 판정하지 않습니다.

검증 근거는 `/private/tmp/go-openstacksdk-cycle9-secret-update-source.py`와 `-source-reference.json`의 실제 Source **61개 관찰**입니다. 실제 Proxy/Secret/Resource 생성·dirty 비교·PUT·응답 병합과 CPython `3.14.8` / Requests `2.34.2` / urllib3 `2.8.0`을 실행했고, HTTP adapter의 put만 memory `http.client.HTTPResponse`로 대체했습니다. live Barbican/auth/retry/운영 권한 검증이 아닙니다. Go 회귀는 기존 native Secret transport, internal/testcloud의 Gophercloud public testhelper fixture, secret physical body fixture를 재사용합니다. 담당자는 Go를 실행하지 않았고 root 통합 검증 대상입니다. Python main AST 및 두 main의 가이드 fence byte 일치는 확인했습니다.
