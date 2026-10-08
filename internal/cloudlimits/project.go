package cloudlimits

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func (p *workflow) resolveProject(ctx context.Context, identity string, result *ProjectResult) error {
	base, err := url.Parse(p.identity.Client.ServiceURL("projects"))
	if err != nil {
		return err
	}
	safe := resource.ID(identity).Validate() == nil
	for _, char := range identity {
		if unicode.IsSpace(char) {
			safe = false
		}
	}
	if safe {
		wire, err := p.get(ctx, p.identity, p.identity.Client.ServiceURL("projects", url.PathEscape(identity)))
		result.Observed = observed(wire)
		if err == nil {
			raw, err := responseObject(wire, "project", true)
			if err != nil {
				return err
			}
			project, err := rawResource(raw, wire)
			if err != nil {
				return err
			}
			id, present := project.Body["id"]
			if !present {
				id, _ = json.Marshal(identity)
			}
			if err := p.guard(ctx); err != nil {
				return wire.Fail(err)
			}
			result.Project, result.ID, result.SeededID = project, bytes.Clone(id), !present
			return nil
		}
		// Only the direct native HTTP rejection is compatible fallback. An
		// accepted decode/read/source or wrapped transport/retry failure is not.
		code, clean := err.(gophercloud.ErrUnexpectedResponseCode)
		if !clean || code.Actual != 400 && code.Actual != 403 && code.Actual != 404 {
			return err
		}
		result.Observed = nil
	}
	initial := *base
	initial.RawQuery = url.Values{"name": []string{identity}}.Encode()
	current := &initial
	seen := map[string]bool{}
	var selected *resource.RawResource
	for {
		key, err := projectPageKey(current, base)
		if err != nil {
			return err
		}
		if seen[key] {
			return &resource.PaginationCycleError{URL: current.String()}
		}
		seen[key] = true
		wire, err := p.get(ctx, p.identity, current.String())
		if wire != nil {
			result.Pages = append(result.Pages, observed(wire))
		}
		if err != nil {
			return err
		}
		fields, rows, err := projectRows(wire)
		if err != nil {
			return wire.Fail(err)
		}
		if len(rows) == 0 {
			break
		}
		for _, raw := range rows {
			project, err := rawResource(raw, wire)
			if err != nil {
				return err
			}
			// Resource.list supplies these keywords itself, so every consumed
			// row rejects collisions even when the JSON value is null.
			for _, key := range []string{"connection", "microversion", "_synchronized"} {
				if _, present := project.Body[key]; present {
					return wire.Fail(fmt.Errorf("project list Resource has duplicate constructor argument %q", key))
				}
			}
			if !projectMatches(project, identity) {
				continue
			}
			if selected != nil {
				return &resource.AmbiguousError{Resource: "project", Name: identity, IDs: []string{projectString(selected, "id"), projectString(project, "id")}}
			}
			selected = project
		}
		href, err := projectNext(wire, fields)
		if err != nil {
			return wire.Fail(err)
		}
		if href == "" {
			break
		}
		current, err = projectContinuation(current, base, href)
		if err != nil {
			return wire.Fail(err)
		}
		key, err = projectPageKey(current, base)
		if err != nil {
			return wire.Fail(err)
		}
		if seen[key] {
			return wire.Fail(&resource.PaginationCycleError{URL: current.String()})
		}
	}
	if err := p.guard(ctx); err != nil {
		return err
	}
	if selected == nil {
		return &resource.NotFoundError{Resource: "project", Reference: identity}
	}
	id, present := selected.Body["id"]
	if !present {
		id = json.RawMessage("null")
	}
	result.Project, result.ID = selected, bytes.Clone(id)
	return nil
}

func projectString(project *resource.RawResource, key string) string {
	var text string
	_ = json.Unmarshal(project.Body[key], &text)
	return text
}

func projectMatches(project *resource.RawResource, identity string) bool {
	return projectString(project, "id") == identity || projectString(project, "name") == identity
}

func projectRows(response *rest.Response) (map[string]json.RawMessage, []json.RawMessage, error) {
	if !utf8.Valid(response.Body) || !json.Valid(response.Body) {
		return nil, nil, fmt.Errorf("projects response must be complete JSON")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(response.Body, &fields); err != nil {
		return nil, nil, err
	}
	if fields == nil {
		return nil, nil, fmt.Errorf("projects response must be a nonnull object")
	}
	raw, present := fields["projects"]
	if !present {
		return nil, nil, fmt.Errorf("projects response lacks projects payload")
	}
	var rows []json.RawMessage
	if len(bytes.TrimSpace(raw)) > 0 && bytes.TrimSpace(raw)[0] == '[' {
		if err := json.Unmarshal(raw, &rows); err != nil {
			return nil, nil, err
		}
	} else {
		rows = []json.RawMessage{raw}
	}
	return fields, rows, nil
}
