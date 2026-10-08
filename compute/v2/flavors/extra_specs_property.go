package flavors

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudread"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// FlavorExtraSpecPropertyRecord preserves the property value without a string
// or dictionary conversion. Missing properties have Value null and Present
// false; an explicit JSON null has Present true. Wire and the receipt preserve
// the actual response independently of Value.
type FlavorExtraSpecPropertyRecord struct {
	Value      json.RawMessage
	Present    bool
	Wire       *resource.RawResource
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
}

func validateFlavorExtraSpecsProperty(property string) error {
	if property == "" || property == "." || property == ".." || !utf8.ValidString(property) || strings.IndexFunc(property, unicode.IsControl) >= 0 {
		return fmt.Errorf("%w: extra-spec property must be nonempty valid text without controls or dot segments", resource.ErrInvalidOption)
	}
	return nil
}

// GetExtraSpecsProperty always reads the single escaped property endpoint. It
// does not update the supplied flavor, fetch the complete extra_specs object,
// or interpret passive response values as a route. Read options and version
// selection are identical to FetchExtraSpecs.
func (a *API) GetExtraSpecsProperty(ctx context.Context, input FlavorExtraSpecsRequest, property string, options ...FlavorExtraSpecsOption) (*FlavorExtraSpecPropertyRecord, error) {
	fail := func(value *FlavorExtraSpecPropertyRecord, err error) (*FlavorExtraSpecPropertyRecord, error) {
		return value, request.Wrap("GetExtraSpecsProperty", "flavors", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return fail(nil, err)
	}
	if err := validateFlavorExtraSpecsProperty(property); err != nil {
		return fail(nil, err)
	}
	read, err := a.prepareFlavorExtraSpecsRead(ctx, input, options...)
	if err != nil {
		return fail(nil, err)
	}
	wire, response, err := read.read(property)
	if response == nil {
		return fail(nil, err)
	}
	result := &FlavorExtraSpecPropertyRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	if err != nil {
		return fail(result, err)
	}
	if err := read.check(read.ctx); err != nil {
		return fail(result, response.Fail(cloudread.ContextError(read.ctx, err)))
	}
	value, present := wire.Body[property]
	if !present {
		value = json.RawMessage(`null`)
	}
	result.Value, result.Present = bytes.Clone(value), present
	result.Wire = wire.Clone()
	return result, nil
}
