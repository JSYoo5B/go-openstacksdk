package attributes

import (
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/cyborg"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// CreateOpts creates one attribute. DeployableID is the numeric database ID,
// not the deployable UUID. Empty values are sent explicitly.
type CreateOpts struct {
	DeployableID int64  `json:"deployable_id"`
	Key          string `json:"key"`
	Value        string `json:"value"`
}

// CreateOption is one option of Create: WithCreateOptions, request.Field (an
// extension body member that cannot replace a typed one), request.Header or
// request.Custom over *request.Config[CreateOpts]. Other shared calls fail
// with resource.ErrInvalidOption before HTTP.
type CreateOption = request.WriteCall

var createCalls = cyborg.FieldHeaderCalls[CreateOpts]()

func WithCreateOptions(value CreateOpts) CreateOption {
	return request.Custom(request.WithOptions(value))
}

func prepareCreate(opts CreateOpts, options ...CreateOption) (json.RawMessage, map[string]string, error) {
	config, err := calls.ApplyConfig(opts, createCalls, options...)
	if err != nil {
		return nil, nil, err
	}
	if err := request.ValidateCapabilities(config, true, false, true); err != nil {
		return nil, nil, err
	}
	body := map[string]any{"deployable_id": config.Options.DeployableID, "key": config.Options.Key, "value": config.Options.Value}
	body, err = request.MergeFieldsFor(body, config.Fields, config.Options)
	if err != nil {
		return nil, nil, err
	}
	encoded, err := json.Marshal(body)
	return encoded, config.Headers, err
}
