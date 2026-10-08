# Glance resource type association 생성·삭제: Python과 Go

namespace scope의 `CreateRecord`는 명시한 association 속성을 POST하고 Source의 declared Resource와 실제 응답을 구분합니다. `DeleteRecord`는 고정 parent와 선택한 child에 DELETE를 보내며 실제 opaque acknowledgement를 남깁니다. 같은 scope의 목록은 [레코드 목록 가이드](metadef-resource-types-records.md), 기존 strict201/204 typed 생성·삭제는 [leaf 가이드](v2/metadefresourcetypes/README.md)에 설명합니다.

| Python `conn.image` | Go | 요청 |
|---|---|---|
| `create_metadef_resource_type_association(namespace, **attrs)` | `scope.CreateRecord(ctx, options...)` | namespace collection POST |
| `delete_metadef_resource_type_association(type, namespace, ignore_missing=True)` | `scope.DeleteRecord(ctx, RecordRequest, options...)` | namespace의 selected child DELETE |

SDK는 global resource type을 별도로 조회·생성·삭제하거나 image/volume/server metadata를 변경하는 요청을 만들지 않습니다. 서버의 association 정책과 저장 결과는 서버가 판단합니다.

## Python 사용

```python
import openstack

conn = openstack.connect(cloud="dev")
namespace = "OS::Compute::Libvirt"
resource_type = "OS::Nova::Server"

created = conn.image.create_metadef_resource_type_association(
    namespace, name=resource_type, prefix="hw:", properties_target="metadata",
)
print(created.to_dict())
conn.image.delete_metadef_resource_type_association(
    resource_type, namespace, ignore_missing=False,
)
```

생성은 새 Resource의 dirty Body만 flat POST에 넣습니다. name의 SDK 필수 조건은 없으며 빈 입력도 `{}`를 보냅니다. delete의 공개 반환값은 None입니다. Resource를 child로 넘기면 `_get_resource`가 그 객체를 재사용하고 namespace URI를 update할 수 있습니다. 이는 공개 API가 먼저 child를 ID로 줄이는 property 삭제와 다른 Source 경로입니다.

## 독립 Go main

기본 `-action list`는 namespace의 association 목록을 출력합니다. `-action create`는 name·prefix·properties_target을 전송하고 `-action delete`는 `-resource-type`으로 지정한 child를 삭제합니다. `-strict-missing`을 주면 삭제404를 오류로 처리합니다. 생성 예제는 string name과 colon으로 끝나는 prefix를 사용하고 서버의 read-only date/id 필드를 추가하지 않습니다.

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
    types "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes"
    "github.com/JSYoo5B/go-openstacksdk/resource"
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    namespace := flag.String("namespace", "OS::Compute::Libvirt", "literal namespace")
    resourceType := flag.String("resource-type", "OS::Nova::Server", "resource type name")
    action := flag.String("action", "list", "list, create, or delete")
    prefix := flag.String("prefix", "hw:", "association property prefix")
    target := flag.String("properties-target", "metadata", "association target")
    strictMissing := flag.Bool("strict-missing", false, "return an error for deletion404")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *namespace, *resourceType, *action, *prefix, *target, *strictMissing); err != nil {
        log.Fatal(err)
    }
}

func printJSON(value any) error {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { return err }
    fmt.Println(string(data))
    return nil
}

func printRecord(record *types.Record) error {
    return printJSON(map[string]any{
        "namespace": record.Namespace,
        "resource": record.Resource, "wire": record.Wire,
        "envelope": string(record.Envelope),
        "header": record.Header, "status_code": record.StatusCode,
    })
}

func printFailure(err error) error {
    var accepted *resource.ResponseError
    var unexpected gophercloud.ErrUnexpectedResponseCode
    switch {
    case errors.As(err, &accepted):
        return printJSON(map[string]any{
            "status_code": accepted.StatusCode, "header": accepted.Header,
            "body_base64": accepted.Body,
        })
    case errors.As(err, &unexpected):
        return printJSON(map[string]any{
            "status_code": unexpected.Actual, "header": unexpected.ResponseHeader,
            "body_base64": unexpected.Body,
        })
    default:
        return nil
    }
}

func run(ctx context.Context, cloud, namespace, resourceType, action, prefix, target string, strictMissing bool) error {
    if action != "list" && action != "create" && action != "delete" {
        return fmt.Errorf("unknown action %q", action)
    }
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    scope, err := service.API.MetadefResourceTypes.InNamespace(ctx, namespace)
    if err != nil { return err }
    switch action {
    case "create":
        record, createErr := scope.CreateRecord(ctx,
            types.WithRecordCreateAttributes(map[string]any{
                "name": resourceType, "prefix": prefix, "properties_target": target,
            }),
            types.WithRecordCreateHeader("X-Request-Source", "association-example"),
        )
        if createErr != nil { return errors.Join(createErr, printFailure(createErr)) }
        return printRecord(record)
    case "delete":
        ack, deleteErr := scope.DeleteRecord(ctx, types.RecordRequest{ID: resourceType},
            types.WithDeleteIgnoreMissing(!strictMissing),
            types.WithDeleteHeader("X-Request-Source", "association-example"),
        )
        if ack != nil {
            if err := printJSON(ack); err != nil { return errors.Join(deleteErr, err) }
        }
        if deleteErr != nil { return errors.Join(deleteErr, printFailure(deleteErr)) }
        return nil
    default:
        records, readErr := scope.AllRecords(ctx,
            types.WithRecordListPaginated(false),
            types.WithRecordListHeader("X-Request-Source", "association-example"),
        )
        for _, record := range records {
            if err := printRecord(record); err != nil { return errors.Join(readErr, err) }
        }
        if readErr != nil { return errors.Join(readErr, printFailure(readErr)) }
        return nil
    }
}
```

acknowledgement와 오류 receipt의 byte body는 `encoding/json`의 base64 표현으로 출력하여 비JSON·비UTF-8 응답도 남깁니다. 예제의 검증 범위는 정확한 main의 컴파일과 로컬 HTTP 계약입니다. 실제 인증·OpenStack·Python 실행을 증명하지 않습니다.

## 생성 Body와 반환 Resource

`RecordCreateOpts`는 Headers와 Attributes carrier를 가집니다. `WithRecordCreateOpts`는 전체 설정을 교체하고 Header(s)·Attribute(s) helper는 해당 설정을 구성합니다. bulk Attributes는 속성 집합을 교체하고 individual Attribute는 해당 key를 추가·교체합니다. map·slice·JSON bytes와 callback 설정은 복사하며 callback은 호출마다 한 번 실행합니다.

생성에서 recognized Body는 **id·name·created_at·updated_at·prefix·properties_target의6개**입니다. 모두 Source의 untyped descriptor이므로 null·숫자·array·object도 raw JSON 값으로 유지하며 문자열 coercion이나 날짜 parse를 하지 않습니다. 명시한 값만 body에 보내고 누락 필드를 합성하지 않습니다. name/id가 slash·빈 값·긴 문자열이어도 생성의 child 경로로 검사하지 않으며 POST는 collection에 고정됩니다. 서버 schema가 이 값을 허용한다는 보장은 별도입니다.

일반 unknown 속성은 전송하지 않으며 그 key의 captured encoding 오류도 노출하지 않습니다. location은 computed, namespace_name은 fixed URI이므로 Body에 들어가지 않습니다. 고정 scope/client 정책으로 resource_type·namespace_name·namespace·base_path·requires_id·session·microversion·headers·connection·_synchronized·__conflicting_attrs·resource_request_key·resource_response_key·resource_type_class·prepend_key·has_body·retry_on_conflict는 속성에서 거부합니다. Source의 namespace_name/resource_type/connection/_synchronized binding 충돌과 Go가 지원하지 않는 제어값을 구분해야 합니다. 예를 들어 Source base_path는 정상 dynamic override이고 Go는 고정 경로를 유지합니다.

반환 Resource는6개 Body·location·namespace_name의 **8필드**입니다. Source 기본 `to_dict()`는 URI를 제외한7필드입니다. 누락 Body 값은 null이고 id 키가 없을 때만 name을 alternate ID로 사용합니다. 명시 id null은 name으로 대체하지 않습니다. 생성 응답에 recognized 값이 있으면 요청 seed에 overlay하고, 누락 값은 명시한 seed를 유지합니다. unknown·self·응답 namespace/location은 실제 Wire에 남습니다.

`Record.Namespace`와 Resource namespace_name은 scope parent입니다. `conn.Image(ctx).API.MetadefResourceTypes`와 `conn.ImageV2(ctx).MetadefResourceTypes`는 생성에서 같은 CurrentLocation dependency를 연결합니다. location은 옵션 전에 snapshot하고 fixed URI Source collector 규칙에 따라 응답 location으로 덮어쓰지 않습니다. 직접 `types.New(client)`를 쓰면 null이며 `NewWithDependencies`로 공급할 수 있습니다. Go 명시 Zone과 고정 Python의 zone=None 차이는 [목록의 location 설명](metadef-resource-types-records.md#결과와-location)을 참고합니다.

## 삭제 identity와 실제 acknowledgement

`RecordRequest`는 `ID string` 또는 `Resource *resource.RawResource` 중 하나입니다. Resource는 옵션 전에 clone하여 identity만 선택합니다. id 키가 있으면 null을 포함해 우선하고, 없을 때만 name을 사용합니다. 선택한 child와 scope parent는 nonempty valid UTF-8·최대80rune의 안전한 literal이어야 합니다. slash·backslash·percent·query·fragment·ASCII control/DEL과 정확한 dot 경로를 거부하며 trim·UUID 검사·Name 검색을 하지 않습니다. 다른 Body·입력 namespace·location·receipt는 삭제 경로나 요청 body를 결정하지 않습니다.

Python association delete는 받은 Resource를 재사용해 parent를 바꿀 수 있지만 Go는 caller 객체를 바꾸지 않고 고정 parent를 유지합니다. 이 immutable literal carrier는 명시적인 Go 차이입니다. source/self/returned name을 따라 다른 resource나 namespace를 선택하지 않습니다.

`DeleteRecord`는 기존 `WithDeleteOpts`·Header(s)·IgnoreMissing 옵션을 재사용합니다. 실제200..399의 body는 JSON·UTF-8을 요구하지 않는 opaque bytes이며 선택한 Namespace·Name과 실제 Body·Header·StatusCode를 `Acknowledgement`로 남깁니다. read·Close·context·source 처리 실패에도 실제 ACK와 오류를 함께 반환하며 accepted body를 replay하지 않습니다.

IgnoreMissing의 기본 nil/true는 직접 받은 physical404를 native404 retry callback 없이 처리하고 **StatusCode404 ACK**를 반환합니다. 이 응답은 missing 처리의 증거이며 삭제 성공이나 실제 부재의 증명이 아닙니다. handling 실패에도404 ACK와 실제 `ResponseError`를 함께 남깁니다. explicit false는 native404 status/body/header/retry 오류를 유지하고 transport가 주장하는404를 physical missing으로 바꾸지 않습니다. Python 공개 반환값 None과 기존 strict204 raw Delete의 clean404 `(nil,nil)`보다 실제 응답 증거를 추가합니다.

## 응답과 검증 경계

생성은 actual200..399의 whole UTF-8을 검증합니다. valid parsed JSON object는 Wire와 declared Resource로 나누고, 빈 body·opaque·문법상 invalid JSON은 요청 seed/default Resource와 Wire nil을 유지하는 성공입니다. Envelope·Header·StatusCode는 실제 응답을 보존합니다. valid parsed nonobject·read/Close/shape/source/context 실패는 실제 receipt와 원인을 가진 오류입니다. 기존 raw Create의 strict201·typed string 응답과 구분해서 사용합니다.

Resource·Wire·Envelope·Header와 ACK body/header는 독립 소유입니다. source/context/outer guard는 callback 전후와 physical/read/Close 경계를 검사하며 관찰한 source/scope 실패는 복원 후에도 해당 작업에서 유지합니다. configured native retry·재인증은 원래 정책을 따르고 accepted handling 오류로 응답을 재전송하지 않습니다. 실제 permission/protection/DB 동작이나 global type side effect를 로컬 fixture로 검증했다고 주장하지 않습니다.

고정 openstacksdk의 두 공개 메서드와 Resource class에 admin 전용 분기는 없습니다. 이 사실은 서버 권한을 보장하지 않으며 실제 서버 policy는 별도로 판단합니다. 비교 기준은 `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [공개 association create/delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1711-L1761), [association class](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_resource_type.py#L34-L61), [Proxy Resource 선택](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L563-L609), [dirty request](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1231-L1336), [Response translation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338-L1398), [Delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2094-L2154)입니다. 실제 실행 근거와 named API 판정은 [판정대장](../docs/sdk-support-ledger.md)에 기록합니다. 전체 Python Resource/session parity는 별도 목표입니다.
