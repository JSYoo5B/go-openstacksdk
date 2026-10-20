# Signing

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:object_store/v1/generate_form_signature`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.Service.GenerateFormSignature`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithGenerateFormSignatureOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithGenerateFormSignatureKey`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithGenerateFormSignatureDigest`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithGenerateFormSignatureTimestamp`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithGenerateFormSignatureHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithGenerateFormSignatureHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithGenerateFormSignatureNewest`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithoutGenerateFormSignatureNewest`

- Explicit nonempty arbitrary binary key signs decoded endpoint/container/prefix path plus literal redirect, positive size/count and captured expiry as five newline-separated fields, returning owned FormSignatureResult without HTTP upload.
  Tests: `objectstorage/v1/temp_url_signing_contracts_test.go:TestTempURLSigningContractsKnownVectorsAndPaths`, `objectstorage/v1/form_signature_core_test.go:TestFormSignatureCoreVectorsAndLiteralPaths`, `connection_objectstorage_signing_test.go:TestConnectionSwiftSigningSharesClientAndCapturesEachWorkflow`.
- Nil key uses fresh container secondary then primary followed only on successful unusable keys by account secondary then primary; HTTP/atomic metadata/source errors stop fallback and missing key is semantic ErrNotFound without fabricated HTTP404.
  Tests: `objectstorage/v1/temp_url_signing_contracts_test.go:TestTempURLSigningContractsDiscoveryAndEvidence`, `objectstorage/v1/form_signature_core_test.go:TestFormSignatureCoreDiscoveryAndPhaseProof`.
- Time is captured before discovery; explicit time.Time epoch and checked int64 relative expiry require nonnegative epochs, positive timeout/limits and no numeric/string coercion or overflow.
  Tests: `objectstorage/v1/temp_url_signing_contracts_test.go:TestTempURLSigningContractsOptionsAndPreflight`, `objectstorage/v1/form_signature_core_test.go:TestFormSignatureCorePreflightAndAcceptedFailures`, `objectstorage/v1/form_signature_options_test.go:TestFormSignatureOptionsParallelReuseAndInvalidValues`.
- Endpoint path removes exactly one trailing slash and appends safe literal container/prefix without path.Clean or caller percent decoding; decoded UTF8 HMAC path and exactly-once escaped full endpoint URL are distinct owned representations.
  Tests: `objectstorage/v1/temp_url_signing_contracts_test.go:TestTempURLSigningContractsKnownVectorsAndPaths`, `objectstorage/v1/form_signature_core_test.go:TestFormSignatureCoreVectorsAndLiteralPaths`.
- Redirect is literal UTF8 emptyallowed, <=4096 bytes and no control/DEL/trailing hyphen, rejecting values altered by pinned FormPost rstrip before any discovery.
  Tests: `objectstorage/v1/temp_url_signing_contracts_test.go:TestTempURLSigningContractsOptionsAndPreflight`, `objectstorage/v1/form_signature_core_test.go:TestFormSignatureCorePreflightAndAcceptedFailures`.
- Concrete full options replace configuration; key/header/timestamp/newest factories and callbacks deep snapshot values, preserve nil newest reset and literal accepted binary key bytes; SHA1 default and SHA256/SHA512 emit lowercase unprefixed hex.
  Tests: `objectstorage/v1/temp_url_signing_contracts_test.go:TestTempURLSigningContractsOptionsAndPreflight`, `objectstorage/v1/form_signature_options_test.go:TestFormSignatureOptionsSnapshotsAndReplacement`, `objectstorage/v1/form_signature_options_test.go:TestFormSignatureOptionsParallelReuseAndInvalidValues`.
- Original Service/client/provider/all4API identities and source route/header guards span discovery; accepted-phase handling failures retain raw Discovery evidence but no signature/URL and advanced native policy remains explicitly separate.
  Tests: `objectstorage/v1/temp_url_signing_contracts_test.go:TestTempURLSigningContractsSourceAndContext`, `objectstorage/v1/temp_url_signing_contracts_test.go:TestTempURLSigningContractsNativePolicy`, `objectstorage/v1/form_signature_core_test.go:TestFormSignatureCorePreflightAndAcceptedFailures`, `objectstorage/v1/form_signature_core_test.go:TestFormSignatureCoreDiscoveryAndPhaseProof`, `internal/cmd/sdkgen/swift_signing_test.go:TestSwiftSigningPreservesNativeAndRegistry`.
- GenerateFormSignature signs endpoint path/container/prefix, redirect, size, count and expiry with HMAC-SHA1 like Python and discovers the key from the container before the account.
  Tests: `objectstorage/v1/python_rejudge_test.go:TestPythonRejudgeSwiftGenerateFormSignature`.

Python/Go differences:

- Go returns structured path/full URL/digest/discovery instead of Python tuple, supports three concrete digests and explicit clock snapshot.
- Go rejects redirect characters/size changed by the pinned parser and safely escapes output path; Python signs its supplied literal redirect/path unchanged.

### `python:object_store/v1/generate_temp_url`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.Service.GenerateTempURL`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithGenerateTempURLOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithGenerateTempURLKey`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithGenerateTempURLDigest`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithGenerateTempURLTimestamp`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithGenerateTempURLHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithGenerateTempURLHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithGenerateTempURLNewest`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithoutGenerateTempURLNewest`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithGenerateTempURLAbsolute`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithGenerateTempURLPrefix`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithGenerateTempURLISO8601`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithGenerateTempURLIPRange`

- Decoded literal UTF8 /v1/account/container/object path is signed without caller percent decoding; literal percent/question/hash/unicode and slash structure survive exactly-once URL path and url.Values query escaping.
  Tests: `objectstorage/v1/temp_url_signing_contracts_test.go:TestTempURLSigningContractsKnownVectorsAndPaths`, `objectstorage/v1/temp_url_core_test.go:TestTempURLCoreVectorsAndEscapedOutput`.
- ASCII HTTP token is uppercased without a local method whitelist; prefix mode permits empty object and signs prefix: path while IP restriction signs accepted literal ip= spelling.
  Tests: `objectstorage/v1/temp_url_signing_contracts_test.go:TestTempURLSigningContractsKnownVectorsAndPaths`, `objectstorage/v1/temp_url_core_test.go:TestTempURLCoreVectorsAndEscapedOutput`.
- Relative or absolute nonnegative int64 seconds use one pre-discovery captured Unix epoch with checked arithmetic; ISO8601 output is UTC year1..9999 with no Python float/string/local-time parsing.
  Tests: `objectstorage/v1/temp_url_signing_contracts_test.go:TestTempURLSigningContractsOptionsAndPreflight`, `objectstorage/v1/temp_url_core_test.go:TestTempURLCorePreflightAndTimeBoundaries`.
- IP restriction accepts untrimmed address or masked CIDR, rejects address zones and host-bit prefixes before HEAD, and does not rewrite accepted caller spelling.
  Tests: `objectstorage/v1/temp_url_signing_contracts_test.go:TestTempURLSigningContractsOptionsAndPreflight`, `objectstorage/v1/temp_url_core_test.go:TestTempURLCorePreflightAndTimeBoundaries`.
- Nil key auto-discovers account-only even when the URL path names a container; nonnil empty bytes are invalid and nonempty arbitrary binary bytes skip all metadata HTTP.
  Tests: `objectstorage/v1/temp_url_signing_contracts_test.go:TestTempURLSigningContractsDiscoveryAndEvidence`, `objectstorage/v1/temp_url_core_test.go:TestTempURLCoreDiscoveryAndNativePolicy`, `connection_objectstorage_signing_test.go:TestConnectionSwiftSigningSharesClientAndCapturesEachWorkflow`.
- Concrete option factories/callbacks own binary key, header maps and timestamp/newest pointers; fullopts replacement and WithoutNewest preserve nil versus explicit bool while SHA1default and SHA256/SHA512 emit raw lowercase hex.
  Tests: `objectstorage/v1/temp_url_signing_contracts_test.go:TestTempURLSigningContractsOptionsAndPreflight`, `objectstorage/v1/temp_url_options_test.go:TestTempURLOptionsSnapshotsAndReplacement`, `objectstorage/v1/temp_url_options_test.go:TestTempURLOptionsParallelReuseAndCallbackCauses`.
- Actual fresh HEAD204 evidence survives accepted failures and missing key; all errors stop selection/signing, Discovery key/flags clear on failure and phase metadata clears on outer original-source drift without fabricating status.
  Tests: `objectstorage/v1/temp_url_signing_contracts_test.go:TestTempURLSigningContractsDiscoveryAndEvidence`, `objectstorage/v1/temp_url_signing_contracts_test.go:TestTempURLSigningContractsSourceAndContext`, `objectstorage/v1/temp_url_core_test.go:TestTempURLCoreDiscoveryAndNativePolicy`.
- Output is owned path-plus-query URL, while original native Objects.CreateTempURL ABI/primary-only discovery and Python str/bytes return/coercion/session behavior remain distinct.
  Tests: `objectstorage/v1/temp_url_signing_contracts_test.go:TestTempURLSigningContractsNativePolicy`, `objectstorage/v1/temp_url_core_test.go:TestTempURLCoreDiscoveryAndNativePolicy`, `internal/cmd/sdkgen/swift_signing_test.go:TestSwiftSigningPreservesNativeAndRegistry`.
- GenerateTempURL signs METHOD, expiry and path with HMAC-SHA1 like Python, sends nothing when the key is given and HEADs only the account when it is not.
  Tests: `objectstorage/v1/python_rejudge_test.go:TestPythonRejudgeSwiftGenerateTempURL`.

Python/Go differences:

- Go accepts decoded UTF8 string and int64 seconds, escapes path/query once and returns structured owned result; Python accepts bytes/numeric/string timestamps and concatenates a literal query.
- Go includes SHA256/SHA512, explicit time snapshot and strict masked CIDR validation, while pinned Python signs SHA1 and applies its own parsing/warning behavior.
- Python accepts ISO 8601 strings as seconds; Go takes int64 seconds and WithGenerateTempURLAbsolute.
- Go escapes the path in the returned URL and orders query keys alphabetically; Python concatenates the literal path and temp_url_sig before temp_url_expires.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

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

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L998-L1210)
- [FormPost](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/formpost.py#L389-L424)
- [TempURL](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/tempurl.py#L759-L788)
- [Formpost](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/formpost.py#L296-L310)
- [Temp Url Key](temp_url_key.md)
- [Form core](form_signature_core_test.go)
- [Form Signature Options Test](form_signature_options_test.go)
- [TempURL core](temp_url_core_test.go)
- [Temp Url Options Test](temp_url_options_test.go)
- [Temp Url Signing Contracts Test](temp_url_signing_contracts_test.go)
- [Connection Objectstorage Signing Test](../../connection_objectstorage_signing_test.go)
- [Swift Signing Test](../../internal/cmd/sdkgen/swift_signing_test.go)
