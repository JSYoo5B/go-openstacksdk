package policies

import (
	"fmt"
	"net/url"
	"strconv"

	"gophercloudsdk/internal/senlin"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

type ListOpts struct {
	Limit         int
	Marker        string
	Name          string
	Type          string
	Sort          string
	GlobalProject *bool
}

type ListOption = request.Option[ListOpts]

func WithListOptions(value ListOpts) ListOption {
	value.GlobalProject = senlin.Bool(value.GlobalProject)
	return func(config *request.Config[ListOpts]) error {
		copy := value
		copy.GlobalProject = senlin.Bool(value.GlobalProject)
		config.Options = copy
		return nil
	}
}

func WithListGlobalProject(value bool) ListOption {
	return func(config *request.Config[ListOpts]) error {
		copy := value
		config.Options.GlobalProject = &copy
		return nil
	}
}

func WithListQuery(key, value string) ListOption { return request.WithQuery[ListOpts](key, value) }

// The API documents sort's grammar, while deployment policy determines the
// supported attribute names. Validate syntax without inventing an allowlist.
func validateSort(value string) error {
	return senlin.SortGrammar(value)
}

func listQuery(config request.Config[ListOpts]) (url.Values, error) {
	if err := request.ValidateCapabilities(config, false, true, false, localFiltersKey); err != nil {
		return nil, err
	}
	value := config.Options
	if err := validateSort(value.Sort); err != nil {
		return nil, err
	}
	query, err := senlin.Query(request.Config[senlin.ListOpts]{Options: senlin.ListOpts{Limit: value.Limit, Marker: value.Marker}})
	if err != nil {
		return nil, err
	}
	for key, field := range map[string]string{"name": value.Name, "type": value.Type, "sort": value.Sort} {
		if field != "" {
			query.Set(key, field)
		}
	}
	if value.GlobalProject != nil {
		query.Set("global_project", strconv.FormatBool(*value.GlobalProject))
	}
	for key, values := range config.Query {
		switch key {
		case "limit", "marker", "name", "type", "sort", "global_project", "data", "spec", "id", "project", "project_id", "domain", "domain_id", "user", "user_id", "created_at", "updated_at":
			return nil, fmt.Errorf("%w: query %q is a concrete option or local policy filter", resource.ErrInvalidOption, key)
		}
		query[key] = append([]string(nil), values...)
	}
	return query, nil
}
