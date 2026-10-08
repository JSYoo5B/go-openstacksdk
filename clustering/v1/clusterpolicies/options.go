package clusterpolicies

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/JSYoo5B/go-openstacksdk/internal/senlin"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type ListOpts struct {
	Enabled    *bool
	PolicyName string
	PolicyType string
	Sort       string
	MaxItems   int
	Paginated  *bool
}

type ListOption = request.Option[ListOpts]

func WithListOptions(value ListOpts) ListOption { return senlin.Snapshot(value) }

func WithListMaxItems(value int) ListOption {
	return func(config *request.Config[ListOpts]) error { config.Options.MaxItems = value; return nil }
}

func WithListPaginated(value bool) ListOption {
	return func(config *request.Config[ListOpts]) error {
		copy := value
		config.Options.Paginated = &copy
		return nil
	}
}

func WithListEnabled(value bool) ListOption {
	return func(config *request.Config[ListOpts]) error {
		copy := value
		config.Options.Enabled = &copy
		return nil
	}
}

func WithListPolicyName(value string) ListOption {
	return func(config *request.Config[ListOpts]) error { config.Options.PolicyName = value; return nil }
}

func WithListPolicyType(value string) ListOption {
	return func(config *request.Config[ListOpts]) error { config.Options.PolicyType = value; return nil }
}

func WithListSort(value string) ListOption {
	return func(config *request.Config[ListOpts]) error { config.Options.Sort = value; return nil }
}

// WithListQuery snapshots a deployment query. Published concrete fields and
// unsupported initial pagination controls cannot be replaced through it.
func WithListQuery(key, value string) ListOption { return request.WithQuery[ListOpts](key, value) }

func validateQuery(query url.Values) error {
	if query.Has("max_items") || query.Has("paginated") {
		return fmt.Errorf("%w: cluster policy pagination controls are local options", resource.ErrInvalidOption)
	}
	if query.Has("is_enabled") {
		return fmt.Errorf("%w: cluster policy is_enabled is an SDK alias; wire queries use enabled", resource.ErrInvalidOption)
	}
	return senlin.SortGrammar(query.Get("sort"))
}

func validateInitialQuery(query url.Values) error {
	if query.Has("limit") || query.Has("marker") {
		return fmt.Errorf("%w: initial cluster policy limit and marker lack a published server contract", resource.ErrUnsupported)
	}
	for _, key := range []string{"cluster_id", "id", "policy_id", "cluster_name", "data"} {
		if query.Has(key) {
			return fmt.Errorf("%w: query %q is a cluster policy response field or fixed scope", resource.ErrInvalidOption, key)
		}
	}
	return validateQuery(query)
}

func listQuery(config request.Config[ListOpts]) (url.Values, error) {
	if err := senlin.ValidateListCapabilities(config, filterSpec.Namespace); err != nil {
		return nil, err
	}
	if err := senlin.RejectListControlQuery(config.Query); err != nil {
		return nil, err
	}
	if config.Options.MaxItems < 0 {
		return nil, fmt.Errorf("%w: maximum items must be non-negative", resource.ErrInvalidOption)
	}
	query := make(url.Values)
	value := config.Options
	if value.Enabled != nil {
		query.Set("enabled", strconv.FormatBool(*value.Enabled))
	}
	for key, field := range map[string]string{"policy_name": value.PolicyName, "policy_type": value.PolicyType, "sort": value.Sort} {
		if field != "" {
			query.Set(key, field)
		}
	}
	for key, values := range config.Query {
		switch key {
		case "enabled", "is_enabled", "policy_name", "policy_type", "sort", "cluster_id", "id", "policy_id", "cluster_name", "data", "max_items", "paginated":
			return nil, fmt.Errorf("%w: query %q is owned by a concrete cluster policy option or response field", resource.ErrInvalidOption, key)
		}
		query[key] = append([]string(nil), values...)
	}
	if err := senlin.RejectBodyFilterQuery(config.Query, filterSpec); err != nil {
		return nil, err
	}
	if err := validateInitialQuery(query); err != nil {
		return nil, err
	}
	return query, nil
}
