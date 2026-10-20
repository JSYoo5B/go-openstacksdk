# Python Migration Actions

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/evacuate_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.EvacuateWithOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithEvacuateServerOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithEvacuateServerHost`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithEvacuateServerAdminPass`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithEvacuateServerForce`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithEvacuateServerOnSharedStorage`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithEvacuateServerHeader`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithEvacuateServerField`

- EvacuateWithOptions posts {"evacuate": {}} to servers/{id}/action by default and only the supplied host, adminPass, force and onSharedStorage keys otherwise, keeping empty strings and false, with the client microversion and a single empty Accept header.
  Tests: `compute/v2/servers/python_migration_actions_test.go:TestPythonMigrationEvacuateAndMigrateSendOnlySuppliedKeys`.
- Each migration action accepts every status below 400, returns 400 and above as the Gophercloud status error inside resource.OperationError named after the method for servers, and rejects a blank server ID, a nil option, a core-key extension and a nil API before HTTP.
  Tests: `compute/v2/servers/python_migration_actions_test.go:TestPythonMigrationActionsStatusesAndPreflight`.

Python/Go differences:

- Python sends min(Server _max_microversion 2.100, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the client.
- Python accepts a Server or an ID; Go takes the server ID string.
- Like Python, Go does not gate onSharedStorage or force by microversion, so Nova validates them.
- WithEvacuateServerField can send extension keys inside the evacuate object, which Python cannot.

### `python:compute/v2/migrate_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.MigrateWithOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithMigrateServerHost`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithMigrateServerHeader`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithMigrateServerField`

- MigrateWithOptions posts {"migrate": null} without a host or with an empty host and {"migrate": {"host": h}} with one, and a host below selected microversion 2.56 fails with resource.ErrUnsupported before HTTP.
  Tests: `compute/v2/servers/python_migration_actions_test.go:TestPythonMigrationEvacuateAndMigrateSendOnlySuppliedKeys`.
- Each migration action accepts every status below 400, returns 400 and above as the Gophercloud status error inside resource.OperationError named after the method for servers, and rejects a blank server ID, a nil option, a core-key extension and a nil API before HTTP.
  Tests: `compute/v2/servers/python_migration_actions_test.go:TestPythonMigrationActionsStatusesAndPreflight`.

Python/Go differences:

- Python sends min(Server _max_microversion 2.100, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the client.
- Python checks the host against the supported server microversion (or the session default) and raises ValueError; Go checks it against the selected client microversion and returns resource.ErrUnsupported.
- Python accepts a Server or an ID; Go takes the server ID string.

### `python:compute/v2/live_migrate_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.LiveMigrateWithOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithLiveMigrateServerOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithLiveMigrateServerHost`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithLiveMigrateServerForce`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithLiveMigrateServerBlockMigration`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithLiveMigrateServerAutoBlockMigration`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithLiveMigrateServerDiskOverCommit`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithLiveMigrateServerHeader`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithLiveMigrateServerField`

- LiveMigrateWithOptions derives Python's body from the selected microversion: from 2.30 (and latest) it sends the 2.30 body with microversion 2.30 (host null unless given, block_migration defaulting to "auto", force only with a host), from 2.25 the 2.25 body with 2.25 and no force key, and below 2.25 or unversioned the legacy body with block_migration and disk_over_commit defaulting to false under the client microversion.
  Tests: `compute/v2/servers/python_migration_actions_test.go:TestPythonMigrationLiveMigrateFollowsSelectedMicroversionShape`.
- Python's ValueError cases (a host without force below 2.30, "auto" below 2.25) plus auto combined with a boolean, a caller version header that conflicts with the fixed body version and a core-key extension fail with resource.ErrInvalidOption inside resource.OperationError before HTTP.
  Tests: `compute/v2/servers/python_migration_actions_test.go:TestPythonMigrationLiveMigrateFollowsSelectedMicroversionShape`.
- Each migration action accepts every status below 400, returns 400 and above as the Gophercloud status error inside resource.OperationError named after the method for servers, and rejects a blank server ID, a nil option, a core-key extension and a nil API before HTTP.
  Tests: `compute/v2/servers/python_migration_actions_test.go:TestPythonMigrationActionsStatusesAndPreflight`.

Python/Go differences:

- Python chooses the 2.30, 2.25 or legacy body from the supported server microversion (or the session default); Go chooses it from the selected client microversion, treating latest as 2.30 or later, and like Python sends 2.30 or 2.25 as the request microversion for those bodies.
- Python accepts a Server or an ID; Go takes the server ID string, and block_migration is expressed by WithLiveMigrateServerBlockMigration(bool) or WithLiveMigrateServerAutoBlockMigration().
- WithLiveMigrateServerField can send extension keys inside os-migrateLive, which Python cannot.

## References

- [`evacuate_server`, `migrate_server`, `live_migrate_server`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py)
- [`Server._action`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server.py)
