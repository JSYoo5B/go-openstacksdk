# SDK 소유 JSON 응답의 공통 경계

`rest.DoJSON`은 SDK가 소유한 API의 HTTP 요청과 응답 증거를 처리하는 내부 helper입니다. service package가 route, 명시적인 success codes, JSON envelope, schema와 기본값을 결정합니다. 이 helper는 public API의 서명을 추가하거나 Python operation의 지원 판정을 바꾸지 않습니다.

context·source·같은 origin의 target과 명시적인 success codes를 HTTP 전에 검사합니다. JSON 입력은 첫 요청 전에 marshal하고, headers와 success codes를 복사합니다. Go의 nil body는 요청 body 없음이며, typed nil 등을 marshal한 JSON `null`은 실제 body입니다. marshal 실패는 요청 전에 원래 오류를 반환합니다.

## 받아들인 응답과 오류 증거

원래 success codes에 해당하는 실제 응답은 raw `Body`, 복사한 `Header`, 실제 `StatusCode`를 가진 `rest.Response`로 반환합니다. 이 단계는 JSON을 decode하지 않으므로 empty accepted body도 유효합니다. `Object`·`Decode`와 service decoder가 필요한 JSON envelope와 모델을 검사합니다.

받아들인 body는 읽은 뒤 한 번 Close합니다. read·Close·context가 함께 실패하면 partial bytes와 실제 header/status를 가진 Response와 `resource.ResponseError`를 함께 반환합니다. `ctx.Err()`와 parent의 custom `context.Cause(ctx)`도 보존하며, read 실패가 Close나 취소 원인을 가리지 않습니다. Response와 ResponseError는 bytes와 header를 각각 소유하므로 한쪽을 수정해도 다른 증거를 바꾸지 않습니다. 원인은 `errors.Is`/`errors.As`로 확인할 수 있습니다.

HTTP body를 받아들인 뒤의 read·Close·context 실패는 요청을 다시 보내지 않습니다. mutation의 실제 성공 여부는 서비스 의미와 서버 응답을 함께 판단해야 하며, 응답 읽기 실패가 서버의 mutation을 되돌리지 않습니다.

## Retry callback과 요청 body 소유권

configured native `RetryFunc`·reauth·backoff는 응답 body를 소유하기 전의 요청 정책으로 유지합니다. 다만 DoJSON은 `KeepResponseBody=true`, `JSONResponse=nil`, `RawBody=nil`과 처음 serialize한 JSON body를 소유합니다. `RetryFunc`가 이 소유권을 바꾸면 다음 요청 전에 `resource.ErrInvalidOption`, 원래 요청 오류와 callback 오류를 함께 반환합니다. 변경한 JSONBody의 marshal이 실패하면 그 marshal 원인도 유지합니다.

JSONBody는 interface identity가 아니라 serialize한 bytes로 비교합니다. callback이 동일한 bytes를 serialize하는 값으로 교체하면 허용하며, 기존 `json.RawMessage`를 제자리에서 바꿔도 별도로 보관한 원본 bytes와 비교하여 거부합니다. 허용한 교체 값도 owned `json.RawMessage` snapshot으로 다시 설정하므로 native retry에서 stateful marshaler를 다시 호출해 body를 바꾸지 않습니다. 요청 body 없음과 JSON null의 전환도 변경입니다. 이 검사는 임의의 native callback·transport·session 정책 전체를 고정하는 계약이 아닙니다.

## 실제 status와 native 경계

최종 success 판정은 처음 복사한 명시적인 codes를 사용합니다. `RetryFunc`가 native `OkCodes`를 확대하여 다른 status를 받아들이게 해도 SDK success로 바뀌지 않습니다. 이 경우 DoJSON은 body를 읽고 한 번 닫은 뒤 method·고정 URL·원래 expected codes·actual status·raw bytes·header를 가진 value `gophercloud.ErrUnexpectedResponseCode`와 read·Close·context 원인을 반환합니다. SDK Response는 nil이며, 잘못된 status의 성공 결과나 accepted `ResponseError`를 합성하지 않습니다.

기본 native policy가 거부한 응답은 native Request의 body 소유권과 error 구조를 유지합니다. 그 native rejected-body 처리 전체를 새 accepted-body 규칙으로 바꾸지 않습니다. native reauth 실패의 `gophercloud.ErrUnableToReauthenticate`는 두 독립적인 원인을 Unwrap하지 않으므로 `errors.As`로 얻은 `ErrOriginal`·`ErrReauth` 필드에서 확인합니다.

기존 `fixedrequest`는 각 attempt의 method·origin·escaped path·query와 authority를 고정하며, 원래 Provider의 live auth를 사용합니다. 별도 scoped client의 HTTP policy를 사용하므로 shared provider를 교체하지 않습니다. 기존 redirect·reauth·retry 경계와 source 설정 계약을 유지하며 client 설정은 concurrent 사용 전에 준비합니다. DoJSON의 context 오류에는 가능한 `ctx.Err()`와 custom cause를 함께 보존합니다.

실제 accepted read/Close/context 증거, serialized body 소유권과 status gate는 [core policy 테스트](response_policy_test.go)와 [공개 package 경계 테스트](response_contracts_test.go)에서 검증합니다. 기존 JSON snapshot·decode·target·native error 계약은 [기존 응답 테스트](response_test.go)로 함께 유지합니다.
