# Keystone Trust: openstacksdk와 Go

Pinned openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 Trust proxy 5개를 `/OS-TRUST/trusts`에 연결합니다. 일반 Credential, EC2 Credential, Application Credential과 다른 Keystone 위임 리소스입니다. 기존 native Trust CRUD·role APIs는 그대로 사용할 수 있습니다.

| openstacksdk | Go Source 프로필 |
|---|---|
| `conn.identity.create_trust(**attrs)` | `service.Trusts.CreateRecord(ctx, options...)` |
| `conn.identity.get_trust(id)` | `service.Trusts.GetRecord(ctx, id)` |
| `conn.identity.trusts(**query)` | `service.Trusts.ListRecords(ctx, options...)` |
| `conn.identity.find_trust(name_or_id)` | `service.Trusts.FindRecord(ctx, nameOrID)` |
| `conn.identity.delete_trust(id)` | `service.Trusts.DeleteRecord(ctx, id)` |

```python
import openstack

conn = openstack.connect(cloud="dev")
created = conn.identity.create_trust(
    trustor_user_id=trustor_id,
    trustee_user_id=trustee_id,
    project_id=project_id,
    is_impersonation=False,
    roles=[{"name": "member"}],
)
for trust in conn.identity.trusts(trustor_user_id=trustor_id):
    print(trust.id, trust.is_impersonation)
found = conn.identity.find_trust(created.id, ignore_missing=False)
conn.identity.delete_trust(created.id, ignore_missing=False)
```

## 독립 Go main

[설치 안내](../../docs/install.md)에 따라 모듈을 준비하고 아래 코드를 `main.go`로 저장합니다. `go run . -cloud dev -trustor TRUSTOR_USER_ID`는 해당 위임자의 Trust 목록을 읽습니다. `-id TRUST_ID`는 GET과 strict Find를 추가합니다. 생성과 삭제는 각각 `-create`와 `-delete TRUST_ID`를 명시해야 실행됩니다.

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
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts"
)

func printTrust(value *trusts.TrustRecord) {
    if value == nil { fmt.Println("null"); return }
    data, err := json.MarshalIndent(value.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    trustor := flag.String("trustor", "", "trustor user ID")
    trustee := flag.String("trustee", "", "trustee user ID for creation")
    project := flag.String("project", "", "project ID for creation")
    role := flag.String("role", "member", "role name for creation")
    id := flag.String("id", "", "Trust ID to get and find")
    create := flag.Bool("create", false, "create a Trust")
    remove := flag.String("delete", "", "Trust ID to delete")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    api := service.Trusts
    options := []trusts.RecordListOption{trusts.WithRecordListMaxItems(100)}
    if *trustor != "" {
        options = append(options, trusts.WithRecordListTrustorUserID(*trustor))
    }
    for value, err := range api.ListRecords(ctx, options...) {
        if err != nil { log.Fatal(err) }
        printTrust(value)
    }
    if *id != "" {
        value, err := api.GetRecord(ctx, *id)
        if err != nil { log.Fatal(err) }
        printTrust(value)
        found, err := api.FindRecord(ctx, *id, trusts.WithRecordFindIgnoreMissing(false))
        if err != nil { log.Fatal(err) }
        printTrust(found)
    }
    if *create {
        value, err := api.CreateRecord(ctx,
            trusts.WithRecordCreateField("trustor_user_id", *trustor),
            trusts.WithRecordCreateField("trustee_user_id", *trustee),
            trusts.WithRecordCreateField("project_id", *project),
            trusts.WithRecordCreateField("is_impersonation", false),
            trusts.WithRecordCreateField("roles", []map[string]string{{"name": *role}}))
        if err != nil { log.Fatal(err) }
        printTrust(value)
    }
    if *remove != "" {
        if err := api.DeleteRecord(ctx, *remove,
            trusts.WithRecordDeleteIgnoreMissing(false)); err != nil { log.Fatal(err) }
    }
}
```

## Body와 descriptor 계약

`WithRecordCreateField` 또는 `WithRecordCreateOpts`는 선언된 속성만 `trust` envelope로 보냅니다. `is_impersonation`과 wire 이름 `impersonation`은 같은 필드입니다. 순서가 있는 필드 옵션은 마지막 값이 이기고, bulk 옵션 뒤의 필드 옵션도 정상적으로 덮어씁니다. 미등록 attrs는 pinned Resource처럼 버립니다. unordered bulk map에 두 alias가 동시에 있으면 명확하지 않은 순서를 선택하지 않고 HTTP 전에 `resource.ErrInvalidOption`을 반환합니다. 응답에서 두 alias가 동시에 나타나면 실제 HTTP 증거를 가진 오류로 반환합니다. 이 충돌 정책은 명시적 Go 차이입니다.

생략한 속성은 request에 추가하지 않으며 null·false·0·빈 string·array·object는 raw JSON 타입 그대로 유지합니다. Source `Resource.new`가 raw attrs를 저장하는 동작과 같이 bool descriptor도 create wire 값을 강제 변환하지 않습니다. `roles`, 사용자 ID, `expires_at`, `remaining_uses`, `redelegation_count`는 pinned Source의 untyped Body descriptor입니다. native role struct·`time.Time`·int로 강제 변환하거나 사용자 ID를 조회하지 않습니다. 실제 Keystone의 필수 필드와 형식 검증은 서버가 수행합니다. 빈 Create도 `{"trust":{}}`로 POST합니다.

`TrustRecord.Resource`는 canonical 선언명으로 15개 속성을 제공합니다. `is_impersonation`과 `allow_redelegation`은 nonnull 값에 Python JSON-domain truthiness를 적용합니다. 예를 들어 wire `"false"`는 nonempty string이므로 true이고 `[]`, `{}`, 0은 false입니다. null은 null입니다. Source 주석의 False 설명과 달리 두 descriptor의 실제 default는 None이므로 누락하면 null입니다. 다른 descriptor와 location도 누락 시 null입니다.

`Wire`에는 원래 wire 이름·unknown attrs·정확한 큰 정수를 보존합니다. `Envelope`는 전체 HTTP body, `Header/StatusCode`는 실제 응답이며 모든 채널을 독립 복사합니다. GET의 literal ID와 Create attrs를 seed로 보관하고 실제 응답에서 제공한 선언 속성을 덮어씁니다. `trust` singular envelope가 있으면 unpack하고 없으면 root 객체를 사용합니다. 빈 본문·잘못된 JSON syntax는 Source처럼 seed를 유지하고 유효한 nonobject JSON은 실제 accepted 응답 증거가 있는 오류입니다.

## 목록·Find·삭제 기본값

`WithRecordListTrustorUserID`와 `WithRecordListTrusteeUserID`는 보통 쓰는 server query 옵션입니다. `WithRecordListQuery`는 이 두 이름을 공통 Requests query encoder로 처리하고 다른 canonical Body 속성을 local filter로 분류합니다. query null은 생략하고 array·object expansion과 Python bool spelling을 재사용합니다. `is_impersonation`은 canonical local filter지만 `impersonation`은 Source의 class Body 이름이 아니므로 local query에서 무시합니다. 다른 미등록 query도 무시합니다. local bool filter는 raw wire 값이 아닌 descriptor view를 비교합니다.

`WithRecordListLimit`, `WithRecordListMarker`, `WithRecordListMaxItems`, `WithRecordListPaginated(false)`는 소유한 typed 목록 제어입니다. iterator는 lazy하고 다시 순회할 때 새 Source snapshot을 사용합니다. max_items는 local filter 전에 관측한 raw row 수를 제한하고, 양수 cap은 limit 생략/0의 초기 query 힌트가 됩니다. Source의 array next link·top-level next·HTTP Link와 initial limit에 따른 short-page marker fallback을 재사용합니다. 소비자가 반환 ID/Wire를 바꿔도 다음 marker는 원본 page에서 가져옵니다. `links:{next:...}`는 별도의 Go 호환 확장입니다. Source의 임의 pagination/JMESPath/adapter 제어 kwargs 전체를 구현했다고 주장하지 않습니다.

Find는 먼저 literal member GET을 수행합니다. clean 400/403/404만 전체 목록 fallback을 허용하며 정확한 ID 또는 name으로 검색합니다. Trust의 query mapping에는 name이 없으므로 자동 wire name query를 만들지 않습니다. 중복은 `resource.AmbiguousError`, 전체 검색에서 없으면 기본 nil,nil 또는 `WithRecordFindIgnoreMissing(false)`의 `resource.NotFoundError`입니다. 앞 페이지에서 일치해도 후속 page 오류를 숨기지 않습니다.

Delete는 기본으로 clean 404만 무시합니다. `WithRecordDeleteIgnoreMissing(false)`는 원래 HTTP 오류를 반환합니다. Create/Get/List/Delete는 실제 200..399를 허용하며, List는 실제 `trusts` JSON 표현이 필요합니다. bodyless 204 List를 빈 목록으로 합성하지 않습니다.

## 공통 엔진·오류·검증 범위

SDK Source capture·REST request guard·rejected response observer·ListWithControl·FindIdentity·JSON filter·Requests encoder를 재사용합니다. 옵션은 raw JSON/map/header/pointer를 복사하고 retained callback pointer를 격리합니다. nil option·잘못된 known JSON/UTF-8·음수 typed limit/cap·보호된 auth header·잘못된 literal ID는 요청 전에 실패합니다. callback 오류와 동시에 발생한 취소/cause 또는 Source 변경도 유지합니다. ID는 한 URL segment로 한 번 escape하며 configured version을 보존하고 Trust에 인위적인 최대 microversion을 추가하지 않습니다.

accepted Read/Close 오류는 `resource.ResponseError`에 실제 status/body/header를 보존하고 rejected 응답은 native `gophercloud.ErrUnexpectedResponseCode`와 모든 physical read/close·취소·Source·retry callback 원인을 함께 보존합니다. 이 정책을 Create/Get/List/Delete 모두에 적용합니다. mixed 404는 삭제에서 무시하거나 Find fallback에 사용하지 않습니다. Read에서 Source가 변하고 Close에서 원복되어도 관측된 실패는 유지합니다. native retry는 정상 재시도를 유지하지만 SDK가 소유한 method/URL/JSON을 바꾸면 다음 물리 요청 전에 실패합니다.

이 프로필은 literal ID·fresh declared view를 제공하며 Python mutable Resource/Munch 객체 입력·same-instance dirty state, Connection location, cache/session/subclass hooks와 임의 dynamic adapter/microversion을 모두 제공한다고 주장하지 않습니다. JSON local filter는 bool과 number를 구분하고 nonobject nested 비교를 false로 처리하는 명시적 Go 정책을 유지합니다. Source에 Trust update proxy가 없어 새 Update API를 만들지 않았습니다.

[계약 테스트](trusts/python_records_test.go)는 기존 `nativeTrustAPI`와 Gophercloud public testhelper를 사용하는 `testcloud`, shared SDK engines를 재사용합니다. create raw attrs·aliases·descriptor defaults, member translation, server/local query, cap/paging/lazy ownership, GET-first 전체 검색, strict/default missing, 옵션/callback 소유권, 실제 accepted/rejected Read/Close/cancel/Source/retry errors를 검증합니다. 독립 문서 main은 빌드 검증 대상으로 제공하며 실제 인증된 Keystone 배포 결과와 별도로 기록합니다.
