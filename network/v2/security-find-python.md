# Neutron Security Group·Rule 찾기

기준은 Gophercloud **v2.15.0**, openstacksdk **ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe**의 [Security Group proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L6598-L6623), [Rule proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L6768-L6793)와 [Resource.find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2489-L2582)입니다.

| openstacksdk | Go | 기본 동작 |
|---|---|---|
| `conn.network.find_security_group(name_or_id, **query)` | `service.SecurityGroups.FindRecord(ctx, identity, options...)` | member GET, clean 400/403/404만 list fallback |
| `conn.network.find_security_group_rule(name_or_id, **query)` | `service.SecurityRules.FindRecord(ctx, identity, options...)` | 같은 lookup, 자동 `name` query 없음 |
| `ignore_missing=False` | `WithRecordFindIgnoreMissing(false)` | 성공한 전체 검색 뒤 missing 오류 |
| `project_id="PROJECT_ID"` | `WithRecordFindQuery("project_id", projectID)` | GET에는 원래 kwargs, LIST에는 Source query mapping |

두 메서드는 반환된 정확한 ID 또는 정확한 name을 비교합니다. Rule proxy 문서는 ID를 설명하지만 Resource가 물려받은 name도 실제 비교 대상입니다. Group만 fallback에 `name=identity`를 자동으로 넣습니다. 입력 ID/name은 literal 문자열로 한 번 escape하며 이름을 먼저 UUID로 판별하지 않습니다.

```python
import openstack

conn = openstack.connect(cloud="dev")
group = conn.network.find_security_group("web", project_id="PROJECT_ID")
rule = conn.network.find_security_group_rule("RULE_ID", security_group_id="GROUP_ID")
strict = conn.network.find_security_group("web", ignore_missing=False)
print(None if group is None else group.to_dict())
```

## 독립 실행 가능한 Go main

[설치 안내](../../docs/install.md)로 모듈을 준비하고 아래를 `main.go`로 저장합니다. `go run . -cloud dev -group web -project PROJECT_ID` 또는 `go run . -cloud dev -rule RULE_ID -security-group GROUP_ID`로 실행합니다. 검색 외에 생성·변경·삭제를 수행하지 않습니다.

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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func printJSON(value *resource.RawResource) {
    if value == nil { fmt.Println("null"); return }
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    group := flag.String("group", "", "security group exact name or ID")
    rule := flag.String("rule", "", "security group rule exact ID or name")
    project := flag.String("project", "", "optional project_id query")
    groupFilter := flag.String("security-group", "", "optional security_group_id rule query")
    strict := flag.Bool("strict", false, "return an error when missing")
    flag.Parse()
    if *group == "" && *rule == "" { log.Fatal("-group or -rule is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    if *group != "" {
        options := []groups.RecordFindOption{groups.WithRecordFindIgnoreMissing(!*strict)}
        if *project != "" {
            options = append(options, groups.WithRecordFindQuery("project_id", *project))
        }
        value, err := service.SecurityGroups.FindRecord(ctx, *group, options...)
        if err != nil { log.Fatal(err) }
        if value == nil { printJSON(nil) } else { printJSON(value.Resource) }
    }
    if *rule != "" {
        options := []rules.RecordFindOption{rules.WithRecordFindIgnoreMissing(!*strict)}
        if *project != "" {
            options = append(options, rules.WithRecordFindQuery("project_id", *project))
        }
        if *groupFilter != "" {
            options = append(options, rules.WithRecordFindQuery("security_group_id", *groupFilter))
        }
        value, err := service.SecurityRules.FindRecord(ctx, *rule, options...)
        if err != nil { log.Fatal(err) }
        if value == nil { printJSON(nil) } else { printJSON(value.Resource) }
    }
}
```

## GET kwargs와 fallback 필터

Source는 kwargs를 `cls.existing(id=identity, **params)`에 넣고 member fetch에도 그대로 넘깁니다. 따라서 `WithRecordFindQuery("ether_type", "IPv4")`는 Rule GET에서 `ether_type=IPv4`이고, fallback LIST에서만 `ethertype=IPv4`입니다. unknown kwargs도 GET query와 member seed에 남지만 LIST의 unknown query는 Source처럼 무시합니다. 응답에 해당 값이 없을 때 member Resource view에는 seed가 남으며 Wire에는 추가되지 않습니다. null은 query에서 생략하고 false·0·빈 문자열은 명시 값으로 남습니다. 배열·object query는 공통 Requests JSON-domain encoder의 반복 값 규칙을 사용합니다.

Group LIST는 `description`, `fields`, `name`, `stateful`, `project_id`, `tenant_id`, `revision_number`, `sort_dir`, `sort_key`, `limit`, `marker`, tag query와 `is_shared → shared`를 지원합니다. Rule LIST는 `description`, `direction`, `fields`, `protocol`, `remote_group_id`, `security_group_id`, `remote_address_group_id`, `port_range_min/max`, `remote_ip_prefix`, `project_id`, `tenant_id`, `revision_number`, `sort_dir`, `sort_key`, `limit`, `marker`, tag query와 `ether_type → ethertype`입니다. 두 Source query mapping에는 id도 있지만 Find의 positional id와 충돌하므로 별도 id kwargs는 HTTP 전에 거부합니다.

`any_tags/not_tags/not_any_tags`는 `tags-any/not-tags/not-tags-any`로 변환합니다. canonical client alias가 wire alias와 함께 있으면 client alias가 null이어도 우선하고 wire query를 생략합니다. 선언된 Body이지만 server mapping에 없는 `created_at`, `updated_at`, Group의 `security_group_rules`와 Rule의 `name`은 fallback에서 로컬 필터입니다. 명시 `name=null`은 Group에서는 자동 name hint와 server query를 생략하고, Rule에서는 nullable local name filter입니다. `all_projects`는 Resource.find가 별도로 소비하지만 이 Neutron class의 query mapping에는 없어 두 phase에서 전송하지 않습니다. 이 옵션으로 프로젝트 접근 권한이 넓어지지 않습니다.

로컬 필터는 공통 Python JSON equality와 recursive dictionary subset 동작을 사용합니다. truthy scalar에는 빈 object filter가 일치하고, nonempty object filter가 scalar의 `mapping.get`을 요구하면 actual page를 가진 오류입니다. 빈 dictionary·list·문자열·null·zero는 false입니다. 응답 decimal/exponent scalar의 dictionary truthiness와 Group bool descriptor는 Python binary64의 underflow/overflow도 반영합니다. 일반 JSON 숫자의 equality는 공유한 정확한 decimal 비교 정책을 사용하며 Python의 모든 float rounding/custom object 동작까지 재현하지 않습니다.

`WithRecordFindQuery` 순서는 member constructor의 canonical/wire Body alias overlay 순서로 보존합니다. 같은 key는 위치를 유지하고 마지막 값으로 갱신합니다. `RecordFindOpts.Query`와 callback으로 추가한 map 값에는 Python kwargs 순서가 없으므로 미등록 key를 정렬해 적용합니다. 복수 local 필터도 정렬된 key 순서로 평가합니다. 단일 canonical name을 사용하면 alias 충돌 순서를 신경 쓸 필요가 없습니다.

## Source view와 실제 응답

`SecurityGroupRecord`·`SecurityGroupRuleRecord`의 `Resource`에는 선언된 canonical Body와 inherited `if_match`, nullable `location`, unknown extension이 있습니다. 미수신 declared field는 null, 미수신 tags만 `[]`입니다. `project_id`가 없을 때만 `tenant_id`를 사용하며 명시 null은 유지합니다. Group `stateful`은 Source에서 untyped이므로 false·0·문자열도 강제 bool 변환하지 않습니다. `is_shared`는 bool descriptor, `security_group_rules`는 list descriptor이며 중첩 rule을 native typed model로 강제 변환하지 않습니다. Rule port range와 두 resource의 revision은 공통 Source int descriptor를 사용하고 nullable port 값·bool subtype·큰 정수·Unicode digit 변환을 보존합니다.

`Wire`에는 실제 응답 row와 원래 timestamp·숫자·unknown keys가 있습니다. `Envelope`는 member 전체 응답 또는 선택된 실제 list page의 본문입니다. Header·StatusCode와 각 view/Wire 메타데이터는 실제 응답의 독립 복사입니다. 실제 응답에 id가 없으면 member view에만 입력 identity를 seed하고 list row에는 합성하지 않습니다. explicit null·숫자 id는 보존하며 비교에 사용할 문자열 id가 없어도 name match는 가능합니다. 응답 alias는 JSON member 순서의 마지막 값이 우선합니다.

Source member translator는 응답 HTTP Header를 `if_match` list descriptor에 적용합니다. Header의 wire 별칭 `if-match`는 대소문자를 구분하지 않고, underscore canonical `if_match`는 정확한 key만 descriptor입니다. `IF_MATCH`는 constructor/LIST row에서 unknown Body extension이고 실제 HTTP Header에서는 미선언 Header로 남습니다. LIST는 `cls.existing(**row)`로 row의 `if_match` 또는 대소문자 혼합 `if-match`를 입력 순서로 읽고 page HTTP Header를 Resource descriptor에 합성하지 않습니다. record의 actual Header에는 page Header가 항상 남습니다. `self`는 view에서 제거하며 Wire에는 남습니다. LIST row의 cls/connection/microversion/_synchronized는 Source constructor와 충돌하므로 actual page를 가진 오류입니다. Source의 mutable Resource/class/subclass/cache/Connection location 전체 동작은 별도 공통 SDK 범위이며 location은 null입니다.

member의 singular envelope 또는 flat object를 받아들입니다. Source처럼 body 없는 204·malformed JSON syntax에서는 seed view를 유지하고 Wire는 nil입니다. 유효 JSON의 null/scalar/array 또는 null singular envelope는 actual HTTP proof를 가진 오류입니다. Go는 잘못된 UTF-8도 오류로 처리합니다. LIST는 plural array 또는 singleton object를 받아들이지만 null row와 body 없는 성공은 정상적인 빈 목록으로 바꾸지 않습니다.

## Pagination·옵션·오류

첫 일치 뒤에도 모든 관찰된 page를 검색합니다. 늦은 중복은 `resource.ErrAmbiguous`, 후속 HTTP·Read·Close·decode 실패는 원래 오류이며 첫 일치를 반환해 숨기지 않습니다. 성공한 전체 누락만 기본 nil,nil이고 strict는 `resource.ErrNotFound`입니다. 억제한 최초 member rejection을 논리적 missing cause에 섞지 않습니다. clean 400/403/404 외에 401, callback·source·취소·physical body 원인이 섞인 오류는 fallback하지 않습니다.

공통 REST pager로 plural links·top-level next·HTTP Link·short-page marker fallback·empty page 종료와 singleton row를 처리합니다. `/v2.0/security-groups`와 `/v2.0/security-group-rules`는 해당 고정 collection의 versioned 상대 경로 alias로 허용합니다. cross-origin·collection/filter 변경·cursor loop를 차단하며 actual page를 오류에 보존합니다. dictionary links는 Go 호환 확장입니다. marker는 반환 view와 독립인 실제 wire string id를 사용합니다. unsafe raw pagination 값은 shared pager에서 거부합니다.

`WithRecordFindLimit/Marker/MaxItems/Paginated`는 fallback만 제어하는 concrete Go 옵션입니다. MaxItems는 로컬 필터 이전 raw 행 수이며, limit 생략·wire zero일 때 최초 hint를 넣습니다. cap/single-page는 next를 해석하기 전에 종료합니다. 이 옵션을 선택하면 관찰하지 않은 page의 중복까지 없다고 보장하지 않습니다. 기본은 무제한 전체 검색입니다. Source의 raw limit/marker kwargs를 GET에도 보내려면 `WithRecordFindQuery`를 사용합니다.

두 class는 `_max_microversion=None`이므로 discovery를 수행하지 않습니다. Source Resource.find처럼 `WithRecordFindMicroversion`은 member GET에만 적용하며 fallback LIST는 캡처한 client의 기본 version을 사용합니다. GET과 LIST 각각의 protocol header 값 또는 부재를 retry·재인증·physical request 동안 고정합니다. GET override와 원본 version Header가 달라도 phase별로 독립 복사하므로 선택한 원래 client는 바꾸지 않습니다. LIST는 Source의 `Accept: application/json` 기본값을 사용하고 명시적 `WithRecordFindHeaders`의 Accept만 우선합니다. `WithRecordFindHeaders`는 두 phase에 적용하는 owned Go 편의 기능이며 credentials·route·protocol override는 허용하지 않습니다. `RecordFindOpts`·Query·Header·pointer·retained callback 값은 독립 복사합니다. Native clean retry와 token 재인증은 원래 route/query/body를 유지합니다. retry가 Authorization·X-Service-Token·Host·Cookie·framing/hop header 또는 다른 서비스의 Nova/Cinder version Header를 추가하거나 selected version을 OmitHeaders로 지우면 추가 HTTP 전에 최초 rejection과 callback cause를 보존한 terminal 오류입니다.

API는 string identity와 concrete options를 받습니다. raw ignore_missing/name_or_id/session/cls/id/self/connection/_synchronized와 base_path/list_base_path/requires_id/resource_response_key/skip_cache/error_message/headers/microversion/paginated/max_items 같은 Resource/session controls는 임의 query로 받지 않습니다. 지원되는 기능은 named 옵션으로 설정하고 cache·custom route·response-key·arbitrary Python Resource/session 처리는 공통 Resource/cache/session 검토 항목에 남깁니다. Identity의 blank/control/invalid UTF-8, typed 음수 limit/cap, malformed JSON seed와 auth header는 HTTP 전에 오류입니다.

계약 테스트는 기존 native transport recorder와 Gophercloud public testhelper 기반 `internal/testcloud`를 재사용합니다. Read/Close·취소·sticky source 변경·retry 오류는 실제 body/status/header/cause를 보존하고 lookup을 재전송하거나 missing으로 바꾸지 않는지 검증합니다. fixture race와 문서 consumer build는 실제 인증된 Neutron endpoint·서버 정책 검증과 별개입니다.
