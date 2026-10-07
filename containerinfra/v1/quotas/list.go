package quotas

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"math/big"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// ListOpts controls the actual quota collection. Limit zero omits page size;
// empty sorting defaults to id/asc. Nil AllTenants omits the false-default flag.
// The first next link may publish a smaller server-capped effective page size.
// Marker identifies a quota row, not a project or resource name.
type ListOpts struct {
	Limit      int
	Marker     string
	SortKey    string
	SortDir    string
	AllTenants *bool
}

type ListOption = request.Option[ListOpts]

func WithListOptions(value ListOpts) ListOption {
	value = cloneListOptions(value)
	return func(config *request.Config[ListOpts]) error {
		config.Options = cloneListOptions(value)
		return nil
	}
}

func WithListAllTenants(value bool) ListOption {
	return func(config *request.Config[ListOpts]) error {
		copy := value
		config.Options.AllTenants = &copy
		return nil
	}
}

// WithListQuery adds a deployment query extension; typed core parameters cannot
// be overwritten. The server validates whether an extension is supported.
func WithListQuery(key, value string) ListOption { return request.WithQuery[ListOpts](key, value) }

func cloneListOptions(value ListOpts) ListOpts {
	if value.AllTenants != nil {
		copy := *value.AllTenants
		value.AllTenants = &copy
	}
	return value
}

func listQuery(config request.Config[ListOpts]) (url.Values, error) {
	input := cloneListOptions(config.Options)
	if input.Limit < 0 {
		return nil, fmt.Errorf("%w: quota page limit must be non-negative", resource.ErrInvalidOption)
	}
	if input.SortKey == "" {
		input.SortKey = "id"
	} else if strings.TrimSpace(input.SortKey) == "" {
		return nil, fmt.Errorf("%w: quota sort key must not be blank", resource.ErrInvalidOption)
	}
	if input.SortDir == "" {
		input.SortDir = "asc"
	}
	if input.SortDir != "asc" && input.SortDir != "desc" {
		return nil, fmt.Errorf("%w: quota sort direction must be asc or desc", resource.ErrInvalidOption)
	}
	query := make(url.Values)
	query.Set("sort_key", input.SortKey)
	query.Set("sort_dir", input.SortDir)
	if input.Limit > 0 {
		query.Set("limit", strconv.Itoa(input.Limit))
	}
	if input.Marker != "" {
		marker, err := quotaMarker(input.Marker)
		if err != nil {
			return nil, err
		}
		query.Set("marker", marker)
	}
	if input.AllTenants != nil {
		query.Set("all_tenants", strconv.FormatBool(*input.AllTenants))
	}
	for key, values := range config.Query {
		switch key {
		case "limit", "marker", "sort_key", "sort_dir", "all_tenants":
			return nil, fmt.Errorf("%w: query extension %q is a core quota list option", resource.ErrInvalidOption, key)
		}
		query[key] = append([]string(nil), values...)
	}
	return query, nil
}

func quotaMarker(value string) (string, error) {
	if value == "" || strings.IndexFunc(value, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
		return "", fmt.Errorf("%w: quota marker must be a positive integer row ID", resource.ErrInvalidOption)
	}
	integer, ok := new(big.Int).SetString(value, 10)
	if !ok || integer.Sign() <= 0 {
		return "", fmt.Errorf("%w: quota marker must be a positive integer row ID", resource.ErrInvalidOption)
	}
	return integer.String(), nil
}

// List lazily reads all quota pages. Break stops row decoding and page fetches;
// each iteration starts fresh. Collection rows have no fixed request pair.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*QuotaResource, error] {
	options = append([]ListOption(nil), options...)
	return func(yield func(*QuotaResource, error) bool) {
		if err := a.validateQuotaClient(ctx); err != nil {
			yield(nil, quotaError("List", "", err))
			return
		}
		config, err := request.Apply(ListOpts{}, options...)
		if err == nil {
			err = request.ValidateCapabilities(config, false, true, false)
		}
		if err != nil {
			yield(nil, quotaError("List", "", err))
			return
		}
		query, err := listQuery(config)
		if err != nil {
			yield(nil, quotaError("List", "", err))
			return
		}
		initial, err := url.Parse(a.client.ServiceURL("quotas"))
		if err != nil {
			yield(nil, quotaError("List", "", err))
			return
		}
		initial.RawQuery = query.Encode()
		next := initial.String()
		visited := make(map[string]bool)
		for next != "" {
			if err := ctx.Err(); err != nil {
				yield(nil, quotaError("List", "", err))
				return
			}
			if visited[next] {
				yield(nil, quotaError("List", "", &resource.PaginationCycleError{URL: next}))
				return
			}
			visited[next] = true
			wire, err := a.requestQuota(ctx, http.MethodGet, next, nil, []int{200})
			if err != nil {
				yield(nil, quotaError("List", "", err))
				return
			}
			var envelope map[string]json.RawMessage
			if err := json.Unmarshal(wire.body, &envelope); err != nil {
				yield(nil, quotaError("List", "", wire.fail(err)))
				return
			}
			var rows []json.RawMessage
			array, exists := envelope["quotas"]
			if !exists || strings.TrimSpace(string(array)) == "null" {
				yield(nil, quotaError("List", "", wire.fail(fmt.Errorf("quota collection requires a quotas array"))))
				return
			}
			if err := json.Unmarshal(array, &rows); err != nil {
				yield(nil, quotaError("List", "", wire.fail(err)))
				return
			}
			for _, row := range rows {
				if err := ctx.Err(); err != nil {
					yield(nil, quotaError("List", "", err))
					return
				}
				value, err := decodeCollectionQuota(wire, row)
				if err != nil {
					yield(nil, quotaError("List", "", err))
					return
				}
				if !yield(value, nil) {
					return
				}
			}
			if err := ctx.Err(); err != nil {
				yield(nil, quotaError("List", "", err))
				return
			}
			next, err = nextQuotaCollectionPage(envelope["next"], next, initial)
			if err != nil {
				yield(nil, quotaError("List", "", wire.fail(err)))
				return
			}
		}
	}
}

func (a *API) All(ctx context.Context, options ...ListOption) ([]*QuotaResource, error) {
	values := make([]*QuotaResource, 0)
	for value, err := range a.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func decodeCollectionQuota(page quotaWire, row json.RawMessage) (*QuotaResource, error) {
	value, err := decodeQuota(quotaWire{body: row, header: page.header, status: page.status}, "", "")
	if err != nil {
		var decode *QuotaResponseError
		if errors.As(err, &decode) {
			err = decode.Cause
		}
		return nil, page.fail(err)
	}
	if value.ID == "" {
		return nil, page.fail(fmt.Errorf("quota collection row requires its ID"))
	}
	if err := validateSegment(value.ProjectID); err != nil {
		return nil, page.fail(err)
	}
	if err := validateSegment(value.Resource); err != nil {
		return nil, page.fail(err)
	}
	return value, nil
}

func nextQuotaCollectionPage(raw json.RawMessage, current string, origin *url.URL) (string, error) {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
		return "", nil
	}
	var link string
	if err := json.Unmarshal(raw, &link); err != nil {
		return "", err
	}
	if link == "" {
		return "", nil
	}
	base, err := url.Parse(current)
	if err != nil {
		return "", err
	}
	candidate, err := url.Parse(link)
	if err != nil {
		return "", err
	}
	candidate = base.ResolveReference(candidate)
	if candidate.User != nil || candidate.Opaque != "" || candidate.Fragment != "" || !strings.EqualFold(candidate.Scheme, origin.Scheme) || !strings.EqualFold(candidate.Host, origin.Host) || candidate.EscapedPath() != origin.EscapedPath() {
		return "", fmt.Errorf("%w: quota pagination link changes service, path or URL authority", resource.ErrUnsupported)
	}
	nextQuery, err := url.ParseQuery(candidate.RawQuery)
	if err != nil {
		return "", err
	}
	query := base.Query()
	if len(nextQuery["marker"]) != 1 {
		return "", fmt.Errorf("%w: quota next link requires one row marker", resource.ErrInvalidOption)
	}
	marker, err := quotaMarker(nextQuery.Get("marker"))
	if err != nil {
		return "", err
	}
	for key, values := range nextQuery {
		if key == "marker" {
			continue
		}
		if key == "limit" {
			if len(values) != 1 {
				return "", fmt.Errorf("%w: quota next link requires one page limit", resource.ErrInvalidOption)
			}
			limit, err := strconv.Atoi(values[0])
			if err != nil || limit <= 0 {
				return "", fmt.Errorf("%w: invalid quota next page limit", resource.ErrInvalidOption)
			}
			value := strconv.Itoa(limit)
			if previous := query.Get("limit"); previous != "" && previous != value {
				requested, err := strconv.Atoi(previous)
				if err != nil || current != origin.String() || limit > requested {
					return "", fmt.Errorf("%w: quota next link changes effective page limit", resource.ErrUnsupported)
				}
			}
			// Magnum validate_limit caps the requested size at CONF.api.max_limit.
			// The first next link establishes that effective size (or supplies
			// it when initially omitted), which stays fixed thereafter.
			query.Set("limit", value)
			continue
		}
		if key == "all_tenants" && len(query[key]) == 0 && slices.Equal(values, []string{"false"}) {
			continue
		}
		if !slices.Equal(values, query[key]) {
			return "", fmt.Errorf("%w: quota next link changes query %q", resource.ErrUnsupported, key)
		}
	}
	// Magnum next links omit all_tenants. Keep it and every original query
	// extension while allowing only the validated row marker to advance.
	query.Set("marker", marker)
	candidate.Scheme, candidate.Host = origin.Scheme, origin.Host
	candidate.RawQuery = query.Encode()
	return candidate.String(), nil
}
