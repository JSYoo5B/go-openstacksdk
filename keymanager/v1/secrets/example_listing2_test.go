package secrets_test

import (
	"context"
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func listExactSecretRowsListing2(ctx context.Context, client *gophercloud.ServiceClient) ([]*secrets.Secret, error) {
	return secrets.New(client).Resources.All(ctx,
		resource.WithFilters(map[string]any{
			"mode":       []any{"cbc", "gcm"},
			"bit_length": json.Number("256"),
			"created_at": "2026-10-02T03:04:05",
		}),
		resource.WithBodyFilter("status", "ACTIVE"),
		resource.WithPaginated(false))
}
