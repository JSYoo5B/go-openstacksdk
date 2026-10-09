# Glance 이미지 레코드 import: Python과 Go

`Connection.ImportImageRecord`와 `image.Service.ImportImageRecord`는 SDK가 반환한 `ImageRecord`의 private 형식·상태를 복사하고 한 번 import POST를 제출합니다. concrete 옵션이 기본 import method, raw JSON 입력, store 선택과 확장 필드를 처리합니다. 별도 builder나 store resolver를 구현할 필요가 없습니다.

`container_format`과 `disk_format`이 Python truthiness 기준으로 참이어야 합니다. helper는 초기 GET·이름 조회를 하지 않으므로 먼저 `GetImageRecord`로 조회하거나 앞선 owned staging의 반환 Record를 전달합니다. literal ID로 새 Resource를 만들면 두 형식이 없어 HTTP 전에 실패합니다. 이미지 status 검사는 서버가 담당합니다.

| 작업 | Python `conn.image` | Go Service·Connection |
|---|---|---|
| 명시적 조회 | `record = conn.image.get_image(id)` | `service.GetImageRecord(ctx, image.ImageRecordRequest{ID: id})` |
| 기본 제출 | `import_image(record)` | `ImportImageRecord(ctx, image.ImageRecordImportRequest{Record: record})` |
| web 다운로드 | `method="web-download", uri=url` | `WithImageRecordImportMethod("web-download")`, `WithImageRecordImportURI(url)` |
| 다른 Glance에서 다운로드 | `method="glance-download", remote_region=..., remote_image_id=..., remote_service_interface=...` | 같은 세 값을 `WithImageRecordImportRemote...`로 지정 |
| 다른 store로 복사 | `method="copy-image", stores=[...]` | 같은 `ImportImageRecord`에 method와 plural stores 옵션 지정 |
| singular store | `store="archive"` | `WithImageRecordImportStore(ImageRecordImportStore{ID: "archive"})` |
| 실제 응답 | raw `requests.Response` | `result.Acknowledgement`와 준비된 `result.Record` |

`glance-direct`는 이미 staging된 데이터를 사용합니다. import helper가 binary 데이터를 읽거나 staging·이미지 생성·active 대기·checksum·자동 삭제를 수행하지 않습니다. [owned staging 가이드](image-record-stage.md)에 데이터 준비와 필수 metadata fetch를 설명합니다.

## Python 사용

```python
import openstack

conn = openstack.connect(cloud="dev")
record = conn.image.get_image("image-id")
response = conn.image.import_image(
    record,
    method="web-download",
    uri="https://images.example.org/image.qcow2",
    stores=["fast", "archive"],
    all_stores_must_succeed=False,
)
print(response.status_code)
```

응답은 접수 증거입니다. backend 저장과 image status 완료는 후속 조회·대기로 확인합니다. 고정 Python Proxy는 기본 `raise_exc=False`로 raw4xx/5xx 응답을 반환할 수 있지만, Go는 native HTTP 오류를 반환합니다.

## 독립 Go main

[설치 안내](../docs/install.md)를 따라 아래 코드를 `main.go`로 저장합니다. 기본 실행은 기존 staged 이미지에 glance-direct를 한 번 제출합니다. `-service`를 지정하면 Service에서 실행하며, 지정하지 않으면 Connection에서 실행합니다. 두 진입점을 중복 호출하지 않습니다. 60초 context는 인증·명시적 GET·import 제출의 전체 예산입니다.

```sh
go run . -cloud dev -image-id ID
go run . -cloud dev -image-id ID -method web-download -uri https://images.example.org/image.qcow2 -stores fast,archive -all-stores-must-succeed=false
go run . -cloud dev -image-id ID -service -method glance-download -remote-region RegionTwo -remote-image-id REMOTE_ID -remote-interface public
go run . -cloud dev -image-id ID -method copy-image -store archive
```

각 명령은 별도 실행 예입니다. 실제 접근 가능한 이미지·URI·region·store로 값을 바꿔 사용합니다. `-all-stores=false`와 `-all-stores-must-succeed=false`는 명시적 false를 전송하며 생략하면 해당 필드가 없습니다. `-store ''`는 빈 singular ID와 빈 호환 헤더를 보존하고, `-stores 'fast,,fast'`는 순서·중복·빈 plural ID를 보존합니다. 이 값의 backend 유효성은 서버가 판단합니다.

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
    "strings"
    "time"

    "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    imageID := flag.String("image-id", "", "required literal image ID")
    viaService := flag.Bool("service", false, "call image.Service instead of Connection")
    method := flag.String("method", "glance-direct", "server import method")
    uri := flag.String("uri", "", "web-download source URI")
    region := flag.String("remote-region", "", "remote Glance region")
    remoteID := flag.String("remote-image-id", "", "remote Glance image ID")
    remoteInterface := flag.String("remote-interface", "", "remote Glance service interface")
    store := flag.String("store", "", "singular store; explicit empty string is retained")
    stores := flag.String("stores", "", "comma-separated plural stores, preserving order and duplicates")
    allStores := flag.Bool("all-stores", false, "explicit all_stores value")
    allMust := flag.Bool("all-stores-must-succeed", false, "explicit all_stores_must_succeed value")
    flag.Parse()
    present := make(map[string]bool)
    flag.Visit(func(value *flag.Flag) { present[value.Name] = true })
    if *imageID == "" { log.Fatal("-image-id is required") }
    options := []image.ImageRecordImportOption{
        image.WithImageRecordImportOpts(image.ImageRecordImportOpts{
            Headers: map[string]string{"X-Request-Source": "image-record-import-example"},
        }),
        image.WithImageRecordImportMethod(*method),
        image.WithImageRecordImportURI(*uri),
        image.WithImageRecordImportRemoteRegion(*region),
        image.WithImageRecordImportRemoteImageID(*remoteID),
        image.WithImageRecordImportRemoteServiceInterface(*remoteInterface),
    }
    if present["store"] {
        options = append(options, image.WithImageRecordImportStore(image.ImageRecordImportStore{ID: *store}))
    }
    if present["stores"] {
        selected := make([]image.ImageRecordImportStore, 0)
        for _, id := range strings.Split(*stores, ",") { selected = append(selected, image.ImageRecordImportStore{ID: id}) }
        options = append(options, image.WithImageRecordImportStores(selected...))
    }
    if present["all-stores"] { options = append(options, image.WithImageRecordImportAllStores(*allStores)) }
    if present["all-stores-must-succeed"] { options = append(options, image.WithImageRecordImportAllStoresMustSucceed(*allMust)) }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, *viaService, options); err != nil {
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

func run(ctx context.Context, cloud, imageID string, viaService bool, options []image.ImageRecordImportOption) error {
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    record, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: imageID})
    if err != nil { return err }
    input := image.ImageRecordImportRequest{Record: record}
    var result *image.ImageRecordImportResult
    if viaService {
        result, err = service.ImportImageRecord(ctx, input, options...)
    } else {
        result, err = conn.ImportImageRecord(ctx, input, options...)
    }
    return printImport(result, err)
}

func printImport(result *image.ImageRecordImportResult, operationErr error) error {
    value := map[string]any{"partial": result != nil && operationErr != nil}
    if result != nil {
        if ack := result.Acknowledgement; ack != nil {
            // Opaque response bytes may not be JSON. []byte renders as base64.
            value["acknowledgement"] = map[string]any{
                "image_id": ack.ImageID, "body": []byte(ack.Body),
                "header": ack.Header, "status_code": ack.StatusCode,
            }
        }
        if result.Record != nil {
            value["record"] = map[string]any{
                "resource": result.Record.Resource,
                "wire": result.Record.Wire,
                "status_code": result.Record.StatusCode,
                "envelope_bytes": len(result.Record.Envelope),
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

출력은 두 칸 들여쓰기 JSON입니다. ACK Body는 empty·invalid JSON·non-UTF-8일 수 있어 예제에서 `[]byte`로 변환해 base64로 출력합니다. `ImportResult.Body`의 raw bytes를 직접 JSON 이미지로 해석하거나 그대로 `json.Marshal`에 넘기지 않습니다. 문서 예제의 검증 범위는 독립 소비자 컴파일이며 실제 인증·OpenStack import 실행을 의미하지 않습니다.

## 입력·기본값·raw JSON presence

`ImageRecordImportRequest{ID string; Record *ImageRecord}`는 둘 중 하나를 선택합니다. SDK Record의 private canonical `id`와 current/original/dirty Body를 복사하므로 public Resource·Wire·Envelope·Header·properties.id의 변경이 route를 바꾸지 않습니다. full65필드 descriptor와 현재 Connection location을 project하고 plain `ImportMethods=[]`를 reset합니다. 형식은 canonical raw Body의 JSON 값을 사용하며 nonempty string뿐 아니라 true·nonzero number·nonempty list/object도 Source truthiness를 따릅니다. missing/null/false/zero/empty 값은 POST 전에 실패합니다. SDK가 source·입력·옵션을 capture하며 source/context 실패는 callback·전송 경계에서 sticky하게 유지됩니다.

`ImageRecordImportOpts`의 `Method`, `URI`, `RemoteRegion`, `RemoteImageID`, `RemoteServiceInterface`, `AllStores`, `AllStoresMustSucceed`는 `json.RawMessage`입니다. nil은 미지정이고 nonnil raw JSON은 명시 값입니다. 각 `With...` helper는 `any`를 즉시 JSON으로 snapshot하므로 일반 Go 값과 exact `json.Number`·`json.RawMessage`를 사용할 수 있습니다.

| 필드 | 생략·null·falsy 관계 |
|---|---|
| Method | nil이면 `"glance-direct"`; raw null 또는 helper에 nil을 전달하면 `{"name":null}`; 빈 문자열·unknown method도 그대로 전송 |
| URI | nil/null/false/zero/empty string/list/object이면 생략; truthy이면 exact method `"web-download"`만 허용하고 원래 JSON 값 전송 |
| RemoteRegion·RemoteImageID·RemoteServiceInterface | 세 값이 모두 truthy일 때만 `glance_region`·`glance_image_id`·`glance_service_interface` 추가; 하나라도 falsy이면 세 필드 전부 생략; method와 독립 |
| AllStores·AllStoresMustSucceed | nil 또는 raw null이면 생략; false·zero·empty JSON 등 명시 값은 그대로 전송 |

예를 들어 `WithImageRecordImportMethod("web-download")`만 지정해도 Source처럼 URI 없이 제출합니다. remote region·ID만 지정하고 interface를 생략하면 전체 remote tuple을 생략합니다. SDK가 누락 값의 server default를 만들어 넣거나 서버가 광고한 method에 따라 입력을 자동 제한하지 않습니다. truthy URI를 다른 method에 전달하는 충돌과 truthy AllStores의 store 충돌은 로컬 오류입니다.

`WithImageRecordImportOpts(image.ImageRecordImportOpts{})`는 전체 정책을 초기화하므로 default method로 돌아갑니다. Method의 nil과 `json.RawMessage("null")`을 구분해야 합니다. raw JSON은 complete UTF-8이어야 하며 escaped unpaired UTF-16 surrogate를 허용하지 않습니다. 일반 Go string의 invalid UTF-8도 lossy replacement로 다른 값을 선택하지 않도록 거부합니다. arbitrary Python object의 `__bool__`·custom descriptor는 JSON domain 밖의 경계입니다.

## Store 선택과 private identity

`ImageRecordImportStore`는 `ID string`, `Record *serviceinfo.StoreRecord`, `RawID json.RawMessage`, `Attributes map[string]any`를 제공합니다. nonempty ID·nonnil Record·nonnil RawID·nonnil Attributes 중 하나만 지정합니다. zero selector는 literal 빈 문자열 ID이고, `Attributes: map[string]any{}`는 ID가 없는 Source constructor를 나타내므로 null입니다.

- `ID`는 literal string이며 빈 ID도 JSON으로 전달합니다. 이름 lookup이나 URI escape는 store 선택에 필요하지 않습니다.
- `RawID`는 direct ID 값입니다. plural store에서 null·bool·number·list/object를 포함하는 JSON 값을 보존합니다.
- `Attributes`는 Source dict/Munch constructor 분기입니다. canonical `id`만 사용하며 missing id는 null입니다. `name`이 있어도 id로 대체하지 않습니다. `connection`·`_synchronized`·`microversion` constructor 충돌은 HTTP 전에 거부합니다.
- `Record`는 SDK `ServiceInfo.ListStoreRecords/AllStoreRecords`가 반환한 private ID를 사용합니다. public Resource/Wire/receipt의 id 수정이나 getter의 반환 raw bytes 수정으로 import를 retarget할 수 없습니다. public view만 만든 StoreRecord는 거부합니다.

`serviceinfo.StoreImportIdentity(json.RawMessage(...))`는 JSON constructor에서 private import ID를 계산하는 로컬 함수이며 HTTP 없이 같은 id-only·missing null·reserved argument 검사를 수행합니다. `(*serviceinfo.StoreRecord).ImportIdentity()`는 SDK가 보관한 raw ID의 독립 복사본을 반환합니다. 이 accessors가 실제 store 존재·접근·가용성을 확인하지는 않습니다.

singular `Store`를 선택하면 `stores:[id]`와 `X-Image-Meta-Store`를 **함께** 설정합니다. plural `Stores`는 JSON 배열만 설정하며 순서·중복을 보존합니다. nil/empty plural은 배열을 생략합니다. singular와 nonempty plural은 충돌하며 truthy AllStores는 singular presence 또는 nonempty plural과 충돌합니다. 명시 AllStores false는 store와 함께 전달할 수 있습니다.

singular ID는 호환 HTTP header에 들어가므로 valid UTF-8 JSON string이어야 하고 header control 문자를 거부합니다. 빈 string은 허용하지만 null·number·bool·list/object는 singular에 사용할 수 없습니다. 이는 Go의 명시적인 안전한 header 표현 경계이며 Python `requests`가 None/nonstring header를 어떻게 처리하는지를 재현했다는 주장이 아닙니다. 같은 값을 plural에는 JSON으로 사용할 수 있습니다.

source client에 설정된 `X-Image-Meta-Store`는 captured child에서 제거하고 이번 호출의 explicit Store만 선택합니다. original client는 수정하지 않습니다. 설정된 숨은 store가 plural/all-store 요청에 섞이지 않도록 하는 Go ownership 선택이며, Python adapter의 configured header precedence와 같다고 주장하지 않습니다. singular·plural 선택과 raw private identity는 option factory에서 freeze하고 callback config를 복사하므로 원본 map/slice/StoreRecord를 이후 수정해도 준비된 선택이 바뀌지 않습니다.

## concrete 옵션과 확장

| helper | 역할 |
|---|---|
| `WithImageRecordImportOpts` | scalar JSON·store identity·slice·map을 snapshot하고 전체 정책 교체 |
| `WithImageRecordImportMethod`·`WithImageRecordImportURI` | raw 값과 presence 보존 |
| `WithImageRecordImportRemoteRegion`·`WithImageRecordImportRemoteImageID`·`WithImageRecordImportRemoteServiceInterface` | 완전한 truthy remote triple을 위한 각 값 지정 |
| `WithImageRecordImportAllStores`·`WithImageRecordImportAllStoresMustSucceed` | 생략/None과 명시 false를 구분 |
| `WithImageRecordImportStore` | singular selector snapshot |
| `WithImageRecordImportStores` | 전체 plural 선택 교체; 빈 invocation은 plural만 제거 |
| `WithImageRecordImportHeader`·`WithImageRecordImportHeaders` | ordinary header의 case-insensitive last-wins와 map snapshot/병합 |
| `WithImageRecordImportField`·`WithImageRecordImportFields` | root JSON 확장 필드 하나·map 병합 |
| `WithImageRecordImportMethodField`·`WithImageRecordImportMethodFields` | method object의 JSON 확장 필드 하나·map 병합 |

16개 helpers는 순서대로 적용됩니다. Fields와 MethodFields는 `map[string]any`이고 helper map merge는 sorted key 순서를 사용합니다. 동일 이름은 교체하고 empty/nil merge는 기존 map을 비우지 않습니다. 전체 replacement는 WithOpts를 사용합니다. nested JSON·null·exact numbers를 보존하고 method/store/booleans 및 method name/URI/remote controls의 core 이름·alias는 extension으로 덮어쓸 수 없습니다. non-JSON 값·invalid key·nil/error option은 POST 전에 실패합니다. custom option이 config를 보관해도 마지막 private snapshot 이후 준비된 요청을 바꾸지 않습니다. SDK가 concrete JSON encoding·snapshot·namespace 검사를 제공하므로 caller builder interface는 필요하지 않습니다.

ordinary source headers와 request target·Provider·selected Microversion은 callback 전에 capture합니다. per-option auth·framing·representation·version·store header는 SDK-owned라 덮어쓸 수 없습니다. captured source의 Content-Type/Accept policy는 기존 JSON transport처럼 유지하며 설정이 없으면 native application/json defaults를 사용합니다. original Provider의 live authentication·native retry/reauth policy와 middleware를 공유하고 callback 변경으로 source/provider/route를 교체하지 않습니다. JSON body·fixed target과 이번 explicit singular store header 또는 store header의 부재는 native retry/reauth callback과 실제 HTTP dispatch에서도 고정합니다. callback이 선택된 store를 바꾸거나 선택하지 않은 store header를 삽입하면 원래 native 오류와 SDK option 오류를 보존하고 다음 전송 전에 멈춥니다. unrelated ordinary retry headers는 기존 native 정책을 유지합니다.

## 응답·pending 상태·실패

성공 result의 Record는 prepared local snapshot입니다. private current/original/dirty Body와 pending properties, borrowed data 참조, 이전 Wire/Envelope/Header/StatusCode 및 passive fields를 보존하고 `ImportMethods=[]` reset·현재 location만 반영합니다. import ACK의 Body/Header로 record를 translate하거나 clean하지 않으며 `uploading`·`active` 상태도 합성하지 않습니다. borrowed data를 읽거나 닫지 않습니다.

`Acknowledgement`는 `*imageimport.ImportResult{ImageID, Body, Header, StatusCode}`이며 fixed private ImageID와 실제 opaque200..399 응답을 담습니다. empty·array·null·invalid JSON·non-UTF-8 bytes도 opaque 증거입니다. Location·foreign ID·import-method headers를 따라가거나 image/task 모델을 만들지 않습니다. Record의 이전 receipt와 이번 ACK bytes/header는 독립 snapshot입니다.

| 종료 지점 | 반환 |
|---|---|
| local 준비 실패, native400..599 rejection, accepted 응답 없는 transport 실패 | nil result + error |
| accepted response Read/Close·source/context 처리 실패 | actual partial Acknowledgement, Record=nil + error |
| accepted response 처리 성공 | prepared Record + actual Acknowledgement |

nonnil result만으로 성공을 판단하지 않고 err를 확인합니다. `errors.As`로 native `gophercloud.ErrUnexpectedResponseCode`·accepted `resource.ResponseError`, `errors.Is`로 Read/Close·context 원인을 확인합니다. 오류는 source/context 원인과 actual evidence를 보존하며 별도 semantic resend·cleanup을 수행하지 않습니다. fixture의 accepted HTTP 범위는 실제 서버의 모든 status를 성공으로 바꾸는 policy와 구분됩니다.

## 기존 native profile과 Source 범위

기존 [`API.ImageImport.ImportImage/ImportKnownImage`](v2/imageimport/README.md)은 strict202 typed profile을 유지합니다. Ref 입력은 name resolution·fresh GET으로 canonical nonempty string formats를 검사하며 KnownImage는 native ID/container/disk만 복사합니다. 기존 method 입력은 web URI를 요구하고 incomplete/orphan/wrong-method remote 값을 거부하며 singular Store는 header-only입니다. owned helper는 이 기존 API를 재명명하거나 해당 테스트를 전체 Source 동작으로 판정하지 않습니다. [CreateAndImport](create-import.md)와 cloud create/upload/Swift task composition도 별도 작업입니다.

openstacksdk pin `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [Proxy.import_image440–536행](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L440-L536), [Image.import_image368–436행](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L368-L436)을 비교합니다. `_get_resource`의 Resource/constructor·header collector와 full Body descriptor는 [공통 image records 가이드](image-records.md)에 설명합니다.

Glance pin `57f7dd9e76ef24e1e9013eceaa703bd442469a24`의 [import controller317–390행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L317-L390)은 glance-direct·web-download·glance-download에서 `modify_image`, copy-image에서 `copy_image`를 검사합니다. [modify 정책80–93행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L80-L93)의 ordinary3 methods는 핵심 user, [copy 정책330–343행](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L330-L343)의 copy-image는 project ADMIN인 핵심 admin 분기입니다. 같은 public function의 완전한 분기를 구현하며 copy를 별도 operation으로 재집계하지 않습니다. 실제 role·소유권·image status·method capability·quota·URL authorization·store/backend·policy override와 비동기 저장 완료는 서버가 판단합니다. [권한 분류 근거](../docs/glance-policy-priorities.md)를 참고하세요.

immutable Go Record·strict Unicode/private identity·safe singular HTTP string header·configured hidden store 제거·explicit native status errors는 Go 선택입니다. arbitrary mutable Resource/subclass와 전체 dirty/Header lifecycle은 SDK-R1, Adapter cache/invalidation은 SDK-C1, 동적 session/microversion/transport kwargs는 SDK-S1에서 계속 추적합니다. Source 계약·consumer 컴파일·fixture 테스트가 Python runtime 전체나 실제 cloud 실행·backend 동등성을 증명하지는 않습니다.
