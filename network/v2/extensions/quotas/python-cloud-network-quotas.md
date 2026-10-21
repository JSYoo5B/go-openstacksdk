# Cloud-level network quota helpers

`Connection.NetworkProjectQuotas` fixes the Neutron quota singleton to an
explicit project ID or an exact Keystone project name (separate Identity
client, resolved once). `ProjectQuotaScope` then maps the Python cloud helpers:

- `get_network_quotas(name_or_id, details=False)`: `Get` reads
  `/quotas/{project}`; `Detail` reads `/quotas/{project}/details.json`.
- `set_network_quotas(name_or_id, **kwargs)`: `Update` sends
  `PUT /quotas/{project}` with `{"quota": {...}}`; nil omits a limit, zero and
  `-1` are sent, and `WithUpdateCheckLimit` sends `check_limit` including false.
- `delete_network_quotas(name_or_id)`: `Delete` sends `DELETE /quotas/{project}`
  with no body (`WithDeleteIgnoreMissing` selects strict missing handling).

Known differences: a project given as an explicit ID is not verified through
Keystone (Python always runs `find_project`); an update with no limits sends an
empty `quota` envelope where Python sends no request; the details route carries
a `.json` suffix; results are typed models instead of `Quota`/`QuotaDetails`
resources.
