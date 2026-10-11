# Address Group 목록: Python과 Go

고정 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의
`conn.network.address_groups(**query)`를
`conn.NetworkV2(ctx).SecurityAddressGroups.ListRecords(ctx, options...)`로 사용합니다.
기존 Find/Get/Create/Update/Delete와 주소 추가·제거는 각각 이미 제공되는 API입니다.
이 목록 구현은 기존 Source 모델과 공유 `internal/neutronfind.Engine.List`를 재사용합니다.
사용자는 builder나 interface를 구현하지 않습니다.

| Python | Go |
|---|---|
| `conn.network.address_groups(project_id=..., addresses=[...])` | `ListRecords(ctx, WithRecordListQuery("project_id", ...), WithRecordListQuery("addresses", ...))` |
| `list(conn.network.address_groups(...))` | `AllRecords(ctx, options...)` |
| `limit=20, marker="ID"` | `WithRecordListLimit(20)`, `WithRecordListMarker("ID")` |
| `max_items=20, paginated=False` | `WithRecordListMaxItems(20)`, `WithRecordListPaginated(false)` |
| underlying list `headers` / `microversion` | `WithRecordListHeaders(...)`, `WithRecordListMicroversion(...)` |

```python
import openstack

conn = openstack.connect(cloud="dev")
rows = list(conn.network.address_groups(
    project_id="PROJECT_ID",
    addresses=["10.0.0.0/24"],
    max_items=20,
))
for row in rows:
    print(row.to_dict())
```

## 독립 Go main

[설치 안내](../../docs/install.md)에 따라 모듈을 준비하고 다음을 `main.go`로 저장합니다.
`go run . -cloud dev -project PROJECT_ID -max-items 20`은 목록만 읽습니다.
`-address`를 생략하면 로컬 주소 필터를 만들지 않습니다.

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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    project := flag.String("project", "", "optional server project filter")
    address := flag.String("address", "", "optional exact one-address list filter")
    cap := flag.Int("max-items", 20, "raw row cap; zero is unbounded")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    network, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    options := []addressgroups.RecordListOption{
        addressgroups.WithRecordListMaxItems(*cap),
    }
    if *project != "" {
        options = append(options, addressgroups.WithRecordListQuery("project_id", *project))
    }
    if *address != "" {
        options = append(options, addressgroups.WithRecordListQuery("addresses", []string{*address}))
    }
    for row, err := range network.SecurityAddressGroups.ListRecords(ctx, options...) {
        if err != nil { log.Fatal(err) }
        raw, err := json.MarshalIndent(row.Resource.Body, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Printf("HTTP %d\n%s\n", row.StatusCode, raw)
    }
}
```

## 기본값·필터와 응답 관측

`RecordListOpts`의 concrete 옵션은 공급한 query/Header/version만 snapshot합니다.
`WithRecordListOpts`는 앞서 적용한 옵션을 교체하고, 개별 옵션은 해당 키를 덮어씁니다.
목록 iterator를 실행할 때 source를 캡처하고 callback 입력·결과를 복사합니다.
같은 옵션은 병렬 호출에 재사용할 수 있습니다. unknown kwargs는 버리며 unknown query의
`MarshalJSON`을 실행하지 않습니다. raw bulk map의 unknown 값도 JSON 평가 전에 버립니다.

서버 query는 `limit`, `marker`, `fields`, `sort_key`, `sort_dir`, `name`, `description`,
`project_id`입니다. `fields` 배열의 이름을 바꾸지 않고 반복 query로 전송합니다.
null query는 생략하고 false/0/빈 문자열/배열은 Requests JSON-domain 표현을 사용합니다.
`id`, `tenant_id`, `addresses`는 선언된 Body의 **로컬 필터**입니다. addresses를 서버 query로
보내지 않습니다. `all_projects`나 shared/tags query는 이 클래스 mapping에 없고 권한을 늘리지
않습니다. 순서가 없는 bulk map은 정렬하고, 개별 query 옵션은 입력 순서를 보존합니다.

`AddressGroupListRecord`는 기존 `AddressGroupFindRecord`의 alias입니다. `Resource.Body`에는
Source의 id/name/description/project_id/tenant_id/addresses descriptor와 unknown 확장 속성을
보존합니다. 빠진 선언 값은 null이고, project_id가 빠졌을 때만 tenant_id로 보완합니다.
addresses의 non-list 값은 원소 하나의 목록으로 변환하고 명시적인 null은 그대로 둡니다.
`Wire.Body`는 받은 row 자체이며 `Envelope`는 그 row가 나온 **전체 페이지**입니다.
Header와 StatusCode도 실제 관측 값이며 채널끼리 독립된 사본입니다. 이 클래스에는 tag 기본값과
NetworkResource Header descriptor가 없으므로 tags·if_match를 합성하지 않습니다. 실제 If-Match
Header는 HTTP metadata에 남고 Body의 if_match는 일반 확장 속성입니다.

로컬 dict filter는 실제 값의 truthiness를 먼저 검사하고 recursive subset을 비교합니다.
비어 있지 않은 dict가 truthy scalar에서 mapping.get을 요구하면 전체 페이지 증거를 가진 오류입니다.
Source JSON float의 binary64 truthiness를 사용하므로 1e-400은 false, 1e400은 true입니다.
동등 비교는 공통 exact-decimal JSON 정책을 유지합니다. 예를 들어 integer9007199254740992와
filter9007199254740993.0은 Go에서 다르지만 Python JSON float decode 후에는 같을 수 있습니다.
이 숫자 동등 비교 차이를 전체 Python numeric parity로 집계하지 않습니다.

## 페이지·버전·오류

기본은 전체 페이지입니다. cap은 로컬 filter 전에 받은 raw row 수를 셉니다. 직접 limit이 없으면
max_items가 wire limit hint가 됩니다. explicit nonzero limit이면 짧은 페이지에도 retained wire ID
marker로 계속 읽습니다. 공개 Resource/Wire를 수정해도 다음 marker는 바뀌지 않습니다.
marker는 원본 JSON의 Requests query expansion에서 정확히 하나의 nonempty 값이어야 합니다.
cap 또는 `paginated=false`에 도달하거나 iterator를 중단하면 다음 페이지를 요청하지 않습니다.

plural/body links, top-level next, HTTP Link와 버전 prefix 경로를 지원합니다. `next:1e-400`은
binary64에서 0이므로 끝나며 truthy 숫자 next는 실제 전체 페이지를 가진 오류입니다.
순환·다른 origin/collection·초기 filter 변경은 거부합니다. 표준 dictionary `links:{next:URL}`은
고정 Source의 dict 변환 루프가 따라가지 못하는 형태이며 명시적인 Go 개선입니다.

클래스 `_max_microversion=None`이므로 기본으로 새 discovery를 요청하지 않습니다. 선택된
서비스 default를 유지하고 명시적 List Microversion은 모든 목록 페이지에 적용합니다.
명시적 빈 version은 이번 호출의 version Header를 생략하는 Go 옵션입니다. shared source/version
Header는 바꾸지 않으며 Source client의 Accept와 관계없이 목록 기본 Accept는 application/json입니다.
명시적 List Headers의 Accept만 이를 덮어씁니다.

Go engine은 최종 service representation의 HTTP200–399를 수용하고 그 밖의 실제 상태는 native
Gophercloud 오류로 유지합니다. Python의 임의 Adapter가 최종 1xx 응답까지 반환하는 모델을
재현하지 않습니다. 400–599·잘못된 객체/envelope/UTF-8·constructor cls(null/false 포함)/connection/
microversion/_synchronized 충돌과 후행 페이지 오류는 성공이나 empty 목록으로 숨기지 않습니다.
accepted Read/Close/decode 실패는 `resource.ResponseError`에 실제 body/Header/status를 남깁니다.
취소 원인과 source 변경이 Close에서 원복되는 경우도 오류로 보존하고 재전송하지 않습니다.

native retry의 version 변경/삭제/Omit, 없던 version·foreign version·auth/framing Header 주입,
bodyless GET에 JSON을 추가하는 변경은 첫 물리 오류와 callback 원인을 보존하고 다음 전송 전에
거부합니다. 일반 추가 retry Header와 live token 갱신·재인증은 native 정책을 유지합니다.
SDK constructor/session/route/cache/base_path/computed-location control은 query 확장으로 받지 않고
명시적인 Go 옵션 범위로 처리합니다. full Python mutable Resource/Connection/session/cache/dirty 상태,
location 계산과 deprecated jmespath/__conflicting_attrs overload는 별도 공통 과제로 남습니다.
기존 Source Find/native 모델·목록과 이 새 List profile을 혼합해 전체 SDK parity 완료로 세지 않습니다.

[Neutron 27.0.0 address-group 정책](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/address_group.py)의
GET/list는 admin 또는 해당 프로젝트 reader 또는 shared address group을 허용합니다.
service 역할만을 허용하는 별도 분기는 없습니다. 배포의 실제 권한·extension/backend acceptance는
서버가 판정하며 fixture 통과가 라이브 클라우드 검증을 뜻하지 않습니다.
