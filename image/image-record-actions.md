# Glance 이미지 레코드 비활성화·재활성화: Python과 Go

`Connection.DeactivateImageRecord/ReactivateImageRecord`와 `image.Service`의 같은 메서드는 literal 이미지 ID 또는 SDK가 반환한 `ImageRecord`를 받아 고정 POST action을 제출합니다. 공통 `ImageRecordActionRequest`·concrete header 옵션을 사용하며 새 local Record와 실제 응답 ACK를 분리합니다. 이름 검색·자동 GET·discovery·wait를 실행하지 않습니다.

**응답 오류 정책에는 의도적인 차이가 있습니다.** 고정 Python Source의 기본 `Proxy.request`는 `raise_exc=False`이며 action은 Response를 검사하거나 translate하지 않고 버립니다. 따라서 transport가 정상 반환한 `400..599` 응답도 공개 `deactivate_image/reactivate_image`에서 `None`으로 끝날 수 있습니다. transport·session 예외는 여전히 전파될 수 있습니다. Go는 actual `200..399`를 accepted 응답으로 처리하고 `400..599`를 native 증거가 있는 오류로 반환합니다. Python의 기본 status 처리를 그대로 재현한 API라는 뜻이 아닙니다.

| 작업 | Python `conn.image` | Go Connection·Service | HTTP |
|---|---|---|---|
| 비활성화 | `deactivate_image(image)` | `DeactivateImageRecord(ctx, input, options...)` | `POST /images/{id}/actions/deactivate` |
| 재활성화 | `reactivate_image(image)` | `ReactivateImageRecord(ctx, input, options...)` | `POST /images/{id}/actions/reactivate` |

기존 [DeactivateImage/ReactivateImage](mutations.md)는 `resource.Ref`와 actual204 계약을 유지합니다. 새 Record API는 owned identity·local descriptor preparation·opaque200..399 ACK를 제공하는 별도 profile입니다. 고정 Gophercloud `images` 패키지에는 이 두 action의 native 선언이 없으므로 `API.Images.Deactivate/Reactivate` 같은 호출은 제공하지 않습니다.

## Python 사용과 서버 정책

```python
import openstack

conn = openstack.connect(cloud="dev")
conn.image.deactivate_image("image-id")

# 재활성화가 필요한 별도 실행에서 사용한다.
# conn.image.reactivate_image("image-id")
```

고정 Glance pin `57f7dd9e76ef24e1e9013eceaa703bd442469a24`의 [deactivate·reactivate 정책301–328행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L301-L328)은 project scope의 `ADMIN_OR_PROJECT_MEMBER`입니다. 두 작업은 이 기본 정책에서 핵심 user 범위입니다. [controller47–102행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/image_actions.py#L47-L102)은 정책과 이미지 상태를 검사합니다. 비활성화는 active, 재활성화는 deactivated 상태에서 허용됩니다. 실제 role·소유권·policy override·backend·상태는 서버가 판단하며 SDK가 role/status preflight를 재구현하지 않습니다. [권한 근거](../docs/glance-policy-priorities.md)를 함께 참고하세요.

## 독립 Go main

[설치 안내](../docs/install.md)를 따라 아래 코드를 `main.go`로 저장합니다. `go run . -cloud dev -image-id ID -action deactivate`는 실제 비활성화 POST를 한 번 수행합니다. `-action reactivate`는 재활성화 POST 하나를 선택합니다. 문서 검증은 외부 consumer 빌드이며 실제 cloud 인증·상태 변경을 실행한 검증은 아닙니다.

기본 ID 입력에는 GET이 없습니다. `-record`를 지정할 때만 예제가 `conn.Image(ctx)`로 얻은 Service의 `GetImageRecord`를 먼저 호출합니다. `-service`는 선택한 action을 Service 메서드로 실행하며 Connection 메서드와 중복 호출하지 않습니다. 명시적 GET의 오류는 action 옵션으로 숨기지 않습니다.

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
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    imageID := flag.String("image-id", "", "required literal image ID")
    action := flag.String("action", "deactivate", "deactivate or reactivate")
    fromRecord := flag.Bool("record", false, "explicitly fetch an SDK record first")
    viaService := flag.Bool("service", false, "use image.Service for the selected POST")
    flag.Parse()
    if *imageID == "" { log.Fatal("-image-id is required") }
    if *action != "deactivate" && *action != "reactivate" {
        log.Fatal("-action must be deactivate or reactivate")
    }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, *action, *fromRecord, *viaService); err != nil {
        var native gophercloud.ErrUnexpectedResponseCode
        if errors.As(err, &native) {
            fmt.Fprintf(os.Stderr, "native HTTP %d, response bytes=%d\n", native.Actual, len(native.Body))
        }
        var response *resource.ResponseError
        if errors.As(err, &response) {
            fmt.Fprintf(os.Stderr, "accepted HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
        }
        log.Fatal(err)
    }
}

func run(ctx context.Context, cloud, imageID, action string, fromRecord, viaService bool) error {
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    input := image.ImageRecordActionRequest{ID: imageID}
    var service *image.Service
    if fromRecord || viaService {
        service, err = conn.Image(ctx)
        if err != nil { return err }
    }
    if fromRecord {
        record, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: imageID})
        if err != nil { return err }
        input = image.ImageRecordActionRequest{Record: record}
    }
    options := []image.ImageRecordActionOption{
        image.WithImageRecordActionOpts(image.ImageRecordActionOpts{
            Headers: map[string]string{"X-Request-Source": "image-record-actions-example"},
        }),
        image.WithImageRecordActionHeader("X-SDK-Example-Action", action),
        image.WithImageRecordActionHeaders(map[string]string{"X-SDK-Example-Mode": "owned"}),
    }
    var result *image.ImageRecordActionResult
    if viaService {
        if action == "deactivate" {
            result, err = service.DeactivateImageRecord(ctx, input, options...)
        } else {
            result, err = service.ReactivateImageRecord(ctx, input, options...)
        }
    } else if action == "deactivate" {
        result, err = conn.DeactivateImageRecord(ctx, input, options...)
    } else {
        result, err = conn.ReactivateImageRecord(ctx, input, options...)
    }
    return printAction(result, err)
}

func printAction(result *image.ImageRecordActionResult, operationErr error) error {
    value := map[string]any{"partial": result != nil && operationErr != nil}
    if result != nil {
        value["acknowledgement"] = result.Acknowledgement
        if result.Record != nil {
            value["record"] = map[string]any{
                "resource": result.Record.Resource,
                "wire": result.Record.Wire,
                "earlier_receipt_status_code": result.Record.StatusCode,
                "earlier_receipt_body_bytes": len(result.Record.Envelope),
                "import_methods": result.Record.ImportMethods,
            }
        }
    }
    if operationErr != nil { value["error"] = operationErr.Error() }
    body, err := json.MarshalIndent(value, "", "  ")
    if err != nil { return errors.Join(operationErr, err) }
    fmt.Println(string(body))
    return operationErr
}
```

출력은 두 칸 들여쓰기 JSON입니다. ACK의 `Body []byte`는 base64 표현이며 opaque bytes를 JSON object로 decode하지 않습니다. 예제의 1분 context는 caller가 정한 인증·선택적 GET·POST 전체 예산입니다. 하나의 SDK action 호출에서 native retry/reauth 정책에 따라 HTTP 시도가 추가될 수 있습니다.

## 입력과 concrete 옵션

`ImageRecordActionRequest{ID string; Record *ImageRecord}`는 둘 중 하나를 선택합니다. Record는 SDK가 반환한 private raw state를 유지해야 하며 public Resource만 직접 만든 Record는 거부합니다. 대상 ID는 private current body의 canonical `id`입니다. public projected Resource·Wire·Envelope·Header·properties.id 수정으로 action을 retarget하지 않습니다. 이름 검색이나 properties/self/file/schema fallback은 없습니다. 요청과 반환 Record는 caller 입력에서 독립된 snapshot입니다.

literal ID는 nonblank UTF-8 문자열이어야 하며 control 문자·정확한 `.`·`..`는 거부합니다. 허용한 ID는 하나의 escaped URI segment로 전달합니다. private raw JSON identity의 unpaired UTF-16 surrogate escape는 거부하며 valid pair·U+FFFD·literal backslash-u 값은 보존합니다. 자세한 identity 경계는 [이미지 레코드 가이드](image-records.md)에 있습니다. Python의 임의 object coercion·falsy 값·`utils.urljoin`의 slash stripping과 동등하다는 주장은 하지 않습니다.

`ImageRecordActionOpts`는 `Headers map[string]string`을 제공합니다. 기본은 추가 ordinary header 없음이며 요청 Body도 없습니다.

- `WithImageRecordActionOpts`: map을 snapshot하고 **전체 옵션을 교체**합니다. 빈 opts는 앞서 추가한 header를 비웁니다.
- `WithImageRecordActionHeader`: ordinary header 하나를 추가하며 같은 HTTP 이름의 이전 값을 교체합니다.
- `WithImageRecordActionHeaders`: map을 snapshot하고 병합합니다. 이전 helper와 충돌하면 뒤의 helper가 우선하며 supplied map 안의 서로 다른 값인 case alias는 거부합니다.

옵션은 전달 순서대로 한 번 적용합니다. nil/error callback, invalid identity/header·보호된 인증/framing/representation/version header 변경은 HTTP 전 오류입니다. SDK가 location과 callback 전후의 source/context를 검사하며 나중 callback에서 source를 복원해도 관측한 실패를 없애지 않습니다. live 인증 token과 native retry/reauth hooks는 유지합니다. caller가 임의 action 문자열·Resource routing/session 설정·body attributes를 주입하는 builder를 구현할 필요가 없습니다.

## local Record와 실제 ACK

Source의 `_get_resource(Image, image)`는 기존 Resource에 `_update()`를 호출합니다. empty attrs여도 Image header collector는 `ImportMethods`를 빈 목록으로 reset하며 descriptor 값은 현재 location으로 projection됩니다. Go도 HTTP 전에 private current body와 이번 호출 location으로 65필드 Resource view를 준비합니다. descriptor conversion 오류는 body 없는 POST라도 요청을 막을 수 있습니다.

accepted 정상 반환은 `ImageRecordActionResult{Record, Acknowledgement}`입니다. Record의 raw current·original·dirty body와 pending 변경은 그대로 보존합니다. action이 body를 보내거나 pending state를 clean하지 않으며 **local status를 active/deactivated로 합성하지 않습니다.** Record의 Wire·Envelope·Header·StatusCode는 이전 fetch/owned 결과의 receipt를 보존합니다. literal ID 입력은 constructor Record를 만들므로 earlier StatusCode는0, Wire·Envelope·Header는 nil입니다.

Source action에는 `_translate_response`가 없습니다. Go도 응답 Body를 Record에 overlay하지 않으며 응답의 `OpenStack-image-import-methods` header를 Record에 소비하지 않습니다. 준비 때 reset한 `ImportMethods=[]`를 유지합니다. 이번 응답의 Body/Header/StatusCode와 고정 ImageID/Action은 `Acknowledgement *ImageActionResult`에서 확인합니다. Record 상태나 성공 ACK만으로 backend 상태 변경 완료를 별도로 관찰했다고 해석하지 않습니다.

actual200..399 Body는 opaque bytes로 보존합니다. invalid JSON·array·null·non-UTF-8 body를 JSON 형태 때문에 거부하지 않습니다. accepted read·Close·source/context guard 실패에는 partial ACK와 error를 반환하며 `Record=nil`입니다. rejected native status 또는 accepted 응답 없는 transport 실패는 nil result와 원인 오류입니다. 404도 오류이며 ignore_missing 옵션은 없습니다. `errors.As`로 native `gophercloud.ErrUnexpectedResponseCode`와 accepted `resource.ResponseError`, `errors.Is`로 원인·context 오류를 확인합니다. nonnil result만으로 성공을 판단하지 않고 err를 확인하며 accepted 처리 실패 때문에 POST를 replay하지 않습니다. native retry 정책과 caller의 명시적인 재호출은 별도입니다.

## 고정 Source와 비교 범위

openstacksdk pin은 `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`입니다. 공개 [Proxy actions1117–1137행](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1117-L1137)은 [Proxy._get_resource563–602행](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L563-L602)을 사용합니다. [Image._action262–282행](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L262-L282)은 `session.post(url)`만 호출하며 Resource request preparation·response translation·status raise helper를 호출하지 않습니다. 기본 예외 정책은 [Proxy.request209–285행](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L209-L285)의 `raise_exc=False`입니다.

local preparation 비교 근거는 [Resource._update785–806행](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L785-L806), [to_dict1112–1189행](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1112-L1189), [Image header 소비429–437행](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L429-L437)입니다. 이 Go profile의 immutable 입력·Record/ACK 반환과 native status rejection은 명시적인 선택입니다. arbitrary mutable Resource/Munch/subclass와 shared dirty/Header lifecycle은 SDK-R1, Adapter cache invalidation은 SDK-C1, 동적 session/microversion/exception·transport kwargs는 SDK-S1의 별도 공통 범위입니다. 빌드와 고정 계약 테스트로 Python runtime 전체나 실제 cloud policy/backend 동등성을 주장하지 않습니다.
