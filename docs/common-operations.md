# Common service operations

Shared helpers exposed by `common/extensions`, `utils` and the Designate quotas package wrap Gophercloud v2.15.0 and do not add Resource-level behavior.

| Go API | Behavior |
|---|---|
| `extensions.API.Get` | `GET /extensions/{alias}`; decodes the extension envelope. A 404 stays an unexpected-response-code error carrying the status; use `errors.As` to read it. |
| `extensions.API.List` | `GET /extensions`; one page streamed as an iterator in response order; accepts no query parameters. |
| `extensions.API.ExtensionURL`, `ListExtensionURL` | Build `{endpoint}/extensions/{alias}` and `{endpoint}/extensions` locally. The context argument is accepted for signature uniformity; no request is made. |
| `quotas.API.URL` (Designate) | Builds `{dns endpoint}/quotas/{project}` locally without a request. |
| `utils.API.ChooseVersion` | Picks the highest-priority recognized version with a current, supported or stable status from the discovery document at the provider identity base (experimental versions are skipped) and returns its self link. An identity endpoint already ending in a recognized suffix is returned without a discovery request. No usable version returns a no-supported-version error. |
| `utils.API.GetServiceVersions` | Answers from the URL alone for a versioned endpoint when discovery is not requested; with discovery, returns normalized versions with status and microversion bounds, highest first. |
| `utils.API.GetSupportedMicroversions` | Returns the minimum and maximum microversion of a single versioned endpoint discovered from the service endpoint; an endpoint without microversion fields, or an unversioned endpoint listing several versions, is an error. |

The `utils` methods read the identity endpoints and HTTP client from the service client's provider client, where the native functions take the provider client directly. Errors are wrapped with the operation and service name and keep their causes reachable through `errors.Is` and `errors.As`.
