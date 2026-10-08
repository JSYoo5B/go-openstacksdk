# Glance 공유 멤버 레코드 목록·검색: Python과 Go

`conn.Image(ctx)`의 `ListImageMemberRecords`, `AllImageMemberRecords`, `FindImageMemberRecord`는 고정 openstacksdk의 Member 속성, 기본 목록 순회와 GET 이후 목록 검색을 SDK가 처리합니다. 부모 이미지를 명시하고 concrete `With…` 옵션을 조합하면 됩니다. 반환하는 `ImageMemberRecord`는 declared `Resource`와 실제 `Wire`, 응답 증거를 구분합니다.

| Python `conn.image` | Go `image.Service` | 요청 |
|---|---|---|
| `members(image)` | `ListImageMemberRecords(ctx, parent, options...)` | `GET images/{image_id}/members`와 광고된 다음 페이지 |
| `list(members(image))` | `AllImageMemberRecords(ctx, parent, options...)` | 같은 순회를 수집하며 후속 오류에서 앞선 행을 보존 |
| `find_member(name_or_id, image, ignore_missing=True)` | `FindImageMemberRecord(ctx, parent, nameOrID, options...)` | 직접 GET; 최종 native400·403·404이면 LIST에서 ID/name과 중복을 검사 |

기존 `ListImageMembers/AllImageMembers`의 strict200·유한 배열 목록과 `FindImageMember`의 literal ID 직접 GET 계약은 유지합니다. 기존 mutation·typed getter와 `API.Members`는 [멤버 사용법](members.md)을 참고하세요.

## Python 사용

```python
import openstack

conn = openstack.connect(cloud="dev")
image_id = "image-id"

for member in conn.image.members(image_id):
    print(member.to_dict())

member = conn.image.find_member("project-id", image_id, ignore_missing=False)
print(member.to_dict())
```

고정 [proxy 소스](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1289-L1354)의 `members(image, **query)`는 **query를 전부 버립니다**. `limit`, `marker`, `max_items`, `paginated`, 속성 필터나 `headers`를 그 공개 proxy에 전달해도 이 pin에서는 적용되지 않습니다. 기본 Resource 목록의 pagination은 true이므로 응답이 다음 페이지를 광고하면 순회합니다. Go의 무옵션 목록은 같은 query 없는 기본 호출을 제공하고, 아래 목록 제어·헤더·필터는 명시적인 Go 확장입니다.

`find_member`의 기본 `ignore_missing=True`는 향후 SDK 6.0 기본 변경에 관한 warning을 발생시킵니다. 예제는 false를 명시했습니다. Go의 nil 옵션도 true지만 Python warning을 재현하지 않습니다.

## 독립 Go main

아래 예제는 [외부 모듈 설치 안내](../docs/install.md)를 따라 `main.go`로 저장해 빌드할 수 있습니다. 실행하려면 해당 `clouds.yaml` 설정과 실제 이미지 ID를 준비하고 `go run . -cloud dev -image image-id -find project-id`처럼 전달합니다. 기본 목록은 query를 보내지 않으며 양수 limit/max-items나 status를 지정할 때만 해당 Go 확장을 사용합니다.

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

    "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    imageID := flag.String("image", "", "literal parent image ID")
    find := flag.String("find", "", "optional member ID or inherited name")
    status := flag.String("status", "", "optional local member status filter")
    limit := flag.Int("limit", 0, "optional server page limit")
    maxItems := flag.Int("max-items", 0, "maximum consumed rows before filtering")
    flag.Parse()
    if *imageID == "" { log.Fatal("-image is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, *find, *status, *limit, *maxItems); err != nil {
        var response *resource.ResponseError
        if errors.As(err, &response) {
            fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
        }
        log.Fatal(err)
    }
}

func printRecord(record *image.ImageMemberRecord) error {
    value := map[string]any{"resource": record.Resource, "status_code": record.StatusCode}
    body, err := json.MarshalIndent(value, "", "  ")
    if err != nil { return err }
    fmt.Println(string(body))
    return nil
}

func run(ctx context.Context, cloud, imageID, find, status string, limit, maxItems int) error {
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    parent := resource.ID(imageID)
    options := []image.ImageMemberRecordListOption{
        image.WithImageMemberRecordListOpts(image.ImageMemberRecordListOpts{Limit: limit, MaxItems: maxItems}),
        image.WithImageMemberRecordListHeader("X-Request-Source", "member-records-example"),
    }
    if status != "" {
        options = append(options, image.WithImageMemberRecordListFilter("status", status))
    }
    for record, readErr := range service.ListImageMemberRecords(ctx, parent, options...) {
        if readErr != nil { return readErr }
        if err := printRecord(record); err != nil { return err }
    }
    if find != "" {
        record, err := service.FindImageMemberRecord(ctx, parent, find,
            image.WithFindImageMemberRecordIgnoreMissing(false))
        if err != nil { return err }
        return printRecord(record)
    }
    return nil
}
```

`resource.ID`는 lookup 없이 부모를 고정합니다. 명시적인 `resource.Name("image-name")`은 기존 image 이름 resolver를 사용하는 Go 확장입니다. 멤버의 검색 문자열은 ID와 inherited name으로 비교하며 프로젝트 이름을 별도 서비스에서 조회하지 않습니다. Find의 검색 문자열은 nonblank valid UTF-8이며 control 문자를 거부합니다. 원래 문자열의 공백·colon·slash·backslash 등을 하나의 escaped URI segment로 보존하며 `.`·`..` 같은 경로 입력은 거부합니다. 부모 ID는 기존 literal ID 규칙을 사용합니다. trim·대소문자 정규화나 UUID 제한은 하지 않습니다.

## 옵션과 반환값

`ImageMemberRecordListOpts`는 `Headers map[string]string`, `Limit int`, `Marker string`, `MaxItems int`, `Paginated *bool`, `Filters map[string]json.RawMessage`를 제공합니다. `WithImageMemberRecordListOpts`는 전체 설정을 교체하고 `Header/Headers`, `Limit`, `Marker`, `MaxItems`, `Paginated`, `Filter/Filters` helper는 순서대로 값을 추가·교체합니다. callback·map·pointer·raw bytes를 각 순회에서 snapshot하고 callback은 한 번 적용합니다. nil/error callback, 음수 limit/cap, 잘못된 header·선택한 Body 필터의 JSON은 요청 전에 오류입니다. unknown semantic 필터는 encoding 전에 무시하며 collection·parent·URL을 변경하는 옵션은 제공하지 않습니다.

- Limit0과 빈 Marker는 생략합니다. 양수 Limit·Marker는 서버 query이며 이 Member 공개 proxy에는 없는 Go 확장입니다.
- `Paginated=nil`은 true이고 false는 첫 페이지까지만 읽습니다. MaxItems0은 무제한입니다.
- 양수 MaxItems는 **필터 전 소비한 원본 행 수**를 제한하며 explicit limit이 없으면 첫 요청에 limit hint를 보냅니다. 따라서 필터가 모두 실패하면 cap에 도달한 뒤 빈 결과로 끝날 수 있습니다.
- 로컬 필터는 declared Body의 `id`, `name`, `member_id`, `created_at`, `status`, `schema`, `updated_at`를 비교합니다. `member` spelling은 member_id에 대한 Go 편의 별칭이며 두 spelling을 충돌하게 지정하면 오류입니다. 부모 image_id·location은 목록 필터가 아닙니다.

필터는 raw JSON 값을 보존하고 object는 recursive subset, scalar·array는 값 비교를 수행합니다. missing 속성은 null로 비교하고 bool과 number를 구분하며 숫자는 exact decimal로 비교합니다. Python의 bool/number equality·binary float rounding·일부 nested filter 예외와 차이가 있습니다. 날짜와 status를 입력 enum으로 변환하지 않습니다.

`ListImageMemberRecords`는 lazy이며 iterator를 다시 순회하면 별도 capture·옵션 적용·HTTP를 수행합니다. `AllImageMemberRecords`는 nonnil empty slice로 시작하고 오류가 나도 이미 성공한 행을 함께 반환합니다. 이 수집 정책은 partial slice를 버리는 기존 `AllImageMembers`와 다릅니다. 두 API를 각각 호출하면 독립 요청입니다.

`FindImageMemberRecordOpts`의 `Headers`와 `IgnoreMissing *bool`은 `WithFindImageMemberRecordOpts/Header/Headers/IgnoreMissing`으로 설정합니다. nil IgnoreMissing은 true입니다. 실제 목록 검색을 마친 뒤 일치 행이 없으면 true는 `nil, nil`, false는 `resource.ErrNotFound`를 반환합니다. 오류에는 nil record를 반환하며 ambiguous는 `resource.ErrAmbiguous`로 확인합니다. 기본 missing 처리는 부모 Name resolver의 실패·transport·응답 처리 오류를 삼키지 않습니다.

## declared Resource와 실제 응답

`ImageMemberRecord.Resource`와 `Wire`는 독립 `*resource.RawResource`이고 `Envelope`, `Header`, `StatusCode`는 실제 physical 응답입니다. `ImageID`는 선택한 고정 부모의 복사본입니다. page의 각 행과 Resource/Wire의 map·raw bytes·header를 독립 복사하며 한 반환값의 변경이 다른 반환값이나 다음 요청 target을 바꾸지 않습니다.

Declared Resource view는 9개 필드입니다: Body의 `id`, `name`, `member_id`, `created_at`, `status`, `schema`, `updated_at`와 고정 URI `image_id`, `location`. [Member 소스](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/member.py#L16-L43)의 일곱 Body 속성은 모두 untyped이므로 string·null·숫자·배열·object 값을 그대로 보존하고 날짜를 parse하지 않습니다. 누락값은 null입니다. 응답 unknown·`self`·remote spelling·큰 숫자는 Wire에 남고 declared 필드로 추가하지 않습니다.

Remote `member`는 canonical `member_id`로 옮깁니다. 두 spelling이 함께 있으면 JSON dictionary의 insertion 순서에서 나중 key가 우선합니다. 동일 key를 반복하면 마지막 값을 쓰되 첫 insertion 위치를 유지합니다. **id 키가 있는 행은 null을 포함해 그 값을 유지하고, 없을 때만 member_id를 alternate ID로 사용합니다.** inherited name은 별도 속성입니다. canonical Glance 서버가 보통 name을 반환하지 않아도 generic Source find는 name 속성을 비교합니다.

목록의 image_id는 응답의 값과 무관하게 선택한 부모 URI입니다. location은 옵션 callback·부모 조회 전에 snapshot한 Connection 위치를 사용하고 응답 location은 route를 만들지 않습니다. 직접 `image.New(client)`로 만들면 Connection dependency가 없으므로 recomputed location은 null입니다. `image.NewWithDependencies`에 concrete CloudLocation getter를 공급할 수도 있습니다. Go의 명시 Zone은 유지하며 Source current_location의 기본 zone=None과 구분합니다. Python 기본 `to_dict()`가 URI를 제외하는 것과 달리 Go는 fixed image_id를 provenance로 포함합니다.

## 목록 순회와 오류

목록은 actual HTTP200..399에서 전체 body가 valid UTF-8의 JSON nonnull object여야 하며 `members` key가 필요합니다. members 배열 외 단일 행 object도 generic Source처럼 받습니다. 소비한 행은 nonnull object여야 하고 raw `connection`, `microversion`, `_synchronized`는 Source constructor 중복 인자 오류에 대응해 거부합니다. caller break·cap 이후 미소비 행을 projection하지 않지만 전체 페이지 JSON/UTF-8 검증은 이미 수행합니다.

Body `links`·`members_links`·`next`와 HTTP Link의 다음 페이지를 처리합니다. `links:{next:"URL"}`는 Go 확장이고 Source dictionary 순회는 rel/href가 함께 있는 item을 찾습니다. top-level next의 null·false·숫자0·빈 문자열·빈 array/object는 Source truthiness처럼 무시하고 HTTP Link나 marker fallback을 계속 검사합니다. truthy nonstring next는 실제 page 증거가 있는 처리 오류입니다. 숫자의 zero 판정은 Go의 exact decimal 규칙을 사용합니다. 양수 초기 limit 또는 max-items hint가 있으면 마지막 소비 행의 ID를 marker fallback에 사용하며 짧은 nonempty 페이지도 이어갑니다. 로컬 필터에 실패한 행도 마지막 marker와 raw cap에 기여합니다. 다음 링크에서 처음 발견한 limit만으로 fallback을 새로 활성화하지 않습니다. 빈 페이지·cap·`Paginated=false`·caller break는 continuation 평가 전에 끝납니다.

Go는 같은 origin·고정 collection·초기 query를 유지하며 충돌 링크·query drift·반복 marker·cycle·다른 target은 다음 HTTP 전에 거부합니다. 고정한 `/v2/images/{image_id}/members`의 정확한 escaped 경로는 service prefix를 유지하는 동일 collection alias로 처리합니다. Source의 자유로운 URL·query를 모두 따라가는 것은 아닙니다. 응답 id·name·schema·self·location은 request route를 바꾸지 않습니다.

accepted read·Close·context·source·UTF-8·JSON·shape·projection·continuation 오류는 실제 응답이 있으면 `*resource.ResponseError`에 전체 Body·Header·StatusCode와 원인을 남깁니다. rejected status는 native HTTP 오류이며 요청 전 오류에는 receipt가 없습니다. body는 한 번 닫고 accepted 처리 실패 후 replay하지 않습니다. 기존 native pre-body retry·reauth·backoff와 현재 provider의 live token을 사용합니다. source/provider나 operation guard에서 관찰한 실패는 뒤에서 원래 상태로 복구되어도 오류로 남습니다.

## Find의 GET와 목록 fallback

Find는 안전하게 escaped한 `nameOrID`로 먼저 `GET images/{image_id}/members/{nameOrID}`를 수행합니다. [Resource.find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2524-L2582)처럼 실제 최종 native400·403·404 오류에서만 같은 부모의 기본 목록으로 넘어갑니다. configured native retry·reauth 이후의 최종 오류를 판단하며 다른 transport·hook·nested 오류나 joined body/Close 실패를 status만 보고 fallback하지 않습니다. RetryFunc가 받은 원래 plain native 오류를 그대로 반환하면 추가 실패 원인으로 만들지 않고 최종 native rejection을 유지합니다.

이 Find의 GET400·403·404에는 native rejected body의 Read·Close·source 실패 관찰을 추가합니다. Gophercloud가 native 상태 오류를 만들면서 생략하는 I/O 원인도 함께 유지해 faulty rejection 뒤 LIST로 넘어가지 않습니다. 세 상태는 성공 코드에 포함하지 않으므로 native retry·재인증 정책이 먼저 적용됩니다. 다른 rejected GET 상태와 목록의 rejected 응답은 기존 native Gophercloud I/O 정책을 유지합니다. 모든 rejected status의 Read/Close 처리를 새 정책으로 바꾼 것은 아닙니다.

직접 GET의 actual200..399 응답은 성공입니다. 새 Resource에 명시한 id=nameOrID를 seed하므로 응답에 id가 없으면 그 값이 유지되고 member alias가 달라도 seed를 바꾸지 않습니다. 응답의 literal id는 null을 포함해 seed를 교체합니다. Member Body만 overlay하고 고정 image_id·Connection location을 유지하며 Wire에 요청 id나 부모를 합성하지 않습니다. invalid/빈 JSON은 Source fetch처럼 seed/default Resource를 반환하고 Wire는 nil이며 실제 bytes는 Envelope에 남습니다. valid JSON null·array·scalar나 invalid UTF-8은 처리 오류입니다.

Fallback은 첫 요청에 name query·limit·marker를 합성하지 않고 기본 멤버 목록을 순회해 **ID 또는 inherited name의 정확한 문자열 일치**를 찾습니다. 한 행이 두 조건에 모두 맞아도 한 후보입니다. 두 번째 일치에서 중복 오류를 반환하며 첫 후보를 다시 GET하지 않습니다. 유일한 후보라도 후속 페이지·행·continuation 오류가 생기면 nil record와 그 오류를 반환합니다. 정상적으로 끝난 하나의 후보는 해당 목록 페이지의 실제 receipt를 유지합니다.

[고정 Glance 정책](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L251-L278)은 `get_member`·`get_members`에 admin 또는 project/shared member reader를 허용합니다. SDK가 role preflight를 추가하지 않으며 정책·이미지 소유권·visibility는 서버가 판단합니다. 서버 show는 Forbidden을404로 숨길 수 있으므로 GET404나 최종 nil 결과가 멤버의 실제 부재를 증명하지는 않습니다.

이 API는 mutable Python Resource/cache/session 객체를 재사용하지 않고 각 호출에 fresh record를 제공합니다. local HTTP 계약·문서 빌드는 고정 소스와 Go 구현을 비교하는 증거이며 실제 인증·cloud 정책이나 Python runtime 실행을 증명하지 않습니다.

목록·옵션·Resource 계약은 [owned 멤버 테스트](member_records_test.go), GET-first·native fallback·중복·선택한 rejected body 실패와 응답 증거는 [owned Find 테스트](member_record_find_test.go)에서 검증합니다. falsey next의 opt-in과 후속 Link/marker 처리는 [공통 목록 테스트](../internal/rest/falsey_next_test.go)에서 검증하며 기존 typed 계약은 [기존 멤버 테스트](members_contracts_test.go)를 재사용합니다.
