# Security Group 목록: Python과 Go

`conn.NetworkV2(ctx).SecurityGroups.ListRecords/AllRecords`는 고정 openstacksdk의 `conn.network.security_groups(**query)`를 Source descriptor 형태로 제공합니다. SDK가 kwargs 분류·기본값·alias·로컬 필터·페이지 순회·실제 응답 소유권을 처리합니다. 호출자가 builder나 Resource decoder를 구현하지 않습니다. 같은 API는 `conn.Network(ctx).API.SecurityGroups`와 `groups.New(client)`에서도 사용합니다.

기준은 openstacksdk [`ef55d7d`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L6647-L6665), Gophercloud `v2.15.0`, 서버 정책 Neutron `27.0.0`입니다. 기존 native typed `List/All`, generated Collection, owned `FindRecord`는 별도 계약으로 유지합니다. 이번 named list 구현은 전체 mutable Resource·dirty/cache·Python Adapter/session 완료를 뜻하지 않습니다.

## Python: 독립 실행

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument('--cloud', required=True)
parser.add_argument('--name')
parser.add_argument('--limit', type=int)
parser.add_argument('--max-items', type=int, default=0)
args = parser.parse_args()
conn = openstack.connect(cloud=args.cloud)
query = {'max_items': args.max_items}
if args.name is not None:
    query['name'] = args.name
if args.limit is not None:
    query['limit'] = args.limit
for group in conn.network.security_groups(**query):
    print(json.dumps(group.to_dict(), ensure_ascii=False))
```

## Go: 독립 main

아래 전체 코드를 `main.go`에 저장하고 이 module을 의존하는 Go module에서 실행할 수 있습니다.

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "os"
    "time"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups"
)

func main() {
    cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud")
    name := flag.String("name", "", "optional server name filter")
    limit := flag.Int("limit", -1, "page limit; -1 omits it")
    cap := flag.Int("max-items", 0, "raw row cap; zero is unlimited")
    paginated := flag.Bool("paginated", true, "follow all retained pages")
    flag.Parse()
    if err := run(*cloud, *name, *limit, *cap, *paginated); err != nil {
        log.Fatal(err)
    }
}

func run(cloud, name string, limit, maxItems int, paginated bool) error {
    if limit < -1 { return fmt.Errorf("-limit must be -1 or nonnegative") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    network, err := conn.NetworkV2(ctx)
    if err != nil { return err }
    options := []groups.RecordListOption{
        groups.WithRecordListMaxItems(maxItems),
        groups.WithRecordListPaginated(paginated),
        groups.WithRecordListHeaders(map[string]string{"X-Request-Source": "security-group-list-example"}),
    }
    if name != "" { options = append(options, groups.WithRecordListQuery("name", name)) }
    if limit >= 0 { options = append(options, groups.WithRecordListLimit(limit)) }
    out := json.NewEncoder(os.Stdout)
    for record, readErr := range network.SecurityGroups.ListRecords(ctx, options...) {
        if readErr != nil { return readErr }
        if err := out.Encode(map[string]any{
            "resource": record.Resource.Body,
            "wire": record.Wire.Body,
            "envelope": record.Envelope,
            "headers": record.Header,
            "status_code": record.StatusCode,
        }); err != nil { return err }
    }
    return nil
}
```

예: `go run . -cloud dev -name web -limit 50`. 예제의1분 제한은 caller 선택이며 SDK의 기본 timeout이 아닙니다. `AllRecords(ctx, options...)`는 성공한 빈 목록을 nonnil slice로 반환하고 뒤 page 오류가 있으면 partial slice를 버립니다. iterator에서 이미 받은 값은 caller에게 남습니다. 자동 member GET·name hint·Find는 추가하지 않습니다.

## query와 옵션

`RecordListOpts`는 `Query map[string]json.RawMessage`, `Headers map[string]string`, `Microversion *string`을 소유합니다. `WithRecordListOpts`는 전체 설정을 교체하며 `Query/Headers/Microversion/Limit/Marker/MaxItems/Paginated` helper는 차례대로 설정합니다. 이름은 server query일 뿐 exact Find를 의미하지 않습니다.

| kwargs | 처리 |
|---|---|
| `limit`, `marker` | inherited server page query |
| `description`, `fields`, `id`, `name`, `stateful`, `project_id`, `tenant_id`, `revision_number`, `sort_dir`, `sort_key`, `tags` | 그대로 server query |
| `is_shared`, `any_tags`, `not_tags`, `not_any_tags` | 각각 `shared`, `tags-any`, `not-tags`, `not-tags-any`로 server query |
| `created_at`, `updated_at`, `security_group_rules` | Source Body descriptor 값을 로컬 필터 |
| `max_items`, `paginated` | 로컬 소비 cap·페이지 제어 |
| unknown·Header-only kwargs | value의 transport encoding 전에 무시 |

canonical/wire alias를 둘 다 주면 canonical이 우선합니다. explicit canonical null도 alias를 억제하며 null server query는 생략합니다. `project_id`와 `tenant_id`는 별도 server query이며 response의 project alias와 혼동하지 않습니다. server query array는 반복 값, false는 Requests 스타일 `False`,0은 `0`으로 전달합니다. `fields` 원소를 Body alias로 재작성하거나 순서를 바꾸지 않습니다.

omitted limit은 생략하고 explicit0은 `limit=0`입니다. 양수 max_items와 생략/0 limit을 함께 쓰면 최초 limit을 cap hint로 설정합니다. max_items0은 무제한, paginated 생략은 true, false는 첫 page입니다. cap은 **필터 전 원본 행 수**로 세므로 제외된 행도 소비합니다. Go limit/cap은 nonnegative machine integer, marker는 한 개의 nonblank UTF-8 값으로 제한합니다. 임의 Python runtime 값의 transport 동치까지 주장하지 않습니다.

Query raw bytes·nested map·Headers·Microversion pointer·옵션 slice와 callback 결과를 복사합니다. callback은 각 지연 순회에서 한 번이며 retained config 변경이 다음 callback이나 실행에 섞이지 않습니다. sequential Query helper는 insertion 순서를 유지하고 bulk map이나 새 callback key는 Go map에 순서가 없으므로 정렬합니다. nil/error callback, invalid known JSON·control/header·negative cap은 HTTP 전 오류입니다. 원래 resource/session/base_path/constructor를 대체하는 controls와 `__conflicting_attrs`, deprecated `jmespath_filters`는 이 concrete JSON profile에서 거부합니다. JMESPath expression runtime·conflicting attrs 복원은 shared Python framework 차이로 남기며 일반 unknown처럼 조용히 성공시키지 않습니다.

로컬 object 조건은 Source의 recursive subset과 truthiness를 적용합니다. falsey 실제 값은 맞지 않고 empty object 조건은 truthy scalar에도 맞습니다. truthy scalar/array에 nonempty object 조건을 적용하면 Source `mapping.get` 경계의 오류를 실제 page 증거와 반환합니다. list 조건은 descriptor가 제공한 list와 값 비교하며 nested rule을 새로운 Resource로 변환하지 않습니다. Bool-number equality는 Python `True == 1`을 반영하고 decimal truthiness는 binary64 underflow를 반영합니다. JSON 숫자 equality는 공통 exact decimal profile이므로 Python float rounding/nonfinite/custom 객체와 완전 동치가 아닙니다.

## declared Resource와 실제 Wire

[`SecurityGroup`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/security_group.py)과 NetworkResource·TagMixin의 기존 검토된 MRO를 재사용합니다. Body는 `id`, `name`, `created_at`, `description`, `stateful`, `project_id`, `security_group_rules`, `tenant_id`, `updated_at`, `is_shared`, `revision_number`, `tags`입니다. Header `if_match`와 computed `location`은 Body transport 필드와 구분합니다.

- untyped 필드는 null default이며 explicit null·false·0·string·큰 정수·array·object를 보존합니다. 날짜를 Go time으로 바꾸거나 stateful을 bool로 강제하지 않습니다.
- `project_id`가 생략된 경우만 `tenant_id`를 fallback으로 읽고 explicit null/false/0은 그대로 남습니다.
- `is_shared`는 wire `shared` 또는 canonical constructor 값을 Python truthiness로 변환합니다. nonempty `"false"`는 true이고 decimal `1e-400`은 false입니다. 둘 다 응답에 있으면 JSON dictionary의 마지막 input이 우선합니다.
- revision은 Source integer descriptor를 사용해 arbitrary precision·Unicode decimal digit 문자열·bool int subclass를 보존합니다. nondecimal isdigit 변환 오류는 실제 accepted page를 포함합니다.
- tags는 생략하면 `[]`, explicit null은 null입니다. tags·security_group_rules·if_match의 list descriptor는 nonnull non-list 값을 singleton list로 감쌉니다. nested rule 값은 원본 JSON이며 typed rule cast를 추가하지 않습니다.
- LIST의 if_match는 row constructor의 canonical exact `if_match` 또는 case-insensitive hyphen `If-Match`에서만 읽습니다. `IF_MATCH`는 unknown extension입니다. 실제 page HTTP If-Match는 각 row의 descriptor가 되지 않습니다.
- unknown Body extension을 보존하고 `self`는 constructor view에서 제거합니다. raw `location`은 computed 값으로 덮어쓸 수 없으며 여기서는 Connection 객체가 없는 owned null view를 사용합니다. 전체 current-location/config synthesis는 별도 shared 범위입니다.

`SecurityGroupRecord.Resource`는 declared/default view, `Wire`는 변환 전 실제 row, `Envelope`는 전체 page, Header/StatusCode는 실제 page receipt입니다. 각 행과 Resource/Wire/raw bytes/헤더는 독립 소유입니다. LIST row의 `cls`, `connection`, `microversion`, `_synchronized` 충돌이나 JSON/UTF-8/descriptor 오류는 전체 실제 page를 포함한 `resource.ResponseError`로 남습니다.

## pager·전송·권한

기존 공통 neutronfind/REST pager를 사용합니다. `security_groups` list 또는 singleton object, plural links·top-level next·HTTP Link·short nonempty page의 truthy limit marker continuation, 빈 page 종료를 처리합니다. decimal falsey next는 종료하고 Wire/Envelope의 원래 값을 바꾸지 않습니다. Go dictionary links와 정확한 상대 `/v2.0/security-groups` alias를 허용하며 외부 origin·다른 collection·filter 변경·cycle은 실제 page 증거와 거부합니다. cap이나 consumer break 뒤의 미소비 행·next는 평가하지 않습니다.

selected Neutron microversion은 유지하고 명시 Microversion은 모든 LIST page에 적용합니다. 이 Source class에는 ceiling이 없으므로 discovery나 임의 버전 gate를 추가하지 않습니다. LIST 기본 Accept application/json이 service default보다 우선하고 명시 안전한 Headers의 Accept는 이를 바꿉니다. captured route/client/version·ordinary header와 auth/framing ownership을 고정하고 provider token renewal을 허용합니다. clean native retry는 유지하되 body·selected version·OmitHeaders·금지된 auth/framing 변경은 실제 rejection과 원인을 남깁니다. 조건부 If-Match·If-None-Match·If-Modified-Since·If-Unmodified-Since가 captured/explicit Headers에 없으면 retry에서도 부재를 유지하고, 선택했다면 같은 값을 보존합니다. 주입·변경·누락은 두 번째 HTTP 전에 원래 rejection과 callback cause를 유지한 오류입니다. 이 request Header는 row의 if_match descriptor를 seed하지 않습니다. OkCodes hook이 retry2를 진행시키더라도 최종 응답은 원래 acceptance로 검사합니다.

고정 Source raise_from_response는 status<400에서 반환합니다. Go는 final200..399를 처리하고 informational100..199는 실제 native status/body/header와 거부하는 명시적 차이가 있습니다. accepted/rejected physical Read/Close·custom cancel·순간적인 Source 변경 후 복원은 실제 증거와 모든 원인을 보존하고 body는 한 번 닫습니다. `StopFaultRetryCallbacks` profile은 physical/source/cancel 오류에서 native retry callback 자체도 실행하지 않습니다. invalid UTF-8 거부는 Requests replacement decoding보다 엄격합니다.

[Neutron27.0.0 get_security_group 정책](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/security_group.py#L125-L153)은 collection/member 모두 project scope의 `ADMIN_OR_PROJECT_READER` 또는 `shared_security_group` 조건을 사용합니다. [extension](https://github.com/openstack/neutron/blob/27.0.0/neutron/extensions/securitygroup.py)은 `/security-groups`를 pagination/sorting controller로 등록하고 [stock Controller._items](https://github.com/openstack/neutron/blob/27.0.0/neutron/api/v2/base.py#L292-L378)는 각 행에 show policy를 적용해 보이지 않는 항목을 제외합니다. 빈 목록은 caller가 모든 group을 볼 수 있다는 증거나 client-side 권한 거부가 아닙니다. server policy override·RBAC/shared 접근·extension enablement와 실제 visibility는 배포가 결정합니다.

[계약 테스트](extensions/security/groups/python_list_contracts_test.go)는 기존 nativeSecGroup/findFaultBody fixture와 public Gophercloud testhelper 기반 internal/testcloud를 재사용합니다. 새 HTTPserver·transport·pagination engine을 만들지 않았습니다. focused race와 독립 main compile은 root 통합에서 확인하며 authenticated cloud policy/endpoints 실행과 구분합니다.
