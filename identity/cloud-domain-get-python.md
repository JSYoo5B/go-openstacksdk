# Domain 단건 조회: openstacksdk와 Go

고정 Source는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 Proxy.get_domain, IdentityCloudMixin.get_domain 및 Domain/Resource/Proxy이고 native 기반은 Gophercloud v2.15.0입니다. 이번 범위는 이 두 named get 선언입니다. 기존 native Domain Get/Create/Update와 full Proxy.find_domain 판정은 유지합니다.

| Python | Go |
|---|---|
| conn.identity.get_domain(id) | service.Domains.GetRecord(ctx, id, ...RecordGetOption) |
| conn.get_domain(domain_id=id) | identity.NewDomainReadCloud(service.Domains).GetDomain(ctx, WithDomainGetID(id)) |
| conn.get_domain(name_or_id=name) | DomainReadCloud.GetDomain(ctx, WithDomainGetNameOrID(name)) |
| ignored deprecated filters | WithDomainGetFilters(any), result.DeprecatedFilters flag only |

## 실행 가능한 Python

OS_CLOUD/clouds.yaml 또는 인증 환경을 준비하고 실제 domain ID/name으로 실행합니다.

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--domain-id", required=True)
    parser.add_argument("--domain-name", default="")
    args = parser.parse_args()
    conn = openstack.connect()
    row = conn.identity.get_domain(args.domain_id)
    print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))
    row = conn.get_domain(domain_id=args.domain_id,
                          filters={"ignored": "without evaluation"})
    print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))
    if args.domain_name:
        found = conn.get_domain(name_or_id=args.domain_name)
        if found is not None:
            print(json.dumps(found.to_dict(), indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
```

## 실행 가능한 Go

같은 인증 환경의 Go module에 아래 main.go를 저장하고 `go run . -domain-id DOMAIN_UUID -domain-name DOMAIN_NAME`으로 실행합니다. Name flag는 생략할 수 있습니다.

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/identity"
)

func printJSON(value any) {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Printf("%s\n", data)
}

func main() {
    id := flag.String("domain-id", "", "required domain UUID")
    name := flag.String("domain-name", "", "optional domain name")
    flag.Parse()
    if *id == "" { log.Fatal("-domain-id is required") }
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    row, err := service.Domains.GetRecord(ctx, *id)
    if err != nil { log.Fatal(err) }
    printJSON(row.Resource.Body)

    cloud := identity.NewDomainReadCloud(service.Domains)
    result, err := cloud.GetDomain(ctx,
        identity.WithDomainGetID(*id),
        identity.WithDomainGetFilters(json.RawMessage(`{invalid-but-ignored`)),
    )
    if err != nil { log.Fatal(err) }
    printJSON(result.Record.Resource.Body)
    fmt.Println("deprecated filters ignored:", result.DeprecatedFilters)
    if *name != "" {
        found, err := cloud.GetDomain(ctx, identity.WithDomainGetNameOrID(*name))
        if err != nil { log.Fatal(err) }
        if found.Record != nil { printJSON(found.Record.Resource.Body) }
    }
}
```

## Direct GET의 선언 view와 실제 증거

GetRecord는 [Domain record 목록](v3/domains/python-records.md)의 private Source profile/converter를 재사용합니다. Raw nullable description/links/id/name, Python bool is_enabled←enabled와 dict options(nonobject {}, null 유지)를 제공합니다. 누락 id에는 constructor에 넣은 요청 ID가 남고 명시 response id:null은 그대로 유지합니다. Unknown response·서버 location은 Wire에만 유지합니다. Resource/원래 Wire/전체 Envelope/실제 Header·StatusCode와 Clone은 별도 소유입니다. Response ID/HREF로 routing을 바꾸지 않습니다.

Literal GET 하나만 보내고 403/404에서 LIST로 바꾸지 않습니다. 새 Domain profile의 valid100..399 status와 non-JSON/flat metadata seed 규칙을 사용하며 bad singular envelope/read/close 오류는 실제 ResponseError, clean rejected 오류는 native status/body/header/method/URL proof를 보존합니다. Native *ErrUnableToReauthenticate는 Unwrap가 없어 ErrOriginal 값형 response error와 ErrReauth 원인을 각각 검사합니다. Physical fault/context/custom cancel/source 변경은 replay·reauth·backoff로 숨기지 않고 clean401/503/429/498만 live token과 captured route/query/header/version으로 처리합니다.

RecordGetOpts/WithRecordGetOpts·Headers·Microversion·Location은 immutable Go 편의 옵션입니다. Proxy.get_domain에는 `**query`가 없으므로 arbitrary Resource.fetch controls를 추가하지 않습니다. Default direct location은 null이고 explicit CloudLocation은 configured project를 유지하며 unknown response scope를 신뢰하지 않습니다. 자동 Connection bridge는 주장하지 않습니다.

## Cloud presence와 ignored filters

DomainID/NameOrID는 string pointer로 omission을 보존합니다. DomainID가 nonnil이면 name_or_id와 무관하게 direct GET이며 missing404는 오류입니다. DomainID가 nil이고 NameOrID가 있으면 closed ResolveRecord가 GET-first → clean400/403/404의 name-hinted 전체 LIST fallback을 수행합니다. Exact ID OR name, repeated ID도 중복, 늦은 HTTP 오류 우선, 기본 logical missing은 Record:nil/nil error입니다. GET-only microversion override와 LIST session default, Source links/marker/actual guard는 기존 공통 엔진을 재사용합니다. 이 helper로 full Proxy.find_domain를 별도 승격하지 않습니다.

Filters는 Source의 deprecated no-op입니다. WithDomainGetFilters는 값 자체를 보관·Marshal·JSON parse·filter·IO하지 않고 nil/non-nil presence flag만 기록합니다. Invalid JSON, marshaler 함수, false도 무시합니다. Source warnings.warn과 Go DeprecatedFilters flag는 표현 방식이 다르고 Python 전역 warning-as-error 설정을 복제하지 않습니다.

Source doc comment는 domain_id/name_or_id 둘 다 None일 때 None 반환이라고 쓰지만, 실제 고정 Resource.find(None)→fetch(requires_id=True)→_prepare_request는 **InvalidRequest를 raise하며 HTTP0**입니다. Actual AST chain으로 확인했고 Go도 두 인자 omission에는 ErrInvalidOption/no fabricated receipt를 반환합니다. Source empty string/0/false ID는 utils.urljoin에서 domains/ collection request로 흘러가는 반면 Go는 required nonblank string literal만 받아 HTTP 전에 거절합니다. Scalar object/dict/mutable Resource 입력과 안전하지 않은 raw URI composition은 명시적 Go string API·SDK-C1/S1 경계이며 numeric ID를 stringify하여 Source wire와 같다고 주장하지 않습니다.

## 권한과 지원 경계

[Keystone28.0.0 get_domain 정책](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/domain.py)은 admin/system reader 외에 token의 own domain 또는 token project의 domain 사용자 경로도 허용합니다. 이는 project reader의 전체 Domain 목록 권한을 뜻하지 않습니다. Closed find의 LIST fallback은 [list_domains](v3/domains/python-records.md)의 system/domain reader 정책을 필요로 합니다. SDK는 token/scope를 바꾸지 않고 실제403을 보존합니다.

새 leaf7개·Cloud6개 테스트는 기존 native/testcloud/Gophercloud public testhelper를 재사용합니다. Nullable raw/options/seed/status·no-fallback·closed missing/duplicate/late proof·version·clone/location·sticky options/header/source/cancel, physical28 lanes와 clean retry4 lanes를 검사합니다. Source oracle은 actual Resource.find/fetch/prepare_request와 Cloud get_domain을 AST로 실행하고 constructor/session/translator collaborators를 stub합니다. Mock dispatch/oracle을 실제 cloud 통합으로 세지 않습니다. Root의 최신 Go/race/main gate만 PASS 근거입니다.

이 두 finite get mapping의 generic Resource/session response cache·synchronized object lifecycle·자동 Connection bridge는 SDK-R1/C1/S1 차이로 남습니다. Generic 범위를 이유로 이 named get의 구현된 options 보존을 다시 미구현으로 세지 않고, fullSDK supported 완료를 주장하지 않습니다. Full find_domain의 별도 named audit와 raw fetch-control 확장은 이번 범위에 포함되지 않습니다. Supported 증가는 0입니다.
