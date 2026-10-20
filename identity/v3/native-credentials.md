# Native Credentials

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:identity/v3/applicationcredentials.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/applicationcredentials.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/applicationcredentials.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/applicationcredentials.WithCreateField`

- Create POSTs {"application_credential":{...}} to users/{user}/application_credentials always sending unrestricted (false included), formatting expires_at without a zone in the time's own location, keeping roles/access rules and extensions inside the envelope, rejects a missing name and core-key collision before HTTP, decodes the one-time secret and accepts only 201.
  Tests: `identity/v3/applicationcredentials/native_contracts_test.go:TestNativeApplicationCredentialRoutesBodiesPagingAndDecode`, `identity/v3/applicationcredentials/native_contracts_test.go:TestNativeApplicationCredentialStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/applicationcredentials call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/applicationcredentials.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/applicationcredentials.API.Get`

- Get GETs users/{user}/application_credentials/{id}, returns nil without error for a missing envelope, wraps a zoned expires_at decode error and accepts only 200.
  Tests: `identity/v3/applicationcredentials/native_contracts_test.go:TestNativeApplicationCredentialRoutesBodiesPagingAndDecode`, `identity/v3/applicationcredentials/native_contracts_test.go:TestNativeApplicationCredentialStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/applicationcredentials call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/applicationcredentials.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/applicationcredentials.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/applicationcredentials.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/applicationcredentials.WithListQuery`

- List GETs users/{user}/application_credentials with the name filter and extension query values, follows the Keystone links.next string until null, reports native page status errors without operation context and rejects a nil option before HTTP.
  Tests: `identity/v3/applicationcredentials/native_contracts_test.go:TestNativeApplicationCredentialRoutesBodiesPagingAndDecode`, `identity/v3/applicationcredentials/native_contracts_test.go:TestNativeApplicationCredentialStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/applicationcredentials call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/applicationcredentials.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/applicationcredentials.API.Delete`

- Delete DELETEs users/{user}/application_credentials/{id} and accepts only 202/204.
  Tests: `identity/v3/applicationcredentials/native_contracts_test.go:TestNativeApplicationCredentialRoutesBodiesPagingAndDecode`, `identity/v3/applicationcredentials/native_contracts_test.go:TestNativeApplicationCredentialStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/applicationcredentials call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/applicationcredentials.ListAccessRules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/applicationcredentials.API.ListAccessRules`

- ListAccessRules GETs users/{user}/access_rules and follows the Keystone links.next string until null.
  Tests: `identity/v3/applicationcredentials/native_contracts_test.go:TestNativeApplicationCredentialRoutesBodiesPagingAndDecode`, `identity/v3/applicationcredentials/native_contracts_test.go:TestNativeApplicationCredentialStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/applicationcredentials call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/applicationcredentials.GetAccessRule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/applicationcredentials.API.GetAccessRule`

- GetAccessRule GETs users/{user}/access_rules/{id}, decodes the access_rule envelope and accepts only 200.
  Tests: `identity/v3/applicationcredentials/native_contracts_test.go:TestNativeApplicationCredentialRoutesBodiesPagingAndDecode`, `identity/v3/applicationcredentials/native_contracts_test.go:TestNativeApplicationCredentialStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/applicationcredentials call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/applicationcredentials.DeleteAccessRule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/applicationcredentials.API.DeleteAccessRule`

- DeleteAccessRule DELETEs users/{user}/access_rules/{id} and accepts only 202/204.
  Tests: `identity/v3/applicationcredentials/native_contracts_test.go:TestNativeApplicationCredentialRoutesBodiesPagingAndDecode`, `identity/v3/applicationcredentials/native_contracts_test.go:TestNativeApplicationCredentialStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/applicationcredentials call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/credentials.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials.WithCreateField`

- Create POSTs {"credential":{...}} to credentials with the JSON blob string, requires blob, type and user_id and rejects a core-key collision before HTTP, and accepts only 201.
  Tests: `identity/v3/credentials/native_contracts_test.go:TestNativeCredentialRoutesBodiesPagingAndDecode`, `identity/v3/credentials/native_contracts_test.go:TestNativeCredentialStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/credentials call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/credentials.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials.API.Get`

- Get GETs credentials/{id}, returns nil without error for a missing envelope and accepts only 200.
  Tests: `identity/v3/credentials/native_contracts_test.go:TestNativeCredentialRoutesBodiesPagingAndDecode`, `identity/v3/credentials/native_contracts_test.go:TestNativeCredentialStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/credentials call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/credentials.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials.WithListQuery`

- List GETs credentials with user_id/type filters and extension query values, follows the Keystone links.next string until null, and rejects a nil option before HTTP.
  Tests: `identity/v3/credentials/native_contracts_test.go:TestNativeCredentialRoutesBodiesPagingAndDecode`, `identity/v3/credentials/native_contracts_test.go:TestNativeCredentialStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/credentials call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/credentials.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials.WithUpdateField`

- Update PATCHes {"credential":{...}} to credentials/{id} omitting empty fields with extensions inside the envelope, rejects a nil option before HTTP and accepts only 200.
  Tests: `identity/v3/credentials/native_contracts_test.go:TestNativeCredentialRoutesBodiesPagingAndDecode`, `identity/v3/credentials/native_contracts_test.go:TestNativeCredentialStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/credentials call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/credentials.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials.API.Delete`

- Delete DELETEs credentials/{id} and accepts only 202/204.
  Tests: `identity/v3/credentials/native_contracts_test.go:TestNativeCredentialRoutesBodiesPagingAndDecode`, `identity/v3/credentials/native_contracts_test.go:TestNativeCredentialStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/credentials call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/ec2credentials.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/ec2credentials.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/ec2credentials.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/ec2credentials.WithCreateField`

- Create POSTs an envelope-free {"tenant_id":...} body with top-level extensions to users/{user}/credentials/OS-EC2, requires tenant_id and rejects a tenant_id extension before HTTP, decodes the credential access/secret and accepts only 201.
  Tests: `identity/v3/ec2credentials/native_contracts_test.go:TestNativeEC2CredentialRoutesBodiesPagingAndDecode`, `identity/v3/ec2credentials/native_contracts_test.go:TestNativeEC2CredentialStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/ec2credentials call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/ec2credentials.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/ec2credentials.API.Get`

- Get GETs users/{user}/credentials/OS-EC2/{access}, decodes the credential envelope with a null trust_id as empty and accepts only 200.
  Tests: `identity/v3/ec2credentials/native_contracts_test.go:TestNativeEC2CredentialRoutesBodiesPagingAndDecode`, `identity/v3/ec2credentials/native_contracts_test.go:TestNativeEC2CredentialStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/ec2credentials call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/ec2credentials.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/ec2credentials.API.List`

- List GETs users/{user}/credentials/OS-EC2 and follows the Keystone links.next string until null.
  Tests: `identity/v3/ec2credentials/native_contracts_test.go:TestNativeEC2CredentialRoutesBodiesPagingAndDecode`, `identity/v3/ec2credentials/native_contracts_test.go:TestNativeEC2CredentialStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/ec2credentials call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/ec2credentials.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/ec2credentials.API.Delete`

- Delete DELETEs users/{user}/credentials/OS-EC2/{access} and accepts only 202/204.
  Tests: `identity/v3/ec2credentials/native_contracts_test.go:TestNativeEC2CredentialRoutesBodiesPagingAndDecode`, `identity/v3/ec2credentials/native_contracts_test.go:TestNativeEC2CredentialStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/ec2credentials call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

## References

- [applicationcredentials](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/applicationcredentials/requests.go)
- [credentials](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/credentials/requests.go)
- [ec2credentials](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/ec2credentials/requests.go)
