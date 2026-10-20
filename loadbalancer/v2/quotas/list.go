package quotas

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ProjectListOpts applies to the project-quota collection, not a scope.
// Optional projections may omit project_id; missing IDs are never invented.
type ProjectListOpts struct {
	ProjectID   string
	Fields      []string
	Limit       int
	Marker      string
	PageReverse *bool
}
type ProjectListOption = request.Option[ProjectListOpts]

func WithProjectListOptions(options ProjectListOpts) ProjectListOption {
	options = cloneProjectListOptions(options)
	return func(config *request.Config[ProjectListOpts]) error {
		config.Options = cloneProjectListOptions(options)
		return nil
	}
}
func cloneProjectListOptions(options ProjectListOpts) ProjectListOpts {
	options.Fields = append([]string(nil), options.Fields...)
	if options.PageReverse != nil {
		value := *options.PageReverse
		options.PageReverse = &value
	}
	return options
}

func (a *API) ListProjects(ctx context.Context, options ...ProjectListOption) iter.Seq2[*QuotaResource, error] {
	options = append([]ProjectListOption(nil), options...)
	return func(yield func(*QuotaResource, error) bool) {
		if err := a.validateQuotaClient(ctx); err != nil {
			yield(nil, quotaError("ListProjects", "", err))
			return
		}
		config, err := request.Apply(ProjectListOpts{}, options...)
		if err == nil {
			err = request.ValidateCapabilities(config, false, false, false)
		}
		if err != nil {
			yield(nil, quotaError("ListProjects", "", err))
			return
		}
		input := config.Options
		if input.Limit < 0 {
			yield(nil, quotaError("ListProjects", "", fmt.Errorf("%w: page limit must be non-negative", resource.ErrInvalidOption)))
			return
		}
		if input.ProjectID != "" {
			if err := validateProjectID(input.ProjectID); err != nil {
				yield(nil, quotaError("ListProjects", "", err))
				return
			}
		}
		if input.Marker != "" {
			if err := resource.ID(input.Marker).Validate(); err != nil {
				yield(nil, quotaError("ListProjects", "", err))
				return
			}
		}
		if err := validateFields(input.Fields); err != nil {
			yield(nil, quotaError("ListProjects", "", err))
			return
		}
		initial, err := url.Parse(a.quotaEndpoint())
		if err != nil {
			yield(nil, quotaError("ListProjects", "", err))
			return
		}
		query := initial.Query()
		if input.ProjectID != "" {
			query.Set("project_id", input.ProjectID)
		}
		if input.Limit > 0 {
			query.Set("limit", strconv.Itoa(input.Limit))
		}
		if input.Marker != "" {
			query.Set("marker", input.Marker)
		}
		if input.PageReverse != nil {
			// Octavia compares the raw value to the case-sensitive string "True".
			value := "False"
			if *input.PageReverse {
				value = "True"
			}
			query.Set("page_reverse", value)
		}
		for _, field := range input.Fields {
			query.Add("fields", field)
		}
		initial.RawQuery = query.Encode()
		next := initial.String()
		visited := make(map[string]bool)
		for next != "" {
			if err := ctx.Err(); err != nil {
				yield(nil, quotaError("ListProjects", "", err))
				return
			}
			key := quotaPageKey(next)
			if visited[key] {
				yield(nil, quotaError("ListProjects", "", &resource.PaginationCycleError{URL: next}))
				return
			}
			visited[key] = true
			wire, err := a.requestQuota(ctx, http.MethodGet, next, nil, []int{200})
			if err != nil {
				yield(nil, quotaError("ListProjects", "", err))
				return
			}
			var envelope map[string]json.RawMessage
			if err = json.Unmarshal(wire.body, &envelope); err != nil {
				yield(nil, quotaError("ListProjects", "", wire.fail(err)))
				return
			}
			var entries []json.RawMessage
			array, exists := envelope["quotas"]
			if !exists || strings.TrimSpace(string(array)) == "null" {
				yield(nil, quotaError("ListProjects", "", wire.fail(fmt.Errorf("quota response requires a quotas array"))))
				return
			}
			if err = json.Unmarshal(array, &entries); err != nil {
				yield(nil, quotaError("ListProjects", "", wire.fail(err)))
				return
			}
			for _, entry := range entries {
				if err := ctx.Err(); err != nil {
					yield(nil, quotaError("ListProjects", "", err))
					return
				}
				quota, body, err := decodeQuotaObject(entry)
				if err != nil {
					yield(nil, quotaError("ListProjects", "", wire.fail(err)))
					return
				}
				id := ""
				if raw, exists := body["project_id"]; exists {
					if err = json.Unmarshal(raw, &id); err == nil {
						err = validateProjectID(id)
					}
					if err != nil {
						yield(nil, quotaError("ListProjects", "", wire.fail(err)))
						return
					}
				}
				value := &QuotaResource{Quota: quota, ProjectID: id, Body: body, Header: wire.header.Clone(), StatusCode: wire.status}
				if !yield(value, nil) {
					return
				}
			}
			if err := ctx.Err(); err != nil {
				yield(nil, quotaError("ListProjects", "", err))
				return
			}
			next, err = nextQuotaPage(envelope["quotas_links"], next, initial)
			if err != nil {
				yield(nil, quotaError("ListProjects", "", wire.fail(err)))
				return
			}
		}
	}
}

func (a *API) AllProjects(ctx context.Context, options ...ProjectListOption) ([]*QuotaResource, error) {
	values := make([]*QuotaResource, 0)
	for value, err := range a.ListProjects(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func nextQuotaPage(raw json.RawMessage, current string, origin *url.URL) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}
	var links []struct {
		Href string `json:"href"`
		Rel  string `json:"rel"`
	}
	if err := json.Unmarshal(raw, &links); err != nil {
		return "", err
	}
	base, err := url.Parse(current)
	if err != nil {
		return "", err
	}
	next := ""
	for _, link := range links {
		if link.Rel != "next" || link.Href == "" {
			continue
		}
		candidate, err := url.Parse(link.Href)
		if err != nil {
			return "", err
		}
		candidate = base.ResolveReference(candidate)
		if candidate.User != nil || !strings.EqualFold(candidate.Scheme, origin.Scheme) || !strings.EqualFold(candidate.Host, origin.Host) {
			return "", fmt.Errorf("%w: quota pagination link changes origin", resource.ErrUnsupported)
		}
		candidate.Fragment = ""
		// Octavia's generated links omit filters and projections. Carry the
		// original options forward while letting explicit link paging values win.
		query := candidate.Query()
		for key, values := range origin.Query() {
			if _, present := query[key]; !present {
				query[key] = append([]string(nil), values...)
			}
		}
		candidate.RawQuery = query.Encode()
		if next != "" && next != candidate.String() {
			return "", fmt.Errorf("%w: quota page has multiple next links", resource.ErrInvalidOption)
		}
		next = candidate.String()
	}
	return next, nil
}

func quotaPageKey(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""
	if query, err := url.ParseQuery(u.RawQuery); err == nil {
		u.RawQuery = query.Encode()
	}
	return u.String()
}
