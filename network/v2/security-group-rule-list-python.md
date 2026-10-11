# Security group rule 목록: Python과 Go

고정 기준은 openstacksdk [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L6820-L6842), Gophercloud v2.15.0, Neutron 27.0.0입니다. `SecurityRules.ListRecords`는 named `security_group_rules(**query)` 목록의 Source descriptor·query·local filter·pagination 계약을 제공합니다. 기존 native typed List와 generated Collection을 다시 완료 API로 집계하지 않습니다.

| Python | Go | 동작 |
|---|---|---|
| `conn.network.security_group_rules(**query)` | `network.SecurityRules.ListRecords(ctx, options...)` | lazy full list, 기본 `Accept: application/json` |
| `list(...)` | `AllRecords(ctx, options...)` | 전체 수집, 성공한 빈 결과는 nonnil slice, 오류 시 partial slice 폐기 |
| `security_group_id=...`, `direction=...` | `WithRecordListQuery(key, value)` | Source의 실제 server query |
| `ether_type=...`, `any_tags=...` | 같은 query option | `ethertype`, `tags-any`로 매핑 |
| `name=...`, `created_at=...`, `updated_at=...` | 같은 query option | declared nonquery Body 필터로 로컬 평가 |
| `limit`, `marker`, `max_items`, `paginated=False` | `WithRecordListLimit/Marker/MaxItems/Paginated` | 서버 hint·marker, 필터 전 raw cap, 단일 page |

아래 Python은 기존 OpenStack 인증 환경으로 실행하는 독립 예제입니다.

```python
import argparse
import json
import os
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default=os.getenv("OS_CLOUD"))
    parser.add_argument("--group", default="")
    parser.add_argument("--direction", default="")
    parser.add_argument("--name", default="")
    parser.add_argument("--page-size", type=int, default=-1)
    parser.add_argument("--max-items", type=int, default=0)
    parser.add_argument("--single-page", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    query = {"max_items": args.max_items, "paginated": not args.single_page}
    if args.group:
        query["security_group_id"] = args.group
    if args.direction:
        query["direction"] = args.direction
    if args.name:
        query["name"] = args.name
    if args.page_size >= 0:
        query["limit"] = args.page_size
    for rule in conn.network.security_group_rules(**query):
        print(json.dumps(rule.to_dict(), ensure_ascii=False))


if __name__ == "__main__":
    main()
```

다음 Go 전체 코드를 `main.go`에 저장해 실행할 수 있습니다.

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "os"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules"
)

func main() {
    cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud")
    group := flag.String("group", "", "server security_group_id filter")
    direction := flag.String("direction", "", "server direction filter")
    name := flag.String("name", "", "local inherited name filter")
    pageSize := flag.Int("page-size", -1, "server limit; -1 omits it")
    maxItems := flag.Int("max-items", 0, "raw cap before local filtering")
    single := flag.Bool("single-page", false, "read first page only")
    flag.Parse()
    if err := run(*cloud, *group, *direction, *name, *pageSize, *maxItems, *single); err != nil { log.Fatal(err) }
}

func run(cloud, group, direction, name string, pageSize, maxItems int, single bool) error {
    if pageSize < -1 || maxItems < 0 { return fmt.Errorf("invalid page-size or max-items") }
    ctx := context.Background()
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    network, err := conn.NetworkV2(ctx)
    if err != nil { return err }
    options := []rules.RecordListOption{
        rules.WithRecordListMaxItems(maxItems),
        rules.WithRecordListPaginated(!single),
    }
    if pageSize >= 0 { options = append(options, rules.WithRecordListLimit(pageSize)) }
    for _, filter := range []struct{ key, value string }{
        {"security_group_id", group}, {"direction", direction}, {"name", name},
    } {
        if filter.value != "" { options = append(options, rules.WithRecordListQuery(filter.key, filter.value)) }
    }
    out := json.NewEncoder(os.Stdout)
    for record, readErr := range network.SecurityRules.ListRecords(ctx, options...) {
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

`go run . --cloud dev --group SECURITY_GROUP_ID --page-size 100`처럼 실행합니다. `rules.New(service.RawClient()).ListRecords`로도 같은 concrete API를 쓸 수 있으며 caller builder·extension interface를 구현할 필요가 없습니다.

[Source SecurityGroupRule](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/security_group_rule.py)는 NetworkResource·TagMixinNetwork를 상속합니다. 기존 `SecurityGroupRuleRecord`를 재사용하고 Source MRO의 Body 18개, Header `if_match`, computed `location`을 제공합니다. missing Body는 null이고 tags는 빈 list입니다. port_range_min/max와 revision_number만 int descriptor로 변환합니다. bool은 Python int subclass이므로 false/true 그대로, float은 binary64 값으로 truncate, digit string은 Unicode decimal 변환, signed/공백/ordinary nonnumeric 문자열·container는0입니다. digit-only 비decimal 문자는 실제 page의 descriptor 오류입니다. protocol·direction·date·remote selector는 untyped입니다.

`ethertype`를 canonical `ether_type`로 표시하고 row에 canonical/wire가 함께 있으면 Source의 행 입력 순서에 따라 마지막 값이 이깁니다. project_id가 생략됐을 때만 tenant_id를 fallback alias로 쓰며 explicit project_id null은 유지합니다. tags와 row Header scalar는 Source list descriptor처럼 단일 요소 list로 감쌉니다. GET page의 `If-Match`는 각 row의 Header descriptor로 합성하지 않습니다. constructor에 들어온 canonical `if_match`·case-insensitive hyphen `if-match`만 소비하며 `IF_MATCH` 같은 별도 unknown Body는 유지합니다.

Resource는 실제 declared/default view, Wire는 unknown·self·원래 숫자와 remote spelling을 포함한 원본 row입니다. Source reserved self는 Resource에서 제외합니다. 서버가 반환한 forged location은 Wire와 전체 Envelope에 남기고 Resource에는 owned null을 제공합니다. 전체 Python Connection/current location·mutable Resource/cache/dirty-state·runtime subclass/session은 별도 SDK-R1/C1/S1 범위입니다. Resource·Wire·Envelope·Header와 다른 행, clone은 독립 소유입니다. 각 receipt는 목록 page의 실제 status/body/header이며 local filter나 descriptor 오류에도 그 page 증거가 유지됩니다.

server query는 Source QueryParameters의 limit/marker, description/direction/fields/id/protocol/remote_group_id/security_group_id/remote_address_group_id/port_range_min/max/remote_ip_prefix/revision_number/project_id/tenant_id/sort_dir/sort_key/ether_type와 tag aliases입니다. canonical alias가 supplied wire spelling보다 우선하며 null은 wire query에서 생략, 배열은 repeated Requests 값, bool은 `True`/`False`입니다. server filter를 SDK가 또 local 검사하지 않습니다. name·created_at·updated_at 같은 declared nonquery Body만 로컬 필터입니다. unknown·Header·all_projects는 ignored list kwargs이고 marshal 전에 버립니다. all_projects가 token의 읽기 권한을 늘리지 않습니다.

[Source Proxy._list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866-L916)의 `jmespath_filters`는 unknown query가 아니라 named control입니다. Source는 truthy string일 때 deprecated JMESPath expression으로 목록 데이터를 변환해 임의 결과를 반환할 수 있습니다. 현재 Go record iterator는 이 변환을 지원하지 않으며 `jmespath_filters` key의 **presence**를 SDK-S1 경계로 명시적으로 거부합니다. 값이 null·false·빈 문자열이어도 `resource.ErrInvalidOption`을 반환하고, 값의 JSON marshaling·HTTP 요청·응답 증거 생성을 하지 않습니다. Source가 falsey 값을 받으면 변환 없이 원래 목록을 반환하는 동작과도 이 명시적 Go guard는 다릅니다. factory·bulk `RecordListOpts.Query`·custom callback 모두 같은 계약을 적용합니다. 선언 Body의 local filter와 server query, 기존 wire aliases는 계속 지원합니다.


concrete `RecordListOpts`와 factory는 nested raw JSON·header map·version pointer·option slice·callback 결과를 복사합니다. SDK는 첫 custom callback과 앞선 callback의 nil reset 뒤에도 Query·Headers map을 준비하므로 직접 key를 쓸 수 있습니다. callback에 남겨둔 map의 사후 변경은 실행 중 query·Header·local filter를 바꾸지 않습니다. 순차 query 옵션의 local filter 순서와 nested raw dict 순서를 보존하며 bulk Go map의 새 key는 정렬합니다. recursive dict subset은 falsey actual을 제외하고 truthy scalar+empty dict는 맞으며 nonempty dict+truthy scalar는 실제 page 오류입니다. bool-number equality와 decimal underflow truthiness를 공통 engine으로 처리합니다. 수치 equality는 SDK-C1 exact-decimal 정책으로 CPython binary64 rounding과 완전 동치가 아닙니다. 예를 들어 actual9007199254740992는 rawfilter9007199254740993.0과 다릅니다.

omitted limit은 요청하지 않고 explicit0은 limit=0입니다. cap0은 무제한, 양수 cap은 필터 제외 행도 세며 falsey limit에 초기 server hint를 제공합니다. consumer break/cap/single page 뒤의 row·next는 평가하지 않습니다. inherited full pager는 plural/singleton envelope, body/plural/HTTP links, exact `/v2.0/security-group-rules` alias와 short nonempty page의 truthy-limit marker fallback, empty page 종료를 제공합니다. body의 falsey decimal next는 종료합니다. dictionary link는 Go compatibility이며 Source HTTP Link의 `uri` 접근 문제를 Go에서 RFC형으로 처리합니다. 동일 origin/collection/filter를 유지하고 cycle·unsafe continuation은 실제 page 오류입니다. Source의 임의 endpoint/collection retargeting은 허용하지 않습니다. empty/malformed/null/wrong-shape/invalid UTF8도 실제 accepted page 오류입니다.

selected Neutron version 또는 부재를 유지하고 explicit `WithRecordListMicroversion`이 해당 목록 실행을 override합니다. Source class ceiling이 없어 discovery·임의 upper bound를 추가하지 않습니다. Source generic status<400과 달리 공통 Engine은 representation200..399만 허용하며 transport-exposed199는 원본 native error입니다. clean400..599는 실제 native method/URL/status/body/header를 유지하고 list는 404 ignore/find fallback을 하지 않습니다. physical Read/Close/mixed/cancel/sticky Source 변경은 원인을 함께 보존하며 body를 한 번 닫고 **native RetryFunc 호출 전** 종료합니다. clean retry와 token renewal은 유지합니다. 물리적 오류가 섞인401은 재인증 hook을 호출하지 않으며 native `*gophercloud.ErrUnableToReauthenticate`의 public `ErrOriginal`에서 실제401 method/URL/status/body/Header, `ErrReauth`에서 Read·Close·Source·cancel 원인을 확인합니다. 이 native wrapper는 두 필드를 Unwrap하지 않으므로 `errors.As`로 pointer를 얻어 각각 검사합니다. auth/framing/foreign Nova·Cinder version/selected network version or absence·bodyless route·기존 명시 header는 고정합니다. 일반 새 retry header는 허용하지만 If-Match·If-None-Match·If-Modified-Since·If-Unmodified-Since는 선택한 값 또는 미지정 상태를 고정합니다. clean native retry callback이 미지정 조건을 주입하거나 명시한 값을 바꾸거나 OmitHeaders로 제거하면 두 번째 HTTP 요청 전에 actual rejection과 callback cause를 포함해 거부합니다. 명시한 조건을 유지한 정상 retry는 계속 허용됩니다. OkCodes 확장 뒤 재전송은 가능해도 SDK original acceptance 밖 status는 성공으로 바뀌지 않습니다.

[Neutron 27.0.0 policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/security_group.py#L228-L247)는 collection/member `get_security_group_rule`에 `ADMIN_OR_SG_OWNER_READER`, project scope를 적용합니다. SG owner는 rule의 project_id를 SDK가 추정한 값과 동일한 개념이 아닙니다. [stock DB](https://github.com/openstack/neutron/blob/27.0.0/neutron/db/securitygroups_db.py#L1021-L1050)는 filters/default project에 맞는 후보를 조회한 뒤 [controller](https://github.com/openstack/neutron/blob/27.0.0/neutron/api/v2/base.py#L292-L331)가 각 row의 정책을 검사해 허용된 행만 반환합니다. 따라서 성공한 빈 목록도 정상이며 전체 cloud 권한 승인 증거는 아닙니다. SDK는 권한을 추정·확대하지 않고 실제 서버 결과를 보존합니다. 배포의 정책 override·tag/filter 확장 지원·live cloud acceptance는 fixture 결과와 별도입니다.

[테스트 13그룹](extensions/security/rules/python_list_contracts_test.go)은 기존 nativeRuleRecorder/transport와 public Gophercloud testhelper 기반 `internal/testcloud`, 공유 neutronfind.Engine.List를 재사용합니다. 신규 transport/pager 구현이나 매 API용 HTTP server framework를 추가하지 않습니다. root 집중/full race와 독립 Go main compile은 통합 단계에서 검증합니다.
