# Swift 객체 대기

`service.Objects.WaitForDelete`는 삭제를 요청한 객체가 HEAD404를 반환할 때까지 관찰합니다. `WaitForStatus`는 선택한 문자열 속성이나 응답 header가 목표 값에 도달할 때까지 관찰합니다. 두 API는 고정된 container/object에 HEAD만 보내며 결과의 `Last`에 마지막 poll의 실제 응답과 native attempt 증거를 보존합니다.

Swift Object에는 기본 `status`와 `progress` 속성이 없습니다. 따라서 selector 없는 `WaitForStatus`는 HTTP 전에 `resource.ErrUnsupported`를 반환합니다. 아래 `X-Object-Meta-State`는 application이 관리하는 metadata 예제입니다. Swift의 표준 lifecycle 상태를 뜻하지 않습니다.

| 정책 | pinned openstacksdk | Go |
|---|---|---|
| 조회 | Resource.fetch의 GET, skip_cache=True | literal container/object의 fresh HEAD |
| 상태 대기 | interval2초, wait=None, 기본 status 속성 | interval2초, SDK timeout 없음, explicit selector 필요 |
| 삭제 대기 | proxy wait120초; generic nil/deleted/NotFound도 성공 | SDK timeout120초; clean physical HEAD404만 성공 |
| 반환 | mutable Resource | `ObjectWaitResult`와 마지막 poll의 owned raw 증거 |
| callback | nonterminal progress 또는0, 반환값 없음 | nonterminal에서0, synchronous `func(int) error` |

이미 인증한 Connection과 존재하는 `backups` container를 사용합니다. metadata 상태 관찰, 알려진 문자열 속성 관찰, 이미 삭제 요청을 보낸 객체 관찰을 보여줍니다. `WaitForDelete` 자체는 DELETE를 보내지 않습니다.

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "time"

    sdk "github.com/JSYoo5B/gophercloudsdk"
    "github.com/JSYoo5B/gophercloudsdk/objectstorage/v1/objects"
    "github.com/JSYoo5B/gophercloudsdk/resource"
)

func showWait(result *objects.ObjectWaitResult, err error) {
    if result != nil {
        status := "<absent>"
        if result.Status != nil {
            status = *result.Status
        }
        fmt.Printf("polls=%d complete=%v deleted=%v status=%q\n",
            result.Polls, result.Complete, result.Deleted, status)
        if result.Last != nil {
            for _, attempt := range result.Last.Attempts {
                fmt.Printf("physical=%d error=%v\n", attempt.PhysicalAttempt, attempt.Error)
                if attempt.Response != nil {
                    fmt.Printf("HTTP%d headers=%v body=%q\n",
                        attempt.Response.StatusCode, attempt.Response.Header, attempt.Response.Body)
                }
            }
            if result.Last.Acknowledgement != nil {
                fmt.Printf("last acknowledgement HTTP%d\n", result.Last.Acknowledgement.StatusCode)
            }
        }
    }
    if err != nil {
        fmt.Println(err)
        if errors.Is(err, resource.ErrUnsupported) {
            fmt.Println("unsupported status source")
        }
        var proof *resource.ResponseError
        if errors.As(err, &proof) {
            fmt.Printf("error proof HTTP%d headers=%v body=%q\n",
                proof.StatusCode, proof.Header, proof.Body)
        }
    }
}

func waitExamples(ctx context.Context, conn *sdk.Connection) {
    service, err := conn.ObjectStorageV1(ctx)
    if err != nil {
        showWait(nil, err)
        return
    }
    interval := 2 * time.Second
    policy := []objects.ObjectWaitOption{
        objects.WithObjectWaitOpts(objects.ObjectWaitOpts{
            Headers: map[string]string{"X-Client": "object-wait"},
            Interval: &interval,
        }),
        objects.WithObjectWaitHeader("X-Trace", "initial"),
        objects.WithObjectWaitHeaders(map[string]string{"x-trace": "last"}),
        objects.WithObjectWaitPollInterval(2 * time.Second),
        objects.WithObjectWaitTimeout(time.Minute),
        objects.WithObjectWaitProgressCallback(func(progress int) error {
            fmt.Printf("nonterminal observation, progress=%d\n", progress)
            return nil
        }),
    }
    statusOptions := append([]objects.ObjectWaitOption{}, policy...)
    statusOptions = append(statusOptions,
        objects.WithObjectWaitStatusHeader("X-Object-Meta-State"),
        objects.WithObjectWaitFailureStates("ERROR", "FAILED"),
    )
    result, err := service.Objects.WaitForStatus(ctx, "backups", "archive.tar", "READY", statusOptions...)
    showWait(result, err)

    // A known string attribute can also be observed, such as the MIME type.
    attributeOptions := append([]objects.ObjectWaitOption{}, policy...)
    attributeOptions = append(attributeOptions,
        objects.WithObjectWaitStatusAttribute("content_type"),
        objects.WithObjectWaitFailureStates(),
        objects.WithoutObjectWaitProgressCallback(),
    )
    result, err = service.Objects.WaitForStatus(ctx, "backups", "archive.tar", "application/octet-stream", attributeOptions...)
    showWait(result, err)

    // UnlimitedWait removes the SDK deadline; this caller still has 30 seconds.
    deleteCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    deleteOptions := append([]objects.ObjectWaitOption{}, policy...)
    deleteOptions = append(deleteOptions,
        objects.WithObjectWaitUnlimitedWait(),
        objects.WithoutObjectWaitProgressCallback(),
    )
    result, err = service.Objects.WaitForDelete(deleteCtx, "backups", "already-deleted.bin", deleteOptions...)
    showWait(result, err)
}

func main() {}
```

첫 poll은 즉시 실행하고 clean nonterminal 결과 뒤에 interval만큼 기다립니다. `Polls`는 logical HEAD exchange 수입니다. native retry/backoff/reauth 또는 같은 target redirect가 한 poll 안에서 여러 physical requests를 만들면 `Last.Attempts`에 각각 남습니다. 이전 poll 전체 history는 누적하지 않으며 마지막으로 physical request가 시작한 poll의 phase만 보존합니다. 다음 logical exchange가 HTTP 전에 중단되면 Polls만 늘 수 있고 이전 Last와 clean Status는 유지합니다. 새 physical poll이 시작하면 그 phase를 Last로 교체하고 Status를 새로 판정합니다. `Last.Acknowledgement`, 각 attempt와 error proof는 저장 공간을 독립 소유합니다.

삭제 대기는 clean HEAD404에서 `Complete=true`, `Deleted=true`를 설정합니다. HEAD200/204는 계속 기다립니다. status/failure selector는 삭제 완료 판정에 사용하지 않고 `Status`는 항상 nil입니다. 읽기·Close·source/context 오류가 있는404, transport 오류 안의404, 다른 HTTP 실패는 삭제 완료가 되지 않습니다. 관찰한 미존재가 모든 replica의 삭제 완료를 보장하지는 않습니다.

상태 대기는 HEAD200/204 뒤에 선택한 값만 검사합니다. target match가 failure state보다 먼저이며 대소문자를 무시해 비교합니다. nil `FailureStates`는 ERROR 기본값, nonnil empty slice 또는 인자 없는 `WithObjectWaitFailureStates()`는 failure 검사를 끕니다. selected response header가 없으면 실제 응답을 남긴 `ErrUnsupported`이며, present empty 값은 nonnil empty string으로 구별합니다. status 대기404는 실패입니다. `Complete=true`는 관찰한 목표 값의 일치이며 `Deleted`는 false입니다.

`WithObjectWaitStatusHeader`는 explicit response header를 선택합니다. `StatusAttribute`의 알려진 문자열 매핑은 다음과 같습니다. `name`과 `container`는 captured literal 값이며 이 경우에도 fresh HEAD를 수행합니다. 숫자·bool·임의 Resource 속성은 지원하지 않습니다.

| StatusAttribute | 값 |
|---|---|
| name, container | captured object, container |
| content_type, etag | Content-Type, ETag |
| content_encoding, content_disposition | Content-Encoding, Content-Disposition |
| manifest, object_manifest | X-Object-Manifest |
| timestamp | X-Timestamp |
| last_modified_at, updated_at | Last-Modified |
| delete_at, accept_ranges | X-Delete-At, Accept-Ranges |
| access_control_allow_origin | Access-Control-Allow-Origin |
| expires_at, signature | Expires, Signature |

선택한 header만 case alias·복수 값·UTF8·control 여부를 엄격히 검사하며 관련 없는 header는 status projection에 영향을 주지 않습니다. 두 selector helper는 서로를 비워 마지막 선택을 사용합니다. full opts에 둘 다 있으면 거부합니다. full options는 설정 전체를 교체하며 maps·duration pointers·failure slices와 callback 결과를 snapshot합니다. header helpers는 canonical 이름으로 순서대로 overlay합니다.

explicit interval/timeout helper는 양수만 허용합니다. nil Interval은2초, nil Timeout은 작업별 기본값입니다. owned Timeout pointer0 또는 `WithObjectWaitUnlimitedWait`는 SDK deadline을 없애고 caller context는 계속 적용합니다. 음수는 preflight 오류입니다. callback은 clean nonterminal poll마다 synchronous하게0을 받으며 terminal success/failure에는 호출하지 않습니다. callback의 error와 cancellation/source 원인은 마지막 실제 증거와 함께 보존합니다. nil callback이나 `WithoutObjectWaitProgressCallback`으로 해제합니다. callback/current phase 처리 오류에서는 그 poll의 Status를 commit하지 않습니다. clean nonterminal 값 뒤의 pause timeout은 마지막 clean Status를 보존하며 Complete는 false입니다.

원래 API/client/provider·Endpoint·ResourceBase·Type·Microversion과 literal target을 고정하고 live auth는 유지합니다. request auth·framing·metadata/routing·ETag·X-Newest는 reserved이며 이 waiter는 Range와 If-* 조건부 headers도 source/native/body 경계에서 거부합니다. metadata response status header 선택은 request metadata header 주입과 별개의 옵션입니다. 기존 ObjectRead의 typed Newest와 native Get의 별도 options는 그대로 사용할 수 있습니다.

[pinned proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L1247-L1306)는 generic [Resource waits](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2585-L2715)에 위임합니다. status utility는 cached `getattr(resource, attribute)`의 target match를 먼저 반환할 수 있습니다. fresh [Resource.fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1778-L1837)는 GET을 보내며 prepared request.headers를 넘기지 않습니다. Object의 `has_body=False`는 body projection 정책이고 HEAD 호출로 바꾸지 않습니다. [Object descriptors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/obj.py#L27-L213)에는 status/progress가 없으므로 기본 status wait는 AttributeError이며 nonterminal progress는0입니다.

Python [iterate_timeout](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53-L101)은 wall-clock timeout과 numeric interval coercion을 사용하며 None interval은2초, zero interval은 작은 양수로 바꿉니다. Go의 duration/context 정책과 오류를 반환하는 callback은 의도적인 차이입니다. mutable Resource·descriptor·session/cache·cached fast path·GET contents의 전체 parity를 완료한 판정은 아닙니다. 기존 공통 [Collection.WaitDeleted](../../../resource/README.md)는5분 기본값과 generic 완료 정책을 유지하며 Swift Objects에 새 Collection을 만들지 않습니다.
