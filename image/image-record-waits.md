# Glance 이미지 레코드 상태·삭제 대기: Python과 Go

`conn.Image(ctx)`의 `WaitForImageRecordStatus`와 `WaitForImageRecordDelete`는 [ImageRecord](image-records.md)를 받아 fresh GET으로 상태 또는 삭제 완료를 관찰합니다. SDK가 입력 snapshot, polling 순서·기본값·시간 예산, 마지막 성공한 응답을 관리하며 호출자가 wait builder를 구현하지 않습니다. 삭제 대기는 DELETE를 보내지 않습니다.

| Python `conn.image` | Go `image.Service` | 기본값·결과 |
|---|---|---|
| `wait_for_status(res, status, ...)` | `WaitForImageRecordStatus(ctx, seed, target, options...)` | status, ERROR 실패, 2초 간격, SDK timeout 없음 |
| `wait_for_delete(res, ...)` | `WaitForImageRecordDelete(ctx, seed, options...)` | 2초 간격, SDK timeout120초 |

두 Go API는 `(*ImageRecord, error)`를 반환합니다. `err == nil`일 때만 대기가 완료됐습니다. 첫 성공한 GET 전 오류에는 nil record, 그 이후 오류에는 마지막 성공적으로 projection한 record를 함께 반환합니다. nil이 아닌 record 자체는 성공 판정이 아닙니다.

## Python 전체 연결·서비스 사용

```python
import openstack

conn = openstack.connect(cloud="dev")
seed = conn.image.get_image("image-id")

def show_progress(progress):
    print("progress:", progress)

ready = conn.image.wait_for_status(
    seed, "active", failures=["ERROR"], interval=2, wait=None,
    attribute="status", callback=show_progress,
)
print(ready.to_dict())

# 다른 흐름에서 이미 시작한 삭제를 관찰한다. 이 호출은 삭제 요청을 보내지 않는다.
last = conn.image.wait_for_delete(
    ready, interval=2, wait=120, callback=show_progress,
)
print(last.to_dict())
```

기준은 고정 [Image proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L2299-L2363), [Resource wait](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2585-L2722), [fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1778-L1837)와 [timeout iterator](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53-L99)입니다. Python은 supplied Resource를 같은 객체로 갱신하고 polling fetch에는 `skip_cache=True`를 전달합니다. Go는 입력을 변경하지 않고 private record를 갱신하며 fresh GET을 수행합니다.

## 독립 Go main

[설치 안내](../docs/install.md)로 모듈을 준비하고 아래 내용을 `main.go`에 저장합니다. 이 신규 API를 포함하는 현재 가이드의 revision 또는 후속 revision을 사용합니다. `go run . -cloud dev -image-id ID -target active`는 상태 대기를, `-wait-delete`를 추가하면 이어서 삭제 완료를 관찰합니다. `clouds.yaml`과 실제 이미지 ID가 필요하며 문서 빌드는 cloud 인증·권한 검증을 의미하지 않습니다.

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
    imageID := flag.String("image-id", "", "required image ID")
    target := flag.String("target", "active", "target status")
    waitDelete := flag.Bool("wait-delete", false, "also observe deletion; does not request deletion")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, *target, *waitDelete); err != nil {
        var response *resource.ResponseError
        if errors.As(err, &response) {
            fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
        }
        log.Fatal(err)
    }
}

func printRecord(value *image.ImageRecord) error {
    body, err := json.MarshalIndent(map[string]any{
        "resource": value.Resource, "wire": value.Wire,
        "status_code": value.StatusCode, "import_methods": value.ImportMethods,
    }, "", "  ")
    if err != nil { return err }
    fmt.Println(string(body))
    return nil
}

func waitResult(value *image.ImageRecord, waitErr error) error {
    if waitErr != nil {
        if value != nil {
            fmt.Fprintln(os.Stderr, "wait incomplete; latest projected observation follows")
            if err := printRecord(value); err != nil { return errors.Join(waitErr, err) }
        }
        return waitErr
    }
    return printRecord(value)
}

func run(ctx context.Context, cloud, imageID, target string, waitDelete bool) error {
    if imageID == "" { return fmt.Errorf("-image-id is required") }
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    seed, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: imageID})
    if err != nil { return err }

    callback := func(progress int) { fmt.Fprintf(os.Stderr, "progress: %d\n", progress) }
    ready, err := service.WaitForImageRecordStatus(ctx, seed, target,
        image.WithImageRecordWaitTimeout(2*time.Minute),
        image.WithImageRecordWaitPollInterval(2*time.Second),
        image.WithImageRecordWaitCallback(callback),
        image.WithImageRecordWaitHeader("X-Request-Source", "image-record-waits-example"))
    if err := waitResult(ready, err); err != nil { return err }

    if waitDelete {
        last, err := service.WaitForImageRecordDelete(ctx, ready,
            image.WithImageRecordWaitCallback(callback))
        if err := waitResult(last, err); err != nil { return err }
        fmt.Println("deletion observed; returned record retains the last successful receipt")
    }
    return nil
}
```

`waitResult`는 오류와 함께 반환한 record를 마지막 관측으로 출력하고 오류를 그대로 반환합니다. accepted GET의 malformed JSON 때문에 Envelope가 JSON이 아닐 수 있으므로 전체 record를 JSON marshal하지 않고 declared Resource·실제 Wire·상태만 출력합니다. 삭제404 성공의 반환 record도 이전 성공한 GET 또는 입력 seed의 receipt이며,404 receipt를 성공한 이미지 응답으로 합성하지 않습니다.

이 main은 전체 Connection에서 서비스를 얻습니다. 이미 준비한 Gophercloud client는 `image.New(client)`로 같은 API를 사용하고, `NewWithDependencies(client, image.Dependencies{CloudLocation: getter})`로 현재 location을 공급합니다. 직접 New에 dependency가 없으면 polling GET의 computed location은 null입니다. 예제의5분 parent context는 인증·초기 GET·두 대기를 함께 제한하며 library 기본 timeout과 별개입니다.

## concrete 옵션과 시간 기본값

`ImageRecordWaitOpts`는 `Headers map[string]string`, `Timeout *time.Duration`, `PollInterval *time.Duration`, `Unlimited bool`, `FailureStates []string`, `Attribute string`, `Callback func(int)`를 받습니다. `WithImageRecordWaitOpts`는 전체 설정을 교체하며 `WithImageRecordWaitTimeout/PollInterval/Unlimited/FailureStates/Attribute/Callback/Header/Headers`를 순서대로 적용합니다. WithFailureStates는 variadic string, Callback은 동기적인 `func(int)`입니다. config Callback nil은 보고하지 않는 기본값이며 explicit WithCallback(nil)은 오류입니다. config Attribute 빈 값은 status 기본값이지만 explicit WithAttribute("")는 오류입니다.

| 선택 | 상태 대기 | 삭제 대기 |
|---|---|---|
| nil Timeout | SDK 시간 제한 없음 | SDK120초 |
| explicit Timeout0 | 초기 target 일치 외에는 GET 없이 deadline | GET 없이 deadline |
| 양수 Timeout | 한 operation의 전체 polling 시간 예산 | 같은 정책 |
| Unlimited | SDK 시간 예산 제거, parent context는 유지 | 같은 정책 |
| nil PollInterval | 2초 | 2초 |
| explicit PollInterval0 | 100ms, 양수 timeout이 더 짧으면 그 값 | 같은 정책 |
| FailureStates nil | 정확한 ERROR, 대소문자 무시 | 실패 상태 목록 사용 안 함 |
| empty FailureStates | 실패 상태 판정 없음 | 같은 삭제 조건 유지 |
| Attribute 기본 | status | status 고정 삭제 조건 |

polling이 필요할 때 음수 duration은 Go preflight 오류입니다. 초기 target이 이미 일치하면 Source 순서대로 duration·FailureStates UTF-8 검증은 생략합니다. Timeout이 nonnil인데 Unlimited도 true인 직접 config는 구조적 오류이므로 초기 target 일치에도 거부합니다. Timeout helper는 Unlimited를 false로 만들고 Unlimited helper는 Timeout을 nil로 지워 마지막 선택을 적용합니다. Timeout/PollInterval pointer·Headers map·FailureStates slice·option slice를 snapshot하고 option callback은 preparation에서 각각 한 번 적용합니다. header는 일반 옵션으로 auth·framing·version·representation 값을 덮어쓰지 못합니다. context가 nil/이미 취소됐거나 seed/identity·descriptor JSON/변환·header·attribute 또는 option callback이 잘못되면 HTTP 전에 오류를 반환합니다. Attribute는 canonical Body64·location·plain image_import_methods를 선택할 수 있고 선택한 값이 nonstring이면 상태 판정 오류입니다. 삭제 대기는 status만 지원하므로 다른 Attribute는 ErrUnsupported입니다.

Unlimited는 무한 실행 보장이 아닙니다. parent deadline, HTTP/native retry, option/progress callback과 source guard는 그대로 적용합니다. SDK deadline은 한 번의 preparation과 초기 target 판정 뒤 시작합니다. parent context는 preparation도 포함한 전체 호출을 제한합니다. polling GET이나 callback마다 SDK budget을 새로 만들지 않습니다. GET·callback·pause 동안 소모한 시간도 같은 대기의 남은 예산에 포함합니다.

## 상태 대기 순서

상태 대기는 입력 Resource의 선택한 canonical 속성이 target과 일치하면 private seed clone을 즉시 반환하고 GET·progress callback을 수행하지 않습니다. HTTP가 없는 성공도 seed·옵션·source의 Go preflight는 통과해야 합니다. target/failure 비교는 고정 Unicode16의 locale-independent full lowercase 결과로 수행하고 whitespace를 trim하지 않습니다. `İ`의 i+combining-dot 확장과 Greek Final_Sigma의 원문 문맥을 처리하며 Go toolchain의 Unicode 표에 의존하지 않습니다. 비교 데이터는 기존 descriptor reference와 같은 Unicode16이고 다른 Python runtime의 Unicode database 차이는 별도 경계입니다. target은 valid UTF-8 string이며 빈 문자열도 허용합니다. null 상태는 빈 문자열 target과 일치하지 않습니다.

그 외에는 첫 GET을 즉시 수행합니다. 기본 ERROR만 실패로 판정하며 killed·deleted를 자동 실패 목록에 추가하지 않습니다. 응답에서 target 일치를 먼저 검사하므로 target을 failure 목록에도 넣으면 target 성공이 우선합니다. 초기 failure 상태만으로 실패를 확정하지 않고 fresh GET을 수행합니다. 상태 대기의404는 조회 오류이며 삭제 완료나 목록 fallback으로 바꾸지 않습니다.

선택한 상태의 null은 pending으로 처리합니다. string이 아닌 nonnull 값은 처리 오류이며 unknown wire property·nested path·Python의 arbitrary attribute를 상태 getter로 자동 승격하지 않습니다. target은 Go string API 범위입니다. Get의 실제200..399·빈/invalid JSON 허용·properties packing·id overlay는 [ImageRecord GET 계약](image-records.md)을 따릅니다. 빈 응답은 이전 private seed의 상태를 유지할 수 있으므로 자동 완료 신호가 아닙니다.

비종료 GET을 관찰한 뒤 callback을 실행하고 그 다음 pause합니다. Image의 declared Resource에는 progress descriptor가 없어 callback 값은0입니다. properties.progress·임의 wire 숫자를 percentage로 추측하지 않습니다. 초기 target, target 성공, 실패 상태·HTTP/처리 오류에는 callback을 실행하지 않습니다. callback 후 context와 source를 검사하며 변경·취소가 관찰되면 추가 GET 없이 종료합니다. callback은 동기 실행이므로 차단된 callback을 SDK가 강제로 중단하지 않으며 반환한 뒤 남은 budget을 검사합니다.

## 삭제 대기와 반환값

삭제 대기는 입력 seed가 이미 deleted 상태여도 첫 GET을 수행합니다. fetched status의 string deleted는 대소문자를 무시해 완료로 처리하고 해당 fresh record를 반환합니다. fetched status null 또는 nonstring은 처리 오류이며 상태 대기의 nullable pending과 다릅니다. 별도 이름 검색·List·DELETE를 수행하지 않습니다.

clean 최종 native404도 완료입니다. 첫 성공한 GET 전에404를 만나면 private initial seed clone을 반환합니다. 하나 이상의 성공한 GET 뒤404를 만나면 마지막 overlay record를 반환합니다. Python의 `orig_resource`가 같은 mutable Resource여서 앞선 fetch가 원본에도 반영되는 결과를, Go는 caller를 변경하지 않고 owned record로 표현합니다. 이 반환값에404 응답 body/header/status를 합성하지 않습니다.

native 재인증·retry가 먼저 실행됩니다. Read/Close·source·context·transport·hook 또는 accepted response 처리 원인을 포함한404는 삭제 완료로 바꾸지 않습니다. RetryFunc가 받은 같은 plain native 오류를 그대로 반환하면 추가 원인으로 만들지 않습니다. 다른 rejected status는 기존 native Gophercloud I/O 정책을 따릅니다. 실패 상태 목록과 target status는 삭제 완료 조건을 바꾸지 않습니다.

## 입력·응답 소유권과 부분 결과

입력 ImageRecord의 Resource·Wire·Envelope·Header·ImportMethods와 raw bytes를 복사합니다. caller의 seed를 변경하지 않으며 반환값도 caller·각 polling receipt와 독립입니다. poll seed에는 선언된 Body64만 사용하고 computed location·receipt metadata를 properties로 다시 pack하지 않습니다. GET overlay가 생략한 id/status 등은 이전 private view를 유지하고 explicit null은 교체합니다. properties는 별도로 Source fetch packing을 따르므로 valid object 응답에서 생략됐어도 빈 dictionary로 교체될 수 있고, invalid JSON은 Body overlay를 생략합니다. 실제 Wire는 해당 GET의 원문이며 seed나 이전 wire를 합성하지 않습니다. 후속 처리 오류의 bounded ResponseError는 실패한 응답을 보존하고 partial record는 그 전에 성공한 별도 관측을 보존할 수 있습니다.

첫 successful projection 뒤 실패 상태가 관찰되면 해당 실패 상태 record와 오류를 반환합니다. timeout·취소·source 변경·후속 HTTP/decoder 오류도 마지막 successful projection을 반환하며, failed projection의 일부 필드를 earlier record에 섞지 않습니다. 첫 성공한 physical GET이 없으면 오류와 함께 seed를 partial result로 반환하지 않습니다. initial target 성공과 first clean404 성공은 이 오류 정책의 예외가 아닌 `err == nil`인 완료입니다.

operation마다 고정 client/provider/endpoint·service binding, 현재 location과 options를 한 번 준비하고 모든 poll에서 source guard를 적용합니다. 초기 target/첫404 반환은 seed location을 보존하고 실제 성공한 poll의 computed location은 새 operation에서 캡처한 현재 facts를 사용합니다. project_id/availability_zone가 없는 Source Image fetch가 기존 computed location을 유지하는 동작과 구분하는 Go 정책입니다. ID는 입력 Resource의 nonblank string id를 고정합니다. 응답 id가 다르거나 null이어도 후속 URL을 바꾸지 않습니다. 허용된 slash·backslash·공백 등 identity는 원문을 단일 escaped segment로 요청하며 wire schema·self·location URL은 따라가지 않습니다. Source mutable fetch가 응답 ID로 다음 route를 바꿀 수 있는 동작과 구분하는 Go 경계입니다.

## 기존 API·권한·남은 비교 범위

기존 typed `image.WaitForState`, `WaitForDelete`, `service.Images.Wait/WaitDeleted`와 native `API.Images.WaitForStatus`의 signature·기본값·결과형은 유지합니다. [기존 서비스 대기](../docs/service-waits.md)의 error-only 삭제 반환이나 strict typed 상태 검증을 이 record API의 규칙으로 해석하지 않습니다.

새 polling은 `GET /images/{id}`만 사용하므로 별도 admin 경로나 쓰기 정책을 추가하지 않습니다. [확인한 get_image reader 정책](../docs/glance-policy-priorities.md#이미지-조회목록검색의-기본-정책)을 따라 핵심 user에 배치하며 실제 policy override·가시성·소유권은 서버가 판단합니다.404 삭제 관찰은 호출자의 접근 범위에서의 결과이며 SDK가 전체 cloud의 부재를 검증한 것은 아닙니다.

고정 Source의 대기 순서·공개 기본값·fresh fetch·반환 Resource를 확인한 범위입니다. Python mutable same-instance/dirty state·subclass·Adapter/session/microversion·configured cache·arbitrary attribute/progress·parser domain 전체는 공통 비교 범위로 남습니다. Go는 명시한 JSON-domain ImageRecord, 고정 input identity와 duration/context 계약을 제공합니다. local fixture·문서 빌드는 실제 cloud 또는 Python runtime 실행을 대신하지 않습니다.

[owned record wait 테스트](image_record_wait_test.go)는 초기 target의 독립 복사, 즉시 fresh GET·실패/nullable 상태·target 우선순위,404의 seed/last 반환, 한 budget과 zero interval/timeout, accepted/rejected physical guard, callback 뒤 변경·부분 결과·concrete preflight를 다룹니다. 기존 ImageRecord transport/body fixture와 공개 Gophercloud testhelper, 공통 Resource wait engine을 재사용합니다. [Connection 대기 테스트](../connection_image_record_waits_test.go)는 공유 service client와 location·polling 연결을 확인합니다. 실제 실행 결과는 [지원 판정대장](../docs/sdk-support-ledger.md)에서 확인합니다.
