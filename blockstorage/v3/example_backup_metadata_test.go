package v3_test

import (
	"context"
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

func fetchBackupMetadata(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]string, error) {
	profile := backups.New(client).PythonMetadata()
	result, err := profile.FetchMetadata(ctx, id,
		request.Header("X-Request-Id", "application-request"))
	if err != nil {
		return nil, err
	}
	var values map[string]string
	if result.Metadata != nil {
		if err := json.Unmarshal(result.Metadata, &values); err != nil {
			return nil, err
		}
	}
	return values, nil
}
