# Glance Metadef property 레코드 목록: Python과 Go

`NamespaceScope.ListRecords`는 namespace의 유한 property dictionary를 한 번 조회하고, SDK가 dictionary key의 name seed·descriptor 변환·로컬 Body 필터를 처리합니다. `AllRecords`는 같은 순회를 slice로 모으며 뒤 entry에서 오류가 나도 앞서 반환한 결과를 함께 남깁니다. 각 record는 declared Resource와 실제 Wire·전체 page·응답 receipt를 구분합니다.

Python의 `conn.image.metadef_properties(namespace, **query)`에 대응하는 owned Go mapping입니다. 고정 경로는 `/metadefs/namespaces/{namespace}/properties`이며 query·body 없는 GET입니다. 기존 strict typed `List`·`All`은 [leaf 가이드](v2/metadefproperties/README.md)의 raw 동작을 유지하고, 같은 descriptor를 사용하는 단건 `GetRecord`는 [조회 가이드](metadef-property-records.md)에 설명합니다.

## Python 사용

```python
import openstack

conn = openstack.connect(cloud="dev")
for value in conn.image.metadef_properties(
    "OS::Compute::Libvirt", min_length=0
):
    print(value.to_dict())
```

고정 `MetadefProperty.list`는 일반 `Resource.list`를 override하여 **한 번만 GET**합니다. `params={}`이므로 min_length 같은 조건을 HTTP query로 보내지 않습니다. 결과 Resource의 canonical Body 값으로 로컬 필터링하며 paginated 설정과 advertised next·Link로 다음 페이지를 요청하지 않습니다.

`max_items`는 이 override에서 **property의 최대 array 길이 descriptor에 대한 조건**입니다. 원본 row cap이나 페이지 limit이 아닙니다.

| 의도 | Python | Go |
|---|---|---|
| property의 최대 array 길이가3인 결과 | `max_items=3` | `WithRecordListFilter("max_items", 3)` |
| 처음3개 entry만 소비 | 이 override에 대응 제어값 없음 | `WithRecordListMaxItems(3)` |
| 누락 기본값을 포함해 min_length가0인 결과 | `min_length=0` | `WithRecordListFilter("min_length", 0)` |

## 독립 Go main

`-cloud`는 clouds.yaml 설정을, `-namespace`는 literal parent를 선택합니다. `-min-length`는 canonical min_length 로컬 필터이며 기본0입니다. `-max-rows=0`은 소비 제한 없음, 양수는 필터 전 원본 entry cap입니다. 예제는 partial 결과를 먼저 출력한 뒤 오류의 실제 receipt를 출력하고 종료합니다. 검증 범위는 컴파일과 로컬 HTTP 계약이며 인증된 cloud·Python 실행을 증명하지 않습니다.

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
    maxRows := flag.Int("max-rows", 0, "maximum consumed dictionary entries")
    minLength := flag.Int("min-length", 0, "local canonical min_length filter")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *namespace, *maxRows, *minLength); err != nil {
        log.Fatal(err)
    }
}

func printRecord(record *properties.Record) error {
    data, err := json.MarshalIndent(map[string]any{
        "key": record.Key, "namespace": record.Namespace,
        "resource": record.Resource, "wire": record.Wire,
        "envelope": string(record.Envelope),
        "header": record.Header, "status_code": record.StatusCode,
    }, "", "  ")
    if err != nil { return err }
    fmt.Println(string(data))
    return nil
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

func run(ctx context.Context, cloud, namespace string, maxRows, minLength int) error {
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    scope, err := service.API.MetadefProperties.InNamespace(ctx, namespace)
    if err != nil { return err }
    records, readErr := scope.AllRecords(ctx,
        properties.WithRecordListOpts(properties.RecordListOpts{MaxItems: maxRows}),
        properties.WithRecordListFilter("min_length", minLength),
        properties.WithRecordListHeader("X-Request-Source", "metadef-property-list-example"),
    )
    for _, record := range records {
        if err := printRecord(record); err != nil { return errors.Join(readErr, err) }
    }
    if readErr != nil { return errors.Join(readErr, printFailure(readErr)) }
    return nil
}
```

`ListRecords(ctx, options...)`는 `for record, err := range scope.ListRecords(...)`로 같은 결과를 순회합니다. iterator는 lazy이며 각 순회 시작 때 source·location·옵션을 capture합니다. callback은 그 순회에서 한 번 실행하고, 같은 iterator를 다시 순회하면 새 GET을 수행합니다. `break` 뒤 미소비 entry를 decode하거나 다음 요청을 만들지 않습니다. Provider의 retry·재인증은 원래 정책을 유지합니다.

## Dictionary key와 반환값

`Record.Key *string`은 outer dictionary의 literal key이며 list에서만 nonnil입니다. `GetRecord`의 Key는 nil입니다. key로 name을 먼저 seed한 뒤 실제 entry의 recognized Body 값이 이를 overlay합니다. 예를 들어 outer key가 `hw_cpu_policy`이고 entry의 name이 `policy`이면 Key는 원래 key, Resource의 name과 alternate id는 `policy`입니다. name이 누락하면 key를 유지하고 명시 null·array·object name도 passive 데이터로 유지합니다. explicit id가 있으면 null을 포함해 그 값이 alternate name보다 우선합니다.

key와 반환 identity는 후속 요청을 정하지 않습니다. slash·empty·긴 key를 요청용 literal 규칙으로 거부하지 않고, namespace만 `InNamespace`의 고정 parent 규칙으로 검증합니다. dictionary 순서를 보존하고 lexical sort를 추가하지 않습니다. 같은 JSON dictionary 키가 반복되면 parsed dictionary처럼 첫 위치와 마지막 값을 사용합니다.

Resource는 inherited id·18 Body·location·fixed namespace_name의21개 필드입니다. Python의 기본 to_dict는 URI를 제외한20개 필드입니다. 누락 min_length/min_items는0, require_unique_items는 false이고 명시 null은 유지합니다. int·bool·list·dict 변환과 wire/canonical response 별칭은 [레코드 조회의 descriptor 설명](metadef-property-records.md#resource-필드와-변환)과 같습니다.

Wire는 key를 name으로 seed하기 전의 **실제 entry object**이며 unknown·self·응답 location·namespace_name도 그대로 남습니다. Envelope는 outer properties dictionary와 next·vendor 같은 envelope 필드를 포함한 전체 physical page입니다. Header·StatusCode도 실제 응답입니다. Key·Resource·Wire·Envelope·Header와 다른 record 사이의 저장 공간은 독립입니다.

Connection location은 옵션·HTTP 전에 snapshot합니다. fixed namespace URI 때문에 seed·응답의 location이 declared Resource의 현재 location을 바꾸지 않습니다. `conn.Image(ctx).API.MetadefProperties`와 `conn.ImageV2(ctx).MetadefProperties` 양쪽이 같은 CurrentLocation dependency를 연결합니다. 직접 `properties.New(client)`를 사용하면 location은 null이며 concrete `NewWithDependencies`로 공급할 수 있습니다. Go의 명시 Zone은 유지하며 고정 Python current_location의 zone=None과 차이가 있습니다.

## 필터와 concrete 옵션

`RecordListOpts`는 Headers·Filters·MaxItems입니다. `WithRecordListOpts`는 전체 설정을 교체하고 Header(s)·Filter(s)·MaxItems helper는 해당 설정을 추가하거나 교체합니다. bulk Filters는 semantic 조건 집합을 교체하고 individual Filter는 같은 canonical 이름의 마지막 값이 우선합니다. map·slice·callback 설정은 복사하며 공통 JSON attribute capture를 재사용합니다.

필터는 다음19개 **canonical Body 이름만** 사용합니다.

| 그룹 | 이름 |
|---|---|
| untyped | `id`, `name`, `type`, `title`, `description`, `default`, `pattern` |
| list·dict | `operators`, `enum`, `items` |
| int | `minimum`, `maximum`, `min_length`, `max_length`, `min_items`, `max_items` |
| bool | `is_readonly`, `require_unique_items`, `allow_additional_items` |

`minLength`·`readonly`·`uniqueItems` 같은 wire spelling, 일반 unknown 이름, limit·marker는 무시하며 그 이름의 captured encoding 오류도 노출하지 않습니다. **응답의 wire 별칭은 descriptor로 인식하지만 필터 이름은 canonical이어야 합니다.** namespace_name·location은 Body 조건이 아닙니다.

고정 parent·client 정책에 따라 resource_type·namespace_name·namespace·paginated·base_path·list_base_path·jmespath_filters·requires_id·session·microversion·headers·allow_unknown_params·__conflicting_attrs는 Filters에 넣으면 오류입니다. Python의 실제 _list/class list 제어 인자와 Go의 fixed scope 선택은 구분합니다. request header는 named Header(s) 옵션으로 공급하고 인증·Accept·version·framing header는 SDK가 소유합니다.

필터는 **descriptor 변환을 마친 Resource**에 적용하며 caller의 filter 값은 coercion하지 않습니다. 응답 minLength가 문자열 `"2"`이면 Resource min_length는 숫자2이며 숫자2 필터는 일치하지만 문자열 `"2"` 필터는 일치하지 않습니다. 소비하는 entry의 전체 declared view를 필터 전에 만들므로 다른 조건이 제외할 row라도 descriptor 오류는 먼저 실패할 수 있습니다. 고정 Python도 cls.existing의 생성자가 to_dict를 materialize한 뒤 필터링합니다.

로컬 dictionary 조건은 nonempty 실제 object에 대한 recursive subset이고, scalar·array는 값 비교입니다. 공통 Go matcher는 JSON bool과 number를 구분하므로 Python의 `True == 1`과 다르고, 숫자는 exact decimal 값으로 비교하므로 Python float rounding과 차이가 있습니다. object 조건에 실제 nonobject가 있으면 mismatch로 처리하며 Python에서 일부 nested 값이 일으킬 수 있는 attribute 오류를 재현하지 않습니다. filter의 JSON snapshot도 Python object 참조 재사용과 다른 Go 소유권 정책입니다.

`MaxItems=0`은 cap 없음, 양수는 필터 전 처음 N개 entry 소비, 음수는 HTTP 전 오류입니다. 필터에 맞지 않는 entry도 cap을 사용합니다. cap 또는 break 이후의 entry shape·descriptor 오류는 검사하지 않지만 **전체 page의 UTF-8·JSON과 mandatory dictionary는 먼저 검증**합니다. 이 cap은 서버 limit hint·query·pagination을 합성하지 않습니다.

## 실제 응답과 검증 범위

actual HTTP200..399를 받은 뒤 whole-page valid UTF-8·nonnull JSON object와 **mandatory nonnull properties dictionary**를 요구합니다. 빈 dictionary는 성공이고 각 소비 entry는 nonnull object여야 합니다. 광고된 next·Link는 passive 응답 데이터입니다. 단건 GetRecord와 달리 빈 body·opaque·invalid JSON을 seed/default 성공으로 처리하지 않습니다.

accepted read·Close·UTF-8·JSON·shape·descriptor·filter·context·source 실패는 실제 Body·Header·StatusCode와 원인을 `*resource.ResponseError`에 보존합니다. rejected status는 native Gophercloud 오류이고 HTTP 전에 실패하면 receipt가 없습니다. `AllRecords`는 이미 일치해 반환한 record와 오류를 함께 남기며 빈 성공은 nonnil empty slice입니다. 관찰한 source/scope 실패는 복원 후에도 그 작업에서 유지하며 accepted body replay·다른 namespace·property·schema fallback을 하지 않습니다.

## 같은 scope의 레코드 삭제

`DeleteRecord(ctx, RecordRequest, ...DeleteOption)`와 `DeleteAllRecords(ctx, ...DeleteAllOption)`도 같은 scope에서 사용하며 실제 opaque ACK·receipt를 반환합니다. 개별 삭제의 기본 missing 처리에서 physical404는 actual StatusCode404의 ACK로 남습니다. 이 ACK는 서버404를 missing으로 처리한 증거이며 handling 실패에도 ACK와 오류를 함께 반환합니다. Python 공개 삭제의 None과 기존 raw Delete의 quiet404 nil보다 추가한 응답 증거입니다. 기존 Delete/DeleteAll 옵션과 strict204 메서드는 유지합니다. Python 대비 identity·missing·accepted status 정책과 삭제 예제는 [leaf 가이드](v2/metadefproperties/README.md)에 설명합니다. 위 독립 main은 목록 조회만 수행합니다.

비교 기준은 고정 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [공개 목록 proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1876-L1898), [Property list override](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_property.py#L86-L185), [Resource 생성자](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L538-L609)입니다. 새 [목록 테스트](v2/metadefproperties/list_records_test.go)와 [Connection 목록 테스트](../connection_image_metadef_property_list_records_test.go)는 기존 public Gophercloud helper·leaf/Connection HTTP/fault fixture·shared record projector·JSON filter·REST guard를 재사용합니다. 실제 검증 결과·독립 main 빌드·named API 판정은 [판정대장](../docs/sdk-support-ledger.md)에 실행 후 기록합니다. 고정 finite list의 Go mapping이며 전체 Python Resource/session·실제 cloud 권한·DB 검증을 주장하지 않습니다.
