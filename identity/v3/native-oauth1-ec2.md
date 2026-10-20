# Native Oauth1 Ec2

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:identity/v3/oauth1.CreateConsumer`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/oauth1.API.CreateConsumer`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/oauth1.WithCreateConsumerOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/oauth1.WithCreateConsumerField`

- CreateConsumer POSTs {"consumer":{"description":...}} to OS-OAUTH1/consumers always sending description (empty included), keeping extensions inside the envelope and rejecting a description extension before HTTP, decodes the consumer id, secret and description and accepts only 201.
  Tests: `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1ConsumerRoutesBodiesPagingAndDecode`, `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1StatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/oauth1 call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/oauth1.GetConsumer`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/oauth1.API.GetConsumer`

- GetConsumer GETs OS-OAUTH1/consumers/{id}, decodes the consumer envelope and accepts only 200.
  Tests: `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1ConsumerRoutesBodiesPagingAndDecode`, `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1StatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/oauth1 call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/oauth1.UpdateConsumer`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/oauth1.API.UpdateConsumer`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/oauth1.WithUpdateConsumerOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/oauth1.WithUpdateConsumerField`

- UpdateConsumer PATCHes {"consumer":{"description":...}} to OS-OAUTH1/consumers/{id} always sending description, keeping extensions inside the envelope and rejecting a description extension before HTTP, decodes the consumer envelope and accepts only 200.
  Tests: `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1ConsumerRoutesBodiesPagingAndDecode`, `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1StatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/oauth1 call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/oauth1.DeleteConsumer`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/oauth1.API.DeleteConsumer`

- DeleteConsumer DELETEs OS-OAUTH1/consumers/{id} without a body and accepts 202 or 204.
  Tests: `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1ConsumerRoutesBodiesPagingAndDecode`, `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1StatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/oauth1 call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/oauth1.ListConsumers`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/oauth1.API.ListConsumers`

- ListConsumers GETs OS-OAUTH1/consumers, follows the links.next string across hosts, stops on a null next link, yields nothing for an empty consumers array, returns io.EOF for a bodyless 204 and an unexpected-code error expecting 200, 204 or 300 for 404.
  Tests: `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1ConsumerRoutesBodiesPagingAndDecode`, `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1StatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/oauth1 call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/oauth1.RequestToken`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/oauth1.API.RequestToken`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/oauth1.WithRequestTokenOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/oauth1.WithRequestTokenHeader`

- RequestToken POSTs no body to OS-OAUTH1/request_token with an OAuth Authorization header carrying sorted oauth_callback=oob, oauth_consumer_key, oauth_nonce, oauth_signature_method, oauth_timestamp and oauth_version=1.0 plus an HMAC-SHA1 or PLAINTEXT oauth_signature over the full URL (nonce and timestamp generated when unset), sends RequestedProjectID as Requested-Project-Id, requires the consumer key and signature method and rejects Authorization or Requested-Project-Id extension headers before HTTP, still sends the provider X-Auth-Token, accepts only 201 and decodes an exactly application/x-www-form-urlencoded body into oauth_token, oauth_token_secret and an RFC3339Milli oauth_expires_at, failing on any other Content-Type.
  Tests: `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1SignedRequestAndAccessTokens`, `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1TokenResponseDecodeFailures`, `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1StatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/oauth1 call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/oauth1.AuthorizeToken`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/oauth1.API.AuthorizeToken`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/oauth1.WithAuthorizeTokenOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/oauth1.WithAuthorizeTokenField`

- AuthorizeToken PUTs an unwrapped {"roles":[...]} body (null when no roles are given) to OS-OAUTH1/authorize/{id} with extensions at the root, rejects an empty role or a roles extension before HTTP, decodes token.oauth_verifier and accepts only 200.
  Tests: `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1AccessTokenRoutesAuthorizePagingAndDecode`, `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1StatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/oauth1 call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/oauth1.CreateAccessToken`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/oauth1.API.CreateAccessToken`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/oauth1.WithCreateAccessTokenOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/oauth1.WithCreateAccessTokenHeader`

- CreateAccessToken POSTs no body to OS-OAUTH1/access_token with an OAuth Authorization header carrying sorted oauth_consumer_key, oauth_nonce, oauth_signature_method, oauth_timestamp, oauth_token, oauth_verifier and oauth_version=1.0 plus an oauth_signature keyed by the consumer and token secrets, requires the consumer key, token, verifier and signature method but not the secrets, rejects an Authorization extension header before HTTP, accepts only 201 and decodes the exact form-encoded token response.
  Tests: `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1SignedRequestAndAccessTokens`, `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1TokenResponseDecodeFailures`, `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1StatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/oauth1 call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/oauth1.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/oauth1.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/oauth1.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/oauth1.WithCreateField`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/oauth1.WithCreateHeader`

- Create POSTs {"auth":{"identity":{"methods":["oauth1"],"oauth1":{}}}} to auth/tokens with extensions inside auth and no scope, signs an OAuth Authorization header with oauth_token and the consumer and token secrets, still sends the provider X-Auth-Token (empty when the provider has none), rejects Authorization or X-Auth-Token extension headers and a missing token before HTTP, accepts only 201, returns a tokens.Token whose ID is X-Subject-Token without the OS-OAUTH1 extension and leaves the ProviderClient token and ReauthFunc unchanged.
  Tests: `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1CreateAuthTokenBodyHeadersAndSubjectToken`, `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1StatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/oauth1 call only; Python identity Resource and keystoneauth OAuth1 plugin parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/oauth1.GetAccessToken`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/oauth1.API.GetAccessToken`

- GetAccessToken GETs users/{user}/OS-OAUTH1/access_tokens/{id}, decodes the access_token envelope with an RFC3339Milli expires_at and accepts only 200.
  Tests: `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1AccessTokenRoutesAuthorizePagingAndDecode`, `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1StatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/oauth1 call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/oauth1.RevokeAccessToken`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/oauth1.API.RevokeAccessToken`

- RevokeAccessToken DELETEs users/{user}/OS-OAUTH1/access_tokens/{id} without a body and accepts 202 or 204.
  Tests: `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1AccessTokenRoutesAuthorizePagingAndDecode`, `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1StatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/oauth1 call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/oauth1.ListAccessTokens`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/oauth1.API.ListAccessTokens`

- ListAccessTokens GETs users/{user}/OS-OAUTH1/access_tokens, follows the links.next string, decodes access_tokens rows, yields nothing for an empty array, returns io.EOF for a bodyless 204 and an unexpected-code error expecting 200, 204 or 300 for 404.
  Tests: `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1AccessTokenRoutesAuthorizePagingAndDecode`, `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1StatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/oauth1 call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/oauth1.ListAccessTokenRoles`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/oauth1.API.ListAccessTokenRoles`

- ListAccessTokenRoles GETs users/{user}/OS-OAUTH1/access_tokens/{id}/roles, follows the links.next string, decodes roles rows with id, name and domain_id, yields nothing for an empty array, returns io.EOF for a bodyless 204 and an unexpected-code error expecting 200, 204 or 300 for 404.
  Tests: `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1AccessTokenRoutesAuthorizePagingAndDecode`, `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1StatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/oauth1 call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/oauth1.GetAccessTokenRole`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/oauth1.API.GetAccessTokenRole`

- GetAccessTokenRole GETs users/{user}/OS-OAUTH1/access_tokens/{id}/roles/{role}, decodes the role envelope and accepts only 200.
  Tests: `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1AccessTokenRoutesAuthorizePagingAndDecode`, `identity/v3/oauth1/native_contracts_test.go:TestNativeOAuth1StatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/oauth1 call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/ec2tokens.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/ec2tokens.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/ec2tokens.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/ec2tokens.WithCreateField`

- Create POSTs {"credentials":{...}} to ec2tokens sending an explicit signature with the non-omitempty fields as-is or computing an AWS V2 (HmacSHA1/HmacSHA256, dropping body_hash and headers) or V4 signature (adding X-Amz-Date and Authorization to the body headers, random body hash and current time when unset), removes token, keeps extensions inside credentials, requires non-nil options and access and rejects unsupported signature versions or methods and core-key extensions before HTTP, accepts only 200 and returns a tokens.Token whose ID is X-Subject-Token without touching the ProviderClient token, and rejects header extensions before HTTP because the native call never calls ToTokenV3HeadersMap.
  Tests: `identity/v3/ec2tokens/native_contracts_test.go:TestNativeEC2TokenBodiesSignaturesAndSubjectToken`, `identity/v3/ec2tokens/native_contracts_test.go:TestNativeEC2TokenHeaderExtensionIsRejected`, `identity/v3/ec2tokens/native_contracts_test.go:TestNativeEC2TokenStatusesAndPreflight`, `internal/cmd/sdkgen/page_extractors_test.go:TestUnreadBuilderMethodsDisableTheirExtensions`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/ec2tokens call only; Python identity Resource and keystoneauth EC2 plugin parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/ec2tokens.ValidateS3Token`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/ec2tokens.API.ValidateS3Token`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/ec2tokens.WithValidateS3TokenOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/ec2tokens.WithValidateS3TokenField`

- ValidateS3Token POSTs {"credentials":{...}} to s3tokens keeping only access, signature, token (the V4 string to sign when computed) and extensions, base64-encoding []byte values, requires non-nil options and access and rejects core-key extensions before HTTP, accepts only 200 and returns a tokens.Token whose ID is X-Subject-Token, and rejects header extensions before HTTP because the native call never calls ToTokenV3HeadersMap.
  Tests: `identity/v3/ec2tokens/native_contracts_test.go:TestNativeEC2TokenBodiesSignaturesAndSubjectToken`, `identity/v3/ec2tokens/native_contracts_test.go:TestNativeEC2TokenHeaderExtensionIsRejected`, `identity/v3/ec2tokens/native_contracts_test.go:TestNativeEC2TokenStatusesAndPreflight`, `internal/cmd/sdkgen/page_extractors_test.go:TestUnreadBuilderMethodsDisableTheirExtensions`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/ec2tokens call only; Python identity Resource and keystoneauth EC2 plugin parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

## References

- [oauth1](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/oauth1/requests.go)
- [ec2tokens](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/ec2tokens/requests.go)
