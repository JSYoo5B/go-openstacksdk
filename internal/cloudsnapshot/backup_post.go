package cloudsnapshot

import (
	"context"
	"encoding/json"
	"github.com/JSYoo5B/go-openstacksdk/internal/cinderrequest"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// Forced backup actions own3.64; ordinary restore keeps the selected policy.
func (p *reader) backupPost(ctx context.Context, target string, body json.RawMessage, forceVersion bool) (*rest.Response, error) {
	var version *string
	if forceVersion {
		selected := "3.64"
		version = &selected
	}
	return p.backupPostPolicy(ctx, target, body, version)
}
func (p *reader) backupPostPolicy(ctx context.Context, target string, body json.RawMessage, version *string) (*rest.Response, error) {
	return cinderrequest.Post(ctx, p.source, target, body, version, sourceCodes()...)
}
