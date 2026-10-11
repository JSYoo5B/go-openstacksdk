# Keystone v3 Credential Python proxy

`conn.IdentityV3(ctx).Credentials`의 `Record` 메서드는 pinned openstacksdk의 일반 Credential `/credentials` proxy를 제공합니다. `EC2Credentials`의 `/users/{user_id}/credentials/OS-EC2`는 별도의 Gophercloud API입니다. 일반 Credential의 `type="ec2"`는 Credential blob의 형식이며, 두 경로를 같은 API로 취급하지 않습니다.

| openstacksdk | Go | 기본 동작 |
|---|---|---|
| `create_credential(**attrs)` | `Credentials.CreateRecord(ctx, options...)` | 명시한 Body attrs만 POST |
| `get_credential(credential)` | `Credentials.GetRecord(ctx, credentialID, options...)` | literal ID로 member GET |
| `credentials(**query)` | `Credentials.ListRecords(ctx, options...)` | type/user_id는 서버 query, 나머지 Body 필드는 local filter |
| `find_credential(name_or_id, ignore_missing=True)` | `Credentials.FindRecord(ctx, nameOrID, options...)` | GET 후 clean 400/403/404만 전체 목록에서 exact ID/name 검색 |
| `delete_credential(credential, ignore_missing=True)` | `Credentials.DeleteRecord(ctx, credentialID, options...)` | clean 404를 기본으로 억제 |

이 5개는 기존 generated `Create/Get/List/Delete/Update`와 native `Resources`의 타입·status·lookup 계약을 유지하면서 추가한 API입니다. Source의 `update_credential` 및 mutable Resource/session surface는 별도 구현 단계입니다.

```python
created = conn.identity.create_credential(
    user_id=user_id, type="ec2", project_id=project_id,
    blob='{"access":"ACCESS","secret":"SECRET"}')
credentials = conn.identity.credentials(user_id=user_id, type="ec2")
found = conn.identity.find_credential(credential_id)
# 필요한 경우: conn.identity.delete_credential(credential_id, ignore_missing=False)
```

## 독립 Go main

[설치 안내](../../docs/install.md)대로 모듈을 준비하고 아래를 `main.go`로 저장합니다. `go run . -cloud dev -user USER_ID -id CREDENTIAL_ID`는 일반 Credential 목록·member GET·Find를 실행합니다. `-create`와 `-delete`를 명시한 경우에만 각각 생성·삭제를 수행합니다.

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
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials"
)

func printRecord(v *credentials.CredentialRecord) {
    if v == nil { fmt.Println("null"); return }
    data, err := json.MarshalIndent(v.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    user := flag.String("user", "", "credential owner user ID")
    project := flag.String("project", "", "credential project ID")
    kind := flag.String("type", "ec2", "credential blob type")
    id := flag.String("id", "", "credential ID to GET and find")
    blob := flag.String("blob", "{}", "credential blob string")
    create := flag.Bool("create", false, "create a generic Credential")
    remove := flag.String("delete", "", "credential ID to delete")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    api := service.Credentials
    for v, err := range api.ListRecords(ctx,
        credentials.WithRecordListType(*kind),
        credentials.WithRecordListUserID(*user),
        credentials.WithRecordListMaxItems(100)) {
        if err != nil { log.Fatal(err) }
        printRecord(v)
    }
    if *id != "" {
        v, err := api.GetRecord(ctx, *id)
        if err != nil { log.Fatal(err) }
        printRecord(v)
        found, err := api.FindRecord(ctx, *id)
        if err != nil { log.Fatal(err) }
        printRecord(found)
    }
    if *create {
        v, err := api.CreateRecord(ctx,
            credentials.WithRecordCreateField("user_id", *user),
            credentials.WithRecordCreateField("project_id", *project),
            credentials.WithRecordCreateField("type", *kind),
            credentials.WithRecordCreateField("blob", *blob))
        if err != nil { log.Fatal(err) }
        printRecord(v)
    }
    if *remove != "" {
        if err := api.DeleteRecord(ctx, *remove,
            credentials.WithRecordDeleteIgnoreMissing(false)); err != nil { log.Fatal(err) }
    }
}
```

## 옵션·응답·오류 계약

Create는 `WithRecordCreateField` 또는 `WithRecordCreateOpts`의 알려진 `id/name/blob/project_id/type/user_id` 필드만 `credential` envelope 안에 보냅니다. 명시한 null·false·빈 문자열·빈 array를 보존하고, 생략한 속성은 추가하지 않습니다. Python proxy처럼 native builder의 required-field 검사나 string 강제 변환을 추가하지 않습니다. 미등록 attrs는 버립니다. `blob`이 string인 실제 Keystone API의 사용 예는 위 main처럼 string으로 넘기며, SDK raw 계약 테스트는 다른 JSON 타입도 보존하는지 검증합니다.

List의 `WithRecordListType`·`WithRecordListUserID`는 보통 쓰는 서버 query 옵션입니다. `WithRecordListQuery(key, value)`는 Source의 kwargs를 분류합니다. `type`·`user_id`는 공통 Requests encoder를 사용해 wire query로 변환하고, 다른 선언된 Body 필드는 local JSON filter로 적용합니다. null인 wire 값은 생략하며 array·object의 query expansion과 Python bool spelling을 재사용합니다. 미등록 query는 버립니다. `WithRecordListLimit`·`WithRecordListMarker`는 페이지 query이고, `WithRecordListMaxItems`는 **local filter 전에 관찰한 raw row 수**를 제한합니다. 양수 max_items는 생략했거나 0인 limit의 초기 힌트가 됩니다. `WithRecordListPaginated(false)`는 첫 페이지만 읽습니다. List iterator는 lazy이고 재사용할 때마다 새 source snapshot으로 요청합니다.

Find의 옵션은 Source와 같이 missing 정책뿐입니다. `WithRecordFindIgnoreMissing(false)`는 전체 검색에서 찾지 못한 결과를 `resource.NotFoundError`로 바꿉니다. 기본값은 nil,nil이며, direct GET이 성공하면 list를 호출하지 않습니다. clean 400·403·404만 GET-first fallback을 허용하고, fallback은 정확한 ID 또는 name으로 모든 페이지를 관찰합니다. 중복은 `resource.AmbiguousError`입니다. 같은 이름을 찾은 뒤의 후속 페이지 오류도 숨기지 않습니다. name은 서버 query mapping에 없으므로 자동 wire name hint를 만들지 않습니다.

Get/Create는 `credential` singular key가 있으면 unpack하고 없으면 root 객체를 사용합니다. 응답에서 생략한 id는 GET의 literal id, 생략한 Create 속성은 생성 attrs에서 유지합니다. 빈 본문·잘못된 JSON syntax는 Source의 ValueError 처리처럼 seed를 유지한 view로 허용하지만, 유효한 nonobject JSON은 실제 accepted 응답을 가진 오류입니다. List는 plural `credentials` JSON 표현이 필요하며 singleton 객체도 받습니다. Get/Create/List/Delete는 실제 200..399를 허용하며, Delete의 opaque body는 JSON으로 decode하지 않습니다. Delete는 기본으로 clean 404만 억제하고, strict 옵션은 원래 HTTP error를 유지합니다. read·close·transport·retry callback·source 변경·취소가 섞인 404는 억제하지 않습니다.

`CredentialRecord.Resource`는 declared view이며 `id/name/blob/project_id/type/user_id/location` 7개 값이 있습니다. 누락된 Body descriptor와 location은 null입니다. `Wire`는 unknown attrs와 정확한 큰 정수를 포함한 원래 응답 객체, `Envelope`는 전체 HTTP 본문, `Header`·`StatusCode`는 실제 응답입니다. 각 채널의 raw fields와 header는 독립 복사합니다. `blob`과 날짜 형태의 unknown field를 native 모델·`time.Time`에 맞추지 않습니다. accepted 처리 오류의 실제 status/body/header는 `resource.ResponseError`, 연산과 cause는 `resource.OperationError`에 남습니다.

## Go profile과 검증 경계

Library-owned concrete option이 map·raw JSON·pointer를 복사합니다. bulk 옵션은 전체 정책을 교체하고, 적용이 끝난 callback pointer를 보관해 변경해도 다음 phase의 정책이 바뀌지 않습니다. callback 오류와 동시에 발생한 취소/cause 또는 source 변경도 보존합니다. 기존 source capture·고정 REST route/header guard·FindIdentity·JSON filter·Requests query encoder를 재사용하며 공통 엔진에 새 구현을 추가하지 않습니다.

Python Resource/Munch 객체 인자·같은 객체 재사용·mutable Resource commit/update·cache/session/subclass hooks·Connection 기반 location·임의 adapter/microversion·모든 동적 kwargs는 별도 범위입니다. Go member는 유효한 UTF-8 문자열을 URL segment로 한 번 escape합니다. JSON filter는 bool과 number를 구분하고 nonobject nested 비교를 false로 처리합니다. typed limit/cap은 음수를 허용하지 않으며 Source의 임의 pagination/JMESPath 제어 kwargs까지 같은 동작을 주장하지 않습니다.

Pagination은 Source의 array next link·top-level next·HTTP Link·양수 initial limit의 short-page marker fallback과 빈 page 종료를 지원합니다. `links:{next:"URL"}`는 pinned Python paginator가 인식하지 못하는 형태를 지원하는 Go 호환 확장입니다. 다른 origin/path, 초기 query에 없던 임의 page/filter 추가, 필터 변경, conflicting next link, marker 반복은 고정 REST 정책에서 오류입니다. 소비자가 반환 view의 id와 Wire를 변경해도 다음 marker는 보관한 원문 page에서 가져옵니다.

[계약 테스트](credentials/python_records_test.go)는 기존 Gophercloud public testhelper fixture를 재사용해 일반 `/credentials` 5개 route와 EC2 전용 경로의 구분, raw body/view, wire/local query 분리, pagination, 전체 Find 검색, missing 정책, 소유권과 응답/cause를 검증합니다. native 계약은 [기존 테스트](credentials/native_contracts_test.go)로 함께 확인합니다. 실제 인증된 Keystone 배포 확인은 로컬 계약 테스트와 별개입니다.


## 거부 응답의 읽기·Close 오류와 find fallback

Member GET의 400/403/404와 DeleteRecord의 404는 공통 `DoJSONGuardedRejections`로 관찰합니다. 정상 404만 기본 삭제에서 무시하며, Read·Close·취소 cause·Source 변경·retry callback 오류가 섞인 거부 응답은 native 상태/본문/헤더와 원인을 함께 반환하고 find의 목록 fallback을 차단합니다. Read에서 변한 Source가 Close에서 원복되어도 이미 관측한 실패는 유지됩니다. native retry hook이 같은 원래 HTTP 거부 오류를 그대로 반환한 경우에는 정상 거부 상태를 유지해 기본 ignore_missing과 find fallback이 계속 동작합니다. accepted 응답 증거와 native ABI는 유지합니다.

회귀는 `TestPythonCredentialRejectedResponseFailuresStopDeleteAndFind`와 `TestPythonCredentialCleanRejectionsRemainEligibleAfterNativeRetryHook`이며 기존 package fault body/transport 및 testcloud를 재사용합니다. 집중 race 실행을 통과했으며 전체 검사·판정 근거는 [지원 판정대장](../../docs/sdk-support-ledger.md#핵심-credentialneutronswiftnova-34개-마감)에 기록합니다.
