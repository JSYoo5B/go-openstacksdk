package cloudsnapshot

import (
	"context"
	"encoding/json"
	"errors"

	"gophercloudsdk/internal/rest"
	"gophercloudsdk/resource"
)

func (p *reader) mutationExchange(ctx context.Context, method, target string, body json.RawMessage) (*rest.Response, error) {
	var payload any
	if body != nil {
		payload = body
	}
	return rest.DoJSONGuarded(ctx, &p.source.Client, p.source.Guard, method, target, payload, nil, sourceCodes()...)
}

// The physical object is returned even if subsequent descriptor conversion
// fails. A merged logical state never changes that object's actual fields.
func (p *reader) mergeMutation(ctx context.Context, state *mutationState, wire *rest.Response) (json.RawMessage, *resource.RawResource, error) {
	raw, actual, err := mutationObject(wire)
	if err != nil {
		return nil, actual, err
	}
	if raw != nil {
		if err := state.overlay(raw); err != nil {
			return nil, actual, wire.Fail(err)
		}
	}
	view, err := state.view(p.location)
	if err != nil {
		var own *locationError
		if errors.As(err, &own) {
			return nil, actual, err
		}
		return nil, actual, wire.Fail(err)
	}
	if err := p.source.Guard(ctx); err != nil {
		return nil, actual, wire.Fail(err)
	}
	return view, actual, nil
}
