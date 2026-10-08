# Glance Metadef property 레코드 조회: Python과 Go

`NamespaceScope.GetRecord`는 property의 초기 속성·응답 overlay·descriptor 기본값과 변환을 SDK에서 처리합니다. 반환값은 declared `Resource`, 실제 `Wire`, 전체 응답 receipt를 구분합니다. 기존 strict typed `Get`과 `WithGetResourceType`의 server query는 [기존 조회 가이드](metadef-property.md)에 설명합니다.

두 조회는 고정 child 경로 `/metadefs/namespaces/{namespace}/properties/{id}`를 사용합니다. 레코드 조회는 namespace scope를 HTTP 없이 고정한 뒤 query·body 없는 GET을 수행합니다. namespace·property 이름 검색이나 schema 조회를 선행하지 않습니다. Provider의 retry·재인증 정책은 별도로 적용됩니다.

## Python 사용

```python
import openstack

conn = openstack.connect(cloud="dev")
value = conn.image.get_metadef_property(
    "hw_cpu_policy", "OS::Compute::Libvirt",
    title="fallback title", min_length=0,
)
print(value.to_dict())
```

공개 getter의 `**query`에서 recognized Body 값은 Resource의 초기 속성입니다. 위 title·min_length를 HTTP query로 보내지 않습니다. 응답에 title이 있으면 그 값으로 교체하고, 생략하면 seed title을 유지합니다. 명시 null도 응답 값으로 교체됩니다.

문자열 입력은 id로 seed합니다. Resource 입력은 기존 속성과 alternate name을 사용할 수 있고, Python은 그 Resource 자체를 갱신합니다. Go는 아래 concrete request를 snapshot하고 새 record를 반환하므로 caller의 입력 Resource를 수정하지 않습니다.

## 독립 Go main

`-cloud`로 clouds.yaml 설정을 선택하고 `-seed-title`은 응답이 title을 생략했을 때 남길 초기 값입니다. accepted empty·invalid JSON도 receipt를 출력할 수 있도록 Envelope를 문자열로 표시합니다. 예제의 검증 범위는 컴파일과 로컬 HTTP 계약이며 인증된 OpenStack·Python 실행을 증명하지 않습니다.

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

    "github.com/JSYoo5B/go-openstacksdk"
    properties "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties"
    "github.com/JSYoo5B/go-openstacksdk/resource"
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    namespace := flag.String("namespace", "OS::Compute::Libvirt", "literal namespace")
    id := flag.String("id", "hw_cpu_policy", "literal property id")
    title := flag.String("seed-title", "fallback title", "title retained when response omits it")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *namespace, *id, *title); err != nil { log.Fatal(err) }
}

func printFailure(err error) error {
    var accepted *resource.ResponseError
    var unexpected gophercloud.ErrUnexpectedResponseCode
    var receipt map[string]any
    switch {
    case errors.As(err, &accepted):
        receipt = map[string]any{
            "status_code": accepted.StatusCode, "header": accepted.Header,
            "body": string(accepted.Body),
        }
    case errors.As(err, &unexpected):
        receipt = map[string]any{
            "status_code": unexpected.Actual, "header": unexpected.ResponseHeader,
            "body": string(unexpected.Body),
        }
    default:
        return nil
    }
    data, printErr := json.MarshalIndent(receipt, "", "  ")
    if printErr != nil { return printErr }
    fmt.Println(string(data))
    return nil
}

func run(ctx context.Context, cloud, namespace, id, title string) error {
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    scope, err := service.API.MetadefProperties.InNamespace(ctx, namespace)
    if err != nil { return err }
    record, err := scope.GetRecord(ctx, properties.RecordRequest{ID: id},
        properties.WithRecordGetAttributes(map[string]any{
            "title": title, "min_length": 0,
        }),
        properties.WithRecordGetHeader("X-Request-Source", "metadef-property-record-example"),
    )
    if err != nil { return errors.Join(err, printFailure(err)) }
    data, err := json.MarshalIndent(map[string]any{
        "namespace": record.Namespace,
        "resource": record.Resource, "wire": record.Wire,
        "envelope": string(record.Envelope),
        "header": record.Header, "status_code": record.StatusCode,
    }, "", "  ")
    if err != nil { return err }
    fmt.Println(string(data))
    return nil
}
```

## Request·속성·identity

`RecordRequest`에는 `ID string`과 `Resource *resource.RawResource`가 있습니다. 둘 중 하나를 지정합니다. Resource는 호출 시작 때 clone하며 recognized Body 값만 seed로 사용합니다. scope가 parent를 고정하므로 입력의 namespace_name·location·unknown 값으로 경로를 바꾸지 않습니다.

Resource의 id 키가 있으면 null을 포함해 그 키를 선택하고, id가 없을 때만 name을 alternate ID로 선택합니다. **HTTP 요청 identity는 nonempty 안전한 literal string이어야 합니다.** namespace·선택한 id는 valid UTF-8, 최대80rune이며 slash·backslash·percent·query·fragment·control과 정확한 dot 경로를 거부합니다. UUID 검사·trim·Name 검색·returned self follow를 하지 않습니다.

옵션 속성은 Resource seed에 먼저 overlay합니다. 따라서 Resource 입력에 `WithRecordGetAttribute("id", "another-property")`를 주면 HTTP 전에 선택한 child가 바뀝니다. `RecordRequest{ID: "property"}`와 id 속성을 함께 주면 Python의 문자열 생성자 keyword 충돌처럼 HTTP 전에 오류입니다. 응답의 id·name은 반환 데이터이며 후속 요청을 재선택하지 않습니다.

`RecordGetOpts`는 Headers와 Attributes carrier를 갖고 `WithRecordGetOpts`는 전체 설정을 교체합니다. `WithRecordGetHeader(s)`와 `WithRecordGetAttribute(s)`는 header·속성을 추가하거나 교체합니다. bulk Attributes는 semantic 속성 집합을 교체하고 individual Attribute는 해당 속성을 추가합니다. 입력 map·slice·callback 설정은 복사하며 각 옵션은 호출마다 한 번 적용합니다. 같은 속성의 뒤 값이 우선하고 unknown 일반 속성의 captured encoding 오류는 노출하지 않습니다.

Attributes의 내부 carrier는 공통 `resource.WithFilter`·`WithFilters`를 재사용하지만 **이 getter에서는 query·로컬 목록 필터가 아니라 초기 Resource 속성**입니다. nil/error 옵션·선택된 속성의 잘못된 JSON·descriptor 변환 실패·보호된 header 충돌은 HTTP 전에 오류입니다. 인증·version·representation·framing header는 SDK가 소유합니다.

## Resource 필드와 변환

고정 Python의 기본 `to_dict()`는18개 선언 Body·inherited id·computed location의20개 필드를 반환합니다. Go Resource는 여기에 고정 namespace_name을 추가한21개 필드이며 `Record.Namespace`에도 parent를 보존합니다. Wire에는 응답의 unknown·self·schema·date·namespace_name·location도 그대로 남습니다.

| Resource 필드 | 누락 기본값 | nonnull 값 처리 |
|---|---|---|
| `id`, `name`, `type`, `title`, `description`, `default`, `pattern` | null | untyped raw JSON 유지 |
| `operators`, `enum` | null | array 유지, nonarray는 `[value]` |
| `items` | null | object 유지, nonobject는 `{}` |
| `minimum`, `maximum`, `max_length`, `max_items` | null | int descriptor 변환 |
| `min_length`, `min_items` | 0 | int descriptor 변환 |
| `is_readonly`, `allow_additional_items` | null | Python truthiness에 따른 bool |
| `require_unique_items` | false | Python truthiness에 따른 bool |
| `namespace_name` | 고정 scope | 응답·seed로 교체하지 않음 |
| `location` | Connection snapshot 또는 null | 응답·seed로 교체하지 않음 |

**명시 null은 모든 descriptor에서 null로 유지**합니다. 기본값은 seed와 응답을 합친 뒤에도 누락한 필드에만 적용합니다. `min_length: null`을0으로 바꾸거나, 응답이 누락한 seed title을 지우지 않습니다. Python의 `items` descriptor는 dict.items와 이름이 겹치므로 Python 예제는 `to_dict()`를 사용합니다.

int descriptor는 정수 JSON을 정확히 보존하고, 소수·exponent JSON은 IEEE-754 float 값으로 해석한 뒤0 방향으로 자릅니다. bool은 Python의 int subclass 동작처럼 bool 그대로 유지합니다. decimal digit 문자열은 정수로 변환하지만 `"-12"`·`" 12"`·일반 nonnumeric container는0입니다. 숫자 변환 오류는 실패이며 무조건0으로 숨기지 않습니다. Unicode digit 분류는 기존 Unicode16 테이블을 사용하고 큰 정수에는 Python runtime의 configurable digit-string limit을 적용하지 않습니다.

bool descriptor는 empty string·array·object와 숫자0을 false로, 나머지를 true로 봅니다. 숫자는 정확한 decimal0 여부를 사용하므로 `1e-400`도 true입니다. Python JSON backend의 float underflow로0이 되는 경우와는 다릅니다. list·dict 변환 뒤의 nested vendor JSON은 추가로 coercion하지 않습니다.

canonical/wire 별칭은 `is_readonly/readonly`, `min_length/minLength`, `max_length/maxLength`, `require_unique_items/uniqueItems`, `min_items/minItems`, `max_items/maxItems`, `allow_additional_items/additionalItems`를 수용합니다. unordered Resource/attribute map에서 둘이 함께 있으면 canonical 값이 null을 포함해 우선합니다. 이때 superseded wire 별칭의 captured encoding 오류도 선택한 값의 검증에 반영하지 않습니다. 응답 JSON에서는 parsed dictionary의 member 순서를 따라 뒤의 recognized 별칭이 우선하고, 같은 키가 반복되면 첫 등장 위치·마지막 값을 기준으로 판단합니다. Wire는 두 spelling을 독립 보존합니다.

Connection location은 옵션·HTTP 전에 한 번 snapshot합니다. fixed namespace URI가 Source collector에 항상 존재하고 fetch는 Body만 overlay하므로 seed·응답 location은 Resource를 교체하지 않습니다. `conn.Image(ctx).API.MetadefProperties`와 `conn.ImageV2(ctx).MetadefProperties` 양쪽에 CurrentLocation이 연결됩니다. 직접 `properties.New(client)`를 사용하면 location은 null이고 `properties.NewWithDependencies`로 concrete getter를 공급할 수 있습니다. Go의 명시 Zone은 유지되며 고정 Python current_location의 zone=None과 차이가 있습니다.

## 실제 응답과 오류

actual HTTP200..399는 receipt를 읽은 뒤 전체 UTF-8을 검증합니다. valid object이면 root 자체의 recognized 필드를 seed에 overlay하고 Wire를 채웁니다. property·properties·schema envelope를 자동 unwrap하지 않습니다. **빈 body·opaque·문법적으로 잘못된 JSON은 성공**이며 seed·defaults·computed location으로 Resource를 만들고 Wire는 nil입니다. Envelope·Header·StatusCode는 이때도 실제 응답입니다.

반면 valid JSON의 null·array·string·number·bool root는 오류입니다. invalid UTF-8와 실제 read·Close·transport·context·source·descriptor 오류도 defaults 성공으로 바꾸지 않습니다. 오류 시 Record는 nil이고 실제 accepted 응답이 있으면 `*resource.ResponseError`에 Body·Header·StatusCode와 원인을 남깁니다. rejected status는 native Gophercloud 오류입니다. HTTP 전에 실패하면 response receipt가 없습니다.

Resource·Wire·Envelope·Header는 독립 소유하며 caller의 변경이 다른 채널이나 입력 Resource를 바꾸지 않습니다. source·scope가 관찰된 뒤 바뀌면 오류이고 원래 target으로 복원해도 그 진행 중 작업을 성공으로 바꾸지 않습니다. accepted body 처리 실패 뒤 replay·다른 property·namespace·schema fallback을 하지 않습니다.

## Go의 고정 범위와 기존 typed Get

공개 Python getter에 `resource_type`·`value`·`namespace_name`을 추가하면 이미 바인딩한 `_get` 인자와 중복되어 **HTTP 전에 TypeError**입니다. 일반 unknown 속성으로 무시하거나 server query로 보내는 동작이 아닙니다. `requires_id`·`base_path`·`skip_cache`는 Python `_get`의 실제 제어값입니다.

Go 레코드 getter는 고정 child 경로·항상 HTTP 조회를 사용합니다. 위 충돌값과 경로·cache·session·version 제어값을 Attributes로 받지 않습니다. mutable Resource 재사용·cache·임의 route·parent lookup·query·pagination을 제공한다는 의미도 아닙니다. 이 선택과 Unicode/runtime·정밀도 차이를 포함한 Go mapping이며 Python 전체 Resource·session 동등성을 주장하지 않습니다.

기존 typed `scope.Get(ctx, name)`은 actual200·strict object를 유지하고 Python descriptor 변환 대신 실제 Body를 보존합니다. `WithGetResourceType`은 이 raw typed getter에 있는 별도 Glance query 기능입니다. 신규 GetRecord로 기존 raw 조회·생성·교체·삭제·dictionary 목록의 동작을 바꾸지 않습니다.

비교 기준은 고정 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [public getter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1900-L1931), [Property descriptor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_property.py#L21-L79), [Proxy getter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L822-L865), [response translation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338-L1398)입니다. 새 [레코드 조회 테스트](v2/metadefproperties/record_get_test.go)와 [Connection 테스트](../connection_image_metadef_property_record_test.go)는 기존 [leaf core fixture](v2/metadefproperties/core_test.go)·[HTTP 계약](v2/metadefproperties/contracts_test.go)·공통 descriptor 변환을 재사용합니다. 실제 실행 결과·예제 빌드·연산 판정은 [판정대장](../docs/sdk-support-ledger.md)에 검증 후 기록합니다.
