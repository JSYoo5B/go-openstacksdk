# Glance 이미지 레코드 태그 추가·삭제: Python과 Go

`conn.Image(ctx)`의 `AddImageRecordTag`와 `RemoveImageRecordTag`는 literal 이미지 ID 또는 supplied [ImageRecord](image-records.md)에 태그 하나를 추가·삭제합니다. SDK가 고정 요청, 입력 snapshot과 성공 후 로컬 tags 변경을 처리하고, 이미지의 기존 조회 증거와 새 태그 요청의 실제 접수 증거를 분리합니다.

| Python `conn.image` | Go `image.Service` | HTTP |
|---|---|---|
| `add_tag(image_or_id, tag)` | `AddImageRecordTag(ctx, ImageRecordTagRequest{...}, tag, options...)` | `PUT /images/{id}/tags/{tag}` |
| `remove_tag(image_or_id, tag)` | `RemoveImageRecordTag(ctx, ImageRecordTagRequest{...}, tag, options...)` | 같은 경로 DELETE |

공개 Python proxy는 None을 반환하고 내부 TagMixin은 변경한 같은 Resource를 반환합니다. Go는 `(*ImageRecordTagResult, error)`로 private 변경 Record와 별도 Acknowledgement를 제공합니다. 기존 strict 204 `AddTag/RemoveTag`, native `service.API.Images`의 호출과 반환형은 유지합니다.

## Python 전체 연결·서비스 사용

```python
import openstack

conn = openstack.connect(cloud="dev")

# ID는 새 Image를 만들며 자동 GET을 하지 않는다. 공개 proxy 결과는 None이다.
result = conn.image.add_tag("image-id", "reviewed")
assert result is None

# 실제 조회한 Resource를 전달하면 HTTP 성공 후 이 객체의 local tags를 갱신한다.
image = conn.image.get_image("image-id")
conn.image.remove_tag(image, "reviewed")
print(image.tags)
```

[고정 Image proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1196-L1220)는 ID/Resource로 Image를 준비한 뒤 [TagMixin add/remove](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/tag.py#L105-L145)를 호출합니다. 요청 body나 반환 body로 tags를 새로 읽지 않고, HTTP 오류 판정 뒤 descriptor의 현재 tags list를 직접 변경합니다. Go의 concrete ImageMutationOption은 별도 builder 없이 일반 요청 헤더를 선택하도록 제공합니다.

## 독립 Go main

[설치 안내](../docs/install.md)로 모듈을 준비하고 아래 내용을 `main.go`에 저장합니다. 신규 ImageRecordTag API를 포함하는 이 가이드의 revision 또는 후속 revision이 필요합니다. `go run . -cloud dev -image-id ID -tag reviewed`는 실제 태그를 추가한 뒤 조회하고 삭제합니다. 문서 검증에서는 빌드만 하며 이 mutation을 실행하지 않습니다.

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
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    imageID := flag.String("image-id", "", "required literal image ID")
    tag := flag.String("tag", "reviewed", "literal tag to add and then remove")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, *tag); err != nil {
        var response *resource.ResponseError
        if errors.As(err, &response) {
            fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
        }
        log.Fatal(err)
    }
}

func printResult(value *image.ImageRecordTagResult, operationErr error) error {
    if value != nil {
        output := make(map[string]any)
        if ack := value.Acknowledgement; ack != nil {
            output["acknowledgement"] = map[string]any{
                "image_id": ack.ImageID, "tag": ack.Tag,
                "status_code": ack.StatusCode, "body_bytes": len(ack.Body),
            }
        }
        if value.Record != nil {
            output["resource"] = value.Record.Resource
            output["wire"] = value.Record.Wire
            output["record_status_code"] = value.Record.StatusCode
        }
        body, err := json.MarshalIndent(output, "", "  ")
        if err != nil { return errors.Join(operationErr, err) }
        fmt.Println(string(body))
    }
    return operationErr
}

func run(ctx context.Context, cloud, imageID, tag string) error {
    if imageID == "" { return fmt.Errorf("-image-id is required") }
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }

    added, err := service.AddImageRecordTag(ctx, image.ImageRecordTagRequest{ID: imageID}, tag,
        image.WithImageMutationHeader("X-Request-Source", "image-record-tags-example"))
    if err := printResult(added, err); err != nil { return err }

    seed, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: imageID})
    if err != nil { return err }
    removed, err := service.RemoveImageRecordTag(ctx, image.ImageRecordTagRequest{Record: seed}, tag,
        image.WithImageMutationHeaders(map[string]string{"X-Request-Source": "image-record-tags-example"}))
    return printResult(removed, err)
}
```

이 main은 전체 Connection에서 Image 서비스를 얻어 ID constructor add와 실제 GET seed remove를 사용합니다. `printResult`는 오류가 있어도 보존된 접수를 출력하고 오류를 그대로 반환합니다. 접수가 남았다고 다음 단계를 실행하지 않습니다. 예제의 1분 parent context는 인증·두 mutation·GET을 함께 제한하는 caller 선택이며 library 기본 timeout이 아닙니다.

이미 준비한 Gophercloud client는 `image.New(client)`로 같은 API를 사용합니다. `NewWithDependencies(client, image.Dependencies{CloudLocation: getter})`는 현재 location을 공급합니다. 실제 cloud·server policy 검증과 문서 빌드는 구분합니다.

## 필수 입력·concrete 옵션

`ImageRecordTagRequest`는 `ID string` 또는 `Record *ImageRecord` 중 하나를 선택합니다. 둘을 동시에 지정하거나 literal identity를 얻을 수 없으면 HTTP 전에 오류입니다. 이름 검색·List·자동 GET을 하지 않으며 tag 자체도 literal string입니다.

- ID는 nonblank UTF-8이고 control 문자와 정확한 `.`·`..`를 거부합니다.
- tag는 빈 문자열·invalid UTF-8·control 문자·정확한 `.`·`..`를 거부합니다. whitespace-only tag, slash·backslash, 255자를 넘는 값은 client가 trim·축약·schema 검증하지 않습니다.
- 허용된 ID/tag를 각각 단일 escaped URI segment로 보냅니다. route·version·origin은 고정하며 wire self/schema/location URL을 따라가지 않습니다.
- Source [urljoin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L43-L50)은 각 문자열 양끝 slash를 strip하고 중간 slash를 그대로 연결합니다. Go의 literal escaping과 validation은 명시적 경계이며 모든 비정상 문자열의 Source URL 결과를 재현하지 않습니다.

기존 `ImageMutationOpts{Headers map[string]string}`와 `WithImageMutationOpts/Header/Headers`를 재사용합니다. 전체 Opts는 설정을 교체하고 helper는 순서대로 헤더를 추가·교체합니다. map·option slice를 복사하고 option callback은 operation마다 한 번 실행하며 callback 사이 source/context를 검사합니다. nil/error option, invalid header와 보호된 auth·framing·version·representation header override는 preflight 오류입니다. caller가 request builder·custom interface를 구현할 필요가 없습니다.

## ID constructor와 supplied record

ID를 지정하면 자동 조회 없이 Image의 65필드 기본 Resource를 만듭니다. id는 입력값, tags는 `[]`, properties는 null, 나머지 declared 기본값과 captured current location을 적용합니다. ImportMethods는 빈 slice이며 Wire·Envelope·Header는 nil, record StatusCode는 0입니다. 이는 HTTP fetch receipt가 없는 constructor이며 mutation의 상태를 record StatusCode에 넣지 않습니다.

Record를 지정하면 caller Record·Resource·Wire·Envelope·Header·ImportMethods와 raw bytes를 복사하고 supplied location을 유지합니다. 다른 declared/unknown 필드와 기존 receipt를 다시 projection·fetch하지 않습니다. 현재 cloud 사실·options는 한 번 준비하지만 supplied record의 location을 현재 값으로 덮어쓰지 않습니다. 입력 Record는 변경하지 않습니다. 고정 public Proxy의 `_get_resource(existing Image)`는 `Resource._update(**{})`를 거쳐 그 Resource에 연결된 connection의 현재 location을 다시 넣고 microversion·component/to_dict도 처리합니다. Go의 supplied location·다른 raw 필드 보존은 이 전체 lifecycle과 구분하는 정책이며 TagMixin의 직접 tag 변경만 재사용합니다. [Source 준비](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L563-L602)와 [Resource 업데이트](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L785-L839)를 비교한 경계입니다.

이 동작은 고정 ImageRecord 입력의 private snapshot입니다. supplied Python Image/Munch/subclass의 same-instance·dirty state·동적 Adapter/session/cache를 재현하는 입력이 아니며 SDK-R1/C1/S1의 공통 비교 범위는 남습니다.

## 성공 후 로컬 tags 변경

actual HTTP 200..399 접수를 받은 뒤 tags descriptor를 읽고 local list를 갱신합니다. 응답 body는 tags나 이미지 Resource로 해석하지 않습니다.

| 로컬 입력 | Add(tag) | Remove(tag) |
|---|---|---|
| missing tags | `[]`에서 append | `[]` 유지 |
| 배열 | 임의 raw member를 유지해 끝에 append | 정확히 일치하는 첫 string member 하나만 제거 |
| nonnull scalar/object | singleton list로 변환해 append | singleton list에서 같은 규칙 적용 |
| explicit null | 접수 뒤 local 오류 | 접수 뒤 local 오류 |

예를 들어 `['a', 'a', 'b']`에 a를 추가하면 `['a', 'a', 'b', 'a']`, a를 삭제하면 `['a', 'b']`입니다. remove할 tag가 로컬 목록에 없으면 local list는 그대로입니다. 비교는 case-sensitive literal string이며 숫자·bool·object를 문자열 tag로 형변환하거나 중복 제거하지 않습니다. JSON의 lone UTF-16 surrogate는 valid U+FFFD tag와 일치하지 않는 raw member로 보존합니다. 정상 surrogate pair와 escaped backslash literal은 decoded string으로 비교하며 이 보완은 encoding/json의 대체 문자 때문에 다른 tag를 삭제하지 않기 위한 JSON 경계입니다.

이 로컬 변경은 서버의 전체 tag set을 새로 조회한 결과가 아닙니다. Glance의 실제 저장소 상태·중복 처리·policy·길이 제한은 서버가 판단합니다. literal ID add는 다른 기존 서버 tag를 알 수 없고 supplied record도 오래된 상태일 수 있습니다. 실제 현재 값이 필요하면 별도 GetImageRecord를 사용합니다.

삭제가 서버 404로 거부되면 오류를 반환하며 local missing noop로 바꾸지 않습니다. 로컬 noop는 HTTP가 성공했을 때만 적용합니다. Source의 HTTP 이후 tags descriptor 접근 순서를 따라 explicit null의 local 오류를 접수 뒤 처리합니다. malformed JSON/nonUTF8 tags는 Go RawResource 입력의 별도 처리 오류이며 같은 후행 순서를 적용합니다. 이 오류가 반환될 때 이미 서버에 변경이 적용됐을 수 있습니다. 이 경우 Acknowledgement를 유지하고 Record는 nil이며 오류를 반환합니다.

## Record와 Acknowledgement·부분 실패

`ImageRecordTagResult`는 `Record *ImageRecord`와 `Acknowledgement *ImageRecordTagAcknowledgement`를 제공합니다. Acknowledgement에는 `ImageID`, `Tag string`, `Body []byte`, `Header http.Header`, `StatusCode int`가 있고 Body는 실제 opaque bytes입니다. JSON이 아닌 본문이나 빈 본문도 접수 증거로 보존합니다.

Record는 성공한 local tags 변경을 반영하고 기존 Wire·Envelope·Header·StatusCode는 원래 image fetch의 증거로 유지합니다. 따라서 Record.Resource의 tags와 이전 Record.Wire의 tags가 다를 수 있습니다. 새 mutation 응답은 Acknowledgement에만 남고, 로컬 tags를 서버가 반환한 필드라고 Wire에 합성하지 않습니다. constructor에는 원래 fetch 증거를 만들지 않습니다.

HTTP 전 실패나 accepted receipt가 없는 native 거부·transport 실패는 nil result와 오류를 반환합니다. accepted 응답의 Read/Close·source/context·hook 오류 또는 local tags 갱신 실패에는 가능한 Acknowledgement와 오류를 반환하고 Record는 nil입니다. 정상 응답의 body를 image JSON으로 decode하지 않습니다. 보존된 접수나 nil이 아닌 result만으로 operation 성공을 판단하지 않고 항상 err를 확인합니다.

native 인증·reauth·retry 정책을 유지하며 captured endpoint/client/provider/service binding과 보호된 source를 guard합니다. 이 owned API의 native 400..599에는 기존 shared rejected observer를 적용해 Read/Close·context·source 원인을 native HTTP 오류와 함께 보존합니다. rejected 응답은 Acknowledgement로 반환하지 않습니다. RetryFunc가 같은 plain native input 오류를 그대로 반환하면 중복 원인으로 join하지 않습니다. 기존 strict 204 API의 rejected I/O 정책을 바꾸는 것은 아닙니다. 반환 record와 접수의 map·raw bytes는 caller와 서로 독립입니다. caller가 접수 body/header 또는 local Resource를 변경해 다른 channel이나 이미 준비한 operation의 고정 target을 바꾸지 못합니다. accepted 처리 실패의 ResponseError도 bounded 실제 body·header·status를 보존합니다.

## 권한·기존 API·검증 범위

이미지 tag는 metadata 사전의 `metadef_tags`와 별도 API입니다. [고정 Glance tag 정책](../docs/glance-policy-priorities.md#이미지-태그-추가삭제의-기본-정책)의 controller는 PUT/DELETE 모두 modify_image를 검사하며 기본 rule은 project scope의 ADMIN_OR_PROJECT_MEMBER입니다. project-owned 이미지에 접근하는 member 경로가 있어 핵심 user에 배치합니다. SDK가 admin 권한·소유권·visibility를 미리 판정하지 않으며 실제 배포의 override와 server 제한은 서버가 결정합니다.

기존 `service.AddTag/RemoveTag(ctx, resource.Ref, tag, options...)`의 strict 204·ACK-only 결과와 명시 Name resolver는 유지합니다. 기존 native `images.Update`의 `ReplaceImageTags` 전체 목록 PATCH도 유지합니다. 새 API는 Source의 sub400 성공 후 Image local list 변경과 private Resource/실제 접수 반환을 제공하는 별도 계층입니다.

고정 public proxy·TagMixin의 ID/Resource 선택·요청·local list 변경을 확인한 범위입니다. Go의 literal/missing-tags 기본 목록은 호출마다 독립적입니다. Python의 mutable descriptor default alias·same-instance·component/to_dict와 dirty state는 SDK-R1에 남습니다. Python Resource 자체의 재사용·subclass·arbitrary attributes·cache/session 전체는 지원 판정을 확대할 근거가 아니며 공통 pending 범위를 유지합니다. local fixture와 문서 빌드는 실제 mutation·cloud 인증 또는 Python runtime 실행을 대신하지 않습니다.

[owned tag 테스트](image_record_tag_test.go)는 ID/record 선택·local list/descriptor·독립 receipt·opaque 200..399·nullable/invalid tags의 후행 오류·header snapshot·accepted/rejected physical guard·native hooks·legacy204·preflight·Unicode string 경계를 다룹니다. [Connection 태그 테스트](../connection_image_record_tags_test.go)는 전체 연결에서 공유 client·현재 location과 mutation 증거의 분리를 확인합니다. 기존 ImageRecord/task transport·body fixture와 공개 Gophercloud testhelper, shared REST observer를 재사용하며 별도 HTTP 서버·parser·fault 엔진·dependency를 추가하지 않습니다. 실행 결과는 [지원 판정대장](../docs/sdk-support-ledger.md)에서 확인합니다.
