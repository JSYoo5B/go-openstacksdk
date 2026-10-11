# Keystone v3 ApplicationCredential / AccessRule Python proxy

`conn.IdentityV3(ctx).ApplicationCredentials`는 사용자에게 속한 application credential과 access rule을 제공합니다. 기존 generated `Create/Get/List/Delete/GetAccessRule/ListAccessRules/DeleteAccessRule`은 Gophercloud의 타입·상태코드 계약을 유지합니다. 아래 `Record` 메서드는 Python proxy의 기본 조회·필터·검색·삭제 정책과 원래 JSON을 사용하는 별도 API입니다.

| openstacksdk | Go API | 고정 사용자 scope |
|---|---|---|
| `application_credentials(user, **query)` | `ListCredentialRecords(ctx, userID, options...)` | `scope.ListRecords(ctx, options...)` |
| `create_application_credential(user, name, **attrs)` | `CreateCredentialRecord(ctx, userID, name, options...)` | `scope.CreateRecord(ctx, name, options...)` |
| `get_application_credential(user, credential)` | `GetCredentialRecord(ctx, userID, credentialID, options...)` | `scope.GetRecord(ctx, credentialID, options...)` |
| `find_application_credential(user, name_or_id, ignore_missing=True, **query)` | `FindCredentialRecord(ctx, userID, nameOrID, options...)` | `scope.FindRecord(ctx, nameOrID, options...)` |
| `delete_application_credential(user, credential, ignore_missing=True)` | `DeleteCredentialRecord(ctx, userID, credentialID, options...)` | `scope.DeleteRecord(ctx, credentialID, options...)` |
| `access_rules(user, **query)` | `ListAccessRuleRecords(ctx, userID, options...)` | `rules.ListRecords(ctx, options...)` |
| `get_access_rule(user, access_rule)` | `GetAccessRuleRecord(ctx, userID, ruleID, options...)` | `rules.GetRecord(ctx, ruleID, options...)` |
| `delete_access_rule(user, access_rule, ignore_missing=True)` | `DeleteAccessRuleRecord(ctx, userID, ruleID, options...)` | `rules.DeleteRecord(ctx, ruleID, options...)` |

`scope := api.InUser(ctx, resource.ID(userID))`와 `rules := api.AccessRules(ctx, resource.ID(userID))`는 사용자 ID를 한 번 고정합니다. 명시적인 사용자 ID는 별도 사용자 GET 없이 `/users/{user_id}/application_credentials`와 `/users/{user_id}/access_rules`에 사용합니다. Go의 `resource.Name`으로 부모 이름을 찾는 기능은 추가 편의 기능입니다. AccessRule에는 Python proxy create/find/update 메서드가 없으며 이 묶음에도 추가하지 않습니다.

```python
credentials = conn.identity.application_credentials(user_id, name="ci", max_items=100)
created = conn.identity.create_application_credential(
    user_id, "ci", description="", unrestricted=False, roles=[{"name": "member"}])
found = conn.identity.find_application_credential(user_id, "ci")
rule = conn.identity.get_access_rule(user_id, rule_id)
# 필요한 경우:
# conn.identity.delete_application_credential(user_id, created.id)
# conn.identity.delete_access_rule(user_id, rule.id, ignore_missing=False)
```

## 독립 Go main

[설치 안내](../../docs/install.md)대로 모듈을 준비하고 아래를 `main.go`에 저장합니다. `go run . -cloud dev -user USER_ID -name ci`는 credential 목록·이름 검색·access rule 목록을 조회합니다. `-credential`과 `-rule`은 member GET을 추가하며, `-create`·`-delete-credential`·`-delete-rule`을 명시한 경우에만 해당 변경을 수행합니다.

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
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/applicationcredentials"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func printRecord(v *applicationcredentials.ApplicationCredentialRecord) {
    if v == nil { fmt.Println("null"); return }
    data, err := json.MarshalIndent(v.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    user := flag.String("user", "", "literal Keystone user ID")
    name := flag.String("name", "ci", "exact credential name or ID")
    credential := flag.String("credential", "", "credential ID to GET")
    rule := flag.String("rule", "", "access rule ID to GET")
    create := flag.Bool("create", false, "create an application credential")
    deleteCredential := flag.String("delete-credential", "", "credential ID to delete")
    deleteRule := flag.String("delete-rule", "", "access rule ID to delete")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    api := service.ApplicationCredentials
    scope, err := api.InUser(ctx, resource.ID(*user))
    if err != nil { log.Fatal(err) }
    rules, err := api.AccessRules(ctx, resource.ID(*user))
    if err != nil { log.Fatal(err) }
    for v, err := range scope.ListRecords(ctx,
        applicationcredentials.WithRecordListFilter("name", *name),
        applicationcredentials.WithRecordListMaxItems(100)) {
        if err != nil { log.Fatal(err) }
        printRecord(v)
    }
    found, err := scope.FindRecord(ctx, *name)
    if err != nil { log.Fatal(err) }
    printRecord(found)
    if *credential != "" {
        v, err := scope.GetRecord(ctx, *credential)
        if err != nil { log.Fatal(err) }
        printRecord(v)
    }
    for v, err := range rules.ListRecords(ctx) {
        if err != nil { log.Fatal(err) }
        printRecord(v)
    }
    if *rule != "" {
        v, err := rules.GetRecord(ctx, *rule)
        if err != nil { log.Fatal(err) }
        printRecord(v)
    }
    if *create {
        v, err := scope.CreateRecord(ctx, *name,
            applicationcredentials.WithRecordCreateField("description", ""),
            applicationcredentials.WithRecordCreateField("unrestricted", false),
            applicationcredentials.WithRecordCreateField("roles", []map[string]string{{"name": "member"}}))
        if err != nil { log.Fatal(err) }
        printRecord(v)
    }
    if *deleteCredential != "" {
        if err := scope.DeleteRecord(ctx, *deleteCredential); err != nil { log.Fatal(err) }
    }
    if *deleteRule != "" {
        if err := rules.DeleteRecord(ctx, *deleteRule,
            applicationcredentials.WithRecordDeleteIgnoreMissing(false)); err != nil { log.Fatal(err) }
    }
}
```

## 옵션과 결과 계약

Create는 명시한 Body 필드만 singular envelope 안에 보냅니다. `WithRecordCreateField`나 `WithRecordCreateOpts`로 null·false·빈 문자열·빈 list를 구분하고, 생략한 `unrestricted`는 보내지 않습니다. `name`과 `user_id`는 명시 인자이므로 attrs에서 다시 지정하면 HTTP 전에 실패합니다. 다른 미등록 attrs는 버립니다. JSON 타입을 지정하지 않은 Python descriptor처럼 `expires_at`, `roles`, `access_rules`, `user`를 `time.Time`·native role/access rule 모델에 강제로 맞추지 않습니다.

List의 `WithRecordListLimit`·`WithRecordListMarker`만 wire query로 보내고, `WithRecordListFilter`는 해당 Resource에 선언된 Body 필드에 적용하는 JSON 필터입니다. 미등록 필터는 버립니다. 이름은 server query에 넣지 않으며, 중첩 object 필터는 부분 일치입니다. `WithRecordListMaxItems(n)`은 **필터 적용 전 관찰한 raw row**를 센다는 Source 기본 동작을 유지합니다. limit을 생략하거나 0으로 주면 양수 max_items를 초기 limit 힌트로 사용합니다. `WithRecordListPaginated(false)`는 첫 페이지만 읽습니다. iterator는 HTTP 전에 생성되며, 반복할 때마다 새 snapshot과 요청을 사용합니다. 소비자가 iteration을 중단하면 다음 페이지를 요청하지 않습니다.

Find는 직접 member GET을 먼저 시도합니다. 이 GET은 Source처럼 `user_id`를 query에도 넣으며, 호출자가 지정한 string query도 함께 보냅니다. 깨끗한 400·403·404만 전체 list fallback을 허용합니다. 기본 이름 query 힌트는 없으며 fallback은 정확한 ID 또는 name으로 비교하고, 중복은 `resource.AmbiguousError`, 전체 목록에서 찾지 못하면 기본값 nil,nil입니다. `resource.WithIdentityFindIgnoreMissing(false)`는 부재를 오류로 바꿉니다. `resource.WithIdentityFindQuery`의 Body 필드는 fallback에서 string 값의 local filter가 되고 limit/marker만 wire query가 됩니다. Direct GET이 응답에서 생략한 id·Body query 값은 호출 seed에서 유지합니다. `id`·`user_id` query로 고정 인자를 덮어쓸 수 없습니다.

Delete는 기본 `ignore_missing=true`이며, 깨끗한 404만 억제합니다. `WithRecordDeleteIgnoreMissing(false)`는 원래 404를 유지합니다. read·close·transport·retry callback·취소 오류와 섞인 404는 숨기지 않습니다. Get/Create/Delete/List는 실제 200..399를 허용합니다. Delete는 opaque 본문을 JSON으로 해석하지 않습니다. Get/Create의 빈 본문·잘못된 JSON syntax는 Source처럼 seed를 유지한 view로 허용하며, 유효한 nonobject JSON은 실제 응답을 보존한 오류입니다. List는 plural envelope의 JSON 표현이 필요하며 singleton object도 허용합니다.

`ApplicationCredentialRecord`와 `AccessRuleRecord`는 같은 owned container입니다. `Resource`는 선언된 descriptor view, `Wire`는 원래 응답 객체, `Envelope`는 전체 원문 HTTP body, `Header`·`StatusCode`는 실제 응답 증거입니다. Credential view는 Body 11개와 `user_id`·`location`, AccessRule view는 Body 6개와 `user_id`·`location`을 포함합니다. 누락된 descriptor는 null이고 `unrestricted`만 JSON truthiness를 적용합니다. fixed `user_id`는 응답 URI 값보다 우선하며 Wire에는 원래 값이 남습니다. 알 수 없는 필드·큰 정수·날짜 원문은 Wire에 유지합니다. 각 채널의 필드와 header는 독립 복사하며, 반환 view를 변경해 후속 요청의 사용자나 pagination marker를 바꿀 수 없습니다.

## Go profile과 검증 경계

옵션은 library 소유 concrete 함수로 구성합니다. `WithRecordCreateOpts`·`WithRecordListOpts`는 전체 정책을 교체하며, map·raw JSON·pointer를 복사합니다. callback 완료 뒤에도 전달 pointer를 보관해 정책을 바꿀 수 없습니다. callback 오류와 동시에 발생한 취소·Source 변경도 cause를 함께 보존합니다. 요청별 client/header snapshot과 공통 guard를 재사용하며 endpoint·provider·서비스·microversion이 변경되면 오류입니다. 보호된 인증·transport header는 호출 옵션으로 덮어쓸 수 없습니다.

Python의 Resource/Munch 객체 인자·cache/session hook·subclass mutation·Connection 기반 location·임의 adapter/microversion 선택·임의 route·JMESPath와 모든 동적 kwargs를 제공하는 범위는 별도입니다. 이 Go view의 location은 null이고 member ID는 유효한 UTF-8 문자열을 URL segment로 한 번 escape합니다. Find는 공통 string query 옵션을 사용하며 JSON object/list 필터, pagination/transport 제어 kwargs와 all_projects 등 서비스가 지원하지 않는 공통 Find 옵션은 받지 않습니다. JSON 필터는 bool과 number를 구분하고 malformed nested 비교를 false로 처리합니다.

Pagination은 기존 고정 collection REST 정책을 재사용합니다. Source의 array next link·top-level next·HTTP Link·양수 initial limit의 marker fallback과 빈 page 종료를 지원합니다. 서비스에서 흔히 사용하는 `links:{next:"URL"}`도 지원하지만 **pinned Python generic paginator가 이 dictionary 형태를 인식하지 못하는 부분을 개선한 Go 확장**입니다. Go는 다른 origin/collection 경로로 이동하거나 서로 충돌하는 continuation·반복 marker를 오류로 처리합니다. 초기 query에 없던 임의 page/filter를 다음 링크가 추가하거나 기존 필터를 변경하는 것도 허용하지 않으며, marker와 초기 server limit의 허용된 변경만 적용합니다. Python의 임의 version-prefix 제거·cross-path continuation·잘못된 pagination shape까지 동일하게 처리한다고 보장하지 않습니다.

[계약 테스트](applicationcredentials/python_records_test.go)는 기존 Gophercloud public testhelper 기반 fixture를 재사용하여 8개 proxy의 route·body·default·raw view·local filter·pagination·전체 Find 검색·response/error 증거·옵션 소유권을 검증합니다. 실제 인증된 Keystone 배포 확인은 이 로컬 계약 테스트와 별개입니다.
