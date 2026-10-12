package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ForcedDownRecordTarget names the service the way update_service_forced_down
// does. From microversion 2.53 a non-empty ID is used directly; without it Host
// and Binary select exactly one listed service. Below 2.53 Host and Binary are
// required and ID is not sent, because the legacy route has no identity.
type ForcedDownRecordTarget struct {
	ID     string
	Host   string
	Binary string
}

// UpdateForcedDownRecord implements update_service_forced_down for the
// client's selected microversion. From 2.53 it sends PUT os-services/{id}
// with {"forced_down": forced}, resolving the id through GET
// os-services?binary=&host= when only Host and Binary are given. Below 2.53 it
// follows Service.set_forced_down: PUT os-services/force-down with host and
// binary, adding forced_down and pinning microversion 2.11 when the client
// selects 2.11 or later. Every status below 400 is accepted; the legacy
// response is kept in Fields without a typed Service because Python returns
// None for it.
func (a *API) UpdateForcedDownRecord(ctx context.Context, target ForcedDownRecordTarget, forced bool) (*ServiceRecord, error) {
	const op = "UpdateForcedDownRecord"
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
	if modern {
		id := target.ID
		if strings.TrimSpace(id) == "" {
			if !named {
				return fail(nil, fmt.Errorf("%w: either a service id or both host and binary are required", resource.ErrInvalidOption))
			}
			if id, err = pythonServiceResolve(ctx, source, target.Host, target.Binary); err != nil {
				return fail(nil, err)
			}
		}
		return pythonServicePut(ctx, source, op, id, map[string]any{"forced_down": forced})
	}
	if !named {
		return fail(nil, fmt.Errorf("%w: host and binary are required below microversion 2.53", resource.ErrInvalidOption))
	}
	body := map[string]any{"host": target.Host, "binary": target.Binary}
	pinned, err := pythonServiceSupports(source.Client.Microversion, "2.11")
	if err == nil && pinned {
		body["forced_down"] = forced
		version := "2.11"
		err = source.WithPolicy(ctx, &version, nil)
	}
	if err != nil {
		return fail(nil, err)
	}
	return pythonServiceSend(ctx, source, op, source.Client.ServiceURL("os-services", "force-down"), body, false)
}

// pythonServiceResolve follows _get_service from 2.53: list by host and
// binary and require exactly one row, whose id addresses the update.
func pythonServiceResolve(ctx context.Context, source *cloudread.Source, host, binary string) (string, error) {
	query := url.Values{"binary": {binary}, "host": {host}}
	response, err := source.Get(ctx, source.Client.ServiceURL("os-services")+"?"+query.Encode(), pythonServiceCodes()...)
	if err != nil {
		return "", err
	}
	var page struct {
		Services *[]struct {
			ID json.RawMessage `json:"id"`
		} `json:"services"`
	}
	if err := json.Unmarshal(response.Body, &page); err != nil || page.Services == nil {
		return "", response.Fail(errors.Join(fmt.Errorf("%w: service list response lacks a services array", resource.ErrInvalidOption), err))
	}
	switch rows := *page.Services; len(rows) {
	case 0:
		return "", response.Fail(fmt.Errorf("%w: found no matches for service host %q binary %q", resource.ErrNotFound, host, binary))
	case 1:
		raw := bytes.TrimSpace(rows[0].ID)
		var text string
		if err := json.Unmarshal(raw, &text); err == nil && strings.TrimSpace(text) != "" {
			return text, nil
		}
		var number json.Number
		if err := json.Unmarshal(raw, &number); err == nil && number != "" {
			return number.String(), nil
		}
		return "", response.Fail(fmt.Errorf("%w: listed service has no usable id", resource.ErrInvalidOption))
	default:
		return "", response.Fail(fmt.Errorf("%w: found multiple matches for service host %q binary %q", resource.ErrAmbiguous, host, binary))
	}
}
