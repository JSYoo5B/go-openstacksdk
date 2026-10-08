# Glance Metadef property 조회

`get_metadef_property(property, namespace)`에 대응하는 기존 typed Go API는 `service.API.MetadefProperties.InNamespace(ctx, namespace)`로 parent를 고정한 뒤 `scope.Get(ctx, property)`를 호출하는 방식입니다. namespace와 property는 이름 그대로 사용하며 namespace 조회·이름 검색·schema 조회를 선행하지 않습니다. 생성·교체·삭제·dictionary 목록은 [leaf 가이드](v2/metadefproperties/README.md)를 참고합니다. 초기 속성·descriptor 기본값과 변환이 필요한 새 `GetRecord`는 [레코드 조회 가이드](metadef-property-records.md)에 설명합니다. 아래 비교는 기존 raw typed Get의 경계를 유지합니다.

## Python과 Go

Python의 기본 호출:

```python
import openstack

conn = openstack.connect(cloud="dev")
value = conn.image.get_metadef_property("hw_cpu_policy", "OS::Compute::Libvirt")
print(value.to_dict())
```

Go의 기본 호출:

```go
service, err := conn.Image(ctx)
if err != nil { return err }
scope, err := service.API.MetadefProperties.InNamespace(ctx, "OS::Compute::Libvirt")
if err != nil { return err }
value, err := scope.Get(ctx, "hw_cpu_policy")
if err != nil { return err }
```

두 호출은 기본적으로 child 경로에 GET을 보냅니다. Go의 scope 생성은 HTTP 없이 namespace와 service target을 검증·고정합니다. 반환된 Name·Self·Schema·Location header는 후속 요청을 정하지 않습니다.

## Python의 `**query`는 일반 HTTP query가 아니다

고정 openstacksdk의 공개 getter는 `**query`를 `Proxy._get`에 전달합니다. `_get`은 `requires_id=True`, `base_path=None`, `skip_cache=False`를 별도 제어값으로 소비하고 나머지는 `_get_resource`의 초기 속성으로 전달합니다. 생성된 Resource의 `fetch`에는 이 초기 속성을 `**params`로 다시 넘기지 않습니다.

다음 Python 호출의 `resource_type`은 공개 getter가 이미 positional로 바인딩한 `Proxy._get`의 resource_type 인자와 중복됩니다. 따라서 **HTTP 전에 TypeError**이며, 일반 unknown 속성으로 무시하거나 HTTP query로 전달하지 않습니다.

```python
# TypeError before HTTP: Proxy._get already binds resource_type.
value = conn.image.get_metadef_property(
    "hw_cpu_policy", "OS::Compute::Libvirt", resource_type="OS::Nova::Server"
)
```

공개 `**query`의 `value`·`namespace_name`도 이미 바인딩한 인자와 충돌합니다. 반면 requires_id·base_path·skip_cache는 `_get`의 실제 제어값입니다. Python의 실패하는 호출을 아래 Go 호출과 같은 wire 동작이라고 설명하면 안 됩니다.

```go
value, err := scope.Get(ctx, "hw_cpu_policy",
    metadefproperties.WithGetResourceType("OS::Nova::Server"))
```

`WithGetResourceType`은 Glance의 server query를 명시적으로 이용하는 Go 추가 기능입니다. nil/default는 query를 생략하고, 명시한 빈 문자열은 `resource_type=`를 보냅니다. 값은 literal query로 한 번 encode하며, SDK가 association을 조회하거나 property 이름에서 prefix를 제거하지 않습니다. `WithGetOpts`는 header와 ResourceType을 포함한 전체 옵션을 교체하고, 이후 옵션이 우선합니다.

인식된 Python Body 속성은 초기 Resource에 남고 응답에 같은 속성이 있으면 교체됩니다. 응답이 그 속성을 생략하면 초기 값이 유지될 수 있습니다. 문자열 property 입력은 `id`로 생성되고, namespace Resource 입력은 `_get_id`로 선택한 id/name을 parent URI에 사용합니다. 이미 존재하는 MetadefProperty 입력은 같은 객체를 갱신합니다. Go의 기존 typed leaf는 고정 literal 입력과 각 응답의 새 DTO를 사용하므로 이런 mutable Resource·seed·cache 동작을 제공하지 않습니다. 공통 Resource 동작은 전체 SDK의 별도 구현 범위입니다.

## 반환값과 기본값

| 항목 | 고정 Python Resource | 기존 Go `*metadefproperties.Property` |
|---|---|---|
| parent | `namespace_name` URI 속성 유지 | scope의 `NamespaceName()`에 있음; 응답에 합성하지 않음 |
| identity | 입력 id 또는 alternate name을 Resource에서 사용 | 요청 이름은 literal; 응답 Name은 nullable string, Key는 nil |
| `minLength`, `minItems` 누락 | descriptor 기본값 0 | Body에 키를 추가하지 않음 |
| `uniqueItems` 누락 | descriptor 기본값 False | Body에 키를 추가하지 않음 |
| 위 세 키의 명시 null | null 유지 | 실제 raw null 유지 |
| int/list/dict/bool keyword | descriptor 조회 시 타입 변환 | Body의 RawMessage에 실제 JSON 유지 |
| keyword 별칭 | 인식한 descriptor 이름으로 수집 | wire 키와 별칭을 독립 raw 키로 유지 |
| unknown 키 | 기본 Resource의 선언 속성에 포함되지 않음 | 전체 Body에 보존 |
| `self` | 응답 처리 중 제거 | nullable Self와 raw Body에 보존 |
| location | Connection의 computed Resource location | 기존 leaf DTO에서 합성하지 않음 |

Python의 min/max/min/max length·item count는 int descriptor이며 fractional 값의 조회 결과가 달라질 수 있습니다. operators·enum은 list, items는 dict, readonly·uniqueItems·additionalItems는 bool descriptor입니다. Go Body는 `1.75`, `9007199254740993`, `1e400`, 명시 null, scalar·array·object를 그대로 보존하고 Python 변환을 적용하지 않습니다. canonical Name·Type·Title·Description·Self·Schema·CreatedAt·UpdatedAt은 string 또는 null이어야 하며 잘못된 nonnull type은 원자적 오류입니다.

## 실제 응답과 오류

기존 typed Get은 실제 HTTP200만 성공으로 처리하고 valid UTF-8의 nonnull JSON object를 요구합니다. 이름 누락을 요청 이름으로 채우거나 이전 성공 결과를 재사용하지 않습니다. HTTP304도 native HTTP 오류이며 cache 결과로 대체하지 않습니다.

고정 Python Resource는 HTTP400 미만 응답에서 translation을 수행하고, JSON 파싱의 ValueError를 무시해 초기 Resource를 반환할 수 있습니다. parsed nonobject의 처리 실패까지 무시하는 것은 아닙니다. 기존 Go leaf의 strict200·strict decoder와 이 동작은 다릅니다.

Go에서 accepted200의 read·Close·UTF-8·JSON·canonical·context·source 처리가 실패하면 Property는 nil이며 `*resource.ResponseError`가 실제 Body·Header·StatusCode와 원인을 보존합니다. rejected status는 Gophercloud의 native `ErrUnexpectedResponseCode` 등으로 확인합니다. 옵션 오류처럼 HTTP 전에 실패한 경우에는 response receipt가 없습니다. 오류 뒤 accepted body를 재전송하거나 다른 namespace·property·schema로 넘어가지 않습니다.

## 독립 Go main

이 예제는 query 없이 한 번 조회하고 실제 response body를 출력합니다. Go 문서 예제의 검증은 컴파일과 로컬 HTTP 계약을 기준으로 하며, 이 문서가 인증된 cloud·Python 실행을 증명하지는 않습니다.

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/resource"
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    namespace := flag.String("namespace", "OS::Compute::Libvirt", "literal namespace")
    name := flag.String("name", "hw_cpu_policy", "literal property name")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *namespace, *name); err != nil { log.Fatal(err) }
}

func printFailure(err error) error {
    var accepted *resource.ResponseError
    var unexpected gophercloud.ErrUnexpectedResponseCode
    var receipt map[string]any
    switch {
    case errors.As(err, &accepted):
        receipt = map[string]any{"status_code": accepted.StatusCode,
            "header": accepted.Header, "body": string(accepted.Body)}
    case errors.As(err, &unexpected):
        receipt = map[string]any{"status_code": unexpected.Actual,
            "header": unexpected.ResponseHeader, "body": string(unexpected.Body)}
    default:
        return nil
    }
    data, printErr := json.MarshalIndent(receipt, "", "  ")
    if printErr != nil { return printErr }
    fmt.Println(string(data))
    return nil
}

func run(ctx context.Context, cloud, namespace, name string) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    scope, err := service.API.MetadefProperties.InNamespace(ctx, namespace)
    if err != nil { return err }
    value, err := scope.Get(ctx, name)
    if err != nil { return errors.Join(err, printFailure(err)) }
    data, err := json.MarshalIndent(value.Body, "", "  ")
    if err != nil { return err }
    fmt.Printf("namespace=%q requested_name=%q status_code=%d\n%s\n",
        scope.NamespaceName(), name, value.StatusCode, data)
    return nil
}
```

기존 [core](v2/metadefproperties/core_test.go)·[계약](v2/metadefproperties/contracts_test.go)·[옵션](v2/metadefproperties/options_test.go)·[Connection](../connection_image_metadef_properties_test.go) 검증을 재사용합니다. [조회 경계 검증](v2/metadefproperties/get_mapping_test.go)은 descriptor keyword/default의 raw 보존과 성공→304→새 성공 사이의 결과 독립성을 추가로 확인합니다. 검증 결과와 연산 판정은 [판정대장](../docs/sdk-support-ledger.md)에 실제 실행 후 기록합니다.

비교 기준은 고정 [public getter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1900-L1931), [MetadefProperty](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_property.py#L21-L79), [Proxy._get_resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L563-L605), [Proxy._get](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L822-L865), [Resource.fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1778-L1849), [response translation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338-L1398)입니다.
