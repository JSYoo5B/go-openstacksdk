# Volume을 image로 업로드

`UploadVolumeToImage`는 명시한 현재 Cinder volume ID와 image name을 받아 v3 `os-volume_upload_image` action을 호출합니다. SDK가 nullable 옵션·기본값·microversion·응답 proof를 소유하므로 caller builder가 필요하지 않습니다. 이 helper는 이미지 파일 bytes를 보내거나 Glance의 이미지 준비 완료를 기다리지 않습니다.

기준 Source는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [Proxy.upload_volume_to_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1387-L1420)와 [Volume.upload_to_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L375-L403)입니다.

| 호출 경로 | API | 입력·결과 |
|---|---|---|
| Python Proxy | `conn.block_storage.upload_volume_to_image(volume_id, image_name, ...)` | 기본 force=False; action envelope의 선택한 값 반환 |
| Go Connection | `conn.UploadVolumeToImage(ctx, request, imageName, options...)` | cached Cinder v3·owned options; `*VolumeImageUploadResult` |
| Go package | `blockstorage.UploadVolumeToImage(ctx, client, request, imageName, options...)` | 선택한 `*gophercloud.ServiceClient`·같은 결과 |
| Go service API | `cinder.Volumes.UploadVolumeToImage(ctx, volumeID, imageName, options...)` | `Volumes` API **필드**; request 대신 ID string |

`VolumeActionRequest{VolumeID: ...}`는 안전한 현재 route ID입니다. image name은 required typed UTF-8 string이며 **명시한 빈 값도** 보내고 enum·nonempty 제약을 추가하지 않습니다. image/volume name finder, CurrentLocation, Resource 모델 refresh, wait·cleanup·rollback을 실행하지 않습니다.

## 한 번의 업로드 요청

다음 독립 프로그램은 clouds.yaml의 `CLOUD_NAME`, 현재 `VOLUME_ID`, 환경변수 `IMAGE_NAME`의 존재를 요구합니다. `IMAGE_UPLOAD_PATH=connection|direct|service` 기본값은 connection이며 한 실행에서 action 한 번만 요청합니다. 30초 parent context는 application의 인증·discovery·POST 전체 제한입니다.

`IMAGE_UPLOAD_FORCE` 기본값은 false입니다. `IMAGE_UPLOAD_DISK_FORMAT`·`IMAGE_UPLOAD_CONTAINER_FORMAT`·`IMAGE_UPLOAD_VISIBILITY`는 생략하면 전송하지 않고, 빈 환경변수를 명시하면 빈 string을 보냅니다. `IMAGE_UPLOAD_PROTECTED=false`도 명시한 bool로 포함되어 required3.1 support gate를 실행합니다. bool 파싱은 예제 application의 입력 처리입니다.

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "strconv"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    if err := run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run() error {
    cloudName, volumeID := os.Getenv("CLOUD_NAME"), os.Getenv("VOLUME_ID")
    if cloudName == "" || volumeID == "" { return fmt.Errorf("CLOUD_NAME and VOLUME_ID are required") }
    imageName, supplied := os.LookupEnv("IMAGE_NAME")
    if !supplied { return fmt.Errorf("IMAGE_NAME is required; explicit empty is allowed") }
    path := os.Getenv("IMAGE_UPLOAD_PATH")
    if path == "" { path = "connection" }
    if path != "connection" && path != "direct" && path != "service" { return fmt.Errorf("unknown IMAGE_UPLOAD_PATH %q", path) }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    var options []blockstorage.VolumeImageUploadOption
    if text, supplied := os.LookupEnv("IMAGE_UPLOAD_FORCE"); supplied {
        value, err := strconv.ParseBool(text)
        if err != nil { return fmt.Errorf("IMAGE_UPLOAD_FORCE: %w", err) }
        options = append(options, blockstorage.WithVolumeImageUploadForce(value))
    }
    if text, supplied := os.LookupEnv("IMAGE_UPLOAD_DISK_FORMAT"); supplied {
        options = append(options, blockstorage.WithVolumeImageUploadDiskFormat(text))
    }
    if text, supplied := os.LookupEnv("IMAGE_UPLOAD_CONTAINER_FORMAT"); supplied {
        options = append(options, blockstorage.WithVolumeImageUploadContainerFormat(text))
    }
    if text, supplied := os.LookupEnv("IMAGE_UPLOAD_VISIBILITY"); supplied {
        options = append(options, blockstorage.WithVolumeImageUploadVisibility(text))
    }
    if text, supplied := os.LookupEnv("IMAGE_UPLOAD_PROTECTED"); supplied {
        value, err := strconv.ParseBool(text)
        if err != nil { return fmt.Errorf("IMAGE_UPLOAD_PROTECTED: %w", err) }
        options = append(options, blockstorage.WithVolumeImageUploadProtected(value))
    }
    policy, err := blockstorage.PrepareVolumeImageUploadOptions(ctx, options...)
    if err != nil { return err }
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil { return err }
    input := blockstorage.VolumeActionRequest{VolumeID: volumeID}
    result, err := apply(ctx, conn, path, input, imageName, policy)
    inspect(result, err)
    return err
}

func apply(ctx context.Context, conn *sdk.Connection, path string, input blockstorage.VolumeActionRequest, imageName string, policy blockstorage.VolumeImageUploadOpts) (*blockstorage.VolumeImageUploadResult, error) {
    option := blockstorage.WithVolumeImageUploadOptions(policy)
    if path == "connection" { return conn.UploadVolumeToImage(ctx, input, imageName, option) }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { return nil, err }
    if path == "direct" { return blockstorage.UploadVolumeToImage(ctx, cinder.RawClient(), input, imageName, option) }
    return cinder.Volumes.UploadVolumeToImage(ctx, input.VolumeID, imageName, option)
}

func inspect(result *blockstorage.VolumeImageUploadResult, err error) {
    if result != nil {
        fmt.Printf("volume ID: %q; microversion: %q\n", result.VolumeID, result.Microversion)
        for index, page := range result.Discovery {
            if page != nil { fmt.Println("admitted discovery:", index, page.StatusCode, len(page.Body)) }
        }
        if page := result.Applied; page != nil { fmt.Printf("admitted action: HTTP %d; actual body %q\n", page.StatusCode, page.Body) }
        if result.Upload != nil {
            fmt.Printf("selected upload JSON: %s\n", result.Upload)
            // Optional application inspection; API success does not require image_id.
            var details map[string]json.RawMessage
            if json.Unmarshal(result.Upload, &details) == nil {
                if id, present := details["image_id"]; present { fmt.Printf("optional image_id raw value: %s\n", id) }
            }
        }
        fmt.Println("helper acknowledgement/parse completed:", result.Completed)
    }
    if err != nil {
        var proof *resource.ResponseError
        if errors.As(err, &proof) { fmt.Println("current admitted error response:", proof.StatusCode, len(proof.Body)) }
    }
}
```

일반 호출은 `conn.UploadVolumeToImage(ctx, request, "new-image", blockstorage.WithVolumeImageUploadForce(false))`처럼 factory를 바로 전달하면 충분합니다. 위 예제는 HTTP 없이 Prepare한 독립 policy를 전체 교체 factory로 재사용합니다. service 패키지 `blockstorage/v3/volumes`에도 같은 opts/option alias·factory·Prepare 이름이 있습니다.

Python에서는 다음처럼 실제 Proxy를 호출합니다. 각 줄은 별도 요청입니다.

```python
import openstack

conn = openstack.connect(cloud="devstack")
upload = conn.block_storage.upload_volume_to_image(volume_id, "new-image")
upload = conn.block_storage.upload_volume_to_image(
    volume_id, "", force=False, disk_format="", container_format="")
upload = conn.block_storage.upload_volume_to_image(
    volume_id, "new-image", visibility="", protected=False)
# The selected value need not be a dict or include image_id.
```

## 옵션 기본값·소유권

`VolumeImageUploadOpts`는 `Force *bool`, `DiskFormat`·`ContainerFormat`·`Visibility *string`, `Protected *bool`을 갖습니다. nil Force는 독립 default=false pointer가 됩니다. `image_name`과 `force`는 항상 action body에 포함합니다. optional string/protected가 nil이면 생략하고 nonnil 빈 string·protected=false면 포함합니다. raw/qcow2·bare 또는 visibility의 새 기본값·enum 검증을 만들지 않습니다. format과 image name의 remote 유효성은 Cinder가 판단합니다.

`WithVolumeImageUploadOptions`는 이전 모든 pointer를 완전한 owned policy로 교체하며 nil Force를 다시 default=false로 적용합니다. field factories는 해당 pointer만 설정합니다. 최종 실제 string만 UTF-8를 검사하여 이전 잘못된 값을 뒤 옵션에서 교체할 수 있습니다. Prepare는 HTTP·service selection 없이 original callbacks를 한 번 실행하고 오류에는 zero policy를 반환합니다.

SDK는 original 옵션 slice를 실행 전에 복사하고 각 callback 경계에서 nullable pointer/policy를 복사합니다. 입력·반환 policy·retained callback pointer의 나중 변경으로 캡처한 요청을 바꿀 수 없으며 native retry마다 callback을 다시 실행하지 않습니다. package/service는 selected Source·현재 route ID·required imageName UTF-8를 검사한 뒤 originals를 실행합니다. Connection은 context/provider 검사→originals Prepare 한 번→required imageName·현재 ID 검사→cached Cinder v3 선택 순서입니다.

## Microversion과 응답

Visibility 또는 Protected가 **nonnil**이면 값이 빈 string/false여도 required3.1 support gate가 있습니다. selected version이 있어도 유한 discovery에서 양쪽 server bounds가 fixed3.1을 포함하는지 확인하고, selected version은 같은 major·3.1 이상이어야 합니다. gate는 selected version 자체가 server maximum 이하인지를 추가 검사하지 않으므로 selected3.90과 server3.0..3.50은 fixed3.1 gate를 통과할 수 있으며 action에는 selected3.90을 그대로 보냅니다. bounds 누락·지원범위/selected 호환성 불충족은 `ErrUnsupported`이며 parse·IO·source/context 오류는 별개입니다.

Visibility/Protected가 둘 다 nil이면 minimum gate가 없습니다. nonempty selected version을 literal로 쓰고, 없으면 ordinary cap3.71/server maximum/optional minimum을 사용합니다. gate가 필요한 unselected 호출도 같은 finite discovery 결과로 ordinary version을 선택합니다. global `latest`는 required3.1의 finite major와 호환되지 않지만 `3.latest`는 같은-major 비교를 통과할 수 있습니다. 원본 client의 version을 바꾸지 않으며 explicit raw selected header는 Keystoneauth Session의 별도 canonicalization까지 재현하지 않습니다. [pin된 dependency의 version_match](https://github.com/openstack/keystoneauth/blob/e759cf88c072d5bc7e21fc3fa884708ace9630af/keystoneauth1/discover.py#L433-L458)는 이 gate 비교를 설명합니다.

`VolumeImageUploadResult`는 `VolumeID`·`Microversion`, 실제 discovery별 `Discovery`, POST의 `Applied`, 선택한 `Upload json.RawMessage`, `Completed`를 분리합니다. `VolumeImageUploadResponse` alias의 Body/Header/StatusCode는 실제 단계별 독립 snapshot입니다. original policy가 admitted한 HTTP100..399라도 action body는 **UTF-8 JSON outer object와 정확한 `os-volume_upload_image` key**가 필요합니다. key의 값은 object·array·string·number·bool·literal null 모두 성공이며 `image_id`의 존재/타입은 필수가 아닙니다. 성공한 null은 Upload의 literal `null` bytes이며 오류의 Upload nil과 다릅니다.

`Upload`는 선택한 JSON 원문을 독립적으로 소유하므로 큰 숫자를 float로 미리 변환하지 않습니다. object 내부 unknown fields와 숫자·escape·duplicate member 표현은 Go raw JSON 값으로 남습니다. optional `image_id` 해석은 application의 선택이며, 응답의 `id` 또는 `image_id`를 새 route로 사용하지 않습니다. `Applied.Body`는 전체 실제 envelope 원문입니다. Upload bytes를 출력할 때 `%s`를 사용할 수 있고, application이 재직렬화하면 표현이 달라질 수 있습니다.

missing key·nonobject outer shape·malformed/invalid-UTF-8 JSON은 Applied proof를 남긴 채 Completed=false/Upload=nil로 실패합니다. accepted Read/Close·source·custom cancellation에도 해당 단계의 실제 proof가 남을 수 있습니다. native rejected action은 Applied가 없고 이전 Discovery를 현재 오류의 proof로 빌리지 않습니다. 파싱 오류를 이유로 POST를 재시도하거나 Glance GET·poll·rollback을 수행하지 않습니다. Completed=true는 helper의 acknowledgement·응답 선택이 끝났다는 뜻이며 image status=active나 데이터 복사 완료를 증명하지 않습니다.

## Proxy create_image: 기본 형식과 Image 반환

고정 Cinder v3 Proxy의 `create_image(name, volume, allow_duplicates, container_format, disk_format, wait, timeout)`는 위 upload 액션에 기본값을 채운 wrapper입니다. Go에서는 `conn.CreateVolumeImageRecord(ctx, openstack.VolumeImageCreateRequest{...})`가 담당합니다.

| Python 인자 | Go 필드 | 처리 |
|---|---|---|
| `name` | `Name` | upload의 `image_name` |
| `volume` | `VolumeID` | 조회 없이 그대로 사용 |
| `allow_duplicates` | `AllowDuplicates` | `force` 값 |
| `disk_format` | `DiskFormat` | 빈 값이면 cloud 설정 `image_format`(기본 `qcow2`), 설정이 null이면 생략 |
| `container_format` | `ContainerFormat` | 빈 값이면 `bare` |
| `wait`, `timeout` | 없음 | 고정 소스가 읽지 않으므로 제공하지 않음 |

액션 응답에서 `image_id`를 읽어 `image.Service.ExistingImageRecord`로 Python `Image.existing(id=...)`와 같은 동기화 record를 만듭니다. 이 record는 id만 가진 Image 선언 필드65개와 현재 Connection location을 담고 Glance 요청을 보내지 않습니다. 결과 `VolumeImageCreateResult`는 Cinder upload 증거와 Image를 분리합니다. 응답이 object가 아니거나 `image_id`가 없거나 문자열이 아니면 upload 증거와 함께 입력 오류를 반환합니다. 문자열이 아닌 설정 `image_format`은 HTTP 전에 거부합니다. 이미지 상태 대기는 Cloud `create_image`의 몫입니다.

## 기존 API·Source 경계

기존 native `cinder.Volumes.UploadImage(ctx, id, UploadImageOpts, ...)`는 모든 기본 DTO field의 omitempty와 HTTP202/typed `VolumeImage` 결과 계약을 유지합니다. 그 DTO의 force=false·protected=false·empty name/formats 생략 정책과 새 helper의 presence policy는 다릅니다. Glance `image.Service.UploadImage`의 metadata+파일 bytes workflow는 [별도 이미지 업로드 가이드](../image/upload-image.md)를 참고하세요. 이 Cinder helper가 그 workflow를 실행하지 않습니다.

Go의 안전한 현재 ID, required UTF-8 image name·nullable typed string/bool·owned raw JSON 결과는 Python의 supplied Resource/dict/Munch·dynamic None/nonstr 입력·constructor/descriptor/location/cache·mutable JSON object와 다른 매핑입니다. Python cast/type annotation은 runtime response shape를 제한하지 않습니다. Go의 ordinary UTF-8 JSON·숫자 precision·raw literal 표현·duplicate 처리·ResponseError는 Python Response.json parser/float/exception taxonomy 전체를 재현하지 않습니다. SDK finite discovery와 Source endpoint-data cache/중복 getter/Session header canonicalization도 별도 경계입니다.

retry/redirect에서 fixed method·URL·serialized body/framing/version과 captured Source를 검사하며 native live authentication을 사용할 수 있습니다. 원래 rejected HTTP는 native evidence로 남지만 native rejected body IO 전체 복원을 주장하지 않습니다. Source 정적 비교·독립 예제 컴파일·로컬 HTTP 검증은 인증된 OpenStack/Python runtime이나 전체 SDK 완성을 뜻하지 않습니다. 이 한 Proxy 선언의 매핑은 native/v2/Resource·Glance API의 지원을 자동 승격하지 않습니다.

[직접 Cinder actions](volume-actions.md), [Volume image metadata](volume-image-metadata.md), [Block Storage API](README.md)도 참고하세요.
