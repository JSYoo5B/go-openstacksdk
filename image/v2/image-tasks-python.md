# 이미지별 task 조회: Python과 Go

`conn.ImageV2(ctx).Images.TaskRecords`와 `AllTaskRecords`는 고정 openstacksdk의 `conn.image.image_tasks(image)`를 Resource 형태로 제공합니다. literal 이미지 ID와 서비스 경로를 고정하고, 기본값·원본 JSON·로컬 필터·페이지 순회·microversion·실제 응답 증거를 SDK가 처리합니다. 같은 API를 `conn.Image(ctx).API.Images`와 `images.New(client)`에서도 사용할 수 있습니다. 호출자가 builder나 확장 interface를 구현할 필요는 없습니다.

비교 기준은 openstacksdk [`ef55d7d`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1181-L1194)와 Gophercloud `v2.15.0`입니다. 이 문서는 named `image_tasks` 조회의 범위를 설명하며 task 생성·상태 wait·정리·전체 mutable Resource API 완료를 뜻하지 않습니다.

## 권한과 선택자

고정 Glance 서버 [`57f7dd9`](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L599-L614)의 `GET /v2/images/{image_id}/tasks`는 parent 이미지를 읽고 `get_image` 정책을 검사합니다. [Glance2025.2 기본 정책](https://docs.openstack.org/glance/2025.2/configuration/glance_policy.html)은 reader와 owner/member/public/community/shared image visibility 경로를 사용합니다. 숨겨진 이미지나 권한이 없는 parent는404로 보일 수 있으며, SDK는 현재 token으로 받은 오류를 그대로 반환합니다. 별도 `/v2/tasks`의 `tasks_api_access` admin 정책과는 다른 endpoint입니다.2025.2 문서와 위 고정 서버 commit은 각각의 증거이며 고정 commit을 release tag로 취급하지 않습니다. 배포 정책은 서버가 결정합니다.

Python proxy의 설명에는 이름이 언급되지만 실제 `_get_id`는 literal string이나 Resource.id를 사용하며 이름 검색을 수행하지 않습니다. Go도 `imageID string`을 한 번 escape해 `images/{imageID}/tasks`를 요청하고 parent GET·Find·이름 검색을 추가하지 않습니다. blank/control/invalid UTF-8 selector와 `.`/`..`는 HTTP 전에 오류입니다. parent404를 무시하는 기본값이나 `ignore_missing` 옵션은 없습니다.

## Python

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument('--cloud', required=True)
parser.add_argument('--image', required=True)
args = parser.parse_args()
conn = openstack.connect(cloud=args.cloud)
for task in conn.image.image_tasks(args.image):
    print(json.dumps(task.to_dict(), ensure_ascii=False))
```

위 proxy는 `image`만 받습니다. query/header/cap/microversion 옵션을 Python proxy kwargs처럼 넘기는 예제는 사용하지 않습니다. 다음 Go 옵션은 필요한 inherited Resource.list 동작을 구체적인 SDK 설정으로 제공하는 추가 편의 기능입니다.

## Go: 독립 main

아래 전체 코드를 `main.go`에 저장하고 이 module을 의존하는 Go module에서 실행할 수 있습니다.

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "os"
    "time"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/images"
)

func main() {
    cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud name")
    imageID := flag.String("image", "", "literal image ID")
    status := flag.String("status", "", "optional local status filter")
    limit := flag.Int("limit", -1, "server page hint; -1 omits it")
    cap := flag.Int("max-items", 0, "raw rows consumed before local filtering; 0 is unlimited")
    paginated := flag.Bool("paginated", true, "follow retained continuation")
    flag.Parse()
    if err := run(*cloud, *imageID, *status, *limit, *cap, *paginated); err != nil {
        log.Fatal(err)
    }
}

func run(cloud, imageID, status string, limit, maxItems int, paginated bool) error {
    if imageID == "" { return fmt.Errorf("-image is required") }
    if limit < -1 { return fmt.Errorf("-limit must be -1 or nonnegative") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.ImageV2(ctx)
    if err != nil { return err }
    options := []images.TaskRecordsOption{
        images.WithTaskRecordsHeaders(map[string]string{"X-Request-Source": "image-tasks-example"}),
        images.WithTaskRecordsMaxItems(maxItems),
        images.WithTaskRecordsPaginated(paginated),
    }
    if limit >= 0 { options = append(options, images.WithTaskRecordsLimit(limit)) }
    if status != "" { options = append(options, images.WithTaskRecordsQuery("status", status)) }
    out := json.NewEncoder(os.Stdout)
    for record, readErr := range service.Images.TaskRecords(ctx, imageID, options...) {
        if readErr != nil { return readErr }
        if err := out.Encode(map[string]any{
            "resource": record.Resource.Body,
            "wire": record.Wire.Body,
            "envelope": record.Envelope,
            "headers": record.Header,
            "status_code": record.StatusCode,
            "microversion": record.Microversion,
            "discovery": record.Discovery,
        }); err != nil { return err }
    }
    return nil
}
```

예: `go run . -cloud dev -image IMAGE_ID`. 이 main의1분 제한은 caller 선택이며 SDK가 부여한 기본 timeout이 아닙니다. `AllTaskRecords(ctx, imageID, options...)`는 성공한 빈 목록을 nonnil slice로 반환하며 뒤 page의 오류가 발생하면 partial slice를 버립니다. iterator에서 이미 받은 행은 caller에게 남습니다.

## Resource·Wire와 기본 필터

[`ImageTasks`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image_tasks.py#L22-L64)의14개 Body 속성은 `type`, `status`, `owner`, `expires_at`, `created_at`, `updated_at`, `deleted_at`, `deleted`, `image_id`, `request_id`, `user_id`, `input`, `result`, `message`이며 inherited `id`·`name`을 포함해16개입니다. 모두 untyped입니다. 생략은 null이고 explicit null·false·0·큰 정수·string·array·object는 그대로 유지합니다. `input`/`result`를 object로 제한하거나 날짜를 Go time으로 변환하거나 `deleted`를 bool로 강제하지 않습니다.

`ImageTaskRecord.Resource.Body`에는 위 declared 값과 computed `location`을 제공합니다. `Wire.Body`에는 unknown·remote 비표준 spelling·`self`·서버 location 등 실제 행의 모든 값이 남습니다. Source constructor는 `self`를 제거하고 unknown Body를 버립니다. `cls`·`connection`·`microversion`·`_synchronized` 행은 Source `existing` constructor 충돌처럼 실제 whole-page 응답을 포함한 오류입니다. 이 class에는 Header descriptor가 없으므로 실제 `If-Match`가 있더라도 Resource에 `if_match`를 추가하지 않습니다.

`Envelope`는 실제 전체 page JSON이고 `Header`·`StatusCode`는 실제 task-list 응답입니다. Resource·Wire·Envelope·Header·다른 행과 discovery 관찰은 각각 독립 소유입니다. `ImageID`는 요청한 고정 parent이며 서버가 반환한 `Resource.Body["image_id"]`를 덮어쓰지 않습니다.

중요한 Source 기본 동작: `image_id`는 URI descriptor가 아닌 **Body descriptor**입니다. proxy가 넘긴 `image_id`는 URL뿐 아니라 inherited list의 기본 **로컬 Body 필터**가 됩니다. 따라서 반환된 `image_id`가 생략/null이거나 다른 parent인 행은 제외됩니다. 이 값은 서버로 query로 보내지 않으며 빠진 값을 parent로 합성하지 않습니다. 새 API는 이 동작을 유지합니다.

location은 현재 recorded Keystone v2/v3 auth scope를 HTTP 없이 읽고, cloud/region 이름은 만들어내지 않습니다. `owner`는 project descriptor가 아니므로 location project를 owner로 바꾸지 않습니다. `WithTaskRecordsLocation`은 명시된 current CloudLocation을 복사해 사용하며 명시 zone도 유지합니다. Python Connection/config/cache 객체를 흉내내는 기능은 아닙니다.

## 옵션·페이지·microversion

`TaskRecordsOpts`와 `WithTaskRecordsOpts/Query/Headers/Microversion/Location/Limit/Marker/MaxItems/Paginated`를 제공합니다. nil Limit은 생략, explicit0은 `limit=0`, MaxItems0은 무제한, nil Paginated는 true이며 false는 첫 page까지만 읽습니다. 음수 cap/limit은 오류입니다. 양수 cap은 필터 전 원본 행 수를 제한하고 limit이 없거나0이면 첫 GET에 cap을 limit hint로 보냅니다. 필터로 제외된 행도 cap을 소비합니다. cap이나 consumer break 뒤의 행·잘못된 next는 평가하지 않습니다.

원시 Query에서 inherited `limit`·`marker`만 서버로 보내고 canonical Body 속성은 로컬 조건으로 비교합니다. unknown은 무시합니다. 고정 parent/Source positional·constructor/control key는 사전에 거부하며 owned 옵션을 사용합니다. Go limit은 한 개의 nonnegative integer, marker는 한 개의 nonblank UTF-8 값으로 제한합니다. 순차 Query 옵션은 입력 순서를 유지하고 bulk map·새 callback key는 Go map의 순서가 없으므로 정렬합니다. option slice·pointer·map·raw JSON·retained callback 상태는 operation별로 복사하며 적용은 각 지연 순회에서 한 번입니다.

로컬 object 필터는 Source recursive subset 규칙을 사용합니다. 실제 값이 falsey면 실패하고 empty object 조건은 truthy scalar에도 맞습니다. nonempty object 조건을 truthy nonobject에 적용하면 Source `mapping.get` 경계의 오류를 실제 page 증거와 반환합니다. bool-number equality는 `True == 1`에 대응하고 decimal truthiness는 Python binary64 underflow를 반영합니다. 숫자 equality는 Go 공통 exact decimal JSON profile이며 임의 Python float rounding·runtime 객체까지 같다고 주장하지 않습니다.

[`Resource.list`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2155-L2359)의 `tasks` envelope를 사용합니다. non-list object는 singleton으로 허용하고 null/scalar/wrongshape·malformed/empty JSON·invalid UTF-8은 실제 응답을 포함한 오류입니다. plural links·top-level next·HTTP Link·short-page marker continuation을 처리하고 빈 page에서 끝냅니다. falsey decimal `next: 1e-400`은 종료하고 실제 Wire에는 원래 숫자가 남습니다. Go dictionary-link compatibility와 정확한 상대 `/v2/images/{imageID}/tasks` alias도 허용합니다. 외부 origin·다른 collection·filter 변경·cycle은 실제 page 증거와 거부합니다. Source의 더 넓은 version-prefix 제거/retargeting을 허용하지 않습니다. 현재 stock controller는 queryfree 전체 DB 행을 반환하지만 이 사실만으로 inherited Source pager를 무시하지 않습니다. 행 정렬·expiry·visibility·limit 실제 지원은 서버 책임입니다.

captured client에 selected Microversion이 있으면 그대로 사용합니다. 없으면 shared finite discovery로 Glance major2의 eligible advertisement를 확인하고 Source class ceiling2.17과 advertised microversion bounds를 선택합니다. Glance의 API ID `v2.17` 자체를 microversion maximum으로 취급하지 않으며 ordinary Glance 광고에 max/min이 없으면 header 없이 요청합니다. `WithTaskRecordsMicroversion("")`은 명시 headerless 선택이고 nonempty 값은 그대로 선택합니다. 원래 client 설정을 바꾸지 않으며 `Discovery`에는 task receipt와 별개인 실제 discovery 관찰을 보존합니다. 최소 API2.12 capability gate·글로벌 discovery cache·Schema 조회·자동 task wait는 추가하지 않습니다.

## 오류·검증 범위

Source list의 기본 `Accept: application/json`은 captured client Accept보다 우선하고, 명시 안전한 Headers의 Accept는 이를 바꿀 수 있습니다. auth·framing·외부 서비스 version·일반 Headers로 지정한 version은 거부합니다. native retry/reauth는 원래 route/query/body/선택 version/header 정책 안에서 유지하고 token renewal은 허용합니다. callback이 protocol header·body·OkCodes·OmitHeaders를 바꿔 원래 정책을 벗어나면 실제 rejection과 callback 원인을 남깁니다.

고정 Source `raise_from_response`는 status<400에서 반환합니다. Go는 최종 representation200..399를 받아 처리하고 informational100..199는 실제 native status/body/header를 포함해 거부하는 명시적 차이가 있습니다. 거부 응답·accepted decode·physical Read/Close·custom cancellation cause·순간적인 Source 변경 후 복원도 실제 응답과 모든 원인을 보존하고 physical body는 한 번 닫습니다. 오류 뒤 parent 이름 검색이나 다른 scope 조회로 재시도하지 않습니다. invalid UTF-8 거부는 Requests의 replacement decoding보다 엄격합니다.

기존 [`image.Service.ImageTasks`/`AllImageTasks`](../image-tasks.md)는 finite typed `ImageTaskInfo` API입니다. Ref Name 확장을 통한 명시 parent 검색, HTTP200 전용, typed date/deleted와 nullable object Input/Result, passive paging 데이터, local cap을 유지합니다. 새 leaf는 literal parent·Source untyped/default/localfilter·retained pager·separate Resource/Wire를 위한 sibling이며 기존 typed 계약을 바꾸지 않습니다.

회귀 테스트는 기존 Glance native fixture와 public Gophercloud testhelper를 사용하는 `internal/testcloud`를 재사용합니다. fixture/race 검사·독립 main build는 실제 배포의 endpoint·policy·광고 내용을 확인한 live cloud 검사와 구분합니다. named contract의 검증은 전체 mutable Resource·dirty/cache·Python class/subclass/session parity 완료를 의미하지 않습니다.
