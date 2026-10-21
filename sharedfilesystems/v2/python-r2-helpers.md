# Manila Python-compatible helpers

These additions complement the strict typed APIs.

- `sharenetworks.API.UpdateMembers` PUTs `{"share_network": {...}}` with
  `request.Optional` members (`name`, `description`, `neutron_net_id`,
  `neutron_subnet_id`, `nova_net_id`): omitted, empty string or explicit null
  are distinct, unlike `UpdateOpts` whose strings are omitted when empty.
  Declared members cannot be overridden by `WithUpdateMembersField`; an update
  with no member fails before HTTP. Success is HTTP 200; the response is a
  `resource.RawResource`. Never replayed.
- `shares.API.FindByNameOrID` and `sharetransfers.API.FindByNameOrID`
  reproduce Python `Resource.find`: direct GET for a safe ID segment, list
  fallback (`GET /shares` or `/share-transfers`, summary view, `name=<identity>`
  unless a `name` query is supplied) after 400, 403 or 404, exact ID-or-name
  match, `resource.ErrAmbiguous` for two matches, `nil, nil` when missing unless
  `WithFindIgnoreMissing(false)`. `WithFindAllProjects` maps to `all_tenants`
  on the fallback only. Other GET errors are returned. Differences: identities
  that are not single path segments skip the GET; Python extra `**query` also
  goes to the GET as URI parameters.
- `shareaccessrules.API.AllowAccessAction` / `DenyAccessAction` send
  `POST /shares/{id}/action` with `allow_access` / `deny_access` directly, with
  no 2.45 floor, parent lookup or share-name resolution, as Python
  `create_access_rule` / `delete_access_rule` do. Allow accepts HTTP 200 and
  returns the full `access` object; Deny accepts 200, 202 and 204, returns the
  actual response, and returns `nil, nil` for 404 unless
  `WithDenyActionIgnoreMissing(false)`. `lock_reason` is a string (Python types
  it as bool). Mutations are never replayed; acceptance is not completion.
