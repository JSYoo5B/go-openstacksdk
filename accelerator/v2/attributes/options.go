package attributes

import (
	"encoding/json"
	"github.com/JSYoo5B/gophercloudsdk/request"
)

// CreateOpts creates one attribute. DeployableID is the numeric database ID,
// not the deployable UUID. Empty values are sent explicitly.
type CreateOpts struct {
	DeployableID int64  `json:"deployable_id"`
	Key          string `json:"key"`
	Value        string `json:"value"`
}

type CreateOption = request.Option[CreateOpts]

func WithCreateOptions(value CreateOpts) CreateOption { return request.WithOptions(value) }
func WithCreateField(key string, value any) CreateOption {
	return request.WithField[CreateOpts](key, value)
}
func WithCreateHeader(key, value string) CreateOption {
	return request.WithHeader[CreateOpts](key, value)
}

func prepareCreate(opts CreateOpts, options ...CreateOption) (json.RawMessage, map[string]string, error) {
	config, err := request.Apply(opts, options...)
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
