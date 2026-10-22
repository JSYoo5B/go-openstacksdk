package stackcloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks"

	"github.com/gophercloud/gophercloud/v2"
)

var acceptedCodes = []int{200, 201, 202, 203, 204}

// DefaultPollInterval is event_utils.poll_for_events' poll_period.
const DefaultPollInterval = 5 * time.Second

// event is one decoded entry of a stack events response.
type event map[string]json.RawMessage

// getEvents ports event_utils.get_events: GET /stacks/{ref}/events with the
// sorted event arguments. A missing events member is an empty list; any other
// non-list, or a non-object entry, is an error.
func getEvents(ctx context.Context, client *gophercloud.ServiceClient, ref string, query url.Values) ([]event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	endpoint := client.ServiceURL("stacks", url.PathEscape(ref), "events")
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	body, _, err := exchange(ctx, client, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	var document map[string]json.RawMessage
	if err := decodeObject(body, &document); err != nil {
		return nil, fmt.Errorf("stack events response: %w", err)
	}
	raw, ok := document["events"]
	if !ok {
		return nil, nil
	}
	var entries []json.RawMessage
	if trimmed := bytes.TrimSpace(raw); len(trimmed) == 0 || trimmed[0] != '[' {
		return nil, fmt.Errorf("stack events response: events must be a list")
	}
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("stack events response: %w", err)
	}
	events := make([]event, len(entries))
	for i, entry := range entries {
		var value map[string]json.RawMessage
		if err := decodeObject(entry, &value); err != nil {
			return nil, fmt.Errorf("stack event %d: %w", i, err)
		}
		events[i] = value
	}
	return events, nil
}

// lastEventMarker ports the marker capture of update_stack and delete_stack:
// get_events(sort_dir=desc, limit=1) and events[0].id. An empty list gives no
// marker; a first event without an id fails like the Munch attribute access.
func lastEventMarker(ctx context.Context, client *gophercloud.ServiceClient, ref string) (json.RawMessage, error) {
	events, err := getEvents(ctx, client, ref, url.Values{"limit": {"1"}, "sort_dir": {"desc"}})
	if err != nil || len(events) == 0 {
		return nil, err
	}
	id, ok := events[0]["id"]
	if !ok {
		return nil, fmt.Errorf("latest stack event has no id")
	}
	return bytes.Clone(id), nil
}

func markerQuery(marker json.RawMessage) (url.Values, error) {
	query := url.Values{"sort_dir": {"asc"}}
	if marker == nil {
		return query, nil
	}
	values, err := cloudfilter.RequestQueryValues(marker)
	if err != nil {
		return nil, fmt.Errorf("event marker: %w", err)
	}
	if len(values) > 0 {
		query["marker"] = values
	}
	return query, nil
}

func jsonStringIs(raw json.RawMessage, want string) bool {
	var value string
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '"' || json.Unmarshal(trimmed, &value) != nil {
		return false
	}
	return value == want
}

// isStackEvent ports poll_for_events.is_stack_event. Values that make the
// Python helper raise (non-list links, non-object links, a non-string stack
// reference) are errors.
func isStackEvent(e event, stackName string) (bool, error) {
	resourceName, ok := e["resource_name"]
	if !ok {
		resourceName = json.RawMessage(`""`)
	}
	physical, ok := e["physical_resource_id"]
	if !ok {
		physical = json.RawMessage(`""`)
	}
	if !jsonStringIs(resourceName, stackName) && !jsonStringIs(physical, stackName) {
		return false, nil
	}
	reference := physical
	if rawLinks, ok := e["links"]; ok {
		trimmed := bytes.TrimSpace(rawLinks)
		if len(trimmed) == 0 || trimmed[0] != '[' {
			return false, fmt.Errorf("stack event links must be a list")
		}
		var links []json.RawMessage
		if err := json.Unmarshal(trimmed, &links); err != nil {
			return false, fmt.Errorf("stack event links: %w", err)
		}
		for _, rawLink := range links {
			var link map[string]json.RawMessage
			if err := decodeObject(rawLink, &link); err != nil {
				return false, fmt.Errorf("stack event link: %w", err)
			}
			rel, ok := link["rel"]
			if !ok {
				rel = json.RawMessage("null")
			}
			if trimmed := bytes.TrimSpace(rel); len(trimmed) > 0 && (trimmed[0] == '[' || trimmed[0] == '{') {
				return false, fmt.Errorf("stack event link rel must be hashable")
			}
			if jsonStringIs(rel, "stack") {
				href, ok := link["href"]
				if !ok {
					href = json.RawMessage("null")
				}
				reference = href
			}
		}
	}
	var text string
	if trimmed := bytes.TrimSpace(reference); len(trimmed) == 0 || trimmed[0] != '"' || json.Unmarshal(trimmed, &text) != nil {
		return false, fmt.Errorf("stack event reference must be a string")
	}
	if i := strings.LastIndex(text, "/"); i >= 0 {
		text = text[i+1:]
	}
	return jsonStringIs(physical, text), nil
}

func stopStatus(action, status string) bool {
	return status == action+"_FAILED" || status == action+"_COMPLETE"
}

// pollForEvents ports event_utils.poll_for_events with a fixed action. It
// returns the terminal status (action_COMPLETE or action_FAILED); reaching a
// failed status is not an error, exactly as in Python. There is no overall
// timeout: ctx bounds the loop.
func pollForEvents(ctx context.Context, client *gophercloud.ServiceClient, stackName, action string, marker json.RawMessage, interval time.Duration) (string, error) {
	noEventPolls := 0
	for {
		query, err := markerQuery(marker)
		if err != nil {
			return "", err
		}
		events, err := getEvents(ctx, client, stackName, query)
		if err != nil {
			return "", err
		}
		if len(events) == 0 {
			noEventPolls++
		} else {
			noEventPolls = 0
			marker = nil
			if id, ok := events[len(events)-1]["id"]; ok {
				marker = bytes.Clone(id)
			}
			for _, e := range events {
				matched, err := isStackEvent(e, stackName)
				if err != nil {
					return "", err
				}
				if !matched {
					continue
				}
				var status string
				if raw, ok := e["resource_status"]; ok {
					_ = json.Unmarshal(raw, &status)
				}
				if stopStatus(action, status) {
					return status, nil
				}
			}
		}
		if noEventPolls >= 2 {
			stack, err := getStack(ctx, client, stackName, false)
			if err != nil {
				return "", err
			}
			if stack != nil {
				if stopStatus(action, stack.Status) {
					return stack.Status, nil
				}
				noEventPolls = 0
			}
		}
		if err := sleep(ctx, interval); err != nil {
			return "", err
		}
	}
}

func sleep(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// getStack ports the cloud get_stack used by these workflows: one GET with
// the optional resolve_outputs=False, absence (404, DELETE_COMPLETE or
// ADOPT_COMPLETE) as nil, then _filter_list's exact or glob name-or-ID check
// on the single result.
func getStack(ctx context.Context, client *gophercloud.ServiceClient, nameOrID string, resolveOutputs bool) (*stacks.StackDetail, error) {
	options := []stacks.FindOption{stacks.WithFindIgnoreMissing()}
	if !resolveOutputs {
		options = append(options, stacks.WithFindResolveOutputs(false))
	}
	detail, err := stacks.New(client).FindNameOrID(ctx, nameOrID, options...)
	if err != nil || detail == nil {
		return nil, err
	}
	view, err := stackViewFromFields(detail.Raw.Body)
	if err != nil {
		return nil, err
	}
	selected, err := cloudfilter.SelectSource([]json.RawMessage{view}, nameOrID, nil, ctx.Err)
	if err != nil {
		return nil, err
	}
	if len(selected.Indices) == 0 {
		return nil, nil
	}
	return detail, nil
}

// exchange sends one request without replay and returns the response body.
func exchange(ctx context.Context, client *gophercloud.ServiceClient, method, endpoint string, body any) (json.RawMessage, *http.Response, error) {
	opts := &gophercloud.RequestOpts{KeepResponseBody: true, OkCodes: acceptedCodes}
	if body != nil {
		opts.JSONBody = body
	}
	response, err := client.Request(ctx, method, endpoint, opts)
	if err != nil {
		return nil, response, err
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, response, err
	}
	return raw, response, nil
}

func decodeObject(raw []byte, target *map[string]json.RawMessage) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return fmt.Errorf("must be a JSON object")
	}
	return json.Unmarshal(trimmed, target)
}
