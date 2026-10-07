package instanceactions

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/compute/v2/servers"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/instanceactions"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

// ActionResource joins the native list and detail models without pretending a
// list response contains detail events. Details is set only by Get. ServerID is
// the fixed request parent; InstanceUUID retains the server's response value.
// Body retains the action object's full JSON, including unknown fields, omitted
// versus null values, and event extensions. Header is an independent snapshot.
type ActionResource struct {
	InstanceAction
	ServerID  string
	Details   *InstanceActionDetail
	UpdatedAt *time.Time
	Events    *[]ActionEvent
	Body      map[string]json.RawMessage
	Header    http.Header
}

// EventFields retains all fields from the native event model. The alias gives
// ActionEvent's embedded fields an unambiguous name alongside its Event string.
type EventFields = Event

// ActionEvent adds Nova's 2.84 fault details and raw extension fields to the
// native event model. Native Host/HostID, traceback and timestamps are retained.
type ActionEvent struct {
	EventFields
	Details *string
	Body    map[string]json.RawMessage
}

func (e *ActionEvent) UnmarshalJSON(body []byte) error {
	fields, err := decodeObject(body, "action event")
	if err != nil {
		return err
	}
	var native Event
	if err := json.Unmarshal(body, &native); err != nil {
		return err
	}
	var extra struct {
		Details *string `json:"details"`
	}
	if err := json.Unmarshal(body, &extra); err != nil {
		return err
	}
	*e = ActionEvent{EventFields: native, Details: extra.Details, Body: fields}
	return nil
}

// ActionScope fixes the server parent once. RequestID is the action identifier;
// an action's operation name (such as reboot) is not a unique resource name.
type ActionScope struct {
	*resource.Collection[ActionResource]
	api      *API
	serverID string
}

// InServer resolves an explicit server ID without a request, or an exact server
// name once using the shared server Collection. It retains the selected client
// microversion; base list/detail operations do not require a new one.
func (a *API) InServer(ctx context.Context, ref resource.Ref) (*ActionScope, error) {
	if err := a.validateActionClient(ctx); err != nil {
		return nil, request.Wrap("InServer", "server action", err)
	}
	id, err := servers.New(a.client).Resources.ResolveID(ctx, ref)
	if err != nil {
		return nil, request.Wrap("InServer", "server action", err)
	}
	scope := &ActionScope{api: a, serverID: id}
	scope.Collection = resource.NewCollection(resource.Adapter[ActionResource]{
		Kind:              "server action",
		Get:               scope.get,
		ID:                func(value *ActionResource) string { return value.RequestID },
		Iterate:           scope.list,
		IterateControlled: scope.listControlled,
	})
	return scope, nil
}

// ServerID returns the resolved parent without a request.
func (s *ActionScope) ServerID() string { return s.serverID }

func (a *API) validateActionClient(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if a == nil || a.client == nil || a.client.ProviderClient == nil {
		return fmt.Errorf("%w: server actions require a service client", resource.ErrInvalidOption)
	}
	return nil
}

func (s *ActionScope) get(ctx context.Context, requestID string) (*ActionResource, error) {
	if err := s.api.validateActionClient(ctx); err != nil {
		return nil, err
	}
	result := upstream.Get(ctx, s.api.client, s.serverID, requestID)
	var envelope struct {
		Action json.RawMessage `json:"instanceAction"`
	}
	if err := result.Result.ExtractInto(&envelope); err != nil {
		return nil, err
	}
	value, err := decodeAction(envelope.Action, s.serverID, true)
	if err != nil {
		return nil, err
	}
	value.Header = result.Header.Clone()
	return value, nil
}

func decodeAction(body json.RawMessage, serverID string, detailed bool) (*ActionResource, error) {
	fields, err := decodeObject(body, "instance action")
	if err != nil {
		return nil, err
	}
	var detail InstanceActionDetail
	if err := json.Unmarshal(body, &detail); err != nil {
		return nil, err
	}
	var extra struct {
		Events *[]ActionEvent `json:"events"`
	}
	if err := json.Unmarshal(body, &extra); err != nil {
		return nil, err
	}
	value := &ActionResource{
		InstanceAction: InstanceAction{
			Action: detail.Action, InstanceUUID: detail.InstanceUUID, Message: detail.Message,
			ProjectID: detail.ProjectID, RequestID: detail.RequestID, StartTime: detail.StartTime, UserID: detail.UserID,
		},
		ServerID: serverID, UpdatedAt: detail.UpdatedAt, Events: extra.Events, Body: fields,
	}
	if detailed {
		value.Details = &detail
	}
	return value, nil
}

func decodeObject(body []byte, kind string) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil, err
	}
	if fields == nil {
		return nil, gophercloud.ErrUnexpectedType{Expected: kind + " JSON object", Actual: "null"}
	}
	return fields, nil
}

// Native v2.15.0 uses SinglePageBase and drops Nova's 2.58 pagination links.
// The scoped SDK uses the documented links array with the shared stream guard.
type actionPage struct{ pagination.LinkedPageBase }

func (p actionPage) IsEmpty() (bool, error) {
	if p.StatusCode == http.StatusNoContent {
		return true, nil
	}
	actions, err := p.actions()
	return len(actions) == 0, err
}

func (p actionPage) actions() ([]json.RawMessage, error) {
	var envelope map[string]json.RawMessage
	if err := p.ExtractInto(&envelope); err != nil {
		return nil, err
	}
	if envelope == nil {
		return nil, gophercloud.ErrUnexpectedType{Expected: "instance action page JSON object", Actual: "null"}
	}
	var actions []json.RawMessage
	if err := json.Unmarshal(envelope["instanceActions"], &actions); err != nil {
		return nil, err
	}
	if actions == nil {
		return nil, gophercloud.ErrUnexpectedType{Expected: "instanceActions JSON array", Actual: "null"}
	}
	return actions, nil
}

func (p actionPage) NextPageURL() (string, error) {
	var envelope struct {
		Links []gophercloud.Link `json:"links"`
	}
	if err := p.ExtractInto(&envelope); err != nil {
		return "", err
	}
	return gophercloud.ExtractNextURL(envelope.Links)
}

func (s *ActionScope) list(ctx context.Context, query url.Values) iter.Seq2[*ActionResource, error] {
	return s.listControlled(ctx, query, resource.ListControl{})
}

func (s *ActionScope) listControlled(ctx context.Context, query url.Values, control resource.ListControl) iter.Seq2[*ActionResource, error] {
	return func(yield func(*ActionResource, error) bool) {
		if err := s.api.validateActionClient(ctx); err != nil {
			yield(nil, err)
			return
		}
		encoded, err := request.ExtendQuery("", query)
		if err != nil {
			yield(nil, err)
			return
		}
		endpoint := s.api.client.ServiceURL("servers", s.serverID, "os-instance-actions") + encoded
		pager := pagination.NewPager(s.api.client, endpoint, func(result pagination.PageResult) pagination.Page {
			return actionPage{pagination.LinkedPageBase{PageResult: result}}
		})
		stream := resource.StreamWithControl(ctx, pager, func(page pagination.Page) ([]ActionResource, error) {
			p := page.(actionPage)
			actions, err := p.actions()
			if err != nil {
				return nil, err
			}
			values := make([]ActionResource, 0, len(actions))
			for _, body := range actions {
				value, err := decodeAction(body, s.serverID, false)
				if err != nil {
					return nil, err
				}
				value.Header = p.Header.Clone()
				values = append(values, *value)
			}
			return values, nil
		}, control)
		for value, err := range stream {
			if !yield(value, err) || err != nil {
				return
			}
		}
	}
}
