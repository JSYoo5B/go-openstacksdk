package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func ConfiguredVolumeBackups(ctx context.Context, cloudName string) (*sdk.Connection, error) {
	return sdk.Connect(ctx, sdk.WithCloud(cloudName))
}

func AdoptVolumeBackups(provider *gophercloud.ProviderClient) (*sdk.Connection, error) {
	return sdk.FromProvider(provider, sdk.WithRegion("RegionOne"))
}

func InspectPagesVolumeBackups(pages []*blockstorage.VolumeBackupsPage) {
	for _, page := range pages {
		fmt.Println("admitted backup page:", page.StatusCode, "bytes:", len(page.Body))
	}
}

func ExplainVolumeBackups(err error) {
	if err == nil {
		return
	}
	var ambiguous *blockstorage.VolumeBackupSelectionError
	if errors.As(err, &ambiguous) {
		fmt.Println("filtered selection length:", ambiguous.Length)
	}
	fmt.Println("ambiguous:", errors.Is(err, resource.ErrAmbiguous), "error:", err)
}

func InspectVolumeBackups(result *blockstorage.GetVolumeBackupResult) {
	if result == nil {
		return
	}
	InspectPagesVolumeBackups(result.Pages)
	if result.Observed != nil {
		fmt.Println("member response:", result.Observed.StatusCode, len(result.Observed.Body))
	}
	fmt.Println("requested ID:", result.RequestedID, "seeded logical ID:", result.SeededID)
	if result.Backup != nil {
		owned := result.Backup.Clone()
		fmt.Println("actual wire ID JSON:", string(owned.Body["id"]))
		var projection struct {
			ID   json.RawMessage `json:"id"`
			Name *string         `json:"name"`
		}
		if err := owned.Decode(&projection); err != nil {
			fmt.Println("optional wire projection:", err)
		} else {
			fmt.Println("projected wire ID:", string(projection.ID))
		}
	}
}

func ViaConnectionVolumeBackups(ctx context.Context, conn *sdk.Connection) error {
	listed, err := conn.ListVolumeBackups(ctx,
		blockstorage.WithVolumeBackupListFilters(json.RawMessage(
			`{"status":"available","volume_id":"volume-id","description":"nightly"}`)))
	if listed != nil {
		InspectPagesVolumeBackups(listed.Pages)
	}
	if err != nil {
		return err
	}
	fmt.Println("complete normalized list:", string(listed.Value), "raw backups:", len(listed.Backups))

	searched, err := conn.SearchVolumeBackups(ctx,
		blockstorage.SearchVolumeBackupsRequest{NameOrID: "nightly*"},
		blockstorage.WithVolumeBackupSearchFilters(json.RawMessage(`{"status":"available"}`)))
	if searched != nil {
		InspectPagesVolumeBackups(searched.Pages)
	}
	if err != nil {
		return err
	}
	fmt.Println("selected normalized backups:", string(searched.Value))

	selected, err := conn.GetVolumeBackup(ctx,
		blockstorage.GetVolumeBackupRequest{NameOrID: "nightly"})
	InspectVolumeBackups(selected) // InspectVolumeBackups actual proof before deciding what to do with err.
	ExplainVolumeBackups(err)
	if err != nil {
		return err
	}
	if selected.Value == nil {
		fmt.Println("completed lookup: no backup found")
	} else {
		fmt.Println("selected normalized JSON:", string(selected.Value))
	}
	return nil
}

func PreparedFirstPageVolumeBackups(ctx context.Context, conn *sdk.Connection) error {
	calls := 0
	policy, err := blockstorage.PrepareVolumeBackupListOptions(ctx,
		request.Custom(func(value *blockstorage.VolumeBackupListOpts) error {
			calls++
			filters := json.RawMessage(`{"status":"available"}`)
			value.Filters = &filters
			return nil
		}),
		blockstorage.WithVolumeBackupListDetailed(false),
		request.Paginated(false),
		request.MaxItems(10),
		request.Headers(map[string]string{
			"X-Request-ID": "backup-read-example",
		}))
	if err != nil {
		return err
	}
	fmt.Println("original callback calls:", calls)
	result, err := conn.ListVolumeBackups(ctx,
		blockstorage.WithVolumeBackupListOptions(policy))
	if result != nil {
		InspectPagesVolumeBackups(result.Pages)
	}
	return err
}

func ProjectNamesVolumeBackups(ctx context.Context, conn *sdk.Connection) error {
	result, err := conn.SearchVolumeBackups(ctx,
		blockstorage.SearchVolumeBackupsRequest{},
		blockstorage.WithVolumeBackupSearchExpression("[].name"))
	if result != nil {
		InspectPagesVolumeBackups(result.Pages)
	}
	if err != nil {
		return err
	}
	fmt.Println("arbitrary expression JSON:", string(result.Value), "raw associations:", len(result.Backups))
	return nil
}

func FilteredGetVolumeBackups(ctx context.Context, conn *sdk.Connection, name string) error {
	// Explicit {} chooses complete Search instead of ordinary GET-first find.
	result, err := conn.GetVolumeBackup(ctx,
		blockstorage.GetVolumeBackupRequest{NameOrID: name},
		blockstorage.WithVolumeBackupSearchFilters(json.RawMessage(`{}`)))
	InspectVolumeBackups(result)
	ExplainVolumeBackups(err)
	return err
}

func DirectVolumeBackups(ctx context.Context, cinder *gophercloud.ServiceClient) error {
	cloud, region, projectName := "owned-cloud", "RegionOne", "configured-project"
	location := resource.CloudLocation{
		Cloud: &cloud, RegionName: &region,
		Project: resource.CloudProject{ID: json.RawMessage(`"project-id"`), Name: &projectName},
	}
	listed, err := blockstorage.ListVolumeBackups(ctx, cinder,
		request.Custom(func(c *blockstorage.VolumeBackupListOpts) error {
			loc := location.Clone()
			c.Location = &loc
			return nil
		}))
	if listed != nil {
		InspectPagesVolumeBackups(listed.Pages)
	}
	if err != nil {
		return err
	}

	policy, err := blockstorage.PrepareVolumeBackupSearchOptions(ctx,
		func(c *blockstorage.VolumeBackupSearchOpts) error {
			loc := location.Clone()
			c.Location = &loc
			return nil
		},
		blockstorage.WithVolumeBackupSearchFilters(json.RawMessage(`{"status":"available"}`)))
	if err != nil {
		return err
	}
	searched, err := blockstorage.SearchVolumeBackups(ctx, cinder,
		blockstorage.SearchVolumeBackupsRequest{NameOrID: "nightly*"},
		blockstorage.WithVolumeBackupSearchOptions(policy))
	if searched != nil {
		InspectPagesVolumeBackups(searched.Pages)
	}
	if err != nil {
		return err
	}

	selected, err := blockstorage.GetVolumeBackup(ctx, cinder,
		blockstorage.GetVolumeBackupRequest{NameOrID: "nightly"},
		func(c *blockstorage.VolumeBackupSearchOpts) error {
			loc := location.Clone()
			c.Location = &loc
			return nil
		})
	InspectVolumeBackups(selected)
	return err
}
