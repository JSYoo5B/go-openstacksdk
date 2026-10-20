package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ServiceActionTarget names the service for EnableRecord and DisableRecord
// the way enable_service and disable_service take service, host and binary.
// It has the same fields and rules as ForcedDownRecordTarget: from microversion
// 2.53 a non-empty ID is used directly, otherwise Host and Binary select
// exactly one listed service; below 2.53 Host and Binary are required.
type ServiceActionTarget = ForcedDownRecordTarget

// EnableRecord implements enable_service for the client's selected
// microversion. From 2.53 it sends PUT os-services/{id} with
// {"status": "enabled"}; when only Host and Binary are given it lists
// os-services?binary=&host=, requires one row and, like Resource.commit,
// sends nothing when that row is already enabled. Below 2.53 it follows
// Service.enable: PUT os-services/enable with host and binary, keeping the
// id-less response in Fields without a typed Service. Every status below 400
// is accepted.
func (a *API) EnableRecord(ctx context.Context, target ServiceActionTarget) (*ServiceRecord, error) {
	return a.serviceStatusRecord(ctx, "EnableRecord", target, "enabled", "")
}

// DisableRecord implements disable_service. From 2.53 it sends
// {"status": "disabled"} plus disabled_reason when reason is non-empty to
// PUT os-services/{id}, with the same host and binary lookup and unchanged
// attribute suppression as EnableRecord. Below 2.53 it follows
// Service.disable: PUT os-services/disable with host and binary, or
// os-services/disable-log-reason with disabled_reason added.
func (a *API) DisableRecord(ctx context.Context, target ServiceActionTarget, reason string) (*ServiceRecord, error) {
	return a.serviceStatusRecord(ctx, "DisableRecord", target, "disabled", reason)
}

func (a *API) serviceStatusRecord(ctx context.Context, op string, target ServiceActionTarget, status, reason string) (*ServiceRecord, error) {
	fail := func(value *ServiceRecord, err error) (*ServiceRecord, error) {
		return value, request.Wrap(op, "services", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return fail(nil, err)
	}
	if a == nil {
		return fail(nil, fmt.Errorf("%w: service API is required", resource.ErrInvalidOption))
	}
	source, err := cloudread.Capture(ctx, a.client, "compute")
	if err != nil {
		return fail(nil, err)
	}
	modern, err := pythonServiceSupports(source.Client.Microversion, "2.53")
	if err != nil {
		return fail(nil, err)
	}
	named := strings.TrimSpace(target.Host) != "" && strings.TrimSpace(target.Binary) != ""
	if !modern {
		if !named {
			return fail(nil, fmt.Errorf("%w: host and binary are required below microversion 2.53", resource.ErrInvalidOption))
		}
		action := "enable"
		body := map[string]any{"host": target.Host, "binary": target.Binary}
		if status == "disabled" {
			action = "disable"
			if reason != "" {
				action = "disable-log-reason"
				body["disabled_reason"] = reason
			}
		}
		return pythonServiceSend(ctx, source, op, source.Client.ServiceURL("os-services", action), body, false)
	}
	attrs := map[string]any{"status": status}
	if reason != "" {
		attrs["disabled_reason"] = reason
	}
	id := target.ID
	if strings.TrimSpace(id) == "" {
		if !named {
			return fail(nil, fmt.Errorf("%w: either a service id or both host and binary are required", resource.ErrInvalidOption))
		}
		resolved, row, response, err := pythonServiceResolve(ctx, source, target.Host, target.Binary)
		if err != nil {
			return fail(nil, err)
		}
		for key, want := range attrs {
			if pythonServiceSame(row[key], want.(string)) {
				delete(attrs, key)
			}
		}
		if len(attrs) == 0 {
			// Nothing became dirty, so Python returns the listed Service.
			listed := &ServiceRecord{Fields: row, Header: response.Header.Clone(), StatusCode: response.StatusCode}
			raw, err := json.Marshal(row)
			service := new(Service)
			if err == nil {
				err = json.Unmarshal(raw, service)
			}
			if err != nil {
				return fail(listed, response.Fail(err))
			}
			listed.Service = service
			return listed, nil
		}
		id = resolved
	}
	return pythonServicePut(ctx, source, op, id, attrs)
}

// pythonServiceSame mirrors _ComponentManager.__setitem__: an attribute stays
// clean only when the listed value equals the requested string.
func pythonServiceSame(raw json.RawMessage, want string) bool {
	var have string
	trimmed := bytes.TrimSpace(raw)
	return bytes.HasPrefix(trimmed, []byte(`"`)) && json.Unmarshal(trimmed, &have) == nil && have == want
}
