package compute

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strconv"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// KeypairQueryOpts preserves absent versus present filters. UserID belongs to
// GetKeypair's unfiltered member lookup; filtered get/search use only Filters.
type KeypairQueryOpts struct {
	Filters      *json.RawMessage
	UserID       string
	Microversion *string
	MaxItems     int
	Paginated    *bool
}
type KeypairQueryOption = request.Option[KeypairQueryOpts]

func cloneKeypairQueryOptions(value KeypairQueryOpts) KeypairQueryOpts {
	if value.Filters != nil {
		owned := json.RawMessage(bytes.Clone(*value.Filters))
		value.Filters = &owned
	}
	if value.Microversion != nil {
		owned := *value.Microversion
		value.Microversion = &owned
	}
	if value.Paginated != nil {
		owned := *value.Paginated
		value.Paginated = &owned
	}
	return value
}
func WithKeypairQueryOptions(value KeypairQueryOpts) KeypairQueryOption {
	owned := cloneKeypairQueryOptions(value)
	return func(c *request.Config[KeypairQueryOpts]) error {
		c.Options = cloneKeypairQueryOptions(owned)
		return nil
	}
}
func WithKeypairQueryFilters(value json.RawMessage) KeypairQueryOption {
	owned := bytes.Clone(value)
	return func(c *request.Config[KeypairQueryOpts]) error {
		c.Options.Filters = nil
		if owned != nil {
			raw := json.RawMessage(bytes.Clone(owned))
			c.Options.Filters = &raw
		}
		return nil
	}
}
func WithKeypairQueryExpression(expression string) KeypairQueryOption {
	raw, _ := json.Marshal(expression)
	return WithKeypairQueryFilters(raw)
}
func WithKeypairQueryUserID(value string) KeypairQueryOption {
	return func(c *request.Config[KeypairQueryOpts]) error { c.Options.UserID = value; return nil }
}
func WithKeypairQueryMicroversion(value string) KeypairQueryOption {
	return func(c *request.Config[KeypairQueryOpts]) error {
		owned := value
		c.Options.Microversion = &owned
		return nil
	}
}
func WithKeypairQueryMaxItems(value int) KeypairQueryOption {
	return func(c *request.Config[KeypairQueryOpts]) error { c.Options.MaxItems = value; return nil }
}
func WithKeypairQueryPaginated(value bool) KeypairQueryOption {
	return func(c *request.Config[KeypairQueryOpts]) error {
		owned := value
		c.Options.Paginated = &owned
		return nil
	}
}
func WithKeypairQueryHeader(key, value string) KeypairQueryOption {
	return request.WithHeader[KeypairQueryOpts](key, value)
}

// Inventory is the actual consumed inventory, including partial work on error.
// Value is only populated after complete collection and successful selection.
// JMESPath can return arbitrary JSON without an invented Keypair association.
type KeypairQueryResult struct {
	Value     json.RawMessage
	Keypairs  []*keypairs.KeypairRecord
	Inventory []*keypairs.KeypairRecord
}
type GetKeypairResult struct {
	Value     json.RawMessage
	Keypair   *keypairs.KeypairRecord
	Inventory []*keypairs.KeypairRecord
}
type keypairWorkflowFailure struct{ error }

func (e keypairWorkflowFailure) Unwrap() error          { return e.error }
func (keypairWorkflowFailure) TerminalSDKFailure() bool { return true }

type keypairWorkflow struct {
	ctx            context.Context
	api            *keypairs.API
	check          func(context.Context) error
	locationReader func() (resource.CloudLocation, error)
	location       json.RawMessage
}

func (s *Service) captureKeypairs(ctx context.Context) (*keypairWorkflow, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if s == nil || s.API == nil || s.API.KeyPairs == nil {
		return nil, invalid("keypair service is required")
	}
	serviceAPI, api, serviceClient := s.API, s.API.KeyPairs, s.client
	collection := s.Servers
	if serviceAPI.RawClient() != serviceClient || api.RawClient() != serviceClient {
		return nil, invalid("keypair components must share their selected client")
	}
	source, err := cloudread.Capture(ctx, api.RawClient(), "compute")
	if err != nil {
		return nil, err
	}
	outer := rest.OperationGuard(ctx)
	var observed error
	guard := func(checkCtx context.Context) error {
		if observed != nil {
			return cloudread.ContextError(checkCtx, observed)
		}
		var outerErr error
		if outer != nil {
			outerErr = outer(checkCtx)
		}
		var changed error
		if s.API != serviceAPI || serviceAPI.KeyPairs != api || s.client != serviceClient || s.Servers != collection || serviceAPI.RawClient() != serviceClient || api.RawClient() != serviceClient {
			changed = invalid("keypair service binding changed")
		}
		if err := errors.Join(source.Guard(checkCtx), outerErr, changed); err != nil {
			observed = keypairWorkflowFailure{err}
		}
		return observed
	}
	if err := guard(ctx); err != nil {
		return nil, err
	}
	p := &keypairWorkflow{ctx: rest.WithOperationGuard(ctx, guard), api: api, check: guard}
	if collection != nil {
		p.locationReader = collection.dependencies.CloudLocation
	}
	return p, nil
}

// The Connection supplies this existing location adapter; standalone clients
// have no Connection location. Only returned resource views receive it.
func (p *keypairWorkflow) captureLocation() error {
	if p.location != nil {
		return p.check(p.ctx)
	}
	if err := p.check(p.ctx); err != nil {
		return err
	}
	value := json.RawMessage("null")
	var err error
	if p.locationReader != nil {
		location, readErr := p.locationReader()
		err = readErr
		if err == nil {
			value, err = location.ForResource(nil, nil)
		}
	}
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		return err
	}
	p.location = bytes.Clone(value)
	return nil
}
func (p *keypairWorkflow) locate(record *keypairs.KeypairRecord) {
	if record != nil && record.Resource != nil {
		record.Resource.Body["location"] = bytes.Clone(p.location)
	}
}
func (p *keypairWorkflow) prepare(options []KeypairQueryOption) (request.Config[KeypairQueryOpts], error) {
	if err := p.captureLocation(); err != nil {
		return request.Config[KeypairQueryOpts]{}, err
	}
	guarded := make([]KeypairQueryOption, len(options))
	for i, option := range append([]KeypairQueryOption(nil), options...) {
		apply := option
		guarded[i] = func(c *request.Config[KeypairQueryOpts]) error {
			if err := p.check(p.ctx); err != nil {
				return err
			}
			if apply == nil {
				return invalid("nil keypair query option")
			}
			c.Options = cloneKeypairQueryOptions(c.Options)
			c.Headers = maps.Clone(c.Headers)
			if c.Headers == nil {
				c.Headers = map[string]string{}
			}
			err := apply(c)
			c.Options = cloneKeypairQueryOptions(c.Options)
			c.Headers = maps.Clone(c.Headers)
			return errors.Join(err, p.check(p.ctx))
		}
	}
	c, err := request.Apply(KeypairQueryOpts{}, guarded...)
	if err == nil {
		err = request.ValidateCapabilities(c, false, false, true)
	}
	if err == nil && c.Options.MaxItems < 0 {
		err = invalid("keypair raw row cap must be nonnegative")
	}
	if err == nil && c.Options.Filters != nil {
		raw := *c.Options.Filters
		if !utf8.Valid(raw) || !json.Valid(raw) {
			err = invalid("keypair filters must be UTF-8 JSON")
		}
	}
	return c, errors.Join(err, p.check(p.ctx))
}
func keypairFilterState(raw *json.RawMessage) (value json.RawMessage, object, present, truthy bool, err error) {
	if raw == nil {
		return
	}
	value = bytes.TrimSpace(*raw)
	present = !bytes.Equal(value, []byte("null"))
	object = len(value) > 0 && value[0] == '{'
	truthy, err = cloudfilter.PythonTruthy(value)
	return
}
func keypairQueryValues(rows []*keypairs.KeypairRecord) ([]json.RawMessage, json.RawMessage, error) {
	views := make([]json.RawMessage, len(rows))
	for i, row := range rows {
		if row == nil || row.Resource == nil {
			return nil, nil, invalid("keypair inventory resource is required")
		}
		var err error
		views[i], err = json.Marshal(row.Resource)
		if err != nil {
			return nil, nil, err
		}
	}
	value, err := json.Marshal(views)
	return views, value, err
}

// listOptions connects the existing semantic classifier and paging controls;
// it does not introduce a second collection reader or matching engine.
func keypairCloudListOptions(c request.Config[KeypairQueryOpts], dictionary json.RawMessage) ([]keypairs.KeypairListOption, error) {
	options := []keypairs.KeypairListOption{keypairs.WithKeypairListMaxItems(c.Options.MaxItems)}
	if c.Options.Paginated != nil {
		options = append(options, keypairs.WithKeypairListPaginated(*c.Options.Paginated))
	}
	if c.Options.Microversion != nil {
		options = append(options, keypairs.WithKeypairListMicroversion(*c.Options.Microversion))
	}
	for key, value := range c.Headers {
		options = append(options, keypairs.WithKeypairListHeader(key, value))
	}
	if dictionary == nil {
		return options, nil
	}
	members, err := cloudfilter.ObjectMembers(dictionary)
	if err != nil {
		return nil, err
	}
	for _, member := range members {
		switch member.Key {
		case "paginated":
			truthy, err := cloudfilter.PythonTruthy(member.Value)
			if err != nil {
				return nil, err
			}
			options = append(options, keypairs.WithKeypairListPaginated(truthy))
		case "max_items":
			truthy, err := cloudfilter.PythonTruthy(member.Value)
			if err != nil {
				return nil, err
			}
			maximum := 0
			if truthy {
				if bytes.Equal(member.Value, []byte("true")) {
					maximum = 1
				} else {
					maximum, err = strconv.Atoi(string(member.Value))
					if err != nil || maximum < 0 {
						return nil, invalid("keypair max_items must be a nonnegative integer")
					}
				}
			}
			options = append(options, keypairs.WithKeypairListMaxItems(maximum))
		case "microversion":
			if !bytes.Equal(member.Value, []byte("null")) {
				var version string
				if err := json.Unmarshal(member.Value, &version); err != nil {
					return nil, invalid("keypair microversion must be a string or null")
				}
				options = append(options, keypairs.WithKeypairListMicroversion(version))
			}
		case "headers":
			if !bytes.Equal(member.Value, []byte("null")) {
				var headers map[string]string
				if err := json.Unmarshal(member.Value, &headers); err != nil || headers == nil {
					return nil, invalid("keypair headers must be a string dictionary or null")
				}
				for key, value := range headers {
					options = append(options, keypairs.WithKeypairListHeader(key, value))
				}
			}
		default:
			options = append(options, keypairs.WithKeypairListFilter(member.Key, json.RawMessage(bytes.Clone(member.Value))))
		}
	}
	return options, nil
}
func (p *keypairWorkflow) inventory(c request.Config[KeypairQueryOpts], dictionary json.RawMessage) (*KeypairQueryResult, error) {
	result := &KeypairQueryResult{Inventory: []*keypairs.KeypairRecord{}}
	options, err := keypairCloudListOptions(c, dictionary)
	if err != nil {
		return result, err
	}
	for row, err := range p.api.ListRecords(p.ctx, options...) {
		if err != nil {
			return result, err
		}
		p.locate(row)
		result.Inventory = append(result.Inventory, row)
	}
	if err := p.check(p.ctx); err != nil {
		return result, err
	}
	return result, nil
}
func keypairCloudFailure(operation string, err error) error {
	return request.Wrap(operation, "keypairs", err)
}

// ListKeypairs eagerly consumes all selected pages. Failure returns only
// Inventory evidence; Value and Keypairs never assert a completed list.
func (s *Service) ListKeypairs(ctx context.Context, options ...KeypairQueryOption) (*KeypairQueryResult, error) {
	p, err := s.captureKeypairs(ctx)
	if err != nil {
		return nil, keypairCloudFailure("ListKeypairs", err)
	}
	c, err := p.prepare(options)
	if err != nil {
		return nil, keypairCloudFailure("ListKeypairs", err)
	}
	raw, object, _, truthy, err := keypairFilterState(c.Options.Filters)
	if err == nil && truthy && !object {
		err = invalid("keypair list filters must be a dictionary")
	}
	if err != nil {
		return nil, keypairCloudFailure("ListKeypairs", err)
	}
	if !truthy {
		raw = nil
	}
	result, err := p.inventory(c, raw)
	if err == nil {
		_, result.Value, err = keypairQueryValues(result.Inventory)
	}
	if err == nil {
		err = p.check(p.ctx)
	}
	if err == nil {
		result.Keypairs = append([]*keypairs.KeypairRecord{}, result.Inventory...)
	} else {
		result.Value = nil
	}
	return result, keypairCloudFailure("ListKeypairs", err)
}
func (p *keypairWorkflow) search(identity string, c request.Config[KeypairQueryOpts]) (*KeypairQueryResult, error) {
	raw, object, _, _, err := keypairFilterState(c.Options.Filters)
	if err != nil {
		return nil, err
	}
	if !object {
		raw = nil
	}
	result, err := p.inventory(c, raw)
	if err != nil {
		return result, err
	}
	views, _, err := keypairQueryValues(result.Inventory)
	if err != nil {
		return result, err
	}
	selected, err := cloudfilter.Select(views, identity, c.Options.Filters, func() error { return p.check(p.ctx) })
	if err != nil {
		return result, fmt.Errorf("%w: keypair local search: %w", resource.ErrInvalidOption, err)
	}
	if err = p.check(p.ctx); err != nil {
		return result, err
	}
	result.Value = bytes.Clone(selected.Value)
	if !selected.Expression {
		result.Keypairs = make([]*keypairs.KeypairRecord, len(selected.Indices))
		for i, index := range selected.Indices {
			result.Keypairs[i] = result.Inventory[index]
		}
	}
	return result, nil
}

// SearchKeypairs reapplies dictionary filters after the declared list filters,
// then uses the existing exact/glob/JMESPath engine on complete inventory.
func (s *Service) SearchKeypairs(ctx context.Context, nameOrID string, options ...KeypairQueryOption) (*KeypairQueryResult, error) {
	p, err := s.captureKeypairs(ctx)
	if err != nil {
		return nil, keypairCloudFailure("SearchKeypairs", err)
	}
	c, err := p.prepare(options)
	if err != nil {
		return nil, keypairCloudFailure("SearchKeypairs", err)
	}
	result, err := p.search(nameOrID, c)
	return result, keypairCloudFailure("SearchKeypairs", err)
}

// GetKeypair only uses member find when filters are absent/null. Even empty
// present filters select cloud search and intentionally do not forward UserID.
func (s *Service) GetKeypair(ctx context.Context, nameOrID string, options ...KeypairQueryOption) (*GetKeypairResult, error) {
	p, err := s.captureKeypairs(ctx)
	if err != nil {
		return nil, keypairCloudFailure("GetKeypair", err)
	}
	c, err := p.prepare(options)
	if err != nil {
		return nil, keypairCloudFailure("GetKeypair", err)
	}
	_, _, present, _, err := keypairFilterState(c.Options.Filters)
	if err != nil {
		return nil, keypairCloudFailure("GetKeypair", err)
	}
	result := &GetKeypairResult{}
	if !present {
		options := []keypairs.KeypairFindOption{keypairs.WithKeypairFindUserID(c.Options.UserID)}
		if c.Options.Microversion != nil {
			options = append(options, keypairs.WithKeypairFindMicroversion(*c.Options.Microversion))
		}
		for key, value := range c.Headers {
			options = append(options, keypairs.WithKeypairFindHeader(key, value))
		}
		result.Keypair, err = p.api.FindKeypair(p.ctx, nameOrID, options...)
		p.locate(result.Keypair)
		if err == nil && result.Keypair != nil {
			result.Value, err = json.Marshal(result.Keypair.Resource)
		}
	} else {
		search, searchErr := p.search(nameOrID, c)
		err = searchErr
		if search != nil {
			result.Inventory = search.Inventory
		}
		if err == nil {
			result.Value, err = cloudfilter.First(search.Value)
			var multiple *cloudfilter.MultipleError
			if errors.As(err, &multiple) {
				err = fmt.Errorf("%w: keypair %q has %d matches", resource.ErrAmbiguous, nameOrID, multiple.Length)
			}
			if err == nil && result.Value != nil && len(search.Keypairs) == 1 {
				result.Keypair = search.Keypairs[0]
			}
		}
	}
	err = errors.Join(err, p.check(p.ctx))
	if err != nil {
		result.Value = nil
	}
	return result, keypairCloudFailure("GetKeypair", err)
}
