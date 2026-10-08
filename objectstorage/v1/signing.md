# Swift FormPost와 TempURL 서명

`objectstorage/v1.Service.GenerateFormSignature`와 `GenerateTempURL`은 설치된 Temp URL 키로 HMAC을 계산합니다. 명시한 `Key`가 non-nil이면 HTTP를 보내지 않습니다. nil이면 기존 메타데이터 조회를 사용합니다. FormPost는 컨테이너의 secondary → primary 키를 먼저 보고, 성공한 조회에 사용 가능한 키가 없을 때 계정의 secondary → primary 키를 조회합니다. TempURL은 경로에 컨테이너가 있어도 계정만 조회합니다. 매번 새 조회를 하며, 조회 오류가 나면 다음 대상으로 넘어가지 않습니다.

이는 [고정된 Python proxy의 실제 두 구현](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L998-L1210)을 따른 선택입니다. Python의 FormPost 반환은 `(expires, signature)`, TempURL 반환은 경로와 query를 담은 `str` 또는 `bytes`입니다. Go는 `Path`, `URL`, `Signature`, `Expires`, `Digest`와 선택적인 `Discovery`를 갖는 독립 결과를 반환합니다. Form 결과의 `URL`은 endpoint origin을 포함한 업로드 대상이며, Temp 결과의 `URL`은 origin 없는 path + query입니다. 파일 업로드, HTML form 전송, 브라우저 redirect는 실행하지 않습니다.

다음 예제는 구성된 `Service`, 실제 설치된 binary key와 기준 시각을 받습니다. 두 옵션 묶음은 모두 명시한 키를 사용하므로 메타데이터 조회가 없습니다. `Headers`와 `Newest`는 nil 키를 사용하는 경우의 조회 설정입니다.

```go
package main

import (
	"context"
	"fmt"
	"time"

	swift "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1"
)

func signRequests(ctx context.Context, service *swift.Service, key []byte, now time.Time) error {
	form, err := service.GenerateFormSignature(ctx, "reports", swift.FormSignatureInput{
		ObjectPrefix:   "incoming/",
		RedirectURL:    "https://example.com/upload-finished",
		MaxFileSize:    64 * 1024 * 1024,
		MaxUploadCount: 3,
		Timeout:       300,
	},
		swift.WithGenerateFormSignatureOpts(swift.GenerateFormSignatureOpts{
			Headers: map[string]string{"X-Audit": "initial"},
		}),
		swift.WithGenerateFormSignatureKey(key),
		swift.WithGenerateFormSignatureDigest(swift.TempURLDigestSHA256),
		swift.WithGenerateFormSignatureTimestamp(now),
		swift.WithGenerateFormSignatureHeaders(map[string]string{"X-Audit": "form"}),
		swift.WithGenerateFormSignatureHeader("X-Audit", "form-final"),
		swift.WithGenerateFormSignatureNewest(true),
		swift.WithoutGenerateFormSignatureNewest(),
	)
	if err != nil {
		if form != nil && form.Discovery != nil {
			fmt.Println("FormPost 키 조회의 실제 응답 증거가 남아 있습니다.")
		}
		return err
	}
	fmt.Printf("FormPost 대상=%s 만료=%d digest=%s\n", form.URL, form.Expires, form.Digest)
	// HTML form의 signature에는 form.Signature를, expires에는 form.Expires를 넣습니다.
	// max_file_count는 FormSignatureInput.MaxUploadCount와 같은 값입니다.

	temp, err := service.GenerateTempURL(ctx, "/v1/AUTH_project/reports/incoming/", 300, "GET",
		swift.WithGenerateTempURLOpts(swift.GenerateTempURLOpts{
			Headers: map[string]string{"X-Audit": "initial"},
		}),
		swift.WithGenerateTempURLKey(key),
		swift.WithGenerateTempURLDigest(swift.TempURLDigestSHA512),
		swift.WithGenerateTempURLTimestamp(now),
		swift.WithGenerateTempURLHeaders(map[string]string{"X-Audit": "temp"}),
		swift.WithGenerateTempURLHeader("X-Audit", "temp-final"),
		swift.WithGenerateTempURLNewest(true),
		swift.WithoutGenerateTempURLNewest(),
		swift.WithGenerateTempURLAbsolute(false),
		swift.WithGenerateTempURLPrefix(true),
		swift.WithGenerateTempURLISO8601(true),
		swift.WithGenerateTempURLIPRange("192.0.2.0/24"),
	)
	if err != nil {
		if temp != nil && temp.Discovery != nil {
			fmt.Println("TempURL 키 조회의 실제 응답 증거가 남아 있습니다.")
		}
		return err
	}
	fmt.Printf("TempURL 경로=%s 만료=%d digest=%s\n", temp.Path, temp.Expires, temp.Digest)
	// 요청에는 temp.URL의 escaped path와 query를 그대로 사용합니다.
	return nil
}

func main() {}
```

`Key(nil)`은 자동 조회를 선택하고, non-nil 빈 byte slice는 유효하지 않습니다. nonempty 키는 UTF-8 문자열로 제한하지 않는 binary 값이며, 결과에 명시한 키를 노출하지 않습니다. 옵션 factory와 전체 옵션은 key bytes, header maps, timestamp/newest pointers를 복사합니다. 전체 옵션은 앞의 구성을 교체하고 callback은 각 실행에서 한 번 호출됩니다. `Without…Newest`는 nil로 되돌려 조회 header를 생략합니다.

`Digest`가 비어 있으면 Python과 같은 SHA1을 사용합니다. Go는 SHA256과 SHA512도 지원하며, `Signature`는 접두어 없는 소문자 hex입니다. 고정된 Swift middleware는 hex 길이로 digest를 구분합니다. [FormPost](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/formpost.py#L389-L424)와 [TempURL](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/tempurl.py#L759-L788)의 허용 digest, method와 middleware 설치 여부는 서버 설정에 따릅니다. SHA1 사용 가능 여부도 서버가 결정합니다.

Go의 문자열 경로는 decoded literal UTF-8입니다. 이미 percent-escaped 문자열로 넘기지 않습니다. 문자 `%`, `?`, `#`, Unicode와 object 내부 slash를 literal 데이터로 서명한 뒤 출력 URL의 path/query를 한 번 escaping합니다. 예를 들어 literal `%2F`는 URL에서 `%252F`가 됩니다. Temp 경로는 `/v1/account/container/object` 모양이며 prefix 옵션에서는 마지막 object가 비어도 됩니다. invalid UTF-8, control/DEL, backslash와 `.`/`..` path segment는 거부합니다. Form 경로는 파싱한 endpoint의 decoded path 끝에서 slash 한 개만 제거하고 literal container/prefix를 붙입니다. `path.Clean`이나 입력 percent decoding을 적용하지 않습니다. proxy의 외부 경로 rewrite는 배포 설정의 경계입니다.

Form의 `RedirectURL`은 빈 값도 가능한 literal UTF-8 필드입니다. control/DEL, UTF-8 byte 길이 4096 초과와 끝의 `-`는 거부합니다. [고정된 FormPost parser](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/formpost.py#L296-L310)가 필드 끝의 CR/LF/hyphen을 `rstrip`으로 제거한 뒤 서명 검증에 사용하기 때문입니다. redirect를 URL로 변환하거나 trim하지 않습니다. `MaxFileSize`, `MaxUploadCount`, `Timeout`은 모두 1 이상이며, 실제 multipart 필드 이름은 `max_file_count`입니다. 서버는 파일별 subrequest를 순서대로 실행하므로 서명 생성은 업로드나 rollback 성공을 뜻하지 않습니다.

시간은 HTTP 조회 전에 한 번 정합니다. nil timestamp는 현재 Unix seconds, 지정 timestamp는 `time.Time.Unix()`를 사용하며 음수 epoch와 상대 시간 덧셈 overflow를 거부합니다. Temp의 `seconds`는 0 이상인 int64이고 기본은 상대 시간입니다. absolute 옵션은 그 값을 그대로 epoch로 사용합니다. ISO8601 출력은 UTC `YYYY-MM-DDTHH:MM:SSZ`이며 연도 1–9999만 허용합니다. Python의 float/string 시간 변환이나 local-time parsing은 제공하지 않습니다. method는 ASCII HTTP token을 요구하고 대문자로 바꾸며 whitelist는 추가하지 않습니다.

`IPRange`는 trim하지 않은 IP address 또는 CIDR이고, address zone과 host bits가 남은 network prefix를 거부합니다. 예를 들어 `192.0.2.7/24` 대신 `192.0.2.0/24`를 사용합니다. 승인된 문자열 spelling은 HMAC에 그대로 넣고 query에는 escaping합니다. 빈 값은 IP 제한을 생략합니다.

자동 조회에서는 현재 [Temp URL 키 워크플로](temp_url_key.md)의 전체 atomic metadata projection과 실제 HEAD204 응답을 보존합니다. 사용 가능한 키가 없으면 `resource.ErrNotFound`로 분류하며 가짜 HTTP404를 만들지 않습니다. 첫 accepted 응답 이전의 오류는 nil 결과이고, accepted 응답 이후에는 오류와 함께 `Discovery`의 raw body/header/status가 남을 수 있습니다. 오류 결과에는 서명이나 URL이 없습니다. malformed counter를 포함한 메타데이터 오류도 fallback을 멈춥니다. 원본 Service/client/provider, 네 개 public API pointer, endpoint/resource base/type/microversion을 조회 단계마다 확인하고, ordinary headers는 snapshot하며 인증 token은 같은 provider의 최신 값을 사용합니다. 명시한 native 재시도 정책의 고급 header 변경 경계는 기존 메타데이터 조회와 같습니다.

Python에서는 다음 두 실제 proxy를 사용합니다. 별도 digest 인자가 없으므로 SHA1입니다. Form은 tuple을 반환하고 Temp는 이 예제에서 문자열 경로를 반환합니다.

```python
key = b"replace-with-installed-secret"
expires, signature = conn.object_store.generate_form_signature(
    "reports",
    "incoming/",
    "https://example.com/upload-finished",
    64 * 1024 * 1024,
    3,
    300,
    temp_url_key=key,
)
temp_path = conn.object_store.generate_temp_url(
    "/v1/AUTH_project/reports/incoming/",
    300,
    "GET",
    prefix=True,
    iso8601=True,
    ip_range="192.0.2.0/24",
    temp_url_key=key,
)
```

기존 native `Objects.CreateTempURL`과 생성된 API는 유지됩니다. native는 primary-only 컨테이너 → 계정 조회, 기존 옵션과 완전한 URL 반환을 사용합니다. Python의 mutable `Container`, numeric coercion, bytes 경로 반환, literal query concatenation은 Go의 concrete 옵션과 escaping 규칙과 다릅니다. 두 실제 proxy에 대한 비교는 partial이며 나머지 Python Resource/session 동작이나 전체 SDK 동등성을 주장하지 않습니다.

서명과 입력 경계는 [Form core](form_signature_core_test.go)·[옵션](form_signature_options_test.go), [TempURL core](temp_url_core_test.go)·[옵션](temp_url_options_test.go), 실제 HTTP·binary key·알려진 HMAC 값은 [외부 계약 테스트](temp_url_signing_contracts_test.go)에 연결되어 있습니다. [Connection 테스트](../../connection_objectstorage_signing_test.go)와 [generator 회귀 테스트](../../internal/cmd/sdkgen/swift_signing_test.go)는 공유 client와 기존 native API 보존을 확인하는 선언입니다.
