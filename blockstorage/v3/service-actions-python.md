# Service Actions Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/enable_service`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services.API.Enable`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services.NewServiceTarget`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services.TargetOf`

- Enable sends PUT os-services/enable with {"binary", "host"} (null when unset) at the configured client microversion only and overlays the root response object.
  Tests: `blockstorage/v3/services/python_actions_test.go:TestPythonServiceActionsSendServiceBodies`.

Python/Go differences:

- Python takes a Service instance or ID string; Go takes a ServiceTarget with Binary and Host, where nil fields reproduce the null body Python sends for an ID string.
- Python returns a mutable Resource; Go returns a typed struct with raw Body/Wire/Envelope, Header, StatusCode and Discovery evidence.
- Per-request header options are a Go extension; authentication, microversion and content headers stay SDK owned.

### `python:block_storage/v3/disable_service`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services.API.Disable`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services.WithDisableReason`

- Disable sends PUT os-services/disable with binary and host, or os-services/disable-log-reason with disabled_reason added for a nonempty reason.
  Tests: `blockstorage/v3/services/python_actions_test.go:TestPythonServiceActionsSendServiceBodies`.

Python/Go differences:

- Python takes a Service instance or ID string; Go takes a ServiceTarget with Binary and Host.
- Python returns a mutable Resource; Go returns a typed struct with raw Body/Wire/Envelope, Header, StatusCode and Discovery evidence.
- Per-request header options are a Go extension; authentication, microversion and content headers stay SDK owned.

### `python:block_storage/v3/thaw_service`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services.API.Thaw`

- Thaw sends PUT os-services/thaw with {"host": host} at the configured client microversion and wraps 400 with operation context.
  Tests: `blockstorage/v3/services/python_actions_test.go:TestPythonServiceActionsSendServiceBodies`.

Python/Go differences:

- Python takes a Service instance or ID string; Go takes a ServiceTarget.
- Python returns a mutable Resource; Go returns a typed struct with raw Body/Wire/Envelope, Header, StatusCode and Discovery evidence.
- Per-request header options are a Go extension; authentication, microversion and content headers stay SDK owned.

### `python:block_storage/v3/freeze_service`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services.API.Freeze`

- Freeze sends PUT os-services/freeze with {"host": host} at the configured client microversion and wraps 400 with operation context.
  Tests: `blockstorage/v3/services/python_actions_test.go:TestPythonServiceActionsSendServiceBodies`.

Python/Go differences:

- Python takes a Service instance or ID string; Go takes a ServiceTarget.
- Python returns a mutable Resource; Go returns a typed struct with raw Body/Wire/Envelope, Header, StatusCode and Discovery evidence.
- Per-request header options are a Go extension; authentication, microversion and content headers stay SDK owned.

### `python:block_storage/v3/failover_service`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services.API.Failover`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services.WithFailoverCluster`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services.WithFailoverBackendID`

- Failover reads Cinder discovery once, sends PUT os-services/failover when 3.26 is supported (and matched by a configured microversion) or failover_host otherwise, with host plus nonempty cluster and backend_id, at the configured client microversion.
  Tests: `blockstorage/v3/services/python_actions_test.go:TestPythonServiceFailoverChoosesActionBySupportedMicroversion`.

Python/Go differences:

- Python takes a Service instance or ID string; Go takes a ServiceTarget.
- Python returns a mutable Resource; Go returns a typed struct with raw Body/Wire/Envelope, Header, StatusCode and Discovery evidence.
- Per-request header options are a Go extension; authentication, microversion and content headers stay SDK owned.
- A discovery failure other than a clean 404/405 is returned as an error, while keystoneauth may treat unavailable endpoint data as unsupported.

### `python:block_storage/v3/set_service_log_levels`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services.API.SetLogLevels`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services.WithLogBinary`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services.WithLogServer`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services.WithLogPrefix`

- SetLogLevels selects the configured microversion or min(3.32, server max), fails with ErrUnsupported before HTTP below 3.32, and sends PUT os-services/set-log with level, binary defaulting to "" and null server/prefix.
  Tests: `blockstorage/v3/services/python_actions_test.go:TestPythonServiceLogLevelsRequireMicroversion332`.

Python/Go differences:

- Python negotiates min(Resource max, server max) through keystoneauth discovery; Go sends the configured client microversion or performs one Cinder version discovery GET with the same cap, and the result keeps the discovery responses.
- Per-request header options are a Go extension; authentication, microversion and content headers stay SDK owned.
- Python returns None; Go returns an ActionReceipt with the actual response. Level and Binary enums are plain strings. An unsupported microversion is resource.ErrUnsupported instead of NotSupported.

### `python:block_storage/v3/get_service_log_levels`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services.API.GetLogLevels`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services.WithLogBinary`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services.WithLogServer`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services.WithLogPrefix`

- GetLogLevels sends PUT os-services/get-log at 3.32 or later with binary, server and prefix, decodes every log_levels entry's binary, host and levels, and fails with the accepted response when log_levels or an entry key is missing.
  Tests: `blockstorage/v3/services/python_actions_test.go:TestPythonServiceLogLevelsRequireMicroversion332`.

Python/Go differences:

- Python negotiates min(Resource max, server max) through keystoneauth discovery; Go sends the configured client microversion or performs one Cinder version discovery GET with the same cap, and the result keeps the discovery responses.
- Per-request header options are a Go extension; authentication, microversion and content headers stay SDK owned.
- Python returns a lazy generator of LogLevel resources that sends the request on first iteration; Go sends the request on call and returns a LogLevels slice with the actual response.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
svc = next(conn.block_storage.services(binary="cinder-volume"))
conn.block_storage.disable_service(svc, reason="maintenance")
for level in conn.block_storage.get_service_log_levels(binary="cinder-api"):
    print(level.host, level.levels)
```

```go
api := services.New(volumeClient)
for svc, err := range api.List(ctx, services.WithListOptions(services.ListOpts{Binary: "cinder-volume"})) {
    if err != nil { return err }
    if _, err := api.Disable(ctx, services.TargetOf(svc), services.WithDisableReason("maintenance")); err != nil { return err }
    break
}
levels, err := api.GetLogLevels(ctx, services.WithLogBinary("cinder-api"))
if err != nil { return err }
for _, level := range levels.Levels { fmt.Println(level.Host, level.Levels) }
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L3423-L3616)
- [`service.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/service.py)
