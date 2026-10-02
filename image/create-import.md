# Glance 이미지 생성과 interoperable import

`image.Service.CreateAndImport`는 새 이미지 metadata를 만들고 import를 시작하는 명시적인 흐름입니다. 기본값은 `qcow2` disk format, `bare` container format, `private` visibility와 `glance-direct` method이며 완료 대기는 기본적으로 꺼져 있습니다. 기존 `Service.Upload`의 직접 file upload 동작은 유지됩니다.

기본 요청 순서는 `POST /images` 201 → `PUT /images/{id}/stage` 204 → `GET /images/{id}` 200 → `POST /images/{id}/import` 202입니다. 모든 옵션·source·요청 본문·wait 설정을 첫 POST나 reader 소비 전에 검사합니다. 다음 예는 입력 stream을 stage하고 import를 시작한 뒤 `active`까지 명시적으로 기다립니다.

```go
package example

import (
    "context"
    "io"

    "github.com/gophercloud/gophercloud/v2"
    "gophercloudsdk/image"
    "gophercloudsdk/image/v2/imagedata"
)

func createAndWait(ctx context.Context, client *gophercloud.ServiceClient, data io.Reader, size int64) (*image.CreateImportResult, error) {
    return image.New(client).CreateAndImport(ctx,
        image.CreateAndImportRequest{Name: "worker", Data: data},
        image.WithCreateImportStage(imagedata.StageOpts{Size: &size}),
        image.WithCreateImportWait(image.CreateImportWaitOpts{}),
    )
}
```

Python의 대응하는 import 선택은 다음과 같습니다. 두 Python 비교 예는 cloud configuration의 `image_api_use_tasks`가 꺼진 일반 import 경로를 전제로 합니다. 이 경로는 `wait`와 `timeout`을 전달하지 않으므로 이 호출 자체가 `active`를 기다린다고 해석하지 않습니다.

```python
image = conn.image.create_image(
    "worker", data=data, disk_format="qcow2", container_format="bare",
    visibility="private", use_import=True, import_method="glance-direct",
    allow_duplicates=True, disable_vendor_agent=False, size=size)
```

Go의 `CreateImportOpts`는 `Metadata`, `Stage`, `Import`, `Wait`를 각각 소유합니다. `WithCreateImportOpts`는 전체 설정을 교체하고 `WithCreateImportMetadata` / `Stage` / `Import`는 해당 설정을 교체합니다. pointer·slice·map·JSON 값을 snapshot하며 reader는 caller에게 남습니다. `WithCreateImportStageOptions`와 `WithCreateImportImportOptions`의 leaf callback은 preparation 단계에서 한 번 적용됩니다. 실행 단계는 그 결과의 SDK-owned snapshot을 사용합니다. leaf callback batch는 최종 concrete policy에 preparation 단계에서 적용됩니다. `WithCreateImportStage` / `Import`는 이전 해당 callback batch를 지우고, batch helper는 이전 batch를 교체합니다.

Metadata는 formats, visibility, optional protected/hidden, min disk/RAM, tags, properties와 POST-only headers를 받습니다. nil 값은 기본값/생략 정책을 따르고 명시적인 false와 zero는 보존됩니다. JSON extension으로 SDK가 소유한 core field를 덮어쓰거나 임의 request builder를 공급하지 않습니다. Glance가 property schema와 deployment의 지원 format을 판단합니다.

## 원격 import와 wait

`web-download`는 URI를, `glance-download`는 remote region과 image ID를 지정합니다. 이 두 경로는 새 metadata 생성 후 import를 시작하며 reader와 stage 설정을 받지 않습니다. `copy-image`는 기존 active image를 위한 경로이므로 새 이미지 workflow에서 허용하지 않습니다. 알려지지 않은 method도 이 workflow에서 사전 거부하며 기존 leaf `ImportImage`의 server-owned passthrough 동작은 유지됩니다.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "gophercloudsdk/image"
    "gophercloudsdk/image/v2/imageimport"
)

func createFromWeb(ctx context.Context, client *gophercloud.ServiceClient, uri string) (*image.CreateImportResult, error) {
    requireAll := false
    return image.New(client).CreateAndImport(ctx,
        image.CreateAndImportRequest{Name: "worker-from-web"},
        image.WithCreateImportImport(imageimport.ImportOpts{
            Method: "web-download", URI: uri,
            Stores: []string{"fast", "reliable"}, AllStoresMustSucceed: &requireAll,
        }),
    )
}
```

```python
image = conn.image.create_image(
    "worker-from-web", use_import=True, import_method="web-download", uri=uri,
    disk_format="qcow2", container_format="bare", visibility="private",
    stores=["fast", "reliable"], all_stores_must_succeed=False,
    allow_duplicates=True, disable_vendor_agent=False)
```

Go는 선택한 stores와 명시적인 all-store boolean을 함께 보존합니다. pinned Python은 `stores is not None`이면 all-store flag를 import kwargs에서 제외하므로 두 번째 비교 예의 `False`도 실제 import 요청에 들어가지 않습니다. singular `Store`는 leaf import의 호환 header-only 정책을 따릅니다. method별 URI/remote/store/header 충돌은 첫 create 전에 검사합니다. method 지원·store 존재·권한·quota·URL 허용과 backend 결과는 서버가 판단하며 추가 capability discovery를 수행하지 않습니다.

`Wait == nil`과 `WithoutCreateImportWait`는 대기를 끕니다. `WithCreateImportWait`를 선택하면 timeout의 nil은 5분, nonnil zero는 SDK timeout 없음, poll interval의 nil은 2초입니다. 부모 context는 전체 흐름과 대기를 제어합니다. failure states의 nil은 `killed`와 `deleted`, nonnil empty slice는 해당 failure 검사를 끕니다. 이 timeout은 대기 단계에서 시작하며 전체 생성·전송 시간 제한은 caller context로 정합니다. 성공한 waiter의 실제 모델만 `Ready`에 들어갑니다.

## 실제 응답과 stream 소유권

`CreateImportResult`의 `Created`, `Staged`, `Imported`는 실제 받아들인 각 단계 응답을 보존합니다. `CreatedImageResponse`에는 실제 201의 native Image·Body·Header·StatusCode가 들어가며 read/decode 실패 시 원문 증거와 error를 함께 반환합니다. `Staged`는 PUT204 acknowledgement와 후속 GET200 증거를 구분하고, `Imported`는 실제 POST202 acknowledgement를 보존합니다. 202만으로 active Image·Task·store 완료를 합성하지 않습니다. 받아들인 201 이전에는 result가 nil입니다.

최초 lowercase canonical `id`가 `ImageID`와 이후 route를 고정합니다. 비어 있거나 안전한 단일 route segment가 아닌 ID는 거부합니다. Location/self/file/schema 또는 typed Image의 case alias·다른 ID가 이를 바꾸지 않습니다. direct import는 생성 응답의 정확한 lowercase `status`가 `queued`인지 검사한 뒤 reader를 소비합니다. import의 형식은 실제 post-stage GET 본문의 정확한 lowercase `container_format`과 `disk_format`에서 읽습니다. stale 생성 모델이나 alias에 의해 다른 format으로 import하지 않습니다. native 전체 Image decoding과 canonical decision 검사는 별도이며 returned typed 모델과 원문 Body/Header는 독립적인 응답 증거입니다.

원격 import는 실제 생성 응답의 canonical formats를 사용하고 stage를 거치지 않습니다. 생성 모델의 import-method/store-ID capability header projection은 native `CreateResult.Extract`를 따르며 원문 Body/Header를 변경하지 않습니다.

`API.PrepareStageOptions`와 `API.PrepareImportOptions`는 source를 검증하고 concrete 옵션·JSON·header 설정을 복사하는 local-only preparation API입니다. HTTP나 reader 소비 없이 callback을 한 번 적용하고, import preparation은 configured source의 store 선택 충돌도 검사합니다. 이 workflow가 이후 단계 옵션 오류를 첫 POST 전에 검출하는 데 사용합니다.

Data는 현재 cursor부터 읽습니다. SDK는 입력을 Close·Seek·buffer하거나 파일 이름·전체 크기를 추론하지 않습니다. 선택한 Size는 nil일 때 생략하고 zero를 포함한 nonnegative int64만 `X-OpenStack-Image-Size`에 전달하며 Content-Length로 바꾸거나 그 크기에 맞춰 stream을 자르지 않습니다. caller가 stream을 닫고, blocking Read의 해제도 caller가 처리합니다.

binary stage는 SDK retry·reauth·backoff·redirect를 끈 private provider로 한 번 전송하며 caller stream을 replay하지 않습니다. metadata/create/import/wait의 JSON 요청은 기존 configured provider 정책을 사용하므로 임의 transport 내부의 replay나 모든 요청의 packet-level exactly-once를 보장하지 않습니다. client의 prefix·headers·선택 설정은 capture하고 원래 Provider의 live token을 사용합니다. source 무효화는 다음 단계 전에 error로 반환합니다.

오류가 나면 이미 받은 단계 증거와 원래 HTTP·read·decode·context 원인을 보존합니다. 자동 DELETE·재생성·restage·recovery request는 수행하지 않습니다. 실패 결과의 `ImageID`로 caller가 정리 여부를 결정할 수 있습니다.

## 원본 SDK와의 범위

비교 기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [Proxy.create_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L178-L438)와 [_upload_image_put](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L727-L822)입니다. Python은 cloud configuration의 formats·vendor flags·Swift/task 선택과 filename/name inference, duplicate/hash freshness, MD5/SHA256·legacy checksum 검증, kwargs 문자열/정수 coercion·meta precedence, seekable whole-file size와 mutable Resource/cache를 포함합니다. 이 Go workflow는 해당 자동 동작을 추가하지 않습니다.

Python의 create-response import-method gate와 upload/import/checksum 실패의 자동 DELETE도 다릅니다. Go는 새로운 side effect와 실제 응답을 caller에게 남기고 method 지원 여부는 서버 응답으로 판단합니다. Python의 일반 import 경로는 create의 wait/timeout을 사용하지 않으며 configured Swift/task 경로만 해당 controls를 전달받습니다. Go의 선택적 active wait는 명시적인 추가 정책입니다. 전체 Resource defaults/descriptors/dirty state/cache/session/adapter와 cloud workflow parity는 남아 있습니다.

[공식 Image API](https://docs.openstack.org/api-ref/image/v2/index.html)에서 201·204·202와 import method별 precondition을 확인할 수 있습니다. 로컬 HTTP 증명은 deployment capability·version·storage 성공을 보장하지 않습니다. 기존 [Stage](v2/imagedata/README.md), [Import](v2/imageimport/README.md), [Service.Upload](README.md)와 native Create/Get/Stage/Upload는 기존 API를 유지합니다.

실제 계약은 [공개 workflow HTTP 테스트](create_import_contracts_test.go), [canonical route·단계 응답·wait 테스트](create_import_core_test.go), [옵션 소유권·metadata 재사용 테스트](create_import_options_test.go), [local-only stage preparation 테스트](v2/imagedata/stage_prepare_test.go), [JSON·store 충돌 import preparation 테스트](v2/imageimport/import_prepare_test.go)에서 확인합니다.
