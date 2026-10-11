# Trunk 목록: openstacksdk와 Go

고정 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의
`network.trunks(**query)`를 대응합니다. Gophercloud pin은 `v2.15.0`입니다.
`ListRecords`는 기존 Trunk Record 모델과 공통 Neutron 목록 엔진을 사용하며,
별도 사용자 builder나 interface 구현을 요구하지 않습니다.

| openstacksdk | Go | 동작 |
|---|---|---|
| `conn.network.trunks(**query)` | `service.Trunks.ListRecords(ctx, options...)` | lazy 목록, 각 iteration에서 source/options snapshot |
| `list(conn.network.trunks(**query))` | `service.Trunks.AllRecords(ctx, options...)` | 전체 관찰 성공 후 반환; 늦은 실패에 부분 결과 폐기 |
| `name`, `status`, `port_id`, `project_id` | `WithRecordListQuery(key, value)` | Source 선언 server query |
| `id`, `tenant_id` | 같은 query 옵션 | Source 선언 Body의 로컬 필터 |
| `limit`, `marker`, `max_items`, `paginated` | `WithRecordListLimit`·`Marker`·`MaxItems`·`Paginated` | raw 행 cap·페이지 제어 |

기존 native `Trunks.List`·`All`과 `Resources.List`는 계속 사용할 수 있습니다.
새 목록은 Python descriptor 뷰와 실제 응답을 함께 제공하는 named API mapping이며,
전체 mutable Resource·session·cache lifecycle 완료를 뜻하지 않습니다.

## 실행 가능한 Python 예제

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--name", default="")
parser.add_argument("--port", default="")
parser.add_argument("--max-items", type=int, default=0)
parser.add_argument("--single-page", action="store_true")
args = parser.parse_args()

conn = openstack.connect(cloud=args.cloud)
query = {"paginated": not args.single_page, "max_items": args.max_items}
if args.name:
    query["name"] = args.name
if args.port:
    query["port_id"] = args.port
for trunk in conn.network.trunks(**query):
    print(json.dumps(trunk.to_dict(), indent=2))
```

## 실행 가능한 Go main

[설치 안내](../../docs/install.md)대로 모듈을 준비하고 `main.go`에 저장합니다.
`go run . -cloud dev -name TRUNK_NAME`은 조회만 실행합니다.
`-name`은 Source server query로 전송하며 추가적인 이름 equality를 강제하지 않습니다.

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    name := flag.String("name", "", "server-side trunk name query")
    port := flag.String("port", "", "parent port ID query")
    cap := flag.Int("max-items", 0, "raw row cap; 0 means unlimited")
    single := flag.Bool("single-page", false, "observe first page only")
    version := flag.String("microversion", "", "optional selected service version")
    flag.Parse()

    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    connOptions := []sdk.ConnectionOption{sdk.WithCloud(*cloud)}
    if *version != "" {
        connOptions = append(connOptions, sdk.WithMicroversion(sdk.Network, *version))
    }
    conn, err := sdk.Connect(ctx, connOptions...)
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    options := []trunks.RecordListOption{
        trunks.WithRecordListMaxItems(*cap),
        trunks.WithRecordListPaginated(!*single),
    }
    if *name != "" { options = append(options, trunks.WithRecordListQuery("name", *name)) }
    if *port != "" { options = append(options, trunks.WithRecordListQuery("port_id", *port)) }
    rows, err := service.Trunks.AllRecords(ctx, options...)
    if err != nil { log.Fatal(err) }
    for _, row := range rows {
        body, err := json.MarshalIndent(row.Resource.Body, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Println(string(body))
        fmt.Printf("HTTP %d request-id=%s\n", row.StatusCode,
            row.Header.Get("X-Openstack-Request-Id"))
    }
}
```

## Descriptor와 실제 응답

`TrunkListRecord`는 기존 `TrunkFindRecord`의 alias로 같은 owned Record 모델을
재사용합니다. `Resource.Body`에는 Source의 10개 유효 Body descriptor와 location이
들어갑니다. Trunk는 `Resource + TagMixin`을 상속합니다. `NetworkResource`의
revision·timestamp·If-Match descriptor를 새로 만들지 않습니다.

미수신 일반 필드는 null, tags는 `[]`입니다. 명시 null은 null이며,
`project_id`가 없을 때만 `tenant_id` alias를 사용합니다. `admin_state_up`은
`is_admin_state_up`으로 투영하고 Python truthiness를 적용합니다. 문자열
`"False"`는 true, binary64에 underflow하는 `1e-400`은 false입니다.
입력 object의 canonical/wire 별칭은 마지막 입력 값이 같은 Body field를 소유합니다.

`sub_ports`와 tags는 Source list descriptor입니다. 비null scalar는 한 원소 list로
변환하지만 subport 내부의 port/segmentation 필드를 native struct로 다시 읽거나
변환하지 않습니다. null·false 원소와 임의 내부 JSON을 그대로 보존합니다.
Unknown Body 필드·정수·timestamp·extension 값도 원시 JSON으로 남습니다.
`self`는 Resource 뷰에서 제외하고 실제 Wire에는 보존합니다. `if_match`라는 응답
Body extension이 있으면 literal extension이며, HTTP `If-Match`와 독립적입니다.

`Wire.Body`는 실제 row, `Envelope`는 실제 전체 page body,
`Header`·`StatusCode`는 실제 HTTP receipt입니다. Resource·Wire·Envelope·Header는
독립 snapshot입니다. 임의 project/zone/runtime Connection을 추정하지 않으며
location은 null 또는 literal extension입니다. Python plain/control/computed
attribute shadowing과 full current_location 계산은 이 JSON Record의 범위 밖입니다.

LIST constructor의 `cls`, `connection`, `microversion`, `_synchronized`는
존재만으로 충돌합니다. 이 오류는 actual whole page receipt를 유지합니다.
subport/tag mutation·조회·Get·Find API의 기존 계약은 별도로 검토됩니다.

## Query와 페이지

Source의 선언 14개 mapping을 사용합니다: limit, marker, name, description,
fields, port_id, status, sub_ports, project_id, is_admin_state_up→admin_state_up,
tags, any_tags→tags-any, not_tags→not-tags, not_any_tags→not-tags-any.
`fields`는 Source LIST처럼 값 자체를 보내며 dedicated GetWithFields의 canonical
Body 이름 변환을 수행하지 않습니다.

canonical query가 wire alias보다 **존재 우선**이며 null도 alias 값을 막습니다.
null은 wire 값을 생략하고 false·0·빈 문자열을 보존합니다. 배열은 반복 URL 값이며
null 원소를 생략합니다. Unknown kwargs는 서버 query나 local filter를 만들지 않고
discard합니다. `WithRecordListQuery("x_unknown", ...)`는 unknown 값의 JSON 변환도
수행하지 않아 function/channel처럼 변환 불가능한 값도 discard할 수 있습니다.

id·tenant_id만 로컬 Body 필터입니다. name/status/sub_ports/project_id 등은
server query이며 응답이 조건과 다르더라도 숨겨서 필터링하지 않습니다. local dict는
Source recursive subset·falsey·truthy scalar empty-dict·mapping.get 오류를 보존합니다.
scalar equality는 Go의 exact JSON decimal+bool-number profile입니다. 모든 Python
binary64 rounding·nonfinite/custom object equality와 동등하다고 주장하지 않습니다.
sequential 옵션은 필터 순서를 유지하고 bulk map/callback 새 map keys는 sorted 순서입니다.

paginated 기본 true, max_items=0은 무제한입니다. 양수 cap은 **로컬 필터 전 raw 행**을
셉니다. cap은 처음 limit hint를 제공하며 limit=0도 양수 cap hint로 대체합니다.
cap·consumer break는 남은 row나 next를 해석하지 않습니다. AllRecords는 관찰 중의
늦은 오류가 발생하면 이미 받은 row도 반환하지 않습니다. 중복 행은 목록에 그대로 남고
추가 member GET·identity lookup을 하지 않습니다.

plural/body/top-level-next/HTTP Link와 정확한 relative `/v2.0/trunks` alias를 지원합니다.
dictionary links는 Go 호환 확장입니다. explicit limit이 있는 짧은 nonempty page도
retained Wire ID marker로 계속하며 빈 page에서 멈춥니다. top-level next의 fraction/
exponent truthiness는 binary64로 판단하므로 `next:1e-400`은 받은 page를 보존한 채
종료합니다. truthy nonstring next는 accepted whole-page receipt와 함께 실패합니다.
foreign origin·collection/filter 변경·cursor loop는 고정 continuation guard로 거부합니다.
한 개의 nonempty URL 값으로 표현할 수 없는 marker도 거부합니다. arbitrary Python
link retargeting과 runtime 객체는 범위 밖입니다. `allow_unknown_params`, deprecated
`jmespath_filters`, `__conflicting_attrs` 복원 및 raw session/route/header controls는
HTTP 전에 명시적으로 거부합니다.

## Source·권한·오류의 경계

Trunk는 class-specific 최대 microversion을 선언하지 않아 기본 discovery를 추가하지
않습니다. 원래 선택된 version/absence를 유지하며 `WithRecordListMicroversion`을
명시하면 모든 page에 적용합니다. 빈 명시 override는 version header를 생략합니다.
원본 service와 header map은 변경하지 않습니다. LIST Accept는 `application/json`이
configured Accept보다 우선하고 safe explicit Headers Accept가 있으면 그 값을 씁니다.
인증/framing/route/protocol override는 owned API로만 처리합니다.

이 공통 엔진의 Go accepted final status 범위는 **200..399**입니다. pinned
`raise_from_response`는 final<400을 받아들이므로 transport가 직접 노출한 1xx
표현은 명시적인 차이입니다. 일반 HTTP client의 interim1xx 처리까지 동일하다고
주장하지 않습니다. accepted204에도 목록 JSON이 필요합니다. 깨끗한 목록
400/403/404와 나머지 HTTP 실패는 missing으로 억제하지 않습니다.

모든 rejected400..599 물리 body의 Read·Close·sticky Source·custom cancellation
원인을 actual body/header/status 및 native rejection과 함께 보존합니다. 물리 오류가
있으면 retry/reauth가 willing이어도 재요청하지 않습니다. Clean retry/reauth는
고정 route/query/body/version/header policy 안에서 live provider token을 사용할 수 있습니다.
401의 재인증 guard 실패는 Gophercloud `ErrUnableToReauthenticate.ErrOriginal`에
actual401 응답을, `ErrReauth`에 guard·물리·취소 원인을 보존합니다. 이 native 오류는
두 독립 실패 때문에 Unwrap하지 않으며 공개 필드로 각각 확인합니다.
알려진 query의 JSON 변환 실패는 `resource.ErrInvalidOption`과 원래 marshal 원인을
함께 보존하고 HTTP 전에 종료합니다. 선언되지 않은 kwargs는 JSON 변환 자체를 하지 않습니다.

Neutron27.0.0 stock `get_trunk` collection/member GET은 project
ADMIN_OR_PROJECT_READER 정책입니다. 실제 배포가 ownership·policy를 검사하며
SDK는 관리자 token 획득이나 권한 우회 없이 실제 거부 응답을 보존합니다.
[고정 정책 Source](https://opendev.org/openstack/neutron/src/tag/27.0.0/neutron/conf/policies/trunk.py).

각 iteration은 실제 HTTP를 관찰합니다. Python Proxy의 optional response cache,
mutable Resource dirty state, session injection과 full cache invalidation lifecycle은
독립 공통 SDK 목표이며 이 named Record mapping에서 구현했다고 주장하지 않습니다.
fixture/race/외부 main 빌드와 실제 인증된 Neutron 배포 시험도 별개입니다.
기존 `nativeTrunkAPI`·physical body fixture 및 public Gophercloud `testhelper`를 쓰는
공통 `internal/testcloud`를 재사용합니다.
