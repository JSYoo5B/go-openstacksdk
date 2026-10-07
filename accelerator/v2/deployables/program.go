package deployables

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/JSYoo5B/gophercloudsdk/accelerator/v2/common"
	"github.com/JSYoo5B/gophercloudsdk/internal/cyborg"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"net/url"
)

// ProgramOperation is the Cyborg program endpoint's JSON-Patch document. Value
// is concrete so callers do not implement a patch or request-body builder.
type ProgramOperation struct {
	Op    string         `json:"op"`
	Path  string         `json:"path"`
	Value []ProgramImage `json:"value"`
}

type ProgramImage struct {
	ImageUUID string `json:"image_uuid"`
}
type ProgramOption = request.Option[struct{}]

func WithProgramHeader(key, value string) ProgramOption {
	return request.WithHeader[struct{}](key, value)
}

// Program builds the valid replace /program patch for one bitstream image.
// Resolving the deployable name is owned by the SDK; programming uses its UUID.
func (a *API) Program(ctx context.Context, deployable resource.Ref, imageUUID string, options ...ProgramOption) (*Deployable, error) {
	return a.Patch(ctx, deployable, []ProgramOperation{{Op: "replace", Path: "/program", Value: []ProgramImage{{ImageUUID: imageUUID}}}}, options...)
}

// Patch calls /deployables/{uuid}/program exactly once. The pinned controller
// accepts one replace /program operation containing one image, not a general
// resource update. Inputs are validated before name lookup or mutation.
func (a *API) Patch(ctx context.Context, deployable resource.Ref, patch []ProgramOperation, options ...ProgramOption) (*Deployable, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := cyborg.RequireMicroversion(a.client, 0); err != nil {
		return nil, err
	}
	if len(patch) != 1 || patch[0].Op != "replace" || patch[0].Path != "/program" || len(patch[0].Value) != 1 {
		return nil, fmt.Errorf("%w: programming requires one replace /program operation with one image", resource.ErrInvalidOption)
	}
	if err := resource.ID(patch[0].Value[0].ImageUUID).Validate(); err != nil {
		return nil, err
	}
	c, err := request.Apply(struct{}{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(c, false, false, true)
	}
	if err != nil {
		return nil, err
	}
	headers, err := cyborg.Headers(c.Headers)
	if err != nil {
		return nil, err
	}
	// Marshal before HTTP so the request owns its nested input slice.
	body, err := json.Marshal(patch)
	if err != nil {
		return nil, err
	}
	id, err := a.ResolveID(ctx, deployable)
	if err != nil {
		return nil, err
	}
	value, err := cyborg.DecodeMutation(ctx, a.client, "PATCH", a.client.ServiceURL("deployables", url.PathEscape(id), "program"), json.RawMessage(body), headers, "deployable", func(v *Deployable) *common.Metadata { return &v.Metadata }, 200)
	return value, request.Wrap("program", "deployables", err)
}
