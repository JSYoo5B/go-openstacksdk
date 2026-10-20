# Root Access

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:db/v1/instances.IsRootEnabled`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/db/v1/instances.API.IsRootEnabled`

- The effective ResourceBase is used for a single bodyless authenticated GET /instances/{id}/root. The public bool,error signature and native literal-true semantics survive: true returns true while false, absent, differently cased, null, string, number, array and object field values return false,nil.
  Tests: `api/trove_root_enabled_contracts_test.go:TestTroveRootEnabledContractsObjectFieldSemantics`.
- Accepted HTTP200 top-level null, array, string, number and bool JSON return false with a descriptive SDK envelope error and IsRootEnabled/instances operation context; no native status error, processing RetryFunc invocation, extra lookup or fallback is introduced.
  Tests: `api/trove_root_enabled_contracts_test.go:TestTroveRootEnabledContractsSuccessfulNonObjectsDoNotRetryOrFallback`.
- Physical403/404/500/204 failures return false and preserve native ErrUnexpectedResponseCode method, URL, expected200, actual status, raw bytes and response headers through IsRootEnabled/instances OperationError, rather than panicking or replacing the cause with envelope validation.
  Tests: `api/trove_root_enabled_contracts_test.go:TestTroveRootEnabledContractsNativeStatusErrors`.
- Accepted empty, incomplete and malformed JSON return false and preserve native EOF, unexpectedEOF or SyntaxError causes with operation context, rather than panicking.
  Tests: `api/trove_root_enabled_contracts_test.go:TestTroveRootEnabledContractsNativeJSONErrors`.
- Native transport failures and canceled caller contexts return false with inspectable original causes; a canceled request does not reach the HTTP server and no SDK lookup is introduced.
  Tests: `api/trove_root_enabled_contracts_test.go:TestTroveRootEnabledContractsTransportAndContext`.
- Provider-owned401 reauthentication,429 backoff and503 RetryFunc replay remain unchanged. Requests retain native GET JSON decoding, routes/source headers and refreshed authentication, and callback header additions survive without replacing the shared client/provider.
  Tests: `api/trove_root_enabled_contracts_test.go:TestTroveRootEnabledContractsNativeReauthenticationBackoffAndRetry`.
- Native JSON decode and transport error RetryFunc handling can recover through one provider replay with a live refreshed token. SDK-owned extraction does not alter that native retry boundary.
  Tests: `api/trove_root_enabled_contracts_test.go:TestTroveRootEnabledContractsNativeDecodeAndTransportRetry`.
- A native retry callback can veto a decode retry with a sentinel; the exact sentinel becomes the OperationError cause without replacement and no additional request occurs.
  Tests: `api/trove_root_enabled_contracts_test.go:TestTroveRootEnabledContractsNativeCallbackFailure`.
- Original HTTPClient redirect policy either returns the native307 status error or follows with the caller-edited header to return a boolean; no SDK fallback route is added.
  Tests: `api/trove_root_enabled_contracts_test.go:TestTroveRootEnabledContractsNativeRedirectPolicy`.
- Every exact map[string]any including a typed nil map preserves the native comparison to literal true. Absent/null/nonbool and named bool values remain false,nil, and extraction does not mutate the map.
  Tests: `internal/troveroot/extract_test.go:TestExtractPreservesLiteralTrueMapSemantics`.
- Successful nil/nonmap, differently typed or named-map bodies return false and a descriptive EnvelopeError instead of an unchecked assertion panic. The helper does not fabricate accepted HTTP status or raw response bytes.
  Tests: `internal/troveroot/extract_test.go:TestExtractRejectsSuccessfulNonObjectEnvelopeWithoutPanic`.
- The original stored native error takes precedence over every envelope, including a true map; extraction returns false and exactly that error without envelope replacement.
  Tests: `internal/troveroot/extract_test.go:TestExtractOriginalErrorWinsBeforeReadingBody`.
- Only db/v1/instances.IsRootEnabled receives safe extraction. Same-named operations in other packages and sibling functions sharing the result retain their original Extract handling; generic extraction and normalizer selection remain unchanged.
  Tests: `internal/cmd/sdkgen/trove_root_enabled_extractors_test.go:TestTroveRootEnabledAllowlistKeepsGenericExtractionAndOtherOperations`.
- Nineteen typed request/result/own-method mutations, including context/client/id/result, parameter arity/variadic shape, result embedding/tag/fields and promoted/pointer/parameterized/nonbool/nonerror extractors, fail validation before emission.
  Tests: `internal/cmd/sdkgen/trove_root_enabled_extractors_test.go:TestTroveRootEnabledRejectsRequestResultAndOwnMethodGraphDrift`.
- The actual compiled pinned native graph and all three request/extractor/URL declaration hashes are verified. Missing declarations or changes to native HTTP policy, recorded errors, rootEnabled key, literal comparison or root route fail generation validation.
  Tests: `internal/cmd/sdkgen/trove_root_enabled_extractors_test.go:TestTroveRootEnabledGuardsActualThreePinnedDeclarations`.
- The source loader rejects missing operation/extractor/URL, duplicate reviewed declarations and absent native metadata/directory/files instead of silently overwriting or trusting incomplete native source.
  Tests: `internal/cmd/sdkgen/trove_root_enabled_extractors_test.go:TestTroveRootEnabledRejectsMissingAndDuplicateNativeSource`.
- Actual emission matches stored IsRootEnabled, EnableRootUser and Get declarations. Only root status calls the safe helper on result.Result with the unchanged bool,error wrapper, and exactly one inventory row carries sdk_trove_root_enabled_boolean while return_policy remains extract.
  Tests: `internal/cmd/sdkgen/trove_root_enabled_extractors_test.go:TestTroveRootEnabledActualEmissionStoredFacadeAndBoundedInventory`.
- Stored declarations for Barbican payload, Cinder snapshot metadata, Swift and Glance downloads and existing GetPassword/authentication/virtual-media normalizers match actual emission and never select the Trove helper.
  Tests: `internal/cmd/sdkgen/trove_root_enabled_extractors_test.go:TestTroveRootEnabledKeepsExistingAuditedResultFamilies`.

Python/Go differences:

- The public bool,error facade uses SDK-owned error-first result handling for this pinned native leaf only. Native Result.Err is returned before envelope inspection and wrapped as IsRootEnabled/instances OperationError. All exact map[string]any values retain native rootEnabled == true semantics, including typed nil maps and absent/null/nonbool fields. Successful nil or nonmap envelopes now return a descriptive error instead of a native assertion panic; native accepted status/raw bytes are not reconstructed.
- Native GET URL, JSON decoding, HTTP200 policy, original shared ServiceClient/ProviderClient/auth/source headers/retries/redirects remain unchanged. Public native result aliases still expose the pinned native Extract; other instance operations and result families retain their contracts. The actual Python Instance.is_root_enabled Resource method casts a dictionary field without runtime bool coercion and has different missing/nonbool behavior; no synthetic Proxy operation or full inherited Resource/session/cache parity is claimed.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk"
)

func rootEnabled(ctx context.Context, conn *openstack.Connection, instanceID string) (bool, error) {
	service, err := conn.DatabaseV1(ctx)
	if err != nil {
		return false, err
	}
	return service.Instances.IsRootEnabled(ctx, instanceID)
}
```

```python
from openstack.database.v1.instance import Instance

instance = Instance(id=instance_id)
enabled = instance.is_root_enabled(conn.database)
```

## References

- [`Instance.is_root_enabled(session)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/database/v1/instance.py#L73-L85)
- [Trove Root Enabled Contracts Test](../../../api/trove_root_enabled_contracts_test.go)
- [..](../README.md)
