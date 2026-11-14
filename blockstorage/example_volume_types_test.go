package blockstorage_test

import (
	"context"
	"encoding/json"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func ReadTypesVolumeTypes(ctx context.Context, conn *sdk.Connection) error {
	listed, err := conn.ListVolumeTypes(ctx)
	if err != nil {
		if listed != nil {
			fmt.Println("admitted list pages:", len(listed.Pages))
		}
		return err
	}
	fmt.Println("normalized types:", string(listed.Value))
	for _, item := range listed.Types {
		owned := item.Clone()
		fmt.Println("original extra_specs:", string(owned.Body["extra_specs"]))
		var projection struct {
			ID         *string                    `json:"id"`
			Name       *string                    `json:"name"`
			ExtraSpecs map[string]json.RawMessage `json:"extra_specs"`
		}
		if err := owned.Decode(&projection); err != nil {
			return err
		}
	}

	searched, err := conn.SearchVolumeTypes(ctx,
		blockstorage.SearchVolumeTypesRequest{NameOrID: "fast*"},
		blockstorage.WithVolumeTypeSearchFilters(json.RawMessage(`{"is_public":true}`)))
	if err != nil {
		if searched != nil {
			fmt.Println("admitted search pages:", len(searched.Pages))
		}
		return err
	}
	fmt.Println("matched types:", string(searched.Value))

	selected, err := conn.GetVolumeType(ctx,
		blockstorage.GetVolumeTypeRequest{NameOrID: "fast"})
	if err != nil {
		if selected != nil {
			fmt.Println("admitted fallback pages:", len(selected.Pages))
			if selected.Observed != nil {
				fmt.Println("admitted member status:", selected.Observed.StatusCode)
			}
		}
		return err
	}
	if selected.Value == nil {
		fmt.Println("type absent")
	} else {
		fmt.Println("selected type:", string(selected.Value))
	}

	expression, err := conn.SearchVolumeTypes(ctx,
		blockstorage.SearchVolumeTypesRequest{},
		blockstorage.WithVolumeTypeSearchExpression("[].name"))
	if err != nil {
		return err
	}
	fmt.Println("expression JSON:", string(expression.Value))
	return nil
}

func ReadTypesWithLocationVolumeTypes(ctx context.Context, conn *sdk.Connection) error {
	cloud, project := "configured-cloud", "configured-project"
	location := resource.CloudLocation{
		Cloud: &cloud, Zone: json.RawMessage(`"owned-zone"`),
		Project: resource.CloudProject{ID: json.RawMessage(`"scope-id"`), Name: &project},
	}
	result, err := conn.ListVolumeTypes(ctx,
		blockstorage.WithVolumeTypeReadOptions(blockstorage.VolumeTypeReadOpts{Location: &location}))
	if err != nil {
		return err
	}
	fmt.Println(string(result.Value))
	return nil
}

func ReadTypesDirectVolumeTypes(ctx context.Context, cinder *gophercloud.ServiceClient) error {
	result, err := blockstorage.GetVolumeType(ctx, cinder,
		blockstorage.GetVolumeTypeRequest{NameOrID: "fast"},
		blockstorage.WithVolumeTypeSearchFilters(json.RawMessage(`{}`)))
	if err != nil {
		return err
	}
	fmt.Println(string(result.Value))
	return nil
}
