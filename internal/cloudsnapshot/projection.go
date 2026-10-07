package cloudsnapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudfilter"
	"github.com/JSYoo5B/gophercloudsdk/internal/jmespath"
)

// An explicit list expression always evaluates the owned complete JSON value,
// including an empty string's syntax error. The raw Source generator bug and
// falsey raw jmespath_filters are separate from this concrete Go control.
func (p *reader) projectList(ctx context.Context, rows []json.RawMessage, expression *string) (result cloudfilter.Result, err error) {
	defer func() {
		if observed := p.source.Guard(ctx); observed != nil {
			result = cloudfilter.Result{}
			err = errors.Join(err, observed)
		}
	}()
	if expression == nil {
		return cloudfilter.Select(rows, "", nil, func() error { return p.source.Guard(ctx) })
	}
	if err := p.source.Guard(ctx); err != nil {
		return cloudfilter.Result{}, err
	}
	input, err := json.Marshal(rows)
	if err != nil {
		return cloudfilter.Result{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.UseNumber()
	var data any
	if err := decoder.Decode(&data); err != nil {
		return cloudfilter.Result{}, err
	}
	if err := p.source.Guard(ctx); err != nil {
		return cloudfilter.Result{}, err
	}
	output, err := jmespath.Search(*expression, data)
	if err != nil {
		return cloudfilter.Result{}, err
	}
	if err := p.source.Guard(ctx); err != nil {
		return cloudfilter.Result{}, err
	}
	value, err := json.Marshal(output)
	if err != nil {
		return cloudfilter.Result{}, err
	}
	return cloudfilter.Result{Value: value, Expression: true}, nil
}
