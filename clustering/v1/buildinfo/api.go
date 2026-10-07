// Package buildinfo reads the Senlin API and engine build revisions.
package buildinfo

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/internal/senlin"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

// Revision preserves additional deployment revision properties in Body.
type Revision struct {
	Revision *string                    `json:"revision"`
	Body     map[string]json.RawMessage `json:"-"`
}

func (r *Revision) UnmarshalJSON(data []byte) error {
	type plain Revision
	var metadata resource.Metadata
	if err := resource.DecodeObject(data, (*plain)(r), &metadata); err != nil {
		return err
	}
	r.Body = metadata.Body
	return nil
}

type BuildInfo struct {
	resource.Metadata
	API    *Revision `json:"api"`
	Engine *Revision `json:"engine"`
}

func (value *BuildInfo) UnmarshalJSON(data []byte) error {
	type plain BuildInfo
	return resource.DecodeObject(data, (*plain)(value), &value.Metadata)
}

// Get fetches the singleton, without appending a fabricated resource ID.
func (a *API) Get(ctx context.Context) (*BuildInfo, error) {
	client := a.RawClient()
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, request.Wrap("Get", "clustering.build_info", err)
	}
	response, err := rest.DoJSON(ctx, client, http.MethodGet, client.ServiceURL("build-info"), nil, nil, http.StatusOK)
	if err != nil {
		return nil, request.Wrap("Get", "clustering.build_info", err)
	}
	value, err := rest.Decode(response, "build_info", func(value *BuildInfo) *resource.Metadata { return &value.Metadata })
	return value, request.Wrap("Get", "clustering.build_info", err)
}
