# Glance 이미지 레코드 자동 수정: Python과 Go

`conn.Image(ctx)`의 `UpdateImageRecord`는 이미지 ID 또는 SDK가 반환한 `ImageRecord`에 변경 속성을 적용하고, 필요한 경우 original/current body에서 JSON Patch를 자동으로 만듭니다. 필드 기본값·별칭·확장 properties와 요청 생략을 SDK가 처리하므로 caller가 builder interface나 Patch 배열을 구현하지 않아도 됩니다.

| Python `conn.image` | Go `image.Service` | 동작 |
|---|---|---|
| `update_image("image-id", name="new")` | `UpdateImageRecord(ctx, ImageRecordUpdateRequest{ID: id, Attributes: attrs}, options...)` | 자동 GET 없이 ID와 변경 속성으로 PATCH 준비 |
| `update_image(existing_image, **attrs)` | 같은 request의 `Record: existingRecord` | 보존된 raw baseline과 변경 속성을 비교 |
| clean Resource에 같은 raw 값 적용 | clean Record에 같은 raw 값 적용 | HTTP 생략 |
| 같은 mutable instance 반환 | 독립 `*ImageRecord` 반환 | Go에서는 반환 Record를 다음 수정에 전달 |

기존 [명시적 UpdateImage·SetImageProperties](update.md)와 [native images.Update](v2/images/update.md)는 유지합니다. 명시적 API는 caller가 지정한 ordered Patch를 보내며 빈 `[]`도 매번 PATCH합니다. 이 가이드의 API는 SDK가 raw component 상태와 자동 diff를 관리하는 별도 호출입니다.

## Python 전체 연결·서비스 사용

```python
import openstack

conn = openstack.connect(cloud="dev")

# ID는 자동 GET·이름 검색 없이 새 Image를 만든다.
updated = conn.image.update_image("image-id", name="release-image")

# 정상 object 응답으로 clean해진 Resource에 같은 raw 값을 넣으면 HTTP를 생략한다.
same = conn.image.update_image(updated, name=updated.name)
assert same is updated
```

고정 [Image proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1056-L1068)는 [Proxy._update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L716-L748)를 통해 Resource를 준비하고 commit합니다. ID 입력은 새 unsynchronized Image, 기존 Image 입력은 그 instance의 `_update(**attrs)`를 사용합니다. Python의 descriptor 표시값과 원래 raw 값이 다르면 표시값을 다시 넣는 것만으로도 변경이 될 수 있습니다. Go의 JSON 표현과 기본값 경계는 [ImageRecord 조회 안내](image-records.md)에 설명합니다.

## 독립 Go main

[설치 안내](../docs/install.md)로 모듈을 준비한 뒤 아래 내용을 `main.go`에 저장합니다. 이 API를 포함하는 가이드와 같은 revision 또는 후속 revision을 사용합니다. `go run . -cloud dev -image-id ID -name release-image`는 실제 이미지 이름을 변경하고, 반환 Record에 같은 이름을 다시 적용합니다. 문서 검증에서는 main을 빌드하며 실제 cloud mutation을 실행하지 않습니다.

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
    name := flag.String("name", "release-image", "new image name")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, *name); err != nil {
        var response *resource.ResponseError
        if errors.As(err, &response) {
            fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
        }
        log.Fatal(err)
    }
}

func printRecord(label string, value *image.ImageRecord, operationErr error) error {
    output := map[string]any{"step": label, "partial": value != nil && operationErr != nil}
    if value != nil {
        output["resource"] = value.Resource
        output["wire"] = value.Wire
        output["receipt_status_code"] = value.StatusCode
        output["receipt_body_bytes"] = len(value.Envelope)
        output["import_methods"] = value.ImportMethods
    }
    if operationErr != nil { output["error"] = operationErr.Error() }
    body, marshalErr := json.MarshalIndent(output, "", "  ")
    if marshalErr != nil { return errors.Join(operationErr, marshalErr) }
    fmt.Println(string(body))
    return operationErr
}

func run(ctx context.Context, cloud, imageID, name string) error {
    if imageID == "" { return fmt.Errorf("-image-id is required") }
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }

    updated, err := conn.UpdateImageRecord(ctx, image.ImageRecordUpdateRequest{
        ID: imageID,
        Attributes: map[string]any{"name": name},
    }, image.WithImageRecordHeader("X-Request-Source", "image-record-update-example"))
    if err := printRecord("update by ID", updated, err); err != nil { return err }

    // name has no descriptor conversion: use the returned raw JSON spelling.
    sameName := json.RawMessage(updated.Resource.Body["name"])
    unchanged, err := service.UpdateImageRecord(ctx, image.ImageRecordUpdateRequest{
        Record: updated,
    }, image.WithImageRecordAttributes(map[string]any{"name": sameName}))
    return printRecord("reuse record with same name", unchanged, err)
}
```

출력은 두 칸 들여쓰기 JSON입니다. 정상 object 응답이 body를 clean한 경우 두 번째 호출은 HTTP를 생략합니다. 첫 응답이 빈 본문이나 invalid JSON이면 pending 변경이 남아 두 번째 호출이 PATCH를 다시 보낼 수 있습니다. no-op의 StatusCode·Envelope·Header는 이전 요청의 receipt를 유지하므로 새로운 응답이 있었다는 뜻이 아닙니다. 모든 경우 err를 확인합니다. 예제의 1분 context는 인증과 두 호출을 함께 제한하는 caller 선택입니다.

전체 Connection의 `conn.UpdateImageRecord(ctx, request, options...)`도 같은 Image 서비스에 위임합니다. 이미 준비한 Gophercloud client는 `image.New(client)`로 같은 API를 사용합니다. `NewWithDependencies(client, image.Dependencies{CloudLocation: getter})`는 현재 location을 제공합니다.

## 입력·옵션·identity

`ImageRecordUpdateRequest`는 `ID string` 또는 `Record *ImageRecord` 중 하나와 `Attributes map[string]any`를 받습니다. `Record`에는 SDK가 반환한 private raw state가 있어야 합니다. `&ImageRecord{Resource: ...}`처럼 표시용 Resource만 직접 만든 값은 commit baseline이 없어 거부합니다. `GetImageRecord`, 목록·검색, 이전 `UpdateImageRecord`나 record tag API에서 받은 Record를 사용합니다.

ID는 nonblank UTF-8이고 control 문자와 정확한 `.`·`..`를 거부합니다. 허용한 ID는 단일 escaped URI segment로 `PATCH /images/{id}`에 전송합니다. 이름 검색·자동 GET·후속 GET·wait는 없습니다. response의 self/file/schema URL, computed location과 properties 안의 id를 요청 주소로 사용하지 않습니다.

literal ID와 direct attribute `id`를 함께 지정하면 같은 값이어도 preflight 오류입니다. Python의 ID constructor도 `id` keyword 충돌이 발생합니다. supplied Record의 direct `id` 변경은 현재 Resource identity를 변경합니다. clean Record에서 id만 바꾸면 id dirty를 제외한 뒤 HTTP를 생략합니다. 기존 pending dirty나 다른 변경이 있으면 준비된 새 id 경로로 PATCH하며 original과 다른 id도 diff에 참여할 수 있습니다. `properties: {"id": ...}`는 펼쳐진 PATCH body만 바꾸며 URL 선택에는 관여하지 않습니다. 서버는 id·다른 readonly 필드의 실제 수정 허용 여부를 판단합니다.

기존 `ImageRecordOption`을 사용합니다. `WithImageRecordAttribute/Attributes`는 속성을, `WithImageRecordHeader/Headers`는 일반 요청 헤더를 지정하고 `WithImageRecordOpts`는 option 설정 전체를 교체합니다. request Attributes에 option Attributes를 반영하며 같은 이름은 마지막 option 값이 이깁니다. Go map 이름은 정렬한 순서로 처리하므로 Python kwargs의 임의 insertion order와 다른 별칭 충돌 결과가 날 수 있습니다. raw response의 별칭은 실제 JSON object 순서로 처리합니다.

SDK는 request Attributes·raw bytes·Record를 현재 location callback 전에 고정하고 option 설정을 복사합니다. option Attributes는 고정한 request Attributes 위에 반영합니다. callback은 호출당 한 번 적용하며 source/context를 앞뒤로 검사합니다. nil/error option, encoding 오류, invalid header나 보호된 auth·framing·version·representation 헤더 override는 HTTP 전에 오류입니다. ordinary header를 추가하는 것만으로 body를 dirty하게 만들거나 no-op에 PATCH를 강제하지 않습니다.

Go의 고정 owned profile은 bound argument·routing·constructor 제어와 혼동하지 않도록 direct `self`, `image`, `base_path`, `microversion`, `connection`, `_synchronized`를 일괄 거부합니다. Python은 입력 branch마다 이를 다르게 소비할 수 있으며 supplied Image의 `connection`·`_synchronized`는 unknown properties가 될 수도 있습니다. nested explicit properties의 같은 이름은 일반 property data입니다. 동적 route·session·microversion을 임의 attribute로 설정하는 계층은 아닙니다.

## raw 값·no-op·자동 JSON Patch

Record의 private current/original body는 [조회 시 raw 값](image_records_model.go)을 descriptor projection 전에 저장합니다. missing, null, `[]`와 scalar를 구분하고 getter 기본값이나 conversion을 commit baseline으로 합성하지 않습니다. public `Record.Resource`, `Wire`, `Envelope`를 직접 변경해 baseline을 덮어쓸 수 없습니다. 수정은 request Attributes 또는 `WithImageRecordAttributes`로 전달합니다.

| 상태·입력 | 동작 |
|---|---|
| ID만 지정 | body의 id dirty를 제외한 뒤 no-op, constructor Record 반환 |
| clean Record에 현재 raw 값과 같은 속성 | no-op, 독립 Record 반환 |
| missing name에 null, missing tags에 `[]` | 기본 view와 같아 보여도 새 raw component이므로 변경 |
| raw `"false"`를 bool descriptor에 입력 | 표시값은 true일 수 있어도 PATCH는 string 값 |
| raw `"04"`를 size에 입력 | 표시값은 4일 수 있어도 PATCH는 string 값 |
| 변경 후 original 값으로 되돌렸으나 pending dirty가 남음 | diff가 비어도 `[]` PATCH 가능 |
| tags API로 current list만 바꾼 뒤 다른 body 속성을 수정 | original/current 전체 diff에 현재 tags도 참여 |

현재 raw 값 비교에는 JSON object·array의 재귀 비교와 Python의 bool/number 동등성을 적용합니다. 예를 들어 present true와 1, false와 0은 same-value로 처리할 수 있습니다. JSON을 벗어난 Python object나 Python float rounding을 모두 재현하는 계약은 아닙니다. 속성 값은 Go `encoding/json`으로 표현 가능한 JSON이며 큰 숫자와 exact raw JSON에는 `json.RawMessage`를 사용할 수 있습니다.

PATCH는 dirty 필드만 뽑은 upsert 배열이 아니라 펼쳐진 original/current 전체를 비교합니다. nested object·array·null·값 형태 변경과 property removal을 add·remove·replace로 표현하고 move·copy는 생성하지 않습니다. JSON Pointer token의 `~`·`/`를 각각 `~0`·`~1`로 escape합니다. object key는 escaped token 순서로 처리하고 array는 공통 subsequence를 보존하며 남은 값을 결정적인 index 순서로 편집합니다. 생성한 operation은 그 순서대로 적용해야 하며 array index 의미를 바꿀 수 있는 전체 path 재정렬을 하지 않습니다. 실제 depth·array index·readonly·property schema 허용 여부는 Glance가 판단합니다.

자동 diff는 RFC JSON 수치 동등성으로 `1`과 `1.0`을 같게 보고 bool과 number를 구분합니다. 이 정책은 dirty 결정에 쓰는 Python bool/number 동등성과 별개입니다. Python jsonpatch의 특정 버전은 scalar의 JSON 직렬화 표현을 비교하거나 array에 Python 동등성을 적용할 수 있으므로 수치 표현·bool/number의 diff bytes도 동일하다고 약속하지 않습니다.

고정 [Resource patch 준비](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1251-L1273)는 `jsonpatch.make_patch`의 결과를 path로 정렬합니다. Source의 [dependency 선언](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/pyproject.toml#L20)은 `jsonpatch!=1.20,>=1.16`이므로 SDK revision만으로 diff runtime 버전이 고정되지는 않습니다. Go는 실행 가능한 array 편집 순서를 유지하므로 Source의 전체 path 정렬과도 구분됩니다. 이 정렬 차이는 외부 dependency 버전 차이와 별도인 명시적 Go 정책입니다. 모든 Python jsonpatch 버전의 move 최적화·최소 operation 수·동일 path tie 순서·array operation bytes를 재현하지 않습니다. Go decoder는 encoding/json의 duplicate-key·escaped surrogate 처리를 사용하므로 임의 Python string object와 exact JSON spelling까지 같은 결과를 주장하지 않습니다.

## properties 교체와 wire 별칭

canonical 이름과 wire 이름을 인식합니다. 예를 들어 `is_hidden`은 `os_hidden`, `is_protected`는 `protected`, `owner_id`와 `owner`는 같은 wire `owner`에 반영합니다. baseline은 raw component이고 PATCH는 wire 이름으로 작성하므로 getter 기본값·canonical 별칭을 서버 필드로 보내지 않습니다.

unknown top-level attributes는 incoming properties로 묶습니다. explicit properties가 dict이면 incoming unknown 속성을 거기에 합치며, 기존 Record의 properties 집합에 자동 merge하지 않습니다. 예를 들어 기존 `{"team":"platform","release":"old"}`에 Attributes `{"release":"new"}`만 전달하면 properties component는 `{"release":"new"}`로 바뀌므로 자동 diff에 team 제거가 포함됩니다. 기존 속성을 유지하려면 유지할 key를 포함한 전체 properties map을 명시합니다.

explicit properties가 dict가 아닌 상태에서 unknown 속성도 있으면 `{"properties": oldValue, ...unknown}`으로 묶습니다. PATCH 준비에서는 properties dict를 wire root의 declared fields 뒤에 펼치며 같은 key가 있으면 properties 값이 이깁니다. truthy string properties는 root properties string으로 남고 null·false·0·empty container·empty string과 다른 non-dict/non-string 형태는 flatten 결과에서 제외합니다. 이는 [Source packing·flatten](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1207-L1229)의 동작이며 일반적인 merge-patch 규칙으로 바꾸지 않습니다.

## 응답·receipt·재사용

Source [Image 요청](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L439-L463)에 맞춰 Content-Type은 `application/openstack-images-v2.1-json-patch`, Accept는 빈 값입니다. shared REST 전송은 이 representation headers를 고정합니다. native retry가 이를 변경하면 다음 요청 전에 오류로 중단하고 이전 native 거부 응답의 증거를 보존합니다. 일반 retry 헤더와 live 인증 token의 기존 정책은 유지합니다. owned 성공 범위는 actual HTTP 200..399이며 native strict200 Update와 별도입니다. request body를 서버 상태와 자동 동기화하거나 native retry·reauth 정책을 대체하지 않습니다.

valid object 응답은 self를 제외하고 declared 필드·unknown properties를 current body에 overlay합니다. 응답에서 생략한 declared 필드는 제출한 current 값이 남습니다. 응답 properties는 fetch 규칙으로 새 집합을 만들며 valid `{}`도 properties `{}`를 적용합니다. 성공한 object translation 뒤 body를 clean하고 full current를 original로 저장합니다.

빈 body나 invalid JSON syntax는 tolerated 성공으로 submitted current seed를 유지합니다. body를 clean하지 않고 이전 original·pending dirty를 보존하므로 반환 Record를 attrs 없이 다시 commit하면 이전 변경을 재전송할 수 있습니다. `GetImageRecord`도 이런 응답에서 constructor의 raw seed attributes를 pending 상태로 보존하므로, 그 반환 Record를 UpdateImageRecord에 전달해 아직 동기화되지 않은 속성을 commit할 수 있습니다. 이미 접수된 변경의 재시도가 필요한지는 caller가 판단합니다. valid JSON null·array·scalar는 Image object가 아니므로 decode 오류입니다. descriptor conversion 오류나 nonUTF-8 응답도 오류입니다.

정상 반환 Record의 `Resource`는 current declared view이며 `Wire`, `Envelope`, `Header`, `StatusCode`는 실제 새 응답의 증거입니다. invalid JSON일 때 Wire는 nil이어도 실제 Envelope·Header·StatusCode는 남습니다. 응답의 import methods는 `ImportMethods`로 새로 읽고 이전 값을 대체·reset합니다. `OpenStack-image-import-methods` attribute는 plain header 속성으로 소비하며 body property나 body dirty의 근거로 쓰지 않습니다. computed location은 이번 호출에서 캡처한 현재 location입니다.

HTTP를 생략하면 supplied Record의 이전 receipt를 보존하고 독립 copy를 반환합니다. ImportMethods는 `_update` 준비 때 reset하며 literal `OpenStack-image-import-methods` attribute가 있으면 그 값을 소비하므로, no-op이어도 이전 ImportMethods를 그대로 유지하는 것은 아닙니다. ID-only constructor에는 Wire·Envelope·Header가 nil, StatusCode가 0입니다. Record와 receipt의 map·raw bytes, caller 입력은 독립이며 다음 호출에 이전 mutable getter view를 자동 채택하지 않습니다.

HTTP 전 실패, accepted 응답이 없는 transport 실패와 native HTTP 거부에는 nil Record와 오류를 반환합니다. HTTP 거부의 실제 body·header·status는 `gophercloud.ErrUnexpectedResponseCode`에서 확인하며 accepted `resource.ResponseError`로 바꾸지 않습니다.

accepted body Read/Close·source/context·decode·projection 실패에는 partial 제출 Record와 오류를 함께 반환합니다. 그 Record의 Resource는 제출한 declared view이며 가능한 actual Envelope·Header·StatusCode, object로 읽을 수 있는 Wire를 별도로 보존합니다. 이 경우 body original·pending dirty를 유지하고 clean한 성공으로 처리하지 않습니다. `resource.ResponseError`에도 실제 응답과 원인 오류가 남습니다. nonnil Record나 보존된 receipt만으로 성공을 판단하지 말고 err를 확인합니다. 오류가 반환됐어도 서버가 이미 변경을 접수했을 수 있습니다.

`errors.As`로 response/native HTTP 오류를, `errors.Is`로 context·custom cause를 확인합니다. accepted 처리 실패를 자동 성공으로 바꾸거나 accepted PATCH를 replay하지 않습니다. 반환한 pending Record를 다음 호출에 전달하는 것은 caller의 명시적 재시도입니다. Python은 같은 mutable Resource에 요청 전 변경을 남길 수 있고 response overlay 후 conversion 단계에서 실패할 수도 있습니다. Go가 오류 때 제출 snapshot을 보존하는 것은 이 arbitrary mutable lifecycle과 구분하는 정책입니다.

## Python 비교 범위·검증

이 계층은 고정 Image의 JSON body를 이용하는 ID/Record 수정, no-op, aliases·properties, 자동 diff와 응답 후 baseline을 제공합니다. Go에서 caller Record를 바꾸거나 같은 pointer를 반환하지 않으므로 반환값을 다음 호출에 전달합니다. Python의 arbitrary Resource/Munch/subclass·공유 descriptor default alias·동시 mutable component history 전체는 SDK-R1의 공통 범위로 남습니다. Python Adapter cache·conditional GET은 SDK-C1, 동적 session·base_path·microversion·exception class와 conflict retry policy는 SDK-S1에 남습니다.

이미지 수정·필드 schema·property protection·readonly·서버 policy·locations backend 효과는 서버가 판단합니다. client에서 JSON을 표현하고 PATCH를 만들 수 있다는 사실이 모든 JSON 형태의 서버 저장 성공이나 cloud rollback·원자성을 증명하지 않습니다. 핵심 user 경로와 admin-only 필드의 실제 배포 권한도 구분합니다.

[owned 수정 테스트](image_record_update_test.go)는 sparse raw presence·same-value·properties 교체·sticky pending·identity·receipt·preflight와 retry guard를 다룹니다. [전체 Connection 테스트](../image_record_update_integration_test.go)는 공통 service binding·live token·현재 location과 반환 Record의 재사용을 확인합니다. [shared diff 테스트](../internal/jsonpatch/diff_test.go)는 생성 코드와 별도인 patch 적용기로 nested array와 JSON 변경 결과를 검증합니다. 기존 ImageRecord/task transport·body fixture와 공개 Gophercloud testhelper를 재사용하며 별도 HTTP 서버를 추가하지 않습니다.

고정 Source의 [component 변경](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L213-L223), [commit/no-op](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1875-L1937), [응답 translation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338-L1394)을 대조한 JSON profile입니다. local fixture와 문서 빌드는 Python/jsonpatch runtime 실행·실제 cloud 인증·API 검증과 별개입니다. 구체적인 테스트와 실행 결과, 남은 operation·공통 계약은 [지원 판정대장](../docs/sdk-support-ledger.md)에서 확인합니다.
