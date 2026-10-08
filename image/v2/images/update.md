# Glance native 이미지 PATCH: Python과 Go

`conn.Image(ctx).API.Images.Update`는 concrete `images.UpdateOpts` 배열을 PATCH로 보내고 Gophercloud의 `Image`를 추출합니다. SDK가 upstream request builder와 Extract 호출을 맡으며, 일반 호출자는 builder interface를 구현할 필요가 없습니다. 이 메서드는 Python `update_image`의 mutable Resource·dirty 비교·자동 no-op를 제공하는 별도 owned 계층과 구분합니다.

| Python 서비스 | Go 서비스 | 요청·반환 |
|---|---|---|
| `conn.image.update_image(image_or_id, **attrs)` | `service.API.Images.Update(ctx, imageID, patches, options...)` | `PATCH /images/{id}`, `(*images.Image, error)` |
| Resource의 변경점으로 자동 JSON Patch 생성 | caller가 concrete Patch 배열을 구성 | Go는 입력 순서대로 매번 전송 |
| mutable Image의 같은 instance를 반환 | native 응답 Image를 Extract | 기존 조회 instance·원문 receipt를 갱신하지 않음 |

준비한 client는 `images.New(client).Update`, 전체 v2 facade는 `conn.ImageV2(ctx).Images.Update`로 같은 API를 사용합니다. 상위 [명시적 UpdateImage·SetImageProperties](../../update.md)와 [owned ImageRecord 조회](../../image-records.md)는 각각 별도 API입니다.

## 독립 Go main

[설치 안내](../../../docs/install.md)를 따른 뒤 아래 내용을 `main.go`에 저장합니다. `go run . -cloud dev -image-id ID -name release-image -tag reviewed`는 실제 이미지를 수정합니다. name·visibility·보호 설정·최소 disk/RAM과 전체 tags를 변경하고 custom property를 추가·교체·삭제합니다. 수정 가능한 이미지와 권한을 선택해야 하며 서버가 최종 허용 여부를 판단합니다. 문서 검증에서는 이 main을 빌드하며 실제 cloud mutation을 실행하지 않습니다.

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
    "github.com/JSYoo5B/go-openstacksdk/image/v2/images"
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    cloud := flag.String("cloud", "", "required clouds.yaml cloud name")
    imageID := flag.String("image-id", "", "required native image ID")
    name := flag.String("name", "release-image", "new image name")
    tag := flag.String("tag", "reviewed", "one tag replacing the whole tag set")
    flag.Parse()

    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, *name, *tag); err != nil {
        var native gophercloud.ErrUnexpectedResponseCode
        if errors.As(err, &native) {
            fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", native.Actual, len(native.Body))
        }
        log.Fatal(err)
    }
}

func run(ctx context.Context, cloud, imageID, name, tag string) error {
    if cloud == "" || imageID == "" {
        return fmt.Errorf("-cloud and -image-id are required")
    }
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }

    patches := images.UpdateOpts{
        images.ReplaceImageName{NewName: name},
        images.UpdateVisibility{Visibility: images.ImageVisibilityPrivate},
        images.ReplaceImageHidden{NewHidden: false},
        images.ReplaceImageProtected{NewProtected: false},
        images.ReplaceImageMinDisk{NewMinDisk: 0},
        images.ReplaceImageMinRam{NewMinRam: 0},
        images.ReplaceImageTags{NewTags: []string{tag}},
        images.UpdateImageProperty{Op: images.AddOp, Name: "sdk_demo_build", Value: "draft"},
        images.UpdateImageProperty{Op: images.ReplaceOp, Name: "sdk_demo_build", Value: "ready"},
        images.UpdateImageProperty{Op: images.AddOp, Name: "sdk_demo_temporary", Value: "temporary"},
        images.UpdateImageProperty{Op: images.RemoveOp, Name: "sdk_demo_temporary"},
    }
    // WithUpdateOptions replaces the base nil array with these concrete patches.
    value, operationErr := service.API.Images.Update(ctx, imageID, nil, images.WithUpdateOptions(patches))
    output := map[string]any{
        "image": value,
        "partial": value != nil && operationErr != nil,
    }
    if operationErr != nil { output["error"] = operationErr.Error() }
    body, marshalErr := json.MarshalIndent(output, "", "  ")
    if marshalErr != nil { return errors.Join(operationErr, marshalErr) }
    fmt.Println(string(body))
    return operationErr
}
```

출력은 두 칸 들여쓰기 JSON이며 nonnil Image와 오류가 함께 반환되면 partial로 표시하고 오류도 그대로 반환합니다. 일부 필드가 보존됐다는 사실은 수정 완료나 완전한 모델 추출을 뜻하지 않습니다. 예제의 1분 parent context는 인증·PATCH를 함께 제한하는 caller 선택이며 Update의 library 기본 timeout이 아닙니다.

## 제공하는 아홉 concrete Patch 타입

모든 타입은 [고정 upstream Patch](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/images/requests.go#L260-L418)의 alias입니다. typed 필드가 있다는 사실과 서버에서 해당 필드를 수정할 수 있다는 사실은 별개입니다.

| concrete 타입 | 입력 필드 | wire operation·path |
|---|---|---|
| `UpdateVisibility` | `Visibility ImageVisibility` | replace `/visibility` |
| `ReplaceImageHidden` | `NewHidden bool` | replace `/os_hidden` |
| `ReplaceImageName` | `NewName string` | replace `/name` |
| `ReplaceImageChecksum` | `Checksum string` | replace `/checksum` |
| `ReplaceImageTags` | `NewTags []string` | replace `/tags` 전체 |
| `ReplaceImageMinDisk` | `NewMinDisk int` | replace `/min_disk`, GB |
| `ReplaceImageMinRam` | `NewMinRam int` | replace `/min_ram`, MB |
| `ReplaceImageProtected` | `NewProtected bool` | replace `/protected` |
| `UpdateImageProperty` | `Op UpdateOp, Name string, Value string` | 아래 property 규칙 |

`UpdateImageProperty`는 `Name` 앞에 `/`를 붙여 path를 만들고 value는 string입니다. `AddOp`와 `ReplaceOp`는 `value`를 포함하고 `RemoveOp`는 이를 생략합니다. 알려지지 않은 Op나 빈 Name을 이 native 계층에서 자동 교정하지 않습니다. RemoveOp는 Value를 지정해도 이를 보내지 않습니다.

| property 입력 | 실제 Patch |
|---|---|
| `Op: AddOp, Name: "build", Value: ""` | `{"op":"add","path":"/build","value":""}` |
| `Op: ReplaceOp, Name: "build", Value: "ready"` | `{"op":"replace","path":"/build","value":"ready"}` |
| `Op: RemoveOp, Name: "build"` | `{"op":"remove","path":"/build"}` |

Name의 `~`·`/`는 자동 escape하지 않습니다. literal root property `build/id`를 선택하려면 JSON Pointer token으로 escape한 `Name: "build~1id"`를 전달합니다. 입력 `build/id`는 두 token의 path이며 서버가 depth를 판단합니다. Go URI ID 처리와 JSON Pointer는 별개입니다. required ID는 native upstream ServiceURL에 전달되며 owned Record API의 literal identity guard·single-segment escaping을 이 호출의 보장으로 옮기지 않습니다.

## 생략·nil·빈 값·옵션

`UpdateOpts`는 배열이며 Patch 타입의 zero value를 명시하면 그대로 전송합니다. 조회·diff·기본값 추론·중복 path 정리·name lookup·후속 GET·wait를 추가하지 않습니다.

| 입력 | native 동작 |
|---|---|
| 해당 Patch 자체를 생략 | 해당 operation을 전송하지 않음 |
| nil `UpdateOpts` 또는 `UpdateOpts{}` | 둘 다 nonnil `[]`를 PATCH, HTTP를 생략하지 않음 |
| `NewName: ""`, `NewHidden: false`, `NewMinDisk: 0` | 빈 string·false·0을 명시적으로 전송 |
| `ReplaceImageTags{}` 또는 `NewTags: nil` | `value: null` |
| `NewTags: []string{}` | `value: []`, 전체 tag set을 비우는 요청 |
| 같은 값을 보내거나 같은 path를 여러 번 지정 | local 비교 없이 배열 순서와 중복 operation을 보존 |
| `UpdateImageProperty.Value` | string만 표현, JSON null·array·number 입력은 이 필드의 범위 밖 |

`images.WithUpdateOptions(value)`는 base UpdateOpts 전체를 교체하고 뒤에 지정한 옵션이 이깁니다. nil option·callback 오류는 HTTP 전에 반환합니다. nil UpdateOpts와 배열 안의 nil Patch는 별개입니다. upstream은 각 Patch의 ToImagePatchMap을 직접 호출하므로 nil 원소를 안전한 빈 변경으로 처리하거나 별도 preflight 오류로 바꾸지 않습니다. generic `request.WithField/WithQuery/WithHeader/WithArgument`로 설정한 nonempty 확장 값은 이 generated Update의 capability 검사에서 거부됩니다. 이 연산에는 `WithUpdateField/Query/Header` helper가 없습니다.

upstream의 `UpdateOptsBuilder` interface 대신 SDK 내부 builder를 사용하지만 배열 원소의 `Patch` interface alias는 유지합니다. 특수 map이 필요하면 caller의 custom Patch 확장이 여전히 가능하며, 이는 inherited native 확장입니다. 새 builder를 구현해야 하는 일반 호출 흐름으로 요구하지 않습니다. Patch 구현체·slice·map에 대한 owned snapshot·arbitrary callback 안전성을 이 native facade의 보장으로 주장하지 않습니다. 호출 중 공유 입력을 동시에 변경하지 않습니다.

## 실제 200 응답과 Extract·오류

[고정 Update](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/images/requests.go#L236-L248)는 `application/openstack-images-v2.1-json-patch` Content-Type과 `OkCodes: []int{200}`을 사용합니다. 기본 정상 응답은 actual 200 하나이며 201·202·204 등은 기본 정책에서 오류입니다. 원래 Gophercloud client/provider의 microversion·서비스 헤더·재인증·retry·redirect 설정을 사용합니다. service MoreHeaders가 요청 헤더를 덮는 native 정책도 유지합니다. 명시적인 native hooks의 정책 변경을 owned source guard로 제한하는 API는 아닙니다.

SDK는 upstream `UpdateResult.Extract()`를 호출하고 `resource.OperationError`로 원인 오류를 감쌉니다. native Image의 nullable 구분·float64 경유 size·날짜·Properties와 header 소비 규칙을 그대로 사용합니다. 응답 header의 import methods/store IDs를 Image 필드에 반영하지만 별도 Envelope·raw receipt·StatusCode를 성공 결과에 추가하지 않습니다. native HTTP 오류는 `errors.As`로 `gophercloud.ErrUnexpectedResponseCode`, context 원인은 `errors.Is`로 확인합니다.

[고정 Extract](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/images/results.go#L161-L183)는 Image pointer와 decode 오류를 함께 반환할 수 있습니다. SDK가 이를 atomic 모델로 바꾸거나 partial Image를 버리지 않으므로 항상 err를 먼저 판단합니다. valid JSON null은 nil Image·nil error가 될 수 있어 nonnil 모델도 별도로 확인합니다. HTTP 접수 뒤 Extract 실패가 발생했다면 오류만으로 서버의 수정이 없었다고 판단할 수 없습니다. native body 읽기·Close·response validation 범위를 owned ResponseError·독립 receipt 계약으로 확대하지 않습니다.

## Python dirty state와 별도 owned 수정 범위

```python
import openstack

conn = openstack.connect(cloud="dev")
# string ID는 자동 GET 없이 ID-only Image를 준비한다.
updated = conn.image.update_image("image-id", name="release-image", is_protected=False)
# 같은 mutable Resource를 재사용한다. clean 상태에 같은 raw값이면 HTTP를 생략할 수 있다.
same = conn.image.update_image(updated, name=updated.name)
assert same is updated
```

[Python proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1056-L1068)는 attrs로 새 Image를 만들거나 기존 instance를 변경한 뒤 commit합니다. [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1875-L1937)은 id의 dirty 표시를 버리고 Body/Header dirty가 없으면 HTTP를 생략합니다. dirty가 있으면 original/current raw Body의 properties를 root로 펼쳐 JSONPatch를 만들고 path 순으로 정렬합니다. 최종 diff가 빈 배열이라는 이유만으로 commit을 생략하는 규칙은 아닙니다.

생략과 명시 null·default-equal 값은 다릅니다. ID-only Image에 `name=None` 또는 `tags=[]`를 지정하면 getter 기본값과 같아도 raw key를 새로 넣은 변경입니다. 이미 저장된 동일 raw값을 지정하면 clean 상태를 유지할 수 있습니다. unknown attrs는 client properties에 packing하지만 한번의 _update에서 properties 묶음을 교체할 수 있어 sibling property의 remove까지 생성할 수 있습니다. 일반 native property add/upsert와 같은 merge 계약이 아닙니다.

현재 `images.API.Update`는 위 Python dirty lifecycle을 재현하지 않으며 계획한 owned `UpdateImageRecord` 수정 계층은 아직 public API로 제공하지 않으며 별도 구현·검증 대상입니다. 기존 [상위 명시적 PATCH 가이드](../../update.md) 역시 입력 Patch를 매번 보내는 계약을 유지합니다. [Source pyproject](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/pyproject.toml#L15-L20)는 jsonpatch dependency를 `!=1.20,>=1.16`으로 허용하므로 SDK revision만으로 모든 nested/array diff의 한 버전 byte 결과를 고정했다고 주장하지 않습니다. arbitrary mutable Resource/subclass·Munch·dynamic session/cache와 lifecycle 비교 범위도 남습니다.

## 권한·검증 경계

[고정 Glance 기본 정책](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L81-L116)의 일반 `modify_image`는 project scope의 `ADMIN_OR_PROJECT_MEMBER`이므로 접근 가능한 project-owned 이미지의 기본 수정 경로는 핵심 user 후보입니다. [API policy](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/policy.py#L203-L229)는 visibility에 추가 검사를 수행합니다. public visibility의 `publicize_image`는 admin, community의 `communitize_image`는 admin 또는 project member입니다.

[controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L657-L718)는 owner replace에 admin을 요구하고 locations의 추가·교체·삭제를 별도 정책으로 처리합니다. [`delete_image_location`의 기본 rule](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L160-L171)은 admin입니다. 따라서 모든 Patch 필드가 일반 user에게 허용된다고 넓히지 않습니다. 서버가 readonly/schema·이미지 상태·property protection·소유권·visibility와 배포의 policy override를 판단합니다. SDK는 role 검사나 자동 권한 상승을 추가하지 않습니다.

기존 직접 native 호출 근거는 [false·빈 값 Patch 테스트](../../../api/contracts_test.go#L133), [native Update의 custom property·false 테스트](../../update_contracts_test.go#L219), [전체 tags PATCH 테스트](../../mutation_contracts_test.go#L877)에 있습니다. 신규 native 전용 계약과 실행 receipt는 [지원 판정대장](../../../docs/sdk-support-ledger.md)에서 별도로 관리합니다. generated 함수가 있다는 사실·문서 빌드·기존 개별 anchor만으로 모든 native 또는 Python 업데이트를 완료로 판정하지 않습니다.
