# Glance 이미지 데이터를 writer로 다운로드

`image.Service.DownloadTo`는 이미지 metadata를 먼저 조회하고 binary 응답을 caller의 `io.Writer`에 기록합니다. 기본 chunk는 1 MiB이고, 사용 가능한 checksum은 기본적으로 검증합니다. 기본 요청은 `GET /images/{id}` 200 → `GET /images/{id}/file` 200입니다. SDK는 전체 payload를 결과에 저장하지 않습니다.

```go
package example

import (
    "context"
    "io"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/gophercloudsdk/image"
    "github.com/JSYoo5B/gophercloudsdk/resource"
)

func downloadTo(ctx context.Context, client *gophercloud.ServiceClient, id string, output io.Writer) (*image.DownloadImageResult, error) {
    return image.New(client).DownloadTo(ctx, resource.ID(id), output)
}
```

Python의 대응하는 output 경로는 다음과 같습니다. 이 예에서 output은 caller가 연 file object입니다. 고정된 Python 구현도 metadata를 먼저 조회하며, output 경로는 응답을 모두 소비한 뒤 사용 가능한 checksum을 확인합니다.

```python
response = conn.image.download_image(image_id, output=output, stream=True)
```

service·source·context·Ref·writer와 모든 옵션은 HTTP나 `Write` 전에 검사합니다. nil과 typed-nil writer는 거부합니다. `resource.ID`는 직접 fresh metadata GET을 수행합니다. `resource.Name`은 모든 목록 page에서 정확한 이름을 찾아 중복을 검사한 뒤, 선택한 ID로 fresh GET을 수행합니다. 이름 검색의 pagination은 기존 Images pager 정책을 따릅니다. ID나 이름을 자동 판별하는 API와는 별개의 명시적인 선택입니다.

선택한 ID는 전체 흐름의 route를 고정합니다. metadata 본문의 정확한 lowercase `id`는 해당 ID와 같아야 합니다. typed Image의 case alias, `self`·`file`·`schema`, Location이나 다른 URL이 binary route를 바꾸지 않습니다. 응답의 상태나 store capability로 추가 discovery를 수행하지 않습니다.

## Chunk와 store 선호 순서

`DownloadImageOpts`는 `ChunkSize`, `StorePreferences`, `VerifyChecksum`을 소유합니다. `WithDownloadImageOpts`는 전체 설정을 교체하고, `WithDownloadChunkSize` / `WithDownloadStorePreferences` / `WithDownloadChecksumVerification`은 해당 설정을 교체합니다. pointer와 slice는 snapshot하며 custom callback은 preparation에서 한 번 적용합니다. retained caller 설정이 전송 중 옵션을 바꾸지 않습니다.

`ChunkSize == nil`은 1 MiB입니다. 명시적인 값은 1 byte부터 64 MiB까지 허용합니다. nil/empty store preferences는 `prefer`를 생략합니다. nonempty preferences는 순서와 중복을 보존해 하나의 comma-separated query 값으로 URL encoding합니다. 빈 식별자, comma, control character와 invalid UTF-8은 첫 metadata GET 전에 거부합니다.

다음 예는 이름으로 선택하고, 64 KiB chunk와 `fast,archive,fast` 선호 순서를 사용합니다. `bytes.Buffer`는 caller가 선택한 메모리 output입니다. 오류가 나도 이미 기록된 byte slice와 실제 진행 결과를 함께 돌려줍니다.

```go
package example

import (
    "bytes"
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/gophercloudsdk/image"
    "github.com/JSYoo5B/gophercloudsdk/resource"
)

func downloadByName(ctx context.Context, client *gophercloud.ServiceClient, name string) ([]byte, *image.DownloadImageResult, error) {
    var output bytes.Buffer
    chunk := 64 * 1024
    result, err := image.New(client).DownloadTo(ctx, resource.Name(name), &output,
        image.WithDownloadImageOpts(image.DownloadImageOpts{
            ChunkSize: &chunk,
            StorePreferences: []string{"fast", "archive", "fast"},
        }),
    )
    return output.Bytes(), result, err
}
```

```python
image = conn.image.find_image(name, ignore_missing=False)
response = conn.image.download_image(
    image, output=output, stream=True, chunk_size=64 * 1024,
    store_preferences=["fast", "archive", "fast"])
```

Python의 `find_image`는 name-or-ID를 해석하고 mutable Image를 다음 호출에 전달합니다. Go의 위 예는 명시적인 정확한 Name 검색과 fresh metadata 응답을 사용합니다. [공식 download API](https://docs.openstack.org/api-ref/image/v2/index.html#download-binary-image-data)는 `prefer`를 2026.1에서 도입한 선호 순서로 정의합니다. cache에서 제공하거나 선호 store에서 찾지 못하면 서버의 기본 동작이 적용될 수 있습니다. SDK는 해당 deployment의 지원 여부·권한·store 존재를 미리 보장하지 않습니다.

## Checksum 선택과 결과

checksum은 raw metadata의 정확한 lowercase field에서 선택합니다. nonempty `os_hash_algo`와 `os_hash_value` pair가 우선이며, incomplete pair는 metadata `checksum`, 마지막으로 binary 응답의 literal `Content-MD5`로 fallback합니다. 대소문자 alias나 typed property 변환은 이 결정을 대신하지 않습니다. digest 문자열을 hex normalization하거나 Content-MD5를 base64 decode하지 않습니다.

지원하는 알고리즘은 `md5`, `sha1`, `sha224`, `sha256`, `sha384`, `sha512`, `sha512_224`, `sha512_256`입니다. complete primary pair의 알고리즘이 지원되지 않으면 binary GET 전에 `resource.ErrUnsupported`를 반환합니다. 검증이 켜진 canonical hash field의 잘못된 타입도 binary 요청 전에 거부합니다. `WithDownloadChecksumVerification(false)` 또는 `VerifyChecksum`의 명시적인 false는 canonical hash 검사를 끄며, 전체 native Image decode의 검사는 유지합니다.

`Checksum`이 nil이면 검증이 꺼졌거나 사용할 hash가 없거나 실제 binary 응답이 204입니다. 선택한 hash가 있으면 `Algorithm`, `Expected`, `Actual`, `Complete`, `Verified`가 검증 증거를 나타냅니다. metadata에서 선택한 `Algorithm`과 `Expected`는 binary GET 실패에도 결과에 남습니다. 전체 EOF와 성공한 모든 write 후에만 `Complete`와 `Actual`을 기록하며, digest가 일치할 때만 `Verified`가 true입니다. mismatch는 `ErrChecksumMismatch`와 `DownloadChecksumMismatchError`를 통해 algorithm·expected·actual을 함께 반환합니다. mismatch 시 이미 output에 기록한 bytes는 남습니다.

최종 HTTP body Close만 실패한 경우에는 완료한 전송과 일치한 digest의 `Complete`/`Verified`를 유지하면서 Close error를 반환합니다. read·write·취소 때문에 중단한 전송은 완료나 검증 성공을 표시하지 않습니다.

## 실제 응답, 진행과 소유권

`DownloadImageResult.ImageID`는 고정된 선택 ID이고 `Image`는 실제 metadata 응답을 native `GetResult.Extract`로 decode한 모델입니다. `Metadata.Body`·`Header`·`StatusCode`는 받아들인 실제 metadata 200의 원문 증거를 보존합니다. read·decode·canonical 검사 실패에도 해당 증거와 error를 함께 반환합니다. 받아들인 metadata 응답 이전에는 result가 nil입니다.

결과의 `Header`와 `StatusCode`는 받아들인 실제 binary 응답만 나타냅니다. binary HTTP/open 실패에는 metadata와 원래 native error를 돌려주며 받아들인 binary 응답을 합성하지 않습니다. `BytesWritten`은 writer가 성공했다고 보고한 실제 byte 수이며 전체 payload나 content length를 대신하지 않습니다. 단계별 header·raw metadata·native Image는 독립적인 증거입니다.

이 workflow는 전체 다운로드의 binary 200과 데이터가 없는 204만 받습니다. 204에서는 writer를 호출하거나 checksum을 검증하지 않습니다. Range 요청을 제공하지 않으며 source의 Range/If-Range header는 metadata GET 전에 거부합니다. 요청하지 않은 partial 206과 다른 status는 거부합니다. 200에 Content-Range가 붙으면 받아들인 header/status는 남기고 첫 Write 전에 오류를 반환합니다. [공식 Image API](https://docs.openstack.org/api-ref/image/v2/index.html#download-binary-image-data)는 full 200·no-data 204·partial 206을 구분합니다.

writer는 현재 위치에서 빌립니다. SDK는 caller writer를 Close·Seek·truncate하거나 파일을 열지 않습니다. bounded copy는 `ReaderFrom`/`WriterTo` shortcut에 의존하지 않으며, short write를 오류로 처리합니다. read와 write가 같은 chunk에서 함께 실패하면 두 원인과 성공한 byte 수를 보존합니다. output을 rollback하거나 이미 소비한 응답을 다시 받아 suffix를 재생하지 않습니다.

SDK가 소유한 받아들인 HTTP body는 모든 종료 경로에서 한 번 닫습니다. 받아들이지 않은 응답 body는 native Request가 소유하고 닫으며 workflow가 다시 닫지 않습니다. read·write·Close·context 원인은 `errors.Is`/`errors.As`로 확인할 수 있게 유지하며, parent context의 custom cause도 함께 보존합니다. blocking `Write`의 해제는 caller가 처리해야 합니다.

client의 prefix·headers·선택 설정은 capture하고 원래 Provider의 live token과 configured reauth/retry를 사용합니다. binary 재인증과 retry는 받아들인 body를 반환하기 전의 요청 단계에만 적용됩니다. 전송 중 read/write 실패 후에는 다시 다운로드하지 않습니다. binary redirect는 거부하며 source 무효화는 후속 요청 전에 검사합니다. caller context와 HTTPClient timeout은 전체 요청과 전송을 제어합니다.

## 원본 SDK와의 범위

비교 기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [Proxy.download_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L919-L968), [DownloadMixin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/_download.py#L26-L183), [Proxy.request](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L209-L285)입니다.

Python의 binary GET은 expected status를 지정하거나 response error를 변환하지 않고, Proxy.request의 `raise_exc` 기본값은 false입니다. 반환된 error body를 output에 소비할 수 있으므로 Go의 명시적인 200/204 검사는 더 강한 정책입니다. Python은 binary GET 뒤 hash를 선택하며 unsupported algorithm의 ValueError가 알고리즘 이름을 포함하면 MD5 또는 검증 없는 전송으로 fallback할 수 있습니다. Go의 unsupported-primary 사전 거부와 bounded hash registry도 차이입니다.

Python의 output path opening·IOBase polymorphism, stream-only response, 기본 full-memory mode, Content-MD5 rewriting, mutable Image/cache/descriptor/dirty state/session/adapter는 이 writer workflow의 범위 밖에 남습니다. Python output 모드는 write count를 검사하지 않고 오류를 문자열로 감싸지만 Go는 short-write 계약과 partial byte/error chain을 보존합니다. Python의 일부 반환 설명과 달리 실제 구현은 output 경로에서도 Response를 반환합니다.

기존 native [ImageData.Download](v2/imagedata/README.md)는 raw body API로 계속 호출할 수 있습니다. 다운로드와 [CreateAndImport](create-import.md), [Service.Upload](README.md)는 독립적인 작업이며, 실제 cloud 다운로드 성공·store 정책·cache 효과나 전체 Python Resource parity는 로컬 HTTP 증명만으로 확정하지 않습니다.

실제 sequence·preflight·canonical integrity·snapshot·partial failure·204/native compatibility·provider·writer 취소 계약은 [공개 HTTP 테스트](download_contracts_test.go), [core 증거 테스트](download_core_test.go), [옵션 소유권 테스트](download_options_test.go)에서 확인합니다. [생성기 회귀 테스트](../internal/cmd/sdkgen/glance_download_test.go)는 native Download signature·Image ID/checksum field와 raw stream binding의 보존을 검사합니다.
