# 이미지 공유 멤버

`image.Service`의 멤버 workflow는 모든 호출에 부모 이미지 `resource.Ref`를 요구합니다. 이미지 ID는 직접 사용하고 명시적인 `resource.Name`만 기존 exact-name resolver로 찾습니다. 멤버는 프로젝트의 literal ID 문자열이며 Name 검색·프로젝트 조회·Member 객체에서 부모 복구를 하지 않습니다.

| 호출 | 고정 요청 | 정상 응답 |
| --- | --- | --- |
| `AddImageMember` | `POST images/{image_id}/members`, `{"member": memberID}` | 200 member object |
| `GetImageMember`·`FindImageMember` | `GET images/{image_id}/members/{member_id}` | 200 member object |
| `UpdateImageMember` | 같은 member endpoint의 PUT, `{"status": status}` | 200 member object |
| `ListImageMembers`·`AllImageMembers` | `GET images/{image_id}/members` | 200 `members` 배열 envelope |
| `RemoveImageMember` | 같은 member endpoint의 DELETE | 204 opaque acknowledgement |

```go
package example

import (
    "context"
    "fmt"

    "github.com/JSYoo5B/gophercloudsdk/image"
    "github.com/JSYoo5B/gophercloudsdk/resource"
)

func shareAndReadMembers(ctx context.Context, owner, member *image.Service, parent resource.Ref, memberID string) (*image.ImageMemberAcknowledgement, error) {
    common := []image.ImageMemberOption{
        image.WithImageMemberOpts(image.ImageMemberOpts{
            Headers: map[string]string{"X-Request-Source": "example"},
        }),
        image.WithImageMemberHeader("X-Member-Request", "single"),
        image.WithImageMemberHeaders(map[string]string{"X-Example": "members"}),
    }
    created, err := owner.AddImageMember(ctx, parent, memberID, common...)
    if err != nil {
        return nil, err
    }
    fmt.Println("created response", created.StatusCode)
    current, err := owner.GetImageMember(ctx, parent, memberID, common...)
    if err != nil {
        return nil, err
    }
    fmt.Println("current response", current.StatusCode)
    updated, err := member.UpdateImageMember(ctx, parent, memberID, "accepted", common...)
    if err != nil {
        return nil, err
    }
    fmt.Println("updated response", updated.StatusCode)

    found, err := owner.FindImageMember(ctx, parent, memberID,
        image.WithFindImageMemberOpts(image.FindImageMemberOpts{}),
        image.WithFindImageMemberHeader("X-Member-Request", "find"),
        image.WithFindImageMemberHeaders(map[string]string{"X-Example": "members"}),
        image.WithFindImageMemberIgnoreMissing(false))
    if err != nil {
        return nil, err
    }
    fmt.Println("found response", found.StatusCode)

    listOptions := []image.ListImageMembersOption{
        image.WithListImageMembersOpts(image.ListImageMembersOpts{}),
        image.WithListImageMembersHeader("X-Member-Request", "list"),
        image.WithListImageMembersHeaders(map[string]string{"X-Example": "members"}),
        image.WithListImageMembersMaxItems(20),
    }
    for row, err := range owner.ListImageMembers(ctx, parent, listOptions...) {
        if err != nil {
            return nil, err
        }
        fmt.Println("member row fields", len(row.Body))
    }
    all, err := owner.AllImageMembers(ctx, parent, listOptions...)
    if err != nil {
        return nil, err
    }
    fmt.Println("collected members", len(all))

    ack, err := owner.RemoveImageMember(ctx, parent, memberID,
        image.WithRemoveImageMemberOpts(image.RemoveImageMemberOpts{}),
        image.WithRemoveImageMemberHeader("X-Member-Request", "remove"),
        image.WithRemoveImageMemberHeaders(map[string]string{"X-Example": "members"}),
        image.WithRemoveImageMemberIgnoreMissing(true))
    return ack, err
}
```

`owner`와 `member`는 해당 작업 권한을 가진 별도 인증 context입니다. 실제 정책에 따라 한 context가 모든 작업을 허용받는다고 가정하지 않습니다. `parent`에는 `resource.ID("image-id")` 또는 명시적인 `resource.Name("image-name")`을 전달합니다. 이 예제의 List와 All은 독립 GET이며 둘 다 최대 20개 row를 로컬에서 소비합니다. 마지막 반환의 acknowledgement와 error를 함께 확인합니다. 실제 204를 받은 뒤 body 처리만 실패했다면 `ack != nil`과 `err != nil`이 함께 가능합니다.

## Python proxy와 비교

```python
def share_and_read_members(owner_conn, member_conn, image_id, member_id):
    created = owner_conn.image.add_member(image_id, member=member_id)
    current = owner_conn.image.get_member(member_id, image_id)
    updated = member_conn.image.update_member(
        member_id, image_id, status="accepted"
    )
    rows = list(owner_conn.image.members(image_id))
    found = owner_conn.image.find_member(
        member_id, image_id, ignore_missing=False
    )
    removed = owner_conn.image.remove_member(
        member_id, image=image_id, ignore_missing=True
    )
    return created, current, updated, rows, found, removed
```

고정된 openstacksdk commit `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [여섯 proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1222-L1386)는 이미지·멤버 Resource 또는 문자열에서 ID를 얻습니다. Add의 속성은 `member=`이며 wire body의 `member_id`가 아닙니다. 문자열을 자동 Name으로 찾지 않습니다.

Python `remove_member`의 `image=None` signature와 설명은 optional parent를 허용하지만, 실제 구현은 `image`에서 ID를 직접 얻고 supplied Member를 버립니다. generic `_get_uri_attribute`로 Member의 cached parent를 회수하지 않으므로 예제도 부모를 명시합니다. `get_member`·`update_member`도 supplied Member를 그대로 fetch/commit하지 않고 ID로 새 Resource를 준비합니다. Go는 항상 명시적인 부모와 literal member ID를 요구합니다.

Python `members(image, **query)`는 이 pin에서 `query`를 `_list`에 전달하지 않습니다. `find_member`는 inherited [Resource.find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2489-L2582)의 직접 GET이 400·403·404이면 LIST로 재시도하고 ID 또는 inherited name으로 비교해 중복을 검사합니다. fallback row를 다시 GET하지 않습니다. Go `FindImageMember`는 literal ID의 GET만 수행하며 400·403을 무시하거나 목록·Name fallback을 추가하지 않습니다. Member의 canonical 모델과 서버에는 name이 없습니다.

Python add/get/update/find는 mutable Member Resource를, members는 Resource generator를 반환합니다. remove proxy는 성공과 ignored NotFound에서 모두 None입니다. Python generic Resource의 descriptor·dirty fields·cache·session 동작과 넓은 성공 status/body 처리까지 복제하지 않습니다. 이 pin의 find 기본 `ignore_missing=True`는 향후 SDK 6.0의 기본 변경 warning을 발생시키므로 예제는 false를 명시했습니다.

## 기본값·입력·옵션

Add/Get/Update의 `ImageMemberOpts`에는 일반 `Headers`만 있습니다. Remove/Find의 `IgnoreMissing *bool`은 nil이면 true이고 명시적인 false는 missing 오류를 유지합니다. `ListImageMembersOpts.MaxItems`는 0이면 무제한, 양수이면 로컬 소비 cap이며 음수는 잘못된 옵션입니다. 전체 zero opts도 이 기본값을 유지합니다. query·limit·marker·pagination header·wait·status filter를 보내지 않습니다.

member ID는 nonempty valid UTF-8의 안전한 단일 unescaped path segment여야 하며 parent lookup·callback·HTTP 전에 검사합니다. UUID whitelist나 trim·case normalization·프로젝트 Name 조회를 하지 않습니다. slash·percent escape·query/fragment처럼 route를 바꿀 수 있는 값은 거부합니다. Add도 같은 bounded ID 규칙을 적용하므로 Python의 arbitrary body attrs보다 제한적입니다.

Update의 입력 status는 정확히 `pending`·`accepted`·`rejected`만 허용합니다. empty·대문자·공백·unknown 값은 callback·Name 조회·HTTP 전에 거부합니다. 응답 status는 별도의 passive nullable 문자열이며 입력 enum을 응답에 강제하지 않습니다.

각 `With…Opts`는 concrete 설정 전체를 교체합니다. `Header`·`Headers`는 일반 header를 합치며 IgnoreMissing·MaxItems helper는 해당 값을 선택합니다. 입력 map·pointer·option slice와 callback 전후 config를 복사하고 각 요청 또는 iterator 순회에서 callback을 한 번 적용합니다. 재사용한 iterator는 순회마다 독립 설정과 요청을 준비합니다. nil/error callback, 잘못된 일반 header, auth·framing·version·representation 보호 header와 충돌 alias는 요청 전에 거부합니다.

## 유한 목록과 passive 응답

고정된 Glance commit `57f7dd9e76ef24e1e9013eceaa703bd442469a24`의 [목록 controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/image_members.py#L234-L280)는 query·limit·marker를 읽지 않고 이미지의 유한한 멤버 목록을 반환합니다. [serializer](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/image_members.py#L396-L435)는 `members`·`schema` envelope를 만들며 다음 페이지를 제공하지 않습니다. [공식 목록 문서](https://docs.openstack.org/api-ref/image/v2/index.html#list-image-members)의 권한별 노출도 서버 소유 정책입니다.

`ListImageMembers`는 lazy합니다. 호출 시 option slice를 복사하고 source 검증·capture·callback·필요한 부모 Name 조회는 순회할 때 시작합니다. 각 순회는 필요한 부모 조회를 제외하고 한 번의 body/query 없는 collection GET을 수행합니다. `members`는 required nonnull array여야 하고 전체 응답은 valid UTF-8 nonnull object여야 합니다. row는 소비할 때 canonical object로 검사하므로 cap·early break 이후의 unused row 오류를 검사하지 않고 추가 HTTP도 수행하지 않습니다. envelope의 schema·next·links·unknown 값은 continuation이나 자동 검증을 만들지 않습니다.

`AllImageMembers`는 목록을 모으며 성공한 empty 응답에는 nonnil empty slice를 반환합니다. 중간 오류가 발생하면 partial rows를 반환하지 않습니다. 각 row는 독립 `Body map[string]json.RawMessage`·Header·실제 StatusCode를 소유하며 소모 전 context·source/provider 변경을 확인합니다.

`ImageMember`의 canonical `ImageID`·`MemberID`·`Status`·`Schema`는 `*string`입니다. missing/null은 nil, empty string은 nonnil이며 nonstring canonical 값은 오류입니다. `Metadata.CreatedAt`·`UpdatedAt`도 literal optional 문자열로 날짜를 parse하지 않습니다. inherited `Metadata.Links`는 nil이고 links의 실제 JSON은 raw Body에 남습니다. unknown·대소문자 decoy·큰 숫자·nested 값을 float64로 바꾸지 않으며 typed 값·raw fields·header를 서로 독립 복사합니다.

응답의 image/member ID가 요청과 다르거나 empty여도 passive 응답 데이터로 보존합니다. 요청 ID를 응답에 합성하거나 일치 검사를 요구하지 않고 응답·schema·links·date가 다음 route를 만들지 않습니다. 기존 native `image/v2/members.Member`의 `time.Time` 날짜·plain string 모델과 SinglePageBase ABI는 유지합니다.

## missing·실패·공유 정책

Remove와 Find 기본 true는 해당 고정 member 요청의 실제 404만 privately 처리합니다. body 전체 read·Close·context 처리가 모두 성공해야 `nil, nil`이며 Read/Close·`ctx.Err()`·custom cause 실패는 실제 404 증거를 가진 `resource.ResponseError`로 남습니다. 이 private 404에만 native `RetryFunc`를 호출하지 않습니다. false는 native 404 오류·retry 정책을 유지합니다. 부모 Name not-found·ambiguous·surfaced native body/decode/HTTP 오류, transport·hook·nested 404는 missing으로 삼키지 않습니다.

404가 멤버 부재를 증명하지는 않습니다. 서버 [show](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/image_members.py#L282-L312)는 정보 유출을 막으려고 Forbidden을 404로 바꾸며 DB 계층도 일부 Forbidden을 NotFound로 변환합니다. Get/Add/Update/List에는 ignore missing 정책을 추가하지 않습니다.

Add/Get/Update/Find actual 200의 read·Close·context·UTF-8·JSON·schema 오류에서는 nil typed result와 전체 실제 bytes/header/status를 가진 ResponseError를 반환합니다. Remove actual 204는 non-JSON·invalid UTF-8도 opaque acknowledgement의 `Body []byte`로 유지하고 accepted 처리 실패에는 nonnil ack와 error를 함께 반환합니다. ack는 선택한 request ImageID·MemberID와 실제 Header·StatusCode를 가지며 ignored 404는 ack를 만들지 않습니다. accepted body는 한 번 닫고 실패 뒤 replay하지 않습니다.

기존 mutation source 준비를 재사용합니다. callback 전에 source/client/provider·prefix·일반 headers를 capture하고 callback 및 부모 Name 성공·실패 뒤 변경을 재검사합니다. 원래 provider의 live auth를 사용하며 기존 부모 Name native pager의 body ownership·continuation 한계는 유지합니다. 공통 REST/fixedrequest는 configured native pre-body retry·reauth·backoff와 동일 target redirect callback을 유지하고 method·origin·path·query·owned body/response 변경을 다음 transport 전에 차단합니다. 동일 serialized JSON 교체만 허용하며 nil body와 JSON null은 다릅니다. native reauth의 `ErrOriginal`·`ErrReauth`는 해당 wrapper 필드에서 확인합니다.

생성은 pending이며 accepted/rejected의 이미지 목록 노출 효과, shared visibility·소유권·RBAC·quota는 서버가 판단합니다. [공식 생성 문서](https://docs.openstack.org/api-ref/image/v2/index.html#create-image-member)와 실제 controller는 shared 이미지를 요구합니다. SDK는 상태 prefetch·권한 발견·자동 수락·cache 갱신·wait·cleanup을 수행하지 않습니다. local 계약 검증은 실제 cloud 권한이나 full Python Resource/cache/session parity를 증명하지 않습니다.

실제 local 계약은 [외부 HTTP tests](members_contracts_test.go), [core tests](members_core_test.go), [option tests](members_options_test.go)에서 검증합니다. [Connection test](../connection_image_members_test.go)는 공유 client와 concrete 기본값을, [generator test](../internal/cmd/sdkgen/glance_members_test.go)는 기존 native member API·scope 보존과 전용 workflow 문서를 확인합니다.

Python `get_member`는 image/member ID로 새 Resource를 만들어 응답에 빠진 identity를 seed에서 유지할 수 있습니다. Go `GetImageMember`의 nullable `ImageID`·`MemberID`는 실제 canonical 응답만 나타냅니다. Python의 `member` descriptor alias가 응답에 있으면 Go에서는 `Metadata.Body["member"]`로 확인하며, typed `MemberID`는 `member_id`를 읽습니다.
