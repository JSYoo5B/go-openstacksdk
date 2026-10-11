# QoS Policy Find와 List의 Python/Go 사용법

고정 Source는 `openstacksdk ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`이며 native transport는 Gophercloud `v2.15.0`입니다. 이 문서의 지원 판정은 아래 두 proxy 선언만 대상으로 합니다. 기존 Get/Create/Update/Delete를 다시 집계하지 않습니다.

| Python | Go | 실제 경로 |
| --- | --- | --- |
| `conn.network.find_qos_policy(name_or_id, ignore_missing=True, **query)` | `network.QoSPolicies.FindRecord(ctx, nameOrID, options...)` | GET `/qos/policies/{id}`, 필요하면 `/qos/policies` |
| `conn.network.qos_policies(**query)` | `network.QoSPolicies.ListRecords(ctx, options...)` 또는 `AllRecords` | GET `/qos/policies` |

`FindRecord`는 먼저 literal ID GET을 시도합니다. 깨끗한 HTTP 400·403·404만 목록 검색으로 이어집니다. 모든 페이지에서 ID 또는 이름이 정확히 일치하는 하나를 찾고, 나중 페이지의 중복은 `resource.ErrAmbiguous`입니다. 성공적으로 전체 목록을 읽고도 없으면 기본값은 `nil, nil`입니다. `WithRecordFindIgnoreMissing(false)`는 이 논리적 없음을 `resource.ErrNotFound`로 바꾸며 목록 HTTP 오류나 응답 처리 오류를 숨기지 않습니다.

다음 Python 코드는 `clouds.yaml`의 cloud에서 정책 하나를 찾고 같은 project의 정책 목록을 읽습니다.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--identity", default="application-qos")
parser.add_argument("--project", default="")
args = parser.parse_args()
query = {"is_shared": False}
if args.project:
    query["project_id"] = args.project

with openstack.connect(cloud=args.cloud) as conn:
    policy = conn.network.find_qos_policy(
        args.identity, ignore_missing=False, **query
    )
    print(json.dumps(policy.to_dict(), indent=2))
    for policy in conn.network.qos_policies(**query):
        print(json.dumps(policy.to_dict(), indent=2))
```

아래는 독립 `main.go`입니다. `-mode find` 또는 `-mode list`로 두 API 중 하나를 선택합니다. `ListRecords`는 반복 중 앞서 읽은 행을 사용할 수 있고, `AllRecords`는 전체 성공 때만 목록을 반환합니다.

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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func run(ctx context.Context, cloud, mode, identity, project string) (any, error) {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return nil, err }
    network, err := conn.NetworkV2(ctx)
    if err != nil { return nil, err }
    switch mode {
    case "find":
        options := []policies.RecordFindOption{
            policies.WithRecordFindIgnoreMissing(false),
            policies.WithRecordFindQuery("is_shared", false),
        }
        if project != "" {
            options = append(options, policies.WithRecordFindQuery("project_id", project))
        }
        record, err := network.QoSPolicies.FindRecord(ctx, identity, options...)
        if err != nil || record == nil { return nil, err }
        return record.Resource, nil
    case "list":
        options := []policies.RecordListOption{
            policies.WithRecordListQuery("is_shared", false),
        }
        if project != "" {
            options = append(options, policies.WithRecordListQuery("project_id", project))
        }
        records, err := network.QoSPolicies.AllRecords(ctx, options...)
        if err != nil { return nil, err }
        values := make([]*resource.RawResource, len(records))
        for i, record := range records { values[i] = record.Resource }
        return values, nil
    default:
        return nil, fmt.Errorf("unknown mode %q", mode)
    }
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    mode := flag.String("mode", "find", "find or list")
    identity := flag.String("identity", "application-qos", "exact policy ID or name")
    project := flag.String("project", "", "optional project filter")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    value, err := run(ctx, *cloud, *mode, *identity, *project)
    if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(value); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}
```

예제 사용 명령은 `go run main.go -cloud devstack -mode find -identity application-qos` 또는 `go run main.go -cloud devstack -mode list -project project-id`입니다. 새로운 interface builder나 사용자 decoder가 필요하지 않습니다. 호출마다 필요한 concrete `WithRecordFind...`·`WithRecordList...` 옵션만 조합합니다.

| List 분류 | 필드 |
| --- | --- |
| 서버 query | `limit`, `marker`, `name`, `description`, `fields`, `id`, `is_default`, `project_id`, `sort_key`, `sort_dir`, `tags` |
| 서버 alias | `is_shared` → `shared`, `any_tags` → `tags-any`, `not_tags` → `not-tags`, `not_any_tags` → `not-tags-any` |
| 로컬 Body 필터 | `rules`, `tenant_id` |
| SDK list 제어 | `max_items`, `paginated`; `WithRecordListLimit`·`Marker`·`MaxItems`·`Paginated` convenience |

List는 선언된 query와 wire alias만 전송합니다. unknown kwargs와 `all_projects`는 버리며, 직접 `status`·`revision_number`·timestamp·`if_match`를 query로 추정하지 않습니다. `fields` 배열의 내용은 이 Source `_query_mapping`에 CSV 변환이 없어 그대로 반복 query로 보냅니다. 기존 `GetWithFields`의 Body field-name 변환과는 별도 선언입니다. `id`는 List query로 허용하지만 Find의 `id` kwargs는 positional identity와 충돌하므로 사전 거부합니다.

Find의 GET은 kwargs를 원래 이름으로 보내고 constructor seed로도 저장합니다. fallback List에서만 alias 변환과 로컬 필터 분류를 적용합니다. List query에 canonical 이름과 wire alias가 함께 있으면 canonical 값이 우선합니다. constructor seed와 응답 Body는 입력 순서대로 alias를 소비하므로 마지막 값이 우선합니다. 예를 들어 Find에서 `is_shared=false` 다음 `shared=true`를 주고 응답이 이 필드를 생략하면 Resource bool은 true지만 fallback wire query는 `shared=False`입니다. bulk `RecordFindOpts.Query`에는 map 삽입 순서가 없으므로 SDK는 key를 사전순 정렬합니다. 실제 응답은 seed를 덮어씁니다. Find가 `name`을 생략하면 identity를 List hint로 추가하고, `name=null`·빈 문자열은 명시한 값으로 유지합니다. standalone List는 name hint를 합성하지 않습니다.

`PolicyRecord`는 다음 관측을 구분합니다.

- `Resource`: 실제 Body와 선언된 constructor/alias/default를 합친 소유 view입니다. 누락 `tags`는 `[]`, 누락 `rules`·`is_shared`·`is_default`와 일반 Body descriptor는 null입니다. `project_id`가 없으면 `tenant_id`를 alias로 사용하고 명시적 null은 유지합니다. list descriptor는 scalar를 한 원소 배열로 감싸므로 `rules={...}`는 `[{...}]`, `tags=false`는 `[false]`입니다.
- `Wire`: 실제 singular object 또는 목록 row를 보존합니다. seed/default를 수신 필드로 만들지 않고, 큰 숫자와 새 extension의 JSON bytes를 유지합니다.
- `Envelope`, `Header`, `StatusCode`: 실제 전체 응답 증거입니다. List record의 Envelope는 해당 행의 전체 페이지이며 각 header view는 별도로 소유합니다.

`QoSPolicy`는 `Resource + TagMixin`입니다. `NetworkResource`의 revision/timestamp/Header descriptor를 상속하지 않습니다. 미선언 `created_at`, `revision_number`, `if_match`가 Body에 있으면 raw extension이고, HTTP `If-Match`를 Resource 필드로 합성하지 않습니다. Find GET의 빈/비 JSON 성공 Body는 Source `_translate_response`처럼 seed/default를 유지합니다. JSON null/scalar와 malformed List는 실제 응답을 가진 오류입니다. List JSON의 단일 object는 하나의 행으로 처리합니다.

Bool descriptor, `paginated`와 로컬 dictionary 필터의 실제 값 검사에는 CPython JSON 숫자 truthiness를 적용합니다. 소수점·exponent 숫자는 binary64로 평가해 `1e-4000`은 false, `1e4000`은 true, `-0.0`은 false입니다. plain integer는 임의 정밀도를 유지합니다. pager의 falsey `next` 검사에도 같은 truthiness를 적용해 numeric underflow는 빈 다음 링크처럼 처리하고, truthy 숫자는 잘못된 링크 타입 오류로 보고합니다. Wire bytes는 바꾸지 않습니다. 로컬 숫자 **동등 비교**는 별도 Go 정책으로 정확한 십진수 값을 사용합니다. 따라서 `1e-4000`과 0은 다르고, `9007199254740993.0`과 정수 `9007199254740993`은 같으며, overflow float 둘을 무조건 infinity로 같게 만들지 않습니다. bool과 숫자 0/1의 비교는 허용합니다. Python float 동등 비교 전체의 parity로 이 정책을 설명하지 않습니다.

로컬 `rules`는 descriptor 변환 뒤 완전한 순서·길이·중첩 값을 비교합니다. dictionary 필터는 실제 값의 truthiness를 먼저 확인하고 recursive subset을 검사합니다. 빈 실제 dict/list는 빈 dict 필터와도 매치하지 않습니다. 비어 있지 않은 dictionary 필터가 list/scalar의 `.get`을 요구하면 실제 페이지 증거와 함께 실패합니다.

`max_items`는 0 이상의 Go 정수이며 0은 무제한입니다. cap은 로컬 필터 전에 읽은 raw row 수를 셉니다. limit이 생략되면 양수 max_items가 wire limit hint가 됩니다. `paginated=false`는 첫 페이지에서 멈추며 caller의 iteration break도 다음 요청을 만들지 않습니다. pager는 plural links, top-level links/next, RFC Link, limit/marker, 짧은 페이지의 marker fallback을 처리합니다. 후속 링크가 필터·origin·컬렉션을 바꾸거나 순환하면 실제 페이지를 담은 오류입니다. 외부 링크를 그대로 따라가는 Python의 일부 동작은 Go에서 이 고정 범위 정책으로 개선합니다.

`WithRecordFindMicroversion`은 GET에만 적용하며 fallback List는 캡처한 서비스 기본 microversion을 사용합니다. `WithRecordListMicroversion`은 standalone List의 모든 페이지에 적용합니다. 원본 client의 version/headers는 수정하지 않습니다. 두 phase와 standalone List는 재시도·reauth 중에도 선택한 version을 고정하며, selected version의 canonical Header alias는 request-local native spelling 하나로 정리합니다. `WithRecord...Headers`는 안전한 header를 소유하는 Go convenience이고 Find의 두 phase에 적용합니다. Python `headers` kwargs가 GET에서는 query/constructor, List에서는 request Header로 소비되는 암묵적 차이는 이 convenience가 자동 추정하지 않습니다. List의 기본 Accept는 `application/json`이며 명시적 Accept 옵션이 우선합니다.

성공 HTTP 정책은 200–399입니다. Source의 단순 final `<400` 판정 중 정상 완료 상태가 아닌 100–199를 성공으로 추가하지 않습니다. 최종 400–599는 native status/body/header/URL을 유지합니다. accepted Read·Close·JSON·descriptor/filter 실패는 `resource.ResponseError`에 실제 응답을 담고 취소·retry callback 원인도 유지합니다. retry hook이 성공 코드, method/URL, response ownership, 선택 version/Accept 또는 금지 auth/protocol Header를 바꾸면 추가 요청 전에 실패합니다. `X-Auth-Token`은 고정 상수가 아닌 기존 ProviderClient의 live 인증 값을 사용합니다. List iterator는 오류 이전에 전달한 행을 호출자가 유지할 수 있으며 `AllRecords`는 오류 때 partial slice를 반환하지 않습니다.

identity는 비공백 UTF-8 literal 문자열이며 control/dot segment는 사전 거부합니다. slash/percent는 segment로 escape합니다. `base_path`, connection/session, constructor/cache/response-envelope controls를 허용하지 않고 `location`은 계산 가능한 mutable Connection state 대신 기본 null view입니다. options의 Query/Headers/포인터와 callbacks에서 넘긴 중간 config를 snapshot합니다. 기존 native typed `Get`, `List`, `All`, `FindIdentity`는 독립적으로 사용할 수 있습니다. mutable Resource의 dirty state·session injection·cache·전체 Connection-derived location까지 완성한 것으로 집계하지 않습니다.

stock [Neutron 27.0.0 QoS policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py)의 `get_policy`는 project scope의 admin 또는 owner project reader 또는 shared-policy branch입니다. 이 두 조회 API를 쓰는 데 관리자 전용 write를 먼저 구현할 필요가 없습니다. 같은 stock의 create/update/delete는 `ADMIN_OR_PROJECT_MANAGER`이며 historic deprecated rule은 admin-only입니다. 프로젝트 reader가 정책을 바꿀 수 있다는 뜻으로 문서화하지 않으며 이 wave는 mutation을 추가 집계하지 않습니다. 실제 cloud의 policy override·extension 가용성·권한은 서버가 결정하고 fixture PASS를 live endpoint 인증 성공으로 주장하지 않습니다.

회귀는 기존 public Gophercloud testhelper 기반 `internal/testcloud`와 QoS native transport fixture를 재사용합니다. Find 20개·List 12개·공통 opt-in/baseline 3개 test group은 GET fallback, MRO/default/unknown attrs, raw-rule 필터와 cap, 실제 200–599, pagination, 옵션 snapshot, physical Read/Close·취소·source 변경, retry/reauth/version/header 소유권을 검증합니다. 전 서비스 validation과 판정대장 반영은 root integration 결과로 별도 확인합니다.
