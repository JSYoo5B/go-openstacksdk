# Python Parity

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/get_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Get`

- Get sends GET servers/{id} with the client microversion and decodes the server fields Python exposes.
  Tests: `compute/v2/servers/python_parity_test.go:TestPythonServerGetUpdateDeleteMatchProxyRequests`.

Python/Go differences:

- Python accepts a Server resource or ID; Go takes the server ID string.
- Python returns a mutable Server Resource with every response attribute; Go returns the Gophercloud servers.Server model, which drops fields it does not declare.
- Python sends min(resource _max_microversion 2.100, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/update_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithUpdateField`

- Update sends PUT servers/{id} with only the given name plus a WithUpdateField description inside the server envelope.
  Tests: `compute/v2/servers/python_parity_test.go:TestPythonServerGetUpdateDeleteMatchProxyRequests`.

Python/Go differences:

- UpdateOpts carries name, access IPs and hostname; other attributes such as description go through WithUpdateField.
- UpdateOpts omits empty strings, so an explicit empty value Python would send cannot be expressed for those typed keys.
- Python skips the request when no attribute changed; Go always sends the PUT.
- Python sends min(resource _max_microversion 2.100, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/delete_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ForceDelete`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove sends DELETE servers/{id}, ignores 404 by default and reports NotFound with WithMissingError; ForceDelete posts the forceDelete action.
  Tests: `compute/v2/servers/python_parity_test.go:TestPythonServerGetUpdateDeleteMatchProxyRequests`.

Python/Go differences:

- Python ignore_missing=True maps to Remove(ctx, resource.ID(id)) and ignore_missing=False to resource.WithMissingError(); the plain generated Delete returns the 404 error.
- force=True sends {"forceDelete": null} in Python and {"forceDelete": ""} in Go; neither applies ignore_missing on that path.
- Python sends min(resource _max_microversion 2.100, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/rebuild_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Rebuild`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithRebuildField`

- Rebuild sends only imageRef plus the given typed keys, and WithRebuildField adds preserve_ephemeral, user_data, hostname and explicit null key_name or description inside the rebuild action; the response server is returned.
  Tests: `compute/v2/servers/python_parity_test.go:TestPythonServerRebuildSendsPassedKeysAndReturnsServer`.

Python/Go differences:

- Python accepts a Server and an Image resource or IDs; Go takes the server ID and RebuildOpts.ImageRef.
- name, adminPass, accessIPv4, accessIPv6 and metadata are typed RebuildOpts keys that omit empty values and cannot be overridden by WithRebuildField, so Go cannot send null or empty strings for them; Nova does not accept null for those keys.
- RebuildOpts.Metadata holds string values only.
- Python updates the mutable Server from the response; Go returns a new servers.Server model.
- Python sends min(resource _max_microversion 2.100, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.
- Python _action sends an empty Accept header and accepts any status below 400; Go keeps Gophercloud's Accept header and the native success statuses of the action (201 or 202 unless noted).

### `python:compute/v2/change_server_password`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ChangeAdminPassword`

- ChangeAdminPassword posts {"changePassword": {"adminPass": ...}} to servers/{id}/action exactly as the Python Server action.
  Tests: `compute/v2/servers/python_parity_test.go:TestPythonServerActionsMatchProxyBodies`.

Python/Go differences:

- Python accepts a Server resource or ID and returns None; Go takes the server ID string and returns only an error.
- Python sends min(resource _max_microversion 2.100, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.
- Python _action sends an empty Accept header and accepts any status below 400; Go keeps Gophercloud's Accept header and the native success statuses of the action (201 or 202 unless noted).

### `python:compute/v2/reboot_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Reboot`

- Reboot posts {"reboot": {"type": ...}} to servers/{id}/action exactly as the Python Server action.
  Tests: `compute/v2/servers/python_parity_test.go:TestPythonServerActionsMatchProxyBodies`.

Python/Go differences:

- Python takes reboot_type as a plain string; Go takes RebootOpts{Type: servers.RebootMethod(...)}, which is required before HTTP.
- Python accepts a Server resource or ID and returns None; Go takes the server ID string and returns only an error.
- Python sends min(resource _max_microversion 2.100, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.
- Python _action sends an empty Accept header and accepts any status below 400; Go keeps Gophercloud's Accept header and the native success statuses of the action (201 or 202 unless noted).

### `python:compute/v2/resize_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Resize`

- Resize posts {"resize": {"flavorRef": ...}} to servers/{id}/action exactly as the Python Server action.
  Tests: `compute/v2/servers/python_parity_test.go:TestPythonServerActionsMatchProxyBodies`.

Python/Go differences:

- Python accepts a Flavor resource or ID; Go takes the flavor ID in ResizeOpts.FlavorRef.
- Python accepts a Server resource or ID and returns None; Go takes the server ID string and returns only an error.
- Python sends min(resource _max_microversion 2.100, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.
- Python _action sends an empty Accept header and accepts any status below 400; Go keeps Gophercloud's Accept header and the native success statuses of the action (201 or 202 unless noted).

### `python:compute/v2/confirm_server_resize`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ConfirmResize`

- ConfirmResize posts {"confirmResize": null} to servers/{id}/action exactly as the Python Server action.
  Tests: `compute/v2/servers/python_parity_test.go:TestPythonServerActionsMatchProxyBodies`.

Python/Go differences:

- Go ConfirmResize also accepts 204 from the action.
- Python accepts a Server resource or ID and returns None; Go takes the server ID string and returns only an error.
- Python sends min(resource _max_microversion 2.100, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.
- Python _action sends an empty Accept header and accepts any status below 400; Go keeps Gophercloud's Accept header and the native success statuses of the action (201 or 202 unless noted).

### `python:compute/v2/revert_server_resize`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.RevertResize`

- RevertResize posts {"revertResize": null} to servers/{id}/action exactly as the Python Server action.
  Tests: `compute/v2/servers/python_parity_test.go:TestPythonServerActionsMatchProxyBodies`.

Python/Go differences:

- Python accepts a Server resource or ID and returns None; Go takes the server ID string and returns only an error.
- Python sends min(resource _max_microversion 2.100, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.
- Python _action sends an empty Accept header and accepts any status below 400; Go keeps Gophercloud's Accept header and the native success statuses of the action (201 or 202 unless noted).

### `python:compute/v2/pause_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Pause`

- Pause posts {"pause": null} to servers/{id}/action exactly as the Python Server action.
  Tests: `compute/v2/servers/python_parity_test.go:TestPythonServerActionsMatchProxyBodies`.

Python/Go differences:

- Python accepts a Server resource or ID and returns None; Go takes the server ID string and returns only an error.
- Python sends min(resource _max_microversion 2.100, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.
- Python _action sends an empty Accept header and accepts any status below 400; Go keeps Gophercloud's Accept header and the native success statuses of the action (201 or 202 unless noted).

### `python:compute/v2/unpause_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Unpause`

- Unpause posts {"unpause": null} to servers/{id}/action exactly as the Python Server action.
  Tests: `compute/v2/servers/python_parity_test.go:TestPythonServerActionsMatchProxyBodies`.

Python/Go differences:

- Python accepts a Server resource or ID and returns None; Go takes the server ID string and returns only an error.
- Python sends min(resource _max_microversion 2.100, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.
- Python _action sends an empty Accept header and accepts any status below 400; Go keeps Gophercloud's Accept header and the native success statuses of the action (201 or 202 unless noted).

### `python:compute/v2/suspend_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Suspend`

- Suspend posts {"suspend": null} to servers/{id}/action exactly as the Python Server action.
  Tests: `compute/v2/servers/python_parity_test.go:TestPythonServerActionsMatchProxyBodies`.

Python/Go differences:

- Python accepts a Server resource or ID and returns None; Go takes the server ID string and returns only an error.
- Python sends min(resource _max_microversion 2.100, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.
- Python _action sends an empty Accept header and accepts any status below 400; Go keeps Gophercloud's Accept header and the native success statuses of the action (201 or 202 unless noted).

### `python:compute/v2/resume_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Resume`

- Resume posts {"resume": null} to servers/{id}/action exactly as the Python Server action.
  Tests: `compute/v2/servers/python_parity_test.go:TestPythonServerActionsMatchProxyBodies`.

Python/Go differences:

- Python accepts a Server resource or ID and returns None; Go takes the server ID string and returns only an error.
- Python sends min(resource _max_microversion 2.100, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.
- Python _action sends an empty Accept header and accepts any status below 400; Go keeps Gophercloud's Accept header and the native success statuses of the action (201 or 202 unless noted).

### `python:compute/v2/unlock_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Unlock`

- Unlock posts {"unlock": null} to servers/{id}/action exactly as the Python Server action.
  Tests: `compute/v2/servers/python_parity_test.go:TestPythonServerActionsMatchProxyBodies`.

Python/Go differences:

- Python accepts a Server resource or ID and returns None; Go takes the server ID string and returns only an error.
- Python sends min(resource _max_microversion 2.100, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.
- Python _action sends an empty Accept header and accepts any status below 400; Go keeps Gophercloud's Accept header and the native success statuses of the action (201 or 202 unless noted).

### `python:compute/v2/unrescue_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Unrescue`

- Unrescue posts {"unrescue": null} to servers/{id}/action exactly as the Python Server action.
  Tests: `compute/v2/servers/python_parity_test.go:TestPythonServerActionsMatchProxyBodies`.

Python/Go differences:

- Python accepts a Server resource or ID and returns None; Go takes the server ID string and returns only an error.
- Python sends min(resource _max_microversion 2.100, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.
- Python _action sends an empty Accept header and accepts any status below 400; Go keeps Gophercloud's Accept header and the native success statuses of the action (201 or 202 unless noted).

### `python:compute/v2/start_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Start`

- Start posts {"os-start": null} to servers/{id}/action exactly as the Python Server action.
  Tests: `compute/v2/servers/python_parity_test.go:TestPythonServerActionsMatchProxyBodies`.

Python/Go differences:

- Python accepts a Server resource or ID and returns None; Go takes the server ID string and returns only an error.
- Python sends min(resource _max_microversion 2.100, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.
- Python _action sends an empty Accept header and accepts any status below 400; Go keeps Gophercloud's Accept header and the native success statuses of the action (201 or 202 unless noted).

### `python:compute/v2/stop_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Stop`

- Stop posts {"os-stop": null} to servers/{id}/action exactly as the Python Server action.
  Tests: `compute/v2/servers/python_parity_test.go:TestPythonServerActionsMatchProxyBodies`.

Python/Go differences:

- Python accepts a Server resource or ID and returns None; Go takes the server ID string and returns only an error.
- Python sends min(resource _max_microversion 2.100, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.
- Python _action sends an empty Accept header and accepts any status below 400; Go keeps Gophercloud's Accept header and the native success statuses of the action (201 or 202 unless noted).

### `python:compute/v2/shelve_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Shelve`

- Shelve posts {"shelve": null} to servers/{id}/action exactly as the Python Server action.
  Tests: `compute/v2/servers/python_parity_test.go:TestPythonServerActionsMatchProxyBodies`.

Python/Go differences:

- Python accepts a Server resource or ID and returns None; Go takes the server ID string and returns only an error.
- Python sends min(resource _max_microversion 2.100, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.
- Python _action sends an empty Accept header and accepts any status below 400; Go keeps Gophercloud's Accept header and the native success statuses of the action (201 or 202 unless noted).

### `python:compute/v2/shelve_offload_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ShelveOffload`

- ShelveOffload posts {"shelveOffload": null} to servers/{id}/action exactly as the Python Server action.
  Tests: `compute/v2/servers/python_parity_test.go:TestPythonServerActionsMatchProxyBodies`.

Python/Go differences:

- Python accepts a Server resource or ID and returns None; Go takes the server ID string and returns only an error.
- Python sends min(resource _max_microversion 2.100, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.
- Python _action sends an empty Accept header and accepts any status below 400; Go keeps Gophercloud's Accept header and the native success statuses of the action (201 or 202 unless noted).

### `python:compute/v2/rescue_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Rescue`

- Rescue sends {"rescue": {}} by default and adds adminPass and rescue_image_ref only when given.
  Tests: `compute/v2/servers/python_parity_test.go:TestPythonServerActionsMatchProxyBodies`.

Python/Go differences:

- Python accepts an Image resource or ID for image; Go takes the image ID in RescueOpts.RescueImageRef.
- Python returns None; Go returns the adminPass from the response and accepts only status 200.
- Python accepts a Server resource or ID; Go takes the server ID string.
- Python sends min(resource _max_microversion 2.100, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/create_server_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.CreateImage`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.API.WaitFor`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithTimeout`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPollInterval`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithFailureStates`

- CreateImage posts createImage and reads image_id from the 2.45 body or the Location header below 2.45; FindIdentity then GETs the Glance image and WaitFor polls it to active, failing on error and on timeout.
  Tests: `compute/v2/servers/python_parity_test.go:TestPythonServerCreateImageFetchesAndWaitsForGlanceImage`.

Python/Go differences:

- Python pins microversion 2.45 for createImage when the server supports it; Go sends the client microversion and picks the body or Location header from the response version header.
- The Go caller chains CreateImage, Images.FindIdentity and, for wait=True, Images.WaitFor(ctx, resource.ID(id), "active", resource.WithTimeout(120*time.Second), resource.WithFailureStates("error")).
- Python keeps waiting while the image lookup returns None; Go WaitFor returns the 404 error at once.
- Python raises ResourceTimeout or SDKException; Go returns a context deadline error or resource.FailedStateError.
- CreateImageOpts.Metadata holds string values and omits an empty map.

### `python:compute/v2/set_server_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.UpdateMetadata`

- UpdateMetadata posts {"metadata": {...}} to servers/{id}/metadata and returns the merged response map.
  Tests: `compute/v2/servers/python_parity_test.go:TestPythonServerMetadataSetAndDeleteMatchMixinRequests`.

Python/Go differences:

- Python returns the Server with metadata set to the passed keys only; Go returns the server's full merged metadata from the response.
- MetadataOpts holds string values; Python passes arbitrary values through.
- Python sends this request without an explicit microversion (session default); Go always sends the microversion configured or negotiated on the Connection client.

### `python:compute/v2/delete_server_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.DeleteMetadatum`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ResetMetadata`

- DeleteMetadatum sends DELETE servers/{id}/metadata/{key}, and ResetMetadata with an empty MetadataOpts sends PUT {"metadata": {}} like delete_server_metadata without keys.
  Tests: `compute/v2/servers/python_parity_test.go:TestPythonServerMetadataSetAndDeleteMatchMixinRequests`.

Python/Go differences:

- The Go caller loops DeleteMetadatum over the keys; Python deduplicates them in set order.
- ResetMetadata with a nil MetadataOpts sends "metadata": null, so the caller passes an empty map.
- Python sends this request without an explicit microversion (session default); Go always sends the microversion configured or negotiated on the Connection client.

### `python:compute/v2/server_ips`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ListAddresses`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ListAddressesByNetwork`

- ListAddresses sends GET servers/{id}/ips and ListAddressesByNetwork sends GET servers/{id}/ips/{label}, both returning the addr and version Python exposes.
  Tests: `compute/v2/servers/python_parity_test.go:TestPythonServerIPsListAllAndByNetworkLabel`.

Python/Go differences:

- Python yields one ServerIP per address carrying network_label; Go ListAddresses yields one label map and ListAddressesByNetwork yields Address rows without the label.
- Python does not check the HTTP status, so an error body surfaces as KeyError or TypeError; Go returns the Gophercloud HTTP error.
- Python sends this request without an explicit microversion (session default); Go always sends the microversion configured or negotiated on the Connection client.

### `python:compute/v2/add_tag_to_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/tags.API.InServer`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/tags.ServerTagScope.Add`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`

- ServerTagScope.Add sends PUT servers/{id}/tags/{tag} without a body.
  Tests: `compute/v2/tags/python_parity_test.go:TestPythonServerTagAddRemoveAndRemoveAllMatchTagMixin`.
- InServer refuses a configured microversion below 2.26 before HTTP, and Add path-escapes the tag and rejects slash or comma.
  Tests: `compute/v2/tags/python_parity_test.go:TestPythonServerTagNeedsConfiguredMicroversion`.

Python/Go differences:

- Python discovers a microversion of at least 2.26; Go requires the configured client microversion to be 2.26 or newer.
- Python joins the tag into the URL unescaped; Go escapes it and validates length, slash and comma before HTTP.
- Python appends the tag to the Server resource; Go returns only an error.

### `python:compute/v2/remove_tag_from_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/tags.API.InServer`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/tags.ServerTagScope.Remove`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/tags.WithMissingError`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`

- ServerTagScope.Remove sends DELETE servers/{id}/tags/{tag}; with WithMissingError a 404 is resource.ErrNotFound as Python raises, and without it the 404 is ignored.
  Tests: `compute/v2/tags/python_parity_test.go:TestPythonServerTagAddRemoveAndRemoveAllMatchTagMixin`.

Python/Go differences:

- Python raises on 404; Go ignores 404 by default, so the Python behavior needs tags.WithMissingError().
- Python discovers a microversion of at least 2.26; Go requires the configured client microversion to be 2.26 or newer.
- Python joins the tag into the URL unescaped; Go escapes and validates it.

### `python:compute/v2/remove_tags_from_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/tags.API.InServer`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/tags.ServerTagScope.RemoveAll`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/tags.WithMissingError`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`

- ServerTagScope.RemoveAll sends DELETE servers/{id}/tags and reports a 404 as resource.ErrNotFound with WithMissingError.
  Tests: `compute/v2/tags/python_parity_test.go:TestPythonServerTagAddRemoveAndRemoveAllMatchTagMixin`.

Python/Go differences:

- Python raises on 404; Go ignores 404 by default, so the Python behavior needs tags.WithMissingError().
- Python discovers a microversion of at least 2.26; Go requires the configured client microversion to be 2.26 or newer.

### `python:compute/v2/fetch_server_security_groups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/secgroups.API.ListByServer`

- ListByServer sends GET servers/{id}/os-security-groups and yields the security groups with their names, descriptions and tenant IDs.
  Tests: `compute/v2/secgroups/python_parity_test.go:TestPythonSecGroupServerFetchAddRemoveMatchProxyRequests`.

Python/Go differences:

- Python returns the Server with security_groups set to the raw response dicts; Go yields typed secgroups.SecurityGroup values.
- Python sends this request without an explicit microversion (session default); Go always sends the microversion configured or negotiated on the Connection client.

### `python:compute/v2/add_security_group_to_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/secgroups.API.AddServer`

- AddServer posts {"addSecurityGroup": {"name": ...}} with the given string and no lookup.
  Tests: `compute/v2/secgroups/python_parity_test.go:TestPythonSecGroupServerFetchAddRemoveMatchProxyRequests`.

Python/Go differences:

- Python accepts a SecurityGroup resource and sends its name, or its ID when the name is unset; Go takes that string directly.
- Python accepts a Server resource or ID and returns None; Go takes the server ID string and returns only an error.
- Python sends min(resource _max_microversion 2.100, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.
- Python _action sends an empty Accept header and accepts any status below 400; Go keeps Gophercloud's Accept header and the native success statuses of the action (201 or 202 unless noted).

### `python:compute/v2/remove_security_group_from_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/secgroups.API.RemoveServer`

- RemoveServer posts {"removeSecurityGroup": {"name": ...}} with the given string and no lookup.
  Tests: `compute/v2/secgroups/python_parity_test.go:TestPythonSecGroupServerFetchAddRemoveMatchProxyRequests`.

Python/Go differences:

- Python accepts a SecurityGroup resource and sends its name, or its ID when the name is unset; Go takes that string directly.
- Python accepts a Server resource or ID and returns None; Go takes the server ID string and returns only an error.
- Python sends min(resource _max_microversion 2.100, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.
- Python _action sends an empty Accept header and accepts any status below 400; Go keeps Gophercloud's Accept header and the native success statuses of the action (201 or 202 unless noted).

### `python:compute/v2/create_server_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.WithCreateField`

- Create posts the server_group envelope with policy and rules or with policies, and decodes policy, rules and members.
  Tests: `compute/v2/servergroups/python_parity_test.go:TestPythonServerGroupCreateGetDeleteMatchProxyRequests`.

Python/Go differences:

- Python rewrites policies to policy at microversion 2.64 or later and policy to policies below it, and rejects rules below 2.64; the Go caller chooses Policy and Rules or Policies for the configured microversion.
- Keys outside CreateOpts go through WithCreateField.
- Python sends min(resource _max_microversion 2.64, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/get_server_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.API.Get`

- Get sends GET os-server-groups/{id} and decodes name, project and user.
  Tests: `compute/v2/servergroups/python_parity_test.go:TestPythonServerGroupCreateGetDeleteMatchProxyRequests`.

Python/Go differences:

- Python accepts a ServerGroup resource or ID; Go takes the ID string and returns the Gophercloud ServerGroup model.
- Python sends min(resource _max_microversion 2.64, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/delete_server_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove sends DELETE os-server-groups/{id}, ignores 404 by default and reports NotFound with WithMissingError.
  Tests: `compute/v2/servergroups/python_parity_test.go:TestPythonServerGroupCreateGetDeleteMatchProxyRequests`.

Python/Go differences:

- Python ignore_missing=True maps to Remove(ctx, resource.ID(id)) and ignore_missing=False to resource.WithMissingError(); the plain generated Delete returns the 404 error.
- Python sends min(resource _max_microversion 2.64, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/server_groups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.API.List`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.WithListQuery`

- List sends GET os-server-groups without a query by default and carries all_projects=True, limit and marker through WithListQuery and ListOpts.
  Tests: `compute/v2/servergroups/python_parity_test.go:TestPythonServerGroupListAllProjectsAndQuery`.

Python/Go differences:

- Typed ListOpts.AllProjects sends all_projects=true; WithListQuery("all_projects", "True") reproduces Python's spelling.
- Go reads one page; Python may request a further page with a marker after a page that fills limit.
- Python sends min(resource _max_microversion 2.64, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/create_server_interface`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/attachinterfaces.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/attachinterfaces.WithCreateField`

- Create posts the interfaceAttachment envelope with port_id or net_id and fixed_ips, plus a WithCreateField tag, and decodes the interface.
  Tests: `compute/v2/attachinterfaces/python_parity_test.go:TestPythonServerInterfaceCreateGetListDeleteMatchProxyRequests`.

Python/Go differences:

- tag is not a CreateOpts field and goes through WithCreateField.
- The Go Interface model has no tag field, so the 2.70 tag in the response is dropped.
- Python accepts a Server resource or ID; Go takes the server ID string.
- Python sends min(resource _max_microversion 2.70, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/get_server_interface`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/attachinterfaces.API.Get`

- Get sends GET servers/{id}/os-interface/{port} and decodes port, network, MAC, state and fixed IPs.
  Tests: `compute/v2/attachinterfaces/python_parity_test.go:TestPythonServerInterfaceCreateGetListDeleteMatchProxyRequests`.

Python/Go differences:

- Python can take the server from the ServerInterface resource's server_id; Go takes both IDs explicitly.
- The Go Interface model has no tag field, so the 2.70 tag in the response is dropped.
- Python sends min(resource _max_microversion 2.70, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/server_interfaces`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/attachinterfaces.API.List`

- List sends one GET servers/{id}/os-interface and yields every interface.
  Tests: `compute/v2/attachinterfaces/python_parity_test.go:TestPythonServerInterfaceCreateGetListDeleteMatchProxyRequests`.

Python/Go differences:

- Python accepts limit and marker query parameters that Nova's interface list does not use; Go List takes no query.
- The Go Interface model has no tag field, so the 2.70 tag in the response is dropped.
- Python sends min(resource _max_microversion 2.70, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/delete_server_interface`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/attachinterfaces.API.InServer`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- The InServer scope's Delete sends DELETE servers/{id}/os-interface/{port}, ignores 404 by default and reports NotFound with WithMissingError.
  Tests: `compute/v2/attachinterfaces/python_parity_test.go:TestPythonServerInterfaceCreateGetListDeleteMatchProxyRequests`.

Python/Go differences:

- Python can take the server from the ServerInterface resource's server_id; Go needs InServer(ctx, resource.ID(serverID)) first.
- Python ignore_missing=True maps to the scope's Delete(ctx, resource.ID(port)); the plain generated API.Delete returns the 404 error.
- Python sends min(resource _max_microversion 2.70, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/create_volume_attachment`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/volumeattach.API.Create`

- Create posts the volumeAttachment envelope with volumeId, device, tag and delete_on_termination and decodes the attachment.
  Tests: `compute/v2/volumeattach/python_parity_test.go:TestPythonVolumeAttachmentCreateGetListDeleteMatchProxyRequests`.

Python/Go differences:

- Python's legacy volume_id and volumeId keyword arguments map to CreateOpts.VolumeID.
- CreateOpts omits delete_on_termination=false, which is also the server default.
- The Go VolumeAttachment model has no attachment_id or bdm_uuid, so those 2.89 fields are dropped.
- Python sends min(resource _max_microversion 2.89, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/get_volume_attachment`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/volumeattach.API.Get`

- Get sends GET servers/{id}/os-volume_attachments/{volume} and decodes device, tag and delete_on_termination.
  Tests: `compute/v2/volumeattach/python_parity_test.go:TestPythonVolumeAttachmentCreateGetListDeleteMatchProxyRequests`.

Python/Go differences:

- Python accepts Server and Volume resources or IDs; Go takes both ID strings.
- The Go VolumeAttachment model has no attachment_id or bdm_uuid, so those 2.89 fields are dropped.
- Python sends min(resource _max_microversion 2.89, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/delete_volume_attachment`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/volumeattach.API.InServer`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- The InServer scope's Delete sends DELETE servers/{id}/os-volume_attachments/{volume}, ignores 404 by default and reports NotFound with WithMissingError.
  Tests: `compute/v2/volumeattach/python_parity_test.go:TestPythonVolumeAttachmentCreateGetListDeleteMatchProxyRequests`.

Python/Go differences:

- With plain ID strings Python first calls find_server(server) to detect swapped arguments, adding GET servers/{id} (and a list fallback when it is missing) and swapping the arguments if no server is found; Go sends only the DELETE.
- Python ignore_missing=True maps to the scope's Delete(ctx, resource.ID(volume)); the plain generated API.Delete returns the 404 error.
- Python sends min(resource _max_microversion 2.89, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/get_server_action`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/instanceactions.API.InServer`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Get`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`

- The InServer scope's Get sends GET servers/{id}/os-instance-actions/{request_id}, decodes the action with its events including 2.84 details, and reports a 404 as resource.ErrNotFound.
  Tests: `compute/v2/instanceactions/python_parity_test.go:TestPythonServerActionGetReturnsDetailWithEvents`.

Python/Go differences:

- Python's ignore_missing argument is not applied by _get, so both sides raise or return an error on 404.
- Python can take the server from the ServerAction resource's server_id; Go needs InServer(ctx, resource.ID(serverID)) first.
- Go returns instanceactions.ActionResource with typed fields, the raw body and response headers instead of a mutable ServerAction.
- Python sends min(resource _max_microversion 2.84, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/get_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/images.API.GetRecord`

- GET /images/{escaped literal ID}; singular image unpack or root JSON object. Missing id retains requested ID; blank or malformed JSON syntax preserves Source seed; valid nonobject representation is an accepted-response error with HTTP proof.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageGetProjectionAliasesAndStatuses`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`, `compute/v2/images/image_records_test.go:TestPythonNovaImageRejectedResponseFailuresStopDeleteAndFind`, `compute/v2/images/image_records_test.go:TestPythonNovaImageCleanRejectionsRemainEligibleAfterNativeRetryHook`.
- Image declared Body view preserves untyped JSON and exact integers; response alias dictionary order, four Python integer descriptors and dictionary metadata descriptor project into 12 fields, including null location. Wire/Envelope/Header/Status preserve separately owned actual evidence.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageGetProjectionAliasesAndStatuses`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`, `compute/v2/images/image_records_test.go:TestPythonNovaImageRejectedResponseFailuresStopDeleteAndFind`, `compute/v2/images/image_records_test.go:TestPythonNovaImageCleanRejectionsRemainEligibleAfterNativeRetryHook`.
- GET400/403/404 and Delete404 observe rejected Read/Close/context/source faults through shared REST; only clean rejection may establish ignoreMissing/fallback. Identical native retry stop-hook return remains a clean status; mixed retry errors preserve causes and stop after one physical request.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageGetProjectionAliasesAndStatuses`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`, `compute/v2/images/image_records_test.go:TestPythonNovaImageRejectedResponseFailuresStopDeleteAndFind`, `compute/v2/images/image_records_test.go:TestPythonNovaImageCleanRejectionsRemainEligibleAfterNativeRetryHook`.

Python/Go differences:

- Explicit legacy Nova selected-version profile, not unrestricted modern Nova coverage: ordinary image methods accept 2.1..2.35; metadata methods accept 2.1..2.38. Empty selection means Nova default2.1; selected version is never downgraded or discovered. Pinned Image has inherited _max_microversion=None and Source default selection is not capped; Go preflight rejects unsupported selections before HTTP.
- Literal valid UTF-8 string IDs/keys, immutable owned records and null location replace Python mutable same-instance Resource/Munch/cache/Connection/session/subclass/warnings. Full Resource/session parity is a separate surface outside these callable Go profiles.
- Library guards protect selected compute source facts and headers; typed nonnegative list controls and fixed continuation origin/path/query are Go profile choices. Arbitrary Python session/path/version hooks and JMESPath surface are not provided.

### `python:compute/v2/find_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/images.API.FindRecord`

- Literal member GET first; only clean 400/403/404 allow fallback GET /images?name=... (summary, not detail). Complete exact ID/name search includes later pages; duplicate/missing/late errors retain shared FindIdentity policy.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageFindSummaryAndDeleteMissing`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`, `compute/v2/images/image_records_test.go:TestPythonNovaImageRejectedResponseFailuresStopDeleteAndFind`, `compute/v2/images/image_records_test.go:TestPythonNovaImageCleanRejectionsRemainEligibleAfterNativeRetryHook`.
- IgnoreMissing defaults true; false returns NotFound; successful member GET avoids list. Source find_image formal options have no details/query, which is reflected in concrete FindOpts.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageFindSummaryAndDeleteMissing`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`, `compute/v2/images/image_records_test.go:TestPythonNovaImageRejectedResponseFailuresStopDeleteAndFind`, `compute/v2/images/image_records_test.go:TestPythonNovaImageCleanRejectionsRemainEligibleAfterNativeRetryHook`.
- GET400/403/404 and Delete404 observe rejected Read/Close/context/source faults through shared REST; only clean rejection may establish ignoreMissing/fallback. Identical native retry stop-hook return remains a clean status; mixed retry errors preserve causes and stop after one physical request.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageFindSummaryAndDeleteMissing`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`, `compute/v2/images/image_records_test.go:TestPythonNovaImageRejectedResponseFailuresStopDeleteAndFind`, `compute/v2/images/image_records_test.go:TestPythonNovaImageCleanRejectionsRemainEligibleAfterNativeRetryHook`.

Python/Go differences:

- Explicit legacy Nova selected-version profile, not unrestricted modern Nova coverage: ordinary image methods accept 2.1..2.35; metadata methods accept 2.1..2.38. Empty selection means Nova default2.1; selected version is never downgraded or discovered. Pinned Image has inherited _max_microversion=None and Source default selection is not capped; Go preflight rejects unsupported selections before HTTP.
- Literal valid UTF-8 string IDs/keys, immutable owned records and null location replace Python mutable same-instance Resource/Munch/cache/Connection/session/subclass/warnings. Full Resource/session parity is a separate surface outside these callable Go profiles.
- Library guards protect selected compute source facts and headers; typed nonnegative list controls and fixed continuation origin/path/query are Go profile choices. Arbitrary Python session/path/version hooks and JMESPath surface are not provided.
- Corrects stale remaining text: pinned find_image fallback is /images, not /images/detail.

### `python:compute/v2/images`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/images.API.ListRecords`

- Lazy reusable GET /images/detail by default and /images when Details=false; server/name/status/type/min_disk/min_ram/changes_since map to exact Source wire query, with client alias precedence. Shared Requests JSON encoder handles null/repeated arrays/object keys/True spelling; other declared Body attrs filter locally; unknown kwargs ignored.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageListQueryDetailLocalFiltersAndCaps`, `compute/v2/images/image_records_test.go:TestPythonNovaImageListPagingAndWireMarker`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`.
- Raw-row MaxItems precedes local filtering and supplies absent/zero limit hint; typed limit/marker/paginated controls, plural singleton, array/top-level/HTTP next, short-page marker fallback with initial positive limit, early break and fixed route proof use shared REST. Fallback marker comes from original raw row independent of consumer mutations.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageListQueryDetailLocalFiltersAndCaps`, `compute/v2/images/image_records_test.go:TestPythonNovaImageListPagingAndWireMarker`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`.

Python/Go differences:

- Explicit legacy Nova selected-version profile, not unrestricted modern Nova coverage: ordinary image methods accept 2.1..2.35; metadata methods accept 2.1..2.38. Empty selection means Nova default2.1; selected version is never downgraded or discovered. Pinned Image has inherited _max_microversion=None and Source default selection is not capped; Go preflight rejects unsupported selections before HTTP.
- Literal valid UTF-8 string IDs/keys, immutable owned records and null location replace Python mutable same-instance Resource/Munch/cache/Connection/session/subclass/warnings. Full Resource/session parity is a separate surface outside these callable Go profiles.
- Library guards protect selected compute source facts and headers; typed nonnegative list controls and fixed continuation origin/path/query are Go profile choices. Arbitrary Python session/path/version hooks and JMESPath surface are not provided.
- Dictionary links:{next:...} is a Go REST extension; pinned generic Resource does not consume dictionary links.
- Local JSON filter retains exact JSON types and nested subset instead of Python bool/number equality and nonobject nested-filter exception behavior; server-spelling aliases that are not actual Body attrs do not become local filters.

### `python:compute/v2/delete_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/images.API.DeleteRecord`

- DELETE /images/{escaped literal ID}; no JSON body decode. Actual 200..399 accepted and clean404 ignored by default; WithDeleteIgnoreMissing(false) retains original error. Unsupported microversion guard runs before HTTP and ignoreMissing.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageFindSummaryAndDeleteMissing`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`, `compute/v2/images/image_records_test.go:TestPythonNovaImageRejectedResponseFailuresStopDeleteAndFind`, `compute/v2/images/image_records_test.go:TestPythonNovaImageCleanRejectionsRemainEligibleAfterNativeRetryHook`.
- GET400/403/404 and Delete404 observe rejected Read/Close/context/source faults through shared REST; only clean rejection may establish ignoreMissing/fallback. Identical native retry stop-hook return remains a clean status; mixed retry errors preserve causes and stop after one physical request.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageFindSummaryAndDeleteMissing`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`, `compute/v2/images/image_records_test.go:TestPythonNovaImageRejectedResponseFailuresStopDeleteAndFind`, `compute/v2/images/image_records_test.go:TestPythonNovaImageCleanRejectionsRemainEligibleAfterNativeRetryHook`.

Python/Go differences:

- Explicit legacy Nova selected-version profile, not unrestricted modern Nova coverage: ordinary image methods accept 2.1..2.35; metadata methods accept 2.1..2.38. Empty selection means Nova default2.1; selected version is never downgraded or discovered. Pinned Image has inherited _max_microversion=None and Source default selection is not capped; Go preflight rejects unsupported selections before HTTP.
- Literal valid UTF-8 string IDs/keys, immutable owned records and null location replace Python mutable same-instance Resource/Munch/cache/Connection/session/subclass/warnings. Full Resource/session parity is a separate surface outside these callable Go profiles.
- Library guards protect selected compute source facts and headers; typed nonnegative list controls and fixed continuation origin/path/query are Go profile choices. Arbitrary Python session/path/version hooks and JMESPath surface are not provided.

### `python:compute/v2/fetch_image_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/images.API.FetchMetadata`

- GET /images/{id}/metadata, using selected version through2.38; strict JSON object parse (no malformed/empty-body tolerance). Only metadata overlays seed; body id/unknown fields retained in Wire but do not override resource ID. Missing metadata stays null, dictionary descriptor projects nonobject metadata to{}.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageMetadataFetchAliasSetDeleteContracts`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`.
- Metadata view contains the literal image ID plus submitted/received metadata; other declared fields null, without a parent image GET. All successful statuses200..399 accepted and actual response evidence retained independently.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageMetadataFetchAliasSetDeleteContracts`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`.

Python/Go differences:

- Explicit legacy Nova selected-version profile, not unrestricted modern Nova coverage: ordinary image methods accept 2.1..2.35; metadata methods accept 2.1..2.38. Empty selection means Nova default2.1; selected version is never downgraded or discovered. Pinned Image has inherited _max_microversion=None and Source default selection is not capped; Go preflight rejects unsupported selections before HTTP.
- Literal valid UTF-8 string IDs/keys, immutable owned records and null location replace Python mutable same-instance Resource/Munch/cache/Connection/session/subclass/warnings. Full Resource/session parity is a separate surface outside these callable Go profiles.
- Library guards protect selected compute source facts and headers; typed nonnegative list controls and fixed continuation origin/path/query are Go profile choices. Arbitrary Python session/path/version hooks and JMESPath surface are not provided.
- Fetch requires nonnull JSON object rather than reproducing Python array/scalar membership behavior.

### `python:compute/v2/get_image_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/images.API.GetMetadata`

- Pinned deprecated alias of fetch_image_metadata; identical route/body/status/metadata projection and strict JSON parse.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageMetadataFetchAliasSetDeleteContracts`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`.
- Metadata view contains the literal image ID plus submitted/received metadata; other declared fields null, without a parent image GET. All successful statuses200..399 accepted and actual response evidence retained independently.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageMetadataFetchAliasSetDeleteContracts`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`.

Python/Go differences:

- Explicit legacy Nova selected-version profile, not unrestricted modern Nova coverage: ordinary image methods accept 2.1..2.35; metadata methods accept 2.1..2.38. Empty selection means Nova default2.1; selected version is never downgraded or discovered. Pinned Image has inherited _max_microversion=None and Source default selection is not capped; Go preflight rejects unsupported selections before HTTP.
- Literal valid UTF-8 string IDs/keys, immutable owned records and null location replace Python mutable same-instance Resource/Munch/cache/Connection/session/subclass/warnings. Full Resource/session parity is a separate surface outside these callable Go profiles.
- Library guards protect selected compute source facts and headers; typed nonnegative list controls and fixed continuation origin/path/query are Go profile choices. Arbitrary Python session/path/version hooks and JMESPath surface are not provided.
- Source deprecation warnings have documentation-only Go representation; fetch object-domain difference applies equally.

### `python:compute/v2/set_image_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/images.API.SetMetadata`

- POST /images/{id}/metadata with metadata envelope; nil input means{}. JSON snapshot before option callbacks owns nested data and exact numbers; no parent GET or native builder coercion.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageMetadataFetchAliasSetDeleteContracts`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`.
- Source ignores response JSON and returns submitted metadata; Go Resource uses submitted owned map, Wire=nil, and Envelope/Header/Status capture opaque actual reply. Accepted read/close/guard errors preserve real proof.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageMetadataFetchAliasSetDeleteContracts`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`.
- Metadata view contains the literal image ID plus submitted/received metadata; other declared fields null, without a parent image GET. All successful statuses200..399 accepted and actual response evidence retained independently.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageMetadataFetchAliasSetDeleteContracts`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`.

Python/Go differences:

- Explicit legacy Nova selected-version profile, not unrestricted modern Nova coverage: ordinary image methods accept 2.1..2.35; metadata methods accept 2.1..2.38. Empty selection means Nova default2.1; selected version is never downgraded or discovered. Pinned Image has inherited _max_microversion=None and Source default selection is not capped; Go preflight rejects unsupported selections before HTTP.
- Literal valid UTF-8 string IDs/keys, immutable owned records and null location replace Python mutable same-instance Resource/Munch/cache/Connection/session/subclass/warnings. Full Resource/session parity is a separate surface outside these callable Go profiles.
- Library guards protect selected compute source facts and headers; typed nonnegative list controls and fixed continuation origin/path/query are Go profile choices. Arbitrary Python session/path/version hooks and JMESPath surface are not provided.

### `python:compute/v2/delete_image_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/images.API.DeleteMetadata`

- Omitted Keys means PUT {metadata:{}}; explicit empty key slice means zero physical requests. Owned keys are validated before mutations, deduplicated, escaped once and deleted individually; strict404 and first error/cancel/source change stop later keys.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageMetadataFetchAliasSetDeleteContracts`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`.
- Metadata route ceiling2.38 applied before HTTP, source guarded across each physical request, successful opaque bodies never decoded.
  Tests: `compute/v2/images/image_records_test.go:TestPythonNovaImageMetadataFetchAliasSetDeleteContracts`, `compute/v2/images/image_records_test.go:TestPythonNovaImageExplicitVersionBounds`, `compute/v2/images/image_records_test.go:TestPythonNovaImageOwnedOptionsGuardsAndEvidence`.

Python/Go differences:

- Explicit legacy Nova selected-version profile, not unrestricted modern Nova coverage: ordinary image methods accept 2.1..2.35; metadata methods accept 2.1..2.38. Empty selection means Nova default2.1; selected version is never downgraded or discovered. Pinned Image has inherited _max_microversion=None and Source default selection is not capped; Go preflight rejects unsupported selections before HTTP.
- Literal valid UTF-8 string IDs/keys, immutable owned records and null location replace Python mutable same-instance Resource/Munch/cache/Connection/session/subclass/warnings. Full Resource/session parity is a separate surface outside these callable Go profiles.
- Library guards protect selected compute source facts and headers; typed nonnegative list controls and fixed continuation origin/path/query are Go profile choices. Arbitrary Python session/path/version hooks and JMESPath surface are not provided.
- Python set(keys) iteration order is unspecified; Go uses first-seen deterministic order.

## References

- [compute v2 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py)
- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
- [Server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server.py)
- [MetadataMixin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/metadata.py)
- [TagMixin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/tag.py)
- [SDK Support Ledger](../../docs/sdk-support-ledger.md)
- [Python User Actions](servers/python-user-actions.md)
- [Backup Server Python](../backup-server-python.md)
- [Python Attachments Groups](python-attachments-groups.md)
- [Python Records](instanceactions/python-records.md)
- [Nova Images Python](nova-images-python.md)
- [Policy](https://docs.openstack.org/nova/2025.2/configuration/policy.html)
- [32.0.0 attachment controller](https://github.com/openstack/nova/blob/32.0.0/nova/api/openstack/compute/volume_attachments.py#L319-L347)
- [attachment/group](python-attachments-groups.md)
- [backup workflow](../backup-server-python.md)
