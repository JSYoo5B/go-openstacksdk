package cloudsnapshot

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// Pinned Backup._action explicitly selects3.64. Ordinary create, lookup and
// wait requests keep the captured selected microversion. Only this action's
// independent client and canonical version headers use the forced policy.
func (p *reader) backupForceDelete(ctx context.Context, target string) (*rest.Response, error) {
	return p.backupPost(ctx, target, json.RawMessage(`{"os-force_delete":null}`), true)
}

func backupActionVersion(key string) (string, bool) {
	switch strings.ToLower(key) {
	case "openstack-api-version":
		return "volume 3.64", true
	case "x-openstack-volume-api-version":
		return "3.64", true
	}
	return "", false
}

// Check the actual physical headers too: redirects and native callbacks may
// modify a request after options validation, before an authenticated resend.
func backupActionHeaders(headers http.Header) error {
	for _, required := range []string{"Openstack-Api-Version", "X-Openstack-Volume-Api-Version"} {
		expected, _ := backupActionVersion(required)
		values := []string{}
		for key, candidates := range headers {
			if strings.EqualFold(key, required) {
				values = append(values, candidates...)
			}
		}
		if len(values) != 1 || values[0] != expected {
			return invalid("backup action requires microversion header %q=%q", required, expected)
		}
	}
	return nil
}
