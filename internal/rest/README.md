# SDK-owned JSON request and response boundaries

`rest.DoJSON` handles HTTP execution and evidence for SDK-owned APIs. Service packages declare routes, explicit accepted codes, JSON envelopes, schemas, and defaults. This internal helper does not itself establish public API or Python operation support.

Before HTTP, it validates context, source, same-origin targets, and explicit success codes. JSON is marshaled once; headers and codes are copied. A nil Go body means no body; serialized null is an actual JSON body. Marshal failures preserve their causes before HTTP.

## Accepted responses

Accepted responses return owned raw Body, copied Header, and actual StatusCode. Empty accepted bodies are valid here; Object/Decode and service decoders enforce envelope/model requirements. Bodies are read and closed once. Read, Close, and context failures can return partial Response plus `resource.ResponseError`, preserving independent bytes/headers, ctx.Err(), and custom context causes. Inspect causes with errors.Is/errors.As.

No mutation is replayed after an accepted-body failure. A read failure does not reverse remote execution.

## Retry ownership

Native RetryFunc, reauthentication, and backoff apply before response-body ownership. DoJSON owns KeepResponseBody=true, JSONResponse=nil, RawBody=nil, and serialized JSON bytes. Retry callbacks that change ownership fail before another request with ErrInvalidOption and the original/callback causes. A JSONBody replacement is allowed only if it serializes to identical bytes, then becomes an owned RawMessage snapshot. In-place RawMessage mutation is detected; absent-body/null transitions differ. This does not freeze every native transport/session policy.

## Status and native boundaries

Final acceptance uses the original copied success codes. Expanding native OkCodes cannot turn another status into SDK success. Such responses are read/closed and return value ErrUnexpectedResponseCode with method, fixed URL, original expected codes, actual status/body/headers, and read/Close/context causes; no successful SDK Response is synthesized.

Responses rejected by native policy retain native body/error ownership. Native ErrUnableToReauthenticate exposes ErrOriginal and ErrReauth fields rather than unwrapping both. Fixedrequest preserves attempt method, origin, escaped path, query, authority, and live provider authentication while retaining scoped HTTP policy. Prepare client configuration before concurrency.

Tests: [response policy](response_policy_test.go), [package contracts](response_contracts_test.go), and [response handling](response_test.go).
