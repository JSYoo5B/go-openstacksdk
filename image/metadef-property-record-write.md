# Glance Metadef property 레코드 생성·수정: Python과 Go

`NamespaceScope.CreateRecord`와 `UpdateRecord`는 명시한 Body 속성을 전송하고, 같은 property descriptor로 반환 Resource를 만듭니다. 요청에는 받은 원래 값을, 반환 view에는 기본값과 타입 변환을 적용합니다. 수정할 Body 속성이 없으면 PUT을 생략하고 로컬 snapshot을 반환합니다.

Python의 `conn.image.create_metadef_property(namespace, **attrs)`와 `update_metadef_property(property, namespace, **attrs)`에 대응하는 owned Go API입니다. 기존 Type·Title 필수·strict201/200 raw `Create`·`Update`는 [leaf 가이드](v2/metadefproperties/README.md)의 별도 동작을 유지합니다. 반환 descriptor와 단건 조회는 [GetRecord 가이드](metadef-property-records.md), 목록·삭제는 [목록 가이드](metadef-property-record-list.md)에 설명합니다.

## Python 사용

```python
import openstack

conn = openstack.connect(cloud="dev")
namespace = "OS::Compute::Libvirt"
id = "hw_cpu_policy"

created = conn.image.create_metadef_property(
    namespace, name=id, type="string", title="CPU Policy",
    description="CPU allocation policy", is_readonly=False, min_length=0,
)
updated = conn.image.update_metadef_property(
    id, namespace, name=id, type="string", title="CPU Policy",
    description="Updated CPU allocation policy", is_readonly=False, min_length=0,
)
local = conn.image.update_metadef_property(id, namespace)
print(created.to_dict(), updated.to_dict(), local.to_dict())
```

마지막 호출은 id만 가진 새 Resource에서 dirty id를 제외한 뒤 전송할 Body가 없어 HTTP를 보내지 않습니다. 공개 update는 Resource를 받아도 먼저 `_get_id`로 줄이므로 기존 객체의 다른 Body·dirty 상태를 이어받지 않습니다. Resource 인스턴스에 직접 `commit()`하는 흐름과 공개 proxy 호출은 구분해야 합니다.

## 독립 Go main

`-cloud`와 `-namespace`는 cloud 설정과 parent를 선택하고 `-id`는 property 이름입니다. 기본 `-action snapshot`은 `UpdateRecord`에 Body 속성을 주지 않아 로컬 snapshot을 출력합니다. `-action create`는 생성, `-action update`는 명시한 정의를 수정합니다. title·description은 flag로 바꿀 수 있습니다. 예제는 문자열 property의 name/type/title과 명시한 optional 값만 전송합니다.

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
    id := flag.String("id", "hw_cpu_policy", "property name")
    action := flag.String("action", "snapshot", "snapshot, create, or update")
    title := flag.String("title", "CPU Policy", "property title")
    description := flag.String("description", "CPU allocation policy", "property description")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *namespace, *id, *action, *title, *description); err != nil {
        log.Fatal(err)
    }
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

func run(ctx context.Context, cloud, namespace, id, action, title, description string) error {
    if action != "snapshot" && action != "create" && action != "update" {
        return fmt.Errorf("unknown action %q", action)
    }
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    scope, err := service.API.MetadefProperties.InNamespace(ctx, namespace)
    if err != nil { return err }
    attributes := map[string]any{
        "name": id, "type": "string", "title": title,
        "description": description, "is_readonly": false, "min_length": 0,
    }
    var record *properties.Record
    switch action {
    case "create":
        record, err = scope.CreateRecord(ctx,
            properties.WithRecordCreateAttributes(attributes),
            properties.WithRecordCreateHeader("X-Request-Source", "metadef-property-write-example"),
        )
    case "update":
        record, err = scope.UpdateRecord(ctx, properties.RecordRequest{ID: id},
            properties.WithRecordUpdateAttributes(attributes),
            properties.WithRecordUpdateHeader("X-Request-Source", "metadef-property-write-example"),
        )
    default:
        record, err = scope.UpdateRecord(ctx, properties.RecordRequest{ID: id},
            properties.WithRecordUpdateHeader("X-Request-Source", "metadef-property-write-example"),
        )
    }
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

예제는 `Connect`로 인증 설정을 읽고 서비스·scope를 준비합니다. snapshot 동작이 생략하는 것은 property PUT입니다. 검증 범위는 정확한 main의 컴파일과 로컬 HTTP 계약이며, 인증된 OpenStack·Python 실행 결과를 뜻하지 않습니다.

## 전송할 값과 반환 view

생성은 `/metadefs/namespaces/{namespace}/properties`에 flat JSON POST를 보냅니다. name/type/title 필수 조건을 SDK가 추가하지 않고, recognized Body 속성이 없으면 `{}`를 보냅니다. 최종 서버 schema·권한·저장 동작은 서버가 판단합니다.

수정은 선택한 child에 flat JSON PUT을 보냅니다. 전송하는 값은 **이번 호출에서 명시한 recognized Body 속성**입니다. 누락 Type·Title을 채우거나 기존 정의를 GET하여 병합하지 않습니다. 이 dirty-only payload는 PATCH를 뜻하지 않으며, 서버가 누락 필드를 유지한다고 보장하지 않습니다. 같은 name을 다시 지정하거나 null·false·0·빈 문자열·빈 list·빈 object를 지정해도 명시한 값은 dirty입니다.

| 명시한 속성 | JSON request의 wire 값 | Resource view |
|---|---|---|
| `min_length: 0` | `minLength: 0` | `min_length: 0` |
| `minimum: "03"` | `minimum: "03"` | `minimum: 3` |
| `is_readonly: "false"` | `readonly: "false"` | `is_readonly: true` |
| `operators: "and"` | `operators: "and"` | `operators: ["and"]` |
| `items: "opaque"` | `items: "opaque"` | `items: {}` |
| `min_length: null` | `minLength: null` | `min_length: null` |
| min_length 누락 | 해당 key 없음 | `min_length: 0` |

표는 SDK의 raw request와 descriptor view 차이를 설명합니다. 문자열 numeric/bool/list 입력이나 dict 대신 문자열을 서버가 허용한다는 뜻은 아닙니다. 독립 main은 해당 정의에 맞는 문자열·bool·숫자 값을 사용합니다. descriptor는 HTTP 전에 전체 view를 평가하므로 변환 실패를 서버 응답으로 숨기지 않습니다.

`WithRecordCreateAttribute(s)`는19개 canonical Body 이름과 wire 별칭을 받고, `WithRecordUpdateAttribute(s)`는 id를 제외한18개 Body 이름과 그 별칭을 받습니다. alias를 wire 이름으로 바꾸되 원래 JSON 값은 유지합니다. unknown 일반 속성은 전송하지 않으며 그 속성의 captured encoding 오류도 무시합니다. map에 canonical과 wire 별칭이 함께 있으면 null을 포함한 **canonical 값이 우선**합니다. Python dict는 입력 순서상 나중 alias가 우선하므로 이 선택은 Go의 명시적 차이입니다.

`RecordCreateOpts`·`RecordUpdateOpts`는 Headers와 Attributes carrier를 갖습니다. `WithRecordCreateOpts`·`WithRecordUpdateOpts`는 전체 설정을 교체하고 Header(s)·Attribute(s) helper는 해당 설정을 구성합니다. bulk Attributes는 속성 집합을 교체하고 individual Attribute는 해당 속성을 추가·교체합니다. map·slice·JSON bytes·callback 설정은 복사하며 옵션 callback은 호출마다 한 번 적용합니다. fixed scope/client 정책으로 resource_type·namespace_name·namespace·base_path·requires_id·session·microversion·headers·connection·_synchronized·__conflicting_attrs·resource_request_key·resource_response_key·resource_type_class·prepend_key·has_body·retry_on_conflict는 속성에서 거부합니다. update는 value·id도 거부하고 create의 value는 일반 unknown 속성처럼 무시합니다. 이 목록은 실제 Python keyword 충돌과 Go가 지원하지 않는 제어값을 함께 담습니다. 예를 들어 Python base_path는 정상 dynamic override이지만 Go는 고정 경로를 유지합니다.

## 생성 identity와 수정 identity

생성의 id/name은 Body 데이터입니다. 명시 null·array·object·slash·긴 문자열을 요청 child literal로 검사하지 않으며 POST는 collection에 고정됩니다. 생성 응답의 identity도 passive 반환값입니다. 이 값이 후속 조회·수정 경로를 자동 선택하지 않습니다.

수정의 `RecordRequest`는 ID 또는 Resource 중 하나를 받습니다. Resource를 옵션 callback 전에 clone하여 **identity만 선택**하고, 다른 입력 필드는 반환 seed나 dirty body로 사용하지 않습니다. id 키가 있으면 null을 포함해 그 키가 우선이고, id가 없을 때만 name을 사용합니다. 선택 결과는 nonempty valid UTF-8·최대80rune의 안전한 literal string이어야 합니다. slash·backslash·percent·query·fragment·ASCII control/DEL과 정확한 dot 경로를 거부하며 trim·UUID 검사·이름 검색을 하지 않습니다. namespace는 `InNamespace`에서 고정합니다.

모든 update 속성의 id는 거부합니다. 선택한 child는 요청 전부터 고정되며 name 속성은 PUT body의 rename 값입니다. 응답의 name/id와 caller의 retained Resource 변경은 그 경로를 바꾸지 않습니다. 고정 Source의 public update에서도 정상 문자열·Resource 입력은 `new(id=value, **attrs)`의 id keyword 충돌이 있습니다. Go는 모든 concrete 입력에 이 고정 identity 정책을 적용하며 no-op에도 literal을 검증합니다.

예를 들어 Resource에 `name="before", title="old"`만 있으면 update는 id `before`만 seed합니다. title은 null, name은 null인 새 view가 기본이며, 옵션에서 title/name을 지정하면 그 값만 추가됩니다. 이는 공개 Python proxy가 먼저 Resource를 ID로 줄이는 동작에 대응합니다. 직접 Resource의 mutable dirty/cache를 재사용하는 흐름을 Go에 암묵적으로 도입하지 않습니다.

## PUT을 생략한 결과와 실제 응답

recognized Body 속성이 없으면 수정은 HTTP를 보내지 않습니다. Header 옵션만 있어도 PUT을 강제하지 않습니다. unknown 일반 속성만 준 경우도 같으며 callback·header·source/context 검증은 수행합니다. 반환 Resource는 선택한 id와21개 declared 필드·현재 location·고정 namespace를 가진 독립 snapshot입니다. **Wire·Envelope·Header는 nil이고 StatusCode는0**이므로 서버 응답과 구분할 수 있습니다. 이 결과는 서버에 존재하는 정의를 조회하거나 변경했다는 증거가 아닙니다.

두 쓰기 API의 실제 응답은 actual200..399입니다. whole UTF-8을 검증하고 valid JSON object이면 present recognized fields를 요청 seed에 overlay합니다. unknown 응답 필드는 Wire에 남습니다. 빈 body·opaque·문법상 invalid JSON은 seed/default Resource를 유지하는 성공이며 Wire는 nil입니다. valid parsed nonobject나 descriptor 실패는 실제 receipt를 가진 오류입니다. 실제 HTTP가 있었다면 Envelope·Header·StatusCode는 그 응답을 보존하므로 Wire nil을 no-op으로 해석하면 안 됩니다.

Resource·Wire·Envelope·Header는 독립 소유이고 `Record.Key`는 nil입니다. read·Close·UTF-8·JSON shape·descriptor·context·source 실패는 실제 Body·Header·StatusCode와 원인을 `*resource.ResponseError`에 보존합니다. rejected status는 native Gophercloud 오류이며, callback·physical/read/Close 경계에서 관찰한 source/scope 실패는 복원 후에도 해당 작업에서 유지합니다. configured retry·재인증은 native 정책을 따르고 accepted body 처리 실패를 재전송하지 않습니다.

Connection current location은 옵션 전에 snapshot합니다. `conn.Image(ctx).API.MetadefProperties`와 `conn.ImageV2(ctx).MetadefProperties`가 같은 dependency를 연결합니다. fixed namespace URI 때문에 supplied/response location으로 declared Resource를 바꾸지 않습니다. 직접 `properties.New(client)`를 쓰면 location은 null이며 `NewWithDependencies`로 공급할 수 있습니다. Go 명시 Zone과 Python current_location의 zone=None 차이는 [조회 가이드](metadef-property-records.md#resource-필드와-변환)에 설명합니다.

비교 기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [공개 create/update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1790-L1841), [Proxy resource 선택·create/update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L563-L811), [constructor·attribute collector](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L538-L609), [dirty body와 request](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1231-L1336), [commit no-op](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881-L2017), [descriptor get/set](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L208-L291)입니다. 검증 결과와 named API 판정은 [판정대장](../docs/sdk-support-ledger.md)에 기록합니다. 전체 Python Resource/session·server schema·권한·DB·authenticated cloud parity를 주장하지 않습니다.

property 생성·수정은 Glance 기본 `metadef_admin` 정책으로 분류합니다. [서버 기본 정책과 구현 순서](../docs/glance-policy-priorities.md)를 참고하며 실제 배포의 권한은 서버가 판단합니다.
