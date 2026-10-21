package stacktemplates

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

// ValidateTemplateOpts describes a template validation request. Unlike
// ValidateOpts it can carry a JSON template object, an environment and Heat's
// ignore_errors query. A nil field is omitted from the request, except that the
// template key is sent as JSON null when only TemplateURL is supplied.
type ValidateTemplateOpts struct {
	// Template is the complete template as a JSON object or string.
	Template json.RawMessage
	// Environment is an optional JSON environment object.
	Environment json.RawMessage
	// TemplateURL locates the template when Template is nil. Heat ignores it
	// when Template is also supplied.
	TemplateURL *string
	// IgnoreErrors is a comma-separated list of error codes to ignore; Heat
	// currently documents only 99001. An empty string omits the query.
	IgnoreErrors string
}

func nonNullJSON(name string, value json.RawMessage) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(value)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil, nil
	}
	if !json.Valid(trimmed) {
		return nil, fmt.Errorf("%w: %s must be valid JSON", resource.ErrInvalidOption, name)
	}
	return json.RawMessage(trimmed), nil
}

// ValidateTemplate validates a template with the request shape of the Python
// openstacksdk call: {"template": template-or-null} plus optional environment
// and template_url, and ?ignore_errors= when requested. The response is the
// complete raw JSON object (Description, Parameters, ParameterGroups and any
// other field) because the native typed model constrains ParameterGroups to a
// map. Only HTTP 200 succeeds and a validation failure keeps its native error.
func (a *API) ValidateTemplate(ctx context.Context, opts ValidateTemplateOpts) (*resource.RawResource, error) {
	if err := ctx.Err(); err != nil {
		return nil, request.Wrap("ValidateTemplate", "stacktemplates", err)
	}
	template, err := nonNullJSON("template", opts.Template)
	if err != nil {
		return nil, request.Wrap("ValidateTemplate", "stacktemplates", err)
	}
	environment, err := nonNullJSON("environment", opts.Environment)
	if err != nil {
		return nil, request.Wrap("ValidateTemplate", "stacktemplates", err)
	}
	if template == nil && opts.TemplateURL == nil {
		return nil, request.Wrap("ValidateTemplate", "stacktemplates", fmt.Errorf("%w: template_url must be specified when template is nil", resource.ErrInvalidOption))
	}
	body := map[string]any{"template": nil}
	if template != nil {
		body["template"] = template
	}
	if environment != nil {
		body["environment"] = environment
	}
	if opts.TemplateURL != nil {
		body["template_url"] = *opts.TemplateURL
	}
	endpoint := a.client.ServiceURL("validate")
	if strings.TrimSpace(opts.IgnoreErrors) != "" {
		endpoint += "?" + url.Values{"ignore_errors": {opts.IgnoreErrors}}.Encode()
	}
	var raw json.RawMessage
	response, err := a.client.Post(ctx, endpoint, body, &raw, &gophercloud.RequestOpts{OkCodes: []int{200}})
	if err != nil {
		return nil, request.Wrap("ValidateTemplate", "stacktemplates", err)
	}
	var document resource.RawResource
	if err := document.UnmarshalJSON(raw); err != nil {
		return nil, request.Wrap("ValidateTemplate", "stacktemplates", err)
	}
	document.Header = response.Header.Clone()
	document.StatusCode = response.StatusCode
	return &document, nil
}
