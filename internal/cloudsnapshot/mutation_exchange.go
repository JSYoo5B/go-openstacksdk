package cloudsnapshot

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
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
	return p.mergeMutationFields(ctx, state[:], descriptors[:], wire)
}

func (p *reader) mergeMutationFields(ctx context.Context, values []json.RawMessage, fields []descriptor, wire *rest.Response) (json.RawMessage, *resource.RawResource, error) {
	raw, actual, err := mutationObjectFor(wire, p.schema.singular)
	if err != nil {
		return nil, actual, err
	}
	if raw != nil {
		if err := overlayMutationFields(values, fields, raw); err != nil {
			return nil, actual, wire.Fail(err)
		}
	}
	view, _, err := p.schema.normalize(mutationFieldsObject(values, fields), nil, false, p.location)
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

func (p *reader) mergeBackupMutation(ctx context.Context, state *backupMutationState, wire *rest.Response) (json.RawMessage, *resource.RawResource, error) {
	return p.mergeMutationFields(ctx, state[:], backupDescriptors[:], wire)
}
