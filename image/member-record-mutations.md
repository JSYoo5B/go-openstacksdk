# Glance 공유 멤버 레코드 조회·추가·수정·삭제: Python과 Go

`Connection`과 `image.Service`의 `GetImageMemberRecord`, `AddImageMemberRecord`, `UpdateImageMemberRecord`, `RemoveImageMemberRecord`는 고정 openstacksdk의 공개 Member proxy가 만드는 요청과 declared Resource 반환을 SDK가 처리합니다. 부모 이미지를 명시하고 라이브러리가 제공하는 concrete `With…` 옵션을 사용합니다. 조회·추가·수정은 `ImageMemberRecord`, 삭제는 실제 응답을 담은 `ImageMemberAcknowledgement`를 반환합니다.

| Python `conn.image` | Go Connection / Service | 요청과 기본 동작 |
|---|---|---|
| `get_member(member, image)` | `GetImageMemberRecord(ctx, parent, input, options...)` | 선택한 ID에 GET; missing은 오류 |
| `add_member(image, **attrs)` | `AddImageMemberRecord(ctx, parent, options...)` | supplied declared 속성의 flat POST; 무옵션 Body는 `{}` |
| `update_member(member, image, **attrs)` | `UpdateImageMemberRecord(ctx, parent, input, options...)` | 새 Member를 만들어 PUT; 이전 레코드 Body를 재전송하지 않음 |
| `remove_member(member, image, ignore_missing=True)` | `RemoveImageMemberRecord(ctx, parent, input, options...)` | DELETE; 기본값은 깨끗한 최종 native404를 무시 |

기존 typed `GetImageMember/AddImageMember/UpdateImageMember/RemoveImageMember`와 `API.Members`는 유지합니다. typed 조회·추가·수정은 strict200와 타입 변환을 제공하고, typed Add는 문자열 member ID, typed Update는 `pending/accepted/rejected` status와 status만 있는 Body를 요구합니다. typed Remove의 strict204 계약도 유지합니다. 새 레코드 API는 Source의 fresh Member 생성, raw 속성, 실제 200..399 응답 정책을 선택하는 별도 API입니다. 목록·검색은 [멤버 레코드 목록·검색](member-records.md), 기존 API는 [멤버 사용법](members.md)을 참고하세요.

## Python 사용과 프로젝트 역할

```python
import openstack

owner = openstack.connect(cloud="owner")
recipient = openstack.connect(cloud="recipient")
image_id = "image-id"
member_id = "recipient-project-id"

created = owner.image.add_member(image_id, member_id=member_id)
fetched = owner.image.get_member(member_id, image_id)
updated = recipient.image.update_member(fetched, image_id, status="accepted")
owner.image.remove_member(member_id, image_id, ignore_missing=False)
```

고정 [Glance 정책](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L223-L292)의 Add/Delete는 `ADMIN_OR_PROJECT_MEMBER`, Get은 `ADMIN_OR_PROJECT_READER_OR_SHARED_MEMBER`, Update는 `ADMIN_OR_SHARED_MEMBER`이며 모두 project scope입니다. 따라서 예제는 이미지 소유자 cloud에서 추가·조회·삭제하고, 공유받는 프로젝트 cloud에서 status를 변경합니다. 실제 소유권, 공유 상태, visibility, 요청 schema, 배포별 policy override의 허용 여부는 서버가 판단합니다.

## 독립 Go main

[외부 모듈 설치 안내](../docs/install.md)를 따라 아래 코드를 `main.go`로 저장합니다. 예를 들어 `OPENSTACKSDK_OWNER_CLOUD=owner OPENSTACKSDK_MEMBER_CLOUD=recipient OPENSTACKSDK_IMAGE_ID=image-id OPENSTACKSDK_MEMBER_ID=recipient-project-id go run .`으로 실행합니다. 같은 값을 flags로 지정할 수도 있습니다. 두 cloud는 `clouds.yaml`에 있어야 합니다.

이 main은 실제 공유 멤버를 추가하고 status를 변경한 뒤 삭제합니다. 문서의 검증은 빌드만 수행하며 실제 cloud 호출은 실행하지 않습니다. 각 오류에서 멈추므로 이미 성공한 변경을 자동으로 되돌리지 않습니다.

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    ownerCloud := flag.String("owner-cloud", os.Getenv("OPENSTACKSDK_OWNER_CLOUD"), "owner clouds.yaml cloud")
    memberCloud := flag.String("member-cloud", os.Getenv("OPENSTACKSDK_MEMBER_CLOUD"), "shared recipient clouds.yaml cloud")
    imageID := flag.String("image-id", os.Getenv("OPENSTACKSDK_IMAGE_ID"), "literal parent image ID")
    memberID := flag.String("member-id", os.Getenv("OPENSTACKSDK_MEMBER_ID"), "literal recipient project ID")
    status := flag.String("status", "accepted", "requested member status")
    flag.Parse()
    if *ownerCloud == "" || *memberCloud == "" || *imageID == "" || *memberID == "" {
        log.Fatal("owner-cloud, member-cloud, image-id and member-id are required")
    }
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    if err := run(ctx, *ownerCloud, *memberCloud, *imageID, *memberID, *status); err != nil {
        var response *resource.ResponseError
        if errors.As(err, &response) {
            fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
        }
        log.Fatal(err)
    }
}

func showRecord(operation string, record *image.ImageMemberRecord, callErr error) error {
    value := map[string]any{"operation": operation}
    if record != nil {
        value["resource"] = record.Resource
        value["wire"] = record.Wire
        value["image_id"] = record.ImageID
        value["status_code"] = record.StatusCode
        value["envelope_bytes"] = len(record.Envelope)
    }
    if callErr != nil { value["error"] = callErr.Error() }
    body, err := json.MarshalIndent(value, "", "  ")
    if err != nil { return err }
    fmt.Println(string(body))
    return callErr
}

func showAcknowledgement(ack *image.ImageMemberAcknowledgement, callErr error) error {
    value := map[string]any{"operation": "remove"}
    if ack != nil {
        value["image_id"] = ack.ImageID
        value["member_id"] = ack.MemberID
        value["status_code"] = ack.StatusCode
        value["response_bytes"] = len(ack.Body)
    }
    if callErr != nil { value["error"] = callErr.Error() }
    body, err := json.MarshalIndent(value, "", "  ")
    if err != nil { return err }
    fmt.Println(string(body))
    return callErr
}

func run(ctx context.Context, ownerCloud, memberCloud, imageID, memberID, status string) error {
    owner, err := openstack.Connect(ctx, openstack.WithCloud(ownerCloud))
    if err != nil { return err }
    recipient, err := openstack.Connect(ctx, openstack.WithCloud(memberCloud))
    if err != nil { return err }
    parent := resource.ID(imageID)
    input := image.ImageMemberRecordRequest{ID: memberID}
    created, err := owner.AddImageMemberRecord(ctx, parent,
        image.WithImageMemberRecordMemberID(memberID),
        image.WithImageMemberRecordWriteHeader("X-Request-Source", "member-record-example"))
    if err := showRecord("add", created, err); err != nil { return err }
    fetched, err := owner.GetImageMemberRecord(ctx, parent, input,
        image.WithImageMemberHeader("X-Request-Source", "member-record-example"))
    if err := showRecord("get", fetched, err); err != nil { return err }
    updated, err := recipient.UpdateImageMemberRecord(ctx, parent,
        image.ImageMemberRecordRequest{Record: fetched},
        image.WithImageMemberRecordStatus(status))
    if err := showRecord("update", updated, err); err != nil { return err }
    service, err := recipient.Image(ctx)
    if err != nil { return err }
    repeated, err := service.UpdateImageMemberRecord(ctx, parent,
        image.ImageMemberRecordRequest{Record: updated},
        image.WithImageMemberRecordStatus(status))
    if err := showRecord("update-again", repeated, err); err != nil { return err }
    ack, err := owner.RemoveImageMemberRecord(ctx, parent, input,
        image.WithRemoveImageMemberIgnoreMissing(false),
        image.WithRemoveImageMemberHeader("X-Request-Source", "member-record-example"))
    return showAcknowledgement(ack, err)
}
```

Connection 메서드는 공유 Image Service로 위임합니다. `conn.Image(ctx)`에서 얻은 Service에도 같은 메서드·입력·옵션이 있으며, 예제의 `update-again`은 같은 status를 명시해도 fresh Member의 PUT을 다시 수행합니다. 예제의 2분 제한은 caller가 선택한 context입니다.

## 입력과 concrete 옵션

`ImageMemberRecordRequest`는 `ID string`과 `Record *ImageMemberRecord` 중 하나로 멤버를 선택합니다. Record를 주면 canonical identity를 읽습니다. `Record.Resource.Body["id"]`가 있으면 null·invalid·빈 값까지 그 값이 우선하며 잘못된 id에 member_id로 fallback하지 않습니다. **id 키 자체가 없을 때만** `member_id`를 alternate ID로 사용하므로, 직접 만든 Resource도 이 형태로 멤버를 선택할 수 있습니다. 입력 Record의 status 등 다른 Body 속성, ImageID, Wire, Envelope, Header, 이전 응답은 새 요청에 섞지 않으며 입력 Record 자체를 수정하지 않습니다. Wire와 ImageID는 identity나 부모의 입력 소스가 아닙니다.

멤버 ID는 공백만 있는 값·비 UTF-8·제어 문자·정확한 `.`/`..`를 거부합니다. slash·percent·Unicode·query·backslash는 literal 값으로 받아 멤버 경로의 한 segment로 한 번 이스케이프합니다. 이는 기존 typed API의 safe segment 검증과 구분됩니다.

요청 identity로 읽는 raw JSON 문자열은 unpaired UTF-16 surrogate escape(예: `"\uD800"`)를 거부합니다. valid pair `"\uD83D\uDE00"`, 실제 U+FFFD와 `"\uFFFD"`, literal backslash-u 문자열 `"\\ud800"`는 각각의 정상 값을 보존합니다. `encoding/json`의 replacement character 치환으로 다른 ID를 선택하지 않으며, 이 규칙은 일반 Body descriptor 변환과 구분됩니다.

부모 `resource.Ref`는 매번 명시합니다. `resource.ID(imageID)`는 부모 lookup을 하지 않으며, `resource.Name(imageName)`은 기존 image 이름 resolver를 사용하는 Go 확장입니다. 멤버 ID를 프로젝트 이름으로 조회하거나 입력 Record에서 부모를 복구하지 않습니다. 고정 Python `remove_member(member, image=None)`도 `_get_id(image)`를 직접 전달하며 멤버의 부모를 복원하지 않으므로, Go는 부모 없는 입력을 요청 전에 거부합니다. 명시적인 부모 Name lookup을 제외하면 이 네 메서드에 추가 GET·목록 검색이 없습니다.

Get은 기존 `ImageMemberOption`의 `WithImageMemberOpts/Header/Headers`를 사용합니다. Add/Update는 `ImageMemberRecordWriteOpts{Headers map[string]string; Attributes []resource.ListOption}`와 다음 helper를 사용합니다.

- `WithImageMemberRecordWriteOpts`: 전체 설정을 snapshot하고 교체합니다.
- `WithImageMemberRecordWriteHeader/Headers`: 일반 HTTP 헤더를 추가·교체합니다.
- `WithImageMemberRecordWriteAttribute(field, value)`: 하나의 declared 속성을 추가·교체합니다. `WithImageMemberRecordWriteAttributes(map[string]any)`는 이전 attribute set 전체를 교체하고 Headers는 유지합니다. nil·빈 map은 이전 속성을 비웁니다.
- `WithImageMemberRecordMemberID(value any)` / `WithImageMemberRecordStatus(value any)`: 해당 속성의 편의 helper입니다.

`Attributes`는 SDK의 semantic `resource.WithFilter/WithFilters` concrete option을 담습니다. 목록의 limit/query/pagination 제어는 write Attributes에 사용할 수 없습니다. 새 builder interface를 구현할 필요가 없습니다. 순서대로 helper를 적용하며 singular helper로 같은 spelling을 다시 지정하면 마지막 값이 우선합니다. plural Attributes helper 이후에는 그 map에 없는 이전 속성을 보내지 않습니다. canonical `member_id`와 wire `member`가 함께 있으면 canonical spelling을 우선하는 Go 선택 규칙을 사용합니다. Python kwargs·Resource constructor의 insertion 순서와 모든 alias 충돌을 동일하게 재현한다는 뜻은 아닙니다.

Body의 declared 속성은 `id`, `name`, `member_id`(wire `member`), `created_at`, `status`, `schema`, `updated_at`입니다. 알려진 값은 `any`의 JSON 표현과 explicit null을 보존하고 status enum·날짜·schema를 client에서 검증하지 않습니다. unknown 속성은 Body에 넣지 않습니다. helper가 먼저 capture한 unknown 값의 marshal 오류도 declared 선택 단계에서 버리지만, custom marshaler 자체가 실행되지 않는다고 보장하지는 않습니다. 선택한 known 속성의 marshal 오류는 HTTP 전에 반환합니다.

Add는 선택한 declared 속성만 flat POST로 보내므로 무옵션 `{}`와 explicit null도 허용합니다. `member_id`와 `member`를 사용할 수 있고 `id`도 POST Body에 포함할 수 있습니다. 서버가 필수 member·허용 schema를 검사합니다.

Update는 선택한 멤버 ID를 wire `member`에 bind한 **새 Member**를 만듭니다. 따라서 attrs가 없거나 같은 status·unknown 속성만 제공해도 `{"member":"selected-id"}`를 포함한 PUT을 수행합니다. supplied Record의 기존 속성을 diff하거나 JSON Patch를 만들지 않습니다. `member`·`member_id`·`image_id` 직접 지정은 Source binding과의 충돌이므로 거부합니다. `id` 속성은 현재 요청 URI의 멤버 ID를 바꿀 수 있지만 PUT Body에서는 제외합니다.

Add의 `image_id`와 두 write 메서드의 `__conflicting_attrs`, `base_path`, `connection`, `_synchronized`, `microversion` 등 Resource 제어 인자는 고정 Go profile에서 명시적으로 거부합니다. Python `_create`의 conflict injection·동적 Resource 인자를 모두 노출하지 않습니다. 특히 Source의 instance `microversion` attrs는 classmethod `_get_microversion`의 request negotiation을 직접 덮어쓰지 않습니다. 일반 헤더 옵션과 Source session의 microversion 설정은 이 attrs와 구분합니다.

## declared Record, 실제 응답과 오류

Get은 선택한 ID를 `member_id`로 bind한 fresh Member를 fetch합니다. 입력 Resource의 id를 constructor의 별도 id Body로 복사하지 않으므로, 응답 `member`가 달라지고 응답 id가 없으면 결과의 canonical id도 응답 member로 달라질 수 있습니다. Add/Update도 constructor 속성 위에 응답의 declared Body를 overlay합니다. 결과 모델의 raw JSON, 일곱 Body 속성과 고정 URI image_id·computed location, missing/null·alternate ID 규칙은 [멤버 레코드 설명](member-records.md#declared-resource와-실제-응답)을 따릅니다.

조회·추가·수정은 실제 HTTP200..399를 받아 처리합니다. valid JSON nonnull object는 constructor 속성에 overlay하고 응답 unknown 값은 Wire에 남깁니다. Source가 허용하는 invalid JSON syntax·빈 Body는 오류 없이 constructor seed를 유지하며 Wire는 nil입니다. valid JSON의 null·배열·scalar, non-UTF-8, projection 실패는 응답 처리 오류입니다.

accepted 응답의 read·Close·context guard·decode·projection 오류에서는 가능한 submitted Record와 실제 Envelope/Header/StatusCode를 오류와 함께 반환하며, 읽은 object가 있으면 Wire 증거도 보존합니다. 호출자는 `record != nil`과 `err != nil`을 함께 처리해야 합니다. 부분 레코드를 다음 Update의 Record로 넘겨도 identity만 선택하므로, 다시 전송할 status 등 속성은 옵션으로 다시 명시해야 합니다. transport·native rejected status는 성공 레코드를 만들지 않으며 오류의 실제 응답 증거를 확인합니다. `errors.As`의 `resource.ResponseError`, `errors.Is`의 원인·context 오류를 사용할 수 있습니다. 응답 소비·projection 실패 때문에 SDK가 새 mutation을 다시 실행하지 않습니다. native client의 설정된 retry 정책은 별개입니다.

Remove는 actual200..399의 opaque Body와 Header·StatusCode·고정 요청 ImageID/MemberID를 acknowledgement로 보존하며 JSON을 decode하지 않습니다. `WithRemoveImageMemberIgnoreMissing`을 생략하거나 true로 지정하면 **깨끗한 최종 native404만** `nil, nil`로 끝납니다. false이면 missing 오류입니다. 404의 read·Close·context·Source 변경 오류를 ignore_missing으로 숨기지 않습니다. accepted 응답 소비 실패의 acknowledgement와 error는 함께 반환합니다. 이미 성공한 mutation을 rollback하거나 완료 상태를 poll하지 않습니다.

## 고정 Source와 범위

이 문서의 공개 proxy 근거는 openstacksdk pin `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`입니다.

- [add_member 1222–1241](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1222-L1241), [get_member 1316–1336](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1316-L1336), [update_member 1356–1386](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1356-L1386), [remove_member 1243–1271](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1243-L1271).
- [Member descriptor 16–43](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/member.py#L16-L43), [generic proxy create/update/delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L669-L799).
- [Resource response translation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338-L1394), [microversion negotiation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1418-L1439), [create](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1519-L1638), [commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881-L1937), [delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2094-L2152).

Glance 권한 분류는 별도 pin `57f7dd9e76ef24e1e9013eceaa703bd442469a24`에 따른 core user API입니다. 멤버 작업 자체에 Swift 연동은 필요하지 않습니다. 실제 cloud의 endpoint/auth/policy 검증, Python runtime과 모든 동적 Resource의 동등성, mutable Resource 공통 모델(SDK-R1), Adapter cache(SDK-C1), 동적 session 설정(SDK-S1)은 별도 범위입니다. Go의 immutable 입력·raw projection·concrete 옵션·acknowledgement 확장을 Python runtime 전체와의 완전한 동등성으로 해석하지 않습니다.
