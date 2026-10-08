# Glance schema record: Python과 Go

`conn.Image(ctx)`의 `GetSchemaRecord`는 고정한 schema 경로 하나를 조회하고, Source class에 맞춘 `Resource`와 실제 응답의 `Wire`·receipt를 구분합니다. ordinary schema와 metadata definition schema의 nullable·dict·bool·list 변환을 SDK가 처리합니다. 기존 16개 strict getter는 [기존 Schema 가이드](schemas.md)의 API로 유지합니다.

Python의 복수형 getter도 schema 문서 하나를 반환합니다. 아래 `SchemaKind`는 route 선택값이며 이미지·task·namespace의 ID나 Name이 아닙니다. 각 호출은 query·body 없이 고정 경로를 GET으로 조회합니다. Provider의 retry·재인증 정책은 별도로 적용됩니다.

| `conn.image`의 Python getter | Go `image.SchemaKind` | 고정 경로 | Source class |
|---|---|---|---|
| `get_image_schema()` | `SchemaImage` | `/schemas/image` | `Schema` |
| `get_images_schema()` | `SchemaImages` | `/schemas/images` | `Schema` |
| `get_member_schema()` | `SchemaMember` | `/schemas/member` | `Schema` |
| `get_members_schema()` | `SchemaMembers` | `/schemas/members` | `Schema` |
| `get_task_schema()` | `SchemaTask` | `/schemas/task` | `Schema` |
| `get_tasks_schema()` | `SchemaTasks` | `/schemas/tasks` | `Schema` |
| `get_metadef_namespace_schema()` | `SchemaMetadefNamespace` | `/schemas/metadefs/namespace` | `MetadefSchema` |
| `get_metadef_namespaces_schema()` | `SchemaMetadefNamespaces` | `/schemas/metadefs/namespaces` | `MetadefSchema` |
| `get_metadef_object_schema()` | `SchemaMetadefObject` | `/schemas/metadefs/object` | `MetadefSchema` |
| `get_metadef_objects_schema()` | `SchemaMetadefObjects` | `/schemas/metadefs/objects` | `MetadefSchema` |
| `get_metadef_property_schema()` | `SchemaMetadefProperty` | `/schemas/metadefs/property` | `MetadefSchema` |
| `get_metadef_properties_schema()` | `SchemaMetadefProperties` | `/schemas/metadefs/properties` | `MetadefSchema` |
| `get_metadef_resource_type_schema()` | `SchemaMetadefResourceType` | `/schemas/metadefs/resource_type` | `MetadefSchema` |
| `get_metadef_resource_types_schema()` | `SchemaMetadefResourceTypes` | `/schemas/metadefs/resource_types` | `MetadefSchema` |
| `get_metadef_tag_schema()` | `SchemaMetadefTag` | `/schemas/metadefs/tag` | `MetadefSchema` |
| `get_metadef_tags_schema()` | `SchemaMetadefTags` | `/schemas/metadefs/tags` | `MetadefSchema` |

Python:

```python
import openstack

conn = openstack.connect(cloud="dev")
namespace_schema = conn.image.get_metadef_namespace_schema()
task_schema = conn.image.get_task_schema()
print(namespace_schema.to_dict())
print(task_schema.to_dict())
```

## 독립 Go main

`-cloud`로 clouds.yaml의 cloud를 선택하고 metadata namespace·task 두 schema를 순서대로 조회합니다. 두 호출은 독립 작업이며 record의 raw 값과 실제 응답을 출력합니다. 이 예제의 검증 범위는 컴파일이며 인증된 OpenStack/Python 실행은 별도입니다.

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
    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    cloudName := flag.String("cloud", "dev", "clouds.yaml cloud name")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloudName); err != nil {
        log.Fatal(err)
    }
}

func printRecord(record *image.SchemaRecord) error {
    if record == nil { return nil }
    data, err := json.MarshalIndent(map[string]any{
        "kind": record.Kind, "resource": record.Resource, "wire": record.Wire,
        "envelope": string(record.Envelope),
        "header": record.Header, "status_code": record.StatusCode,
    }, "", "  ")
    if err != nil { return err }
    fmt.Println(string(data))
    return nil
}

func printFailure(err error) error {
    var data map[string]any
    var accepted *resource.ResponseError
    var unexpected gophercloud.ErrUnexpectedResponseCode
    switch {
    case errors.As(err, &accepted):
        data = map[string]any{
            "status_code": accepted.StatusCode, "header": accepted.Header,
            "body": string(accepted.Body),
        }
    case errors.As(err, &unexpected):
        data = map[string]any{
            "status_code": unexpected.Actual, "header": unexpected.ResponseHeader,
            "body": string(unexpected.Body),
        }
    default:
        return nil
    }
    raw, printErr := json.MarshalIndent(data, "", "  ")
    if printErr != nil { return printErr }
    fmt.Println(string(raw))
    return nil
}

func run(ctx context.Context, cloudName string) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    options := []image.GetSchemaOption{
        image.WithGetSchemaHeader("X-Request-Source", "schema-record-example"),
    }
    for _, kind := range []image.SchemaKind{image.SchemaMetadefNamespace, image.SchemaTask} {
        record, readErr := service.GetSchemaRecord(ctx, kind, options...)
        if printErr := printRecord(record); printErr != nil {
            return errors.Join(readErr, printErr)
        }
        if readErr != nil { return errors.Join(readErr, printFailure(readErr)) }
    }
    return nil
}
```

Envelope는 accepted opaque·불완전 JSON도 담을 수 있어 문자열로 출력합니다. 성공이면 record를 출력합니다. physical·shape·source·context 오류에서는 record가 nil이며, 예제는 오류 체인의 `*resource.ResponseError` 또는 native HTTP 오류에 실제 receipt가 있을 때 이를 출력한 뒤 오류를 반환합니다. HTTP 전에 발생한 오류에는 응답 receipt가 없습니다. `$ref`나 응답 URI를 조회하는 추가 요청은 하지 않습니다.

## Source class별 기본값과 변환

`SchemaRecord`는 `Kind`, `Resource`, `Wire`, `Envelope`, `Header`, `StatusCode`를 갖습니다. `Resource`는 Source descriptor에 맞춘 owned view이고 `Wire`는 실제 root JSON object 전체입니다. root 자체를 소비하며 `schema` 같은 envelope를 unwrap하지 않습니다.

| class | Resource의 필드 | nonnull 값 처리 |
|---|---|---|
| ordinary `Schema` 6종 | `id`, `name`, `additional_properties`, `properties`, `location` | id/name raw; 두 dict 필드는 object 유지·비객체는 `{}` |
| `MetadefSchema` 10종 | 위 필드와 `definitions`, `required` | additional_properties는 Python truthiness bool; properties/definitions는 object 유지·비객체는 `{}`; required는 array 유지·비배열은 `[value]` |

**모든 누락·명시 null 필드는 null입니다.** dict/list 기본값을 임의로 `{}`·`[]`로 만들지 않습니다. ordinary schema의 definitions·required는 Source class에 선언되지 않아 Resource에서 제외하고 Wire에 보존합니다. id/name도 passive JSON이므로 숫자·배열을 문자열 ID로 바꾸거나 요청 identity로 검증하지 않습니다.

예를 들어 metadata `required:[1,null,{}]`는 그대로이고 `required:"name"`은 `["name"]`입니다. metadata `additionalProperties:"false"`는 nonempty 문자열의 Python truthiness에 따라 true이며, ordinary의 같은 nonobject 값은 `{}`가 됩니다. null은 두 class 모두 null입니다. Go의 bool 변환은 JSON 숫자의 정확한 0 여부를 사용하므로 `1e-400`도 true입니다. Python JSON backend가 binary float로 변환해 0으로 underflow시키는 경우와는 다릅니다. 큰 정수와 nested vendor 값은 raw 정밀도를 유지합니다.

`additional_properties`와 wire 이름 `additionalProperties`가 함께 있으면 Python dictionary 순서처럼 뒤의 인식된 member가 선택됩니다. 같은 JSON 키가 반복되면 첫 등장 위치를 유지하면서 마지막 값을 사용한 뒤 별칭 순서를 평가합니다. 이 선택 뒤 class별 변환을 적용합니다. `self`, `$ref`, unknown 키와 응답의 location은 Wire에 남지만 declared view를 확장하거나 Connection location을 덮어쓰지 않습니다.

## location·옵션·실제 응답

Connection이 제공하는 Service는 옵션·HTTP 전에 현재 location을 한 번 snapshot하여 Resource에 넣고 응답 필드로 다시 계산하지 않습니다. 고정 Python의 `current_location`은 zone=None으로 생성합니다. Go는 명시한 `WithCloudLocation`의 Zone 또는 직접 공급한 Zone을 보존하는 추가 동작을 제공합니다. 직접 생성할 때는 `image.NewWithDependencies(client, image.Dependencies{CloudLocation: getter})`로 concrete dependency를 공급할 수 있고, 기존 `image.New(client)`는 계속 사용하며 location은 null입니다. view·Wire·Envelope·header를 독립 복사하므로 한 채널의 caller 수정이 다른 채널이나 원래 transport 응답을 바꾸지 않습니다.

기존 `GetSchemaOption`을 그대로 사용합니다. `WithGetSchemaOpts`는 일반 header 설정을 교체하고 `WithGetSchemaHeader`·`WithGetSchemaHeaders`는 값을 추가합니다. 옵션 map/slice와 callback carrier를 복사하고 한 호출에 한 번 적용합니다. 인증·version·representation·framing header는 SDK가 소유하며 nil/error 옵션·보호된 header 충돌·알 수 없는 kind는 HTTP 전에 오류입니다. query·body·Resource/Ref 입력·임의 URI·pagination·CRUD 옵션은 없습니다.

HTTP200..399에서는 raw receipt를 확보한 뒤 응답을 읽습니다. valid object이면 Content-Type과 관계없이 Wire와 class view를 구성합니다. 빈 body·opaque·문법적으로 잘못된 JSON은 고정 Python의 JSON ValueError 처리처럼 bare defaults와 computed location을 반환하고 Wire는 nil입니다. Envelope/Header/StatusCode는 이때도 실제 응답을 보존합니다.

**parsed null·array·string·number·bool root는 오류입니다.** invalid UTF8와 실제 transport/read/Close/context/source 실패도 bare 성공으로 바꾸지 않습니다. 오류 시 record는 nil이며, accepted 응답을 확보했다면 `*resource.ResponseError`가 실제 body/header/status와 원인을 보존합니다. HTTP401/403/404는 native HTTP 오류로 남으며 빈 성공이나 다른 경로로 전환하지 않습니다. Source의 requests JSON backend에 따른 encoding·비유한 숫자 corner까지 Go가 동일하게 실행한다고 주장하지 않습니다.

## 기존 strict getter와 검증 범위

`GetImageSchema`·`GetTaskSchema`·`GetMetadefNamespaceSchema` 등 기존 16개 getter는 actual200만 받고 기존 `*image.Schema`의 nullable string Name·object Properties/Definitions·`[]string` Required·raw AdditionalProperties를 유지합니다. 이 stable typed 대안은 Source class별 dict/bool/list coercion과 다르며 `GetSchemaRecord`의 accepted JSON tolerance로 동작을 바꾸지 않습니다. [기존 가이드](schemas.md)에 strict 정책과 전체 raw Body 사용법을 설명합니다.

새 record 검증은 [class/응답 테스트](schema_records_test.go)와 [Connection location 테스트](../connection_image_schema_records_test.go)를 기존 [16경로 core](schemas_core_test.go)·[HTTP 계약](schemas_contracts_test.go)·[옵션](schemas_options_test.go)·[Connection](../connection_image_schemas_test.go)·공통 REST/source 검증에 연결합니다. 공개 Gophercloud testhelper와 기존 SDK HTTP/fault fixture를 재사용하며 서비스별 새로운 응답 엔진이나 동일 fault matrix를 만들지 않습니다. 실제 통과 결과·예제 빌드·연산별 판정은 [판정대장](../docs/sdk-support-ledger.md)에 검증 후 기록합니다.

고정 소스의 [16개 getter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1951-L2160), [Schema](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/schema.py#L17-L26), [MetadefSchema](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_schema.py#L17-L30)가 비교 기준입니다. Schema 문서의 링크·이름·dialect는 데이터이며 자동 follow·schema validation·Resource lifecycle/session을 제공한다는 뜻은 아닙니다. 해당 공통 범위는 전체 SDK 목표에서 별도로 추적합니다.
