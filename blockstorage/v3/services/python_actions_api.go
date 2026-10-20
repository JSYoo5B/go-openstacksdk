package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/internal/cinderowned"
	"github.com/gophercloud/gophercloud/v2"
)

// The service action calls below are SDK owned: Gophercloud v2.15.0 binds
// only GET os-services. They follow the pinned openstacksdk Service
// Resource actions and classmethods.

const actionPath = "os-services"

var actionSpec = &cinderowned.Spec{
	Kind: "services", ResourcesKey: "services",
	Fields: []string{"id", "name", "active_backend_id", "zone", "backend_state", "binary", "cluster",
		"disabled_reason", "host", "frozen", "replication_status", "state", "status", "updated_at"},
}

// defaultVersion is Service._action: session.default_microversion, that is
// the configured client microversion without discovery.
var defaultVersion = cinderowned.Version{Default: true}

// logVersion is Service._assert_microversion_for(3.32): the configured
// microversion or min(3.32, server maximum), which must be at least 3.32.
var logVersion = cinderowned.Version{Ceiling: "3.32", Minimum: "3.32"}

func (a *API) actionTarget() cinderowned.Target {
	var client func() *gophercloud.ServiceClient
	if a != nil {
		client = func() *gophercloud.ServiceClient { return a.client }
	}
	return cinderowned.Target{Spec: actionSpec, Client: client, Path: actionPath}
}

func actionHeaders(v ActionOpts) map[string]string { return v.Headers }

// Enable implements enable_service: PUT os-services/enable with
// {"binary": binary, "host": host} (JSON null for a nil field).
func (a *API) Enable(ctx context.Context, service ServiceTarget, options ...ActionOption) (*ServiceRecord, error) {
	body := map[string]json.RawMessage{"binary": service.binary(), "host": service.host()}
	return serviceAction(ctx, a, "Enable", "enable", service, body, false, options)
}

// Disable implements disable_service: PUT os-services/disable with binary and
// host, or os-services/disable-log-reason with disabled_reason added when
// WithDisableReason gives a nonempty reason.
func (a *API) Disable(ctx context.Context, service ServiceTarget, options ...DisableOption) (*ServiceRecord, error) {
	return serviceAction(ctx, a, "Disable", "disable", service, nil, false, options)
}

// Thaw implements thaw_service: PUT os-services/thaw with {"host": host}.
func (a *API) Thaw(ctx context.Context, service ServiceTarget, options ...ActionOption) (*ServiceRecord, error) {
	return serviceAction(ctx, a, "Thaw", "thaw", service, map[string]json.RawMessage{"host": service.host()}, false, options)
}

// Freeze implements freeze_service: PUT os-services/freeze with {"host": host}.
func (a *API) Freeze(ctx context.Context, service ServiceTarget, options ...ActionOption) (*ServiceRecord, error) {
	return serviceAction(ctx, a, "Freeze", "freeze", service, map[string]json.RawMessage{"host": service.host()}, false, options)
}

// Failover implements failover_service: one version discovery decides
// supports_microversion(3.26); when true it sends PUT os-services/failover,
// otherwise os-services/failover_host, with {"host": host} plus cluster and
// backend_id when nonempty, at the configured client microversion.
func (a *API) Failover(ctx context.Context, service ServiceTarget, options ...FailoverOption) (*ServiceRecord, error) {
	return serviceAction(ctx, a, "Failover", "failover", service, nil, true, options)
}

func serviceAction[O ~func(*ActionOpts) error](ctx context.Context, a *API, op, action string, service ServiceTarget, body map[string]json.RawMessage, probe bool, options []O) (*ServiceRecord, error) {
	t := a.actionTarget()
	p, v, err := cinderowned.Begin(ctx, t, ActionOpts{}, copyAction, actionHeaders, defaultVersion, options)
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	switch action {
	case "disable":
		body = map[string]json.RawMessage{"binary": service.binary(), "host": service.host()}
		if v.Reason != "" {
			action = "disable-log-reason"
			body["disabled_reason"], _ = json.Marshal(v.Reason)
		}
	case "failover":
		body = map[string]json.RawMessage{"host": service.host()}
		if v.Cluster != "" {
			body["cluster"], _ = json.Marshal(v.Cluster)
		}
		if v.BackendID != "" {
			body["backend_id"], _ = json.Marshal(v.BackendID)
		}
	}
	if probe {
		supported, err := p.Supports("3.26")
		if err != nil {
			return serviceConvert(p.Evidence(nil, nil), t.Wrap(ctx, op, err))
		}
		if !supported {
			action = "failover_host"
		}
	}
	response, err := p.Do(http.MethodPut, p.ServiceURL(actionPath, action), body)
	if err != nil {
		return serviceConvert(p.Evidence(response, err), t.Wrap(ctx, op, err))
	}
	return serviceConvert(t.Wrapper(ctx, op)(p.Record(response, service.seed())))
}

// SetLogLevels implements set_service_log_levels: PUT os-services/set-log
// with {"level", "binary" ("" when unset), "server", "prefix"} (JSON null for
// an unset server or prefix) at microversion 3.32 or later.
func (a *API) SetLogLevels(ctx context.Context, level string, options ...LogOption) (*ActionReceipt, error) {
	const op = "SetLogLevels"
	t := a.actionTarget()
	p, v, err := cinderowned.Begin(ctx, t, LogOpts{}, copyLog, func(v LogOpts) map[string]string { return v.Headers }, logVersion, options)
	if err != nil {
		return receipt(p, nil, nil), t.Wrap(ctx, op, err)
	}
	body := v.body()
	body["level"], _ = json.Marshal(level)
	response, err := p.Do(http.MethodPut, p.ServiceURL(actionPath, "set-log"), body)
	return receipt(p, response, err), t.Wrap(ctx, op, err)
}

// GetLogLevels implements get_service_log_levels: PUT os-services/get-log with
// {"binary" ("" when unset), "server", "prefix"} at microversion 3.32 or
// later. Every log_levels entry must carry binary, host and levels.
func (a *API) GetLogLevels(ctx context.Context, options ...LogOption) (*LogLevels, error) {
	const op = "GetLogLevels"
	t := a.actionTarget()
	p, v, err := cinderowned.Begin(ctx, t, LogOpts{}, copyLog, func(v LogOpts) map[string]string { return v.Headers }, logVersion, options)
	if err != nil {
		return logLevels(receipt(p, nil, nil)), t.Wrap(ctx, op, err)
	}
	response, err := p.Do(http.MethodPut, p.ServiceURL(actionPath, "get-log"), v.body())
	result := logLevels(receipt(p, response, err))
	if err != nil {
		return result, t.Wrap(ctx, op, err)
	}
	result.Levels, err = decodeLogLevels(response.Body)
	if err = errors.Join(err, p.Guard(ctx)); err != nil {
		return result, t.Wrap(ctx, op, response.Fail(err))
	}
	return result, nil
}

func decodeLogLevels(body []byte) ([]LogLevel, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil || envelope == nil {
		return nil, cinderowned.Invalid("get-log response must be a JSON object")
	}
	raw, ok := envelope["log_levels"]
	if !ok {
		return nil, cinderowned.Invalid("get-log response has no log_levels")
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, cinderowned.Invalid("get-log log_levels must be a JSON array")
	}
	levels := make([]LogLevel, 0, len(entries))
	for _, entry := range entries {
		var members map[string]json.RawMessage
		if err := json.Unmarshal(entry, &members); err != nil || members == nil {
			return nil, cinderowned.Invalid("get-log entry must be a JSON object")
		}
		value := LogLevel{Body: map[string]json.RawMessage{}}
		for _, key := range []string{"binary", "host", "levels"} {
			field, ok := members[key]
			if !ok {
				return nil, cinderowned.Invalid("get-log entry has no %q", key)
			}
			value.Body[key] = bytes.Clone(field)
		}
		value.Binary = cinderowned.StringValue(members["binary"])
		value.Host = cinderowned.StringValue(members["host"])
		if levels := cinderowned.Decode[map[string]string](members["levels"]); levels != nil {
			value.Levels = *levels
		}
		levels = append(levels, value)
	}
	return levels, nil
}
