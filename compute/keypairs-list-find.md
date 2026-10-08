# Keypair 목록과 find: Python과 Go

`conn.Compute(ctx)`의 `service.API.KeyPairs`에서 목록과 자동 이름 조회를 사용합니다. `ListRecords`와 `FindKeypair`는 SDK가 기본값·필터·페이지·응답 해석을 처리하는 API이며 애플리케이션에 builder interface를 요구하지 않습니다. 기존 native `List`, `Get`, `Find(resource.Ref)`는 그대로 사용할 수 있습니다.

| 고정 openstacksdk 호출 | Go 호출 | 결과 |
|---|---|---|
| `conn.compute.keypairs(**query)` | `service.API.KeyPairs.ListRecords(ctx, options...)` | `iter.Seq2[*KeypairRecord, error]` |
| `conn.compute.find_keypair(name, ignore_missing=True, user_id=owner)` | `service.API.KeyPairs.FindKeypair(ctx, name, options...)` | `*KeypairRecord`, missing 기본값은 `nil, nil` |

키페어는 user 소유입니다. 자신의 목록/find는 핵심 user 작업이며 다른 owner의 조회 권한은 Nova의 배포 정책이 판단합니다. SDK는 role을 추정하거나 owner를 인증 사용자로 바꿔 권한을 대신 결정하지 않습니다. `user_id`를 전달한 경우 실제 서버의403 등도 유지합니다. 기본 정책 근거는 [Nova policy](https://docs.openstack.org/nova/latest/configuration/policy.html)의 keypair index/show입니다.

```python
import openstack

conn = openstack.connect()
for keypair in conn.compute.keypairs(type="ssh"):
    print(keypair.to_dict())

found = conn.compute.find_keypair("workstation", ignore_missing=True)
if found is not None:
    print(found.to_dict())
```

## 독립 Go main

인증 환경을 준비하고 실행합니다. 기본 호출은 목록이며 `-find NAME`을 지정하면 find만 수행합니다. `-owner-user-id`는 선택 입력입니다. `-microversion`을 생략하면 SDK가 선택을 처리하고, `-microversion=`을 명시하면 version header 없이 조회합니다. 예제 검증은 main의 빌드이며 실제 cloud 실행은 별도입니다.

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "log"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs"
)

func main() {
    name := flag.String("find", "", "keypair name to find instead of listing")
    owner := flag.String("owner-user-id", "", "optional owner")
    kind := flag.String("type", "", "optional local list filter, for example ssh")
    maximum := flag.Int("max-items", 0, "raw list row cap; zero is unlimited")
    strict := flag.Bool("strict", false, "return an error when find has no match")
    version := flag.String("microversion", "", "optional operation version; explicit empty is versionless")
    flag.Parse()
    versionSet := false
    flag.Visit(func(f *flag.Flag) {
        if f.Name == "microversion" { versionSet = true }
    })
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *name, *owner, *kind, *maximum, *strict, *version, versionSet); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, name, owner, kind string, maximum int, strict bool, version string, versionSet bool) error {
    conn, err := sdk.Connect(ctx)
    if err != nil { return err }
    service, err := conn.Compute(ctx)
    if err != nil { return err }
    api := service.API.KeyPairs
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    printRecord := func(record *keypairs.KeypairRecord) error {
        return encoder.Encode(map[string]any{
            "resource": record.Resource, "wire": record.Wire,
            "envelope": string(record.Envelope),
            "status_code": record.StatusCode, "header": record.Header,
        })
    }
    if name != "" {
        options := []keypairs.KeypairFindOption{
            keypairs.WithKeypairFindUserID(owner),
            keypairs.WithKeypairFindIgnoreMissing(!strict),
        }
        if versionSet { options = append(options, keypairs.WithKeypairFindMicroversion(version)) }
        record, err := api.FindKeypair(ctx, name, options...)
        if record != nil {
            if printErr := printRecord(record); printErr != nil { return errors.Join(err, printErr) }
        }
        if err != nil { return fmt.Errorf("find keypair: %w", err) }
        if record == nil { fmt.Println("keypair not found") }
        return nil
    }
    options := []keypairs.KeypairListOption{
        keypairs.WithKeypairListUserID(owner),
        keypairs.WithKeypairListMaxItems(maximum),
    }
    if kind != "" { options = append(options, keypairs.WithKeypairListFilter("type", kind)) }
    if versionSet { options = append(options, keypairs.WithKeypairListMicroversion(version)) }
    for record, err := range api.ListRecords(ctx, options...) {
        if err != nil { return fmt.Errorf("list keypairs: %w", err) }
        if err := printRecord(record); err != nil { return err }
    }
    return nil
}
```

명시적인 pagination `limit`과 `marker`는 [Nova API 기준](https://docs.openstack.org/api-ref/compute/#list-keypairs)2.35 이상에서 지원합니다. raw row cap도 caller limit이 없으면 limit hint를 보내므로 필요하면 예제에 `-microversion 2.35` 등 서버가 지원하는 버전을 지정합니다. SDK는 cap 때문에 버전을 자동 높이지 않습니다. 예제의 기본 cap0은 이 hint를 보내지 않습니다.

## 목록 옵션과 페이지

`KeypairListOpts`는 `UserID`, `Limit`, `Marker`, `MaxItems`, `Paginated *bool`, `Microversion *string`을 갖습니다. `WithKeypairListOptions`와 개별 UserID/Limit/Marker/MaxItems/Paginated/Microversion helper로 설정하며, Header/Query/Filter/Filters는 별도 입력을 유지합니다. bulk Options는 typed 값만 교체합니다. helper가 받은 pointer·filter map과 callback이 만든 mutable carrier는 SDK가 소유하고, 같은 iterator를 다시 순회하면 옵션은 새 상태에 한 번씩 적용합니다. iterator 생성 자체는 HTTP나 callback을 실행하지 않습니다.

| semantic Filter/Filters 속성 | 처리 |
|---|---|
| `user_id`, `limit`, `marker` | 서버 query |
| `id`, `created_at`, `is_deleted`, `fingerprint`, `name`, `private_key`, `public_key`, `type` | Resource에 로컬 필터 |
| unknown 속성, wire 이름 `deleted` | semantic filter에서 버림 |

`WithKeypairListFilters(map)`는 semantic filter만 교체하며 nil/empty로 비웁니다. 반복된 개별 Filter는 마지막 값이 이깁니다. raw `WithKeypairListQuery`는 같은 typed query를 덮지만, semantic query와 typed/raw query가 겹치면 값이 같아도 오류입니다. Body filter의 객체 조건은 공통 matcher를 사용합니다. 응답의 `deleted`만 bool view로 변환하고 caller 필터 값은 변환하지 않습니다.

`MaxItems`는 반환된 일치 행 수가 아니라 **로컬 필터 이전에 처리한 raw 행 수**를 제한합니다. typed Limit/MaxItems의0은 생략/무제한, 음수는 오류이며, 전송되는 limit은 하나의 양의 정수이고 marker는 하나의 비어 있지 않은 문자열이어야 합니다. `WithKeypairListPaginated(false)`는 첫 페이지만 읽고 `break`는 뒤 요청을 중단합니다. 늦은 오류가 발생하면 이미 받은 결과는 유효하지만 전체 목록 성공으로 취급하면 안 됩니다.

SDK는 rel/href `links`, `keypairs_links`, top-level `next`, HTTP Link를 처리하고, 초기 limit이 있을 때 마지막 raw 행의 논리 name을 fallback marker로 사용합니다. 그 행이 로컬 필터에서 빠지거나 caller가 반환한 Resource를 수정해도 marker는 원래 행에서 계산합니다. 첫 advertised next가 positive limit을 추가하면 이후 페이지에서 유지하지만, server가 추가한 limit만으로 marker fallback을 켜지는 않습니다. 빈 페이지는 next가 있어도 중단합니다. `links:{next:...}`는 Python의 ordinary next dictionary 처리와 다른 **Go 추가 호환**입니다. 서로 다른 next·cycle·다른 collection route/query로의 전환은 오류입니다.

Python public query에서 도달하는 arbitrary `base_path`, deprecated `jmespath_filters`, `allow_unknown_params`는 이 고정 경로 API에서 지원하지 않습니다. controls를 raw query나 semantic 속성으로 보내지 않고 dedicated 옵션으로 표현하거나 명시적 unsupported 오류로 처리합니다.

## Resource·Wire·Envelope

두 owned API는 기존 CreateKeypair와 같은 `KeypairRecord`를 반환합니다. Resource는 `created_at`, `is_deleted`, `fingerprint`, `id`, `name`, `private_key`, `public_key`, `type`, `user_id`의9개 Body 속성을 갖습니다. Python의 computed Connection location과 mutable Resource 메서드는 이 leaf 결과에 합성하지 않습니다.

목록은 wrapped/flat row를 모두 처리하고 nested keypair 값이 outer 값을 덮습니다. 논리 id는 name이며 name이 있으면 null·빈 문자열도 literal id보다 우선합니다. 없는 type만 ssh, present null type은 null이며 다른 없는 속성은 null입니다. deleted의 문자열 `"false"`는 true, 빈 배열은 false, null은 null입니다. `deleted`/`is_deleted`가 함께 있는 응답에서는 Go의 canonical deleted가 우선하므로 Python insertion order와 구별합니다.

| 반환 필드 | ListRecords | FindKeypair의 직접 GET |
|---|---|---|
| `Resource` | 병합·alias·default를 적용한9속성 view | 요청 name/optional owner seed 위에 실제 응답 값을 덮은 view |
| `Wire` | wrapper·outer·unknown 값을 포함한 실제 행 | keypair envelope가 있으면 그 실제 객체, 없으면 flat root |
| `Envelope` | 실제 행 JSON | 실제 HTTP body 전체 |
| `Header`, `StatusCode` | 실제 페이지 응답 | 실제 member 응답 |

Resource·Wire·Envelope·Header는 독립적으로 소유합니다. 큰 JSON 수와 unknown 필드는 Wire에 유지되며 요청 seed/default를 Wire에 넣지 않습니다. Find의 parsed response name/id는 숫자·null 등도 passive 값으로 유지하고 다른 요청 target으로 사용하지 않습니다. 숫자 name은 fallback의 caller 문자열과 같다고 취급하지 않습니다.

owned HTTP 허용 범위는200..399입니다. 목록은 JSON object의 keypairs array 또는 singleton object를 요구하고 malformed/empty/null representation은 오류입니다. 따라서 빈204도 목록 오류입니다. Find는 source fetch처럼 accepted nonJSON·malformed·빈 body에 seed를 유지하고 nil Wire를 반환하지만, 유효 JSON의 root/keypair가 null·scalar·array이면 오류입니다. invalid UTF-8는 목록에서 오류, find에서는 unavailable JSON으로 처리하며 Python response charset/replacement decoding까지 재현하지 않습니다.

수락한 응답의 read/Close/source/context 실패는 실제 body·header·status와 원래 cause를 `resource.ResponseError`에 보존합니다. Find는 receipt record를 error와 함께 반환할 수 있고 Resource가 nil일 수 있습니다. 목록 오류의 ResponseError는 whole page를 보존하며 성공 행의 Envelope는 row입니다. 오류를 nested404로 감추거나 accepted 응답을 재조회하지 않습니다.

## Find 정책과 버전

Find는 입력을 trim하거나 UUID로 추정하지 않고 UTF-8 literal name을 한 번 escape하여 `/os-keypairs/{name}`를 먼저 GET합니다. 내부 공백·reserved 문자는 허용하며 empty/whitespace-only·controls·dot routing은 Go에서 거부합니다. 명시 owner의 빈 문자열은 query에서 생략합니다. 서버가 처리할 이름/owner의 유효성·권한을 별도 로컬 검사로 대신하지 않습니다.

직접 GET의 **clean actual400·403·404만** 같은 owner의 전체 목록 fallback을 허용합니다. name query hint는 추가하지 않습니다. 유일성 확인을 위해 match 이후에도 목록을 읽고, 중복은 `resource.ErrAmbiguous`, 성공한 전체 목록의0건은 기본 `nil, nil` 또는 strict `resource.ErrNotFound`입니다. 409·5xx·late 목록 오류 및 nested HTTP 오류가 포함된 physical/source/context 오류는 missing 성공으로 바뀌지 않습니다. 기존 `Find(resource.ID/Name)`의 explicit 선택 정책과는 별개입니다.

Microversion pointer가 nil이면 선택된 client version을 그대로 쓰고, 미선택일 때만 [공통 Nova discovery](console-selection.md)로 최대2.10을 선택합니다. 광고 max가 없거나 min이 ceiling보다 높으면 versionless로 계속합니다. `WithKeypairListMicroversion("")`/`WithKeypairFindMicroversion("")`는 discovery 없이 header를 생략하는 명시 override입니다. 원래 client는 수정하지 않고 live provider 인증을 사용하며, 이미 선택된 버전을 ceiling에 맞춰 바꾸지 않습니다. Find의 discovery·옵션은 member와 fallback 전체에서 한 번 처리합니다. Python 전체 discovery sorting/cache/session과 같은 구현이라고 주장하지 않으며 discovery JSON/physical/source/context 오류는 terminal로 보존합니다.

native `List(ctx, ...ListOption)`는 기존 `[200,204,300]` 허용 코드, `SinglePageBase`,6개 string 필드 `Name/Fingerprint/PublicKey/PrivateKey/UserID/Type`를 유지합니다. 광고 next를 따르지 않고 native missing/null type은 빈 문자열입니다. 검증한 JSON Content-Type 없는 empty204는 빈 페이지이며, JSON Content-Type이 있으면 native decoder가 먼저 실행되므로 빈 body의 decode 오류가 생길 수 있습니다. owned ListRecords의 필터·default·페이지 정책이 native List를 변경하지 않습니다.

고정 소스는 [Proxy keypairs/find_keypair](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L910-L978), [Keypair 모델](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/keypair.py), Resource.list/find입니다. [생성과 console](keypairs-console.md), [Keypair leaf API](v2/keypairs/README.md)와 함께 사용합니다. 실행 검증과 named 지원 판정은 [검증 기록](../docs/sdk-support-ledger.md)에 별도로 남깁니다.
