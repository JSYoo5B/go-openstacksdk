# Trunk과 Address Group의 Python식 find

고정 Source `openstacksdk ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 두 proxy 선언을 대상으로 합니다.

| Python | Go | 경로 |
| --- | --- | --- |
| `conn.network.find_trunk(name_or_id, ignore_missing=True, **query)` | `network.Trunks.FindRecord(ctx, nameOrID, options...)` | `/trunks` |
| `conn.network.find_address_group(name_or_id, ignore_missing=True, **query)` | `network.SecurityAddressGroups.FindRecord(ctx, nameOrID, options...)` | `/address-groups` |

첫 GET이 성공하면 그 Resource를 바로 반환합니다. 깨끗한 HTTP 400·403·404만 목록 검색으로 이어집니다. 모든 페이지에서 ID 또는 이름이 정확히 일치하는 Resource 하나를 찾고, 중복은 `resource.ErrAmbiguous`, 완전한 목록 관측 뒤 없음은 기본적으로 `nil, nil`입니다. `WithRecordFindIgnoreMissing(false)`는 논리적 없음을 `resource.ErrNotFound`로 바꿉니다. 목록의 404나 나중 페이지 오류는 이 옵션으로 숨기지 않습니다.

```python
import json
import openstack

with openstack.connect(cloud="devstack") as conn:
    trunk = conn.network.find_trunk(
        "application-trunk", ignore_missing=False,
        project_id="project-id", is_admin_state_up=False,
        fields=["id", "name", "admin_state_up"],
    )
    address_group = conn.network.find_address_group(
        "application-addresses", ignore_missing=False,
        project_id="project-id", addresses=["10.0.0.0/24"],
    )
    print(json.dumps(trunk.to_dict(), indent=2))
    print(json.dumps(address_group.to_dict(), indent=2))
```

아래는 독립 `main.go`입니다. 인증에는 기존 `clouds.yaml`의 cloud를 사용하며 find만 수행합니다. `-kind trunk` 또는 `-kind address-group`을 선택합니다.

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func run(ctx context.Context, cloud, kind, identity, project string) (*resource.RawResource, error) {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return nil, err }
    network, err := conn.NetworkV2(ctx)
    if err != nil { return nil, err }
    switch kind {
    case "trunk":
        options := []trunks.RecordFindOption{
            trunks.WithRecordFindIgnoreMissing(false),
            trunks.WithRecordFindQuery("is_admin_state_up", false),
        }
        if project != "" { options = append(options, trunks.WithRecordFindQuery("project_id", project)) }
        record, err := network.Trunks.FindRecord(ctx, identity, options...)
        if err != nil || record == nil { return nil, err }
        return record.Resource, nil
    case "address-group":
        options := []addressgroups.RecordFindOption{
            addressgroups.WithRecordFindIgnoreMissing(false),
        }
        if project != "" { options = append(options, addressgroups.WithRecordFindQuery("project_id", project)) }
        record, err := network.SecurityAddressGroups.FindRecord(ctx, identity, options...)
        if err != nil || record == nil { return nil, err }
        return record.Resource, nil
    default:
        return nil, fmt.Errorf("unknown kind %q", kind)
    }
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    kind := flag.String("kind", "trunk", "trunk or address-group")
    identity := flag.String("identity", "application-trunk", "exact ID or name")
    project := flag.String("project", "", "optional project filter")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    value, err := run(ctx, *cloud, *kind, *identity, *project)
    if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(value); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}
```

`WithRecordFindQuery`는 값을 JSON으로 복사합니다. null, false, 0, 빈 문자열, 빈 배열을 구분합니다. GET은 kwargs를 원래 이름으로 전송하고 constructor 상태로도 보존합니다. LIST는 아래 실제 선언된 query만 전송합니다. 알려지지 않은 query는 LIST에서 버리지만 실제 응답의 새로운 Body 필드는 Resource에 보존합니다.

| 서비스 | LIST query와 별칭 | 로컬 필터 |
| --- | --- | --- |
| Trunk | `limit`, `marker`, `name`, `description`, `fields`, `port_id`, `status`, `sub_ports`, `project_id`; `is_admin_state_up` → `admin_state_up`; `any_tags` → `tags-any`; `not_tags` → `not-tags`; `not_any_tags` → `not-tags-any`; `tags` | `tenant_id` |
| Address Group | `limit`, `marker`, `fields`, `sort_key`, `sort_dir`, `name`, `description`, `project_id` | `tenant_id`, `addresses` |

LIST query에서 canonical 이름과 wire 별칭을 함께 주면 canonical 값이 우선합니다. GET의 constructor seed는 kwargs 삽입 순서대로 소비하므로 같은 Body descriptor를 가리키는 마지막 별칭의 값이 우선합니다. 예를 들어 `WithRecordFindQuery("is_admin_state_up", false)` 뒤에 `WithRecordFindQuery("admin_state_up", true)`를 주고 응답이 이 필드를 생략하면 GET의 Resource bool은 true입니다. 반대로 주면 false입니다. `WithRecordFindOpts`의 bulk Query map은 삽입 순서가 없으므로 key를 사전순으로 정렬해 일관된 결과를 만듭니다. 실제 응답 필드는 이 seed를 덮어씁니다. `fields` 배열의 내용은 Source처럼 바꾸지 않고 반복 query로 보냅니다. `name`을 생략하면 검색 문자열을 목록 hint로 추가합니다. `name=null` 또는 빈 문자열은 명시한 값이므로 hint로 덮어쓰지 않습니다. `all_projects`는 두 서비스에서 wire query로 보내지 않습니다. `id` kwargs는 `name_or_id`와 충돌하므로 사전 거부합니다.

로컬 필터는 선언된 Body descriptor view에 적용합니다. 예를 들어 Address Group의 `addresses="one"` 응답은 list descriptor에 따라 `["one"]`으로 보이므로 문자열 필터와 다릅니다. bool과 숫자의 동등 비교는 서로 비교할 수 있지만, Go는 JSON 숫자의 정확한 십진수 값을 사용합니다. Python JSON float의 binary64 반올림·underflow·overflow를 동등 비교에 그대로 적용하지 않습니다. 예를 들어 `1e-4000`과 0은 이 Go 필터에서 다릅니다. `9007199254740993.0`과 정수 `9007199254740993`은 Go에서는 같지만 Python의 float 변환 뒤에는 다를 수 있습니다. 이는 명시적 Go 비교 profile이며 Python 숫자 동등 비교 전체의 parity로 세지 않습니다. 딕셔너리 필터는 실제 값의 Python truthiness를 먼저 확인하고, 비어 있지 않은 필터가 mapping 이외의 값에서 `.get`을 요구하면 실제 페이지를 담은 오류를 반환합니다.

Trunk의 bool descriptor, 로컬 딕셔너리 필터의 실제 값 검사, `paginated`는 같은 Source JSON truthiness를 사용합니다. 소수점 또는 exponent가 있는 숫자는 binary64로 평가하므로 `1e-4000`은 false, `1e4000`은 true, `-0.0`은 false입니다. 소수점·exponent가 없는 정수는 임의 정밀도를 유지하고, 실제 Wire·Envelope의 숫자 bytes는 바꾸지 않습니다.

`paginated`는 이 truthiness로 첫 페이지 사용 여부를 선택합니다. `max_items`는 0 이상의 Go 정수이며, 로컬 필터와 이름 비교 전에 읽은 raw row 수를 제한합니다. 직접 `limit`을 주지 않으면 `max_items`가 wire limit hint가 됩니다. LIST는 해당 plural links, 최상위 links/next, RFC Link header, limit/marker 이동을 처리합니다. 나중 페이지까지 확인해 중복과 늦은 오류를 잡습니다. next가 기존 필터·origin·컬렉션을 바꾸거나 순환하면 페이지 증거와 함께 거부합니다.

`WithRecordFindMicroversion`은 Source와 같이 GET에만 적용합니다. fallback LIST는 캡처한 서비스 기본 microversion을 사용합니다. 선택된 원본 client의 microversion과 header는 변경하지 않습니다. `WithRecordFindHeaders`는 Go의 명시적 추가 옵션으로 두 phase에 안전한 header를 적용합니다. LIST의 Accept 기본값은 `application/json`이고 이 옵션의 명시적 Accept가 우선합니다. Python `headers` kwargs가 GET에서는 query/constructor 값, LIST에서는 header로 소비되는 암묵적 차이는 자동 추정하지 않습니다. auth·프레이밍·충돌 version header는 SDK가 사전 거부합니다.

`TrunkFindRecord`와 `AddressGroupFindRecord`는 세 가지 관측을 구분합니다.

- `Resource`: constructor seed, 실제 Body, 선언된 alias/default/descriptor 변환을 합친 소유 view입니다. Trunk의 누락 tags는 `[]`, 누락 bool과 sub_ports는 null입니다. Address Group의 누락 addresses는 null입니다. 명시적 null은 유지합니다. 프로젝트 ID가 없을 때 tenant_id를 alias로 사용합니다.
- `Wire`: 해당 HTTP 응답에서 실제 받은 singular object 또는 목록 row입니다. 숫자의 정밀도와 미선언 extension을 보존합니다.
- `Envelope`, `Header`, `StatusCode`: 원래 전체 HTTP 응답 증거입니다. 목록 Resource의 Envelope는 전체 페이지입니다. seed와 descriptor default를 실제 응답 필드로 만들지 않습니다.

두 클래스는 NetworkResource를 상속하지 않으므로 `if_match` Body 필드가 존재하면 일반 extension입니다. HTTP If-Match를 가상의 Resource descriptor로 합성하지 않습니다. GET의 비 JSON/빈 성공 Body는 Python `_translate_response`처럼 seed/default view를 유지합니다. JSON null, scalar 또는 잘못된 목록은 실제 응답을 가진 오류입니다.

Go 경계도 명시합니다. identity는 UTF-8·비공백·control 없는 literal 문자열이며 slash와 percent는 URL segment로 escape합니다. 임의 `base_path`, session/connection, constructor `self`/`_synchronized`, response-envelope/cache controls, Computed location과 같은 제어 kwargs는 사전 거부합니다. options와 응답은 immutable snapshot이며 Python mutable Resource dirty-state, Connection-derived location과 선택적 GET cache 전체를 구현한 것으로 세지 않습니다. 기존 typed `FindIdentity`, `Get`, `List`와 이 Source profile은 별도로 사용할 수 있습니다.

성공 HTTP 범위는 200–399입니다. 거부된 400–599의 native status/body/header를 보존하며, accepted Read/Close/decode 실패는 `resource.ResponseError`에 실제 응답을 담습니다. 취소의 원인과 retry callback 오류도 유지합니다. 깨끗한 재시도는 고정 route·Body·phase별 version/Accept 정책을 유지하고, 변조는 추가 HTTP 전에 거부합니다. 재인증은 기존 provider를 사용하므로 갱신된 token이 다음 시도에 반영됩니다.

서비스별 회귀 테스트는 기존 `internal/testcloud`와 Gophercloud testhelper fixture, 공유 guarded REST와 collection/identity 구현을 사용합니다. 인증된 실제 Neutron 클라우드의 endpoint·권한 acceptance는 이 fixture 결과로 주장하지 않습니다.
