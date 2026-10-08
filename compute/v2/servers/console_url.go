package servers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ConsoleURL invokes the legacy server console action. The returned JSON is the
// exact console value, including null, arrays, scalars and extension fields.
// The console type is required; this method does not select a default, negotiate
// a version, or fall back to the modern remote-console creation endpoint.
func (a *API) ConsoleURL(ctx context.Context, serverID, consoleType string) (json.RawMessage, error) {
	fail := func(err error) (json.RawMessage, error) {
		return nil, request.Wrap("ConsoleURL", "servers", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return fail(err)
	}
	if err := resource.ID(serverID).Validate(); err != nil {
		return fail(err)
	}
	if !utf8.ValidString(serverID) || strings.IndexFunc(serverID, unicode.IsControl) >= 0 {
		return fail(fmt.Errorf("%w: server ID must be valid text without controls", resource.ErrInvalidOption))
	}
	action, supported := map[string]string{
		"novnc": "os-getVNCConsole", "xvpvnc": "os-getVNCConsole",
		"spice-html5": "os-getSPICEConsole", "spice-direct": "os-getSPICEConsole",
		"rdp-html5": "os-getRDPConsole", "serial": "os-getSerialConsole",
	}[consoleType]
	if !supported {
		return fail(fmt.Errorf("%w: unsupported console type %q", resource.ErrInvalidOption, consoleType))
	}
	if a == nil {
		return fail(fmt.Errorf("%w: server API is required", resource.ErrInvalidOption))
	}
	original := a.client
	source, err := cloudread.Capture(ctx, original, "compute")
	if err != nil {
		return fail(err)
	}
	guard := func(ctx context.Context) error {
		var changed error
		if a.client != original {
			changed = fmt.Errorf("%w: server API source changed", resource.ErrInvalidOption)
		}
		return errors.Join(changed, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	if err := guard(ctx); err != nil {
		return fail(err)
	}

	// Server.get_console_url uses Server._action without a local type/version
	// gate. In particular spice-direct is sent literally on the selected client.
	body := map[string]any{action: map[string]string{"type": consoleType}}
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = i + 200
	}
	target := source.Client.ServiceURL("servers", url.PathEscape(serverID), "action")
	response, prior := rest.DoJSONGuarded(ctx, &source.Client, guard, http.MethodPost, target, body, map[string]string{"Accept": ""}, codes...)
	if response == nil {
		return fail(prior)
	}
	if err := errors.Join(prior, guard(ctx)); err != nil {
		return fail(response.Fail(cloudread.ContextError(ctx, err)))
	}

	// Decode the root only. A singular-object envelope decoder would reject
	// console:null and nonobject values which resp.json()['console'] returns.
	root, decodeErr := rest.Decode(response, "", func(value *resource.RawResource) *resource.Metadata { return &value.Metadata })
	if decodeErr != nil {
		return fail(decodeErr)
	}
	console, present := root.Body["console"]
	if !present {
		return fail(response.Fail(fmt.Errorf("%w: console response is missing key %q", resource.ErrInvalidOption, "console")))
	}
	if err := guard(ctx); err != nil {
		return fail(response.Fail(cloudread.ContextError(ctx, err)))
	}
	return bytes.Clone(console), nil
}
